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
