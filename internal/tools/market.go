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
	Message string              `json:"message"`
	Data    *hyperliquid.L2Book `json:"data"`
}

type GetRecentTradesArgs struct {
	Coin string `json:"coin" jsonschema:"Asset symbol (e.g. 'BTC', 'ETH', 'SOL')"`
}

type GetRecentTradesResult struct {
	Message string              `json:"message"`
	Data    []hyperliquid.Trade `json:"data"`
}

type GetHistoricalFundingArgs struct {
	Coin      string `json:"coin" jsonschema:"Asset symbol (e.g. 'BTC', 'ETH', 'SOL')"`
	StartTime int64  `json:"startTime" jsonschema:"Start time in milliseconds"`
	EndTime   int64  `json:"endTime,omitempty" jsonschema:"End time in milliseconds (optional, defaults to current time)"`
}

type GetHistoricalFundingResult struct {
	Message string                       `json:"message"`
	Data    []hyperliquid.FundingHistory `json:"data"`
}

type GetCandlesArgs struct {
	Coin      string `json:"coin" jsonschema:"Asset symbol (e.g. 'BTC', 'ETH', 'SOL')"`
	Interval  string `json:"interval" jsonschema:"Candle interval: 1m, 5m, 15m, 1h, 4h, or 1d"`
	StartTime int64  `json:"startTime" jsonschema:"Start time in milliseconds"`
	EndTime   int64  `json:"endTime,omitempty" jsonschema:"End time in milliseconds (optional, defaults to current time)"`
}

type GetCandlesResult struct {
	Message string               `json:"message"`
	Data    []hyperliquid.Candle `json:"data"`
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
