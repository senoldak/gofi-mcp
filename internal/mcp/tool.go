package mcp

import "context"

type Tool struct {
	Name        string                                                      `json:"name"`
	Description string                                                      `json:"description"`
	InputSchema map[string]any                                              `json:"inputSchema"`
	Call        func(ctx context.Context, args map[string]any) (any, error) `json:"-"`
}

func stringArg(args map[string]any, key string) string {
	v, _ := args[key].(string)
	return v
}

func schema(description string, required []string, props map[string]any) map[string]any {
	s := map[string]any{
		"type":        "object",
		"properties":  props,
		"description": description,
	}
	if required != nil {
		s["required"] = required
	}
	return s
}

func strProp(description string) map[string]any {
	return map[string]any{"type": "string", "description": description}
}

func tickerSchema(description string) map[string]any {
	return schema(description, []string{"ticker"}, map[string]any{
		"ticker": strProp("Ticker in EXCHANGE:SYMBOL (GOOGL:NASDAQ), BASE-QUOTE (BTC-USD), or .SYMBOL:INDEXDJX format. Examples: THYAO:IST, AAPL:NASDAQ, USD-TRY."),
	})
}
