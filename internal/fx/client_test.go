package fx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/senoldak/gofi-mcp/internal/httpget"
)

func TestRateParsesFrankfurterResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("from") != "USD" || r.URL.Query().Get("to") != "TRY" {
			t.Fatalf("unexpected query: %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"amount":1.0,"base":"USD","date":"2026-01-02","rates":{"TRY":35.2}}`))
	}))
	defer srv.Close()

	c := &Client{inner: httpget.New(srv.URL)}
	r, err := c.Rate(context.Background(), "USD", "TRY")
	if err != nil {
		t.Fatalf("Rate returned error: %v", err)
	}
	if r.Rate != 35.2 {
		t.Fatalf("Rate = %v, want 35.2", r.Rate)
	}
	if r.From != "USD" || r.To != "TRY" {
		t.Fatalf("unexpected pair: %s-%s", r.From, r.To)
	}
}

func TestRateReturnsErrorWhenQuoteMissing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"amount":1.0,"base":"USD","date":"2026-01-02","rates":{}}`))
	}))
	defer srv.Close()

	c := &Client{inner: httpget.New(srv.URL)}
	if _, err := c.Rate(context.Background(), "USD", "XXX"); err == nil {
		t.Fatal("expected error when quote currency absent")
	}
}
