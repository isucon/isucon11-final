package generate

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type countingWriter struct {
	buf    bytes.Buffer
	offset int
}

func (cw *countingWriter) Write(b []byte) (int, error) {
	n, err := cw.buf.Write(b)
	cw.offset += n
	return n, err
}

func (cw *countingWriter) Bytes() []byte {
	return cw.buf.Bytes()
}

type obj interface {
	write(w io.Writer, objID int) error
}

// PDF generates PDF1.7(ISO 32000-1) compatible PDF data.
func PDF(text string) []byte {
	w := &countingWriter{
		buf: bytes.Buffer{},
	}

	header(w)

	objs := []obj{
		&catalog{},
		&pages{
			PageCount:  1,
			Kids:       "3 0 R",
			pageHeight: 1000,
			pageWidth:  1600,
		},
		&page{
			contents: "4 0 R",
		},
		&contents{
			text:           text,
			x:              64 * 2,
			y:              1000 - 100,
			fontCountIndex: 1,
			fontSize:       64,
		},
	}

	linelens := body(w, objs)
	objNum := len(objs)

	xref(w, objNum, linelens)

	trailer(w, objNum)

	EOF(w)

	return w.Bytes()
}

// - components

// -- header
func header(w *countingWriter) {
	fmt.Fprint(w, "%PDF-1.7\n\n")
	return
}

// -- body
func body(w *countingWriter, objs []obj) []int {
	linelens := make([]int, len(objs))

	for i := range objs {
		objID := i + 1 // 1-indexed
		linelens[i] = w.offset
		fmt.Fprintf(w, "%d 0 obj\n", objID)
		objs[i].write(w, objID)
		io.WriteString(w, "endobj\n\n")
		i++
	}

	return linelens
}

// -- xref
func xref(w *countingWriter, objNum int, linelens []int) {
	io.WriteString(w, "xref\n")
	fmt.Fprintf(w, "0 %d\n", objNum+1)
	io.WriteString(w, "0000000000 65535 f \n")

	for i := range linelens {
		linelen := linelens[i]
		fmt.Fprintf(w, "%s 00000 n \n", formatXrefLinelen(linelen))
	}

	return
}

func formatXrefLinelen(n int) string {
	str := strconv.Itoa(n)
	for len(str) < 10 {
		str = "0" + str
	}
	return str
}

// -- trailer
func trailer(w io.Writer, objNum int) {
	io.WriteString(w, "trailer\n")
	io.WriteString(w, "<<\n")
	fmt.Fprintf(w, "/Size %d\n", objNum+1)
	io.WriteString(w, "/Root 1 0 R\n")
	return
}

// -- EOF
func EOF(w *countingWriter) {
	io.WriteString(w, ">>\n")
	io.WriteString(w, "startxref\n")
	fmt.Fprintf(w, "%d", w.offset)
	io.WriteString(w, "\n%%EOF\n")
	return
}

// - obj implementations

// -- catalog

type catalog struct{}

func (c *catalog) write(w io.Writer, _ int) error {
	io.WriteString(w, "<<\n")
	io.WriteString(w, "  /Type /Catalog\n")
	io.WriteString(w, "  /Pages 2 0 R\n")
	io.WriteString(w, ">>\n")
	return nil
}

// -- pages

type pages struct {
	PageCount  int
	Kids       string
	pageWidth  float64
	pageHeight float64
}

func (p *pages) write(w io.Writer, _ int) error {
	io.WriteString(w, "<<\n")
	io.WriteString(w, "  /Type /Pages\n")

	fmt.Fprintf(w, "  /MediaBox [ 0 0 %0.2f %0.2f ]\n", p.pageWidth, p.pageHeight)
	fmt.Fprintf(w, "  /Count %d\n", p.PageCount)
	fmt.Fprintf(w, "  /Kids [ %s ]\n", p.Kids) //sample Kids [ 3 0 R ]
	io.WriteString(w, ">>\n")
	return nil
}

// -- page

type page struct {
	contents string
}

func (p *page) write(w io.Writer, _ int) error {
	io.WriteString(w, "<<\n")

	io.WriteString(w, "  /Type /Page\n")
	io.WriteString(w, "  /Parent 2 0 R\n")

	io.WriteString(w, "  /Resources\n")
	fontResource(1, "Helvetica").WriteTo(w)

	fmt.Fprintf(w, "  /Contents %s\n", p.contents)

	io.WriteString(w, ">>\n")

	return nil
}

// TODO: 日本語対応
func fontResource(index int, baseFont string) *bytes.Buffer {
	buf := &bytes.Buffer{}

	io.WriteString(buf, "  << /Font\n")
	fmt.Fprintf(buf, "    << /F%d\n", index)
	io.WriteString(buf, "      << /Type /Font\n")
	io.WriteString(buf, "         /Subtype /Type1\n")
	fmt.Fprintf(buf, "         /BaseFont /%s\n", baseFont)
	io.WriteString(buf, "      >>\n")
	io.WriteString(buf, "    >>\n")
	io.WriteString(buf, "  >>\n")

	return buf
}

// -- page contents

type contents struct {
	text           string
	x              float64
	y              float64
	fontCountIndex int
	fontSize       float64
}

func (c *contents) write(w io.Writer, _ int) error {
	buf := &bytes.Buffer{}

	io.WriteString(buf, "  BT\n")
	fmt.Fprintf(buf, "    %0.2f %0.2f Td\n", c.x, c.y)
	fmt.Fprintf(buf, "    /F%d %0.2f Tf\n", c.fontCountIndex, c.fontSize)
	fmt.Fprintf(buf, "    %0.2f TL\n", c.fontSize*1.1)
	for _, line := range strings.Split(c.text, "\n") {
		fmt.Fprintf(buf, "    (%s) Tj T*\n", line)
	}
	io.WriteString(buf, "  ET\n")

	io.WriteString(w, "<<\n")
	fmt.Fprintf(w, "  /Length %d\n", buf.Len())
	io.WriteString(w, ">>\n")

	io.WriteString(w, "stream\n")
	buf.WriteTo(w)
	io.WriteString(w, "endstream\n")

	return nil
}
