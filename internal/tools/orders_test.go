// internal/tools/orders_test.go
package tools

import (
	"context"
	"errors"
	"testing"

	hyperliquid "github.com/sonirico/go-hyperliquid"

	"github.com/duongnv129/hyperliquid-mcp/internal/hlclient"
)

type fakeOrdersClient struct {
	accountAddress string
	coinForAsset   map[int]string
	assetIndex     map[string]int

	placeOrderReq    hyperliquid.CreateOrderRequest
	placeOrderResult hyperliquid.OrderStatus
	placeOrderErr    error

	bracketParams hlclient.BracketOrderParams
	bracketResp   *hyperliquid.APIResponse[hyperliquid.OrderResponse]
	bracketErr    error

	cancelCoin string
	cancelOid  int64
	cancelResp *hyperliquid.APIResponse[hyperliquid.CancelOrderResponse]
	cancelErr  error

	cancelAllUser      string
	cancelAllDex       string
	cancelAllCancelled int
	cancelAllResp      *hyperliquid.APIResponse[hyperliquid.CancelOrderResponse]
	cancelAllErr       error

	modifyReq    hyperliquid.ModifyOrderRequest
	modifyResult hyperliquid.OrderStatus
	modifyErr    error
}

func (f *fakeOrdersClient) AccountAddress() string { return f.accountAddress }

func (f *fakeOrdersClient) AssetIndex(coin string) (int, bool) {
	idx, ok := f.assetIndex[coin]
	return idx, ok
}

func (f *fakeOrdersClient) CoinForAsset(ctx context.Context, asset int) (string, error) {
	coin, ok := f.coinForAsset[asset]
	if !ok {
		return "", errors.New("unknown asset")
	}
	return coin, nil
}

func (f *fakeOrdersClient) PlaceOrder(ctx context.Context, req hyperliquid.CreateOrderRequest) (hyperliquid.OrderStatus, error) {
	f.placeOrderReq = req
	return f.placeOrderResult, f.placeOrderErr
}

func (f *fakeOrdersClient) PlaceBracketOrder(ctx context.Context, p hlclient.BracketOrderParams) (*hyperliquid.APIResponse[hyperliquid.OrderResponse], error) {
	f.bracketParams = p
	return f.bracketResp, f.bracketErr
}

func (f *fakeOrdersClient) CancelOrder(ctx context.Context, coin string, oid int64) (*hyperliquid.APIResponse[hyperliquid.CancelOrderResponse], error) {
	f.cancelCoin, f.cancelOid = coin, oid
	return f.cancelResp, f.cancelErr
}

func (f *fakeOrdersClient) CancelAllOrders(ctx context.Context, user, dex string) (int, *hyperliquid.APIResponse[hyperliquid.CancelOrderResponse], error) {
	f.cancelAllUser, f.cancelAllDex = user, dex
	return f.cancelAllCancelled, f.cancelAllResp, f.cancelAllErr
}

func (f *fakeOrdersClient) ModifyOrder(ctx context.Context, req hyperliquid.ModifyOrderRequest) (hyperliquid.OrderStatus, error) {
	f.modifyReq = req
	return f.modifyResult, f.modifyErr
}

func TestPlaceOrderHandler_LimitOrder(t *testing.T) {
	oid := int64(1)
	client := &fakeOrdersClient{
		coinForAsset:     map[int]string{0: "BTC"},
		placeOrderResult: hyperliquid.OrderStatus{Resting: &hyperliquid.OrderStatusResting{Oid: oid, Status: "resting"}},
	}

	_, out, err := placeOrderHandler(client)(context.Background(), nil, PlaceOrderArgs{
		Asset: 0, IsBuy: true, Size: "0.1", Price: "65000",
	})
	if err != nil {
		t.Fatalf("handler unexpected error: %v", err)
	}
	if client.placeOrderReq.Coin != "BTC" || client.placeOrderReq.OrderType.Limit == nil {
		t.Fatalf("expected a resolved BTC limit order, got %+v", client.placeOrderReq)
	}
	if out.Data.Resting == nil || out.Data.Resting.Oid != 1 {
		t.Fatalf("placeOrder output = %+v", out)
	}
}

func TestPlaceOrderHandler_TriggerOrder(t *testing.T) {
	client := &fakeOrdersClient{coinForAsset: map[int]string{5: "SOL"}}

	_, _, err := placeOrderHandler(client)(context.Background(), nil, PlaceOrderArgs{
		Asset: 5, IsBuy: false, Size: "1", Price: "220", TriggerPx: "219.5", Tpsl: "tp",
	})
	if err != nil {
		t.Fatalf("handler unexpected error: %v", err)
	}
	if client.placeOrderReq.OrderType.Trigger == nil || client.placeOrderReq.OrderType.Trigger.Tpsl != hyperliquid.TakeProfit {
		t.Fatalf("expected a take-profit trigger order, got %+v", client.placeOrderReq.OrderType)
	}
}

func TestPlaceOrderHandler_UnknownAsset(t *testing.T) {
	client := &fakeOrdersClient{coinForAsset: map[int]string{}}

	_, _, err := placeOrderHandler(client)(context.Background(), nil, PlaceOrderArgs{Asset: 99, IsBuy: true, Size: "0.1"})
	if err == nil {
		t.Fatal("expected an error for an unresolvable asset index")
	}
}

func TestPlaceOrderHandler_InvalidSize(t *testing.T) {
	client := &fakeOrdersClient{coinForAsset: map[int]string{0: "BTC"}}

	_, _, err := placeOrderHandler(client)(context.Background(), nil, PlaceOrderArgs{Asset: 0, IsBuy: true, Size: "not-a-number"})
	if err == nil {
		t.Fatal("expected an error for a non-numeric size")
	}
}

func TestPlaceBracketOrderHandler(t *testing.T) {
	client := &fakeOrdersClient{
		coinForAsset: map[int]string{5: "SOL"},
		bracketResp: &hyperliquid.APIResponse[hyperliquid.OrderResponse]{
			Ok: true,
			Data: hyperliquid.OrderResponse{Statuses: []hyperliquid.OrderStatus{
				{Resting: &hyperliquid.OrderStatusResting{Oid: 1}},
				{Resting: &hyperliquid.OrderStatusResting{Oid: 2}},
				{Resting: &hyperliquid.OrderStatusResting{Oid: 3}},
			}},
		},
	}

	_, out, err := placeBracketOrderHandler(client)(context.Background(), nil, PlaceBracketOrderArgs{
		Asset: 5, IsBuy: true, Size: "4.12", EntryPrice: "218.0", TakeProfitPrice: "219.5", StopLossPrice: "216.8",
	})
	if err != nil {
		t.Fatalf("handler unexpected error: %v", err)
	}
	if client.bracketParams.Coin != "SOL" || client.bracketParams.TakeProfitPrice != 219.5 {
		t.Fatalf("bracketParams = %+v", client.bracketParams)
	}
	if len(out.Data.Statuses) != 3 {
		t.Fatalf("placeBracketOrder output = %+v", out)
	}
}

func TestCancelOrderHandler(t *testing.T) {
	client := &fakeOrdersClient{
		cancelResp: &hyperliquid.APIResponse[hyperliquid.CancelOrderResponse]{Ok: true},
	}

	_, out, err := cancelOrderHandler(client)(context.Background(), nil, CancelOrderArgs{Coin: "BTC", Oid: 1})
	if err != nil {
		t.Fatalf("handler unexpected error: %v", err)
	}
	if client.cancelCoin != "BTC" || client.cancelOid != 1 {
		t.Fatalf("cancel args = (%q, %d)", client.cancelCoin, client.cancelOid)
	}
	if out.CancelledOrder.Coin != "BTC" || out.CancelledOrder.OrderID != 1 {
		t.Fatalf("cancelOrder output = %+v", out)
	}
}

func TestCancelAllOrdersHandler_NoOrders(t *testing.T) {
	client := &fakeOrdersClient{accountAddress: "0xConfigured", cancelAllCancelled: 0, cancelAllResp: nil}

	_, out, err := cancelAllOrdersHandler(client)(context.Background(), nil, CancelAllOrdersArgs{})
	if err != nil {
		t.Fatalf("handler unexpected error: %v", err)
	}
	if client.cancelAllUser != "0xConfigured" {
		t.Fatalf("expected default address, got %q", client.cancelAllUser)
	}
	if out.CancelledCount != 0 {
		t.Fatalf("cancelAllOrders output = %+v, want CancelledCount 0", out)
	}
}

func TestCancelAllOrdersHandler_WithOrders(t *testing.T) {
	client := &fakeOrdersClient{
		accountAddress:     "0xConfigured",
		cancelAllCancelled: 2,
		cancelAllResp:      &hyperliquid.APIResponse[hyperliquid.CancelOrderResponse]{Ok: true},
	}

	_, out, err := cancelAllOrdersHandler(client)(context.Background(), nil, CancelAllOrdersArgs{})
	if err != nil {
		t.Fatalf("handler unexpected error: %v", err)
	}
	if out.CancelledCount != 2 {
		t.Fatalf("cancelAllOrders output = %+v, want CancelledCount 2", out)
	}
}

func TestModifyOrderHandler(t *testing.T) {
	client := &fakeOrdersClient{
		modifyResult: hyperliquid.OrderStatus{Resting: &hyperliquid.OrderStatusResting{Oid: 1, Status: "resting"}},
	}

	_, out, err := modifyOrderHandler(client)(context.Background(), nil, ModifyOrderArgs{
		Oid: 1, Coin: "BTC", IsBuy: true, Size: "0.2", Price: "64000",
	})
	if err != nil {
		t.Fatalf("handler unexpected error: %v", err)
	}
	if client.modifyReq.Oid == nil || *client.modifyReq.Oid != 1 || client.modifyReq.Order.Coin != "BTC" {
		t.Fatalf("modifyReq = %+v", client.modifyReq)
	}
	if out.Data.Resting == nil || out.Data.Resting.Oid != 1 {
		t.Fatalf("modifyOrder output = %+v", out)
	}
}

func TestPlaceTwapOrderHandler_Stub(t *testing.T) {
	client := &fakeOrdersClient{}

	_, out, err := placeTwapOrderHandler(client)(context.Background(), nil, PlaceTwapOrderArgs{})
	if err != nil {
		t.Fatalf("stub handler should not error: %v", err)
	}
	if out.Message == "" {
		t.Fatal("expected a non-empty 'not implemented' message")
	}
}

func TestCancelTwapOrderHandler_Stub(t *testing.T) {
	client := &fakeOrdersClient{}

	_, out, err := cancelTwapOrderHandler(client)(context.Background(), nil, CancelTwapOrderArgs{})
	if err != nil {
		t.Fatalf("stub handler should not error: %v", err)
	}
	if out.Message == "" {
		t.Fatal("expected a non-empty 'not implemented' message")
	}
}
