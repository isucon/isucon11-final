package generate

import (
	"bytes"
	"fmt"
	"image/color"
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
func PDF(text string, img *Image) []byte {
	w := &countingWriter{
		buf: bytes.Buffer{},
	}

	if err := header(w); err != nil {
		panic(err)
	}

	objs := []obj{
		&catalog{
			pages: "2 0 R",
		},
		&pages{
			pageCount:  2,
			kids:       "3 0 R 7 0 R",
			pageHeight: 1000,
			pageWidth:  1600,
		},
		// p1
		&page{
			parent:    "2 0 R",
			resources: "4 0 R",
			contents:  "6 0 R",
		},
		&procset{
			fonts: map[string]string{"F1": "5 0 R"},
		},
		&font{
			baseFont: "Helvetica",
		},
		&textContents{
			text:     text,
			x:        64 * 2,
			y:        1000 - 100,
			fontID:   "F1",
			fontSize: 64,
		},
		// p2
		&page{
			parent:    "2 0 R",
			resources: "8 0 R",
			contents:  "10 0 R",
		},
		&procset{
			xObjects: map[string]string{"I1": "9 0 R"},
		},
		img,
		&imageContents{
			imageID: "I1",
			x:       100,
			y:       100,
			mag:     700,
		},
	}

	linelens, _ := body(w, objs)
	objNum := len(objs)

	if err := xref(w, objNum, linelens); err != nil {
		panic(err)
	}

	if err := trailer(w, "1 0 R", objNum); err != nil {
		panic(err)
	}

	return w.Bytes()
}

// - components

// -- header
func header(w *countingWriter) error {
	if _, err := fmt.Fprint(w, "%PDF-1.7\n\n"); err != nil {
		return err
	}
	return nil
}

// -- body
func body(w *countingWriter, objs []obj) ([]int, error) {
	linelens := make([]int, len(objs))

	for i := range objs {
		objID := i + 1 // 1-indexed
		linelens[i] = w.offset
		if _, err := fmt.Fprintf(w, "%d 0 obj\n", objID); err != nil {
			return linelens, err
		}
		if err := objs[i].write(w, objID); err != nil {
			return nil, err
		}
		if _, err := io.WriteString(w, "endobj\n\n"); err != nil {
			return nil, err
		}
		i++
	}

	return linelens, nil
}

// -- xref
func xref(w *countingWriter, objNum int, linelens []int) error {
	var content strings.Builder

	content.WriteString("xref\n")
	content.WriteString(fmt.Sprintf("0 %d\n", objNum+1))
	content.WriteString("0000000000 65535 f \n")

	for i := range linelens {
		linelen := linelens[i]
		content.WriteString(fmt.Sprintf("%s 00000 n \n", formatXrefLinelen(linelen)))
	}

	if _, err := io.WriteString(w, content.String()); err != nil {
		return err
	}
	return nil
}

func formatXrefLinelen(n int) string {
	str := strconv.Itoa(n)
	for len(str) < 10 {
		str = "0" + str
	}
	return str
}

// -- trailer
func trailer(w *countingWriter, rootRef string, objNum int) error {
	var content strings.Builder

	content.WriteString("trailer\n")
	content.WriteString("<<\n")
	content.WriteString(fmt.Sprintf("\t/Size %d\n", objNum+1))
	content.WriteString(fmt.Sprintf("\t/Root %s\n", rootRef))
	content.WriteString(">>\n")

	content.WriteString("startxref\n")
	content.WriteString(fmt.Sprintf("%d", w.offset))
	content.WriteString("\n%%EOF\n")

	if _, err := io.WriteString(w, content.String()); err != nil {
		return err
	}
	return nil
}

// - obj implementations

// -- catalog

type catalog struct {
	pages string
}

func (c *catalog) write(w io.Writer, _ int) error {
	var content strings.Builder

	content.WriteString("<<\n")
	content.WriteString("\t/Type /Catalog\n")
	content.WriteString(fmt.Sprintf("\t/Pages %s\n", c.pages))
	content.WriteString(">>\n")

	if _, err := io.WriteString(w, content.String()); err != nil {
		return err
	}
	return nil
}

// -- pages

type pages struct {
	pageCount  int
	kids       string
	pageWidth  float64
	pageHeight float64
}

func (p *pages) write(w io.Writer, _ int) error {
	var content strings.Builder

	content.WriteString("<<\n")

	content.WriteString("\t/Type /Pages\n")
	content.WriteString(fmt.Sprintf("\t/MediaBox [ 0 0 %0.2f %0.2f ]\n", p.pageWidth, p.pageHeight))
	content.WriteString(fmt.Sprintf("\t/Count %d\n", p.pageCount))
	content.WriteString(fmt.Sprintf("\t/Kids [ %s ]\n", p.kids))

	content.WriteString(">>\n")

	if _, err := io.WriteString(w, content.String()); err != nil {
		return err
	}
	return nil
}

// -- page

type page struct {
	parent    string
	resources string
	contents  string
}

func (p *page) write(w io.Writer, _ int) error {
	var content strings.Builder
	content.WriteString("<<\n")

	content.WriteString("\t/Type /Page\n")
	content.WriteString(fmt.Sprintf("\t/Parent %s\n", p.parent))
	content.WriteString(fmt.Sprintf("\t/Resources %s\n", p.resources))
	if p.contents != "" {
		content.WriteString(fmt.Sprintf("\t/Contents %s\n", p.contents))
	}

	content.WriteString(">>\n")

	if _, err := io.WriteString(w, content.String()); err != nil {
		return err
	}

	return nil
}

// -- procedure set

type procset struct {
	fonts    map[string]string
	xObjects map[string]string
}

func (ps *procset) write(w io.Writer, _ int) error {
	var content strings.Builder

	content.WriteString("<<\n")
	//content.WriteString("\t/ProcSet [/PDF /Text /ImageB /ImageC /ImageI]\n")

	content.WriteString("\t/Font <<\n")
	for name, ref := range ps.fonts {
		content.WriteString(fmt.Sprintf("\t\t/%s %s\n", name, ref))
	}
	content.WriteString("\t>>\n")

	content.WriteString("\t/XObject <<\n")
	for name, ref := range ps.xObjects {
		content.WriteString(fmt.Sprintf("\t\t/%s %s\n", name, ref))
	}
	content.WriteString("\t>>\n")

	content.WriteString(">>\n")

	if _, err := io.WriteString(w, content.String()); err != nil {
		return err
	}
	return nil
}

// -- font

type font struct {
	baseFont string
}

func (f *font) write(w io.Writer, _ int) error {
	var content strings.Builder

	content.WriteString("<<\n")
	content.WriteString("\t/Type /Font\n")
	content.WriteString("\t/Subtype /Type1\n")
	content.WriteString(fmt.Sprintf("\t/BaseFont /%s\n", f.baseFont))
	content.WriteString(">>\n")

	if _, err := io.WriteString(w, content.String()); err != nil {
		return err
	}
	return nil
}

// -- image

func (i *Image) write(w io.Writer, _ int) error {
	var content strings.Builder

	content.WriteString("<<\n")

	content.WriteString("\t/Type /XObject\n")
	content.WriteString("\t/Subtype /Image\n")
	content.WriteString(fmt.Sprintf("\t/Width %d\n", i.width))
	content.WriteString(fmt.Sprintf("\t/Height %d\n", i.height))

	var colorSpace string
	switch i.colorModel {
	case color.YCbCrModel:
		colorSpace = "DeviceRGB"
	case color.GrayModel:
		colorSpace = "DeviceGray"
	case color.CMYKModel:
		colorSpace = "DeviceCMYK"
	default:
		panic("this color model is not supported")
	}
	content.WriteString(fmt.Sprintf("\t/ColorSpace /%s\n", colorSpace))
	if colorSpace == "DeviceCMYK" {
		content.WriteString("\t/Decode [1 0 1 0 1 0 1 0]\n")
	}

	// jpeg specific
	content.WriteString(fmt.Sprintf("\t/BitsPerComponent %d\n", 8))
	content.WriteString(fmt.Sprintf("\t/Filter /%s\n", "DCTDecode"))

	content.WriteString(fmt.Sprintf("\t/Length %d\n", len(i.data)))

	content.WriteString(">>\n")

	// image data stream
	content.WriteString("stream\n")
	content.Write(i.data)
	content.WriteString("\nendstream\n")

	if _, err := io.WriteString(w, content.String()); err != nil {
		return err
	}
	return nil
}

// -- text contents

type textContents struct {
	text     string
	x        float64
	y        float64
	fontID   string
	fontSize float64
}

func (tc *textContents) write(w io.Writer, _ int) error {
	var content strings.Builder

	content.WriteString("\tBT\n")
	content.WriteString(fmt.Sprintf("\t\t%0.2f %0.2f Td\n", tc.x, tc.y))
	content.WriteString(fmt.Sprintf("\t\t%s %0.2f Tf\n", tc.fontID, tc.fontSize))
	content.WriteString(fmt.Sprintf("\t\t%0.2f TL\n", tc.fontSize*1.1))
	for _, line := range strings.Split(tc.text, "\n") {
		content.WriteString(fmt.Sprintf("\t\t(%s) Tj T*\n", line))
	}
	content.WriteString("\tET\n")

	var content2 strings.Builder
	content2.WriteString("<<\n")
	content2.WriteString(fmt.Sprintf("\t/Length %d\n", content.Len()))
	content2.WriteString(">>\n")

	content2.WriteString("stream\n")
	content2.WriteString(content.String())
	content2.WriteString("endstream\n")

	if _, err := io.WriteString(w, content2.String()); err != nil {
		return err
	}
	return nil
}

// -- image contents

type imageContents struct {
	imageID string
	x       float64
	y       float64
	mag     int
}

func (ic *imageContents) write(w io.Writer, _ int) error {
	var content strings.Builder

	content.WriteString("\tq\n")
	content.WriteString(fmt.Sprintf("\t%d 0 0 %d %0.2f %0.2f cm\n", ic.mag, ic.mag, ic.x, ic.y))
	content.WriteString(fmt.Sprintf("\t/%s Do\n", ic.imageID))
	content.WriteString("\tQ\n")

	var content2 strings.Builder
	content2.WriteString("<<\n")
	content2.WriteString(fmt.Sprintf("\t/Length %d\n", content.Len()))
	content2.WriteString(">>\n")

	content2.WriteString("stream\n")
	content2.WriteString(content.String())
	content2.WriteString("endstream\n")

	if _, err := io.WriteString(w, content2.String()); err != nil {
		return err
	}
	return nil
}
