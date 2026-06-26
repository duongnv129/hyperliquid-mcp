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
