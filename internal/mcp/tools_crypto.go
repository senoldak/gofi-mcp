package mcp

import (
	"context"
	"fmt"

	"github.com/senoldak/gofi-mcp/internal/coingecko"
)

type cryptoClient interface {
	Price(ctx context.Context, id string) (coingecko.Price, error)
	Market(ctx context.Context, category string) ([]coingecko.Price, error)
}

func CryptoTools(c cryptoClient) []Tool {
	return []Tool{
		{
			Name:        "crypto_price",
			Description: "Get current price, market cap and 24h change for a cryptocurrency.",
			InputSchema: schema("Cryptocurrency price.", []string{"id"}, map[string]any{
				"id": strProp("CoinGecko coin id, e.g. bitcoin, ethereum, tether."),
			}),
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				id := stringArg(args, "id")
				if id == "" {
					return nil, fmt.Errorf("id is required")
				}
				return c.Price(ctx, id)
			},
		},
		{
			Name:        "crypto_market",
			Description: "Get a ranked list of cryptocurrencies. Category: most-active (default), gainers, or volume.",
			InputSchema: schema("Crypto market list.", nil, map[string]any{
				"category": strProp("Category: most-active, gainers, volume. Default: most-active."),
			}),
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				cat := stringArg(args, "category")
				if cat == "" {
					cat = "most-active"
				}
				return c.Market(ctx, cat)
			},
		},
	}
}
