package xmlrpc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/rpc"
	"net/url"
	"sync"
)

const defaultUserAgent = "alexejk.io/go-xmlrpc"

// Codec implements methods required by rpc.ClientCodec
// In this implementation Codec is the one performing actual RPC requests with http.Client.
type Codec struct {
	endpoint      *url.URL
	httpClient    *http.Client
	customHeaders map[string]string

	mutex sync.Mutex
	// contains completed but not processed responses by sequence ID
	pending map[uint64]*rpcCall

	// Current in-flight response
	response *Response
	encoder  Encoder
	decoder  Decoder

	// presents completed requests by sequence ID
	ready chan uint64

	userAgent string
	shutdown  chan struct{}
}

type rpcCall struct {
	Seq           uint64
	ServiceMethod string
	httpResponse  *http.Response
}

// NewCodec creates a new Codec bound to provided endpoint.
// Provided client will be used to perform RPC requests.
func NewCodec(endpoint *url.URL, httpClient *http.Client) *Codec {
	return &Codec{
		endpoint:   endpoint,
		httpClient: httpClient,
		encoder:    &StdEncoder{},
		decoder:    &StdDecoder{},

		pending:  make(map[uint64]*rpcCall),
		response: nil,
		ready:    make(chan uint64),

		userAgent: defaultUserAgent,
		shutdown:  make(chan struct{}),
	}
}

// SetEncoder allows setting a new Encoder on the codec
func (c *Codec) SetEncoder(encoder Encoder) {
	c.encoder = encoder
}

// SetDecoder allows setting a new Decoder on the codec
func (c *Codec) SetDecoder(decoder Decoder) {
	c.decoder = decoder
}

func (c *Codec) WriteRequest(req *rpc.Request, args interface{}) error {
	bodyBuffer := new(bytes.Buffer)
	err := c.encoder.Encode(bodyBuffer, req.ServiceMethod, args)
	if err != nil {
		return err
	}

	httpRequest, err := http.NewRequestWithContext(context.TODO(), "POST", c.endpoint.String(), bodyBuffer)
	if err != nil {
		return err
	}

	httpRequest.Header.Set("Content-Type", "text/xml")
	httpRequest.Header.Set("User-Agent", c.userAgent)

	// Apply customer headers if set, this allows overwriting static default headers
	for key, value := range c.customHeaders {
		httpRequest.Header.Set(key, value)
	}

	httpRequest.Header.Set("Content-Length", fmt.Sprintf("%d", bodyBuffer.Len()))

	httpResponse, err := c.httpClient.Do(httpRequest) //nolint:bodyclose // Handled in ReadResponseHeader
	if err != nil {
		return err
	}

	c.mutex.Lock()
	c.pending[req.Seq] = &rpcCall{
		Seq:           req.Seq,
		ServiceMethod: req.ServiceMethod,
		httpResponse:  httpResponse,
	}
	c.mutex.Unlock()

	c.ready <- req.Seq

	return nil
}

func (c *Codec) ReadResponseHeader(resp *rpc.Response) error {
	select {
	case seq := <-c.ready:
		// Handle request that is ready
		c.mutex.Lock()
		call := c.pending[seq]
		delete(c.pending, seq)
		c.mutex.Unlock()

		resp.Seq = call.Seq
		resp.ServiceMethod = call.ServiceMethod

		r := call.httpResponse

		defer r.Body.Close()

		if r.StatusCode < 200 || r.StatusCode >= 300 {
			resp.Error = fmt.Sprintf("bad response code: %d", r.StatusCode)
			return nil
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			resp.Error = err.Error()
			return nil
		}

		decodableResponse, err := NewResponse(body)
		if err != nil {
			resp.Error = err.Error()
			return nil
		}

		// Return response Fault already at this stage
		if err := c.decoder.DecodeFault(decodableResponse); err != nil {
			resp.Error = err.Error()
			return nil
		}

		c.response = decodableResponse
		return nil

	case <-c.shutdown:
		// Handle shutdown signal
		return net.ErrClosed
	}
}
func (c *Codec) ReadResponseBody(v interface{}) error {
	if v == nil {
		return nil
	}

	if c.response == nil {
		return errors.New("no in-flight response found")
	}

	return c.decoder.Decode(c.response, v)
}

func (c *Codec) Close() error {
	c.shutdown <- struct{}{}
	c.httpClient.CloseIdleConnections()
	return nil
}
