package mcp

import (
	"context"
	"testing"

	"github.com/senoldak/gofi-mcp/internal/fx"
)

type fakeFXClient struct {
	rate fx.Rate
	err  error
}

func (f *fakeFXClient) Rate(_ context.Context, from, to string) (fx.Rate, error) {
	return f.rate, f.err
}

func TestFXToolsRegistersFxRate(t *testing.T) {
	tools := FXTools(&fakeFXClient{})
	if len(tools) != 1 || tools[0].Name != "fx_rate" {
		t.Fatalf("unexpected tools: %+v", tools)
	}
}

func TestFxRateCallReturnsRate(t *testing.T) {
	c := &fakeFXClient{rate: fx.Rate{From: "USD", To: "TRY", Rate: 35.2, Date: "2026-01-02"}}
	tools := FXTools(c)
	out, err := tools[0].Call(context.Background(), map[string]any{"from": "USD", "to": "TRY"})
	if err != nil {
		t.Fatalf("Call error: %v", err)
	}
	r := out.(fx.Rate)
	if r.Rate != 35.2 {
		t.Fatalf("Rate = %v, want 35.2", r.Rate)
	}
}

func TestFxRateCallRequiresArgs(t *testing.T) {
	tools := FXTools(&fakeFXClient{})
	if _, err := tools[0].Call(context.Background(), map[string]any{}); err == nil {
		t.Fatal("expected error for missing from/to")
	}
}
