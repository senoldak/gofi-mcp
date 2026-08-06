package mcp

import (
	"context"
	"testing"

	"github.com/senoldak/gofi-mcp/internal/sec"
)

type fakeSECClient struct {
	fin     sec.Financials
	filings []sec.Filing
	err     error
}

func (f *fakeSECClient) Financials(_ context.Context, _ string) (sec.Financials, error) {
	return f.fin, f.err
}

func (f *fakeSECClient) Filings(_ context.Context, _ string) ([]sec.Filing, error) {
	return f.filings, f.err
}

func TestSECToolsRegistersTwo(t *testing.T) {
	tools := SECTools(&fakeSECClient{})
	if len(tools) != 2 {
		t.Fatalf("len(tools) = %d, want 2", len(tools))
	}
	if tools[0].Name != "sec_financials" || tools[1].Name != "sec_filing" {
		t.Fatalf("unexpected names: %s, %s", tools[0].Name, tools[1].Name)
	}
}

func TestSecFinancialsCallRequiresTicker(t *testing.T) {
	tools := SECTools(&fakeSECClient{})
	if _, err := tools[0].Call(context.Background(), map[string]any{}); err == nil {
		t.Fatal("expected error for missing ticker")
	}
}

func TestSecFinancialsCallReturnsNormalized(t *testing.T) {
	c := &fakeSECClient{fin: sec.Financials{Ticker: "0000320193", Periods: []sec.Period{{FiscalYear: 2025}}}}
	tools := SECTools(c)
	out, err := tools[0].Call(context.Background(), map[string]any{"ticker": "AAPL"})
	if err != nil {
		t.Fatalf("Call error: %v", err)
	}
	if out.(sec.Financials).Periods[0].FiscalYear != 2025 {
		t.Fatalf("unexpected result: %+v", out)
	}
}
