package httpget

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

const defaultTimeout = 30 * time.Second

type Client struct {
	BaseURL string
	Header  http.Header
	Timeout time.Duration

	httpOnce sync.Once
	http     *http.Client
}

func New(baseURL string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Header:  http.Header{},
		Timeout: defaultTimeout,
	}
}

func (c *Client) httpClient() *http.Client {
	c.httpOnce.Do(func() {
		c.http = &http.Client{
			Timeout: c.Timeout,
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
			},
		}
	})
	return c.http
}

func (c *Client) Get(ctx context.Context, path string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	for k, vs := range c.Header {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "application/json")
	}
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}
