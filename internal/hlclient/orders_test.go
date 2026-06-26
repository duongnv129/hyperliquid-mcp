// internal/hlclient/orders_test.go
package hlclient

import (
	"context"
	"net/http"
	"testing"

	hyperliquid "github.com/sonirico/go-hyperliquid"
)

// actionType extracts the "type" field from body["action"] for exchange requests.
// Exchange POST bodies have no top-level "type"; the type lives in body["action"]["type"].
func actionType(body map[string]any) string {
	action, ok := body["action"].(map[string]any)
	if !ok {
		return ""
	}
	t, _ := action["type"].(string)
	return t
}

func TestPlaceOrder(t *testing.T) {
	var captured map[string]any
	srv := testServer(t, func(w http.ResponseWriter, body map[string]any) {
		w.Header().Set("Content-Type", "application/json")
		if actionType(body) == "order" {
			captured = body["action"].(map[string]any)
		}
		_, _ = w.Write([]byte(`{"status":"ok","response":{"type":"order","data":{"statuses":[{"resting":{"oid":1,"status":"resting"}}]}}}`))
	})
	cfg := testConfig(srv.URL)
	client, err := newForBaseURL(context.Background(), cfg, srv.URL)
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	status, err := client.PlaceOrder(context.Background(), hyperliquid.CreateOrderRequest{
		Coin:  "BTC",
		IsBuy: true,
		Price: 65000,
		Size:  0.1,
		OrderType: hyperliquid.OrderType{
			Limit: &hyperliquid.LimitOrderType{Tif: hyperliquid.TifGtc},
		},
	})
	if err != nil {
		t.Fatalf("PlaceOrder() unexpected error: %v", err)
	}
	if status.Resting == nil || status.Resting.Oid != 1 {
		t.Fatalf("PlaceOrder() status = %+v, want a resting order with oid 1", status)
	}
	if captured == nil {
		t.Fatal("expected the order action to reach the test server")
	}
}

func TestPlaceBracketOrder(t *testing.T) {
	var orderCount int
	srv := testServer(t, func(w http.ResponseWriter, body map[string]any) {
		w.Header().Set("Content-Type", "application/json")
		if actionType(body) == "order" {
			action := body["action"].(map[string]any)
			orders, _ := action["orders"].([]any)
			orderCount = len(orders)
		}
		_, _ = w.Write([]byte(`{"status":"ok","response":{"type":"order","data":{"statuses":[
			{"resting":{"oid":1,"status":"resting"}},
			{"resting":{"oid":2,"status":"resting"}},
			{"resting":{"oid":3,"status":"resting"}}
		]}}}`))
	})
	cfg := testConfig(srv.URL)
	client, err := newForBaseURL(context.Background(), cfg, srv.URL)
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	resp, err := client.PlaceBracketOrder(context.Background(), BracketOrderParams{
		Coin:            "SOL",
		IsBuy:           true,
		Size:            4.12,
		EntryPrice:      218.0,
		TakeProfitPrice: 219.5,
		StopLossPrice:   216.8,
	})
	if err != nil {
		t.Fatalf("PlaceBracketOrder() unexpected error: %v", err)
	}
	if len(resp.Data.Statuses) != 3 {
		t.Fatalf("PlaceBracketOrder() returned %d statuses, want 3", len(resp.Data.Statuses))
	}
	if orderCount != 3 {
		t.Fatalf("expected a single bulk request with 3 orders, server saw %d", orderCount)
	}
}

func TestCancelOrder(t *testing.T) {
	srv := testServer(t, func(w http.ResponseWriter, body map[string]any) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","response":{"type":"cancel","data":{"statuses":["success"]}}}`))
	})
	cfg := testConfig(srv.URL)
	client, err := newForBaseURL(context.Background(), cfg, srv.URL)
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	resp, err := client.CancelOrder(context.Background(), "BTC", 1)
	if err != nil {
		t.Fatalf("CancelOrder() unexpected error: %v", err)
	}
	if !resp.Ok {
		t.Fatalf("CancelOrder() response not ok: %+v", resp)
	}
}

func TestCancelAllOrders_NoOpenOrders(t *testing.T) {
	srv := testServer(t, func(w http.ResponseWriter, body map[string]any) {
		w.Header().Set("Content-Type", "application/json")
		if body["type"] == "openOrders" {
			_, _ = w.Write([]byte(`[]`))
			return
		}
		t.Fatalf("unexpected request type %q when there are no open orders", body["type"])
	})
	cfg := testConfig(srv.URL)
	client, err := newForBaseURL(context.Background(), cfg, srv.URL)
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	cancelled, resp, err := client.CancelAllOrders(context.Background(), client.AccountAddress(), "")
	if err != nil {
		t.Fatalf("CancelAllOrders() unexpected error: %v", err)
	}
	if cancelled != 0 || resp != nil {
		t.Fatalf("CancelAllOrders() = (%d, %+v), want (0, nil)", cancelled, resp)
	}
}

func TestCancelAllOrders_WithOpenOrders(t *testing.T) {
	srv := testServer(t, func(w http.ResponseWriter, body map[string]any) {
		w.Header().Set("Content-Type", "application/json")
		// Exchange requests have no top-level "type"; they have body["action"]["type"]
		if body["type"] == "openOrders" {
			_, _ = w.Write([]byte(`[
				{"coin":"BTC","limitPx":"65000","oid":1,"origSz":"0.1","side":"B","sz":"0.1","timestamp":1700000000000},
				{"coin":"ETH","limitPx":"3000","oid":2,"origSz":"1","side":"A","sz":"1","timestamp":1700000000000}
			]`))
			return
		}
		if actionType(body) == "cancel" {
			_, _ = w.Write([]byte(`{"status":"ok","response":{"type":"cancel","data":{"statuses":["success","success"]}}}`))
			return
		}
		t.Fatalf("unexpected request type %q / action type %q", body["type"], actionType(body))
	})
	cfg := testConfig(srv.URL)
	client, err := newForBaseURL(context.Background(), cfg, srv.URL)
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	cancelled, resp, err := client.CancelAllOrders(context.Background(), client.AccountAddress(), "")
	if err != nil {
		t.Fatalf("CancelAllOrders() unexpected error: %v", err)
	}
	if cancelled != 2 {
		t.Fatalf("CancelAllOrders() cancelled = %d, want 2", cancelled)
	}
	if resp == nil || !resp.Ok {
		t.Fatalf("CancelAllOrders() resp = %+v, want ok", resp)
	}
}

func TestModifyOrder(t *testing.T) {
	srv := testServer(t, func(w http.ResponseWriter, body map[string]any) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","response":{"type":"modify","data":{"statuses":[{"resting":{"oid":1,"status":"resting"}}]}}}`))
	})
	cfg := testConfig(srv.URL)
	client, err := newForBaseURL(context.Background(), cfg, srv.URL)
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	oid := int64(1)
	status, err := client.ModifyOrder(context.Background(), hyperliquid.ModifyOrderRequest{
		Oid: &oid,
		Order: hyperliquid.CreateOrderRequest{
			Coin:      "BTC",
			IsBuy:     true,
			Price:     64000,
			Size:      0.1,
			OrderType: hyperliquid.OrderType{Limit: &hyperliquid.LimitOrderType{Tif: hyperliquid.TifGtc}},
		},
	})
	if err != nil {
		t.Fatalf("ModifyOrder() unexpected error: %v", err)
	}
	if status.Resting == nil || status.Resting.Oid != 1 {
		t.Fatalf("ModifyOrder() status = %+v", status)
	}
}
