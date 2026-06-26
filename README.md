# hyperliquid-mcp (Go)

A Go MCP (Model Context Protocol) server for Hyperliquid perpetual trading, providing AI assistants tool access to account data, order management, market data, and vaults. Built on [`modelcontextprotocol/go-sdk`](https://github.com/modelcontextprotocol/go-sdk) and [`sonirico/go-hyperliquid`](https://github.com/sonirico/go-hyperliquid). Full tool parity with the original Python [`edkdev/hyperliquid-mcp`](https://github.com/edkdev/hyperliquid-mcp).

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
2. Run `go run ./cmd` and connect an MCP client (or use the http transport with `curl`).
3. Call `hyperliquid_get_meta`, then `hyperliquid_place_order` with a small size, then `hyperliquid_cancel_order` to confirm round-trip order placement and cancellation.

## Tool list

23 tools across six categories:

| Category | Tools |
|---|---|
| Account | `get_account_info`, `get_positions`, `get_balance` |
| Market | `get_meta`, `get_all_mids`, `get_order_book`, `get_recent_trades`, `get_historical_funding`, `get_candles` |
| Orders | `place_order`, `place_bracket_order`, `place_twap_order`, `cancel_order`, `cancel_all_orders`, `cancel_twap_order`, `modify_order` |
| Queries | `get_open_orders`, `get_order_status`, `get_user_fills`, `get_user_funding` |
| Vaults | `vault_details`, `vault_performance` |
| Utility | `get_server_time` |

All tool names are prefixed with `hyperliquid_`.
