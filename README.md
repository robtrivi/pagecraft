# Pagecraft

Pagecraft turns HTML and a practical subset of CSS into paginated PDF documents. Use it to generate invoices, reports, forms, catalogs, labels, and other documents where page size, pagination, and readable output matter. The rendering engine is written in Go and is inspired by [WeasyPrint](https://weasyprint.org/).

## Use Pagecraft from Go

Add the renderer to your Go application:

```sh
go get github.com/robtrivi/pagecraft
```

Pagecraft accepts an HTML string and returns the finished PDF as bytes. Build the HTML from your application data with `html/template`, another template package, or your own code, then pass the result to `render.RenderHTMLToPDF`:

```go
package main

import (
 "bytes"
 "html/template"
 "os"

 "github.com/robtrivi/pagecraft/render"
)

type Invoice struct {
 Number string
 Client string
 Total  string
}

func main() {
 const document = `<!doctype html>
<html>
<head>
  <style>
    @page { size: A4; margin: 20mm; }
    body { font: 12pt sans-serif; }
    h1 { color: #183b3a; }
  </style>
</head>
<body>
  <h1>Invoice {{.Number}}</h1>
  <p>Bill to: {{.Client}}</p>
  <p>Total due: {{.Total}}</p>
</body>
</html>`

 tmpl := template.Must(template.New("invoice").Parse(document))
 var html bytes.Buffer
 err := tmpl.Execute(&html, Invoice{
  Number: "INV-1042",
  Client: "Northstar Studio",
  Total:  "$1,460.00",
 })
 if err != nil {
  panic(err)
 }

 pdf, err := render.RenderHTMLToPDF(html.String())
 if err != nil {
  panic(err)
 }
 if err := os.WriteFile("invoice.pdf", pdf, 0o644); err != nil {
  panic(err)
 }
}
```

The renderer is a Go package in your process: generate HTML from the data and templates you already have, call the rendering function, and decide where to store, return, or send the PDF.

## Supported HTML and CSS

Pagecraft implements a document-focused CSS subset rather than a full browser. It supports stylesheets and inline styles; physical page sizes and margins through `@page`; page footers with page counters; text wrapping and lists; tables, floats, flexbox, and grid; and multi-page layout. It can render borders, backgrounds, common typography, PNG and JPEG images, SVG graphics, links, metadata, and TrueType fonts declared with `@font-face`.

Unsupported CSS may be ignored or render differently from Chromium. The examples show these features in complete documents.

## Examples

The [`examples/`](examples/) directory contains complete HTML documents you can open, adapt, and render with Pagecraft. The previews below show the first page of each document; select one to view its HTML source.

<table>
  <tr>
    <td width="50%" align="center"><a href="examples/invoice.html"><img src="examples/screenshots/invoice-720x1080.jpg" width="360" height="540" alt="Preview of the Northstar Studio invoice" /><br /><strong>Invoice</strong></a></td>
    <td width="50%" align="center"><a href="examples/operations-report.html"><img src="examples/screenshots/operations-report-720x1080.jpg" width="360" height="540" alt="Preview of the Meridian Services operations report" /><br /><strong>Operations report</strong></a></td>
  </tr>
  <tr>
    <td width="50%" align="center"><a href="examples/product-catalog.html"><img src="examples/screenshots/product-catalog-720x1080.jpg" width="360" height="540" alt="Preview of the North House Goods product catalog" /><br /><strong>Product catalog</strong></a></td>
    <td width="50%" align="center"><a href="examples/shipping-label.html"><img src="examples/screenshots/shipping-label-720x1080.jpg" width="360" height="540" alt="Preview of the Northstar Logistics shipping label" /><br /><strong>Shipping label</strong></a></td>
  </tr>
  <tr>
    <td width="50%" align="center"><a href="examples/regional-expansion-plan.html"><img src="examples/screenshots/regional-expansion-plan-720x1080.jpg" width="360" height="540" alt="Preview of the Meridian Services regional expansion plan" /><br /><strong>Regional expansion plan</strong></a></td>
    <td width="50%"></td>
  </tr>
</table>

The examples include invoices, operational and planning reports, a product catalog, and a shipping label. The regional expansion plan also demonstrates landscape appendices. Use them as starting points for your own HTML and CSS.

## Support

If you run into unexpected behavior or need a feature that is not listed above, [open an issue](https://github.com/robtrivi/pagecraft/issues). Include a small HTML/CSS document that reproduces the problem so it is easier to investigate.

## Other ways to use Pagecraft

The Go package is the direct integration for Go applications. Pagecraft also ships as a standalone command-line program and can run as an HTTP service, so applications written in other languages can use the same renderer.

### Command-line interface

Install the command with Go:

```sh
go install github.com/robtrivi/pagecraft/cmd/pagecraft@latest
```

Or download a prebuilt binary for your operating system and CPU from the [latest release](https://github.com/robtrivi/pagecraft/releases/latest). The release binaries are standalone; Go does not need to be installed.

Render a file to PDF:

```sh
pagecraft -o report.pdf report.html
```

Pagecraft also reads HTML from standard input and writes PDF to standard output, which is useful for connecting it to other tools:

```sh
cat report.html | pagecraft > report.pdf
```

When the input is a file, relative stylesheets, images, and fonts are resolved from that file's directory. An explicit HTML `<base>` element takes precedence. For HTML read from standard input, relative resources are resolved from the process working directory.

### HTTP service

For applications that render documents frequently, run Pagecraft as a persistent service instead of starting a process for each document:

```sh
pagecraft serve -addr 127.0.0.1:8080 -workers 4
```

Send HTML to `POST /render`; the service returns `application/pdf`. `GET /healthz` reports service availability. The worker count defaults to `GOMAXPROCS` and can be capped with `-workers`. The request body limit defaults to 10 MiB and can be configured in bytes or directly in MiB:

```sh
pagecraft serve -max-body-mib 90
```

Here, `90` means 90 MiB (94,371,840 bytes). If both size flags are set, `-max-body-mib` takes precedence.

```sh
curl --data-binary @invoice.html \
  -H 'Content-Type: text/html; charset=utf-8' \
  http://127.0.0.1:8080/render \
  -o invoice.pdf
```

A service written in Java, Python, JavaScript, or another language can build HTML using its existing data and templates, then send that HTML over HTTP. Pagecraft binds to loopback by default. Only send it trusted HTML, and restrict network access if you bind it to another interface. Add a `<base href="...">` when the document refers to relative resources.

## Development

Run the test suite and static checks:

```sh
go test ./...
go vet ./...
```

## License

GNU GPL version 3 or later (`GPL-3.0-or-later`). See [`LICENSE`](LICENSE).
