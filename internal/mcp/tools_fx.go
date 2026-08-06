package mcp

import (
	"context"
	"fmt"

	"github.com/senoldak/gofi-mcp/internal/fx"
)

type fxClient interface {
	Rate(ctx context.Context, from, to string) (fx.Rate, error)
}

func FXTools(c fxClient) []Tool {
	return []Tool{
		{
			Name:        "fx_rate",
			Description: "Get an official ECB (European Central Bank) exchange rate between two currencies.",
			InputSchema: schema("Official ECB exchange rate.", []string{"from", "to"}, map[string]any{
				"from": strProp("Base currency ISO 4217 code, e.g. USD, EUR, TRY."),
				"to":   strProp("Quote currency ISO 4217 code, e.g. TRY, USD, EUR."),
			}),
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				from := stringArg(args, "from")
				to := stringArg(args, "to")
				if from == "" || to == "" {
					return nil, fmt.Errorf("from and to are required")
				}
				return c.Rate(ctx, from, to)
			},
		},
	}
}
