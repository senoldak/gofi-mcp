package sec

import (
	"context"
	"net/http"
	"net/http/httptest"
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

func TestFinancialsReturnsTickerNotCIK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/files/company_tickers.json":
			w.Write([]byte(`{"0":{"cik_str":320193,"ticker":"AAPL","title":"Apple Inc."}}`))
		case "/api/xbrl/companyfacts/CIK0000320193.json":
			w.Write([]byte(`{"facts":{"us-gaap":{}}}`))
		}
	}))
	defer srv.Close()

	c := &Client{inner: httpget.New(srv.URL), tickers: httpget.New(srv.URL), userAgent: "Agent/1.0 test@example.com"}
	f, err := c.Financials(context.Background(), "AAPL")
	if err != nil {
		t.Fatalf("Financials error: %v", err)
	}
	if f.Ticker != "AAPL" {
		t.Fatalf("Ticker = %q, want AAPL (not CIK)", f.Ticker)
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
	c.inner.Header.Set("User-Agent", "Agent/1.0 test@example.com")
	c.tickers.Header.Set("User-Agent", "Agent/1.0 test@example.com")
	f, err := c.Financials(context.Background(), "AAPL")
	if err != nil {
		t.Fatalf("Financials error: %v", err)
	}
	if f.Ticker != "AAPL" {
		t.Fatalf("Ticker = %q, want AAPL", f.Ticker)
	}
}

func TestNormalizeFallsThroughTagWithNoAnnualRecords(t *testing.T) {
	raw := []byte(`{
		"cik": 320193,
		"entityName": "APPLE INC",
		"facts": {
			"us-gaap": {
				"RevenueFromContractWithCustomerExcludingAssessedTax": {"units": {"USD": [
					{"end": "2025-06-28", "val": 94000, "fy": 2025, "fp": "Q3", "form": "10-Q"}
				]}},
				"Revenues": {"units": {"USD": [
					{"end": "2024-09-28", "val": 391000000000, "fy": 2024, "fp": "FY", "form": "10-K"}
				]}}
			}
		}
	}`)
	f, err := normalizeFinancials("0000320193", raw)
	if err != nil {
		t.Fatalf("normalizeFinancials error: %v", err)
	}
	if len(f.Periods) != 1 {
		t.Fatalf("len(Periods) = %d, want 1", len(f.Periods))
	}
	if f.Periods[0].Revenue != 391000000000 {
		t.Fatalf("Revenue = %v, want 391000000000 (fell through to Revenues)", f.Periods[0].Revenue)
	}
}

func TestNormalizeUsesExplicitUSDSharesFallback(t *testing.T) {
	raw := []byte(`{
		"cik": 320193,
		"entityName": "APPLE INC",
		"facts": {
			"us-gaap": {
				"EarningsPerShareBasic": {"units": {"XYZ": [
					{"end": "2025-09-27", "val": 6.1, "fy": 2025, "fp": "FY", "form": "10-K"}
				]}}
			}
		}
	}`)
	f, err := normalizeFinancials("0000320193", raw)
	if err != nil {
		t.Fatalf("normalizeFinancials error: %v", err)
	}
	if len(f.Periods) != 1 {
		t.Fatalf("len(Periods) = %d, want 1", len(f.Periods))
	}
	if f.Periods[0].EPS != 6.1 {
		t.Fatalf("EPS = %v, want 6.1 (fell through USD/shares to XYZ)", f.Periods[0].EPS)
	}
}

func TestNormalizeRealWorldTagShapes(t *testing.T) {
	raw := []byte(`{
		"cik": 320193,
		"entityName": "APPLE INC",
		"facts": {
			"us-gaap": {
				"RevenueFromContractWithCustomerExcludingAssessedTax": {"units": {"USD": [
					{"end": "2025-09-27", "val": 416161000000, "fy": 2025, "fp": "FY", "form": "10-K"}
				]}},
				"NetIncomeLoss": {"units": {"USD": [
					{"end": "2025-09-27", "val": 93700000000, "fy": 2025, "fp": "FY", "form": "10-K"}
				]}},
				"EarningsPerShareBasic": {"units": {"USD/shares": [
					{"end": "2025-09-27", "val": 6.1, "fy": 2025, "fp": "FY", "form": "10-K"}
				]}},
				"Assets": {"units": {"USD": [
					{"end": "2025-09-27", "val": 375000000000, "fy": 2025, "fp": "FY", "form": "10-K"}
				]}},
				"Liabilities": {"units": {"USD": [
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
	if len(f.Periods) != 1 {
		t.Fatalf("len(Periods) = %d, want 1", len(f.Periods))
	}
	p := f.Periods[0]
	if p.Revenue != 416161000000 {
		t.Fatalf("Revenue = %v, want 416161000000 (ASC 606 tag)", p.Revenue)
	}
	if p.TotalLiabilities != 279000000000 {
		t.Fatalf("TotalLiabilities = %v, want 279000000000 (Liabilities tag)", p.TotalLiabilities)
	}
	if p.EPS != 6.1 {
		t.Fatalf("EPS = %v, want 6.1 (USD/shares unit)", p.EPS)
	}
}

func TestTickerMapIsCached(t *testing.T) {
	var tickerHits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/files/company_tickers.json":
			tickerHits++
			w.Write([]byte(`{"0":{"cik_str":320193,"ticker":"AAPL","title":"Apple Inc."},"1":{"cik_str":789019,"ticker":"MSFT","title":"Microsoft"}}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	c := &Client{inner: httpget.New(srv.URL), tickers: httpget.New(srv.URL), userAgent: "Agent/1.0 test@example.com"}

	for i := 0; i < 5; i++ {
		cik, err := c.lookupCIK(context.Background(), "AAPL")
		if err != nil {
			t.Fatalf("lookupCIK error: %v", err)
		}
		if cik != "0000320193" {
			t.Fatalf("cik = %q", cik)
		}
	}
	if tickerHits != 1 {
		t.Fatalf("ticker map fetch count = %d, want 1 (should cache)", tickerHits)
	}

	cik, err := c.lookupCIK(context.Background(), "MSFT")
	if err != nil {
		t.Fatalf("lookupCIK MSFT error: %v", err)
	}
	if cik != "0000789019" {
		t.Fatalf("MSFT cik = %q", cik)
	}
	if tickerHits != 1 {
		t.Fatalf("ticker map fetch count after MSFT = %d, want 1 (cached)", tickerHits)
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
