package xmlrpc

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClient_Call(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		m := &struct {
			Name   string       `xml:"methodName"`
			Params []*respParam `xml:"params>param"`
		}{}
		body, err := ioutil.ReadAll(r.Body)
		assert.NoError(t, err, "test server: read body")

		err = xml.Unmarshal(body, m)
		assert.NoError(t, err, "test server: unmarshal body")

		nameParts := strings.Split(m.Name, ".")
		assert.Equal(t, 2, len(nameParts))
		assert.Equal(t, "my", nameParts[0], "test server: method should start with 'my.'")
		assert.Equal(t, 1, len(m.Params))
		assert.Equal(t, "12345", m.Params[0].Value.Int)

		file := nameParts[1]
		fmt.Fprintln(w, string(loadTestFile(t, fmt.Sprintf("response_%s.xml", file))))
	}))
	defer ts.Close()

	c, err := NewClient(ts.URL)
	assert.NoError(t, err)
	assert.NotNil(t, c)

	req := &struct {
		Index int
	}{
		Index: 12345,
	}
	resp := &struct {
		Area  string
		Index int
	}{}

	err = c.Call("my.simple", req, resp)
	assert.NoError(t, err)
	assert.Equal(t, "South Dakota", resp.Area)
	assert.Equal(t, 12345, resp.Index)
}

func TestClient_Fault(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, string(loadTestFile(t, "response_fault.xml")))
	}))
	defer ts.Close()

	c, err := NewClient(ts.URL)
	assert.NoError(t, err)
	assert.NotNil(t, c)

	req := &struct{}{}
	resp := &struct{}{}

	err = c.Call("my.fault", req, resp)
	assert.Error(t, err)
}

func TestClient_Bugzilla(t *testing.T) {

	c, err := NewClient("https://bugzilla.mozilla.org/xmlrpc.cgi")
	assert.NoError(t, err)
	assert.NotNil(t, c)

	resp := &struct {
		BugzillaVersion struct {
			Version string
		}
	}{}

	err = c.Call("Bugzilla.version", nil, resp)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.BugzillaVersion.Version)
}
