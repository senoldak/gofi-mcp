package coingecko

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/senoldak/gofi-mcp/internal/httpget"
)

const baseURL = "https://api.coingecko.com/api/v3"

type Price struct {
	ID           string  `json:"id"`
	Symbol       string  `json:"symbol"`
	Name         string  `json:"name"`
	CurrentPrice float64 `json:"current_price"`
	MarketCap    float64 `json:"market_cap"`
	Change24h    float64 `json:"price_change_percentage_24h"`
}

type Client struct {
	inner *httpget.Client
}

func New(apiKey string) *Client {
	c := httpget.New(baseURL)
	if apiKey != "" {
		c.Header.Set("x-cg-pro-api-key", apiKey)
	}
	return &Client{inner: c}
}

func (c *Client) Price(ctx context.Context, id string) (Price, error) {
	path := "/coins/markets?vs_currency=usd&ids=" + url.QueryEscape(id) + "&per_page=1"
	body, err := c.inner.Get(ctx, path)
	if err != nil {
		return Price{}, fmt.Errorf("coingecko price: %w", err)
	}
	var list []Price
	if err := json.Unmarshal(body, &list); err != nil {
		return Price{}, fmt.Errorf("coingecko decode: %w", err)
	}
	if len(list) == 0 {
		return Price{}, fmt.Errorf("no coin found for id %q", id)
	}
	return list[0], nil
}

var marketOrders = map[string]string{
	"most-active": "market_cap_desc",
	"gainers":     "gecko_desc",
	"volume":      "volume_desc",
}

func (c *Client) Market(ctx context.Context, category string) ([]Price, error) {
	order := marketOrders[category]
	if order == "" {
		order = marketOrders["most-active"]
	}
	path := "/coins/markets?vs_currency=usd&per_page=20&order=" + url.QueryEscape(order)
	body, err := c.inner.Get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("coingecko market: %w", err)
	}
	var list []Price
	if err := json.Unmarshal(body, &list); err != nil {
		return nil, fmt.Errorf("coingecko decode: %w", err)
	}
	return list, nil
}
