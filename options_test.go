package xmlrpc

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

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
