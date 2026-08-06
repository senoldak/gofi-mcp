package sec

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/senoldak/gofi-mcp/internal/httpget"
)

const (
	dataURL    = "https://data.sec.gov"
	tickersURL = "https://www.sec.gov"
)

type Client struct {
	inner     *httpget.Client
	tickers   *httpget.Client
	userAgent string
}

func New(userAgent string) *Client {
	inner := httpget.New(dataURL)
	inner.Header.Set("User-Agent", userAgent)
	tickers := httpget.New(tickersURL)
	tickers.Header.Set("User-Agent", userAgent)
	return &Client{inner: inner, tickers: tickers, userAgent: userAgent}
}

func (c *Client) Financials(ctx context.Context, ticker string) (Financials, error) {
	c.inner.Header.Set("User-Agent", c.userAgent)
	cik, err := c.lookupCIK(ctx, ticker)
	if err != nil {
		return Financials{}, err
	}
	body, err := c.inner.Get(ctx, "/api/xbrl/companyfacts/CIK"+cik+".json")
	if err != nil {
		return Financials{}, fmt.Errorf("sec companyfacts: %w", err)
	}
	return normalizeFinancials(cik, body)
}

func (c *Client) lookupCIK(ctx context.Context, ticker string) (string, error) {
	tickers := c.tickers
	if tickers == nil {
		tickers = c.inner
	}
	tickers.Header.Set("User-Agent", c.userAgent)
	body, err := tickers.Get(ctx, "/files/company_tickers.json")
	if err != nil {
		return "", fmt.Errorf("sec tickers: %w", err)
	}
	var m map[string]struct {
		CIK    int    `json:"cik_str"`
		Ticker string `json:"ticker"`
		Title  string `json:"title"`
	}
	if err := json.Unmarshal(body, &m); err != nil {
		return "", fmt.Errorf("sec tickers decode: %w", err)
	}
	want := strings.ToUpper(ticker)
	for _, e := range m {
		if strings.ToUpper(e.Ticker) == want {
			return fmt.Sprintf("%010d", e.CIK), nil
		}
	}
	return "", fmt.Errorf("ticker %q not found in SEC mapping", ticker)
}
