# gofi-mcp

Zero-dependency MCP (Model Context Protocol) server that exposes
[GOFI](https://github.com/senoldak/GOFI) market data as MCP tools to any
MCP-capable AI client (Claude Code/Cowork, opencode, Antigravity, Cursor, â€¦).

Data is served over GOFI's REST API â€” no API keys, no subscriptions.

## Quick Start

```bash
# Point at the public GOFI instance (no local server needed)
export GOFI_URL=https://finance.hermestech.uk
go run ./cmd/gofi-mcp

# Or point at a local GOFI
export GOFI_URL=http://localhost:8080
go run ./cmd/gofi-mcp
```

## Claude Code

```bash
claude mcp add gofi -s -- go run /path/to/gofi-mcp/cmd/gofi-mcp
```

## opencode

Add to `opencode.json`:

```json
{
  "mcp": {
    "gofi": {
      "type": "stdio",
      "command": ["gofi-mcp"],
      "env": { "GOFI_URL": "http://localhost:8080" }
    }
  }
}
```

## Tools

| Tool | Description |
|---|---|
| `get_quote` | Real-time quote for a ticker |
| `get_company` | Company profile & valuation metrics |
| `get_chart` | Historical price series (1Dâ€¦MAX) |
| `get_financials` | Financial statements (quarterly/annual) |
| `get_news` | Ticker news |
| `get_related` | Peer companies |
| `get_analyst` | Analyst reports / commentary |
| `get_context` | Cross-exchange listings |
| `get_full` | Quote + company + chart + news bundle |
| `search` | Ticker / asset search |
| `market_indices` | Global indices |
| `market_movers` | Gainers / losers / most-active |
| `market_trending` | Trending assets |
| `market_earnings` | Earnings calendar |
| `market_headlines` | Top headlines |

## Environment

| Variable | Default | Description |
|---|---|---|
| `GOFI_URL` | `http://localhost:8080` | Base URL of the GOFI server |

## Docker

```bash
docker build -t gofi-mcp .
```

## License

MIT
