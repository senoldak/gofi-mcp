package goficlient

import (
	"context"

	"github.com/senoldak/gofi-mcp/internal/httpget"
)

type Fetcher interface {
	Get(ctx context.Context, path string) ([]byte, error)
}

type Client struct {
	inner *httpget.Client
}

func New(baseURL string) *Client {
	return &Client{inner: httpget.New(baseURL)}
}

func (c *Client) Get(ctx context.Context, path string) ([]byte, error) {
	return c.inner.Get(ctx, path)
}
