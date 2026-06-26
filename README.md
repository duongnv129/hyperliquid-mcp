# hyperliquid-mcp

A Go MCP (Model Context Protocol) server for [Hyperliquid](https://hyperliquid.xyz) perpetual trading. Gives AI assistants like Claude direct access to your account, orders, market data, and vaults.

Built on [`modelcontextprotocol/go-sdk`](https://github.com/modelcontextprotocol/go-sdk) and [`sonirico/go-hyperliquid`](https://github.com/sonirico/go-hyperliquid).

## Tools

23 tools across 6 categories:

| Category | Tools |
|---|---|
| Account | `get_account_info`, `get_positions`, `get_balance` |
| Market | `get_meta`, `get_all_mids`, `get_order_book`, `get_recent_trades`, `get_historical_funding`, `get_candles` |
| Orders | `place_order`, `place_bracket_order`, `place_twap_order`, `cancel_order`, `cancel_all_orders`, `cancel_twap_order`, `modify_order` |
| Queries | `get_open_orders`, `get_order_status`, `get_user_fills`, `get_user_funding` |
| Vaults | `vault_details`, `vault_performance` |
| Utility | `get_server_time` |

All tool names are prefixed with `hyperliquid_`.

## Prerequisites

- A Hyperliquid account with a private key
- For trading: wallet must be registered on Hyperliquid (deposit any amount from Arbitrum at [app.hyperliquid.xyz](https://app.hyperliquid.xyz), or the [testnet](https://app.hyperliquid.xyz/trade) equivalent)

## Configuration

| Variable | Required | Description |
|---|---|---|
| `HYPERLIQUID_PRIVATE_KEY` | yes | 0x-prefixed 64-character hex private key |
| `HYPERLIQUID_ACCOUNT_ADDRESS` | no | Agent mode: trading account address if different from signer |
| `HYPERLIQUID_TESTNET` | no | `"true"` to use testnet; defaults to mainnet |
| `HYPERLIQUID_VAULT_ADDRESS` | no | Vault address for vault trading |

## Quick Start

### Option 1: Docker (recommended)

```bash
docker run -e HYPERLIQUID_PRIVATE_KEY=0x... \
           -e HYPERLIQUID_TESTNET=false \
           -p 8080:8080 \
           duongnv129/hyperliquid-mcp:latest
```

The server listens on `http://localhost:8080` (HTTP/SSE transport).

### Option 2: Build from source

```bash
git clone https://github.com/duongnv129/hyperliquid-mcp.git
cd hyperliquid-mcp
go build -o hyperliquid-mcp ./cmd

# stdio (for Claude Desktop / Kiro)
HYPERLIQUID_PRIVATE_KEY=0x... ./hyperliquid-mcp

# HTTP/SSE (for network clients)
HYPERLIQUID_PRIVATE_KEY=0x... ./hyperliquid-mcp --transport=http --addr=:8080
```

## Connecting to Claude Desktop

Add this to your `claude_desktop_config.json`:

**Using Docker:**

```json
{
  "mcpServers": {
    "hyperliquid": {
      "command": "docker",
      "args": [
        "run", "--rm", "-i",
        "-e", "HYPERLIQUID_PRIVATE_KEY",
        "-e", "HYPERLIQUID_TESTNET",
        "duongnv129/hyperliquid-mcp:latest",
        "--transport=stdio"
      ],
      "env": {
        "HYPERLIQUID_PRIVATE_KEY": "0x...",
        "HYPERLIQUID_TESTNET": "false"
      }
    }
  }
}
```

**Using local binary:**

```json
{
  "mcpServers": {
    "hyperliquid": {
      "command": "/path/to/hyperliquid-mcp",
      "env": {
        "HYPERLIQUID_PRIVATE_KEY": "0x...",
        "HYPERLIQUID_TESTNET": "false"
      }
    }
  }
}
```

## Connecting to Kiro / other MCP clients

```json
{
  "mcpServers": {
    "hyperliquid": {
      "command": "docker",
      "args": [
        "run", "--rm", "-i",
        "-e", "HYPERLIQUID_PRIVATE_KEY",
        "duongnv129/hyperliquid-mcp:latest",
        "--transport=stdio"
      ],
      "env": {
        "HYPERLIQUID_PRIVATE_KEY": "0x..."
      }
    }
  }
}
```

## HTTP Transport

Run the server in HTTP mode for network MCP clients or testing:

```bash
docker run -e HYPERLIQUID_PRIVATE_KEY=0x... \
           -p 8080:8080 \
           duongnv129/hyperliquid-mcp:latest
```

Initialize and list tools:

```bash
# Step 1: initialize and capture session ID
SESSION=$(curl -s -D - -X POST http://localhost:8080/ \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' \
  | grep "Mcp-Session-Id" | awk '{print $2}' | tr -d '\r\n')

# Step 2: list all tools
curl -s -X POST http://localhost:8080/ \
  -H "Content-Type: application/json" \
  -H "Mcp-Session-Id: $SESSION" \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}'
```

## Testing

```bash
go test ./...
```

To verify against testnet manually:

1. Set `HYPERLIQUID_TESTNET=true` and a funded testnet key
2. Start the server
3. Call `hyperliquid_get_meta` → `hyperliquid_place_order` → `hyperliquid_cancel_order`

## Docker Images

Images are published to [Docker Hub](https://hub.docker.com/r/duongnv129/hyperliquid-mcp):

```bash
docker pull duongnv129/hyperliquid-mcp:latest   # latest stable
docker pull duongnv129/hyperliquid-mcp:1.0.1    # specific version
```
