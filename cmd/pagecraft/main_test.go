package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPagecraft_StdinToStdout_WritesPDF(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run(nil, strings.NewReader("<p>Hello from stdin</p>"), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %q", code, stderr.String())
	}
	if !bytes.HasPrefix(stdout.Bytes(), []byte("%PDF-")) {
		t.Fatalf("stdout does not start with PDF header: %q", stdout.Bytes()[:min(8, stdout.Len())])
	}
	if stderr.Len() != 0 {
		t.Errorf("unexpected stderr: %q", stderr.String())
	}
}

func TestPagecraft_InputFileAndOutputFlag_WritesPDFFile(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "input.html")
	output := filepath.Join(dir, "result.pdf")
	if err := os.WriteFile(input, []byte("<p>Hello from file</p>"), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := run([]string{"-o", output, input}, strings.NewReader("unused"), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %q", code, stderr.String())
	}
	pdf, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		t.Fatalf("output file does not start with PDF header: %q", pdf[:min(8, len(pdf))])
	}
	if stdout.Len() != 0 {
		t.Errorf("PDF should go to the output file, got %d stdout bytes", stdout.Len())
	}
}

func TestPagecraft_InputFile_ResolvesRelativeStylesheet(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "input.html")
	stylesheet := filepath.Join(dir, "style.css")
	if err := os.WriteFile(stylesheet, []byte("p { color: #123456; }"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(input, []byte(`<html><head><link rel="stylesheet" href="style.css"></head><body><p>Styled text</p></body></html>`), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := run([]string{input}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %q", code, stderr.String())
	}
	if !bytes.HasPrefix(stdout.Bytes(), []byte("%PDF-")) {
		t.Fatalf("stdout does not start with PDF header: %q", stdout.Bytes()[:min(8, stdout.Len())])
	}
}

func TestPagecraft_InvalidHTML_ReturnsErrorWithoutPDF(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run(nil, strings.NewReader("   "), &stdout, &stderr)
	if code == 0 {
		t.Fatal("expected a non-zero exit code")
	}
	if stdout.Len() != 0 {
		t.Errorf("unexpected stdout: %q", stdout.Bytes())
	}
	if stderr.Len() == 0 {
		t.Error("expected an error message on stderr")
	}
}
