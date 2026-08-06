package sec

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/senoldak/gofi-mcp/internal/httpget"
)

func TestNormalizeFinancials(t *testing.T) {
	raw := []byte(`{
		"cik": 320193,
		"entityName": "APPLE INC",
		"facts": {
			"us-gaap": {
				"Revenues": {"units": {"USD": [
					{"end": "2025-09-27", "val": 391000000000, "fy": 2025, "fp": "FY", "form": "10-K"}
				]}},
				"NetIncomeLoss": {"units": {"USD": [
					{"end": "2025-09-27", "val": 93700000000, "fy": 2025, "fp": "FY", "form": "10-K"}
				]}},
				"EarningsPerShareBasic": {"units": {"USD": [
					{"end": "2025-09-27", "val": 6.1, "fy": 2025, "fp": "FY", "form": "10-K"}
				]}},
				"Assets": {"units": {"USD": [
					{"end": "2025-09-27", "val": 375000000000, "fy": 2025, "fp": "FY", "form": "10-K"}
				]}},
				"TotalLiabilities": {"units": {"USD": [
					{"end": "2025-09-27", "val": 279000000000, "fy": 2025, "fp": "FY", "form": "10-K"}
				]}},
				"NetCashProvidedByUsedInOperatingActivities": {"units": {"USD": [
					{"end": "2025-09-27", "val": 118000000000, "fy": 2025, "fp": "FY", "form": "10-K"}
				]}}
			}
		}
	}`)

	f, err := normalizeFinancials("0000320193", raw)
	if err != nil {
		t.Fatalf("normalizeFinancials error: %v", err)
	}
	if f.Ticker != "0000320193" {
		t.Fatalf("Ticker = %q", f.Ticker)
	}
	if len(f.Periods) != 1 {
		t.Fatalf("len(Periods) = %d, want 1", len(f.Periods))
	}
	p := f.Periods[0]
	if p.FiscalYear != 2025 || p.Revenue != 391000000000 || p.NetIncome != 93700000000 {
		t.Fatalf("unexpected period: %+v", p)
	}
	if p.EPS != 6.1 || p.OperatingCashFlow != 118000000000 {
		t.Fatalf("unexpected period: %+v", p)
	}
}

func TestFinancialsFetchesCompanyfacts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != "Agent/1.0 test@example.com" {
			t.Fatalf("User-Agent = %q", r.Header.Get("User-Agent"))
		}
		switch r.URL.Path {
		case "/files/company_tickers.json":
			w.Write([]byte(`{"0":{"cik_str":320193,"ticker":"AAPL","title":"Apple Inc."}}`))
		case "/api/xbrl/companyfacts/CIK0000320193.json":
			w.Write([]byte(`{"facts":{"us-gaap":{}}}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	c := &Client{inner: httpget.New(srv.URL), tickers: httpget.New(srv.URL), userAgent: "Agent/1.0 test@example.com"}
	f, err := c.Financials(context.Background(), "AAPL")
	if err != nil {
		t.Fatalf("Financials error: %v", err)
	}
	if !strings.HasPrefix(f.Ticker, "0000320193") {
		t.Fatalf("Ticker = %q", f.Ticker)
	}
}

func TestNewUsesSeparateHosts(t *testing.T) {
	c := New("Agent/1.0 test@example.com")
	if c.inner.BaseURL != "https://data.sec.gov" {
		t.Fatalf("inner BaseURL = %q, want https://data.sec.gov", c.inner.BaseURL)
	}
	if c.tickers.BaseURL != "https://www.sec.gov" {
		t.Fatalf("tickers BaseURL = %q, want https://www.sec.gov", c.tickers.BaseURL)
	}
	if c.inner.Header.Get("User-Agent") != "Agent/1.0 test@example.com" {
		t.Fatalf("inner User-Agent = %q", c.inner.Header.Get("User-Agent"))
	}
	if c.tickers.Header.Get("User-Agent") != "Agent/1.0 test@example.com" {
		t.Fatalf("tickers User-Agent = %q", c.tickers.Header.Get("User-Agent"))
	}
}
