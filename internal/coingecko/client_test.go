package coingecko

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/senoldak/gofi-mcp/internal/httpget"
)

func TestPriceReturnsFirstCoin(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("ids") != "bitcoin" {
			t.Fatalf("ids = %q", q.Get("ids"))
		}
		if q.Get("per_page") != "1" {
			t.Fatalf("per_page = %q, want 1 (matches docs contract)", q.Get("per_page"))
		}
		if q.Get("vs_currency") != "usd" {
			t.Fatalf("vs_currency = %q, want usd", q.Get("vs_currency"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"id":"bitcoin","symbol":"btc","name":"Bitcoin",
			"current_price":43456.12,"market_cap":850000000000,
			"price_change_percentage_24h":-1.5}]`))
	}))
	defer srv.Close()

	c := &Client{inner: httpget.New(srv.URL)}
	p, err := c.Price(context.Background(), "bitcoin")
	if err != nil {
		t.Fatalf("Price error: %v", err)
	}
	if p.ID != "bitcoin" || p.CurrentPrice != 43456.12 {
		t.Fatalf("unexpected price: %+v", p)
	}
}

func TestPriceEmptyArrayIsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	c := &Client{inner: httpget.New(srv.URL)}
	if _, err := c.Price(context.Background(), "nope"); err == nil {
		t.Fatal("expected error for empty array")
	}
}

func TestMarketUsesOrderParam(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("order") != "volume_desc" {
			t.Fatalf("order = %q", q.Get("order"))
		}
		if q.Get("per_page") != "20" {
			t.Fatalf("per_page = %q, want 20", q.Get("per_page"))
		}
		if q.Get("vs_currency") != "usd" {
			t.Fatalf("vs_currency = %q, want usd", q.Get("vs_currency"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"id":"bitcoin","symbol":"btc","name":"Bitcoin","current_price":1}]`))
	}))
	defer srv.Close()

	c := &Client{inner: httpget.New(srv.URL)}
	list, err := c.Market(context.Background(), "volume")
	if err != nil {
		t.Fatalf("Market error: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("len(list) = %d, want 1", len(list))
	}
}
