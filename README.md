# 📈 gofi-mcp

[![Go Version](https://img.shields.io/badge/Go-1.25-00ADD8?style=flat&logo=go)](https://go.dev/)
[![MCP Protocol](https://img.shields.io/badge/MCP%20Protocol-2025--03--26-purple)](https://modelcontextprotocol.io/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Zero Dependencies](https://img.shields.io/badge/Dependencies-Zero%20External-green)](#-key-features)

**`gofi-mcp`** is a high-performance, zero-external-dependency Model Context Protocol (MCP) server written in pure Go. It seamlessly connects AI assistants (such as Claude Code, Cursor, OpenCode, Antigravity, Windsurf, and Roo Code) directly to live financial market data, macroeconomic indicators, official regulatory SEC filings, exchange rates, and cryptocurrency analytics.

By querying the underlying REST endpoints dynamically, **`gofi-mcp` requires no paid API keys or subscriptions for core financial data.**

---

## 📋 Table of Contents

- [✨ Key Features](#-key-features)
- [📁 Project Architecture](#-project-architecture)
- [⚙️ Environment Variables & Conditional Tools](#%EF%B8%8F-environment-variables--conditional-tools)
- [🚀 Quick Start](#-quick-start)
- [🤖 AI Assistant Client Integrations](#-ai-assistant-client-integrations)
  - [Claude Code](#claude-code)
  - [OpenCode](#opencode)
  - [Cursor / Windsurf / Antigravity / Roo Code](#cursor--windsurf--antigravity--roo-code)
- [🛠️ Detailed MCP Tools Reference](#%EF%B8%8F-detailed-mcp-tools-reference)
  - [1. Ticker & Company Specific Tools (GOFI REST API)](#1-ticker--company-specific-tools-gofi-rest-api)
  - [2. Market Overview & Calendar Tools (GOFI REST API)](#2-market-overview--calendar-tools-gofi-rest-api)
  - [3. Multi-Source Financial & Economic Tools](#3-multi-source-financial--economic-tools)
- [📌 Supported Ticker & Asset Symbol Formats](#-supported-ticker--asset-symbol-formats)
- [🧪 Testing & Quality Assurance](#-testing--quality-assurance)
- [🤝 Contributing & License](#-contributing--license)

---

## ✨ Key Features

- ⚡ **Zero External Dependencies:** Built strictly using the Go standard library (`net/http`, `encoding/json`, `bufio`, `context`, `sync`, etc.). No bloated node modules or heavy binary footprint.
- 📡 **Full MCP Specification Compliance:** Implements MCP specification version `2025-03-26` over standard input/output (`stdio`) using standard JSON-RPC 2.0 message framing.
- 📊 **Up to 21 Financial & Economic Tools:** Comprehensive access to real-time equity quotes, valuation multiples, financial statements, interactive charts, analyst ratings, market movers, macro series, exchange rates, SEC EDGAR 10-K/10-Q filings, and crypto market metrics.
- 🌍 **Multi-Market & Asset Support:** Supports global stock exchanges (BIST, NASDAQ, NYSE, LSE, XETRA), forex currency pairs (`USD-TRY`, `EUR-USD`), stock market indices (`.DJI:INDEXDJX`), and crypto assets (`BTC-USD`, `ethereum`).
- 🧩 **Multi-Source Data Architecture:** Seamlessly merges:
  - **GOFI REST API:** Live stock quotes, financials, charts, news, earnings calendars, and market movers.
  - **European Central Bank (ECB):** Official daily foreign exchange rates via `fx_rate`.
  - **FRED (St. Louis Fed):** Macroeconomic indicators (Federal Funds Rate, Inflation CPI, GDP, Unemployment Rate).
  - **SEC EDGAR:** Official XBRL financial statements (`sec_financials`) and recent regulatory filings (`sec_filing`).
  - **CoinGecko:** Live cryptocurrency prices, market caps, 24h volume, and top gainers.
- 🐳 **Ultra-Small Docker Image:** Multi-stage build resulting in a minimal, highly secure static binary running on an empty `scratch` image with bundled CA root certificates (`ca-certificates.crt`).

---

## 📁 Project Architecture

The codebase follows idiomatic Go package layout and clean architecture patterns, cleanly separating transport mechanisms (stdio / JSON-RPC 2.0), MCP registry routing, and independent service clients:

```
GOFI-MCP/
├── cmd/
│   └── gofi-mcp/
│       ├── main.go            # Entry point: Environment inspection, client setup & MCP server launcher
│       └── main_test.go       # Unit tests for URL resolution and configuration logic
├── internal/
│   ├── goficlient/
│   │   ├── client.go          # HTTP client for GOFI REST API (implements Fetcher interface)
│   │   └── client_test.go     # Mock HTTP tests for GOFI client response parsing
│   ├── httpget/
│   │   └── client.go          # Reusable stdlib HTTP GET client wrapper (custom headers, timeouts)
│   ├── fx/
│   │   └── client.go          # ECB exchange rate data client & parser
│   ├── fred/
│   │   ├── client.go          # St. Louis Fed FRED API series observation client
│   │   └── client_test.go     # FRED client response unmarshaling tests
│   ├── sec/
│   │   ├── client.go          # SEC EDGAR CIK lookup and XBRL companyfacts client
│   │   ├── client_test.go     # CIK lookup & XBRL normalization tests
│   │   └── financials.go      # XBRL GAAP facts normalization into standard financial statements
│   ├── coingecko/
│   │   └── client.go          # CoinGecko market & price API client
│   └── mcp/
│       ├── tool.go            # Tool definitions, JSON schemas, input validation & error formatting
│       ├── registry.go        # Dynamic tool registry for dispatching JSON-RPC requests
│       ├── registry_test.go   # Tool registration and routing validation tests
│       ├── tools_fx.go        # fx_rate MCP tool implementation
│       ├── tools_fred.go      # macro_indicator MCP tool implementation
│       ├── tools_sec.go       # sec_financials & sec_filing MCP tool implementations
│       ├── tools_crypto.go    # crypto_price & crypto_market MCP tool implementations
│       ├── server.go          # JSON-RPC 2.0 stdio server handling request dispatch & lifecycle
│       └── server_test.go     # Protocol level unit tests (initialize, tools/list, tools/call)
├── Dockerfile                 # Multi-stage Docker build -> distroless scratch image
├── go.mod                     # Go 1.25 module definition
└── README.md                  # Comprehensive project documentation
```

---

## ⚙️ Environment Variables & Conditional Tools

`gofi-mcp` automatically inspects environment variables on startup and dynamically registers or omits specific tools based on credentials provided.

### Environment Variables Matrix

| Variable | Default | Required For | Description |
|---|---|---|---|
| `GOFI_URL` | `http://localhost:8080` | Core GOFI Tools (15) | Base URL of the GOFI REST API server (e.g., `https://finance.hermestech.uk` or local instance). |
| `SEC_USER_AGENT` | *(unset)* | `sec_financials`, `sec_filing` | User-Agent string formatted according to SEC EDGAR policy (e.g., `YourName user@example.com`). Required by SEC API. |
| `FRED_API_KEY` | *(unset)* | `macro_indicator` | Free API key obtained from [St. Louis Fed FRED](https://fred.stlouisfed.org/docs/api/api_key.html). |
| `COINGECKO_API_KEY` | *(unset)* | Crypto Rate Limits | Optional CoinGecko API key. `crypto_price` and `crypto_market` are **always enabled**; providing a key increases rate limits. |

### Dynamic Tool Registration Count

Depending on configured environment variables, `gofi-mcp` will dynamically advertise the available tools via `tools/list`:

| `SEC_USER_AGENT` | `FRED_API_KEY` | Total Active MCP Tools | Included Tools |
|:---:|:---:|:---:|---|
| **Set** | **Set** | **21 Tools** | 15 GOFI + `fx_rate` + 2 Crypto + 2 SEC + `macro_indicator` |
| **Set** | *Unset* | **20 Tools** | 15 GOFI + `fx_rate` + 2 Crypto + 2 SEC |
| *Unset* | **Set** | **19 Tools** | 15 GOFI + `fx_rate` + 2 Crypto + `macro_indicator` |
| *Unset* | *Unset* | **18 Tools** | 15 GOFI + `fx_rate` + 2 Crypto |

*Note: `fx_rate`, `crypto_price`, and `crypto_market` are **always registered** regardless of API keys.*

---

## 🚀 Quick Start

### 1. Run directly with Go

Point `gofi-mcp` at a public or local GOFI REST API server:

```bash
# Using the public GOFI endpoint
export GOFI_URL=https://finance.hermestech.uk
export SEC_USER_AGENT="FinancialAgent admin@example.com"
export FRED_API_KEY="your_fred_api_key"

go run ./cmd/gofi-mcp
```

### 2. Build local executable

```bash
go build -o gofi-mcp ./cmd/gofi-mcp
./gofi-mcp
```

### 3. Run inside Docker container

```bash
# Build lightweight scratch Docker image
docker build -t gofi-mcp .

# Run container over stdio
docker run -i --rm \
  -e GOFI_URL=https://finance.hermestech.uk \
  -e SEC_USER_AGENT="FinancialAgent admin@example.com" \
  gofi-mcp
```

---

## 🤖 AI Assistant Client Integrations

### Claude Code

Add `gofi-mcp` to your Claude Code workspace using the CLI:

```bash
# Option A: Direct go run execution
claude mcp add gofi -e GOFI_URL=https://finance.hermestech.uk -e SEC_USER_AGENT="ClaudeAgent contact@example.com" -- go run /absolute/path/to/GOFI-MCP/cmd/gofi-mcp

# Option B: Using compiled binary
claude mcp add gofi -e GOFI_URL=https://finance.hermestech.uk -- /absolute/path/to/GOFI-MCP/gofi-mcp
```

### OpenCode

Add the server definition to your `opencode.json` configuration file (in workspace root or `~/.config/opencode/opencode.json`):

```json
{
  "mcp": {
    "gofi": {
      "type": "local",
      "command": ["go", "run", "./cmd/gofi-mcp"],
      "environment": {
        "GOFI_URL": "https://finance.hermestech.uk",
        "SEC_USER_AGENT": "OpenCodeAgent user@example.com",
        "FRED_API_KEY": "your_fred_api_key"
      }
    }
  }
}
```

### Cursor / Windsurf / Antigravity / Roo Code

Add the following JSON block to your `mcp.json` configuration file:

```json
{
  "mcpServers": {
    "gofi": {
      "command": "go",
      "args": ["run", "c:/path/to/GOFI-MCP/cmd/gofi-mcp"],
      "env": {
        "GOFI_URL": "https://finance.hermestech.uk",
        "SEC_USER_AGENT": "AntigravityAgent user@example.com",
        "FRED_API_KEY": "your_fred_api_key"
      }
    }
  }
}
```

---

## 🛠️ Detailed MCP Tools Reference

Below is the complete specification of all **21 supported MCP tools**, categorized by function.

### 1. Ticker & Company Specific Tools (GOFI REST API)

| Tool Name | Description | Required Parameters | Optional Parameters | Example Call |
|---|---|---|---|---|
| `get_quote` | Fetches real-time price quotes, daily price change, percentage change, and previous close. | `ticker` (string) | None | `{"ticker": "NVDA:NASDAQ"}` |
| `get_company` | Retrieves detailed company profile, business overview, CEO, sector, market cap, P/E ratio, and valuation metrics. | `ticker` (string) | None | `{"ticker": "THYAO:IST"}` |
| `get_chart` | Fetches historical price chart data points over specified time horizons. | `ticker` (string) | `range` (`1D`, `5D`, `1M`, `6M`, `YTD`, `1Y`, `5Y`, `MAX` - default: `1M`) | `{"ticker": "AAPL:NASDAQ", "range": "1Y"}` |
| `get_financials` | Retrieves income statements and key financial metrics (revenue, net income, operating margins, EPS). | `ticker` (string) | `type` (`quarterly`, `annual` - default: `quarterly`) | `{"ticker": "GARAN:IST", "type": "annual"}` |
| `get_news` | Retrieves recent news articles, press releases, and media mentions for a symbol. | `ticker` (string) | None | `{"ticker": "TSLA:NASDAQ"}` |
| `get_related` | Finds peer companies, industry competitors, and related stocks. | `ticker` (string) | None | `{"ticker": "MSFT:NASDAQ"}` |
| `get_analyst` | Retrieves analyst price targets, consensus recommendations (Buy/Hold/Sell), and rating reports. | `ticker` (string) | None | `{"ticker": "AMZN:NASDAQ"}` |
| `get_context` | Retrieves multi-exchange listing context and cross-market ticker availability. | `ticker` (string) | None | `{"ticker": "KCHOL:IST"}` |
| `get_full` | Bundles quote, company profile, historical chart, and news into a single aggregated JSON payload. | `ticker` (string) | `range` (string) | `{"ticker": "ASELS:IST"}` |
| `search` | Searches for equities, indices, and asset symbols by company name or ticker query string. | `query` (string) | None | `{"query": "NVIDIA"}` |

### 2. Market Overview & Calendar Tools (GOFI REST API)

| Tool Name | Description | Required Parameters | Optional Parameters | Example Call |
|---|---|---|---|---|
| `market_indices` | Fetches live quotes for major global stock indices (S&P 500, NASDAQ Composite, Dow Jones, BIST 100, DAX, FTSE 100). | None | None | `{}` |
| `market_movers` | Identifies top market gainers, top losers, or most active stocks by trading volume. | None | `category` (`gainers`, `losers`, `most-active` - default: `most-active`) | `{"category": "gainers"}` |
| `market_trending` | Retrieves most searched / trending tickers on Google Finance. | None | None | `{}` |
| `market_earnings` | Fetches the upcoming earnings release calendar for public companies. | None | None | `{}` |
| `market_headlines` | Retrieves top macroeconomic and financial market news headlines. | None | None | `{}` |

### 3. Multi-Source Financial & Economic Tools

| Tool Name | Source | Description | Parameters | Status / Requirements |
|---|---|---|---|---|
| `fx_rate` | European Central Bank | Gets foreign exchange rates between any two currencies (e.g., USD to TRY, EUR to USD). | `from` (string, required), `to` (string, required) | **Always Enabled** |
| `crypto_price` | CoinGecko | Gets real-time price, 24h price change percentage, and market cap for a cryptocurrency by CoinGecko ID. | `id` (string, required - e.g. `bitcoin`, `ethereum`) | **Always Enabled** |
| `crypto_market` | CoinGecko | Gets ranked cryptocurrency listings by market performance category. | `category` (`most-active`, `gainers`, `volume` - default: `most-active`) | **Always Enabled** |
| `sec_financials` | SEC EDGAR | Fetches normalized XBRL financial statements (Revenues, Net Income, Assets, Liabilities, Operating Income) directly from US SEC filings. | `ticker` (string, required - e.g. `NVDA`, `AAPL`) | Requires `SEC_USER_AGENT` |
| `sec_filing` | SEC EDGAR | Fetches recent official SEC filings (Form 10-K, 10-Q, 8-K) with filing dates and document links. | `ticker` (string, required - e.g. `TSLA`) | Requires `SEC_USER_AGENT` |
| `macro_indicator` | FRED (St. Louis Fed) | Fetches historical macroeconomic time-series observations (e.g. `FEDFUNDS`, `CPIAUCSL`, `GDP`, `UNRATE`). | `series` (string, required - e.g. `FEDFUNDS`) | Requires `FRED_API_KEY` |

---

## 📌 Supported Ticker & Asset Symbol Formats

When calling tools accepting a `ticker` or `symbol` parameter, use the following standard formats:

- **Stock Exchange & Ticker Symbol:**
  - Borsa Istanbul (BIST): `THYAO:IST`, `GARAN:IST`, `ASELS:IST`, `EREGL:IST`
  - US Exchanges (NASDAQ / NYSE): `NVDA:NASDAQ`, `AAPL:NASDAQ`, `TSLA:NASDAQ`, `JPM:NYSE`
- **Forex & Crypto Trading Pairs:**
  - Crypto Pairs: `BTC-USD`, `ETH-USD`, `SOL-USD`
  - Fiat Currency Pairs: `USD-TRY`, `EUR-TRY`, `EUR-USD`
- **Global Market Indices:**
  - `.DJI:INDEXDJX` (Dow Jones Industrial Average)
  - `.INX:INDEXSP` (S&P 500)
  - `.IXIC:INDEXNASDAQ` (NASDAQ Composite)
  - `XU100:IST` (BIST 100)

---

## 🧪 Testing & Quality Assurance

`gofi-mcp` maintains comprehensive unit test suites covering JSON-RPC protocol compliance, tool registration, mock HTTP responses, XBRL normalization, and client error handling.

To run all unit tests across the repository:

```bash
go test -v ./...
```

To execute tests with code coverage summary:

```bash
go test -cover ./...
```

---

## 🤝 Contributing & License

Contributions, issue reports, and feature requests are welcome! Feel free to open a pull request or submit an issue on GitHub.

This project is open-source software licensed under the **[MIT License](LICENSE)**.

