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
}

// NewClient creates a Client with http.DefaultClient.
// If provided endpoint is not valid, an error is returned.
func NewClient(endpoint string) (*Client, error) {

	return NewClientWithHttpClient(endpoint, http.DefaultClient)
}

// NewClientWithHttpClient allows customization of http.Client used to make RPC calls.
// If provided endpoint is not valid, an error is returned.
func NewClientWithHttpClient(endpoint string, httpClient *http.Client) (*Client, error) {

	// Parse Endpoint URL
	endpointUrl, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint url: %w", err)
	}

	codec := NewCodec(endpointUrl, httpClient)

	c := &Client{
		Client: rpc.NewClientWithCodec(codec),
	}

	return c, nil
}
