package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/robtrivi/pagecraft/render"
)

const defaultMaxHTMLBytes int64 = 10 << 20

func newRenderHandler(workers int, maxBodyBytes int64, renderHTML func(string) ([]byte, error)) http.Handler {
	if workers < 1 {
		workers = 1
	}
	if maxBodyBytes < 1 {
		maxBodyBytes = defaultMaxHTMLBytes
	}
	if renderHTML == nil {
		renderHTML = render.RenderHTMLToPDF
	}

	workerSlots := make(chan struct{}, workers)
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = io.WriteString(w, "ok\n")
	})
	mux.HandleFunc("/render", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		select {
		case workerSlots <- struct{}{}:
			defer func() { <-workerSlots }()
		case <-r.Context().Done():
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		body, err := io.ReadAll(r.Body)
		if err != nil {
			var maxBytesErr *http.MaxBytesError
			if errors.As(err, &maxBytesErr) {
				http.Error(w, "HTML body exceeds size limit", http.StatusRequestEntityTooLarge)
				return
			}
			http.Error(w, "could not read HTML body", http.StatusBadRequest)
			return
		}
		html := string(body)
		if strings.TrimSpace(html) == "" {
			http.Error(w, "HTML body is empty", http.StatusBadRequest)
			return
		}

		pdf, err := renderHTML(html)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Length", fmt.Sprint(len(pdf)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(pdf)
	})
	return mux
}

func maxBodyBytesFromMiB(mib int64) (int64, error) {
	const bytesPerMiB = int64(1 << 20)
	if mib < 1 || mib > (1<<63-1)/bytesPerMiB {
		return 0, fmt.Errorf("max-body-mib must be between 1 and %d", (1<<63-1)/bytesPerMiB)
	}
	return mib * bytesPerMiB, nil
}

func runServer(ctx context.Context, args []string, stderr io.Writer) int {
	flags := flag.NewFlagSet("pagecraft serve", flag.ContinueOnError)
	flags.SetOutput(stderr)
	addr := flags.String("addr", "127.0.0.1:8080", "address for the rendering service")
	workers := flags.Int("workers", runtime.GOMAXPROCS(0), "maximum simultaneous renders")
	maxBodyBytes := flags.Int64("max-body-bytes", defaultMaxHTMLBytes, "maximum HTML request size in bytes")
	maxBodyMiB := flags.Int64("max-body-mib", 0, "maximum HTML request size in MiB (overrides -max-body-bytes)")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: pagecraft serve [options]")
		fmt.Fprintln(stderr, "Accept HTML at POST /render and return a paginated PDF.")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "pagecraft serve: unexpected positional argument")
		flags.Usage()
		return 2
	}
	if *workers < 1 {
		fmt.Fprintln(stderr, "pagecraft serve: workers must be greater than zero")
		return 2
	}
	maxBodyMiBSet := false
	flags.Visit(func(parsed *flag.Flag) {
		if parsed.Name == "max-body-mib" {
			maxBodyMiBSet = true
		}
	})
	if maxBodyMiBSet {
		configuredBytes, err := maxBodyBytesFromMiB(*maxBodyMiB)
		if err != nil {
			fmt.Fprintf(stderr, "pagecraft serve: %v\n", err)
			return 2
		}
		*maxBodyBytes = configuredBytes
	}
	if *maxBodyBytes < 1 {
		fmt.Fprintln(stderr, "pagecraft serve: max-body-bytes must be greater than zero")
		return 2
	}

	listener, err := net.Listen("tcp", *addr)
	if err != nil {
		fmt.Fprintf(stderr, "pagecraft serve: listen %s: %v\n", *addr, err)
		return 1
	}
	server := &http.Server{
		Handler:           newRenderHandler(*workers, *maxBodyBytes, nil),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	fmt.Fprintf(stderr, "pagecraft: rendering service listening on http://%s with %d workers\n", listener.Addr(), *workers)
	return serveHTTP(ctx, server, listener, stderr)
}

func serveHTTP(ctx context.Context, server *http.Server, listener net.Listener, stderr io.Writer) int {
	serveResult := make(chan error, 1)
	go func() { serveResult <- server.Serve(listener) }()
	select {
	case err := <-serveResult:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintf(stderr, "pagecraft serve: %v\n", err)
			return 1
		}
		return 0
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			_ = server.Close()
			fmt.Fprintf(stderr, "pagecraft serve: shutdown: %v\n", err)
			return 1
		}
		err := <-serveResult
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintf(stderr, "pagecraft serve: %v\n", err)
			return 1
		}
		return 0
	}
}
