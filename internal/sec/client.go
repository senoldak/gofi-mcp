package sec

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/senoldak/gofi-mcp/internal/httpget"
)

const (
	dataURL    = "https://data.sec.gov"
	tickersURL = "https://www.sec.gov"
	// tickerCacheTTL bounds how often we re-download the SEC ticker map
	// (~2 MB). SEC refreshes the file daily; 24h is a comfortable margin.
	tickerCacheTTL = 24 * time.Hour
)

type tickerEntry struct {
	cik uint32
}

type Client struct {
	inner     *httpget.Client
	tickers   *httpget.Client
	userAgent string

	tickerMu    sync.RWMutex
	tickerCache map[string]tickerEntry
	tickerAt    time.Time
}

func New(userAgent string) *Client {
	inner := httpget.New(dataURL)
	inner.Header.Set("User-Agent", userAgent)
	tickers := httpget.New(tickersURL)
	tickers.Header.Set("User-Agent", userAgent)
	return &Client{inner: inner, tickers: tickers, userAgent: userAgent}
}

func (c *Client) Financials(ctx context.Context, ticker string) (Financials, error) {
	cik, err := c.lookupCIK(ctx, ticker)
	if err != nil {
		return Financials{}, err
	}
	body, err := c.inner.Get(ctx, "/api/xbrl/companyfacts/CIK"+cik+".json")
	if err != nil {
		return Financials{}, fmt.Errorf("sec companyfacts: %w", err)
	}
	return normalizeFinancials(ticker, body)
}

func (c *Client) lookupCIK(ctx context.Context, ticker string) (string, error) {
	want := strings.ToUpper(ticker)
	if entry, ok := c.lookupCached(want); ok {
		return fmt.Sprintf("%010d", entry.cik), nil
	}
	if err := c.refreshTickerMap(ctx); err != nil {
		return "", err
	}
	c.tickerMu.RLock()
	defer c.tickerMu.RUnlock()
	e, ok := c.tickerCache[want]
	if !ok {
		return "", fmt.Errorf("ticker %q not found in SEC mapping", ticker)
	}
	return fmt.Sprintf("%010d", e.cik), nil
}

func (c *Client) lookupCached(want string) (tickerEntry, bool) {
	c.tickerMu.RLock()
	defer c.tickerMu.RUnlock()
	if c.tickerCache == nil || time.Since(c.tickerAt) > tickerCacheTTL {
		return tickerEntry{}, false
	}
	e, ok := c.tickerCache[want]
	return e, ok
}

func (c *Client) refreshTickerMap(ctx context.Context) error {
	tickers := c.tickers
	if tickers == nil {
		tickers = c.inner
	}
	body, err := tickers.Get(ctx, "/files/company_tickers.json")
	if err != nil {
		return fmt.Errorf("sec tickers: %w", err)
	}
	var m map[string]struct {
		CIK    uint32 `json:"cik_str"`
		Ticker string `json:"ticker"`
	}
	if err := json.Unmarshal(body, &m); err != nil {
		return fmt.Errorf("sec tickers decode: %w", err)
	}
	cache := make(map[string]tickerEntry, len(m))
	for _, e := range m {
		cache[strings.ToUpper(e.Ticker)] = tickerEntry{cik: e.CIK}
	}
	c.tickerMu.Lock()
	c.tickerCache = cache
	c.tickerAt = time.Now()
	c.tickerMu.Unlock()
	return nil
}
