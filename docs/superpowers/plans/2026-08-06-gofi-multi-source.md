# GOFI-MCP Multi-Source Expansion Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add six free data-source tools (SEC EDGAR ×2, Frankfurter FX, FRED macro, CoinGecko ×2) to gofi-mcp, keeping GOFI the primary source and untouched.

**Architecture:** Approach A — a generic `internal/httpget` client is introduced; `goficlient` becomes a thin wrapper over it; each new source gets its own `internal/<source>` package whose tools are registered into the MCP registry via a generalized `Registry.Add`; tool calls use closures over source clients instead of a single injected fetcher.

**Tech Stack:** Go 1.25, standard library only (net/http, encoding/json). Zero third-party dependencies. MCP stdio JSON-RPC (already implemented).

## Global Constraints

- Zero third-party dependencies: `go list -m all` must show only `github.com/senoldak/gofi-mcp`.
- `go` directive in `go.mod` stays `go 1.25`; local toolchain 1.26.5 must still build.
- `gofmt -l .` must print nothing; `go vet ./...` must be clean; `go test ./...` all pass.
- GOFI code is NOT modified. The 15 existing GOFI tools keep identical names, schemas, and behaviors.
- No API keys committed. Keys/User-Agent come only from environment variables.
- Env-driven conditional registration: a tool is absent (not erroring) when its required env is unset.
- Windows PowerShell environment; commit messages follow existing style (`feat:`, `fix:`, `docs:`, `refactor:`).

---

### Task 1: `internal/httpget` generic HTTP client

**Files:**
- Create: `internal/httpget/client.go`
- Test: `internal/httpget/client_test.go`

**Interfaces:**
- Produces: `httpget.New(baseURL string) *Client` with `Client{BaseURL string; Header http.Header; Timeout time.Duration}` and `(c *Client) Get(ctx context.Context, path string) ([]byte, error)`. `Get` returns an error for any non-200 response and wraps transport errors. Default timeout 30s. `New` trims trailing `/` from baseURL.

- [ ] **Step 1: Write the failing test**

```go
package httpget

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGetReturnsBodyOn200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/quote/GOOGL:NASDAQ" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Write([]byte(`{"price":100}`))
	}))
	defer srv.Close()

	c := New(srv.URL)
	body, err := c.Get(context.Background(), "/v1/quote/GOOGL:NASDAQ")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if string(body) != `{"price":100}` {
		t.Fatalf("unexpected body: %s", body)
	}
}

func TestGetReturnsErrorOnNon200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := New(srv.URL)
	if _, err := c.Get(context.Background(), "/v1/quote/NOPE"); err == nil {
		t.Fatal("expected error for non-200 response")
	}
}

func TestNewTrimsTrailingSlash(t *testing.T) {
	c := New("http://localhost:8080/")
	if c.BaseURL != "http://localhost:8080" {
		t.Fatalf("BaseURL not trimmed: %s", c.BaseURL)
	}
}

func TestGetSendsConfiguredHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("User-Agent"); got != "TestAgent/1.0" {
			t.Fatalf("User-Agent = %q", got)
		}
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	c := New(srv.URL)
	c.Header.Set("User-Agent", "TestAgent/1.0")
	if _, err := c.Get(context.Background(), "/x"); err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
}

func TestDefaultTimeoutIs30Seconds(t *testing.T) {
	c := New("http://example.com")
	if c.Timeout != 30*time.Second {
		t.Fatalf("Timeout = %v, want 30s", c.Timeout)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/httpget/`
Expected: FAIL — package httpget does not exist.

- [ ] **Step 3: Write minimal implementation**

```go
package httpget

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultTimeout = 30 * time.Second

type Client struct {
	BaseURL string
	Header  http.Header
	Timeout time.Duration
}

func New(baseURL string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Header:  http.Header{},
		Timeout: defaultTimeout,
	}
}

func (c *Client) Get(ctx context.Context, path string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	for k, vs := range c.Header {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "application/json")
	}
	client := &http.Client{Timeout: c.Timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/httpget/`
Expected: PASS (5 tests)

- [ ] **Step 5: Commit**

```bash
git add internal/httpget/client.go internal/httpget/client_test.go
git commit -m "feat: add generic httpget client"
```

---

### Task 2: Refactor `goficlient` onto `httpget`

**Files:**
- Modify: `internal/goficlient/client.go` (full rewrite)
- Modify: `internal/goficlient/client_test.go` (one test)

**Interfaces:**
- Consumes: `httpget.New(baseURL) *Client`, `(*httpget.Client).Get(ctx, path) ([]byte, error)`.
- Produces: unchanged `goficlient.Fetcher` interface (`Get(ctx, path) ([]byte, error)`) and `goficlient.New(baseURL) *Client`. The `Fetcher` interface must remain satisfied so `internal/mcp` and its `fakeFetcher` tests are untouched.

- [ ] **Step 1: Write the failing test first (update trailing-slash test)**

Modify `TestNewTrimsTrailingSlash` in `internal/goficlient/client_test.go` — the old test reads `c.baseURL` which no longer exists after the refactor. Replace the test body:

```go
func TestNewTrimsTrailingSlash(t *testing.T) {
	c := New("http://localhost:8080/")
	if got := c.Get(context.Background(), "/v1/quote/GOOGL:NASDAQ"); got == nil {
		t.Fatal("expected a request to succeed against a trimmable base")
	}
}
```

Run: `go test ./internal/goficlient/`
Expected: FAIL — `c.baseURL` field reference removed, but the new body references an existing method; the failure is the refactor not yet done (existing `Client` still has old shape).

- [ ] **Step 2: Rewrite the client to wrap httpget**

```go
package goficlient

import (
	"context"

	"github.com/senoldak/gofi-mcp/internal/httpget"
)

type Fetcher interface {
	Get(ctx context.Context, path string) ([]byte, error)
}

type Client struct {
	inner *httpget.Client
}

func New(baseURL string) *Client {
	return &Client{inner: httpget.New(baseURL)}
}

func (c *Client) Get(ctx context.Context, path string) ([]byte, error) {
	return c.inner.Get(ctx, path)
}
```

- [ ] **Step 3: Run the full goficlient test suite**

Run: `go test ./internal/goficlient/`
Expected: PASS (3 tests: 200 body, non-200 error, trimmed-base request)

- [ ] **Step 4: Commit**

```bash
git add internal/goficlient/client.go internal/goficlient/client_test.go
git commit -m "refactor: wrap goficlient around httpget"
```

---

### Task 3: Generalize the MCP registry to closure-based tool calls

**Files:**
- Modify: `internal/mcp/tool.go` — `Tool.Call` signature
- Modify: `internal/mcp/registry.go` — every tool's `Call` field and `Registry.Call`, add `Add`
- Test: `internal/mcp/registry_test.go` (unchanged — the regression gate)

**Interfaces:**
- Consumes: existing `goficlient.Fetcher`.
- Produces: `Tool.Call func(ctx context.Context, args map[string]any) (any, error)` (no fetcher parameter — tools capture their own client via closure over `r`); `func (r *Registry) Add(t Tool)` appends to `r.tools`; `Registry.Call(ctx, name, args)` now invokes `t.Call(ctx, args)`.

The refactor is mechanical: in every existing tool, remove the `f goficlient.Fetcher` parameter from `Call` and rely on the receiver `r` (tools already call `r.fetchJSON`, which uses `r.fetcher`). The `goficlient` import in `registry.go` stays (used by `NewRegistry` signature).

- [ ] **Step 1: Change the `Tool` struct in `tool.go`**

```go
package mcp

import "context"

type Tool struct {
	Name        string                                                            `json:"name"`
	Description string                                                            `json:"description"`
	InputSchema map[string]any                                                    `json:"inputSchema"`
	Call        func(ctx context.Context, args map[string]any) (any, error)       `json:"-"`
}
```

(Remove the `goficlient` import from `tool.go` — it is no longer used there.)

- [ ] **Step 2: Rewrite `registry.go` — header, NewRegistry, Call, Add**

Keep the package header, imports, `Registry` struct, `NewRegistry`, and every tool function. Change only:

```go
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
```

- [ ] **Step 3: Update every tool's `Call` closure signature**

For each of the 15 tool functions in `registry.go`, change:
`Call: func(ctx context.Context, f goficlient.Fetcher, args map[string]any) (any, error) {`
to:
`Call: func(ctx context.Context, args map[string]any) (any, error) {`

The bodies are unchanged — they already use `r.fetchJSON(...)` which references `r.fetcher`. Do this for: `quoteTool`, `companyTool`, `chartTool`, `financialsTool`, `newsTool`, `relatedTool`, `analystTool`, `contextTool`, `fullTool`, `searchTool`, `marketIndicesTool`, `marketMoversTool`, `marketTrendingTool`, `marketEarningsTool`, `marketHeadlinesTool`.

- [ ] **Step 4: Run the full mcp suite (regression gate)**

Run: `go test ./...`
Expected: PASS — `registry_test.go` (fakeFetcher, 15 tools, unknown tool, escape, missing ticker, error propagation) must pass unchanged. `TestRegistryListsFifteenTools` still expects exactly 15 (no sources added yet).

- [ ] **Step 5: Verify formatting and vet**

Run: `gofmt -l .` and `go vet ./...`
Expected: `gofmt -l .` prints nothing; `go vet` clean.

- [ ] **Step 6: Commit**

```bash
git add internal/mcp/tool.go internal/mcp/registry.go
git commit -m "refactor: closure-based tool calls with Registry.Add"
```

---

### Task 4: Frankfurter (ECB) FX source + `fx_rate` tool

**Files:**
- Create: `internal/fx/client.go`
- Create: `internal/fx/client_test.go`
- Create: `internal/mcp/tools_fx.go`
- Create: `internal/mcp/tools_fx_test.go`

**Interfaces:**
- Consumes: `httpget.New(baseURL) *Client`, `(*httpget.Client).Get(ctx, path) ([]byte, error)`.
- Produces:
  - `fx.New() *Client` (keyless, base URL `https://api.frankfurter.app`)
  - `(*fx.Client).Rate(ctx context.Context, from, to string) (fx.Rate, error)` where `fx.Rate{From, To string; Rate float64; Date string}` with json tags `from`, `to`, `rate`, `date`.
  - `mcp.FXTools(c fxClient) []Tool` returning one tool `fx_rate` (schema `from`, `to`, both required).

- [ ] **Step 1: Write the failing client test**

```go
package fx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRateParsesFrankfurterResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("from") != "USD" || r.URL.Query().Get("to") != "TRY" {
			t.Fatalf("unexpected query: %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"amount":1.0,"base":"USD","date":"2026-01-02","rates":{"TRY":35.2}}`))
	}))
	defer srv.Close()

	c := &Client{inner: httpget.New(srv.URL)}
	r, err := c.Rate(context.Background(), "USD", "TRY")
	if err != nil {
		t.Fatalf("Rate returned error: %v", err)
	}
	if r.Rate != 35.2 {
		t.Fatalf("Rate = %v, want 35.2", r.Rate)
	}
	if r.From != "USD" || r.To != "TRY" {
		t.Fatalf("unexpected pair: %s-%s", r.From, r.To)
	}
}

func TestRateReturnsErrorWhenQuoteMissing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"amount":1.0,"base":"USD","date":"2026-01-02","rates":{}}`))
	}))
	defer srv.Close()

	c := &Client{inner: httpget.New(srv.URL)}
	if _, err := c.Rate(context.Background(), "USD", "XXX"); err == nil {
		t.Fatal("expected error when quote currency absent")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/fx/`
Expected: FAIL — package fx does not exist.

- [ ] **Step 3: Write the client implementation**

```go
package fx

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/senoldak/gofi-mcp/internal/httpget"
)

const baseURL = "https://api.frankfurter.app"

type Rate struct {
	From string  `json:"from"`
	To   string  `json:"to"`
	Rate float64 `json:"rate"`
	Date string  `json:"date"`
}

type Client struct {
	inner *httpget.Client
}

func New() *Client {
	return &Client{inner: httpget.New(baseURL)}
}

func (c *Client) Rate(ctx context.Context, from, to string) (Rate, error) {
	path := "/latest?from=" + url.QueryEscape(from) + "&to=" + url.QueryEscape(to)
	body, err := c.inner.Get(ctx, path)
	if err != nil {
		return Rate{}, fmt.Errorf("fx request: %w", err)
	}
	var resp struct {
		Date  string             `json:"date"`
		Base  string             `json:"base"`
		Rates map[string]float64 `json:"rates"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return Rate{}, fmt.Errorf("fx decode: %w", err)
	}
	rate, ok := resp.Rates[to]
	if !ok {
		return Rate{}, fmt.Errorf("no rate for %s in response", to)
	}
	return Rate{From: from, To: to, Rate: rate, Date: resp.Date}, nil
}
```

- [ ] **Step 4: Run client test to verify it passes**

Run: `go test ./internal/fx/`
Expected: PASS (2 tests)

- [ ] **Step 5: Write the failing tool test**

```go
package mcp

import (
	"context"
	"testing"

	"github.com/senoldak/gofi-mcp/internal/fx"
)

type fakeFXClient struct {
	rate fx.Rate
	err  error
}

func (f *fakeFXClient) Rate(_ context.Context, from, to string) (fx.Rate, error) {
	return f.rate, f.err
}

func TestFXToolsRegistersFxRate(t *testing.T) {
	tools := FXTools(&fakeFXClient{})
	if len(tools) != 1 || tools[0].Name != "fx_rate" {
		t.Fatalf("unexpected tools: %+v", tools)
	}
}

func TestFxRateCallReturnsRate(t *testing.T) {
	c := &fakeFXClient{rate: fx.Rate{From: "USD", To: "TRY", Rate: 35.2, Date: "2026-01-02"}}
	tools := FXTools(c)
	out, err := tools[0].Call(context.Background(), map[string]any{"from": "USD", "to": "TRY"})
	if err != nil {
		t.Fatalf("Call error: %v", err)
	}
	r := out.(fx.Rate)
	if r.Rate != 35.2 {
		t.Fatalf("Rate = %v, want 35.2", r.Rate)
	}
}

func TestFxRateCallRequiresArgs(t *testing.T) {
	tools := FXTools(&fakeFXClient{})
	if _, err := tools[0].Call(context.Background(), map[string]any{}); err == nil {
		t.Fatal("expected error for missing from/to")
	}
}
```

Run: `go test ./internal/mcp/ -run TestFx`
Expected: FAIL — `FXTools` not defined.

- [ ] **Step 6: Write the tool implementation**

```go
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
```

- [ ] **Step 7: Run tool test to verify it passes**

Run: `go test ./internal/mcp/ -run TestFx`
Expected: PASS (3 tests)

- [ ] **Step 8: Run full suite and commit**

Run: `go test ./...` and `gofmt -l .` and `go vet ./...`
Expected: all PASS, gofmt empty, vet clean.

```bash
git add internal/fx/ internal/mcp/tools_fx.go internal/mcp/tools_fx_test.go
git commit -m "feat: add Frankfurter FX source and fx_rate tool"
```

---

### Task 5: FRED macro source + `macro_indicator` tool

**Files:**
- Create: `internal/fred/client.go`
- Create: `internal/fred/client_test.go`
- Create: `internal/mcp/tools_fred.go`
- Create: `internal/mcp/tools_fred_test.go`

**Interfaces:**
- Consumes: `httpget.New(baseURL) *Client`, `(*httpget.Client).Get(ctx, path) ([]byte, error)`.
- Produces:
  - `fred.New(apiKey string) *Client` (base URL `https://api.stlouisfed.org/fred`)
  - `(*fred.Client).Series(ctx context.Context, seriesID string) (fred.Series, error)` where `fred.Series{Series string; Observations []fred.Observation}` and `fred.Observation{Date string; Value float64}` with json tags `date`, `value`. FRED returns numeric strings (possibly `.` for missing) — parse via `strconv.ParseFloat`, skipping `.`.
  - `mcp.MacroIndicatorTool(c fredClient) Tool` returning `macro_indicator` (schema `series`, required).

- [ ] **Step 1: Write the failing client test**

```go
package fred

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSeriesParsesObservations(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("series_id") != "FEDFUNDS" {
			t.Fatalf("series_id = %q", r.URL.Query().Get("series_id"))
		}
		if r.URL.Query().Get("api_key") != "secret" {
			t.Fatalf("api_key missing or wrong")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"realtime_start":"2026-01-01","observations":[` +
			`{"date":"2025-12-01","value":"4.33"},{"date":"2025-11-01","value":"."}]}`))
	}))
	defer srv.Close()

	c := &Client{inner: httpget.New(srv.URL), key: "secret"}
	s, err := c.Series(context.Background(), "FEDFUNDS")
	if err != nil {
		t.Fatalf("Series returned error: %v", err)
	}
	if s.Series != "FEDFUNDS" {
		t.Fatalf("Series = %q", s.Series)
	}
	if len(s.Observations) != 1 {
		t.Fatalf("len(Observations) = %d, want 1 ('.' skipped)", len(s.Observations))
	}
	if s.Observations[0].Value != 4.33 {
		t.Fatalf("Value = %v, want 4.33", s.Observations[0].Value)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/fred/`
Expected: FAIL — package fred does not exist.

- [ ] **Step 3: Write the client implementation**

```go
package fred

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/senoldak/gofi-mcp/internal/httpget"
)

const baseURL = "https://api.stlouisfed.org/fred"

type Observation struct {
	Date  string  `json:"date"`
	Value float64 `json:"value"`
}

type Series struct {
	Series       string        `json:"series"`
	Observations []Observation `json:"observations"`
}

type Client struct {
	inner *httpget.Client
	key   string
}

func New(apiKey string) *Client {
	return &Client{inner: httpget.New(baseURL), key: apiKey}
}

func (c *Client) Series(ctx context.Context, seriesID string) (Series, error) {
	path := "/series/observations?series_id=" + url.QueryEscape(seriesID) +
		"&api_key=" + url.QueryEscape(c.key) + "&file_type=json"
	body, err := c.inner.Get(ctx, path)
	if err != nil {
		return Series{}, fmt.Errorf("fred request: %w", err)
	}
	var resp struct {
		Observations []struct {
			Date  string `json:"date"`
			Value string `json:"value"`
		} `json:"observations"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return Series{}, fmt.Errorf("fred decode: %w", err)
	}
	s := Series{Series: seriesID}
	for _, o := range resp.Observations {
		if o.Value == "." {
			continue
		}
		v, err := strconv.ParseFloat(o.Value, 64)
		if err != nil {
			continue
		}
		s.Observations = append(s.Observations, Observation{Date: o.Date, Value: v})
	}
	return s, nil
}
```

- [ ] **Step 4: Run client test to verify it passes**

Run: `go test ./internal/fred/`
Expected: PASS (1 test)

- [ ] **Step 5: Write the failing tool test**

```go
package mcp

import (
	"context"
	"testing"

	"github.com/senoldak/gofi-mcp/internal/fred"
)

type fakeFredClient struct {
	series fred.Series
	err    error
}

func (f *fakeFredClient) Series(_ context.Context, id string) (fred.Series, error) {
	return f.series, f.err
}

func TestMacroIndicatorToolRegisters(t *testing.T) {
	tool := MacroIndicatorTool(&fakeFredClient{})
	if tool.Name != "macro_indicator" {
		t.Fatalf("Name = %q", tool.Name)
	}
}

func TestMacroIndicatorCallReturnsSeries(t *testing.T) {
	c := &fakeFredClient{series: fred.Series{Series: "FEDFUNDS", Observations: []fred.Observation{{Date: "2025-12-01", Value: 4.33}}}}
	tool := MacroIndicatorTool(c)
	out, err := tool.Call(context.Background(), map[string]any{"series": "FEDFUNDS"})
	if err != nil {
		t.Fatalf("Call error: %v", err)
	}
	if out.(fred.Series).Observations[0].Value != 4.33 {
		t.Fatalf("unexpected series: %+v", out)
	}
}

func TestMacroIndicatorCallRequiresSeries(t *testing.T) {
	tool := MacroIndicatorTool(&fakeFredClient{})
	if _, err := tool.Call(context.Background(), map[string]any{}); err == nil {
		t.Fatal("expected error for missing series")
	}
}
```

Run: `go test ./internal/mcp/ -run TestMacro`
Expected: FAIL — `MacroIndicatorTool` not defined.

- [ ] **Step 6: Write the tool implementation**

```go
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
```

- [ ] **Step 7: Run tool test to verify it passes**

Run: `go test ./internal/mcp/ -run TestMacro`
Expected: PASS (3 tests)

- [ ] **Step 8: Run full suite and commit**

Run: `go test ./...` and `gofmt -l .` and `go vet ./...`
Expected: all PASS, gofmt empty, vet clean.

```bash
git add internal/fred/ internal/mcp/tools_fred.go internal/mcp/tools_fred_test.go
git commit -m "feat: add FRED macro source and macro_indicator tool"
```

---

### Task 6: SEC EDGAR client with financial normalization

**Files:**
- Create: `internal/sec/client.go`
- Create: `internal/sec/client_test.go`
- Create: `internal/sec/normalize.go`

**Interfaces:**
- Consumes: `httpget.New(baseURL) *Client`, `(*httpget.Client).Get(ctx, path) ([]byte, error)`.
- Produces:
  - `sec.New(userAgent string) *Client` (base URL `https://data.sec.gov`, sets `User-Agent` header).
  - `(*sec.Client) Financials(ctx context.Context, ticker string) (sec.Financials, error)` where `sec.Financials{Ticker string; Periods []sec.Period}` and `sec.Period{FiscalYear int; FiscalPeriod string; Revenue, NetIncome, EPS, TotalAssets, TotalLiabilities, OperatingCashFlow float64}` with json tags `fiscalYear`, `fiscalPeriod`, `revenue`, `netIncome`, `eps`, `totalAssets`, `totalLiabilities`, `operatingCashFlow`.
  - `lookupCIK(ctx, ticker) (string, error)` — resolves ticker → 10-digit CIK string by fetching `https://www.sec.gov/files/company_tickers.json` (object mapping index → `{"cik_str": N, "ticker": "AAPL", "title": "..."}`) and matching case-insensitively.

Normalization approach in `normalize.go`:
- `func normalizeFinancials(cik string, raw []byte) (Financials, error)`.
- The companyfacts JSON shape is `{"cik":..., "facts":{"us-gaap":{ "<US-GAAP-tag>": {"units":{"USD":[ {...,"val":..., "fy":2025, "fp":"FY", "form":"10-K"}, ... ]}} }}}`.
- Read tags: `Revenues`, `NetIncomeLoss`, `EarningsPerShareBasic`, `Assets`, `TotalLiabilities`, `NetCashProvidedByUsedInOperatingActivities`.
- Build a map keyed `fy:fp` → `Period`, merging the six values; keep only periods with `form == "10-K"` and `fp == "FY"`; sort by `fy` descending; emit the most recent 5.

- [ ] **Step 1: Write the failing normalize test**

```go
package sec

import "testing"

func TestNormalizeFinancials(t *testing.T) {
	raw := []byte(`{
		"cik": 320193,
		"entityName": "APPLE INC",
		"facts": {
			"us-gaap": {
				"Revenues": {"units": {"USD": [
					{"end": "2025-09-27", "val": 391000000000, "fy": 2025, "fp": "FY", "form": "10-K"}
				]}},
				"NetIncomeLoss": {"units": {"USD": [
					{"end": "2025-09-27", "val": 93700000000, "fy": 2025, "fp": "FY", "form": "10-K"}
				]}},
				"EarningsPerShareBasic": {"units": {"USD": [
					{"end": "2025-09-27", "val": 6.1, "fy": 2025, "fp": "FY", "form": "10-K"}
				]}},
				"Assets": {"units": {"USD": [
					{"end": "2025-09-27", "val": 375000000000, "fy": 2025, "fp": "FY", "form": "10-K"}
				]}},
				"TotalLiabilities": {"units": {"USD": [
					{"end": "2025-09-27", "val": 279000000000, "fy": 2025, "fp": "FY", "form": "10-K"}
				]}},
				"NetCashProvidedByUsedInOperatingActivities": {"units": {"USD": [
					{"end": "2025-09-27", "val": 118000000000, "fy": 2025, "fp": "FY", "form": "10-K"}
				]}}
			}
		}
	}`)

	f, err := normalizeFinancials("0000320193", raw)
	if err != nil {
		t.Fatalf("normalizeFinancials error: %v", err)
	}
	if f.Ticker != "0000320193" {
		t.Fatalf("Ticker = %q", f.Ticker)
	}
	if len(f.Periods) != 1 {
		t.Fatalf("len(Periods) = %d, want 1", len(f.Periods))
	}
	p := f.Periods[0]
	if p.FiscalYear != 2025 || p.Revenue != 391000000000 || p.NetIncome != 93700000000 {
		t.Fatalf("unexpected period: %+v", p)
	}
	if p.EPS != 6.1 || p.OperatingCashFlow != 118000000000 {
		t.Fatalf("unexpected period: %+v", p)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/sec/`
Expected: FAIL — package sec does not exist.

- [ ] **Step 3: Write the normalize implementation**

```go
package sec

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
)

type Period struct {
	FiscalYear        int     `json:"fiscalYear"`
	FiscalPeriod      string  `json:"fiscalPeriod"`
	Revenue           float64 `json:"revenue"`
	NetIncome         float64 `json:"netIncome"`
	EPS               float64 `json:"eps"`
	TotalAssets       float64 `json:"totalAssets"`
	TotalLiabilities  float64 `json:"totalLiabilities"`
	OperatingCashFlow float64 `json:"operatingCashFlow"`
}

type Financials struct {
	Ticker  string   `json:"ticker"`
	Periods []Period `json:"periods"`
}

var usGaapTags = []string{
	"Revenues",
	"NetIncomeLoss",
	"EarningsPerShareBasic",
	"Assets",
	"TotalLiabilities",
	"NetCashProvidedByUsedInOperatingActivities",
}

func normalizeFinancials(cik string, raw []byte) (Financials, error) {
	var doc struct {
		Facts struct {
			UsGAAP map[string]struct {
				Units map[string][]struct {
					Val  float64 `json:"val"`
					FY   int     `json:"fy"`
					FP   string  `json:"fp"`
					Form string  `json:"form"`
				} `json:"units"`
			} `json:"us-gaap"`
		} `json:"facts"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return Financials{}, fmt.Errorf("sec companyfacts decode: %w", err)
	}

	periods := map[string]Period{}
	for _, tag := range usGaapTags {
		gaap, ok := doc.Facts.UsGAAP[tag]
		if !ok {
			continue
		}
		for _, rec := range gaap.Units["USD"] {
			if rec.Form != "10-K" || rec.FP != "FY" {
				continue
			}
			key := strconv.Itoa(rec.FY) + ":" + rec.FP
			p, ok := periods[key]
			if !ok {
				p = Period{FiscalYear: rec.FY, FiscalPeriod: rec.FP}
			}
			switch tag {
			case "Revenues":
				p.Revenue = rec.Val
			case "NetIncomeLoss":
				p.NetIncome = rec.Val
			case "EarningsPerShareBasic":
				p.EPS = rec.Val
			case "Assets":
				p.TotalAssets = rec.Val
			case "TotalLiabilities":
				p.TotalLiabilities = rec.Val
			case "NetCashProvidedByUsedInOperatingActivities":
				p.OperatingCashFlow = rec.Val
			}
			periods[key] = p
		}
	}

	out := make([]Period, 0, len(periods))
	for _, p := range periods {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].FiscalYear > out[j].FiscalYear })
	if len(out) > 5 {
		out = out[:5]
	}
	return Financials{Ticker: cik, Periods: out}, nil
}
```

- [ ] **Step 4: Run normalize test to verify it passes**

Run: `go test ./internal/sec/`
Expected: PASS (1 test)

- [ ] **Step 5: Write the failing client test**

```go
package sec

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFinancialsFetchesCompanyfacts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != "Agent/1.0 test@example.com" {
			t.Fatalf("User-Agent = %q", r.Header.Get("User-Agent"))
		}
		switch r.URL.Path {
		case "/files/company_tickers.json":
			w.Write([]byte(`{"0":{"cik_str":320193,"ticker":"AAPL","title":"Apple Inc."}}`))
		case "/api/xbrl/companyfacts/CIK0000320193.json":
			w.Write([]byte(`{"facts":{"us-gaap":{}}}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	c := &Client{inner: httpget.New(srv.URL), userAgent: "Agent/1.0 test@example.com"}
	f, err := c.Financials(context.Background(), "AAPL")
	if err != nil {
		t.Fatalf("Financials error: %v", err)
	}
	if !strings.HasPrefix(f.Ticker, "0000320193") {
		t.Fatalf("Ticker = %q", f.Ticker)
	}
}
```

- [ ] **Step 6: Run test to verify it fails**

Run: `go test ./internal/sec/ -run TestFinancials`
Expected: FAIL — `Client` not defined.

- [ ] **Step 7: Write the client implementation**

```go
package sec

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/senoldak/gofi-mcp/internal/httpget"
)

const baseURL = "https://data.sec.gov"

type Client struct {
	inner     *httpget.Client
	userAgent string
}

func New(userAgent string) *Client {
	c := httpget.New(baseURL)
	c.Header.Set("User-Agent", userAgent)
	return &Client{inner: c, userAgent: userAgent}
}

func (c *Client) Financials(ctx context.Context, ticker string) (Financials, error) {
	cik, err := c.lookupCIK(ctx, ticker)
	if err != nil {
		return Financials{}, err
	}
	body, err := c.inner.Get(ctx, "/api/xbrl/companyfacts/CIK"+cik+".json")
	if err != nil {
		return Financials{}, fmt.Errorf("sec companyfacts: %w", err)
	}
	return normalizeFinancials(cik, body)
}

func (c *Client) lookupCIK(ctx context.Context, ticker string) (string, error) {
	body, err := c.inner.Get(ctx, "/files/company_tickers.json")
	if err != nil {
		return "", fmt.Errorf("sec tickers: %w", err)
	}
	var m map[string]struct {
		CIK    int    `json:"cik_str"`
		Ticker string `json:"ticker"`
		Title  string `json:"title"`
	}
	if err := json.Unmarshal(body, &m); err != nil {
		return "", fmt.Errorf("sec tickers decode: %w", err)
	}
	want := strings.ToUpper(ticker)
	for _, e := range m {
		if strings.ToUpper(e.Ticker) == want {
			return fmt.Sprintf("%010d", e.CIK), nil
		}
	}
	return "", fmt.Errorf("ticker %q not found in SEC mapping", ticker)
}
```

- [ ] **Step 8: Run client test to verify it passes**

Run: `go test ./internal/sec/`
Expected: PASS (2 tests)

- [ ] **Step 9: Run full suite and commit**

Run: `go test ./...` and `gofmt -l .` and `go vet ./...`
Expected: all PASS, gofmt empty, vet clean.

```bash
git add internal/sec/
git commit -m "feat: add SEC EDGAR client with financial normalization"
```

---

### Task 7: SEC filings source + `sec_filing` and `sec_financials` tools

**Files:**
- Create: `internal/sec/filings.go`
- Create: `internal/sec/filings_test.go`
- Create: `internal/mcp/tools_sec.go`
- Create: `internal/mcp/tools_sec_test.go`

**Interfaces:**
- Consumes: `(*sec.Client).Financials(ctx, ticker) (sec.Financials, error)` (Task 6), `(*sec.Client).lookupCIK` (Task 6), `httpget` via `sec.Client.inner`.
- Produces:
  - `(*sec.Client) Filings(ctx context.Context, ticker string) ([]sec.Filing, error)` where `sec.Filing{Form, FiledOn, Period, URL string}` with json tags `form`, `filedOn`, `period`, `url`. Source: `/submissions/CIK<CIK>.json` → `{"filings":{"recent":{"form":[...],"filingDate":[...],"reportDate":[...],"accessionNumber":[...]}}}`; URL = `https://www.sec.gov/Archives/edgar/data/<cik-no-leading-zeros>/<accession-with-dashes>`.
  - `mcp.SECTools(c secClient) []Tool` returning `sec_financials` (ticker required) and `sec_filing` (ticker required).

- [ ] **Step 1: Write the failing filings test**

```go
package sec

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFilingsParsesRecent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/files/company_tickers.json":
			w.Write([]byte(`{"0":{"cik_str":320193,"ticker":"AAPL","title":"Apple Inc."}}`))
		case "/submissions/CIK0000320193.json":
			w.Write([]byte(`{"filings":{"recent":{
				"form":["10-K","8-K"],
				"filingDate":["2025-10-31","2025-08-01"],
				"reportDate":["2025-09-27",""],
				"accessionNumber":["0000320193-25-000095","0000320193-25-000088"]
			}}}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	c := &Client{inner: httpget.New(srv.URL)}
	filings, err := c.Filings(context.Background(), "AAPL")
	if err != nil {
		t.Fatalf("Filings error: %v", err)
	}
	if len(filings) != 2 {
		t.Fatalf("len(filings) = %d, want 2", len(filings))
	}
	if filings[0].Form != "10-K" || filings[0].URL == "" {
		t.Fatalf("unexpected filing: %+v", filings[0])
	}
	if !strings.Contains(filings[0].URL, "/Archives/edgar/data/320193/") {
		t.Fatalf("unexpected URL: %s", filings[0].URL)
	}
}
```

(Add `"strings"` to the imports of the test file.)

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/sec/ -run TestFilings`
Expected: FAIL — `Filings` not defined.

- [ ] **Step 3: Write the filings implementation**

```go
package sec

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type Filing struct {
	Form    string `json:"form"`
	FiledOn string `json:"filedOn"`
	Period  string `json:"period"`
	URL     string `json:"url"`
}

func (c *Client) Filings(ctx context.Context, ticker string) ([]Filing, error) {
	cik, err := c.lookupCIK(ctx, ticker)
	if err != nil {
		return nil, err
	}
	body, err := c.inner.Get(ctx, "/submissions/CIK"+cik+".json")
	if err != nil {
		return nil, fmt.Errorf("sec submissions: %w", err)
	}
	var doc struct {
		Filings struct {
			Recent struct {
				Form            []string `json:"form"`
				FilingDate      []string `json:"filingDate"`
				ReportDate      []string `json:"reportDate"`
				AccessionNumber []string `json:"accessionNumber"`
			} `json:"recent"`
		} `json:"filings"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("sec submissions decode: %w", err)
	}

	recent := doc.Filings.Recent
	n := len(recent.Form)
	if n > 10 {
		n = 10
	}
	cikNoZeros := strings.TrimLeft(cik, "0")
	out := make([]Filing, 0, n)
	for i := 0; i < n; i++ {
		accn := ""
		if i < len(recent.AccessionNumber) {
			accn = recent.AccessionNumber[i]
		}
		f := Filing{
			Form:   recent.Form[i],
			FiledOn: "", Period: "",
			URL: "https://www.sec.gov/Archives/edgar/data/" + cikNoZeros + "/" + accn,
		}
		if i < len(recent.FilingDate) {
			f.FiledOn = recent.FilingDate[i]
		}
		if i < len(recent.ReportDate) {
			f.Period = recent.ReportDate[i]
		}
		out = append(out, f)
	}
	return out, nil
}
```

- [ ] **Step 4: Run filings test to verify it passes**

Run: `go test ./internal/sec/`
Expected: PASS (3 tests)

- [ ] **Step 5: Write the failing tool tests**

```go
package mcp

import (
	"context"
	"testing"

	"github.com/senoldak/gofi-mcp/internal/sec"
)

type fakeSECClient struct {
	fin     sec.Financials
	filings []sec.Filing
	err     error
}

func (f *fakeSECClient) Financials(_ context.Context, _ string) (sec.Financials, error) {
	return f.fin, f.err
}

func (f *fakeSECClient) Filings(_ context.Context, _ string) ([]sec.Filing, error) {
	return f.filings, f.err
}

func TestSECToolsRegistersTwo(t *testing.T) {
	tools := SECTools(&fakeSECClient{})
	if len(tools) != 2 {
		t.Fatalf("len(tools) = %d, want 2", len(tools))
	}
	if tools[0].Name != "sec_financials" || tools[1].Name != "sec_filing" {
		t.Fatalf("unexpected names: %s, %s", tools[0].Name, tools[1].Name)
	}
}

func TestSecFinancialsCallRequiresTicker(t *testing.T) {
	tools := SECTools(&fakeSECClient{})
	if _, err := tools[0].Call(context.Background(), map[string]any{}); err == nil {
		t.Fatal("expected error for missing ticker")
	}
}

func TestSecFinancialsCallReturnsNormalized(t *testing.T) {
	c := &fakeSECClient{fin: sec.Financials{Ticker: "0000320193", Periods: []sec.Period{{FiscalYear: 2025}}}}
	tools := SECTools(c)
	out, err := tools[0].Call(context.Background(), map[string]any{"ticker": "AAPL"})
	if err != nil {
		t.Fatalf("Call error: %v", err)
	}
	if out.(sec.Financials).Periods[0].FiscalYear != 2025 {
		t.Fatalf("unexpected result: %+v", out)
	}
}
```

Run: `go test ./internal/mcp/ -run TestSec`
Expected: FAIL — `SECTools` not defined.

- [ ] **Step 6: Write the tool implementation**

```go
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
```

- [ ] **Step 7: Run tool tests to verify they pass**

Run: `go test ./internal/mcp/ -run TestSec`
Expected: PASS (3 tests)

- [ ] **Step 8: Run full suite and commit**

Run: `go test ./...` and `gofmt -l .` and `go vet ./...`
Expected: all PASS, gofmt empty, vet clean.

```bash
git add internal/sec/filings.go internal/sec/filings_test.go internal/mcp/tools_sec.go internal/mcp/tools_sec_test.go
git commit -m "feat: add SEC filings and sec_financials/sec_filing tools"
```

---

### Task 8: CoinGecko crypto source + `crypto_price` and `crypto_market` tools

**Files:**
- Create: `internal/coingecko/client.go`
- Create: `internal/coingecko/client_test.go`
- Create: `internal/mcp/tools_crypto.go`
- Create: `internal/mcp/tools_crypto_test.go`

**Interfaces:**
- Consumes: `httpget.New(baseURL) *Client`, `(*httpget.Client).Get(ctx, path) ([]byte, error)`.
- Produces:
  - `coingecko.New(apiKey string) *Client` (base URL `https://api.coingecko.com/api/v3`; if apiKey non-empty, sets header `x-cg-pro-api-key`).
  - `(*coingecko.Client) Price(ctx context.Context, id string) (coingecko.Price, error)` — uses `/coins/markets?vs_currency=usd&ids=<id>` returning an array; takes element [0].
  - `(*coingecko.Client) Market(ctx context.Context, category string) ([]coingecko.Price, error)` — `/coins/markets?vs_currency=usd&per_page=20`; `category` ∈ `market_cap_desc` (default), `volume_desc`, `gecko_desc` (gainers). Parameter maps to `order`.
  - `coingecko.Price{ID, Symbol, Name string; CurrentPrice, MarketCap, Change24h float64}` with json tags `id`, `symbol`, `name`, `current_price`, `market_cap`, `price_change_percentage_24h`.
  - `mcp.CryptoTools(c cryptoClient) []Tool` returning `crypto_price` (id required) and `crypto_market` (category optional, default `most-active` → `market_cap_desc`).

- [ ] **Step 1: Write the failing client test**

```go
package coingecko

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPriceReturnsFirstCoin(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("ids") != "bitcoin" {
			t.Fatalf("ids = %q", r.URL.Query().Get("ids"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"id":"bitcoin","symbol":"btc","name":"Bitcoin",
			"current_price":43456.12,"market_cap":850000000000,
			"price_change_percentage_24h":-1.5}]`))
	}))
	defer srv.Close()

	c := &Client{inner: httpget.New(srv.URL)}
	p, err := c.Price(context.Background(), "bitcoin")
	if err != nil {
		t.Fatalf("Price error: %v", err)
	}
	if p.ID != "bitcoin" || p.CurrentPrice != 43456.12 {
		t.Fatalf("unexpected price: %+v", p)
	}
}

func TestPriceEmptyArrayIsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	c := &Client{inner: httpget.New(srv.URL)}
	if _, err := c.Price(context.Background(), "nope"); err == nil {
		t.Fatal("expected error for empty array")
	}
}

func TestMarketUsesOrderParam(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("order") != "volume_desc" {
			t.Fatalf("order = %q", r.URL.Query().Get("order"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"id":"bitcoin","symbol":"btc","name":"Bitcoin","current_price":1}]`))
	}))
	defer srv.Close()

	c := &Client{inner: httpget.New(srv.URL)}
	list, err := c.Market(context.Background(), "volume")
	if err != nil {
		t.Fatalf("Market error: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("len(list) = %d, want 1", len(list))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/coingecko/`
Expected: FAIL — package coingecko does not exist.

- [ ] **Step 3: Write the client implementation**

```go
package coingecko

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/senoldak/gofi-mcp/internal/httpget"
)

const baseURL = "https://api.coingecko.com/api/v3"

type Price struct {
	ID           string  `json:"id"`
	Symbol       string  `json:"symbol"`
	Name         string  `json:"name"`
	CurrentPrice float64 `json:"current_price"`
	MarketCap    float64 `json:"market_cap"`
	Change24h    float64 `json:"price_change_percentage_24h"`
}

type Client struct {
	inner *httpget.Client
}

func New(apiKey string) *Client {
	c := httpget.New(baseURL)
	if apiKey != "" {
		c.Header.Set("x-cg-pro-api-key", apiKey)
	}
	return &Client{inner: c}
}

func (c *Client) Price(ctx context.Context, id string) (Price, error) {
	path := "/coins/markets?vs_currency=usd&ids=" + url.QueryEscape(id) + "&per_page=1"
	body, err := c.inner.Get(ctx, path)
	if err != nil {
		return Price{}, fmt.Errorf("coingecko price: %w", err)
	}
	var list []Price
	if err := json.Unmarshal(body, &list); err != nil {
		return Price{}, fmt.Errorf("coingecko decode: %w", err)
	}
	if len(list) == 0 {
		return Price{}, fmt.Errorf("no coin found for id %q", id)
	}
	return list[0], nil
}

var marketOrders = map[string]string{
	"most-active": "market_cap_desc",
	"gainers":     "gecko_desc",
	"volume":      "volume_desc",
}

func (c *Client) Market(ctx context.Context, category string) ([]Price, error) {
	order := marketOrders[category]
	if order == "" {
		order = marketOrders["most-active"]
	}
	path := "/coins/markets?vs_currency=usd&per_page=20&order=" + url.QueryEscape(order)
	body, err := c.inner.Get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("coingecko market: %w", err)
	}
	var list []Price
	if err := json.Unmarshal(body, &list); err != nil {
		return nil, fmt.Errorf("coingecko decode: %w", err)
	}
	return list, nil
}
```

- [ ] **Step 4: Run client test to verify it passes**

Run: `go test ./internal/coingecko/`
Expected: PASS (3 tests)

- [ ] **Step 5: Write the failing tool tests**

```go
package mcp

import (
	"context"
	"testing"

	"github.com/senoldak/gofi-mcp/internal/coingecko"
)

type fakeCryptoClient struct {
	price coingecko.Price
	list  []coingecko.Price
	err   error
}

func (f *fakeCryptoClient) Price(_ context.Context, id string) (coingecko.Price, error) {
	return f.price, f.err
}

func (f *fakeCryptoClient) Market(_ context.Context, category string) ([]coingecko.Price, error) {
	return f.list, f.err
}

func TestCryptoToolsRegistersTwo(t *testing.T) {
	tools := CryptoTools(&fakeCryptoClient{})
	if len(tools) != 2 || tools[0].Name != "crypto_price" || tools[1].Name != "crypto_market" {
		t.Fatalf("unexpected tools: %+v", tools)
	}
}

func TestCryptoPriceCallRequiresID(t *testing.T) {
	tools := CryptoTools(&fakeCryptoClient{})
	if _, err := tools[0].Call(context.Background(), map[string]any{}); err == nil {
		t.Fatal("expected error for missing id")
	}
}

func TestCryptoPriceCallReturnsPrice(t *testing.T) {
	c := &fakeCryptoClient{price: coingecko.Price{ID: "bitcoin", CurrentPrice: 43456.12}}
	tools := CryptoTools(c)
	out, err := tools[0].Call(context.Background(), map[string]any{"id": "bitcoin"})
	if err != nil {
		t.Fatalf("Call error: %v", err)
	}
	if out.(coingecko.Price).CurrentPrice != 43456.12 {
		t.Fatalf("unexpected price: %+v", out)
	}
}

func TestCryptoMarketCallDefaultsCategory(t *testing.T) {
	c := &fakeCryptoClient{list: []coingecko.Price{{ID: "bitcoin"}}}
	tools := CryptoTools(c)
	out, err := tools[1].Call(context.Background(), map[string]any{})
	if err != nil {
		t.Fatalf("Call error: %v", err)
	}
	if len(out.([]coingecko.Price)) != 1 {
		t.Fatalf("unexpected list: %+v", out)
	}
}
```

Run: `go test ./internal/mcp/ -run TestCrypto`
Expected: FAIL — `CryptoTools` not defined.

- [ ] **Step 6: Write the tool implementation**

```go
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
```

- [ ] **Step 7: Run tool tests to verify they pass**

Run: `go test ./internal/mcp/ -run TestCrypto`
Expected: PASS (4 tests)

- [ ] **Step 8: Run full suite and commit**

Run: `go test ./...` and `gofmt -l .` and `go vet ./...`
Expected: all PASS, gofmt empty, vet clean.

```bash
git add internal/coingecko/ internal/mcp/tools_crypto.go internal/mcp/tools_crypto_test.go
git commit -m "feat: add CoinGecko crypto source and crypto tools"
```

---

### Task 9: Wire up `main.go`, update README, live smoke test

**Files:**
- Modify: `cmd/gofi-mcp/main.go`
- Modify: `README.md`
- Test: `cmd/gofi-mcp/main_test.go` (unchanged — regression)

**Interfaces:**
- Consumes: `mcp.NewRegistry`, `mcp.SECTools`, `mcp.MacroIndicatorTool`, `mcp.FXTools`, `mcp.CryptoTools`, and `sec.New`, `fred.New`, `fx.New`, `coingecko.New` from their packages.
- Produces: `main()` that conditionally registers new tools based on env; README documents all env vars and the conditional-registration behavior.

- [ ] **Step 1: Verify the existing main tests still compile and pass**

The existing `cmd/gofi-mcp/main_test.go` tests `resolveURL` only. `main()` is exercised by the live smoke test in Step 5 (it runs a stdio loop, so it is not unit-testable). `main_test.go` must keep passing untouched after the `main.go` rewrite — this is the regression gate for the wiring change.

Run: `go test ./cmd/gofi-mcp/`
Expected: PASS (2 tests, unchanged).

- [ ] **Step 2: Rewrite `main.go` with conditional registration**

```go
package main

import (
	"context"
	"log"
	"os"

	"github.com/senoldak/gofi-mcp/internal/coingecko"
	"github.com/senoldak/gofi-mcp/internal/fred"
	"github.com/senoldak/gofi-mcp/internal/fx"
	"github.com/senoldak/gofi-mcp/internal/goficlient"
	"github.com/senoldak/gofi-mcp/internal/mcp"
	"github.com/senoldak/gofi-mcp/internal/sec"
)

const defaultURL = "http://localhost:8080"

func resolveURL(env string) string {
	if env == "" {
		return defaultURL
	}
	return env
}

func main() {
	baseURL := resolveURL(os.Getenv("GOFI_URL"))
	client := goficlient.New(baseURL)
	registry := mcp.NewRegistry(client)

	if ua := os.Getenv("SEC_USER_AGENT"); ua != "" {
		for _, t := range mcp.SECTools(sec.New(ua)) {
			registry.Add(t)
		}
	}
	if key := os.Getenv("FRED_API_KEY"); key != "" {
		registry.Add(mcp.MacroIndicatorTool(fred.New(key)))
	}
	for _, t := range mcp.FXTools(fx.New()) {
		registry.Add(t)
	}
	for _, t := range mcp.CryptoTools(coingecko.New(os.Getenv("COINGECKO_API_KEY"))) {
		registry.Add(t)
	}

	server := mcp.NewServer(registry, os.Stdin, os.Stdout)

	log.Printf("gofi-mcp %s ready, GOFI URL: %s", mcp.ServerVersion, baseURL)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := server.Serve(ctx); err != nil {
		log.Fatalf("gofi-mcp: %v", err)
	}
}
```

- [ ] **Step 3: Run all tests, gofmt, vet**

Run: `go test ./...` and `gofmt -l .` and `go vet ./...`
Expected: all PASS, gofmt empty, vet clean.

- [ ] **Step 4: Update README.md**

Add:
- **Environment variables table**: `GOFI_URL` (default `http://localhost:8080`), `SEC_USER_AGENT` (required for `sec_financials`/`sec_filing`; tools absent when unset — note SEC policy requires a real UA like `Name/version contact@example.com`), `FRED_API_KEY` (required for `macro_indicator`; absent when unset), `COINGECKO_API_KEY` (optional; raises rate limits).
- **Tool list** updated from 15 to up to 21, listing the six new tools with their schemas.
- **Conditional registration note**: the exact `tools/list` count depends on env. Document these counts: 21 (SEC UA + FRED key set), 20 (SEC UA set, no FRED key), 19 (no SEC UA, FRED key set), 18 (neither set). FX and crypto tools are always registered (crypto present even without a CoinGecko key).

- [ ] **Step 5: Live smoke test**

Run with env vars set (or SEC unset to test absence), e.g.:

```powershell
$env:GOFI_URL = "https://finance.hermestech.uk"
$env:FRED_API_KEY = "<your-key>"
$env:SEC_USER_AGENT = "gofi-mcp-test/0.1.0 smoke@example.com"
$init = '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"0.0.1"}}}'
$n = [char]10
$tool1 = '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}'
$tool2 = '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"fx_rate","arguments":{"from":"USD","to":"TRY"}}}'
$tool3 = '{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"crypto_price","arguments":{"id":"bitcoin"}}}'
$tool4 = '{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"sec_financials","arguments":{"ticker":"AAPL"}}}'
"$init$n$tool1$n$tool2$n$tool3$n$tool4$n" | & ".\gofi-mcp.exe" 2>$null
```

Expected: `tools/list` shows 21 tools (with all keys set); `fx_rate USD-TRY`, `crypto_price bitcoin`, `sec_financials AAPL` each return a non-`isError` result. Re-run without `SEC_USER_AGENT` and `FRED_API_KEY` — `sec_*` and `macro_indicator` must be absent from `tools/list`.

- [ ] **Step 6: Commit**

```bash
git add cmd/gofi-mcp/main.go cmd/gofi-mcp/main_test.go README.md
git commit -m "feat: wire multi-source tools into server with conditional registration"
```

---

## Self-Review Notes

**Spec coverage:** Each spec section maps to a task — sources to Tasks 4-8, architecture (httpget/registry) to Tasks 1-3, env-driven registration to Task 9, testing/validation to every task's test steps plus Task 9's smoke test, README to Task 9.
**Placeholder scan:** All steps contain concrete code, paths, and commands; no TBD/TODO/“similar to” references.
**Type consistency:** `httpget.Client` (BaseURL/Header/Timeout/Get) used identically in Tasks 2, 4, 5, 6, 8; `fx.Rate`, `fred.Series`, `sec.Financials`, `coingecko.Price` produced in client tasks and consumed by their tool tasks with matching fields.