package fred

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/senoldak/gofi-mcp/internal/httpget"
)

func TestSeriesReturnsErrorOnErrorMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"error_code":400,"error_message":"Bad Request: Series does not exist"}`))
	}))
	defer srv.Close()

	c := &Client{inner: httpget.New(srv.URL), key: "secret"}
	s, err := c.Series(context.Background(), "DOES_NOT_EXIST")
	if err == nil {
		t.Fatalf("expected error, got series: %+v", s)
	}
	if !strings.Contains(err.Error(), "Series does not exist") {
		t.Fatalf("error = %v, want to contain 'Series does not exist'", err)
	}
}

func TestSeriesParsesObservations(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("series_id") != "FEDFUNDS" {
			t.Fatalf("series_id = %q", r.URL.Query().Get("series_id"))
		}
		if r.URL.Query().Get("api_key") != "secret" {
			t.Fatalf("api_key missing or wrong")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"realtime_start":"2026-01-01","observations":[` +
			`{"date":"2025-12-01","value":"4.33"},{"date":"2025-11-01","value":"."}]}`))
	}))
	defer srv.Close()

	c := &Client{inner: httpget.New(srv.URL), key: "secret"}
	s, err := c.Series(context.Background(), "FEDFUNDS")
	if err != nil {
		t.Fatalf("Series returned error: %v", err)
	}
	if s.Series != "FEDFUNDS" {
		t.Fatalf("Series = %q", s.Series)
	}
	if len(s.Observations) != 1 {
		t.Fatalf("len(Observations) = %d, want 1 ('.' skipped)", len(s.Observations))
	}
	if s.Observations[0].Value != 4.33 {
		t.Fatalf("Value = %v, want 4.33", s.Observations[0].Value)
	}
}
