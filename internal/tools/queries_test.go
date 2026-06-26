// internal/tools/queries_test.go
package tools

import (
	"context"
	"encoding/json"
	"testing"

	hyperliquid "github.com/sonirico/go-hyperliquid"
)

type fakeQueriesClient struct {
	accountAddress string
	openOrders     []hyperliquid.OpenOrder
	orderStatus    *hyperliquid.OrderQueryResult
	fills          []hyperliquid.Fill
	funding        []hyperliquid.UserFundingHistory
	gotAddress     string
	gotDex         string
}

func (f *fakeQueriesClient) AccountAddress() string { return f.accountAddress }

func (f *fakeQueriesClient) OpenOrders(ctx context.Context, address, dex string) ([]hyperliquid.OpenOrder, error) {
	f.gotAddress, f.gotDex = address, dex
	return f.openOrders, nil
}

func (f *fakeQueriesClient) OrderStatus(ctx context.Context, user string, oid int64) (*hyperliquid.OrderQueryResult, error) {
	return f.orderStatus, nil
}

func (f *fakeQueriesClient) UserFills(ctx context.Context, address string) ([]hyperliquid.Fill, error) {
	f.gotAddress = address
	return f.fills, nil
}

func (f *fakeQueriesClient) UserFundingHistory(ctx context.Context, user string, startTime int64, endTime *int64) ([]hyperliquid.UserFundingHistory, error) {
	return f.funding, nil
}

func TestGetOpenOrders(t *testing.T) {
	client := &fakeQueriesClient{
		accountAddress: "0xConfigured",
		openOrders:     []hyperliquid.OpenOrder{{Coin: "BTC", Oid: 1}},
	}

	_, out, err := getOpenOrdersHandler(client)(context.Background(), nil, GetOpenOrdersArgs{})
	if err != nil {
		t.Fatalf("handler unexpected error: %v", err)
	}
	if client.gotAddress != "0xConfigured" {
		t.Fatalf("expected default address, got %q", client.gotAddress)
	}
	if len(out.Data) != 1 || out.Data[0].Coin != "BTC" {
		t.Fatalf("getOpenOrders output = %+v", out)
	}
}

func TestGetOrderStatus(t *testing.T) {
	client := &fakeQueriesClient{
		accountAddress: "0xConfigured",
		orderStatus:    &hyperliquid.OrderQueryResult{Status: hyperliquid.OrderQueryStatusSuccess},
	}

	_, out, err := getOrderStatusHandler(client)(context.Background(), nil, GetOrderStatusArgs{Oid: 1})
	if err != nil {
		t.Fatalf("handler unexpected error: %v", err)
	}
	if len(out.Data) == 0 {
		t.Fatal("getOrderStatus: expected non-empty raw JSON data")
	}
	var parsed map[string]any
	if err := json.Unmarshal(out.Data, &parsed); err != nil {
		t.Fatalf("getOrderStatus: data is not valid JSON: %v", err)
	}
	if parsed["status"] != string(hyperliquid.OrderQueryStatusSuccess) {
		t.Fatalf("getOrderStatus: unexpected status in JSON: %v", parsed["status"])
	}
}

func TestGetUserFills(t *testing.T) {
	client := &fakeQueriesClient{
		accountAddress: "0xConfigured",
		fills:          []hyperliquid.Fill{{Coin: "BTC", Oid: 1}},
	}

	_, out, err := getUserFillsHandler(client)(context.Background(), nil, GetUserFillsArgs{})
	if err != nil {
		t.Fatalf("handler unexpected error: %v", err)
	}
	if client.gotAddress != "0xConfigured" {
		t.Fatalf("expected default address, got %q", client.gotAddress)
	}
	if len(out.Data) != 1 || out.Data[0].Coin != "BTC" {
		t.Fatalf("getUserFills output = %+v", out)
	}
}

func TestGetUserFunding(t *testing.T) {
	client := &fakeQueriesClient{
		accountAddress: "0xConfigured",
		funding:        []hyperliquid.UserFundingHistory{{Hash: "0xabc"}},
	}

	_, out, err := getUserFundingHandler(client)(context.Background(), nil, GetUserFundingArgs{StartTime: 1700000000000})
	if err != nil {
		t.Fatalf("handler unexpected error: %v", err)
	}
	if len(out.Data) != 1 || out.Data[0].Hash != "0xabc" {
		t.Fatalf("getUserFunding output = %+v", out)
	}
}
