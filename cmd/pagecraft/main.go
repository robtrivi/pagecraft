package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/robtrivi/pagecraft/render"
	"golang.org/x/net/html"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("pagecraft", flag.ContinueOnError)
	flags.SetOutput(stderr)
	outputPath := flags.String("o", "", "write PDF to this file (default: stdout; use - for stdout)")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: pagecraft [options] [input.html]")
		fmt.Fprintln(stderr, "Render an HTML document to PDF. Read HTML from stdin when no input file is given.")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() > 1 {
		fmt.Fprintln(stderr, "pagecraft: expected at most one input file")
		flags.Usage()
		return 2
	}

	var source []byte
	var inputPath string
	if flags.NArg() == 1 && flags.Arg(0) != "-" {
		inputPath = flags.Arg(0)
		var err error
		source, err = os.ReadFile(inputPath)
		if err != nil {
			fmt.Fprintf(stderr, "pagecraft: read %s: %v\n", inputPath, err)
			return 1
		}
	} else {
		var err error
		source, err = io.ReadAll(stdin)
		if err != nil {
			fmt.Fprintf(stderr, "pagecraft: read stdin: %v\n", err)
			return 1
		}
	}

	htmlSource := string(source)
	if inputPath != "" {
		var err error
		htmlSource, err = setBaseToInputDirectory(htmlSource, inputPath)
		if err != nil {
			fmt.Fprintf(stderr, "pagecraft: prepare input: %v\n", err)
			return 1
		}
	}

	pdf, err := render.RenderHTMLToPDF(htmlSource)
	if err != nil {
		fmt.Fprintf(stderr, "pagecraft: render: %v\n", err)
		return 1
	}

	if *outputPath == "" || *outputPath == "-" {
		if _, err := stdout.Write(pdf); err != nil {
			fmt.Fprintf(stderr, "pagecraft: write stdout: %v\n", err)
			return 1
		}
		return 0
	}
	if err := os.WriteFile(*outputPath, pdf, 0o644); err != nil {
		fmt.Fprintf(stderr, "pagecraft: write %s: %v\n", *outputPath, err)
		return 1
	}
	return 0
}

// setBaseToInputDirectory lets local stylesheets, images, and fonts resolve
// relative to the HTML file. An explicit <base> element keeps precedence.
func setBaseToInputDirectory(source, inputPath string) (string, error) {
	doc, err := html.Parse(strings.NewReader(source))
	if err != nil {
		return "", err
	}
	var head *html.Node
	var hasBase bool
	var find func(*html.Node)
	find = func(node *html.Node) {
		if node.Type == html.ElementNode {
			if node.Data == "head" && head == nil {
				head = node
			}
			if node.Data == "base" {
				hasBase = true
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			find(child)
		}
	}
	find(doc)
	if hasBase || head == nil {
		return source, nil
	}
	absolutePath, err := filepath.Abs(inputPath)
	if err != nil {
		return "", err
	}
	directory := filepath.ToSlash(filepath.Dir(absolutePath)) + "/"
	base := &html.Node{Type: html.ElementNode, Data: "base",
		Attr: []html.Attribute{{Key: "href", Val: directory}}}
	if head.FirstChild == nil {
		head.AppendChild(base)
	} else {
		head.InsertBefore(base, head.FirstChild)
	}
	var rendered bytes.Buffer
	if err := html.Render(&rendered, doc); err != nil {
		return "", err
	}
	return rendered.String(), nil
}
