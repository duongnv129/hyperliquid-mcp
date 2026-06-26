// internal/tools/market_test.go
package tools

import (
	"context"
	"testing"

	hyperliquid "github.com/sonirico/go-hyperliquid"
)

type fakeMarketClient struct {
	meta    *hyperliquid.Meta
	mids    map[string]string
	book    *hyperliquid.L2Book
	trades  []hyperliquid.Trade
	funding []hyperliquid.FundingHistory
	candles []hyperliquid.Candle
}

func (f *fakeMarketClient) Meta(ctx context.Context) (*hyperliquid.Meta, error) { return f.meta, nil }
func (f *fakeMarketClient) AllMids(ctx context.Context) (map[string]string, error) {
	return f.mids, nil
}
func (f *fakeMarketClient) L2Book(ctx context.Context, coin string) (*hyperliquid.L2Book, error) {
	return f.book, nil
}
func (f *fakeMarketClient) RecentTrades(ctx context.Context, coin string) ([]hyperliquid.Trade, error) {
	return f.trades, nil
}
func (f *fakeMarketClient) FundingHistory(ctx context.Context, coin string, startTime int64, endTime *int64) ([]hyperliquid.FundingHistory, error) {
	return f.funding, nil
}
func (f *fakeMarketClient) Candles(ctx context.Context, coin, interval string, startTime, endTime int64) ([]hyperliquid.Candle, error) {
	return f.candles, nil
}

func TestGetMeta(t *testing.T) {
	client := &fakeMarketClient{meta: &hyperliquid.Meta{Universe: []hyperliquid.AssetInfo{{Name: "BTC"}}}}

	_, out, err := getMetaHandler(client)(context.Background(), nil, GetMetaArgs{})
	if err != nil {
		t.Fatalf("handler unexpected error: %v", err)
	}
	if len(out.Data.Universe) != 1 || out.Data.Universe[0].Name != "BTC" {
		t.Fatalf("getMeta output = %+v", out)
	}
}

func TestGetAllMids(t *testing.T) {
	client := &fakeMarketClient{mids: map[string]string{"BTC": "65000"}}

	_, out, err := getAllMidsHandler(client)(context.Background(), nil, GetAllMidsArgs{})
	if err != nil {
		t.Fatalf("handler unexpected error: %v", err)
	}
	if out.Data["BTC"] != "65000" {
		t.Fatalf("getAllMids output = %+v", out)
	}
}

func TestGetOrderBook(t *testing.T) {
	client := &fakeMarketClient{book: &hyperliquid.L2Book{Coin: "BTC"}}

	_, out, err := getOrderBookHandler(client)(context.Background(), nil, GetOrderBookArgs{Coin: "BTC"})
	if err != nil {
		t.Fatalf("handler unexpected error: %v", err)
	}
	if out.Data.Coin != "BTC" {
		t.Fatalf("getOrderBook output = %+v", out)
	}
}

func TestGetRecentTrades(t *testing.T) {
	client := &fakeMarketClient{trades: []hyperliquid.Trade{{Coin: "BTC", Hash: "0xabc"}}}

	_, out, err := getRecentTradesHandler(client)(context.Background(), nil, GetRecentTradesArgs{Coin: "BTC"})
	if err != nil {
		t.Fatalf("handler unexpected error: %v", err)
	}
	if len(out.Data) != 1 || out.Data[0].Hash != "0xabc" {
		t.Fatalf("getRecentTrades output = %+v", out)
	}
}

func TestGetHistoricalFunding(t *testing.T) {
	client := &fakeMarketClient{funding: []hyperliquid.FundingHistory{{Coin: "BTC", FundingRate: "0.0001"}}}

	_, out, err := getHistoricalFundingHandler(client)(context.Background(), nil, GetHistoricalFundingArgs{Coin: "BTC", StartTime: 1700000000000})
	if err != nil {
		t.Fatalf("handler unexpected error: %v", err)
	}
	if len(out.Data) != 1 || out.Data[0].FundingRate != "0.0001" {
		t.Fatalf("getHistoricalFunding output = %+v", out)
	}
}

func TestGetCandles(t *testing.T) {
	client := &fakeMarketClient{candles: []hyperliquid.Candle{{Symbol: "BTC", Close: "65000"}}}

	_, out, err := getCandlesHandler(client)(context.Background(), nil, GetCandlesArgs{
		Coin: "BTC", Interval: "1h", StartTime: 1700000000000,
	})
	if err != nil {
		t.Fatalf("handler unexpected error: %v", err)
	}
	if len(out.Data) != 1 || out.Data[0].Close != "65000" {
		t.Fatalf("getCandles output = %+v", out)
	}
}

func TestGetCandles_DefaultsEndTimeToNow(t *testing.T) {
	client := &fakeMarketClient{}

	_, _, err := getCandlesHandler(client)(context.Background(), nil, GetCandlesArgs{
		Coin: "BTC", Interval: "1h", StartTime: 1700000000000, EndTime: 0,
	})
	if err != nil {
		t.Fatalf("handler unexpected error: %v", err)
	}
}
