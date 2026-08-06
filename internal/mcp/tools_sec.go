package mcp

import (
	"context"
	"fmt"

	"github.com/senoldak/gofi-mcp/internal/sec"
)

type secClient interface {
	Financials(ctx context.Context, ticker string) (sec.Financials, error)
	Filings(ctx context.Context, ticker string) ([]sec.Filing, error)
}

func SECTools(c secClient) []Tool {
	return []Tool{
		{
			Name:        "sec_financials",
			Description: "Get normalized financial statements (revenue, net income, EPS, assets, cash flow) from SEC EDGAR filings.",
			InputSchema: tickerSchema("Ticker to look up on SEC EDGAR, e.g. AAPL, MSFT, TSLA."),
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				t := stringArg(args, "ticker")
				if t == "" {
					return nil, fmt.Errorf("ticker is required")
				}
				return c.Financials(ctx, t)
			},
		},
		{
			Name:        "sec_filing",
			Description: "Get the most recent SEC EDGAR filings (form, period, URL) for a ticker.",
			InputSchema: tickerSchema("Ticker to look up on SEC EDGAR, e.g. AAPL, MSFT, TSLA."),
			Call: func(ctx context.Context, args map[string]any) (any, error) {
				t := stringArg(args, "ticker")
				if t == "" {
					return nil, fmt.Errorf("ticker is required")
				}
				return c.Filings(ctx, t)
			},
		},
	}
}
