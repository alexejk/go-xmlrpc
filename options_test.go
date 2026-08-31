package xmlrpc

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestClient_Option_Headers(t *testing.T) {
	tests := []struct {
		name   string
		opts   []Option
		expect http.Header
	}{
		{
			name: "default",
			expect: http.Header{
				"Accept-Encoding": []string{"gzip"},
				"User-Agent":      []string{defaultUserAgent},
				"Content-Length":  []string{"61"},
				"Content-Type":    []string{"text/xml"},
			},
		},
		{
			name: "header addition",
			opts: []Option{
				Headers(map[string]string{
					"X-Header": "my-value",
				}),
			},
			expect: http.Header{
				"Accept-Encoding": []string{"gzip"},
				"User-Agent":      []string{defaultUserAgent},
				"Content-Length":  []string{"61"},
				"Content-Type":    []string{"text/xml"},
				"X-Header":        []string{"my-value"},
			},
		},
		{
			name: "header replacement",
			opts: []Option{
				Headers(map[string]string{
					"Content-Type": "text/xml+custom",
					"X-Header":     "my-value",
				}),
			},
			expect: http.Header{
				"Accept-Encoding": []string{"gzip"},
				"User-Agent":      []string{defaultUserAgent},
				"Content-Length":  []string{"61"},
				"Content-Type":    []string{"text/xml+custom"},
				"X-Header":        []string{"my-value"},
			},
		},
		{
			name: "content-length not replaced",
			opts: []Option{
				Headers(map[string]string{
					"Content-Length": "999999",
					"X-Header":       "my-value",
				}),
			},
			expect: http.Header{
				"Accept-Encoding": []string{"gzip"},
				"User-Agent":      []string{defaultUserAgent},
				"Content-Length":  []string{"61"},
				"Content-Type":    []string{"text/xml"},
				"X-Header":        []string{"my-value"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serverCalled := false
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.EqualValues(t, tt.expect, r.Header)

				serverCalled = true
				_, _ = fmt.Fprintln(w, string(loadTestFile(t, "response_simple.xml")))
			}))
			defer ts.Close()

			c, err := NewClient(ts.URL, tt.opts...)
			require.NoError(t, err)

			err = c.Call("test.Method", nil, nil)
			require.NoError(t, err)

			require.True(t, serverCalled, "server must be called")
		})
	}
}

func TestClient_Option_UserAgent(t *testing.T) {
	tests := []struct {
		name   string
		opts   []Option
		expect string
	}{
		{
			name:   "default user-agent",
			expect: defaultUserAgent,
		},
		{
			name: "new user-agent",
			opts: []Option{
				UserAgent("my-new-agent/1.2.3"),
			},
			expect: "my-new-agent/1.2.3",
		},
		{
			name: "empty user-agent",
			opts: []Option{
				UserAgent(""),
			},
			expect: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serverCalled := false
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ua := r.UserAgent()

				require.Equal(t, tt.expect, ua)

				serverCalled = true
				_, _ = fmt.Fprintln(w, string(loadTestFile(t, "response_simple.xml")))
			}))
			defer ts.Close()

			c, err := NewClient(ts.URL, tt.opts...)
			require.NoError(t, err)

			err = c.Call("test.Method", nil, nil)
			require.NoError(t, err)

			require.True(t, serverCalled, "server must be called")
		})
	}
}

type RoundTripFunc func(req *http.Request) *http.Response

func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

func TestClient_Option_HttpClient(t *testing.T) {
	tests := []struct {
		name             string
		opts             []Option
		expectServerCall bool
	}{
		{
			name:             "default client",
			expectServerCall: true,
		},
		{
			name: "customized client",
			opts: []Option{
				HttpClient(&http.Client{
					Transport: RoundTripFunc(func(req *http.Request) *http.Response {
						return &http.Response{
							StatusCode: 200,
							Body:       io.NopCloser(bytes.NewBuffer(loadTestFile(t, "response_simple.xml"))),
							Header:     map[string][]string{},
						}
					}),
				}),
			},
			expectServerCall: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serverCalled := false
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				serverCalled = true
				_, _ = fmt.Fprintln(w, string(loadTestFile(t, "response_simple.xml")))
			}))
			defer ts.Close()

			c, err := NewClient(ts.URL, tt.opts...)
			require.NoError(t, err)

			err = c.Call("test.Method", nil, nil)
			require.NoError(t, err)

			require.Equal(t, tt.expectServerCall, serverCalled)
		})
	}
}

func TestClient_Option_SkipUnknownFields(t *testing.T) {
	tests := []struct {
		name   string
		opts   []Option
		expect bool
	}{
		{
			name:   "default setting",
			expect: false,
		},
		{
			name: "new setting - false",
			opts: []Option{
				SkipUnknownFields(false),
			},
			expect: false,
		},
		{
			name: "new setting - false",
			opts: []Option{
				SkipUnknownFields(true),
			},
			expect: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serverCalled := false
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				serverCalled = true
				_, _ = fmt.Fprintln(w, string(loadTestFile(t, "response_bugzilla_version.xml")))
			}))
			defer ts.Close()

			c, err := NewClient(ts.URL, tt.opts...)
			require.NoError(t, err)

			v := &struct {
				Bugzilla struct {
				}
			}{}

			err = c.Call("test.Method", nil, v)
			if tt.expect {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}

			require.True(t, serverCalled, "server must be called")
		})
	}
}

// recordingTimeFormatter is a third-party TimeFormatter, proving a single instance is
// used for both directions and that its output is escaped on the way out.
type recordingTimeFormatter struct {
	formatted int
	parsed    int
}

func (f *recordingTimeFormatter) FormatTime(_ time.Time) string {
	f.formatted++

	return "encoded&value"
}

func (f *recordingTimeFormatter) ParseTime(_ string) (time.Time, error) {
	f.parsed++

	return time.Date(2001, 2, 3, 4, 5, 6, 0, time.UTC), nil
}

func TestClient_Option_TimeFormat_CustomFormatter(t *testing.T) {
	formatter := &recordingTimeFormatter{}

	var body string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		body = string(b)

		_, _ = fmt.Fprintln(w, string(loadTestFile(t, "response_datetime_compact.xml")))
	}))
	defer ts.Close()

	c, err := NewClient(ts.URL, TimeFormat(formatter))
	require.NoError(t, err)

	resp := &struct{ When time.Time }{}
	require.NoError(t, c.Call("test.Method", &struct{ When time.Time }{When: time.Now()}, resp))

	require.Equal(t, 1, formatter.formatted, "the same instance must encode")
	require.Equal(t, 1, formatter.parsed, "the same instance must decode")
	require.Equal(t, 2001, resp.When.Year())

	// formatter output must be escaped, keeping the request well-formed
	require.Contains(t, body, "<dateTime.iso8601>encoded&amp;value</dateTime.iso8601>")
	require.NoError(t, xml.Unmarshal([]byte(body), new(struct{})), "request body must be well-formed XML")
}

func TestClient_Option_TimeFormat_TypedNilFormatter(t *testing.T) {
	// A typed nil is non-nil as an interface - it must not panic
	var formatter *LayoutTimeFormatter

	serverCalled := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.Contains(t, string(body), "<dateTime.iso8601>2019-10-11T13:40:30Z</dateTime.iso8601>")

		serverCalled = true
		_, _ = fmt.Fprintln(w, string(loadTestFile(t, "response_datetime_rfc3339.xml")))
	}))
	defer ts.Close()

	c, err := NewClient(ts.URL, TimeFormat(formatter))
	require.NoError(t, err)

	args := &struct{ When time.Time }{When: time.Date(2019, 10, 11, 13, 40, 30, 0, time.UTC)}
	resp := &struct{ When time.Time }{}
	require.NoError(t, c.Call("test.Method", args, resp))

	// decode must go through the same fallback
	require.Equal(t, "2019-10-11T13:40:30Z", resp.When.Format(time.RFC3339))
	require.True(t, serverCalled, "server must be called")
}

func TestClient_Option_TimeFormat(t *testing.T) {
	input := time.Date(2019, 10, 11, 13, 40, 30, 0, time.UTC)

	tests := []struct {
		name         string
		opts         []Option
		expectEncode string
		expectDecode string
		expectErr    bool
	}{
		{
			name:         "default formatter",
			expectEncode: "<dateTime.iso8601>2019-10-11T13:40:30Z</dateTime.iso8601>",
			// Server responds with the compact form, which RFC3339 cannot parse
			expectErr: true,
		},
		{
			name: "nil formatter falls back to default",
			opts: []Option{
				TimeFormat(nil),
			},
			expectEncode: "<dateTime.iso8601>2019-10-11T13:40:30Z</dateTime.iso8601>",
			expectErr:    true,
		},
		{
			name: "compact formatter",
			opts: []Option{
				TimeFormat(&LayoutTimeFormatter{FormatLayout: LayoutISO8601Compact}),
			},
			expectEncode: "<dateTime.iso8601>20191011T13:40:30</dateTime.iso8601>",
			expectDecode: "2019-10-11T13:40:30Z",
		},
		{
			name: "permissive parse layouts with compact encoding",
			opts: []Option{
				TimeFormat(&LayoutTimeFormatter{
					FormatLayout: LayoutISO8601Compact,
					ParseLayouts: CommonParseLayouts(),
				}),
			},
			expectEncode: "<dateTime.iso8601>20191011T13:40:30</dateTime.iso8601>",
			expectDecode: "2019-10-11T13:40:30Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serverCalled := false
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				require.Contains(t, string(body), tt.expectEncode)

				serverCalled = true
				_, _ = fmt.Fprintln(w, string(loadTestFile(t, "response_datetime_compact.xml")))
			}))
			defer ts.Close()

			c, err := NewClient(ts.URL, tt.opts...)
			require.NoError(t, err)

			args := &struct{ When time.Time }{When: input}
			resp := &struct{ When time.Time }{}

			err = c.Call("test.Method", args, resp)
			if tt.expectErr {
				require.ErrorContains(t, err, "does not match expected time layouts")
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectDecode, resp.When.Format(time.RFC3339))
			}

			require.True(t, serverCalled, "server must be called")
		})
	}
}
