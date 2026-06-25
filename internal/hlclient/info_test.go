package hlclient

import (
	"context"
	"net/http"
	"testing"
)

func TestMetaAllMidsL2Book(t *testing.T) {
	srv := testServer(t, func(w http.ResponseWriter, body map[string]any) {
		w.Header().Set("Content-Type", "application/json")
		switch body["type"] {
		case "allMids":
			_, _ = w.Write([]byte(`{"BTC": "65000.5", "ETH": "3000.1"}`))
		case "l2Book":
			_, _ = w.Write([]byte(`{"coin": "BTC", "time": 1700000000000, "levels": [[{"n":1,"px":"65000","sz":"1.5"}],[{"n":1,"px":"65001","sz":"2"}]]}`))
		default:
			t.Fatalf("unexpected request type %q", body["type"])
		}
	})
	cfg := testConfig(srv.URL)
	client, err := newForBaseURL(context.Background(), cfg, srv.URL)
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	meta, err := client.Meta(context.Background())
	if err != nil || len(meta.Universe) != 3 {
		t.Fatalf("Meta() = (%+v, %v), want 3 universe entries", meta, err)
	}

	mids, err := client.AllMids(context.Background())
	if err != nil || mids["BTC"] != "65000.5" {
		t.Fatalf("AllMids() = (%+v, %v)", mids, err)
	}

	book, err := client.L2Book(context.Background(), "BTC")
	if err != nil || book.Coin != "BTC" || len(book.Levels) != 2 {
		t.Fatalf("L2Book() = (%+v, %v)", book, err)
	}
}

func TestUserStateOpenOrders(t *testing.T) {
	srv := testServer(t, func(w http.ResponseWriter, body map[string]any) {
		w.Header().Set("Content-Type", "application/json")
		switch body["type"] {
		case "clearinghouseState":
			_, _ = w.Write([]byte(`{"assetPositions": [], "crossMarginSummary": {"accountValue":"100","totalMarginUsed":"0","totalNtlPos":"0","totalRawUsd":"100"}, "marginSummary": {"accountValue":"100","totalMarginUsed":"0","totalNtlPos":"0","totalRawUsd":"100"}, "withdrawable": "100"}`))
		case "openOrders":
			_, _ = w.Write([]byte(`[{"coin":"BTC","limitPx":"65000","oid":1,"origSz":"0.1","side":"B","sz":"0.1","timestamp":1700000000000}]`))
		default:
			t.Fatalf("unexpected request type %q", body["type"])
		}
	})
	cfg := testConfig(srv.URL)
	client, err := newForBaseURL(context.Background(), cfg, srv.URL)
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	state, err := client.UserState(context.Background(), client.AccountAddress(), "")
	if err != nil || state.Withdrawable != "100" {
		t.Fatalf("UserState() = (%+v, %v)", state, err)
	}

	orders, err := client.OpenOrders(context.Background(), client.AccountAddress(), "")
	if err != nil || len(orders) != 1 || orders[0].Coin != "BTC" {
		t.Fatalf("OpenOrders() = (%+v, %v)", orders, err)
	}
}

func TestFundingHistoryCandlesRecentTrades(t *testing.T) {
	srv := testServer(t, func(w http.ResponseWriter, body map[string]any) {
		w.Header().Set("Content-Type", "application/json")
		switch body["type"] {
		case "fundingHistory":
			_, _ = w.Write([]byte(`[{"coin":"BTC","fundingRate":"0.0001","premium":"0.0002","time":1700000000000}]`))
		case "candleSnapshot":
			_, _ = w.Write([]byte(`[{"t":1700000000000,"T":1700000060000,"i":"1m","n":10,"o":"65000","h":"65010","l":"64990","c":"65005","s":"BTC","v":"12.5"}]`))
		case "recentTrades":
			_, _ = w.Write([]byte(`[{"coin":"BTC","side":"B","px":"65000","sz":"0.1","time":1700000000000,"hash":"0xabc","tid":1,"users":["0x1","0x2"]}]`))
		default:
			t.Fatalf("unexpected request type %q", body["type"])
		}
	})
	cfg := testConfig(srv.URL)
	client, err := newForBaseURL(context.Background(), cfg, srv.URL)
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	funding, err := client.FundingHistory(context.Background(), "BTC", 1700000000000, nil)
	if err != nil || len(funding) != 1 || funding[0].FundingRate != "0.0001" {
		t.Fatalf("FundingHistory() = (%+v, %v)", funding, err)
	}

	candles, err := client.Candles(context.Background(), "BTC", "1m", 1700000000000, 1700000060000)
	if err != nil || len(candles) != 1 || candles[0].Close != "65005" {
		t.Fatalf("Candles() = (%+v, %v)", candles, err)
	}

	trades, err := client.RecentTrades(context.Background(), "BTC")
	if err != nil || len(trades) != 1 || trades[0].Hash != "0xabc" {
		t.Fatalf("RecentTrades() = (%+v, %v)", trades, err)
	}
}

func TestUserFillsUserFundingOrderStatus(t *testing.T) {
	srv := testServer(t, func(w http.ResponseWriter, body map[string]any) {
		w.Header().Set("Content-Type", "application/json")
		switch body["type"] {
		case "userFills":
			_, _ = w.Write([]byte(`[{"closedPnl":"10","coin":"BTC","crossed":true,"dir":"Open Long","hash":"0xabc","oid":1,"px":"65000","side":"B","startPosition":"0","sz":"0.1","time":1700000000000,"fee":"0.1","feeToken":"USDC","tid":1}]`))
		case "userFunding":
			_, _ = w.Write([]byte(`[{"delta":{"coin":"BTC","fundingRate":"0.0001","size":"0.1","type":"funding","usdc":"0.01"},"hash":"0xabc","time":1700000000000}]`))
		case "orderStatus":
			_, _ = w.Write([]byte(`{"status":"order","order":{}}`))
		default:
			t.Fatalf("unexpected request type %q", body["type"])
		}
	})
	cfg := testConfig(srv.URL)
	client, err := newForBaseURL(context.Background(), cfg, srv.URL)
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	fills, err := client.UserFills(context.Background(), client.AccountAddress())
	if err != nil || len(fills) != 1 || fills[0].Coin != "BTC" {
		t.Fatalf("UserFills() = (%+v, %v)", fills, err)
	}

	funding, err := client.UserFundingHistory(context.Background(), client.AccountAddress(), 1700000000000, nil)
	if err != nil || len(funding) != 1 || funding[0].Delta.Coin != "BTC" {
		t.Fatalf("UserFundingHistory() = (%+v, %v)", funding, err)
	}

	status, err := client.OrderStatus(context.Background(), client.AccountAddress(), 1)
	if err != nil || status.Status != "order" {
		t.Fatalf("OrderStatus() = (%+v, %v)", status, err)
	}
}
