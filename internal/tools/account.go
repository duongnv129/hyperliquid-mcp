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
	Message string               `json:"message"`
	Data    *hyperliquid.UserState `json:"data"`
}

type GetPositionsArgs struct {
	UserAddress string `json:"userAddress,omitempty" jsonschema:"User address (optional, defaults to configured account)"`
	Dex         string `json:"dex,omitempty" jsonschema:"Perp dex name (optional, defaults to empty string)"`
}

type GetPositionsResult struct {
	Message string                      `json:"message"`
	Data    []hyperliquid.AssetPosition `json:"data"`
	Summary hyperliquid.MarginSummary   `json:"marginSummary"`
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
