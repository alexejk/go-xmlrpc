package xmlrpc

import (
	"fmt"
	"net/http"
	"net/rpc"
	"net/url"
)

// Client is responsible for making calls to RPC services with help of underlying rpc.Client.
type Client struct {
	*rpc.Client
	codec *Codec
}

// NewClient creates a Client with http.DefaultClient.
// If provided endpoint is not valid, an error is returned.
func NewClient(endpoint string) (*Client, error) {

	return NewCustomClient(endpoint, http.DefaultClient, make(map[string]string))
}

// NewCustomClient allows customization of http.Client and headers used to make RPC calls.
// If provided endpoint is not valid, an error is returned.
func NewCustomClient(endpoint string, httpClient *http.Client, headers map[string]string) (*Client, error) {

	// Parse Endpoint URL
	endpointUrl, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint url: %w", err)
	}

	codec := NewCodec(endpointUrl, httpClient, headers)

	c := &Client{
		codec:  codec,
		Client: rpc.NewClientWithCodec(codec),
	}

	return c, nil
}

// UserAgent returns currently configured User-Agent header that will be sent to remote server on every RPC call.
func (c *Client) UserAgent() string {
	return c.codec.userAgent
}

// SetUserAgent allows customization to User-Agent header.
// If set to an empty string, User-Agent header will be sent with an empty value.
func (c *Client) SetUserAgent(ua string) {
	c.codec.userAgent = ua
}
