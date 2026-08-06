package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/senoldak/gofi-mcp/internal/goficlient"
)

type Registry struct {
	fetcher goficlient.Fetcher
	tools   []Tool
}

func NewRegistry(f goficlient.Fetcher) *Registry {
	r := &Registry{fetcher: f}
	r.tools = []Tool{
		r.quoteTool(),
		r.companyTool(),
		r.chartTool(),
		r.financialsTool(),
		r.newsTool(),
		r.relatedTool(),
		r.analystTool(),
		r.contextTool(),
		r.fullTool(),
		r.searchTool(),
		r.marketIndicesTool(),
		r.marketMoversTool(),
		r.marketTrendingTool(),
		r.marketEarningsTool(),
		r.marketHeadlinesTool(),
		ChartTool(),
	}
	return r
}

func (r *Registry) List() []Tool { return r.tools }

func (r *Registry) Add(t Tool) {
	r.tools = append(r.tools, t)
}

func (r *Registry) Call(ctx context.Context, name string, args map[string]any) (any, error) {
	for _, t := range r.tools {
		if t.Name == name {
			return t.Call(ctx, args)
		}
	}
	return nil, fmt.Errorf("unknown tool: %s", name)
}

func (r *Registry) fetchJSON(ctx context.Context, path string) (json.RawMessage, error) {
	body, err := r.fetcher.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	var raw json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decode GOFI response: %w", err)
	}
	return raw, nil
}

func (r *Registry) quoteTool() Tool {
	return Tool{
		Name:        "get_quote",
		Description: "Get a real-time quote (price, change, previous close) for a ticker.",
		InputSchema: tickerSchema("Ticker to look up."),
		Call: func(ctx context.Context, args map[string]any) (any, error) {
			t := stringArg(args, "ticker")
			if t == "" {
				return nil, fmt.Errorf("ticker is required")
			}
			return r.fetchJSON(ctx, "/v1/quote/"+url.PathEscape(t))
		},
	}
}

func (r *Registry) companyTool() Tool {
	return Tool{
		Name:        "get_company",
		Description: "Get company profile data: description, CEO, sector, market cap, P/E ratio, valuation metrics.",
		InputSchema: tickerSchema("Ticker to look up."),
		Call: func(ctx context.Context, args map[string]any) (any, error) {
			t := stringArg(args, "ticker")
			if t == "" {
				return nil, fmt.Errorf("ticker is required")
			}
			return r.fetchJSON(ctx, "/v1/company/"+url.PathEscape(t))
		},
	}
}

func (r *Registry) chartTool() Tool {
	return Tool{
		Name:        "get_chart",
		Description: "Get historical price chart data for a ticker. Range options: 1D, 5D, 1M, 6M, YTD, 1Y, 5Y, MAX (default 1M).",
		InputSchema: schema("Historical price series.", []string{"ticker"}, map[string]any{
			"ticker": strProp("Ticker to look up."),
			"range":  strProp("Chart range: 1D, 5D, 1M, 6M, YTD, 1Y, 5Y, MAX. Default: 1M."),
		}),
		Call: func(ctx context.Context, args map[string]any) (any, error) {
			t := stringArg(args, "ticker")
			if t == "" {
				return nil, fmt.Errorf("ticker is required")
			}
			rng := stringArg(args, "range")
			if rng == "" {
				rng = "1M"
			}
			path := "/v1/chart/" + url.PathEscape(t) + "?range=" + url.QueryEscape(rng)
			return r.fetchJSON(ctx, path)
		},
	}
}

func (r *Registry) financialsTool() Tool {
	return Tool{
		Name:        "get_financials",
		Description: "Get financial statements (revenue, net income, EPS, operating income, margins) for a ticker. Type: quarterly or annual (default quarterly).",
		InputSchema: schema("Financial statement periods.", []string{"ticker"}, map[string]any{
			"ticker": strProp("Ticker to look up."),
			"type":   strProp("Period type: quarterly or annual. Default: quarterly."),
		}),
		Call: func(ctx context.Context, args map[string]any) (any, error) {
			t := stringArg(args, "ticker")
			if t == "" {
				return nil, fmt.Errorf("ticker is required")
			}
			path := "/v1/financials/" + url.PathEscape(t)
			if typ := stringArg(args, "type"); typ != "" {
				path += "?type=" + url.QueryEscape(typ)
			}
			return r.fetchJSON(ctx, path)
		},
	}
}

func (r *Registry) newsTool() Tool {
	return Tool{
		Name:        "get_news",
		Description: "Get news articles associated with a ticker.",
		InputSchema: tickerSchema("Ticker to look up."),
		Call: func(ctx context.Context, args map[string]any) (any, error) {
			t := stringArg(args, "ticker")
			if t == "" {
				return nil, fmt.Errorf("ticker is required")
			}
			return r.fetchJSON(ctx, "/v1/news/"+url.PathEscape(t))
		},
	}
}

func (r *Registry) relatedTool() Tool {
	return Tool{
		Name:        "get_related",
		Description: "Get peer companies and related stocks for a ticker.",
		InputSchema: tickerSchema("Ticker to look up."),
		Call: func(ctx context.Context, args map[string]any) (any, error) {
			t := stringArg(args, "ticker")
			if t == "" {
				return nil, fmt.Errorf("ticker is required")
			}
			return r.fetchJSON(ctx, "/v1/related/"+url.PathEscape(t))
		},
	}
}

func (r *Registry) analystTool() Tool {
	return Tool{
		Name:        "get_analyst",
		Description: "Get analyst reports and market commentary for a ticker.",
		InputSchema: tickerSchema("Ticker to look up."),
		Call: func(ctx context.Context, args map[string]any) (any, error) {
			t := stringArg(args, "ticker")
			if t == "" {
				return nil, fmt.Errorf("ticker is required")
			}
			return r.fetchJSON(ctx, "/v1/analyst/"+url.PathEscape(t))
		},
	}
}

func (r *Registry) contextTool() Tool {
	return Tool{
		Name:        "get_context",
		Description: "Get multi-exchange listings and cross-market context for a ticker.",
		InputSchema: tickerSchema("Ticker to look up."),
		Call: func(ctx context.Context, args map[string]any) (any, error) {
			t := stringArg(args, "ticker")
			if t == "" {
				return nil, fmt.Errorf("ticker is required")
			}
			return r.fetchJSON(ctx, "/v1/context/"+url.PathEscape(t))
		},
	}
}

func (r *Registry) fullTool() Tool {
	return Tool{
		Name:        "get_full",
		Description: "Aggregate quote, company, chart and news for a ticker in one response. Range options: 1D, 5D, 1M, 6M, YTD, 1Y, 5Y, MAX (default 1M).",
		InputSchema: schema("Combined data bundle.", []string{"ticker"}, map[string]any{
			"ticker": strProp("Ticker to look up."),
			"range":  strProp("Chart range: 1D, 5D, 1M, 6M, YTD, 1Y, 5Y, MAX. Default: 1M."),
		}),
		Call: func(ctx context.Context, args map[string]any) (any, error) {
			t := stringArg(args, "ticker")
			if t == "" {
				return nil, fmt.Errorf("ticker is required")
			}
			rng := stringArg(args, "range")
			if rng == "" {
				rng = "1M"
			}
			path := "/v1/full/" + url.PathEscape(t) + "?range=" + url.QueryEscape(rng)
			return r.fetchJSON(ctx, path)
		},
	}
}

func (r *Registry) searchTool() Tool {
	return Tool{
		Name:        "search",
		Description: "Search tickers and asset names. Returns matching symbols with exchange, type, price and change.",
		InputSchema: schema("Ticker search.", []string{"query"}, map[string]any{
			"query": strProp("Search text, e.g. 'Apple' or 'THY'."),
		}),
		Call: func(ctx context.Context, args map[string]any) (any, error) {
			q := stringArg(args, "query")
			if q == "" {
				return nil, fmt.Errorf("query is required")
			}
			return r.fetchJSON(ctx, "/v1/search?q="+url.QueryEscape(q))
		},
	}
}

func (r *Registry) marketIndicesTool() Tool {
	return Tool{
		Name:        "market_indices",
		Description: "Get major global stock indices (S&P 500, Nasdaq, Dow Jones, BIST 100, etc.).",
		InputSchema: schema("No arguments.", nil, map[string]any{}),
		Call: func(ctx context.Context, args map[string]any) (any, error) {
			return r.fetchJSON(ctx, "/v1/market/indices")
		},
	}
}

func (r *Registry) marketMoversTool() Tool {
	return Tool{
		Name:        "market_movers",
		Description: "Get top market gainers, losers, or most active assets. Category: gainers, losers, most-active (default most-active).",
		InputSchema: schema("Market movers.", nil, map[string]any{
			"category": strProp("Category: gainers, losers, most-active. Default: most-active."),
		}),
		Call: func(ctx context.Context, args map[string]any) (any, error) {
			cat := stringArg(args, "category")
			if cat == "" {
				cat = "most-active"
			}
			return r.fetchJSON(ctx, "/v1/market/movers?category="+url.QueryEscape(cat))
		},
	}
}

func (r *Registry) marketTrendingTool() Tool {
	return Tool{
		Name:        "market_trending",
		Description: "Get most searched / trending assets on Google Finance.",
		InputSchema: schema("No arguments.", nil, map[string]any{}),
		Call: func(ctx context.Context, args map[string]any) (any, error) {
			return r.fetchJSON(ctx, "/v1/market/trending")
		},
	}
}

func (r *Registry) marketEarningsTool() Tool {
	return Tool{
		Name:        "market_earnings",
		Description: "Get upcoming earnings calendar announcements.",
		InputSchema: schema("No arguments.", nil, map[string]any{}),
		Call: func(ctx context.Context, args map[string]any) (any, error) {
			return r.fetchJSON(ctx, "/v1/market/earnings")
		},
	}
}

func (r *Registry) marketHeadlinesTool() Tool {
	return Tool{
		Name:        "market_headlines",
		Description: "Get top financial news headlines.",
		InputSchema: schema("No arguments.", nil, map[string]any{}),
		Call: func(ctx context.Context, args map[string]any) (any, error) {
			return r.fetchJSON(ctx, "/v1/market/headlines")
		},
	}
}
