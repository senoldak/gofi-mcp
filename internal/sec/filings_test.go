package sec

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/senoldak/gofi-mcp/internal/httpget"
)

func TestFilingsParsesRecent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/files/company_tickers.json":
			w.Write([]byte(`{"0":{"cik_str":320193,"ticker":"AAPL","title":"Apple Inc."}}`))
		case "/submissions/CIK0000320193.json":
			w.Write([]byte(`{"filings":{"recent":{
				"form":["10-K","8-K"],
				"filingDate":["2025-10-31","2025-08-01"],
				"reportDate":["2025-09-27",""],
				"accessionNumber":["0000320193-25-000095","0000320193-25-000088"]
			}}}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	c := &Client{inner: httpget.New(srv.URL)}
	filings, err := c.Filings(context.Background(), "AAPL")
	if err != nil {
		t.Fatalf("Filings error: %v", err)
	}
	if len(filings) != 2 {
		t.Fatalf("len(filings) = %d, want 2", len(filings))
	}
	if filings[0].Form != "10-K" || filings[0].URL == "" {
		t.Fatalf("unexpected filing: %+v", filings[0])
	}
	if !strings.Contains(filings[0].URL, "/Archives/edgar/data/320193/000032019325000095") {
		t.Fatalf("unexpected URL: %s", filings[0].URL)
	}
}
