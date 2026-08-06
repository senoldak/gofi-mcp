package mcp

import (
	"context"
	"fmt"

	"github.com/senoldak/gofi-mcp/internal/fred"
)

type fredClient interface {
	Series(ctx context.Context, seriesID string) (fred.Series, error)
}

func MacroIndicatorTool(c fredClient) Tool {
	return Tool{
		Name:        "macro_indicator",
		Description: "Get a macroeconomic indicator series from FRED (e.g. GDP, UNRATE, CPIAUCSL, FEDFUNDS).",
		InputSchema: schema("FRED macro series.", []string{"series"}, map[string]any{
			"series": strProp("FRED series ID, e.g. GDP, UNRATE, CPIAUCSL, FEDFUNDS."),
		}),
		Call: func(ctx context.Context, args map[string]any) (any, error) {
			id := stringArg(args, "series")
			if id == "" {
				return nil, fmt.Errorf("series is required")
			}
			return c.Series(ctx, id)
		},
	}
}
