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
	Message string                  `json:"message"`
	Data    []hyperliquid.OpenOrder `json:"data"`
}

type GetOrderStatusArgs struct {
	UserAddress string `json:"userAddress,omitempty" jsonschema:"User address (optional, defaults to configured account)"`
	Oid         int64  `json:"oid" jsonschema:"Order ID (oid) to look up"`
}

type GetOrderStatusResult struct {
	Message string                        `json:"message"`
	Data    *hyperliquid.OrderQueryResult `json:"data"`
}

type GetUserFillsArgs struct {
	UserAddress string `json:"userAddress,omitempty" jsonschema:"User address (optional, defaults to configured account)"`
}

type GetUserFillsResult struct {
	Message string             `json:"message"`
	Data    []hyperliquid.Fill `json:"data"`
}

type GetUserFundingArgs struct {
	UserAddress string `json:"userAddress,omitempty" jsonschema:"User address (optional, defaults to configured account)"`
	StartTime   int64  `json:"startTime" jsonschema:"Start time in milliseconds"`
	EndTime     int64  `json:"endTime,omitempty" jsonschema:"End time in milliseconds (optional, defaults to current time)"`
}

type GetUserFundingResult struct {
	Message string                           `json:"message"`
	Data    []hyperliquid.UserFundingHistory `json:"data"`
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
