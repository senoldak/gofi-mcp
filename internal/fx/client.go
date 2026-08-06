package fx

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/senoldak/gofi-mcp/internal/httpget"
)

const baseURL = "https://api.frankfurter.app"

type Rate struct {
	From string  `json:"from"`
	To   string  `json:"to"`
	Rate float64 `json:"rate"`
	Date string  `json:"date"`
}

type Client struct {
	inner *httpget.Client
}

func New() *Client {
	return &Client{inner: httpget.New(baseURL)}
}

func (c *Client) Rate(ctx context.Context, from, to string) (Rate, error) {
	path := "/latest?from=" + url.QueryEscape(from) + "&to=" + url.QueryEscape(to)
	body, err := c.inner.Get(ctx, path)
	if err != nil {
		return Rate{}, fmt.Errorf("fx request: %w", err)
	}
	var resp struct {
		Date  string             `json:"date"`
		Base  string             `json:"base"`
		Rates map[string]float64 `json:"rates"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return Rate{}, fmt.Errorf("fx decode: %w", err)
	}
	rate, ok := resp.Rates[to]
	if !ok {
		return Rate{}, fmt.Errorf("no rate for %s in response", to)
	}
	return Rate{From: from, To: to, Rate: rate, Date: resp.Date}, nil
}
