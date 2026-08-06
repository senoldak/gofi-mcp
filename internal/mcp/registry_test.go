package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

type fakeFetcher struct {
	body []byte
	err  error
	path string
}

func (f *fakeFetcher) Get(_ context.Context, path string) ([]byte, error) {
	f.path = path
	if f.err != nil {
		return nil, f.err
	}
	return f.body, nil
}

func TestRegistryListsSixteenTools(t *testing.T) {
	r := NewRegistry(&fakeFetcher{})
	tools := r.List()
	if len(tools) != 16 {
		t.Fatalf("expected 16 tools, got %d", len(tools))
	}
	expected := []string{
		"get_quote", "get_company", "get_chart", "get_financials",
		"get_news", "get_related", "get_analyst", "get_context",
		"get_full", "search", "market_indices", "market_movers",
		"market_trending", "market_earnings", "market_headlines",
		"generate_chart",
	}
	for i, name := range expected {
		if tools[i].Name != name {
			t.Fatalf("tool[%d] = %q, want %q", i, tools[i].Name, name)
		}
	}
}

func TestCallGetQuoteEscapesTicker(t *testing.T) {
	f := &fakeFetcher{body: []byte(`{"ticker":"GOOGL:NASDAQ","price":100}`)}
	r := NewRegistry(f)
	out, err := r.Call(context.Background(), "get_quote", map[string]any{"ticker": "GOOGL:NASDAQ"})
	if err != nil {
		t.Fatalf("Call error: %v", err)
	}
	if !strings.Contains(f.path, "/v1/quote/") {
		t.Fatalf("unexpected path: %s", f.path)
	}
	if _, ok := out.(json.RawMessage); !ok {
		t.Fatalf("expected json.RawMessage output, got %T", out)
	}
}

func TestCallUnknownTool(t *testing.T) {
	r := NewRegistry(&fakeFetcher{})
	if _, err := r.Call(context.Background(), "nope", nil); err == nil {
		t.Fatal("expected error for unknown tool")
	}
}

func TestCallGetQuoteMissingTicker(t *testing.T) {
	r := NewRegistry(&fakeFetcher{})
	if _, err := r.Call(context.Background(), "get_quote", map[string]any{}); err == nil {
		t.Fatal("expected error for missing ticker")
	}
}

func TestCallPropagatesFetcherError(t *testing.T) {
	f := &fakeFetcher{err: errors.New("boom")}
	r := NewRegistry(f)
	if _, err := r.Call(context.Background(), "get_quote", map[string]any{"ticker": "AAPL:NASDAQ"}); err == nil {
		t.Fatal("expected fetcher error to propagate")
	}
}
