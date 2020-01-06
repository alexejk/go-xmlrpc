package xmlrpc

import (
	"fmt"
	"net/http"
	"net/rpc"
	"net/url"
)

type Client struct {
	*rpc.Client
}

func NewClient(endpoint string) (*Client, error) {

	return NewClientWithHttpClient(endpoint, http.DefaultClient)
}

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
