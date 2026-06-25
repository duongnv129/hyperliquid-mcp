package hlclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/duongnv129/hyperliquid-mcp/internal/config"
)

// testServer returns an httptest.Server that answers the /info "meta" and
// "spotMeta" calls go-hyperliquid makes automatically on construction, plus
// whatever extra handler the caller supplies for other request types.
func testServer(t *testing.T, extra func(w http.ResponseWriter, body map[string]any)) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		switch body["type"] {
		case "meta":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"universe": [
					{"name": "BTC", "szDecimals": 5, "maxLeverage": 50, "marginTableId": 1, "onlyIsolated": false, "isDelisted": false},
					{"name": "ETH", "szDecimals": 4, "maxLeverage": 50, "marginTableId": 1, "onlyIsolated": false, "isDelisted": false},
					{"name": "SOL", "szDecimals": 2, "maxLeverage": 20, "marginTableId": 2, "onlyIsolated": false, "isDelisted": false}
				],
				"marginTables": [],
				"collateralToken": 0
			}`))
		case "spotMeta":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"universe": [], "tokens": []}`))
		default:
			if extra != nil {
				extra(w, body)
				return
			}
			t.Fatalf("unexpected request type %q", body["type"])
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// testPrivateKeyHex is a syntactically valid (64 hex chars), arbitrary test
// key — never used against a real network in these tests.
const testPrivateKeyHex = "0x1234567890123456789012345678901234567890123456789012345678901234"

func testConfig(_ string) config.Config {
	return config.Config{PrivateKeyHex: testPrivateKeyHex}
}

func TestNew_Success(t *testing.T) {
	srv := testServer(t, nil)
	cfg := testConfig(srv.URL)

	client, err := newForBaseURL(context.Background(), cfg, srv.URL)
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}
	if client.AccountAddress() == "" {
		t.Fatal("expected a derived account address, got empty string")
	}
}

func TestNew_InvalidPrivateKey(t *testing.T) {
	cfg := config.Config{PrivateKeyHex: "0xnothex000000000000000000000000000000000000000000000000000000000"}

	_, err := newForBaseURL(context.Background(), cfg, "http://example.invalid")
	if err == nil {
		t.Fatal("expected an error for an invalid private key, got nil")
	}
}

func TestNew_RecoversMetaFetchPanic(t *testing.T) {
	// A server that returns 500 for every request forces go-hyperliquid's
	// NewInfo to panic while fetching meta; New must recover and return an error.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	cfg := testConfig(srv.URL)
	_, err := newForBaseURL(context.Background(), cfg, srv.URL)
	if err == nil {
		t.Fatal("expected New to convert the underlying panic into an error")
	}
}

func TestAssetIndexAndCoinForAsset(t *testing.T) {
	srv := testServer(t, nil)
	cfg := testConfig(srv.URL)
	client, err := newForBaseURL(context.Background(), cfg, srv.URL)
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	idx, ok := client.AssetIndex("SOL")
	if !ok || idx != 2 {
		t.Fatalf("AssetIndex(SOL) = (%d, %v), want (2, true)", idx, ok)
	}

	coin, err := client.CoinForAsset(context.Background(), 1)
	if err != nil || coin != "ETH" {
		t.Fatalf("CoinForAsset(1) = (%q, %v), want (ETH, nil)", coin, err)
	}

	if _, err := client.CoinForAsset(context.Background(), 99); err == nil {
		t.Fatal("expected an out-of-range asset index to error")
	}
}
