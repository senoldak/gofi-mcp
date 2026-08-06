package mcp

import (
	"context"
	"testing"

	"github.com/senoldak/gofi-mcp/internal/fred"
)

type fakeFredClient struct {
	series fred.Series
	err    error
}

func (f *fakeFredClient) Series(_ context.Context, id string) (fred.Series, error) {
	return f.series, f.err
}

func TestMacroIndicatorToolRegisters(t *testing.T) {
	tool := MacroIndicatorTool(&fakeFredClient{})
	if tool.Name != "macro_indicator" {
		t.Fatalf("Name = %q", tool.Name)
	}
}

func TestMacroIndicatorCallReturnsSeries(t *testing.T) {
	c := &fakeFredClient{series: fred.Series{Series: "FEDFUNDS", Observations: []fred.Observation{{Date: "2025-12-01", Value: 4.33}}}}
	tool := MacroIndicatorTool(c)
	out, err := tool.Call(context.Background(), map[string]any{"series": "FEDFUNDS"})
	if err != nil {
		t.Fatalf("Call error: %v", err)
	}
	if out.(fred.Series).Observations[0].Value != 4.33 {
		t.Fatalf("unexpected series: %+v", out)
	}
}

func TestMacroIndicatorCallRequiresSeries(t *testing.T) {
	tool := MacroIndicatorTool(&fakeFredClient{})
	if _, err := tool.Call(context.Background(), map[string]any{}); err == nil {
		t.Fatal("expected error for missing series")
	}
}
