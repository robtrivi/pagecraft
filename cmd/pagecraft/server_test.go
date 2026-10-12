package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestPagecraftServer_Healthz_ReturnsOK(t *testing.T) {
	handler := newRenderHandler(1, 1024, nil)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK || response.Body.String() != "ok\n" {
		t.Fatalf("response = (%d, %q), want (%d, %q)", response.Code, response.Body.String(), http.StatusOK, "ok\n")
	}
}

func TestPagecraftServer_RenderHTML_ReturnsPDF(t *testing.T) {
	handler := newRenderHandler(1, 1024, func(html string) ([]byte, error) {
		if html != "<p>invoice</p>" {
			return nil, fmt.Errorf("unexpected HTML: %q", html)
		}
		return []byte("%PDF-test"), nil
	})
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/render", strings.NewReader("<p>invoice</p>"))

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %q", response.Code, response.Body.String())
	}
	if got := response.Header().Get("Content-Type"); got != "application/pdf" {
		t.Errorf("Content-Type = %q, want application/pdf", got)
	}
	if got := response.Body.String(); got != "%PDF-test" {
		t.Errorf("body = %q, want %q", got, "%PDF-test")
	}
}

func TestPagecraftServer_RenderHTML_UsesPagecraftRenderer(t *testing.T) {
	handler := newRenderHandler(1, 1024, nil)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/render", strings.NewReader("<p>invoice</p>"))

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %q", response.Code, response.Body.String())
	}
	if !strings.HasPrefix(response.Body.String(), "%PDF-") {
		t.Fatalf("body does not contain a PDF: %q", response.Body.String()[:min(8, response.Body.Len())])
	}
}

func TestPagecraftServer_RenderHTML_HandlesConcurrentPDFs(t *testing.T) {
	server := httptest.NewServer(newRenderHandler(4, 1024, nil))
	defer server.Close()
	client := server.Client()

	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			response, err := client.Post(server.URL+"/render", "text/html", strings.NewReader("<p>invoice</p>"))
			if err != nil {
				t.Errorf("POST /render: %v", err)
				return
			}
			defer response.Body.Close()
			var body bytes.Buffer
			if _, err := io.Copy(&body, response.Body); err != nil {
				t.Errorf("read PDF: %v", err)
				return
			}
			if response.StatusCode != http.StatusOK || !bytes.HasPrefix(body.Bytes(), []byte("%PDF-")) {
				t.Errorf("response = (%d, %q), want a PDF response", response.StatusCode, body.Bytes()[:min(8, body.Len())])
			}
		}()
	}
	wg.Wait()
}

func TestPagecraftServer_RenderHTML_RejectsOversizeBody(t *testing.T) {
	handler := newRenderHandler(1, 4, func(string) ([]byte, error) {
		t.Fatal("renderer called for oversized request")
		return nil, nil
	})
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/render", strings.NewReader("12345"))

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestPagecraftServer_RenderHTML_RejectsUnsupportedMethod(t *testing.T) {
	handler := newRenderHandler(1, 1024, nil)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/render", nil)

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusMethodNotAllowed)
	}
	if got := response.Header().Get("Allow"); got != http.MethodPost {
		t.Errorf("Allow = %q, want %q", got, http.MethodPost)
	}
}

func TestPagecraftServer_RenderHTML_ReportsRenderError(t *testing.T) {
	handler := newRenderHandler(1, 1024, func(string) ([]byte, error) {
		return nil, errors.New("bad document")
	})
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/render", strings.NewReader("<p>broken</p>"))

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnprocessableEntity)
	}
}

func TestPagecraftServer_RenderHTML_LimitsConcurrentRenders(t *testing.T) {
	var active, peak atomic.Int32
	handler := newRenderHandler(2, 1024, func(string) ([]byte, error) {
		current := active.Add(1)
		for {
			oldPeak := peak.Load()
			if current <= oldPeak || peak.CompareAndSwap(oldPeak, current) {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
		active.Add(-1)
		return []byte("%PDF-test"), nil
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			response, err := http.Post(server.URL+"/render", "text/html", strings.NewReader("<p>invoice</p>"))
			if err != nil {
				t.Errorf("POST /render: %v", err)
				return
			}
			defer response.Body.Close()
			if response.StatusCode != http.StatusOK {
				t.Errorf("status = %d, want %d", response.StatusCode, http.StatusOK)
			}
		}()
	}
	wg.Wait()

	if got := peak.Load(); got != 2 {
		t.Errorf("peak concurrent renders = %d, want exactly 2", got)
	}
}

func TestPagecraftServe_InvalidWorkerCount_ReturnsUsageError(t *testing.T) {
	var stderr strings.Builder
	code := runWithContext(context.Background(), []string{"serve", "-workers", "0"}, strings.NewReader(""), &strings.Builder{}, &stderr)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2; stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "workers") {
		t.Errorf("stderr = %q, want worker flag error", stderr.String())
	}
}

func TestPagecraftServe_MaxBodyMiB_ConvertsToBytes(t *testing.T) {
	got, err := maxBodyBytesFromMiB(90)
	if err != nil {
		t.Fatal(err)
	}
	if want := int64(90 << 20); got != want {
		t.Errorf("max body bytes = %d, want %d", got, want)
	}
}

func TestPagecraftServe_MaxBodyMiB_RejectsInvalidValues(t *testing.T) {
	for _, mib := range []int64{0, -1, int64(^uint64(0)>>1)/(1<<20) + 1} {
		if _, err := maxBodyBytesFromMiB(mib); err == nil {
			t.Errorf("maxBodyBytesFromMiB(%d) succeeded, want error", mib)
		}
	}
}

func TestPagecraftServer_ShutdownWaitsForActiveRender(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	release := make(chan struct{})
	handler := newRenderHandler(1, 1024, func(string) ([]byte, error) {
		close(started)
		<-release
		return []byte("%PDF-test"), nil
	})
	server := &http.Server{Handler: handler}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	serveDone := make(chan int, 1)
	go func() { serveDone <- serveHTTP(ctx, server, listener, io.Discard) }()

	responseCh := make(chan *http.Response, 1)
	requestErrCh := make(chan error, 1)
	go func() {
		response, err := http.Post("http://"+listener.Addr().String()+"/render", "text/html", strings.NewReader("<p>invoice</p>"))
		if err != nil {
			requestErrCh <- err
			return
		}
		responseCh <- response
	}()
	select {
	case <-started:
	case err := <-requestErrCh:
		t.Fatalf("POST /render: %v", err)
	case <-time.After(time.Second):
		t.Fatal("render did not start")
	}

	cancel()
	select {
	case code := <-serveDone:
		t.Fatalf("server exited with %d before the active render completed", code)
	case <-time.After(20 * time.Millisecond):
	}
	close(release)
	select {
	case response := <-responseCh:
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			t.Errorf("status = %d, want %d", response.StatusCode, http.StatusOK)
		}
	case err := <-requestErrCh:
		t.Errorf("POST /render: %v", err)
	case <-time.After(time.Second):
		t.Fatal("active render did not return")
	}
	select {
	case code := <-serveDone:
		if code != 0 {
			t.Errorf("server exit code = %d, want 0", code)
		}
	case <-time.After(time.Second):
		t.Fatal("server did not shut down")
	}
}
