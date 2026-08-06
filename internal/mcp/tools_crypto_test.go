package mcp

import (
	"context"
	"testing"

	"github.com/senoldak/gofi-mcp/internal/coingecko"
)

type fakeCryptoClient struct {
	price coingecko.Price
	list  []coingecko.Price
	err   error
}

func (f *fakeCryptoClient) Price(_ context.Context, id string) (coingecko.Price, error) {
	return f.price, f.err
}

func (f *fakeCryptoClient) Market(_ context.Context, category string) ([]coingecko.Price, error) {
	return f.list, f.err
}

func TestCryptoToolsRegistersTwo(t *testing.T) {
	tools := CryptoTools(&fakeCryptoClient{})
	if len(tools) != 2 || tools[0].Name != "crypto_price" || tools[1].Name != "crypto_market" {
		t.Fatalf("unexpected tools: %+v", tools)
	}
}

func TestCryptoPriceCallRequiresID(t *testing.T) {
	tools := CryptoTools(&fakeCryptoClient{})
	if _, err := tools[0].Call(context.Background(), map[string]any{}); err == nil {
		t.Fatal("expected error for missing id")
	}
}

func TestCryptoPriceCallReturnsPrice(t *testing.T) {
	c := &fakeCryptoClient{price: coingecko.Price{ID: "bitcoin", CurrentPrice: 43456.12}}
	tools := CryptoTools(c)
	out, err := tools[0].Call(context.Background(), map[string]any{"id": "bitcoin"})
	if err != nil {
		t.Fatalf("Call error: %v", err)
	}
	if out.(coingecko.Price).CurrentPrice != 43456.12 {
		t.Fatalf("unexpected price: %+v", out)
	}
}

func TestCryptoMarketCallDefaultsCategory(t *testing.T) {
	c := &fakeCryptoClient{list: []coingecko.Price{{ID: "bitcoin"}}}
	tools := CryptoTools(c)
	out, err := tools[1].Call(context.Background(), map[string]any{})
	if err != nil {
		t.Fatalf("Call error: %v", err)
	}
	if len(out.([]coingecko.Price)) != 1 {
		t.Fatalf("unexpected list: %+v", out)
	}
}
