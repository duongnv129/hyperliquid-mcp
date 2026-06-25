# Hyperliquid MCP Server (Go) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Go MCP server with full tool parity to the Python `edkdev/hyperliquid-mcp` original (20 tools: account, orders, market data, vaults, utility).

**Architecture:** A single binary (`cmd/main.go`) wires env-var config (`internal/config`) into a Hyperliquid client wrapper (`internal/hlclient`, built on `sonirico/go-hyperliquid`) and registers MCP tools (`internal/tools/*.go`) on an `mcp.Server` (`modelcontextprotocol/go-sdk`), served over stdio or streamable HTTP.

**Tech Stack:** Go 1.25, `github.com/modelcontextprotocol/go-sdk` (MCP), `github.com/sonirico/go-hyperliquid` (Hyperliquid client + EIP-712 signing), `github.com/ethereum/go-ethereum/crypto` (key handling), standard library `net/http`/`testing`.

## Global Constraints

- Module path: `github.com/duongnv129/hyperliquid-mcp`.
- Go version: `go 1.25` in `go.mod` (matches both dependencies' floors).
- Config via env vars only: `HYPERLIQUID_PRIVATE_KEY` (required), `HYPERLIQUID_ACCOUNT_ADDRESS`, `HYPERLIQUID_TESTNET`, `HYPERLIQUID_VAULT_ADDRESS` (all optional). No `.env` file support.
- No client-side minimum-order-value check — the original does not enforce one in code; Hyperliquid's API returns its own validation error, which we surface as-is.
- `hyperliquid.NewExchange`/`NewInfo` **panics** if it cannot fetch `Meta`/`SpotMeta` from the API when constructed with `nil` meta args. `internal/hlclient.New` MUST `recover()` this panic and convert it to a normal `error`.
- Every tools file defines its own minimal consumer interface (structural typing) so handlers are unit-testable without a live network call; `internal/hlclient.Client` satisfies all of them by having the matching concrete methods.
- Bracket orders are placed via a single `Exchange.BulkOrders` call (atomic). TWAP tools are stubs returning "not yet implemented", matching the original.
- Commit after every task using `git add <files> && git commit -m "..."`.

---

## Task 1: Project scaffolding

**Files:**
- Create: `go.mod`
- Create: `cmd/main.go`
- Create: `.gitignore` (append, don't replace — `reference/` is already ignored)

**Interfaces:**
- Produces: a buildable Go module named `github.com/duongnv129/hyperliquid-mcp` with a `main()` that compiles and exits 0, so later tasks can `go build ./...` after every change.

- [ ] **Step 1: Initialize the Go module**

```bash
cd /Users/duong/workspace/hyperliquid-mcp
go mod init github.com/duongnv129/hyperliquid-mcp
go mod edit -go=1.25
```

- [ ] **Step 2: Add dependencies**

```bash
go get github.com/modelcontextprotocol/go-sdk@latest
go get github.com/sonirico/go-hyperliquid@latest
go get github.com/ethereum/go-ethereum@latest
```

- [ ] **Step 3: Write a placeholder `cmd/main.go`**

```go
package main

import "fmt"

func main() {
	fmt.Println("hyperliquid-mcp: scaffolding placeholder")
}
```

- [ ] **Step 4: Verify the module builds**

Run: `go build ./...`
Expected: exits 0, no output, produces no errors.

- [ ] **Step 5: Run it to confirm**

Run: `go run ./cmd`
Expected output: `hyperliquid-mcp: scaffolding placeholder`

- [ ] **Step 6: Commit**

```bash
git add go.mod go.sum cmd/main.go
git commit -m "chore: scaffold Go module and dependencies"
```

---

## Task 2: `internal/config` — environment variable loading

**Files:**
- Create: `internal/config/config.go`
- Test: `internal/config/config_test.go`

**Interfaces:**
- Produces: `config.Config{PrivateKeyHex, AccountAddress, VaultAddress, Testnet}`, `config.Load() (Config, error)`, `(Config).BaseURL() string`.

- [ ] **Step 1: Write the failing tests**

```go
// internal/config/config_test.go
package config

import "testing"

func TestLoad(t *testing.T) {
	cases := []struct {
		name    string
		env     map[string]string
		want    Config
		wantErr string
	}{
		{
			name: "valid mainnet config",
			env: map[string]string{
				"HYPERLIQUID_PRIVATE_KEY": "0x1234567890123456789012345678901234567890123456789012345678901234",
			},
			want: Config{
				PrivateKeyHex: "0x1234567890123456789012345678901234567890123456789012345678901234",
				Testnet:       false,
			},
		},
		{
			name: "testnet flag true is case-insensitive",
			env: map[string]string{
				"HYPERLIQUID_PRIVATE_KEY": "0x1234567890123456789012345678901234567890123456789012345678901234",
				"HYPERLIQUID_TESTNET":     "TRUE",
			},
			want: Config{
				PrivateKeyHex: "0x1234567890123456789012345678901234567890123456789012345678901234",
				Testnet:       true,
			},
		},
		{
			name: "optional addresses are passed through and trimmed",
			env: map[string]string{
				"HYPERLIQUID_PRIVATE_KEY":     "0x1234567890123456789012345678901234567890123456789012345678901234",
				"HYPERLIQUID_ACCOUNT_ADDRESS": "  0xAccount  ",
				"HYPERLIQUID_VAULT_ADDRESS":   "  0xVault  ",
			},
			want: Config{
				PrivateKeyHex:  "0x1234567890123456789012345678901234567890123456789012345678901234",
				AccountAddress: "0xAccount",
				VaultAddress:   "0xVault",
				Testnet:        false,
			},
		},
		{
			name:    "missing private key is an error",
			env:     map[string]string{},
			wantErr: "HYPERLIQUID_PRIVATE_KEY is required",
		},
		{
			name: "malformed private key is an error",
			env: map[string]string{
				"HYPERLIQUID_PRIVATE_KEY": "not-a-key",
			},
			wantErr: "HYPERLIQUID_PRIVATE_KEY must be a 0x-prefixed 64-character hex string",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, key := range []string{
				"HYPERLIQUID_PRIVATE_KEY",
				"HYPERLIQUID_ACCOUNT_ADDRESS",
				"HYPERLIQUID_VAULT_ADDRESS",
				"HYPERLIQUID_TESTNET",
			} {
				t.Setenv(key, "")
			}
			for k, v := range tc.env {
				t.Setenv(k, v)
			}

			got, err := Load()

			if tc.wantErr != "" {
				if err == nil || err.Error() != tc.wantErr {
					t.Fatalf("Load() error = %v, want %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Load() unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("Load() = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestBaseURL(t *testing.T) {
	if (Config{Testnet: false}).BaseURL() != "https://api.hyperliquid.xyz" {
		t.Fatal("mainnet BaseURL mismatch")
	}
	if (Config{Testnet: true}).BaseURL() != "https://api.hyperliquid-testnet.xyz" {
		t.Fatal("testnet BaseURL mismatch")
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/config/...`
Expected: FAIL — `package config: no Go files in .../internal/config` (package doesn't exist yet).

- [ ] **Step 3: Write the implementation**

```go
// internal/config/config.go
package config

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
)

const (
	mainnetBaseURL = "https://api.hyperliquid.xyz"
	testnetBaseURL = "https://api.hyperliquid-testnet.xyz"
)

var privateKeyPattern = regexp.MustCompile(`^0x[0-9a-fA-F]{64}$`)

// Config holds Hyperliquid MCP server configuration loaded from environment
// variables, matching the original Python server's mcp.json `env` block.
type Config struct {
	PrivateKeyHex  string
	AccountAddress string
	VaultAddress   string
	Testnet        bool
}

// Load reads and validates configuration from environment variables.
func Load() (Config, error) {
	pk := strings.TrimSpace(os.Getenv("HYPERLIQUID_PRIVATE_KEY"))
	if pk == "" {
		return Config{}, errors.New("HYPERLIQUID_PRIVATE_KEY is required")
	}
	if !privateKeyPattern.MatchString(pk) {
		return Config{}, fmt.Errorf("HYPERLIQUID_PRIVATE_KEY must be a 0x-prefixed 64-character hex string")
	}

	testnet := strings.EqualFold(strings.TrimSpace(os.Getenv("HYPERLIQUID_TESTNET")), "true")

	return Config{
		PrivateKeyHex:  pk,
		AccountAddress: strings.TrimSpace(os.Getenv("HYPERLIQUID_ACCOUNT_ADDRESS")),
		VaultAddress:   strings.TrimSpace(os.Getenv("HYPERLIQUID_VAULT_ADDRESS")),
		Testnet:        testnet,
	}, nil
}

// BaseURL returns the Hyperliquid REST API base URL for this configuration.
func (c Config) BaseURL() string {
	if c.Testnet {
		return testnetBaseURL
	}
	return mainnetBaseURL
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/config/... -v`
Expected: PASS — all subtests of `TestLoad` and `TestBaseURL` pass.

- [ ] **Step 5: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat: add env-based config loading"
```

---

## Task 3: `internal/hlclient` — Hyperliquid client wrapper

**Files:**
- Create: `internal/hlclient/client.go`
- Create: `internal/hlclient/orders.go`
- Create: `internal/hlclient/info.go`
- Test: `internal/hlclient/client_test.go`
- Test: `internal/hlclient/orders_test.go`
- Test: `internal/hlclient/info_test.go`

**Interfaces:**
- Consumes: `config.Config` (Task 2).
- Produces (used by every `internal/tools/*.go` file in Tasks 4-9):
  - `hlclient.New(ctx context.Context, cfg config.Config) (*hlclient.Client, error)`
  - `(*Client).AccountAddress() string`
  - `(*Client).AssetIndex(coin string) (int, bool)`
  - `(*Client).CoinForAsset(ctx context.Context, asset int) (string, error)`
  - `(*Client).Meta(ctx context.Context) (*hyperliquid.Meta, error)`
  - `(*Client).AllMids(ctx context.Context) (map[string]string, error)`
  - `(*Client).L2Book(ctx context.Context, coin string) (*hyperliquid.L2Book, error)`
  - `(*Client).RecentTrades(ctx context.Context, coin string) ([]hyperliquid.Trade, error)`
  - `(*Client).FundingHistory(ctx context.Context, coin string, startTime int64, endTime *int64) ([]hyperliquid.FundingHistory, error)`
  - `(*Client).Candles(ctx context.Context, coin, interval string, startTime, endTime int64) ([]hyperliquid.Candle, error)`
  - `(*Client).UserState(ctx context.Context, address, dex string) (*hyperliquid.UserState, error)`
  - `(*Client).OpenOrders(ctx context.Context, address, dex string) ([]hyperliquid.OpenOrder, error)`
  - `(*Client).OrderStatus(ctx context.Context, user string, oid int64) (*hyperliquid.OrderQueryResult, error)`
  - `(*Client).UserFills(ctx context.Context, address string) ([]hyperliquid.Fill, error)`
  - `(*Client).UserFundingHistory(ctx context.Context, user string, startTime int64, endTime *int64) ([]hyperliquid.UserFundingHistory, error)`
  - `(*Client).PlaceOrder(ctx context.Context, req hyperliquid.CreateOrderRequest) (hyperliquid.OrderStatus, error)`
  - `(*Client).PlaceBracketOrder(ctx context.Context, p hlclient.BracketOrderParams) (*hyperliquid.APIResponse[hyperliquid.OrderResponse], error)`
  - `(*Client).CancelOrder(ctx context.Context, coin string, oid int64) (*hyperliquid.APIResponse[hyperliquid.CancelOrderResponse], error)`
  - `(*Client).CancelAllOrders(ctx context.Context, user, dex string) (cancelled int, resp *hyperliquid.APIResponse[hyperliquid.CancelOrderResponse], err error)`
  - `(*Client).ModifyOrder(ctx context.Context, req hyperliquid.ModifyOrderRequest) (hyperliquid.OrderStatus, error)`
  - `(*Client).VaultDetails(ctx context.Context, vaultAddress string) (json.RawMessage, error)`
  - `(*Client).VaultPerformance(ctx context.Context, vaultAddress string, startTime int64, endTime *int64) (json.RawMessage, error)`
  - `(*Client).ServerTimeMillis() int64`
  - `hlclient.BracketOrderParams{Coin, IsBuy, Size, EntryPrice, TakeProfitPrice, StopLossPrice}`

### Step group A: client construction (`client.go`)

- [ ] **Step 1: Write the failing construction tests**

```go
// internal/hlclient/client_test.go
package hlclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/duongnv129/hyperliquid-mcp/internal/config"
)

// testServer returns an httptest.Server that answers the /info "meta" and
// "spotMeta" calls go-hyperliquid makes automatically on construction, plus
// whatever extra handler the caller supplies for other request types.
func testServer(t *testing.T, extra func(w http.ResponseWriter, body map[string]any)) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		switch body["type"] {
		case "meta":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"universe": [
					{"name": "BTC", "szDecimals": 5, "maxLeverage": 50, "marginTableId": 1, "onlyIsolated": false, "isDelisted": false},
					{"name": "ETH", "szDecimals": 4, "maxLeverage": 50, "marginTableId": 1, "onlyIsolated": false, "isDelisted": false},
					{"name": "SOL", "szDecimals": 2, "maxLeverage": 20, "marginTableId": 2, "onlyIsolated": false, "isDelisted": false}
				],
				"marginTables": [],
				"collateralToken": 0
			}`))
		case "spotMeta":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"universe": [], "tokens": []}`))
		default:
			if extra != nil {
				extra(w, body)
				return
			}
			t.Fatalf("unexpected request type %q", body["type"])
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// testPrivateKeyHex is a syntactically valid (64 hex chars), arbitrary test
// key — never used against a real network in these tests.
const testPrivateKeyHex = "0x1234567890123456789012345678901234567890123456789012345678901234"

func testConfig(_ string) config.Config {
	return config.Config{PrivateKeyHex: testPrivateKeyHex}
}

func TestNew_Success(t *testing.T) {
	srv := testServer(t, nil)
	cfg := testConfig(srv.URL)

	client, err := newForBaseURL(context.Background(), cfg, srv.URL)
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}
	if client.AccountAddress() == "" {
		t.Fatal("expected a derived account address, got empty string")
	}
}

func TestNew_InvalidPrivateKey(t *testing.T) {
	cfg := config.Config{PrivateKeyHex: "0xnothex000000000000000000000000000000000000000000000000000000000"}

	_, err := newForBaseURL(context.Background(), cfg, "http://example.invalid")
	if err == nil {
		t.Fatal("expected an error for an invalid private key, got nil")
	}
}

func TestNew_RecoversMetaFetchPanic(t *testing.T) {
	// A server that returns 500 for every request forces go-hyperliquid's
	// NewInfo to panic while fetching meta; New must recover and return an error.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	cfg := testConfig(srv.URL)
	_, err := newForBaseURL(context.Background(), cfg, srv.URL)
	if err == nil {
		t.Fatal("expected New to convert the underlying panic into an error")
	}
}

func TestAssetIndexAndCoinForAsset(t *testing.T) {
	srv := testServer(t, nil)
	cfg := testConfig(srv.URL)
	client, err := newForBaseURL(context.Background(), cfg, srv.URL)
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	idx, ok := client.AssetIndex("SOL")
	if !ok || idx != 2 {
		t.Fatalf("AssetIndex(SOL) = (%d, %v), want (2, true)", idx, ok)
	}

	coin, err := client.CoinForAsset(context.Background(), 1)
	if err != nil || coin != "ETH" {
		t.Fatalf("CoinForAsset(1) = (%q, %v), want (ETH, nil)", coin, err)
	}

	if _, err := client.CoinForAsset(context.Background(), 99); err == nil {
		t.Fatal("expected an out-of-range asset index to error")
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/hlclient/... -run 'TestNew|TestAssetIndex' -v`
Expected: FAIL — `package hlclient` has no `New`/`newForBaseURL`/`Client` symbols yet.

- [ ] **Step 3: Write `client.go`**

```go
// internal/hlclient/client.go
package hlclient

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	hyperliquid "github.com/sonirico/go-hyperliquid"

	"github.com/duongnv129/hyperliquid-mcp/internal/config"
)

// Client wraps a sonirico/go-hyperliquid Exchange/Info pair with the
// additional operations the MCP tools need (bracket orders, cancel-all,
// vault queries, server time) that the underlying SDK does not expose
// directly.
type Client struct {
	exchange       *hyperliquid.Exchange
	info           *hyperliquid.Info
	httpClient     *http.Client
	baseURL        string
	accountAddress string
}

// New constructs a Client from configuration, deriving the trading account
// address from the private key when HYPERLIQUID_ACCOUNT_ADDRESS is unset.
func New(ctx context.Context, cfg config.Config) (*Client, error) {
	return newForBaseURL(ctx, cfg, cfg.BaseURL())
}

// newForBaseURL is the testable constructor: it takes the base URL
// explicitly so tests can point it at an httptest.Server.
func newForBaseURL(ctx context.Context, cfg config.Config, baseURL string) (client *Client, err error) {
	defer func() {
		if r := recover(); r != nil {
			client = nil
			err = fmt.Errorf("failed to initialize hyperliquid client: %v", r)
		}
	}()

	privateKey, perr := crypto.HexToECDSA(strings.TrimPrefix(cfg.PrivateKeyHex, "0x"))
	if perr != nil {
		return nil, fmt.Errorf("invalid private key: %w", perr)
	}

	pub, ok := privateKey.Public().(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("invalid private key: could not derive public key")
	}
	derivedAddress := crypto.PubkeyToAddress(*pub).Hex()

	accountAddress := cfg.AccountAddress
	if accountAddress == "" {
		accountAddress = derivedAddress
	}

	exchange := hyperliquid.NewExchange(
		ctx,
		privateKey,
		baseURL,
		nil, // Meta fetched automatically
		cfg.VaultAddress,
		accountAddress,
		nil, // SpotMeta fetched automatically
		nil, // PerpDexs fetched automatically
	)

	return &Client{
		exchange:       exchange,
		info:           exchange.Info(),
		httpClient:     &http.Client{Timeout: 30 * time.Second},
		baseURL:        baseURL,
		accountAddress: accountAddress,
	}, nil
}

// AccountAddress returns the trading account address (configured, or
// derived from the private key if HYPERLIQUID_ACCOUNT_ADDRESS was unset).
func (c *Client) AccountAddress() string {
	return c.accountAddress
}

// ServerTimeMillis returns the current time as Unix milliseconds, matching
// the original Python server's get_server_time (a local clock read, not an
// API call).
func (c *Client) ServerTimeMillis() int64 {
	return time.Now().UnixMilli()
}

// AssetIndex resolves a coin symbol (e.g. "SOL") to its asset index, using
// the metadata go-hyperliquid already cached at construction time.
func (c *Client) AssetIndex(coin string) (int, bool) {
	return c.info.CoinToAsset(coin)
}

// CoinForAsset resolves an asset index back to its coin symbol by looking
// it up in the exchange's universe metadata.
func (c *Client) CoinForAsset(ctx context.Context, asset int) (string, error) {
	meta, err := c.info.Meta(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to fetch meta: %w", err)
	}
	if asset < 0 || asset >= len(meta.Universe) {
		return "", fmt.Errorf("asset index %d out of range (0-%d)", asset, len(meta.Universe)-1)
	}
	return meta.Universe[asset].Name, nil
}

// postInfo issues a raw POST to {baseURL}/info for endpoints go-hyperliquid
// does not expose as a typed method (recentTrades, vaultDetails,
// vaultPerformance), decoding the JSON response into out.
func (c *Client) postInfo(ctx context.Context, payload map[string]any, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/info", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request %q failed: %w", payload["type"], err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response for %q: %w", payload["type"], err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("request %q failed with status %d: %s", payload["type"], resp.StatusCode, string(respBody))
	}

	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("failed to decode response for %q: %w", payload["type"], err)
	}
	return nil
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/hlclient/... -run 'TestNew|TestAssetIndex' -v`
Expected: PASS for all four tests.

- [ ] **Step 5: Commit**

```bash
git add internal/hlclient/client.go internal/hlclient/client_test.go
git commit -m "feat: add hlclient construction with panic recovery"
```

### Step group B: read-only info passthroughs (`info.go`)

- [ ] **Step 1: Write the failing tests**

```go
// internal/hlclient/info_test.go
package hlclient

import (
	"context"
	"net/http"
	"testing"
)

func TestMetaAllMidsL2Book(t *testing.T) {
	srv := testServer(t, func(w http.ResponseWriter, body map[string]any) {
		w.Header().Set("Content-Type", "application/json")
		switch body["type"] {
		case "allMids":
			_, _ = w.Write([]byte(`{"BTC": "65000.5", "ETH": "3000.1"}`))
		case "l2Book":
			_, _ = w.Write([]byte(`{"coin": "BTC", "time": 1700000000000, "levels": [[{"n":1,"px":65000,"sz":1.5}],[{"n":1,"px":65001,"sz":2}]]}`))
		default:
			t.Fatalf("unexpected request type %q", body["type"])
		}
	})
	cfg := testConfig(srv.URL)
	client, err := newForBaseURL(context.Background(), cfg, srv.URL)
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	meta, err := client.Meta(context.Background())
	if err != nil || len(meta.Universe) != 3 {
		t.Fatalf("Meta() = (%+v, %v), want 3 universe entries", meta, err)
	}

	mids, err := client.AllMids(context.Background())
	if err != nil || mids["BTC"] != "65000.5" {
		t.Fatalf("AllMids() = (%+v, %v)", mids, err)
	}

	book, err := client.L2Book(context.Background(), "BTC")
	if err != nil || book.Coin != "BTC" || len(book.Levels) != 2 {
		t.Fatalf("L2Book() = (%+v, %v)", book, err)
	}
}

func TestUserStateOpenOrders(t *testing.T) {
	srv := testServer(t, func(w http.ResponseWriter, body map[string]any) {
		w.Header().Set("Content-Type", "application/json")
		switch body["type"] {
		case "clearinghouseState":
			_, _ = w.Write([]byte(`{"assetPositions": [], "crossMarginSummary": {"accountValue":"100","totalMarginUsed":"0","totalNtlPos":"0","totalRawUsd":"100"}, "marginSummary": {"accountValue":"100","totalMarginUsed":"0","totalNtlPos":"0","totalRawUsd":"100"}, "withdrawable": "100"}`))
		case "openOrders":
			_, _ = w.Write([]byte(`[{"coin":"BTC","limitPx":"65000","oid":1,"origSz":"0.1","side":"B","sz":"0.1","timestamp":1700000000000}]`))
		default:
			t.Fatalf("unexpected request type %q", body["type"])
		}
	})
	cfg := testConfig(srv.URL)
	client, err := newForBaseURL(context.Background(), cfg, srv.URL)
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	state, err := client.UserState(context.Background(), client.AccountAddress(), "")
	if err != nil || state.Withdrawable != "100" {
		t.Fatalf("UserState() = (%+v, %v)", state, err)
	}

	orders, err := client.OpenOrders(context.Background(), client.AccountAddress(), "")
	if err != nil || len(orders) != 1 || orders[0].Coin != "BTC" {
		t.Fatalf("OpenOrders() = (%+v, %v)", orders, err)
	}
}

func TestFundingHistoryCandlesRecentTrades(t *testing.T) {
	srv := testServer(t, func(w http.ResponseWriter, body map[string]any) {
		w.Header().Set("Content-Type", "application/json")
		switch body["type"] {
		case "fundingHistory":
			_, _ = w.Write([]byte(`[{"coin":"BTC","fundingRate":"0.0001","premium":"0.0002","time":1700000000000}]`))
		case "candleSnapshot":
			_, _ = w.Write([]byte(`[{"t":1700000000000,"T":1700000060000,"i":"1m","n":10,"o":"65000","h":"65010","l":"64990","c":"65005","s":"BTC","v":"12.5"}]`))
		case "recentTrades":
			_, _ = w.Write([]byte(`[{"coin":"BTC","side":"B","px":"65000","sz":"0.1","time":1700000000000,"hash":"0xabc","tid":1,"users":["0x1","0x2"]}]`))
		default:
			t.Fatalf("unexpected request type %q", body["type"])
		}
	})
	cfg := testConfig(srv.URL)
	client, err := newForBaseURL(context.Background(), cfg, srv.URL)
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	funding, err := client.FundingHistory(context.Background(), "BTC", 1700000000000, nil)
	if err != nil || len(funding) != 1 || funding[0].FundingRate != "0.0001" {
		t.Fatalf("FundingHistory() = (%+v, %v)", funding, err)
	}

	candles, err := client.Candles(context.Background(), "BTC", "1m", 1700000000000, 1700000060000)
	if err != nil || len(candles) != 1 || candles[0].Close != "65005" {
		t.Fatalf("Candles() = (%+v, %v)", candles, err)
	}

	trades, err := client.RecentTrades(context.Background(), "BTC")
	if err != nil || len(trades) != 1 || trades[0].Hash != "0xabc" {
		t.Fatalf("RecentTrades() = (%+v, %v)", trades, err)
	}
}

func TestUserFillsUserFundingOrderStatus(t *testing.T) {
	srv := testServer(t, func(w http.ResponseWriter, body map[string]any) {
		w.Header().Set("Content-Type", "application/json")
		switch body["type"] {
		case "userFills":
			_, _ = w.Write([]byte(`[{"closedPnl":"10","coin":"BTC","crossed":true,"dir":"Open Long","hash":"0xabc","oid":1,"px":"65000","side":"B","startPosition":"0","sz":"0.1","time":1700000000000,"fee":"0.1","feeToken":"USDC","tid":1}]`))
		case "userFunding":
			_, _ = w.Write([]byte(`[{"delta":{"coin":"BTC","fundingRate":"0.0001","size":"0.1","type":"funding","usdc":"0.01"},"hash":"0xabc","time":1700000000000}]`))
		case "orderStatus":
			_, _ = w.Write([]byte(`{"status":"order","order":{}}`))
		default:
			t.Fatalf("unexpected request type %q", body["type"])
		}
	})
	cfg := testConfig(srv.URL)
	client, err := newForBaseURL(context.Background(), cfg, srv.URL)
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	fills, err := client.UserFills(context.Background(), client.AccountAddress())
	if err != nil || len(fills) != 1 || fills[0].Coin != "BTC" {
		t.Fatalf("UserFills() = (%+v, %v)", fills, err)
	}

	funding, err := client.UserFundingHistory(context.Background(), client.AccountAddress(), 1700000000000, nil)
	if err != nil || len(funding) != 1 || funding[0].Delta.Coin != "BTC" {
		t.Fatalf("UserFundingHistory() = (%+v, %v)", funding, err)
	}

	status, err := client.OrderStatus(context.Background(), client.AccountAddress(), 1)
	if err != nil || status.Status != "order" {
		t.Fatalf("OrderStatus() = (%+v, %v)", status, err)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/hlclient/... -run 'TestMeta|TestUserState|TestFundingHistory|TestUserFills' -v`
Expected: FAIL — methods don't exist yet.

- [ ] **Step 3: Write `info.go`**

```go
// internal/hlclient/info.go
package hlclient

import (
	"context"
	"fmt"

	hyperliquid "github.com/sonirico/go-hyperliquid"
)

func (c *Client) Meta(ctx context.Context) (*hyperliquid.Meta, error) {
	meta, err := c.info.Meta(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch meta: %w", err)
	}
	return meta, nil
}

func (c *Client) AllMids(ctx context.Context) (map[string]string, error) {
	mids, err := c.info.AllMids(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch all mids: %w", err)
	}
	return mids, nil
}

func (c *Client) L2Book(ctx context.Context, coin string) (*hyperliquid.L2Book, error) {
	book, err := c.info.L2Snapshot(ctx, coin)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch order book for %s: %w", coin, err)
	}
	return book, nil
}

func (c *Client) FundingHistory(
	ctx context.Context,
	coin string,
	startTime int64,
	endTime *int64,
) ([]hyperliquid.FundingHistory, error) {
	history, err := c.info.FundingHistory(ctx, coin, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch funding history for %s: %w", coin, err)
	}
	return history, nil
}

func (c *Client) Candles(
	ctx context.Context,
	coin, interval string,
	startTime, endTime int64,
) ([]hyperliquid.Candle, error) {
	candles, err := c.info.CandlesSnapshot(ctx, coin, interval, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch candles for %s: %w", coin, err)
	}
	return candles, nil
}

func (c *Client) UserState(ctx context.Context, address, dex string) (*hyperliquid.UserState, error) {
	state, err := c.info.UserState(ctx, address, dex)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user state for %s: %w", address, err)
	}
	return state, nil
}

func (c *Client) OpenOrders(ctx context.Context, address, dex string) ([]hyperliquid.OpenOrder, error) {
	orders, err := c.info.OpenOrders(ctx, address, dex)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch open orders for %s: %w", address, err)
	}
	return orders, nil
}

func (c *Client) OrderStatus(ctx context.Context, user string, oid int64) (*hyperliquid.OrderQueryResult, error) {
	status, err := c.info.QueryOrderByOid(ctx, user, oid)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch order status for oid %d: %w", oid, err)
	}
	return status, nil
}

func (c *Client) UserFills(ctx context.Context, address string) ([]hyperliquid.Fill, error) {
	fills, err := c.info.UserFills(ctx, hyperliquid.UserFillsParams{Address: address})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch fills for %s: %w", address, err)
	}
	return fills, nil
}

func (c *Client) UserFundingHistory(
	ctx context.Context,
	user string,
	startTime int64,
	endTime *int64,
) ([]hyperliquid.UserFundingHistory, error) {
	history, err := c.info.UserFundingHistory(ctx, user, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user funding history for %s: %w", user, err)
	}
	return history, nil
}

// RecentTrades calls the "recentTrades" /info endpoint, which go-hyperliquid
// does not expose as a typed method. It reuses the SDK's exported Trade type
// (same shape as the WS trades channel) so callers get a typed result.
func (c *Client) RecentTrades(ctx context.Context, coin string) ([]hyperliquid.Trade, error) {
	var trades []hyperliquid.Trade
	if err := c.postInfo(ctx, map[string]any{"type": "recentTrades", "coin": coin}, &trades); err != nil {
		return nil, fmt.Errorf("failed to fetch recent trades for %s: %w", coin, err)
	}
	return trades, nil
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/hlclient/... -run 'TestMeta|TestUserState|TestFundingHistory|TestUserFills' -v`
Expected: PASS for all four tests.

- [ ] **Step 5: Commit**

```bash
git add internal/hlclient/info.go internal/hlclient/info_test.go
git commit -m "feat: add hlclient read-only info passthroughs"
```

### Step group C: orders — place, bracket, cancel, modify (`orders.go`)

- [ ] **Step 1: Write the failing tests**

```go
// internal/hlclient/orders_test.go
package hlclient

import (
	"context"
	"net/http"
	"testing"

	hyperliquid "github.com/sonirico/go-hyperliquid"
)

func decodeExchangeAction(t *testing.T, body map[string]any) map[string]any {
	t.Helper()
	action, ok := body["action"].(map[string]any)
	if !ok {
		t.Fatalf("request body has no action object: %+v", body)
	}
	return action
}

func TestPlaceOrder(t *testing.T) {
	var captured map[string]any
	srv := testServer(t, func(w http.ResponseWriter, body map[string]any) {
		w.Header().Set("Content-Type", "application/json")
		if body["type"] == "order" {
			captured = decodeExchangeAction(t, body)
		}
		_, _ = w.Write([]byte(`{"status":"ok","response":{"type":"order","data":{"statuses":[{"resting":{"oid":1,"status":"resting"}}]}}}`))
	})
	cfg := testConfig(srv.URL)
	client, err := newForBaseURL(context.Background(), cfg, srv.URL)
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	status, err := client.PlaceOrder(context.Background(), hyperliquid.CreateOrderRequest{
		Coin:  "BTC",
		IsBuy: true,
		Price: 65000,
		Size:  0.1,
		OrderType: hyperliquid.OrderType{
			Limit: &hyperliquid.LimitOrderType{Tif: hyperliquid.TifGtc},
		},
	})
	if err != nil {
		t.Fatalf("PlaceOrder() unexpected error: %v", err)
	}
	if status.Resting == nil || status.Resting.Oid != 1 {
		t.Fatalf("PlaceOrder() status = %+v, want a resting order with oid 1", status)
	}
	if captured == nil {
		t.Fatal("expected the order action to reach the test server")
	}
}

func TestPlaceBracketOrder(t *testing.T) {
	var orderCount int
	srv := testServer(t, func(w http.ResponseWriter, body map[string]any) {
		w.Header().Set("Content-Type", "application/json")
		if body["type"] == "order" {
			action := decodeExchangeAction(t, body)
			orders, _ := action["orders"].([]any)
			orderCount = len(orders)
		}
		_, _ = w.Write([]byte(`{"status":"ok","response":{"type":"order","data":{"statuses":[
			{"resting":{"oid":1,"status":"resting"}},
			{"resting":{"oid":2,"status":"resting"}},
			{"resting":{"oid":3,"status":"resting"}}
		]}}}`))
	})
	cfg := testConfig(srv.URL)
	client, err := newForBaseURL(context.Background(), cfg, srv.URL)
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	resp, err := client.PlaceBracketOrder(context.Background(), BracketOrderParams{
		Coin:            "SOL",
		IsBuy:           true,
		Size:            4.12,
		EntryPrice:      218.0,
		TakeProfitPrice: 219.5,
		StopLossPrice:   216.8,
	})
	if err != nil {
		t.Fatalf("PlaceBracketOrder() unexpected error: %v", err)
	}
	if len(resp.Data.Statuses) != 3 {
		t.Fatalf("PlaceBracketOrder() returned %d statuses, want 3", len(resp.Data.Statuses))
	}
	if orderCount != 3 {
		t.Fatalf("expected a single bulk request with 3 orders, server saw %d", orderCount)
	}
}

func TestCancelOrder(t *testing.T) {
	srv := testServer(t, func(w http.ResponseWriter, body map[string]any) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","response":{"type":"cancel","data":{"statuses":["success"]}}}`))
	})
	cfg := testConfig(srv.URL)
	client, err := newForBaseURL(context.Background(), cfg, srv.URL)
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	resp, err := client.CancelOrder(context.Background(), "BTC", 1)
	if err != nil {
		t.Fatalf("CancelOrder() unexpected error: %v", err)
	}
	if !resp.Ok {
		t.Fatalf("CancelOrder() response not ok: %+v", resp)
	}
}

func TestCancelAllOrders_NoOpenOrders(t *testing.T) {
	srv := testServer(t, func(w http.ResponseWriter, body map[string]any) {
		w.Header().Set("Content-Type", "application/json")
		if body["type"] == "openOrders" {
			_, _ = w.Write([]byte(`[]`))
			return
		}
		t.Fatalf("unexpected request type %q when there are no open orders", body["type"])
	})
	cfg := testConfig(srv.URL)
	client, err := newForBaseURL(context.Background(), cfg, srv.URL)
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	cancelled, resp, err := client.CancelAllOrders(context.Background(), client.AccountAddress(), "")
	if err != nil {
		t.Fatalf("CancelAllOrders() unexpected error: %v", err)
	}
	if cancelled != 0 || resp != nil {
		t.Fatalf("CancelAllOrders() = (%d, %+v), want (0, nil)", cancelled, resp)
	}
}

func TestCancelAllOrders_WithOpenOrders(t *testing.T) {
	srv := testServer(t, func(w http.ResponseWriter, body map[string]any) {
		w.Header().Set("Content-Type", "application/json")
		switch body["type"] {
		case "openOrders":
			_, _ = w.Write([]byte(`[
				{"coin":"BTC","limitPx":"65000","oid":1,"origSz":"0.1","side":"B","sz":"0.1","timestamp":1700000000000},
				{"coin":"ETH","limitPx":"3000","oid":2,"origSz":"1","side":"A","sz":"1","timestamp":1700000000000}
			]`))
		case "cancel":
			_, _ = w.Write([]byte(`{"status":"ok","response":{"type":"cancel","data":{"statuses":["success","success"]}}}`))
		default:
			t.Fatalf("unexpected request type %q", body["type"])
		}
	})
	cfg := testConfig(srv.URL)
	client, err := newForBaseURL(context.Background(), cfg, srv.URL)
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	cancelled, resp, err := client.CancelAllOrders(context.Background(), client.AccountAddress(), "")
	if err != nil {
		t.Fatalf("CancelAllOrders() unexpected error: %v", err)
	}
	if cancelled != 2 {
		t.Fatalf("CancelAllOrders() cancelled = %d, want 2", cancelled)
	}
	if resp == nil || !resp.Ok {
		t.Fatalf("CancelAllOrders() resp = %+v, want ok", resp)
	}
}

func TestModifyOrder(t *testing.T) {
	srv := testServer(t, func(w http.ResponseWriter, body map[string]any) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","response":{"type":"modify","data":{"statuses":[{"resting":{"oid":1,"status":"resting"}}]}}}`))
	})
	cfg := testConfig(srv.URL)
	client, err := newForBaseURL(context.Background(), cfg, srv.URL)
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	oid := int64(1)
	status, err := client.ModifyOrder(context.Background(), hyperliquid.ModifyOrderRequest{
		Oid: &oid,
		Order: hyperliquid.CreateOrderRequest{
			Coin:      "BTC",
			IsBuy:     true,
			Price:     64000,
			Size:      0.1,
			OrderType: hyperliquid.OrderType{Limit: &hyperliquid.LimitOrderType{Tif: hyperliquid.TifGtc}},
		},
	})
	if err != nil {
		t.Fatalf("ModifyOrder() unexpected error: %v", err)
	}
	if status.Resting == nil || status.Resting.Oid != 1 {
		t.Fatalf("ModifyOrder() status = %+v", status)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/hlclient/... -run 'TestPlaceOrder|TestPlaceBracketOrder|TestCancelOrder|TestCancelAllOrders|TestModifyOrder' -v`
Expected: FAIL — `PlaceOrder`, `PlaceBracketOrder`, `BracketOrderParams`, `CancelOrder`, `CancelAllOrders`, `ModifyOrder` are undefined.

- [ ] **Step 3: Write `orders.go`**

```go
// internal/hlclient/orders.go
package hlclient

import (
	"context"
	"fmt"

	hyperliquid "github.com/sonirico/go-hyperliquid"
)

// BracketOrderParams describes an entry + take-profit + stop-loss bracket,
// placed atomically via a single BulkOrders call.
type BracketOrderParams struct {
	Coin            string
	IsBuy           bool
	Size            float64
	EntryPrice      float64
	TakeProfitPrice float64
	StopLossPrice   float64
}

func (c *Client) PlaceOrder(ctx context.Context, req hyperliquid.CreateOrderRequest) (hyperliquid.OrderStatus, error) {
	status, err := c.exchange.Order(ctx, req, nil)
	if err != nil {
		return hyperliquid.OrderStatus{}, fmt.Errorf("failed to place order for %s: %w", req.Coin, err)
	}
	return status, nil
}

// PlaceBracketOrder submits the entry limit order plus reduce-only
// take-profit and stop-loss trigger orders in a single BulkOrders call, so
// they succeed or fail together at the API level (matching the original
// Python server's bracket order behavior).
func (c *Client) PlaceBracketOrder(
	ctx context.Context,
	p BracketOrderParams,
) (*hyperliquid.APIResponse[hyperliquid.OrderResponse], error) {
	closeSide := !p.IsBuy

	entry := hyperliquid.CreateOrderRequest{
		Coin:      p.Coin,
		IsBuy:     p.IsBuy,
		Price:     p.EntryPrice,
		Size:      p.Size,
		OrderType: hyperliquid.OrderType{Limit: &hyperliquid.LimitOrderType{Tif: hyperliquid.TifGtc}},
	}

	takeProfit := hyperliquid.CreateOrderRequest{
		Coin:       p.Coin,
		IsBuy:      closeSide,
		Price:      p.TakeProfitPrice,
		Size:       p.Size,
		ReduceOnly: true,
		OrderType: hyperliquid.OrderType{Trigger: &hyperliquid.TriggerOrderType{
			TriggerPx: p.TakeProfitPrice,
			IsMarket:  false,
			Tpsl:      hyperliquid.TakeProfit,
		}},
	}

	stopLoss := hyperliquid.CreateOrderRequest{
		Coin:       p.Coin,
		IsBuy:      closeSide,
		Price:      p.StopLossPrice,
		Size:       p.Size,
		ReduceOnly: true,
		OrderType: hyperliquid.OrderType{Trigger: &hyperliquid.TriggerOrderType{
			TriggerPx: p.StopLossPrice,
			IsMarket:  false,
			Tpsl:      hyperliquid.StopLoss,
		}},
	}

	resp, err := c.exchange.BulkOrders(ctx, []hyperliquid.CreateOrderRequest{entry, takeProfit, stopLoss}, nil)
	if err != nil {
		return resp, fmt.Errorf("failed to place bracket order for %s: %w", p.Coin, err)
	}
	return resp, nil
}

func (c *Client) CancelOrder(
	ctx context.Context,
	coin string,
	oid int64,
) (*hyperliquid.APIResponse[hyperliquid.CancelOrderResponse], error) {
	resp, err := c.exchange.Cancel(ctx, coin, oid)
	if err != nil {
		return resp, fmt.Errorf("failed to cancel order %d for %s: %w", oid, coin, err)
	}
	return resp, nil
}

// CancelAllOrders fetches the user's open orders and cancels all of them in
// a single bulk request, matching the original Python server's
// fetch-then-cancel approach. It returns (0, nil, nil) when there is
// nothing to cancel.
func (c *Client) CancelAllOrders(
	ctx context.Context,
	user, dex string,
) (cancelled int, resp *hyperliquid.APIResponse[hyperliquid.CancelOrderResponse], err error) {
	orders, err := c.info.OpenOrders(ctx, user, dex)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to fetch open orders for %s: %w", user, err)
	}
	if len(orders) == 0 {
		return 0, nil, nil
	}

	requests := make([]hyperliquid.CancelOrderRequest, len(orders))
	for i, o := range orders {
		requests[i] = hyperliquid.CancelOrderRequest{Coin: o.Coin, OrderID: o.Oid}
	}

	resp, err = c.exchange.BulkCancel(ctx, requests)
	if err != nil {
		return 0, resp, fmt.Errorf("failed to cancel %d open orders for %s: %w", len(orders), user, err)
	}
	return len(orders), resp, nil
}

func (c *Client) ModifyOrder(ctx context.Context, req hyperliquid.ModifyOrderRequest) (hyperliquid.OrderStatus, error) {
	status, err := c.exchange.ModifyOrder(ctx, req)
	if err != nil {
		return hyperliquid.OrderStatus{}, fmt.Errorf("failed to modify order: %w", err)
	}
	return status, nil
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/hlclient/... -run 'TestPlaceOrder|TestPlaceBracketOrder|TestCancelOrder|TestCancelAllOrders|TestModifyOrder' -v`
Expected: PASS for all six tests.

- [ ] **Step 5: Commit**

```bash
git add internal/hlclient/orders.go internal/hlclient/orders_test.go
git commit -m "feat: add hlclient order placement, bracket orders, cancel, modify"
```

### Step group D: vault raw HTTP queries (`vault.go`)

- [ ] **Step 1: Write the failing tests**

```go
// internal/hlclient/vault_test.go
package hlclient

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestVaultDetailsAndPerformance(t *testing.T) {
	srv := testServer(t, func(w http.ResponseWriter, body map[string]any) {
		w.Header().Set("Content-Type", "application/json")
		switch body["type"] {
		case "vaultDetails":
			if body["vaultAddress"] != "0xVault" {
				t.Fatalf("vaultDetails request missing vaultAddress: %+v", body)
			}
			_, _ = w.Write([]byte(`{"name":"Test Vault","vaultAddress":"0xVault","leader":"0xLeader"}`))
		default:
			t.Fatalf("unexpected request type %q", body["type"])
		}
	})
	cfg := testConfig(srv.URL)
	client, err := newForBaseURL(context.Background(), cfg, srv.URL)
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	raw, err := client.VaultDetails(context.Background(), "0xVault")
	if err != nil {
		t.Fatalf("VaultDetails() unexpected error: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("VaultDetails() returned invalid JSON: %v", err)
	}
	if decoded["name"] != "Test Vault" {
		t.Fatalf("VaultDetails() = %+v, want name=Test Vault", decoded)
	}
}

func TestVaultPerformance(t *testing.T) {
	srv := testServer(t, func(w http.ResponseWriter, body map[string]any) {
		w.Header().Set("Content-Type", "application/json")
		switch body["type"] {
		case "vaultDetails":
			if body["startTime"] == nil {
				t.Fatalf("vaultPerformance-style request missing startTime: %+v", body)
			}
			_, _ = w.Write([]byte(`{"vaultAddress":"0xVault","pnls":[]}`))
		default:
			t.Fatalf("unexpected request type %q", body["type"])
		}
	})
	cfg := testConfig(srv.URL)
	client, err := newForBaseURL(context.Background(), cfg, srv.URL)
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	raw, err := client.VaultPerformance(context.Background(), "0xVault", 1700000000000, nil)
	if err != nil {
		t.Fatalf("VaultPerformance() unexpected error: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("VaultPerformance() returned invalid JSON: %v", err)
	}
	if decoded["vaultAddress"] != "0xVault" {
		t.Fatalf("VaultPerformance() = %+v", decoded)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/hlclient/... -run 'TestVaultDetails|TestVaultPerformance' -v`
Expected: FAIL — `VaultDetails`/`VaultPerformance` are undefined.

- [ ] **Step 3: Write `vault.go`**

The Hyperliquid `vaultDetails` `/info` endpoint's response shape isn't exposed as a Go type by go-hyperliquid or documented in our reference material, so this passes the raw decoded JSON through (`json.RawMessage`) rather than guessing at typed fields. The MCP tool layer (Task 8) returns this raw JSON directly as the tool's structured content.

```go
// internal/hlclient/vault.go
package hlclient

import (
	"context"
	"encoding/json"
	"fmt"
)

// VaultDetails calls the "vaultDetails" /info endpoint. go-hyperliquid does
// not expose this as a typed method, so the response is returned as raw
// JSON for the caller to pass through or inspect.
func (c *Client) VaultDetails(ctx context.Context, vaultAddress string) (json.RawMessage, error) {
	var raw json.RawMessage
	payload := map[string]any{
		"type":         "vaultDetails",
		"vaultAddress": vaultAddress,
	}
	if err := c.postInfo(ctx, payload, &raw); err != nil {
		return nil, fmt.Errorf("failed to fetch vault details for %s: %w", vaultAddress, err)
	}
	return raw, nil
}

// VaultPerformance calls "vaultDetails" with a time range, matching the
// original Python server's vault_details(vault_address, start_time, end_time)
// overload (the underlying Hyperliquid endpoint is the same; supplying a
// time range returns performance/PnL history instead of static details).
func (c *Client) VaultPerformance(
	ctx context.Context,
	vaultAddress string,
	startTime int64,
	endTime *int64,
) (json.RawMessage, error) {
	var raw json.RawMessage
	payload := map[string]any{
		"type":         "vaultDetails",
		"vaultAddress": vaultAddress,
		"startTime":    startTime,
	}
	if endTime != nil {
		payload["endTime"] = *endTime
	}
	if err := c.postInfo(ctx, payload, &raw); err != nil {
		return nil, fmt.Errorf("failed to fetch vault performance for %s: %w", vaultAddress, err)
	}
	return raw, nil
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/hlclient/... -v`
Expected: PASS for every test in the package (all of Step Groups A-D).

- [ ] **Step 5: Commit**

```bash
git add internal/hlclient/vault.go internal/hlclient/vault_test.go
git commit -m "feat: add hlclient vault details/performance raw queries"
```

---

## Task 4: `internal/tools/account.go` — account & position tools

**Files:**
- Create: `internal/tools/account.go`
- Test: `internal/tools/account_test.go`

**Interfaces:**
- Consumes: `hlclient.Client` (Task 3) structurally via a local `accountClient` interface; `hyperliquid.UserState` (from `sonirico/go-hyperliquid`).
- Produces: `tools.RegisterAccountTools(server *mcp.Server, client accountClient)`, called from `cmd/main.go` (Task 10).

- [ ] **Step 1: Write the failing tests**

```go
// internal/tools/account_test.go
package tools

import (
	"context"
	"errors"
	"testing"

	hyperliquid "github.com/sonirico/go-hyperliquid"
)

type fakeAccountClient struct {
	state          *hyperliquid.UserState
	err            error
	accountAddress string
	gotAddress     string
	gotDex         string
}

func (f *fakeAccountClient) UserState(ctx context.Context, address, dex string) (*hyperliquid.UserState, error) {
	f.gotAddress = address
	f.gotDex = dex
	if f.err != nil {
		return nil, f.err
	}
	return f.state, nil
}

func (f *fakeAccountClient) AccountAddress() string { return f.accountAddress }

func sampleUserState() *hyperliquid.UserState {
	return &hyperliquid.UserState{
		AssetPositions: []hyperliquid.AssetPosition{
			{Type: "oneWay", Position: hyperliquid.Position{Coin: "BTC", Szi: "0.1"}},
		},
		MarginSummary: hyperliquid.MarginSummary{
			AccountValue:    "1000",
			TotalMarginUsed: "100",
		},
		Withdrawable: "900",
	}
}

func TestGetAccountInfo(t *testing.T) {
	client := &fakeAccountClient{accountAddress: "0xConfigured", state: sampleUserState()}

	result, _, err := getAccountInfoHandler(client)(context.Background(), nil, GetAccountInfoArgs{})
	if err != nil {
		t.Fatalf("handler unexpected error: %v", err)
	}
	if client.gotAddress != "0xConfigured" {
		t.Fatalf("expected default address to be used, got %q", client.gotAddress)
	}
	if result == nil || len(result.Content) == 0 {
		t.Fatal("expected non-empty tool result content")
	}
}

func TestGetAccountInfo_ExplicitAddress(t *testing.T) {
	client := &fakeAccountClient{accountAddress: "0xConfigured", state: sampleUserState()}

	_, _, err := getAccountInfoHandler(client)(context.Background(), nil, GetAccountInfoArgs{UserAddress: "0xOther", Dex: "spot"})
	if err != nil {
		t.Fatalf("handler unexpected error: %v", err)
	}
	if client.gotAddress != "0xOther" || client.gotDex != "spot" {
		t.Fatalf("expected explicit address/dex to be passed through, got (%q, %q)", client.gotAddress, client.gotDex)
	}
}

func TestGetAccountInfo_Error(t *testing.T) {
	client := &fakeAccountClient{accountAddress: "0xConfigured", err: errors.New("network down")}

	_, _, err := getAccountInfoHandler(client)(context.Background(), nil, GetAccountInfoArgs{})
	if err == nil {
		t.Fatal("expected handler to surface the client error")
	}
}

func TestGetPositions(t *testing.T) {
	client := &fakeAccountClient{accountAddress: "0xConfigured", state: sampleUserState()}

	_, out, err := getPositionsHandler(client)(context.Background(), nil, GetPositionsArgs{})
	if err != nil {
		t.Fatalf("handler unexpected error: %v", err)
	}
	if len(out.Data) != 1 || out.Data[0].Position.Coin != "BTC" {
		t.Fatalf("getPositions output = %+v, want one BTC position", out)
	}
}

func TestGetBalance(t *testing.T) {
	client := &fakeAccountClient{accountAddress: "0xConfigured", state: sampleUserState()}

	_, out, err := getBalanceHandler(client)(context.Background(), nil, GetBalanceArgs{})
	if err != nil {
		t.Fatalf("handler unexpected error: %v", err)
	}
	if out.Data.AccountValue != "1000" || out.Data.Withdrawable != "900" {
		t.Fatalf("getBalance output = %+v", out)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/tools/... -run 'TestGetAccountInfo|TestGetPositions|TestGetBalance' -v`
Expected: FAIL — `package tools: no Go files in .../internal/tools` (package doesn't exist yet).

- [ ] **Step 3: Write the implementation**

```go
// internal/tools/account.go
package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	hyperliquid "github.com/sonirico/go-hyperliquid"
)

// accountClient is the subset of hlclient.Client this file depends on,
// declared locally so handlers are unit-testable with a fake.
type accountClient interface {
	UserState(ctx context.Context, address, dex string) (*hyperliquid.UserState, error)
	AccountAddress() string
}

type GetAccountInfoArgs struct {
	UserAddress string `json:"userAddress,omitempty" jsonschema:"User address (optional, defaults to configured account)"`
	Dex         string `json:"dex,omitempty" jsonschema:"Perp dex name (optional, defaults to empty string)"`
}

type GetAccountInfoResult struct {
	Message string                   `json:"message"`
	Data    *hyperliquid.UserState   `json:"data"`
}

type GetPositionsArgs struct {
	UserAddress string `json:"userAddress,omitempty" jsonschema:"User address (optional, defaults to configured account)"`
	Dex         string `json:"dex,omitempty" jsonschema:"Perp dex name (optional, defaults to empty string)"`
}

type GetPositionsResult struct {
	Message string                          `json:"message"`
	Data    []hyperliquid.AssetPosition     `json:"data"`
	Summary hyperliquid.MarginSummary       `json:"marginSummary"`
}

type GetBalanceArgs struct {
	UserAddress string `json:"userAddress,omitempty" jsonschema:"User address (optional, defaults to configured account)"`
	Dex         string `json:"dex,omitempty" jsonschema:"Perp dex name (optional, defaults to empty string)"`
}

type GetBalanceData struct {
	AccountValue    string `json:"accountValue"`
	TotalMarginUsed string `json:"totalMarginUsed"`
	Withdrawable    string `json:"withdrawable"`
}

type GetBalanceResult struct {
	Message string         `json:"message"`
	Data    GetBalanceData `json:"data"`
}

func resolveAddress(client accountClient, userAddress string) string {
	if userAddress != "" {
		return userAddress
	}
	return client.AccountAddress()
}

func getAccountInfoHandler(client accountClient) mcp.ToolHandlerFor[GetAccountInfoArgs, GetAccountInfoResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args GetAccountInfoArgs) (*mcp.CallToolResult, GetAccountInfoResult, error) {
		state, err := client.UserState(ctx, resolveAddress(client, args.UserAddress), args.Dex)
		if err != nil {
			return nil, GetAccountInfoResult{}, fmt.Errorf("hyperliquid_get_account_info: %w", err)
		}
		return nil, GetAccountInfoResult{Message: "Account information retrieved successfully", Data: state}, nil
	}
}

func getPositionsHandler(client accountClient) mcp.ToolHandlerFor[GetPositionsArgs, GetPositionsResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args GetPositionsArgs) (*mcp.CallToolResult, GetPositionsResult, error) {
		state, err := client.UserState(ctx, resolveAddress(client, args.UserAddress), args.Dex)
		if err != nil {
			return nil, GetPositionsResult{}, fmt.Errorf("hyperliquid_get_positions: %w", err)
		}
		return nil, GetPositionsResult{
			Message: "Positions retrieved successfully",
			Data:    state.AssetPositions,
			Summary: state.MarginSummary,
		}, nil
	}
}

func getBalanceHandler(client accountClient) mcp.ToolHandlerFor[GetBalanceArgs, GetBalanceResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args GetBalanceArgs) (*mcp.CallToolResult, GetBalanceResult, error) {
		state, err := client.UserState(ctx, resolveAddress(client, args.UserAddress), args.Dex)
		if err != nil {
			return nil, GetBalanceResult{}, fmt.Errorf("hyperliquid_get_balance: %w", err)
		}
		return nil, GetBalanceResult{
			Message: "Balance retrieved successfully",
			Data: GetBalanceData{
				AccountValue:    state.MarginSummary.AccountValue,
				TotalMarginUsed: state.MarginSummary.TotalMarginUsed,
				Withdrawable:    state.Withdrawable,
			},
		}, nil
	}
}

// RegisterAccountTools registers the account & position MCP tools on server.
func RegisterAccountTools(server *mcp.Server, client accountClient) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "hyperliquid_get_account_info",
		Description: "Get user's perpetual account summary including positions and margin",
	}, getAccountInfoHandler(client))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "hyperliquid_get_positions",
		Description: "Get user's open positions with margin summary",
	}, getPositionsHandler(client))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "hyperliquid_get_balance",
		Description: "Get user's account balance and withdrawable amount",
	}, getBalanceHandler(client))
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/tools/... -run 'TestGetAccountInfo|TestGetPositions|TestGetBalance' -v`
Expected: PASS for all five tests.

- [ ] **Step 5: Commit**

```bash
git add internal/tools/account.go internal/tools/account_test.go
git commit -m "feat: add account info, positions, and balance MCP tools"
```

---

## Task 5: `internal/tools/market.go` — market data tools

**Files:**
- Create: `internal/tools/market.go`
- Test: `internal/tools/market_test.go`

**Interfaces:**
- Consumes: `hlclient.Client` (Task 3) via a local `marketClient` interface.
- Produces: `tools.RegisterMarketTools(server *mcp.Server, client marketClient)`.

- [ ] **Step 1: Write the failing tests**

```go
// internal/tools/market_test.go
package tools

import (
	"context"
	"testing"

	hyperliquid "github.com/sonirico/go-hyperliquid"
)

type fakeMarketClient struct {
	meta    *hyperliquid.Meta
	mids    map[string]string
	book    *hyperliquid.L2Book
	trades  []hyperliquid.Trade
	funding []hyperliquid.FundingHistory
	candles []hyperliquid.Candle
}

func (f *fakeMarketClient) Meta(ctx context.Context) (*hyperliquid.Meta, error)      { return f.meta, nil }
func (f *fakeMarketClient) AllMids(ctx context.Context) (map[string]string, error)   { return f.mids, nil }
func (f *fakeMarketClient) L2Book(ctx context.Context, coin string) (*hyperliquid.L2Book, error) {
	return f.book, nil
}
func (f *fakeMarketClient) RecentTrades(ctx context.Context, coin string) ([]hyperliquid.Trade, error) {
	return f.trades, nil
}
func (f *fakeMarketClient) FundingHistory(ctx context.Context, coin string, startTime int64, endTime *int64) ([]hyperliquid.FundingHistory, error) {
	return f.funding, nil
}
func (f *fakeMarketClient) Candles(ctx context.Context, coin, interval string, startTime, endTime int64) ([]hyperliquid.Candle, error) {
	return f.candles, nil
}

func TestGetMeta(t *testing.T) {
	client := &fakeMarketClient{meta: &hyperliquid.Meta{Universe: []hyperliquid.AssetInfo{{Name: "BTC"}}}}

	_, out, err := getMetaHandler(client)(context.Background(), nil, GetMetaArgs{})
	if err != nil {
		t.Fatalf("handler unexpected error: %v", err)
	}
	if len(out.Data.Universe) != 1 || out.Data.Universe[0].Name != "BTC" {
		t.Fatalf("getMeta output = %+v", out)
	}
}

func TestGetAllMids(t *testing.T) {
	client := &fakeMarketClient{mids: map[string]string{"BTC": "65000"}}

	_, out, err := getAllMidsHandler(client)(context.Background(), nil, GetAllMidsArgs{})
	if err != nil {
		t.Fatalf("handler unexpected error: %v", err)
	}
	if out.Data["BTC"] != "65000" {
		t.Fatalf("getAllMids output = %+v", out)
	}
}

func TestGetOrderBook(t *testing.T) {
	client := &fakeMarketClient{book: &hyperliquid.L2Book{Coin: "BTC"}}

	_, out, err := getOrderBookHandler(client)(context.Background(), nil, GetOrderBookArgs{Coin: "BTC"})
	if err != nil {
		t.Fatalf("handler unexpected error: %v", err)
	}
	if out.Data.Coin != "BTC" {
		t.Fatalf("getOrderBook output = %+v", out)
	}
}

func TestGetRecentTrades(t *testing.T) {
	client := &fakeMarketClient{trades: []hyperliquid.Trade{{Coin: "BTC", Hash: "0xabc"}}}

	_, out, err := getRecentTradesHandler(client)(context.Background(), nil, GetRecentTradesArgs{Coin: "BTC"})
	if err != nil {
		t.Fatalf("handler unexpected error: %v", err)
	}
	if len(out.Data) != 1 || out.Data[0].Hash != "0xabc" {
		t.Fatalf("getRecentTrades output = %+v", out)
	}
}

func TestGetHistoricalFunding(t *testing.T) {
	client := &fakeMarketClient{funding: []hyperliquid.FundingHistory{{Coin: "BTC", FundingRate: "0.0001"}}}

	_, out, err := getHistoricalFundingHandler(client)(context.Background(), nil, GetHistoricalFundingArgs{Coin: "BTC", StartTime: 1700000000000})
	if err != nil {
		t.Fatalf("handler unexpected error: %v", err)
	}
	if len(out.Data) != 1 || out.Data[0].FundingRate != "0.0001" {
		t.Fatalf("getHistoricalFunding output = %+v", out)
	}
}

func TestGetCandles(t *testing.T) {
	client := &fakeMarketClient{candles: []hyperliquid.Candle{{Symbol: "BTC", Close: "65000"}}}

	_, out, err := getCandlesHandler(client)(context.Background(), nil, GetCandlesArgs{
		Coin: "BTC", Interval: "1h", StartTime: 1700000000000,
	})
	if err != nil {
		t.Fatalf("handler unexpected error: %v", err)
	}
	if len(out.Data) != 1 || out.Data[0].Close != "65000" {
		t.Fatalf("getCandles output = %+v", out)
	}
}

func TestGetCandles_DefaultsEndTimeToNow(t *testing.T) {
	client := &fakeMarketClient{}

	_, _, err := getCandlesHandler(client)(context.Background(), nil, GetCandlesArgs{
		Coin: "BTC", Interval: "1h", StartTime: 1700000000000, EndTime: 0,
	})
	if err != nil {
		t.Fatalf("handler unexpected error: %v", err)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/tools/... -run 'TestGetMeta|TestGetAllMids|TestGetOrderBook|TestGetRecentTrades|TestGetHistoricalFunding|TestGetCandles' -v`
Expected: FAIL — market tool symbols are undefined.

- [ ] **Step 3: Write the implementation**

```go
// internal/tools/market.go
package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	hyperliquid "github.com/sonirico/go-hyperliquid"
)

type marketClient interface {
	Meta(ctx context.Context) (*hyperliquid.Meta, error)
	AllMids(ctx context.Context) (map[string]string, error)
	L2Book(ctx context.Context, coin string) (*hyperliquid.L2Book, error)
	RecentTrades(ctx context.Context, coin string) ([]hyperliquid.Trade, error)
	FundingHistory(ctx context.Context, coin string, startTime int64, endTime *int64) ([]hyperliquid.FundingHistory, error)
	Candles(ctx context.Context, coin, interval string, startTime, endTime int64) ([]hyperliquid.Candle, error)
}

type GetMetaArgs struct{}

type GetMetaResult struct {
	Message string            `json:"message"`
	Data    *hyperliquid.Meta `json:"data"`
}

type GetAllMidsArgs struct{}

type GetAllMidsResult struct {
	Message string            `json:"message"`
	Data    map[string]string `json:"data"`
}

type GetOrderBookArgs struct {
	Coin string `json:"coin" jsonschema:"Asset symbol (e.g. 'BTC', 'ETH', 'SOL')"`
}

type GetOrderBookResult struct {
	Message string               `json:"message"`
	Data    *hyperliquid.L2Book  `json:"data"`
}

type GetRecentTradesArgs struct {
	Coin string `json:"coin" jsonschema:"Asset symbol (e.g. 'BTC', 'ETH', 'SOL')"`
}

type GetRecentTradesResult struct {
	Message string                `json:"message"`
	Data    []hyperliquid.Trade   `json:"data"`
}

type GetHistoricalFundingArgs struct {
	Coin      string `json:"coin" jsonschema:"Asset symbol (e.g. 'BTC', 'ETH', 'SOL')"`
	StartTime int64  `json:"startTime" jsonschema:"Start time in milliseconds"`
	EndTime   int64  `json:"endTime,omitempty" jsonschema:"End time in milliseconds (optional, defaults to current time)"`
}

type GetHistoricalFundingResult struct {
	Message string                          `json:"message"`
	Data    []hyperliquid.FundingHistory     `json:"data"`
}

type GetCandlesArgs struct {
	Coin      string `json:"coin" jsonschema:"Asset symbol (e.g. 'BTC', 'ETH', 'SOL')"`
	Interval  string `json:"interval" jsonschema:"Candle interval: 1m, 5m, 15m, 1h, 4h, or 1d"`
	StartTime int64  `json:"startTime" jsonschema:"Start time in milliseconds"`
	EndTime   int64  `json:"endTime,omitempty" jsonschema:"End time in milliseconds (optional, defaults to current time)"`
}

type GetCandlesResult struct {
	Message string                `json:"message"`
	Data    []hyperliquid.Candle  `json:"data"`
}

func endTimeOrNow(endTime int64) int64 {
	if endTime == 0 {
		return time.Now().UnixMilli()
	}
	return endTime
}

func optionalEndTime(endTime int64) *int64 {
	if endTime == 0 {
		return nil
	}
	return &endTime
}

func getMetaHandler(client marketClient) mcp.ToolHandlerFor[GetMetaArgs, GetMetaResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, _ GetMetaArgs) (*mcp.CallToolResult, GetMetaResult, error) {
		meta, err := client.Meta(ctx)
		if err != nil {
			return nil, GetMetaResult{}, fmt.Errorf("hyperliquid_get_meta: %w", err)
		}
		return nil, GetMetaResult{Message: "Exchange metadata retrieved successfully", Data: meta}, nil
	}
}

func getAllMidsHandler(client marketClient) mcp.ToolHandlerFor[GetAllMidsArgs, GetAllMidsResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, _ GetAllMidsArgs) (*mcp.CallToolResult, GetAllMidsResult, error) {
		mids, err := client.AllMids(ctx)
		if err != nil {
			return nil, GetAllMidsResult{}, fmt.Errorf("hyperliquid_get_all_mids: %w", err)
		}
		return nil, GetAllMidsResult{Message: "Mid prices retrieved successfully", Data: mids}, nil
	}
}

func getOrderBookHandler(client marketClient) mcp.ToolHandlerFor[GetOrderBookArgs, GetOrderBookResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args GetOrderBookArgs) (*mcp.CallToolResult, GetOrderBookResult, error) {
		book, err := client.L2Book(ctx, args.Coin)
		if err != nil {
			return nil, GetOrderBookResult{}, fmt.Errorf("hyperliquid_get_order_book: %w", err)
		}
		return nil, GetOrderBookResult{Message: "Order book retrieved successfully", Data: book}, nil
	}
}

func getRecentTradesHandler(client marketClient) mcp.ToolHandlerFor[GetRecentTradesArgs, GetRecentTradesResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args GetRecentTradesArgs) (*mcp.CallToolResult, GetRecentTradesResult, error) {
		trades, err := client.RecentTrades(ctx, args.Coin)
		if err != nil {
			return nil, GetRecentTradesResult{}, fmt.Errorf("hyperliquid_get_recent_trades: %w", err)
		}
		return nil, GetRecentTradesResult{Message: "Recent trades retrieved successfully", Data: trades}, nil
	}
}

func getHistoricalFundingHandler(client marketClient) mcp.ToolHandlerFor[GetHistoricalFundingArgs, GetHistoricalFundingResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args GetHistoricalFundingArgs) (*mcp.CallToolResult, GetHistoricalFundingResult, error) {
		history, err := client.FundingHistory(ctx, args.Coin, args.StartTime, optionalEndTime(args.EndTime))
		if err != nil {
			return nil, GetHistoricalFundingResult{}, fmt.Errorf("hyperliquid_get_historical_funding: %w", err)
		}
		return nil, GetHistoricalFundingResult{Message: "Historical funding retrieved successfully", Data: history}, nil
	}
}

func getCandlesHandler(client marketClient) mcp.ToolHandlerFor[GetCandlesArgs, GetCandlesResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args GetCandlesArgs) (*mcp.CallToolResult, GetCandlesResult, error) {
		candles, err := client.Candles(ctx, args.Coin, args.Interval, args.StartTime, endTimeOrNow(args.EndTime))
		if err != nil {
			return nil, GetCandlesResult{}, fmt.Errorf("hyperliquid_get_candles: %w", err)
		}
		return nil, GetCandlesResult{Message: "Candles retrieved successfully", Data: candles}, nil
	}
}

// RegisterMarketTools registers the market data MCP tools on server.
func RegisterMarketTools(server *mcp.Server, client marketClient) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "hyperliquid_get_meta",
		Description: "Get exchange metadata (assets, leverage, etc.)",
	}, getMetaHandler(client))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "hyperliquid_get_all_mids",
		Description: "Get current mid prices for all assets",
	}, getAllMidsHandler(client))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "hyperliquid_get_order_book",
		Description: "Get order book depth for an asset",
	}, getOrderBookHandler(client))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "hyperliquid_get_recent_trades",
		Description: "Get recent trades for an asset",
	}, getRecentTradesHandler(client))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "hyperliquid_get_historical_funding",
		Description: "Get funding rate history for an asset",
	}, getHistoricalFundingHandler(client))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "hyperliquid_get_candles",
		Description: "Get historical candle/OHLCV data for an asset",
	}, getCandlesHandler(client))
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/tools/... -run 'TestGetMeta|TestGetAllMids|TestGetOrderBook|TestGetRecentTrades|TestGetHistoricalFunding|TestGetCandles' -v`
Expected: PASS for all seven tests.

- [ ] **Step 5: Commit**

```bash
git add internal/tools/market.go internal/tools/market_test.go
git commit -m "feat: add market data MCP tools"
```

---

## Task 6: `internal/tools/queries.go` — order/fill/funding query tools

**Files:**
- Create: `internal/tools/queries.go`
- Test: `internal/tools/queries_test.go`

**Interfaces:**
- Consumes: `hlclient.Client` (Task 3) via a local `queriesClient` interface.
- Produces: `tools.RegisterQueryTools(server *mcp.Server, client queriesClient)`.

- [ ] **Step 1: Write the failing tests**

```go
// internal/tools/queries_test.go
package tools

import (
	"context"
	"testing"

	hyperliquid "github.com/sonirico/go-hyperliquid"
)

type fakeQueriesClient struct {
	accountAddress string
	openOrders     []hyperliquid.OpenOrder
	orderStatus    *hyperliquid.OrderQueryResult
	fills          []hyperliquid.Fill
	funding        []hyperliquid.UserFundingHistory
	gotAddress     string
	gotDex         string
}

func (f *fakeQueriesClient) AccountAddress() string { return f.accountAddress }

func (f *fakeQueriesClient) OpenOrders(ctx context.Context, address, dex string) ([]hyperliquid.OpenOrder, error) {
	f.gotAddress, f.gotDex = address, dex
	return f.openOrders, nil
}

func (f *fakeQueriesClient) OrderStatus(ctx context.Context, user string, oid int64) (*hyperliquid.OrderQueryResult, error) {
	return f.orderStatus, nil
}

func (f *fakeQueriesClient) UserFills(ctx context.Context, address string) ([]hyperliquid.Fill, error) {
	f.gotAddress = address
	return f.fills, nil
}

func (f *fakeQueriesClient) UserFundingHistory(ctx context.Context, user string, startTime int64, endTime *int64) ([]hyperliquid.UserFundingHistory, error) {
	return f.funding, nil
}

func TestGetOpenOrders(t *testing.T) {
	client := &fakeQueriesClient{
		accountAddress: "0xConfigured",
		openOrders:     []hyperliquid.OpenOrder{{Coin: "BTC", Oid: 1}},
	}

	_, out, err := getOpenOrdersHandler(client)(context.Background(), nil, GetOpenOrdersArgs{})
	if err != nil {
		t.Fatalf("handler unexpected error: %v", err)
	}
	if client.gotAddress != "0xConfigured" {
		t.Fatalf("expected default address, got %q", client.gotAddress)
	}
	if len(out.Data) != 1 || out.Data[0].Coin != "BTC" {
		t.Fatalf("getOpenOrders output = %+v", out)
	}
}

func TestGetOrderStatus(t *testing.T) {
	client := &fakeQueriesClient{
		accountAddress: "0xConfigured",
		orderStatus:    &hyperliquid.OrderQueryResult{Status: hyperliquid.OrderQueryStatusSuccess},
	}

	_, out, err := getOrderStatusHandler(client)(context.Background(), nil, GetOrderStatusArgs{Oid: 1})
	if err != nil {
		t.Fatalf("handler unexpected error: %v", err)
	}
	if out.Data.Status != hyperliquid.OrderQueryStatusSuccess {
		t.Fatalf("getOrderStatus output = %+v", out)
	}
}

func TestGetUserFills(t *testing.T) {
	client := &fakeQueriesClient{
		accountAddress: "0xConfigured",
		fills:          []hyperliquid.Fill{{Coin: "BTC", Oid: 1}},
	}

	_, out, err := getUserFillsHandler(client)(context.Background(), nil, GetUserFillsArgs{})
	if err != nil {
		t.Fatalf("handler unexpected error: %v", err)
	}
	if client.gotAddress != "0xConfigured" {
		t.Fatalf("expected default address, got %q", client.gotAddress)
	}
	if len(out.Data) != 1 || out.Data[0].Coin != "BTC" {
		t.Fatalf("getUserFills output = %+v", out)
	}
}

func TestGetUserFunding(t *testing.T) {
	client := &fakeQueriesClient{
		accountAddress: "0xConfigured",
		funding:        []hyperliquid.UserFundingHistory{{Hash: "0xabc"}},
	}

	_, out, err := getUserFundingHandler(client)(context.Background(), nil, GetUserFundingArgs{StartTime: 1700000000000})
	if err != nil {
		t.Fatalf("handler unexpected error: %v", err)
	}
	if len(out.Data) != 1 || out.Data[0].Hash != "0xabc" {
		t.Fatalf("getUserFunding output = %+v", out)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/tools/... -run 'TestGetOpenOrders|TestGetOrderStatus|TestGetUserFills|TestGetUserFunding' -v`
Expected: FAIL — query tool symbols are undefined.

- [ ] **Step 3: Write the implementation**

```go
// internal/tools/queries.go
package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	hyperliquid "github.com/sonirico/go-hyperliquid"
)

type queriesClient interface {
	AccountAddress() string
	OpenOrders(ctx context.Context, address, dex string) ([]hyperliquid.OpenOrder, error)
	OrderStatus(ctx context.Context, user string, oid int64) (*hyperliquid.OrderQueryResult, error)
	UserFills(ctx context.Context, address string) ([]hyperliquid.Fill, error)
	UserFundingHistory(ctx context.Context, user string, startTime int64, endTime *int64) ([]hyperliquid.UserFundingHistory, error)
}

type GetOpenOrdersArgs struct {
	UserAddress string `json:"userAddress,omitempty" jsonschema:"User address (optional, defaults to configured account)"`
	Dex         string `json:"dex,omitempty" jsonschema:"Perp dex name (optional)"`
}

type GetOpenOrdersResult struct {
	Message string                     `json:"message"`
	Data    []hyperliquid.OpenOrder    `json:"data"`
}

type GetOrderStatusArgs struct {
	UserAddress string `json:"userAddress,omitempty" jsonschema:"User address (optional, defaults to configured account)"`
	Oid         int64  `json:"oid" jsonschema:"Order ID (oid) to look up"`
}

type GetOrderStatusResult struct {
	Message string                          `json:"message"`
	Data    *hyperliquid.OrderQueryResult   `json:"data"`
}

type GetUserFillsArgs struct {
	UserAddress string `json:"userAddress,omitempty" jsonschema:"User address (optional, defaults to configured account)"`
}

type GetUserFillsResult struct {
	Message string               `json:"message"`
	Data    []hyperliquid.Fill   `json:"data"`
}

type GetUserFundingArgs struct {
	UserAddress string `json:"userAddress,omitempty" jsonschema:"User address (optional, defaults to configured account)"`
	StartTime   int64  `json:"startTime" jsonschema:"Start time in milliseconds"`
	EndTime     int64  `json:"endTime,omitempty" jsonschema:"End time in milliseconds (optional, defaults to current time)"`
}

type GetUserFundingResult struct {
	Message string                              `json:"message"`
	Data    []hyperliquid.UserFundingHistory     `json:"data"`
}

func getOpenOrdersHandler(client queriesClient) mcp.ToolHandlerFor[GetOpenOrdersArgs, GetOpenOrdersResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args GetOpenOrdersArgs) (*mcp.CallToolResult, GetOpenOrdersResult, error) {
		address := args.UserAddress
		if address == "" {
			address = client.AccountAddress()
		}
		orders, err := client.OpenOrders(ctx, address, args.Dex)
		if err != nil {
			return nil, GetOpenOrdersResult{}, fmt.Errorf("hyperliquid_get_open_orders: %w", err)
		}
		return nil, GetOpenOrdersResult{Message: "Open orders retrieved successfully", Data: orders}, nil
	}
}

func getOrderStatusHandler(client queriesClient) mcp.ToolHandlerFor[GetOrderStatusArgs, GetOrderStatusResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args GetOrderStatusArgs) (*mcp.CallToolResult, GetOrderStatusResult, error) {
		address := args.UserAddress
		if address == "" {
			address = client.AccountAddress()
		}
		status, err := client.OrderStatus(ctx, address, args.Oid)
		if err != nil {
			return nil, GetOrderStatusResult{}, fmt.Errorf("hyperliquid_get_order_status: %w", err)
		}
		return nil, GetOrderStatusResult{Message: "Order status retrieved successfully", Data: status}, nil
	}
}

func getUserFillsHandler(client queriesClient) mcp.ToolHandlerFor[GetUserFillsArgs, GetUserFillsResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args GetUserFillsArgs) (*mcp.CallToolResult, GetUserFillsResult, error) {
		address := args.UserAddress
		if address == "" {
			address = client.AccountAddress()
		}
		fills, err := client.UserFills(ctx, address)
		if err != nil {
			return nil, GetUserFillsResult{}, fmt.Errorf("hyperliquid_get_user_fills: %w", err)
		}
		return nil, GetUserFillsResult{Message: "User fills retrieved successfully", Data: fills}, nil
	}
}

func getUserFundingHandler(client queriesClient) mcp.ToolHandlerFor[GetUserFundingArgs, GetUserFundingResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args GetUserFundingArgs) (*mcp.CallToolResult, GetUserFundingResult, error) {
		address := args.UserAddress
		if address == "" {
			address = client.AccountAddress()
		}
		history, err := client.UserFundingHistory(ctx, address, args.StartTime, optionalEndTime(args.EndTime))
		if err != nil {
			return nil, GetUserFundingResult{}, fmt.Errorf("hyperliquid_get_user_funding: %w", err)
		}
		return nil, GetUserFundingResult{Message: "User funding history retrieved successfully", Data: history}, nil
	}
}

// RegisterQueryTools registers the order/fill/funding query MCP tools on server.
func RegisterQueryTools(server *mcp.Server, client queriesClient) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "hyperliquid_get_open_orders",
		Description: "Get all open orders for the user",
	}, getOpenOrdersHandler(client))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "hyperliquid_get_order_status",
		Description: "Get status of a specific order by oid",
	}, getOrderStatusHandler(client))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "hyperliquid_get_user_fills",
		Description: "Get trade fill history for the user",
	}, getUserFillsHandler(client))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "hyperliquid_get_user_funding",
		Description: "Get funding payment history for the user",
	}, getUserFundingHandler(client))
}
```

Note: `optionalEndTime` is defined in `internal/tools/market.go` (Task 5) and reused here — both files are in package `tools`.

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/tools/... -run 'TestGetOpenOrders|TestGetOrderStatus|TestGetUserFills|TestGetUserFunding' -v`
Expected: PASS for all four tests.

- [ ] **Step 5: Commit**

```bash
git add internal/tools/queries.go internal/tools/queries_test.go
git commit -m "feat: add open orders, order status, fills, and funding query tools"
```

---

## Task 7: `internal/tools/orders.go` — order management tools

**Files:**
- Create: `internal/tools/orders.go`
- Test: `internal/tools/orders_test.go`

**Interfaces:**
- Consumes: `hlclient.Client` (Task 3) via a local `ordersClient` interface; `hlclient.BracketOrderParams`.
- Produces: `tools.RegisterOrderTools(server *mcp.Server, client ordersClient)`.

- [ ] **Step 1: Write the failing tests**

```go
// internal/tools/orders_test.go
package tools

import (
	"context"
	"errors"
	"testing"

	hyperliquid "github.com/sonirico/go-hyperliquid"

	"github.com/duongnv129/hyperliquid-mcp/internal/hlclient"
)

type fakeOrdersClient struct {
	accountAddress string
	coinForAsset   map[int]string
	assetIndex     map[string]int

	placeOrderReq    hyperliquid.CreateOrderRequest
	placeOrderResult hyperliquid.OrderStatus
	placeOrderErr    error

	bracketParams hlclient.BracketOrderParams
	bracketResp   *hyperliquid.APIResponse[hyperliquid.OrderResponse]
	bracketErr    error

	cancelCoin string
	cancelOid  int64
	cancelResp *hyperliquid.APIResponse[hyperliquid.CancelOrderResponse]
	cancelErr  error

	cancelAllUser      string
	cancelAllDex       string
	cancelAllCancelled int
	cancelAllResp      *hyperliquid.APIResponse[hyperliquid.CancelOrderResponse]
	cancelAllErr       error

	modifyReq    hyperliquid.ModifyOrderRequest
	modifyResult hyperliquid.OrderStatus
	modifyErr    error
}

func (f *fakeOrdersClient) AccountAddress() string { return f.accountAddress }

func (f *fakeOrdersClient) AssetIndex(coin string) (int, bool) {
	idx, ok := f.assetIndex[coin]
	return idx, ok
}

func (f *fakeOrdersClient) CoinForAsset(ctx context.Context, asset int) (string, error) {
	coin, ok := f.coinForAsset[asset]
	if !ok {
		return "", errors.New("unknown asset")
	}
	return coin, nil
}

func (f *fakeOrdersClient) PlaceOrder(ctx context.Context, req hyperliquid.CreateOrderRequest) (hyperliquid.OrderStatus, error) {
	f.placeOrderReq = req
	return f.placeOrderResult, f.placeOrderErr
}

func (f *fakeOrdersClient) PlaceBracketOrder(ctx context.Context, p hlclient.BracketOrderParams) (*hyperliquid.APIResponse[hyperliquid.OrderResponse], error) {
	f.bracketParams = p
	return f.bracketResp, f.bracketErr
}

func (f *fakeOrdersClient) CancelOrder(ctx context.Context, coin string, oid int64) (*hyperliquid.APIResponse[hyperliquid.CancelOrderResponse], error) {
	f.cancelCoin, f.cancelOid = coin, oid
	return f.cancelResp, f.cancelErr
}

func (f *fakeOrdersClient) CancelAllOrders(ctx context.Context, user, dex string) (int, *hyperliquid.APIResponse[hyperliquid.CancelOrderResponse], error) {
	f.cancelAllUser, f.cancelAllDex = user, dex
	return f.cancelAllCancelled, f.cancelAllResp, f.cancelAllErr
}

func (f *fakeOrdersClient) ModifyOrder(ctx context.Context, req hyperliquid.ModifyOrderRequest) (hyperliquid.OrderStatus, error) {
	f.modifyReq = req
	return f.modifyResult, f.modifyErr
}

func TestPlaceOrderHandler_LimitOrder(t *testing.T) {
	oid := int64(1)
	client := &fakeOrdersClient{
		coinForAsset:     map[int]string{0: "BTC"},
		placeOrderResult: hyperliquid.OrderStatus{Resting: &hyperliquid.OrderStatusResting{Oid: oid, Status: "resting"}},
	}

	_, out, err := placeOrderHandler(client)(context.Background(), nil, PlaceOrderArgs{
		Asset: 0, IsBuy: true, Size: "0.1", Price: "65000",
	})
	if err != nil {
		t.Fatalf("handler unexpected error: %v", err)
	}
	if client.placeOrderReq.Coin != "BTC" || client.placeOrderReq.OrderType.Limit == nil {
		t.Fatalf("expected a resolved BTC limit order, got %+v", client.placeOrderReq)
	}
	if out.Data.Resting == nil || out.Data.Resting.Oid != 1 {
		t.Fatalf("placeOrder output = %+v", out)
	}
}

func TestPlaceOrderHandler_TriggerOrder(t *testing.T) {
	client := &fakeOrdersClient{coinForAsset: map[int]string{5: "SOL"}}

	_, _, err := placeOrderHandler(client)(context.Background(), nil, PlaceOrderArgs{
		Asset: 5, IsBuy: false, Size: "1", Price: "220", TriggerPx: "219.5", Tpsl: "tp",
	})
	if err != nil {
		t.Fatalf("handler unexpected error: %v", err)
	}
	if client.placeOrderReq.OrderType.Trigger == nil || client.placeOrderReq.OrderType.Trigger.Tpsl != hyperliquid.TakeProfit {
		t.Fatalf("expected a take-profit trigger order, got %+v", client.placeOrderReq.OrderType)
	}
}

func TestPlaceOrderHandler_UnknownAsset(t *testing.T) {
	client := &fakeOrdersClient{coinForAsset: map[int]string{}}

	_, _, err := placeOrderHandler(client)(context.Background(), nil, PlaceOrderArgs{Asset: 99, IsBuy: true, Size: "0.1"})
	if err == nil {
		t.Fatal("expected an error for an unresolvable asset index")
	}
}

func TestPlaceOrderHandler_InvalidSize(t *testing.T) {
	client := &fakeOrdersClient{coinForAsset: map[int]string{0: "BTC"}}

	_, _, err := placeOrderHandler(client)(context.Background(), nil, PlaceOrderArgs{Asset: 0, IsBuy: true, Size: "not-a-number"})
	if err == nil {
		t.Fatal("expected an error for a non-numeric size")
	}
}

func TestPlaceBracketOrderHandler(t *testing.T) {
	client := &fakeOrdersClient{
		coinForAsset: map[int]string{5: "SOL"},
		bracketResp: &hyperliquid.APIResponse[hyperliquid.OrderResponse]{
			Ok: true,
			Data: hyperliquid.OrderResponse{Statuses: []hyperliquid.OrderStatus{
				{Resting: &hyperliquid.OrderStatusResting{Oid: 1}},
				{Resting: &hyperliquid.OrderStatusResting{Oid: 2}},
				{Resting: &hyperliquid.OrderStatusResting{Oid: 3}},
			}},
		},
	}

	_, out, err := placeBracketOrderHandler(client)(context.Background(), nil, PlaceBracketOrderArgs{
		Asset: 5, IsBuy: true, Size: "4.12", EntryPrice: "218.0", TakeProfitPrice: "219.5", StopLossPrice: "216.8",
	})
	if err != nil {
		t.Fatalf("handler unexpected error: %v", err)
	}
	if client.bracketParams.Coin != "SOL" || client.bracketParams.TakeProfitPrice != 219.5 {
		t.Fatalf("bracketParams = %+v", client.bracketParams)
	}
	if len(out.Data.Statuses) != 3 {
		t.Fatalf("placeBracketOrder output = %+v", out)
	}
}

func TestCancelOrderHandler(t *testing.T) {
	client := &fakeOrdersClient{
		cancelResp: &hyperliquid.APIResponse[hyperliquid.CancelOrderResponse]{Ok: true},
	}

	_, out, err := cancelOrderHandler(client)(context.Background(), nil, CancelOrderArgs{Coin: "BTC", Oid: 1})
	if err != nil {
		t.Fatalf("handler unexpected error: %v", err)
	}
	if client.cancelCoin != "BTC" || client.cancelOid != 1 {
		t.Fatalf("cancel args = (%q, %d)", client.cancelCoin, client.cancelOid)
	}
	if out.CancelledOrder.Coin != "BTC" || out.CancelledOrder.OrderID != 1 {
		t.Fatalf("cancelOrder output = %+v", out)
	}
}

func TestCancelAllOrdersHandler_NoOrders(t *testing.T) {
	client := &fakeOrdersClient{accountAddress: "0xConfigured", cancelAllCancelled: 0, cancelAllResp: nil}

	_, out, err := cancelAllOrdersHandler(client)(context.Background(), nil, CancelAllOrdersArgs{})
	if err != nil {
		t.Fatalf("handler unexpected error: %v", err)
	}
	if client.cancelAllUser != "0xConfigured" {
		t.Fatalf("expected default address, got %q", client.cancelAllUser)
	}
	if out.CancelledCount != 0 {
		t.Fatalf("cancelAllOrders output = %+v, want CancelledCount 0", out)
	}
}

func TestCancelAllOrdersHandler_WithOrders(t *testing.T) {
	client := &fakeOrdersClient{
		accountAddress:     "0xConfigured",
		cancelAllCancelled: 2,
		cancelAllResp:      &hyperliquid.APIResponse[hyperliquid.CancelOrderResponse]{Ok: true},
	}

	_, out, err := cancelAllOrdersHandler(client)(context.Background(), nil, CancelAllOrdersArgs{})
	if err != nil {
		t.Fatalf("handler unexpected error: %v", err)
	}
	if out.CancelledCount != 2 {
		t.Fatalf("cancelAllOrders output = %+v, want CancelledCount 2", out)
	}
}

func TestModifyOrderHandler(t *testing.T) {
	client := &fakeOrdersClient{
		modifyResult: hyperliquid.OrderStatus{Resting: &hyperliquid.OrderStatusResting{Oid: 1, Status: "resting"}},
	}

	_, out, err := modifyOrderHandler(client)(context.Background(), nil, ModifyOrderArgs{
		Oid: 1, Coin: "BTC", IsBuy: true, Size: "0.2", Price: "64000",
	})
	if err != nil {
		t.Fatalf("handler unexpected error: %v", err)
	}
	if client.modifyReq.Oid == nil || *client.modifyReq.Oid != 1 || client.modifyReq.Order.Coin != "BTC" {
		t.Fatalf("modifyReq = %+v", client.modifyReq)
	}
	if out.Data.Resting == nil || out.Data.Resting.Oid != 1 {
		t.Fatalf("modifyOrder output = %+v", out)
	}
}

func TestPlaceTwapOrderHandler_Stub(t *testing.T) {
	client := &fakeOrdersClient{}

	_, out, err := placeTwapOrderHandler(client)(context.Background(), nil, PlaceTwapOrderArgs{})
	if err != nil {
		t.Fatalf("stub handler should not error: %v", err)
	}
	if out.Message == "" {
		t.Fatal("expected a non-empty 'not implemented' message")
	}
}

func TestCancelTwapOrderHandler_Stub(t *testing.T) {
	client := &fakeOrdersClient{}

	_, out, err := cancelTwapOrderHandler(client)(context.Background(), nil, CancelTwapOrderArgs{})
	if err != nil {
		t.Fatalf("stub handler should not error: %v", err)
	}
	if out.Message == "" {
		t.Fatal("expected a non-empty 'not implemented' message")
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/tools/... -run 'TestPlaceOrderHandler|TestPlaceBracketOrderHandler|TestCancelOrderHandler|TestCancelAllOrdersHandler|TestModifyOrderHandler|TestPlaceTwapOrderHandler|TestCancelTwapOrderHandler' -v`
Expected: FAIL — order tool symbols are undefined.

- [ ] **Step 3: Write the implementation**

```go
// internal/tools/orders.go
package tools

import (
	"context"
	"fmt"
	"strconv"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	hyperliquid "github.com/sonirico/go-hyperliquid"

	"github.com/duongnv129/hyperliquid-mcp/internal/hlclient"
)

type ordersClient interface {
	AccountAddress() string
	AssetIndex(coin string) (int, bool)
	CoinForAsset(ctx context.Context, asset int) (string, error)
	PlaceOrder(ctx context.Context, req hyperliquid.CreateOrderRequest) (hyperliquid.OrderStatus, error)
	PlaceBracketOrder(ctx context.Context, p hlclient.BracketOrderParams) (*hyperliquid.APIResponse[hyperliquid.OrderResponse], error)
	CancelOrder(ctx context.Context, coin string, oid int64) (*hyperliquid.APIResponse[hyperliquid.CancelOrderResponse], error)
	CancelAllOrders(ctx context.Context, user, dex string) (int, *hyperliquid.APIResponse[hyperliquid.CancelOrderResponse], error)
	ModifyOrder(ctx context.Context, req hyperliquid.ModifyOrderRequest) (hyperliquid.OrderStatus, error)
}

type PlaceOrderArgs struct {
	Asset      int    `json:"asset" jsonschema:"Asset index (e.g. 0 for BTC, 1 for ETH, 5 for SOL). Use hyperliquid_get_meta for the full list."`
	IsBuy      bool   `json:"isBuy" jsonschema:"True for buy/long orders, false for sell/short orders"`
	Size       string `json:"size" jsonschema:"Order size/quantity as a string (e.g. '0.1' for 0.1 BTC)"`
	Price      string `json:"price,omitempty" jsonschema:"Limit price as a string. Set to '0' or omit for market orders."`
	ReduceOnly bool   `json:"reduceOnly,omitempty" jsonschema:"Whether this is a reduce-only order"`
	Tif        string `json:"tif,omitempty" jsonschema:"Time-in-force for limit orders: Gtc, Ioc, or Alo. Ignored if triggerPx is set. Defaults to Gtc."`
	TriggerPx  string `json:"triggerPx,omitempty" jsonschema:"Trigger price for stop/take-profit orders. When set, this becomes a trigger order instead of a limit order."`
	IsMarket   bool   `json:"isMarket,omitempty" jsonschema:"For trigger orders only: true for a market trigger, false for a limit trigger"`
	Tpsl       string `json:"tpsl,omitempty" jsonschema:"For trigger orders only: 'tp' for take-profit or 'sl' for stop-loss"`
	Cloid      string `json:"cloid,omitempty" jsonschema:"Client order ID (optional, for tracking)"`
}

type PlaceOrderResult struct {
	Message string                  `json:"message"`
	Data    hyperliquid.OrderStatus `json:"data"`
}

type PlaceBracketOrderArgs struct {
	Asset           int    `json:"asset" jsonschema:"Asset index (e.g. 0 for BTC, 1 for ETH, 5 for SOL)"`
	IsBuy           bool   `json:"isBuy" jsonschema:"True for long positions, false for short positions"`
	Size            string `json:"size" jsonschema:"Position size as a string (e.g. '4.12' for 4.12 SOL)"`
	EntryPrice      string `json:"entryPrice" jsonschema:"Entry limit price as a string"`
	TakeProfitPrice string `json:"takeProfitPrice" jsonschema:"Take profit trigger price"`
	StopLossPrice   string `json:"stopLossPrice" jsonschema:"Stop loss trigger price"`
}

type PlaceBracketOrderResult struct {
	Message string                           `json:"message"`
	Data    hyperliquid.OrderResponse        `json:"data"`
}

type CancelOrderArgs struct {
	Coin string `json:"coin" jsonschema:"Coin/asset name (e.g. 'BTC', 'ETH', 'SOL')"`
	Oid  int64  `json:"oid" jsonschema:"Order ID (oid) - the unique order identifier returned when the order was placed"`
}

type CancelledOrder struct {
	Coin    string `json:"coin"`
	OrderID int64  `json:"orderId"`
}

type CancelOrderResult struct {
	Message        string          `json:"message"`
	CancelledOrder CancelledOrder  `json:"cancelledOrder"`
}

type CancelAllOrdersArgs struct {
	UserAddress string `json:"userAddress,omitempty" jsonschema:"User address (optional, defaults to configured account)"`
	Dex         string `json:"dex,omitempty" jsonschema:"Perp dex name (optional)"`
}

type CancelAllOrdersResult struct {
	Message        string `json:"message"`
	CancelledCount int    `json:"cancelledCount"`
}

type ModifyOrderArgs struct {
	Oid        int64  `json:"oid" jsonschema:"Order ID to modify"`
	Coin       string `json:"coin" jsonschema:"Coin/asset name (e.g. 'BTC', 'ETH', 'SOL')"`
	IsBuy      bool   `json:"isBuy" jsonschema:"True for buy orders, false for sell orders"`
	Size       string `json:"size" jsonschema:"New order size"`
	Price      string `json:"price,omitempty" jsonschema:"New limit price"`
	ReduceOnly bool   `json:"reduceOnly,omitempty" jsonschema:"Whether this is a reduce-only order"`
	Tif        string `json:"tif,omitempty" jsonschema:"Time-in-force for limit orders: Gtc, Ioc, or Alo. Defaults to Gtc."`
	TriggerPx  string `json:"triggerPx,omitempty" jsonschema:"Trigger price, if converting this order to a trigger order"`
	IsMarket   bool   `json:"isMarket,omitempty" jsonschema:"For trigger orders only: true for a market trigger"`
	Tpsl       string `json:"tpsl,omitempty" jsonschema:"For trigger orders only: 'tp' or 'sl'"`
	Cloid      string `json:"cloid,omitempty" jsonschema:"Client order ID for the modified order (optional)"`
}

type ModifyOrderResult struct {
	Message string                  `json:"message"`
	Data    hyperliquid.OrderStatus `json:"data"`
}

type PlaceTwapOrderArgs struct {
	Asset int    `json:"asset,omitempty" jsonschema:"Asset index"`
	IsBuy bool   `json:"isBuy,omitempty" jsonschema:"True for buy, false for sell"`
	Size  string `json:"size,omitempty" jsonschema:"Total size to execute over the TWAP duration"`
}

type CancelTwapOrderArgs struct {
	TwapID string `json:"twapId,omitempty" jsonschema:"TWAP order identifier to cancel"`
}

type TwapStubResult struct {
	Message string `json:"message"`
}

func parsePrice(price string) (float64, error) {
	if price == "" {
		return 0, nil
	}
	value, err := strconv.ParseFloat(price, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid price %q: %w", price, err)
	}
	return value, nil
}

func parseSize(size string) (float64, error) {
	value, err := strconv.ParseFloat(size, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size %q: %w", size, err)
	}
	return value, nil
}

// buildOrderType builds a limit or trigger OrderType from the shared fields
// used by both place_order and modify_order: a triggerPx switches the order
// from limit to trigger.
func buildOrderType(tif, triggerPx string, isMarket bool, tpsl string) (hyperliquid.OrderType, error) {
	if triggerPx != "" {
		px, err := strconv.ParseFloat(triggerPx, 64)
		if err != nil {
			return hyperliquid.OrderType{}, fmt.Errorf("invalid triggerPx %q: %w", triggerPx, err)
		}
		tpslValue := hyperliquid.Tpsl(tpsl)
		if tpslValue != hyperliquid.TakeProfit && tpslValue != hyperliquid.StopLoss {
			return hyperliquid.OrderType{}, fmt.Errorf("tpsl must be %q or %q, got %q", hyperliquid.TakeProfit, hyperliquid.StopLoss, tpsl)
		}
		return hyperliquid.OrderType{Trigger: &hyperliquid.TriggerOrderType{
			TriggerPx: px,
			IsMarket:  isMarket,
			Tpsl:      tpslValue,
		}}, nil
	}

	tifValue := hyperliquid.Tif(tif)
	if tifValue == "" {
		tifValue = hyperliquid.TifGtc
	}
	return hyperliquid.OrderType{Limit: &hyperliquid.LimitOrderType{Tif: tifValue}}, nil
}

func cloidPointer(cloid string) *string {
	if cloid == "" {
		return nil
	}
	return &cloid
}

func placeOrderHandler(client ordersClient) mcp.ToolHandlerFor[PlaceOrderArgs, PlaceOrderResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args PlaceOrderArgs) (*mcp.CallToolResult, PlaceOrderResult, error) {
		coin, err := client.CoinForAsset(ctx, args.Asset)
		if err != nil {
			return nil, PlaceOrderResult{}, fmt.Errorf("hyperliquid_place_order: %w", err)
		}

		size, err := parseSize(args.Size)
		if err != nil {
			return nil, PlaceOrderResult{}, fmt.Errorf("hyperliquid_place_order: %w", err)
		}
		price, err := parsePrice(args.Price)
		if err != nil {
			return nil, PlaceOrderResult{}, fmt.Errorf("hyperliquid_place_order: %w", err)
		}
		orderType, err := buildOrderType(args.Tif, args.TriggerPx, args.IsMarket, args.Tpsl)
		if err != nil {
			return nil, PlaceOrderResult{}, fmt.Errorf("hyperliquid_place_order: %w", err)
		}

		status, err := client.PlaceOrder(ctx, hyperliquid.CreateOrderRequest{
			Coin:          coin,
			IsBuy:         args.IsBuy,
			Price:         price,
			Size:          size,
			ReduceOnly:    args.ReduceOnly,
			OrderType:     orderType,
			ClientOrderID: cloidPointer(args.Cloid),
		})
		if err != nil {
			return nil, PlaceOrderResult{}, fmt.Errorf("hyperliquid_place_order: %w", err)
		}
		return nil, PlaceOrderResult{Message: "Order placed successfully", Data: status}, nil
	}
}

func placeBracketOrderHandler(client ordersClient) mcp.ToolHandlerFor[PlaceBracketOrderArgs, PlaceBracketOrderResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args PlaceBracketOrderArgs) (*mcp.CallToolResult, PlaceBracketOrderResult, error) {
		coin, err := client.CoinForAsset(ctx, args.Asset)
		if err != nil {
			return nil, PlaceBracketOrderResult{}, fmt.Errorf("hyperliquid_place_bracket_order: %w", err)
		}

		size, err := parseSize(args.Size)
		if err != nil {
			return nil, PlaceBracketOrderResult{}, fmt.Errorf("hyperliquid_place_bracket_order: %w", err)
		}
		entry, err := parsePrice(args.EntryPrice)
		if err != nil {
			return nil, PlaceBracketOrderResult{}, fmt.Errorf("hyperliquid_place_bracket_order: %w", err)
		}
		takeProfit, err := parsePrice(args.TakeProfitPrice)
		if err != nil {
			return nil, PlaceBracketOrderResult{}, fmt.Errorf("hyperliquid_place_bracket_order: %w", err)
		}
		stopLoss, err := parsePrice(args.StopLossPrice)
		if err != nil {
			return nil, PlaceBracketOrderResult{}, fmt.Errorf("hyperliquid_place_bracket_order: %w", err)
		}

		resp, err := client.PlaceBracketOrder(ctx, hlclient.BracketOrderParams{
			Coin:            coin,
			IsBuy:           args.IsBuy,
			Size:            size,
			EntryPrice:      entry,
			TakeProfitPrice: takeProfit,
			StopLossPrice:   stopLoss,
		})
		if err != nil {
			return nil, PlaceBracketOrderResult{}, fmt.Errorf("hyperliquid_place_bracket_order: %w", err)
		}
		return nil, PlaceBracketOrderResult{Message: "Bracket order placed successfully", Data: resp.Data}, nil
	}
}

func cancelOrderHandler(client ordersClient) mcp.ToolHandlerFor[CancelOrderArgs, CancelOrderResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args CancelOrderArgs) (*mcp.CallToolResult, CancelOrderResult, error) {
		if _, err := client.CancelOrder(ctx, args.Coin, args.Oid); err != nil {
			return nil, CancelOrderResult{}, fmt.Errorf("hyperliquid_cancel_order: %w", err)
		}
		return nil, CancelOrderResult{
			Message:        fmt.Sprintf("Order %d cancelled for %s", args.Oid, args.Coin),
			CancelledOrder: CancelledOrder{Coin: args.Coin, OrderID: args.Oid},
		}, nil
	}
}

func cancelAllOrdersHandler(client ordersClient) mcp.ToolHandlerFor[CancelAllOrdersArgs, CancelAllOrdersResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args CancelAllOrdersArgs) (*mcp.CallToolResult, CancelAllOrdersResult, error) {
		address := args.UserAddress
		if address == "" {
			address = client.AccountAddress()
		}
		cancelled, _, err := client.CancelAllOrders(ctx, address, args.Dex)
		if err != nil {
			return nil, CancelAllOrdersResult{}, fmt.Errorf("hyperliquid_cancel_all_orders: %w", err)
		}
		if cancelled == 0 {
			return nil, CancelAllOrdersResult{Message: "No open orders to cancel", CancelledCount: 0}, nil
		}
		return nil, CancelAllOrdersResult{
			Message:        fmt.Sprintf("Cancelled %d open orders", cancelled),
			CancelledCount: cancelled,
		}, nil
	}
}

func modifyOrderHandler(client ordersClient) mcp.ToolHandlerFor[ModifyOrderArgs, ModifyOrderResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args ModifyOrderArgs) (*mcp.CallToolResult, ModifyOrderResult, error) {
		size, err := parseSize(args.Size)
		if err != nil {
			return nil, ModifyOrderResult{}, fmt.Errorf("hyperliquid_modify_order: %w", err)
		}
		price, err := parsePrice(args.Price)
		if err != nil {
			return nil, ModifyOrderResult{}, fmt.Errorf("hyperliquid_modify_order: %w", err)
		}
		orderType, err := buildOrderType(args.Tif, args.TriggerPx, args.IsMarket, args.Tpsl)
		if err != nil {
			return nil, ModifyOrderResult{}, fmt.Errorf("hyperliquid_modify_order: %w", err)
		}

		oid := args.Oid
		status, err := client.ModifyOrder(ctx, hyperliquid.ModifyOrderRequest{
			Oid: &oid,
			Order: hyperliquid.CreateOrderRequest{
				Coin:          args.Coin,
				IsBuy:         args.IsBuy,
				Price:         price,
				Size:          size,
				ReduceOnly:    args.ReduceOnly,
				OrderType:     orderType,
				ClientOrderID: cloidPointer(args.Cloid),
			},
		})
		if err != nil {
			return nil, ModifyOrderResult{}, fmt.Errorf("hyperliquid_modify_order: %w", err)
		}
		return nil, ModifyOrderResult{Message: "Order modified successfully", Data: status}, nil
	}
}

// placeTwapOrderHandler and cancelTwapOrderHandler are stubs: the original
// Python server lists TWAP orders as "coming soon" and the underlying Go SDK
// exposes no TWAP action, so these tools report not-implemented rather than
// silently doing nothing.
func placeTwapOrderHandler(_ ordersClient) mcp.ToolHandlerFor[PlaceTwapOrderArgs, TwapStubResult] {
	return func(_ context.Context, _ *mcp.CallToolRequest, _ PlaceTwapOrderArgs) (*mcp.CallToolResult, TwapStubResult, error) {
		return nil, TwapStubResult{Message: "TWAP orders are not yet implemented"}, nil
	}
}

func cancelTwapOrderHandler(_ ordersClient) mcp.ToolHandlerFor[CancelTwapOrderArgs, TwapStubResult] {
	return func(_ context.Context, _ *mcp.CallToolRequest, _ CancelTwapOrderArgs) (*mcp.CallToolResult, TwapStubResult, error) {
		return nil, TwapStubResult{Message: "TWAP orders are not yet implemented"}, nil
	}
}

// RegisterOrderTools registers the order management MCP tools on server.
func RegisterOrderTools(server *mcp.Server, client ordersClient) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "hyperliquid_place_order",
		Description: "Place a single order on Hyperliquid. Use asset index from hyperliquid_get_meta.",
	}, placeOrderHandler(client))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "hyperliquid_place_bracket_order",
		Description: "Place entry + take profit + stop loss atomically in a single batch",
	}, placeBracketOrderHandler(client))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "hyperliquid_cancel_order",
		Description: "Cancel a specific order by coin name and order ID (oid)",
	}, cancelOrderHandler(client))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "hyperliquid_cancel_all_orders",
		Description: "Cancel all open orders for the user",
	}, cancelAllOrdersHandler(client))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "hyperliquid_modify_order",
		Description: "Modify an existing order",
	}, modifyOrderHandler(client))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "hyperliquid_place_twap_order",
		Description: "Place a TWAP order (not yet implemented)",
	}, placeTwapOrderHandler(client))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "hyperliquid_cancel_twap_order",
		Description: "Cancel a TWAP order (not yet implemented)",
	}, cancelTwapOrderHandler(client))
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/tools/... -run 'TestPlaceOrderHandler|TestPlaceBracketOrderHandler|TestCancelOrderHandler|TestCancelAllOrdersHandler|TestModifyOrderHandler|TestPlaceTwapOrderHandler|TestCancelTwapOrderHandler' -v`
Expected: PASS for all eleven tests.

- [ ] **Step 5: Commit**

```bash
git add internal/tools/orders.go internal/tools/orders_test.go
git commit -m "feat: add order placement, bracket order, cancel, modify, and TWAP stub tools"
```

---

## Task 8: `internal/tools/vault.go` — vault tools

**Files:**
- Create: `internal/tools/vault.go`
- Test: `internal/tools/vault_test.go`

**Interfaces:**
- Consumes: `hlclient.Client` (Task 3) via a local `vaultClient` interface.
- Produces: `tools.RegisterVaultTools(server *mcp.Server, client vaultClient)`.

- [ ] **Step 1: Write the failing tests**

```go
// internal/tools/vault_test.go
package tools

import (
	"context"
	"encoding/json"
	"testing"
)

type fakeVaultClient struct {
	details         json.RawMessage
	performance     json.RawMessage
	gotVaultAddress string
	gotStartTime    int64
	err             error
}

func (f *fakeVaultClient) VaultDetails(ctx context.Context, vaultAddress string) (json.RawMessage, error) {
	f.gotVaultAddress = vaultAddress
	return f.details, f.err
}

func (f *fakeVaultClient) VaultPerformance(ctx context.Context, vaultAddress string, startTime int64, endTime *int64) (json.RawMessage, error) {
	f.gotVaultAddress, f.gotStartTime = vaultAddress, startTime
	return f.performance, f.err
}

func TestVaultDetailsHandler(t *testing.T) {
	client := &fakeVaultClient{details: json.RawMessage(`{"name":"Test Vault"}`)}

	_, out, err := vaultDetailsHandler(client)(context.Background(), nil, VaultDetailsArgs{VaultAddress: "0xVault"})
	if err != nil {
		t.Fatalf("handler unexpected error: %v", err)
	}
	if client.gotVaultAddress != "0xVault" {
		t.Fatalf("expected vault address to be passed through, got %q", client.gotVaultAddress)
	}
	if string(out.Data) != `{"name":"Test Vault"}` {
		t.Fatalf("vaultDetails output = %s", out.Data)
	}
}

func TestVaultPerformanceHandler(t *testing.T) {
	client := &fakeVaultClient{performance: json.RawMessage(`{"vaultAddress":"0xVault","pnls":[]}`)}

	_, out, err := vaultPerformanceHandler(client)(context.Background(), nil, VaultPerformanceArgs{
		VaultAddress: "0xVault", StartTime: 1700000000000,
	})
	if err != nil {
		t.Fatalf("handler unexpected error: %v", err)
	}
	if client.gotStartTime != 1700000000000 {
		t.Fatalf("expected startTime to be passed through, got %d", client.gotStartTime)
	}
	if string(out.Data) != `{"vaultAddress":"0xVault","pnls":[]}` {
		t.Fatalf("vaultPerformance output = %s", out.Data)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/tools/... -run 'TestVaultDetailsHandler|TestVaultPerformanceHandler' -v`
Expected: FAIL — vault tool symbols are undefined.

- [ ] **Step 3: Write the implementation**

```go
// internal/tools/vault.go
package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type vaultClient interface {
	VaultDetails(ctx context.Context, vaultAddress string) (json.RawMessage, error)
	VaultPerformance(ctx context.Context, vaultAddress string, startTime int64, endTime *int64) (json.RawMessage, error)
}

type VaultDetailsArgs struct {
	VaultAddress string `json:"vaultAddress" jsonschema:"Vault address in 42-character hexadecimal format"`
}

type VaultDetailsResult struct {
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type VaultPerformanceArgs struct {
	VaultAddress string `json:"vaultAddress" jsonschema:"Vault address in 42-character hexadecimal format"`
	StartTime    int64  `json:"startTime" jsonschema:"Start time in milliseconds"`
	EndTime      int64  `json:"endTime,omitempty" jsonschema:"End time in milliseconds (optional, defaults to current time)"`
}

type VaultPerformanceResult struct {
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func vaultDetailsHandler(client vaultClient) mcp.ToolHandlerFor[VaultDetailsArgs, VaultDetailsResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args VaultDetailsArgs) (*mcp.CallToolResult, VaultDetailsResult, error) {
		data, err := client.VaultDetails(ctx, args.VaultAddress)
		if err != nil {
			return nil, VaultDetailsResult{}, fmt.Errorf("hyperliquid_vault_details: %w", err)
		}
		return nil, VaultDetailsResult{Message: "Vault details retrieved successfully", Data: data}, nil
	}
}

func vaultPerformanceHandler(client vaultClient) mcp.ToolHandlerFor[VaultPerformanceArgs, VaultPerformanceResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args VaultPerformanceArgs) (*mcp.CallToolResult, VaultPerformanceResult, error) {
		data, err := client.VaultPerformance(ctx, args.VaultAddress, args.StartTime, optionalEndTime(args.EndTime))
		if err != nil {
			return nil, VaultPerformanceResult{}, fmt.Errorf("hyperliquid_vault_performance: %w", err)
		}
		return nil, VaultPerformanceResult{Message: "Vault performance retrieved successfully", Data: data}, nil
	}
}

// RegisterVaultTools registers the vault MCP tools on server.
func RegisterVaultTools(server *mcp.Server, client vaultClient) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "hyperliquid_vault_details",
		Description: "Get detailed information about a specific vault",
	}, vaultDetailsHandler(client))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "hyperliquid_vault_performance",
		Description: "Get performance metrics for a specific vault",
	}, vaultPerformanceHandler(client))
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/tools/... -run 'TestVaultDetailsHandler|TestVaultPerformanceHandler' -v`
Expected: PASS for both tests.

- [ ] **Step 5: Commit**

```bash
git add internal/tools/vault.go internal/tools/vault_test.go
git commit -m "feat: add vault details and performance MCP tools"
```

---

## Task 9: `internal/tools/misc.go` — server time tool

**Files:**
- Create: `internal/tools/misc.go`
- Test: `internal/tools/misc_test.go`

**Interfaces:**
- Consumes: `hlclient.Client` (Task 3) via a local `miscClient` interface.
- Produces: `tools.RegisterMiscTools(server *mcp.Server, client miscClient)`.

- [ ] **Step 1: Write the failing test**

```go
// internal/tools/misc_test.go
package tools

import (
	"context"
	"testing"
)

type fakeMiscClient struct {
	now int64
}

func (f *fakeMiscClient) ServerTimeMillis() int64 { return f.now }

func TestGetServerTimeHandler(t *testing.T) {
	client := &fakeMiscClient{now: 1700000000000}

	_, out, err := getServerTimeHandler(client)(context.Background(), nil, GetServerTimeArgs{})
	if err != nil {
		t.Fatalf("handler unexpected error: %v", err)
	}
	if out.Data.ServerTime != 1700000000000 {
		t.Fatalf("getServerTime output = %+v", out)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/tools/... -run TestGetServerTimeHandler -v`
Expected: FAIL — `getServerTimeHandler`/`GetServerTimeArgs` are undefined.

- [ ] **Step 3: Write the implementation**

```go
// internal/tools/misc.go
package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type miscClient interface {
	ServerTimeMillis() int64
}

type GetServerTimeArgs struct{}

type ServerTimeData struct {
	ServerTime int64 `json:"serverTime"`
}

type GetServerTimeResult struct {
	Message string          `json:"message"`
	Data    ServerTimeData  `json:"data"`
}

func getServerTimeHandler(client miscClient) mcp.ToolHandlerFor[GetServerTimeArgs, GetServerTimeResult] {
	return func(_ context.Context, _ *mcp.CallToolRequest, _ GetServerTimeArgs) (*mcp.CallToolResult, GetServerTimeResult, error) {
		return nil, GetServerTimeResult{
			Message: "Server time retrieved successfully",
			Data:    ServerTimeData{ServerTime: client.ServerTimeMillis()},
		}, nil
	}
}

// RegisterMiscTools registers the utility MCP tools on server.
func RegisterMiscTools(server *mcp.Server, client miscClient) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "hyperliquid_get_server_time",
		Description: "Get estimated server time",
	}, getServerTimeHandler(client))
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/tools/... -v`
Expected: PASS for every test in the package (Tasks 4-9 combined).

- [ ] **Step 5: Commit**

```bash
git add internal/tools/misc.go internal/tools/misc_test.go
git commit -m "feat: add server time MCP tool"
```

---

## Task 10: `cmd/main.go` — wire config, client, and transport

**Files:**
- Modify: `cmd/main.go` (replace the Task 1 placeholder)

**Interfaces:**
- Consumes: `config.Load()` (Task 2), `hlclient.New()` (Task 3), all six `tools.Register*Tools()` functions (Tasks 4-9).
- Produces: the runnable server binary; no other task depends on `main.go`.

`*hlclient.Client` already has the exact methods each `tools.Register*Tools` interface needs (Tasks 4-9), so it satisfies `accountClient`, `marketClient`, `queriesClient`, `ordersClient`, `vaultClient`, and `miscClient` structurally — no adapter code is needed here.

- [ ] **Step 1: Replace the placeholder with the real entrypoint**

```go
// cmd/main.go
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/duongnv129/hyperliquid-mcp/internal/config"
	"github.com/duongnv129/hyperliquid-mcp/internal/hlclient"
	"github.com/duongnv129/hyperliquid-mcp/internal/tools"
)

func main() {
	transport := flag.String("transport", "stdio", "transport to serve on: stdio or http")
	addr := flag.String("addr", ":8080", "address to listen on when --transport=http")
	flag.Parse()

	if err := run(*transport, *addr); err != nil {
		fmt.Fprintln(os.Stderr, "hyperliquid-mcp:", err)
		os.Exit(1)
	}
}

func run(transport, addr string) error {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config error: %w", err)
	}

	client, err := hlclient.New(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to connect to hyperliquid: %w", err)
	}

	server := mcp.NewServer(&mcp.Implementation{Name: "hyperliquid-mcp", Version: "0.1.0"}, nil)
	tools.RegisterAccountTools(server, client)
	tools.RegisterMarketTools(server, client)
	tools.RegisterQueryTools(server, client)
	tools.RegisterOrderTools(server, client)
	tools.RegisterVaultTools(server, client)
	tools.RegisterMiscTools(server, client)

	switch transport {
	case "stdio":
		return server.Run(ctx, &mcp.StdioTransport{})
	case "http":
		handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, nil)
		log.Printf("hyperliquid-mcp listening on %s (http transport)", addr)
		return http.ListenAndServe(addr, handler)
	default:
		return fmt.Errorf("unknown transport %q (want stdio or http)", transport)
	}
}
```

- [ ] **Step 2: Verify the module builds and vets cleanly**

Run: `go build ./... && go vet ./...`
Expected: both exit 0 with no output.

- [ ] **Step 3: Run the full test suite**

Run: `go test ./...`
Expected: PASS for `internal/config`, `internal/hlclient`, `internal/tools`; `cmd` has no test files, which is fine — its only job is wiring, exercised manually below.

- [ ] **Step 4: Manually smoke-test the http transport**

This needs a syntactically valid private key (no funds required — `Meta`/`SpotMeta` are public endpoints, so construction succeeds against testnet even for an unfunded, freshly generated key):

```bash
export HYPERLIQUID_PRIVATE_KEY=0x4646464646464646464646464646464646464646464646464646464646464646
export HYPERLIQUID_TESTNET=true
go run ./cmd --transport=http --addr=:8080 &
sleep 1
curl -s -X POST http://localhost:8080/mcp \
  -H 'Content-Type: application/json' \
  -H 'Accept: application/json, text/event-stream' \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'
kill %1
```

Expected: the server starts without error/panic, and the `curl` response is a JSON-RPC result listing 20 tools (`hyperliquid_get_account_info`, `hyperliquid_place_order`, etc.) — confirming all six `Register*Tools` calls wired up successfully. If the private key fails to decode or the testnet API is unreachable, the process exits non-zero with a clear `hyperliquid-mcp: ...` error from `run()` instead of a panic.

- [ ] **Step 5: Commit**

```bash
git add cmd/main.go
git commit -m "feat: wire config, hlclient, and MCP tool registration into the server entrypoint"
```

---

## Task 11: README and final verification

**Files:**
- Create: `README.md`

**Interfaces:**
- None — this task documents the finished server; no other task depends on it.

- [ ] **Step 1: Write `README.md`**

```markdown
# hyperliquid-mcp (Go)

A Go MCP (Model Context Protocol) server for Hyperliquid perpetual trading, providing AI assistants tool access to account data, order management, market data, and vaults. Built on [`modelcontextprotocol/go-sdk`](https://github.com/modelcontextprotocol/go-sdk) and [`sonirico/go-hyperliquid`](https://github.com/sonirico/go-hyperliquid). Full tool parity with the original Python [`edkdev/hyperliquid-mcp`](https://github.com/edkdev/hyperliquid-mcp).

See [docs/specs/2026-06-25-hyperliquid-mcp-go-design.md](docs/specs/2026-06-25-hyperliquid-mcp-go-design.md) for the design.

## Build

```bash
go build ./...
```

## Configuration

Set these environment variables before running:

| Variable | Required | Description |
|---|---|---|
| `HYPERLIQUID_PRIVATE_KEY` | yes | 0x-prefixed 64-character hex private key used to sign transactions |
| `HYPERLIQUID_ACCOUNT_ADDRESS` | no | Agent mode: trading account address, if different from the signer |
| `HYPERLIQUID_TESTNET` | no | `"true"` to use testnet; defaults to mainnet |
| `HYPERLIQUID_VAULT_ADDRESS` | no | Vault address, for vault trading |

Your wallet must be registered on Hyperliquid (deposit any amount from Arbitrum at app.hyperliquid.xyz, or the testnet equivalent) before trading tools will work.

## Running

stdio (for Claude Desktop / Kiro `mcp.json`):

```bash
go run ./cmd
```

Example `mcp.json` entry:

```json
{
  "mcpServers": {
    "hyperliquid": {
      "command": "go",
      "args": ["run", "/path/to/hyperliquid-mcp/cmd"],
      "env": {
        "HYPERLIQUID_PRIVATE_KEY": "0x...",
        "HYPERLIQUID_TESTNET": "false"
      }
    }
  }
}
```

Streamable HTTP/SSE (for network MCP clients):

```bash
go run ./cmd --transport=http --addr=:8080
```

## Testing

```bash
go test ./...
```

Unit tests cover config parsing, the Hyperliquid client wrapper (against an `httptest.Server`, no live network calls), and every MCP tool handler (against fakes implementing each file's small consumer interface). There is no automated testnet order-placement test — verify that manually:

1. Set `HYPERLIQUID_TESTNET=true` and a funded testnet private key.
2. Run `go run ./cmd` and connect an MCP client (or use the http transport with `curl`, see Task 10 of the implementation plan for an example request).
3. Call `hyperliquid_get_meta`, then `hyperliquid_place_order` with a small size, then `hyperliquid_cancel_order` to confirm round-trip order placement and cancellation.

## Tool list

20 tools across account/positions, order management, order queries, market data, vaults, and utility — see [docs/specs/2026-06-25-hyperliquid-mcp-go-design.md](docs/specs/2026-06-25-hyperliquid-mcp-go-design.md) for the full table.
```

- [ ] **Step 2: Run the full verification suite one last time**

```bash
go build ./...
go vet ./...
go test ./... -v
```

Expected: all three commands exit 0; `go test` shows every test from Tasks 2-9 passing, with no skipped or failing tests.

- [ ] **Step 3: Commit**

```bash
git add README.md
git commit -m "docs: add README with configuration, running, and testing instructions"
```

