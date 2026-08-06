# 📈 gofi-mcp

**gofi-mcp** is a high-performance, zero-dependency Model Context Protocol (MCP) server written in Go. It exposes financial market data from [GOFI](https://github.com/senoldak/GOFI) to any MCP-compliant AI assistant (Claude Code, Cursor, opencode, Antigravity, etc.).

Data is fetched dynamically over GOFI's REST API — **no API keys, no subscriptions required.**

---

## 🚀 Features

- ⚡ **Zero External Dependencies:** Built strictly using the Go standard library (`net/http`, `encoding/json`, `bufio`, etc.).
- 📡 **MCP Protocol Compliant:** Fully implements MCP protocol version `2025-03-26` over JSON-RPC 2.0 standard input/output (`stdio`).
- 📊 **Up to 21 Financial Tools:** Complete access to real-time quotes, company valuations, financial statements, historical charts, market movers, news, and analyst reports — plus exchange rates, macro indicators, SEC filings, and crypto data.
- 🌍 **Multi-Market & Asset Support:** Works seamlessly with BIST, NASDAQ, NYSE, Crypto pairs (e.g., `BTC-USD`), and global market indices.
- 🧩 **Multi-Source Data:** GOFI REST API, ECB exchange rates, FRED macro series, SEC EDGAR filings, and CoinGecko crypto prices.
- 🐳 **Minimal Docker Container:** Multi-stage build producing an ultra-small, secure `scratch` image with CA certificates included.

---

## 📁 Project Architecture

```
GOFI-MCP/
├── cmd/
│   └── gofi-mcp/
│       ├── main.go            # Application entry point & environment variable resolution
│       └── main_test.go       # URL resolution unit tests
├── internal/
│   ├── goficlient/
│   │   ├── client.go          # HTTP client for GOFI REST API (Fetcher interface)
│   │   └── client_test.go     # HTTP client unit tests using httptest
│   ├── httpget/
│   │   └── client.go          # Reusable stdlib HTTP GET helper (base URL, headers, timeout)
│   ├── fx/                    # ECB exchange rate client
│   ├── fred/                  # FRED macro series client
│   ├── sec/                   # SEC EDGAR client (financials + filings)
│   ├── coingecko/             # CoinGecko crypto client
│   └── mcp/
│       ├── tool.go            # MCP tool data structures & JSON Schema helpers
│       ├── registry.go        # Registry of GOFI tools + Add() for multi-source tools
│       ├── registry_test.go   # Tool registration & routing unit tests
│       ├── tools_fx.go        # fx_rate tool factory
│       ├── tools_fred.go      # macro_indicator tool factory
│       ├── tools_sec.go       # sec_financials / sec_filing tool factories
│       ├── tools_crypto.go    # crypto_price / crypto_market tool factories
│       ├── server.go          # JSON-RPC 2.0 stdio MCP server implementation
│       └── server_test.go     # MCP protocol unit tests
├── Dockerfile                 # Multi-stage Go build -> Scratch runtime image
├── go.mod                     # Go 1.25 module definition
└── README.md                  # Project documentation
```

---

## 🛠️ Quick Start

### 1. Run with Go

```bash
# Point at the public GOFI instance (no local server needed)
export GOFI_URL=https://finance.hermestech.uk
go run ./cmd/gofi-mcp

# Or point at a local GOFI instance
export GOFI_URL=http://localhost:8080
go run ./cmd/gofi-mcp
```

### 2. Build Binary

```bash
go build -o gofi-mcp ./cmd/gofi-mcp
./gofi-mcp
```

### 3. Run with Docker

```bash
# Build the Docker image
docker build -t gofi-mcp .

# Run the container
docker run -i --rm -e GOFI_URL=https://finance.hermestech.uk gofi-mcp
```

---

## ⚙️ Environment Variables

| Variable | Default Value | Description |
|---|---|---|
| `GOFI_URL` | `http://localhost:8080` | Base URL of the target GOFI REST API server |
| `SEC_USER_AGENT` | *(unset)* | User-Agent required for `sec_financials` / `sec_filing`. SEC EDGAR policy requires a real UA like `Name/version contact@example.com`. The `sec_*` tools are **absent from `tools/list` when unset**. |
| `FRED_API_KEY` | *(unset)* | API key required for `macro_indicator`. The `macro_indicator` tool is **absent from `tools/list` when unset**. |
| `COINGECKO_API_KEY` | *(unset)* | Optional CoinGecko API key. `crypto_price` / `crypto_market` are **always registered**; the key only raises the public-tier rate limits. |

### Conditional Tool Registration

The exact `tools/list` count depends on which environment variables are set:

| `SEC_USER_AGENT` | `FRED_API_KEY` | Tools in `tools/list` |
|---|---|---|
| set | set | **21** |
| set | unset | **20** |
| unset | set | **19** |
| unset | unset | **18** |

FX (`fx_rate`) and crypto (`crypto_price`, `crypto_market`) tools are **always registered** — crypto is present even without a `COINGECKO_API_KEY`.

---

## 🤖 AI Client Integration Guide

### Claude Code

Add `gofi-mcp` to Claude Code CLI using `go run`:

```bash
claude mcp add gofi -s -- go run /path/to/gofi-mcp/cmd/gofi-mcp
```

Or using the built binary with custom `GOFI_URL`:

```bash
claude mcp add gofi -e GOFI_URL=https://finance.hermestech.uk -- /path/to/gofi-mcp
```

### opencode

Add the following configuration to `opencode.json`:

```json
{
  "mcp": {
    "gofi": {
      "type": "stdio",
      "command": ["gofi-mcp"],
      "env": {
        "GOFI_URL": "https://finance.hermestech.uk"
      }
    }
  }
}
```

### Cursor / Windsurf / Antigravity / Roo Code (VS Code)

Add this server block to your MCP client configuration (`mcp.json` or plugin settings):

```json
{
  "mcpServers": {
    "gofi": {
      "command": "go",
      "args": ["run", "./cmd/gofi-mcp"],
      "env": {
        "GOFI_URL": "https://finance.hermestech.uk"
      }
    }
  }
}
```

---

## 🛠️ MCP Tools Reference

`gofi-mcp` exposes **up to 21 dedicated tools** to AI agents via JSON-RPC 2.0. The 15 GOFI tools below are always registered; the 6 multi-source tools are registered conditionally (see [Conditional Tool Registration](#conditional-tool-registration)).

### 1. Ticker & Company Specific Tools

| Tool Name | Description | Parameters | Example Values |
|---|---|---|---|
| `get_quote` | Fetch real-time price quote, change percentage, and previous close. | `ticker` *(required)* | `THYAO:IST`, `AAPL:NASDAQ`, `BTC-USD` |
| `get_company` | Get company profile, CEO, sector, market cap, P/E ratio, and valuation metrics. | `ticker` *(required)* | `GARAN:IST`, `NVDA:NASDAQ` |
| `get_chart` | Historical price chart series data. | `ticker` *(required)*, `range` *(optional: 1D, 5D, 1M, 6M, YTD, 1Y, 5Y, MAX - default: 1M)* | `ticker: "MSFT:NASDAQ", range: "1Y"` |
| `get_financials` | Financial statements (revenue, net income, EPS, operating margins). | `ticker` *(required)*, `type` *(optional: quarterly, annual - default: quarterly)* | `ticker: "ASELS:IST", type: "annual"` |
| `get_news` | Latest news articles associated with a specific ticker. | `ticker` *(required)* | `TSLA:NASDAQ` |
| `get_related` | Peer companies and related stocks. | `ticker` *(required)* | `AMZN:NASDAQ` |
| `get_analyst` | Analyst reports, target prices, and market commentary. | `ticker` *(required)* | `GOOGL:NASDAQ` |
| `get_context` | Multi-exchange listings and cross-market listings context. | `ticker` *(required)* | `KCHOL:IST` |
| `get_full` | Aggregated quote + company profile + chart + news bundle in a single call. | `ticker` *(required)*, `range` *(optional)* | `ticker: "THYAO:IST"` |
| `search` | Search tickers and asset names by keyword. | `query` *(required)* | `query: "Apple"` or `"THY"` |

### 2. Market Overview Tools

| Tool Name | Description | Parameters |
|---|---|---|
| `market_indices` | Major global stock market indices (S&P 500, Nasdaq, Dow Jones, BIST 100, etc.). | None |
| `market_movers` | Top market gainers, losers, or most active assets. | `category` *(optional: gainers, losers, most-active - default: most-active)* |
| `market_trending` | Most searched / trending assets on Google Finance. | None |
| `market_earnings` | Upcoming earnings announcement calendar. | None |
| `market_headlines` | Top financial and economic news headlines. | None |

### 3. Multi-Source Tools

| Tool Name | Description | Parameters | Example Values |
|---|---|---|---|
| `fx_rate` *(always)* | Get an official ECB (European Central Bank) exchange rate between two currencies. | `from` *(required)*, `to` *(required)* | `from: "USD", to: "TRY"` |
| `macro_indicator` *(requires `FRED_API_KEY`)* | Get a macroeconomic indicator series from FRED (e.g. GDP, UNRATE, CPIAUCSL, FEDFUNDS). | `series` *(required)* | `series: "GDP"` |
| `sec_financials` *(requires `SEC_USER_AGENT`)* | Get normalized financial statements (revenue, net income, EPS, assets, cash flow) from SEC EDGAR filings. | `ticker` *(required)* | `ticker: "AAPL"` |
| `sec_filing` *(requires `SEC_USER_AGENT`)* | Get the most recent SEC EDGAR filings (form, period, URL) for a ticker. | `ticker` *(required)* | `ticker: "AAPL"` |
| `crypto_price` *(always)* | Get current price, market cap and 24h change for a cryptocurrency. | `id` *(required)* | `id: "bitcoin"` |
| `crypto_market` *(always)* | Get a ranked list of cryptocurrencies. | `category` *(optional: most-active, gainers, volume - default: most-active)* | `category: "gainers"` |

---

## 📌 Supported Ticker Formats

All tools accepting a `ticker` parameter support the following symbol formats:

- **Stock Exchange & Symbol:** `THYAO:IST`, `AAPL:NASDAQ`, `GARAN:IST`, `NVDA:NASDAQ`
- **Forex & Crypto Pairs (Base-Quote):** `BTC-USD`, `ETH-USD`, `USD-TRY`, `EUR-TRY`
- **Market Indices:** `.SYMBOL:INDEXDJX` (e.g., `.DJI:INDEXDJX`)

---

## 🧪 Testing

Run all unit tests across all packages:

```bash
go test -v ./...
```

To run tests with code coverage analysis:

```bash
go test -cover ./...
```

---

## 📄 License

This project is licensed under the **MIT License**.
