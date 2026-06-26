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
	Message string                    `json:"message"`
	Data    hyperliquid.OrderResponse `json:"data"`
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
	Message        string         `json:"message"`
	CancelledOrder CancelledOrder `json:"cancelledOrder"`
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
