package fred

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/senoldak/gofi-mcp/internal/httpget"
)

const baseURL = "https://api.stlouisfed.org/fred"

type Observation struct {
	Date  string  `json:"date"`
	Value float64 `json:"value"`
}

type Series struct {
	Series       string        `json:"series"`
	Observations []Observation `json:"observations"`
}

type Client struct {
	inner *httpget.Client
	key   string
}

func New(apiKey string) *Client {
	return &Client{inner: httpget.New(baseURL), key: apiKey}
}

func (c *Client) Series(ctx context.Context, seriesID string) (Series, error) {
	path := "/series/observations?series_id=" + url.QueryEscape(seriesID) +
		"&api_key=" + url.QueryEscape(c.key) + "&file_type=json"
	body, err := c.inner.Get(ctx, path)
	if err != nil {
		return Series{}, fmt.Errorf("fred request: %w", err)
	}
	var resp struct {
		ErrorMessage string `json:"error_message"`
		Observations []struct {
			Date  string `json:"date"`
			Value string `json:"value"`
		} `json:"observations"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return Series{}, fmt.Errorf("fred decode: %w", err)
	}
	if resp.ErrorMessage != "" {
		return Series{}, fmt.Errorf("fred: %s", resp.ErrorMessage)
	}
	s := Series{Series: seriesID}
	for _, o := range resp.Observations {
		if o.Value == "." {
			continue
		}
		v, err := strconv.ParseFloat(o.Value, 64)
		if err != nil {
			continue
		}
		s.Observations = append(s.Observations, Observation{Date: o.Date, Value: v})
	}
	return s, nil
}
