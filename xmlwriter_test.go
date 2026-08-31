package xmlrpc

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_xmlWriter(t *testing.T) {
	t.Run("element escapes character data", func(t *testing.T) {
		buf := new(strings.Builder)
		x := newXMLWriter(buf)
		x.element("name", `a&b<c>d"e`)

		require.NoError(t, x.err)
		require.Equal(t, "<name>a&amp;b&lt;c&gt;d&#34;e</name>", buf.String())
	})

	t.Run("raw is written verbatim", func(t *testing.T) {
		buf := new(strings.Builder)
		x := newXMLWriter(buf)
		x.raw("<struct>")

		require.NoError(t, x.err)
		require.Equal(t, "<struct>", buf.String())
	})

	t.Run("first error is latched and later writes are dropped", func(t *testing.T) {
		w := &failingWriter{limit: 4}
		x := newXMLWriter(w)

		x.raw("<ab>")   // fits
		x.raw("<cdef>") // fails
		first := x.err
		require.Error(t, first)

		x.text("more")
		x.element("name", "value")

		require.Same(t, first, x.err, "the first error must be retained")
		require.Equal(t, 4, w.written, "no bytes may be written after a failure")
	})
}
