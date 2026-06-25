# Hyperliquid MCP Server (Go) — Design

## Summary

Reimplement [edkdev/hyperliquid-mcp](https://github.com/edkdev/hyperliquid-mcp) — a Python Model Context Protocol (MCP) server for Hyperliquid perpetual trading — as a Go server with full feature parity (all ~20 tools). The Go server gives AI assistants tool access to Hyperliquid account data, order management, market data, and vault information.

## Goals

- Full parity with the Python original's tool surface (account, orders, market data, vaults, utility).
- No hand-rolled EIP-712 signing or REST plumbing — delegate to an existing, well-tested Go Hyperliquid client.
- Usable as a stdio subprocess (Claude Desktop / Kiro via `mcp.json`) and as a streamable HTTP/SSE server for remote MCP clients.
- Config via environment variables only, matching the original's `mcp.json` `env` block convention.

## Non-goals

- Real TWAP order implementation (kept as stub tools, matching the original's "coming soon" status).
- Sub-accounts, multi-sig, validator/consensus-layer actions, spot deployment, or other advanced features exposed by the underlying SDK but not present in the original Python tool list.
- A persistent database, caching layer, or background jobs — the server is stateless per request (aside from an in-memory asset-index cache).

## Dependencies

- [`github.com/modelcontextprotocol/go-sdk`](https://github.com/modelcontextprotocol/go-sdk) — official Go MCP SDK (maintained with Google), provides `mcp.Server`, tool registration, `StdioTransport`, and streamable HTTP transport.
- [`github.com/sonirico/go-hyperliquid`](https://github.com/sonirico/go-hyperliquid) — unofficial but full-parity Go client for the Hyperliquid API (REST `Info`/`Exchange`, EIP-712 signing via `go-ethereum/crypto`, WebSocket support though unused here). MIT licensed.

Both pinned to Go 1.25+. Local copies are checked out under `reference/` (git-ignored) for implementation reference only — not vendored into the build.

## Architecture

A single Go binary that wires together config, a Hyperliquid client wrapper, and an MCP server with ~20 registered tools.

```
cmd/hyperliquid-mcp/
  main.go              — load config, build hlclient, build mcp.Server, run transport

internal/config/
  config.go            — env var parsing & validation (private key, testnet, account/vault address)

internal/hlclient/
  client.go            — wraps go-hyperliquid Exchange + Info; asset symbol <-> index cache;
                          bracket-order request builder (single BulkOrders call)

internal/tools/
  account.go           — get_account_info, get_positions, get_balance
  orders.go            — place_order, place_bracket_order, cancel_order, cancel_all_orders,
                          modify_order, place_twap_order (stub), cancel_twap_order (stub)
  queries.go           — get_open_orders, get_order_status, get_user_fills, get_user_funding
  market.go            — get_meta, get_all_mids, get_order_book, get_recent_trades,
                          get_historical_funding, get_candles
  vault.go             — vault_details, vault_performance
  misc.go              — get_server_time
```

`main.go` selects transport via a `--transport stdio|http` flag (default `stdio`); the HTTP mode binds an address from `--addr` (default `:8080`) and serves the SDK's streamable HTTP/SSE handler.

## Configuration

Environment variables, read once at startup:

| Variable | Required | Notes |
|---|---|---|
| `HYPERLIQUID_PRIVATE_KEY` | yes | Signing key for the `Exchange` client |
| `HYPERLIQUID_ACCOUNT_ADDRESS` | no | Agent mode: trading account if different from signer |
| `HYPERLIQUID_TESTNET` | no | `"true"` for testnet, default mainnet |
| `HYPERLIQUID_VAULT_ADDRESS` | no | Vault trading |

`internal/config` validates the private key format and required fields at startup; the process exits with a clear error before the MCP loop starts if config is invalid. No `.env` file support — matches the original's current (env-only) configuration approach.

## Tool surface (full parity, 20 tools)

| Category | Tools |
|---|---|
| Account & positions | `hyperliquid_get_account_info`, `hyperliquid_get_positions`, `hyperliquid_get_balance` |
| Order management | `hyperliquid_place_order`, `hyperliquid_place_bracket_order`, `hyperliquid_cancel_order`, `hyperliquid_cancel_all_orders`, `hyperliquid_modify_order`, `hyperliquid_place_twap_order` (stub), `hyperliquid_cancel_twap_order` (stub) |
| Order queries | `hyperliquid_get_open_orders`, `hyperliquid_get_order_status`, `hyperliquid_get_user_fills`, `hyperliquid_get_user_funding` |
| Market data | `hyperliquid_get_meta`, `hyperliquid_get_all_mids`, `hyperliquid_get_order_book`, `hyperliquid_get_recent_trades`, `hyperliquid_get_historical_funding`, `hyperliquid_get_candles` |
| Vault | `hyperliquid_vault_details`, `hyperliquid_vault_performance` |
| Utility | `hyperliquid_get_server_time` |

Each tool's JSON input schema mirrors the original Python tool's parameters (asset index or symbol, size, price, side, time ranges, etc.), translated to Go MCP SDK tool definitions.

### Bracket orders

`hyperliquid_place_bracket_order` builds three order requests (entry limit, take-profit trigger reduce-only, stop-loss trigger reduce-only) and submits them in a single `Exchange.BulkOrders()` call, matching the original's atomic-placement behavior.

### TWAP stubs

`hyperliquid_place_twap_order` / `hyperliquid_cancel_twap_order` are registered tools that immediately return a "not yet implemented" tool result, matching the original's "coming soon" status. No TWAP action exists in the underlying SDK's exposed methods.

## Error handling

- **Startup**: config validation failures (bad/missing private key) print a clear message to stderr and exit non-zero — the server never starts the MCP loop with invalid credentials.
- **Tool input validation**: each handler validates arguments before calling the SDK — asset symbol/index resolution via `Info.CoinToAsset`, minimum order value ($10, matching the original's documented minimum), required-field checks — and returns an MCP tool error result (not a panic) with a message matching the original's user-facing error strings where applicable (e.g. `"Order value must be at least $10"`).
- **SDK/network errors**: wrapped with context (which call failed, which asset/order) and returned as tool-call failures. Never silently discarded.

## Testing

- **Unit tests**: `internal/config` (env parsing, validation edge cases) and `internal/hlclient` (asset index cache behavior, bracket-order request construction) — table-driven, no network calls.
- **Tool handler tests**: each `internal/tools/*.go` file tested against a small mockable client interface (the subset of `hlclient` methods it needs), so handlers can be tested independently of the real SDK.
- **Testnet smoke test**: one integration test for `get_meta` + `get_all_mids` against Hyperliquid testnet, gated behind a build tag (e.g. `//go:build testnet`) or `RUN_TESTNET_TESTS=1` env var so it's opt-in, not part of default `go test ./...` / CI.
- **Manual verification**: README documents manual testnet steps for order placement/cancellation, since automated tests against live order placement require funded testnet wallets and aren't run by default.

## Open questions / risks

- `go-hyperliquid` is unofficial; if the Hyperliquid API changes in ways the SDK hasn't caught up to, tool calls relying on those endpoints will fail until the dependency is updated. Pin a specific version/tag rather than always tracking `main`.
- The official MCP Go SDK is relatively young (collaboration with Google, evolving toward v1.5). Pin a stable tagged release rather than `main`.
