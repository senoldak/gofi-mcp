# Design: GOFI-MCP Multi-Source Data Expansion

**Date:** 2026-08-06
**Status:** Approved
**Scope:** Add free, keyless-or-free-key external financial data sources to gofi-mcp, leaving GOFI as the primary source and untouched.

## Problem

gofi-mcp currently serves 15 tools backed solely by GOFI (Google Finance data). GOFI covers quotes, company profile, charts, financials (Google-derived), news, and market aggregates. It is weak at: authoritative US SEC filings/financials (needed for robust DCF), official exchange rates, macroeconomic indicators, and cryptocurrency. The original `financial-services` skills (`/comps`, `/dcf`, `/earnings`) relied on paid providers for several of these; this work substitutes free sources.

## Goals / Non-Goals

Goals:
- Add free, zero-dependency data sources to gofi-mcp, each exposed as tools with source-prefixed names.
- Keep GOFI the primary source; new sources only fill documented gaps.
- Preserve the existing 15 GOFI tools with zero regression.
- Preserve the zero-third-party-dependency principle (all new sources are plain HTTP).

Non-goals:
- Modify GOFI itself. All new-source calls go directly from gofi-mcp to the provider APIs.
- Paid / authenticated (billing) providers.
- Historical backfill beyond what providers freely give.

## Sources & Tool Scope

Chosen combination (from brainstorm): keyless sources + free-key sources. Six new tools (15 → 21).

| Source | Key | Env | Tool | Description |
|--------|-----|-----|------|-------------|
| SEC EDGAR | none (User-Agent required) | `SEC_USER_AGENT` | `sec_financials` | Normalized financial statements from `companyfacts` (10-K/10-Q periods) |
| SEC EDGAR | none (User-Agent required) | `SEC_USER_AGENT` | `sec_filing` | Recent filing list for a ticker (form, period, URL) |
| Frankfurter (ECB) | none | – | `fx_rate` | Official ECB exchange rate FOR-TO, optional historical date |
| FRED | free | `FRED_API_KEY` | `macro_indicator` | Macro series (GDP, UNRATE, CPIAUCSL, FEDFUNDS, ...) |
| CoinGecko | optional free | `COINGECKO_API_KEY` | `crypto_price` | Price, market cap, 24h change for a coin |
| CoinGecko | optional free | `COINGECKO_API_KEY` | `crypto_market` | Ranked list (gainers/losers/high-volume/category) |

Tool naming is source-prefixed (`sec_*`, `fx_*`, `macro_*`, `crypto_*`).

## Architecture

**Approach A (chosen): multiple fetchers + separate source packages + generalized registry.**

New package layout:

```
internal/
  httpget/          generic HTTP getter (baseURL, header/key, timeout, error wrapping)
    client.go       Client{BaseURL, Header, Timeout}; Get(ctx, path) ([]byte, error)
  goficlient/       existing — thin wrapper over httpget.Client; keeps Fetcher interface
    client.go       Get() delegates to httpget
  sec/              SEC EDGAR client (keyless, User-Agent required)
    client.go       GetFinancials(ticker), GetFilings(ticker)
    normalize.go    XBRL companyfacts -> GOFI-style flat JSON
  fred/             FRED client (FRED_API_KEY)
    client.go       GetSeries(seriesID)
  coingecko/        CoinGecko client (optional key)
    client.go       Price(id/key), Market(category)
  fx/               Frankfurter client (keyless)
    client.go       Rate(from, to, date?)
  mcp/
    registry.go     REGENERATED: Registry supports the existing single Fetcher PLUS
                    arbitrary additional Tools via Registry.Add(t Tool)
    tools_sec.go    sec_financials, sec_filing
    tools_fred.go   macro_indicator
    tools_crypto.go crypto_price, crypto_market
    tools_fx.go     fx_rate
```

### Registry generalization

Today `Registry` takes a single `goficlient.Fetcher`, and every tool has signature
`Call(ctx, f Fetcher, args) (any, error)`. Change:

- Keep `NewRegistry(f Fetcher)` for the 15 GOFI tools.
- Add `func (r *Registry) Add(t Tool)`.
- Simplify `Tool.Call` to `func(ctx, args map[string]any) (any, error)` using closures
  over each source's own client. The existing 15 GOFI tools are adapted to this shape.
- `Registry.Call` iterates `r.tools`; source clients are captured in each tool's closure;
  no fetcher passed per call.

### Env-driven conditional registration

- `FRED_API_KEY` set → register `macro_indicator`; unset → tool absent (not an error).
- `SEC_USER_AGENT` set → register `sec_*`; unset → SEC tools absent. SEC policy requires a real User-Agent; do not ship a default.
- `COINGECKO_API_KEY` optional: set → CoinGecko pro header (adds rate headroom); unset → public tier still works (both tools present).
- Conditions documented in README.

## Data normalization

Only SEC is normalized (raw XBRL → flat). Others return tidy provider JSON:

| Source | Request | Returned shape |
|--------|---------|----------------|
| SEC | `companyfacts/CIK.json` via ticker→CIK mapping; `frames` for periods | `sec_financials`: `{ticker, periods:[{fiscalYear, fiscalPeriod, revenue, netIncome, eps, totalAssets, totalLiabilities, operatingCashFlow}]}` |
| SEC | company filings index | `sec_filing`: `{ticker, filings:[{form, filedOn, period, url}]}` |
| Frankfurter | `/latest?from=USD&to=TRY` or dated | `fx_rate`: `{from, to, rate, date}` |
| FRED | `series/observations?series_id=...` | `macro_indicator`: `{series, unit, observations:[{date, value}]}` |
| CoinGecko | `simple/price`, `coins/markets` | `crypto_price`: `{id, symbol, price, marketCap, change24h}`; `crypto_market`: sorted list |

SEC financial periods render the most recent annual (FY) plus optional recent quarterly (Q1–Q4) for the selected ticker.

## Error handling

- Every new tool follows the existing MCP pattern: errors return as `isError:true` text content; server never crashes.
- 30s timeout per source.
- SEC rate-limit policy (10 req/s): satisfied by low-frequency agent usage; a real `Sec-User-Agent` header is required.
- FRED / SEC tools simply absent when their required env is missing.

## Testing & validation

- Each source package gets `httptest`-based unit tests using JSON fixtures (mirroring existing `gofetcher` tests namesake).
- SEC normalize logic unit-tested separately (raw XBRL → flat).
- Gates remain: `go test ./...`, `go vet ./...`, `gofmt -l .` empty, `go list -m all` single module (zero deps).
- Live smoke: `sec_financials AAPL`, `fx_rate USD-TRY`, `macro_indicator FEDFUNDS`, `crypto_price bitcoin`.
- Regression: the existing 15 GOFI tools still pass their current tests (unchanged fixtures); these tests are the regression gate for the registry refactor.
- `tools/list` expectation: 21 (or 20 without FRED key / fewer without SEC_USER_AGENT) depending on env.

## Risks

- SEC rate-limit / IP block: low risk at agent frequency; `Sec-User-Agent` mandatory.
- CoinGecko keyless rate-limit: low; resolved by the optional key.
- FRED needs a key: key absent → tool absent by design.
- GOFI regression from the `Fetcher`→closure refactor: the existing 15 GOFI tests remain the regression gate.

## Success Criteria

- `tools/list` exposes all new tools that their env prerequisites allow.
- Live smoke: SEC, FX, macro, and crypto calls return real, correctly normalized data.
- All existing GOFI tools and tests pass unchanged.
- Build remains a single module with zero third-party dependencies.