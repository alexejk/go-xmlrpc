package xmlrpc

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	c, err := NewClient(":8080/rpc")

	require.Error(t, err)
	require.Nil(t, c)

	c, err = NewClient("http://localhost")
	require.NoError(t, err)
	require.NotNil(t, c)
}

func TestClient_Call(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m := &struct {
			Name   string           `xml:"methodName"`
			Params []*ResponseParam `xml:"params>param"`
		}{}
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err, "test server: read body")

		err = xml.Unmarshal(body, m)
		require.NoError(t, err, "test server: unmarshal body")

		nameParts := strings.Split(m.Name, ".")
		require.Equal(t, 2, len(nameParts))
		require.Equal(t, "my", nameParts[0], "test server: method should start with 'my.'")
		require.Equal(t, 1, len(m.Params))
		require.Equal(t, "12345", *m.Params[0].Value.Int)

		file := nameParts[1]
		_, _ = fmt.Fprintln(w, string(loadTestFile(t, fmt.Sprintf("response_%s.xml", file))))
	}))
	defer ts.Close()

	c, err := NewClient(ts.URL)
	require.NoError(t, err)
	require.NotNil(t, c)

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
	require.NoError(t, err)
	require.Equal(t, "South Dakota", resp.Area)
	require.Equal(t, 12345, resp.Index)
}

func TestClient_Fault(t *testing.T) {
	ts := mockupServer(t, "response_fault.xml")
	defer ts.Close()

	c, err := NewClient(ts.URL)
	require.NoError(t, err)
	require.NotNil(t, c)

	req := &struct{}{}
	resp := &struct{}{}

	err = c.Call("my.fault", req, resp)
	require.Error(t, err)
}

func TestClient_Bugzilla(t *testing.T) {
	ts := mockupServer(t, "response_bugzilla_version.xml")
	defer ts.Close()

	c, err := NewClient(ts.URL)
	require.NoError(t, err)
	require.NotNil(t, c)

	resp := &struct {
		BugzillaVersion struct {
			Version string
		}
	}{}

	err = c.Call("Bugzilla.version", nil, resp)
	require.NoError(t, err)
	require.NotEmpty(t, resp.BugzillaVersion.Version)
	require.Equal(t, "20220802.1", resp.BugzillaVersion.Version)
}

// Checks Issue 52 (https://github.com/alexejk/go-xmlrpc/issues/52)
// Makes several calls to ensure there is no request-response confusion caused by the changes
// Test must have a small delay in order to compare go-routines (due to defers)
func TestClient_GoRoutineLeak_Issue52(t *testing.T) {
	ts := mockupBugzillaVersionServer(t)
	var request = func() {
		client, err := NewClient(ts.URL)
		if err != nil {
			require.FailNow(t, "panic when generating clients")
		}
		defer client.Close()

		result := struct {
			BugzillaVersion struct {
				Version string
			}
		}{}

		// Make several calls to the server
		for i := 1; i < 5; i++ {
			err = client.Call(fmt.Sprintf("Bugzilla.version.%d", i), nil, &result)
			require.NoError(t, err)
			require.Equal(t, fmt.Sprintf("20220802.%d", i), result.BugzillaVersion.Version)
		}
	}

	// Baseline (BEFORE)
	preTestRoutines := runtime.NumGoroutine()

	for i := 0; i < 5; i++ {
		fmt.Printf("...Request %d\n", i)
		request()
	}

	// Allow for all things to close down properly
	time.Sleep(1 * time.Second)

	// Result (AFTER)
	postTestRoutines := runtime.NumGoroutine()

	// Ensure that the amount of go routines AFTER <= BEFORE (meaning - we did not leak any)
	require.LessOrEqual(t, postTestRoutines, preTestRoutines)
}

func mockupServer(t *testing.T, respFile string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "text/xml", r.Header.Get("Content-Type"))
		w.WriteHeader(200)
		_, _ = fmt.Fprint(w, string(loadTestFile(t, respFile)))
	}))
}

// mockupBugzillaVersionServer returns a test server that is able to parse the method name and inject the last part of
// the method name into response version. This follows Bugzilla response format, however does some modifications to make
// requests more dynamic and comparable
func mockupBugzillaVersionServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "text/xml", r.Header.Get("Content-Type"))

		m := &struct {
			Name string `xml:"methodName"`
		}{}
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err, "test server: read body")

		err = xml.Unmarshal(body, m)
		require.NoError(t, err, "test server: unmarshal body")

		nameParts := strings.Split(m.Name, ".")
		num := nameParts[2] // Bugzilla.Version.%d

		w.WriteHeader(200)
		_, _ = fmt.Fprintf(w, `
<?xml version="1.0" encoding="UTF-8"?>
<methodResponse>
    <params>
        <param>
            <value>
                <struct>
                    <member>
                        <name>version</name>
                        <value>
                            <string>20220802.%s</string>
                        </value>
                    </member>
                </struct>
            </value>
        </param>
    </params>
</methodResponse>
`, num)
	}))
}
