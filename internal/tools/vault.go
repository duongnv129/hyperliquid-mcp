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
