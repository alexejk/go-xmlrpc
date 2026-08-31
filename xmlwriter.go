package xmlrpc

import (
	"encoding/xml"
	"io"
)

// xmlWriter writes XML fragments to an underlying writer, latching the first write error.
// This keeps individual writes readable while making it impossible to silently discard a
// failure: every method is a no-op once an error has occurred, and err is checked once
// the document is complete.
type xmlWriter struct {
	w   io.Writer
	err error
}

func newXMLWriter(w io.Writer) *xmlWriter {
	return &xmlWriter{w: w}
}

// raw writes markup verbatim and must only be given values controlled by this package.
// Anything originating from a caller belongs in text or element instead.
func (x *xmlWriter) raw(markup string) {
	if x.err != nil {
		return
	}

	_, x.err = io.WriteString(x.w, markup)
}

// text writes character data, escaped so that it cannot alter the document structure.
func (x *xmlWriter) text(chardata string) {
	if x.err != nil {
		return
	}

	x.err = xml.EscapeText(x.w, []byte(chardata))
}

// element writes a complete element around escaped character data.
func (x *xmlWriter) element(name, chardata string) {
	x.raw("<" + name + ">")
	x.text(chardata)
	x.raw("</" + name + ">")
}
