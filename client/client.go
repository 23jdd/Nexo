// Package client provides a small HTTP/1.1 client built on Nexo's protocol
// implementation.
package client

import (
	"fmt"
	"io"
	"net/url"

	"github.com/23jdd/Nexo/protocol"
	"github.com/23jdd/Nexo/protocol/http1"
)

type Transport interface {
	RoundTrip(*http1.Request) (*http1.Response, error)
}

type Client struct {
	Transport Transport
}

func New() *Client {
	return &Client{Transport: NewHTTP1Transport()}
}

func (c *Client) Do(req *http1.Request) (*http1.Response, error) {
	if c == nil {
		return nil, fmt.Errorf("nil client")
	}
	transport := c.Transport
	if transport == nil {
		transport = NewHTTP1Transport()
		c.Transport = transport
	}
	return transport.RoundTrip(req)
}

func (c *Client) Get(rawURL string) (*http1.Response, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	return c.Do(&http1.Request{
		Protocol: "HTTP/1.1",
		Method:   string(protocol.Get),
		URL:      u,
		Header:   make(protocol.Header),
		Body:     io.NopCloser(&emptyReader{}),
	})
}

type emptyReader struct{}

func (*emptyReader) Read([]byte) (int, error) { return 0, io.EOF }
