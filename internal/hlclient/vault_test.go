// internal/hlclient/vault_test.go
package hlclient

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestVaultDetailsAndPerformance(t *testing.T) {
	srv := testServer(t, func(w http.ResponseWriter, body map[string]any) {
		w.Header().Set("Content-Type", "application/json")
		switch body["type"] {
		case "vaultDetails":
			if body["vaultAddress"] != "0xVault" {
				t.Fatalf("vaultDetails request missing vaultAddress: %+v", body)
			}
			_, _ = w.Write([]byte(`{"name":"Test Vault","vaultAddress":"0xVault","leader":"0xLeader"}`))
		default:
			t.Fatalf("unexpected request type %q", body["type"])
		}
	})
	cfg := testConfig(srv.URL)
	client, err := newForBaseURL(context.Background(), cfg, srv.URL)
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	raw, err := client.VaultDetails(context.Background(), "0xVault")
	if err != nil {
		t.Fatalf("VaultDetails() unexpected error: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("VaultDetails() returned invalid JSON: %v", err)
	}
	if decoded["name"] != "Test Vault" {
		t.Fatalf("VaultDetails() = %+v, want name=Test Vault", decoded)
	}
}

func TestVaultPerformance(t *testing.T) {
	srv := testServer(t, func(w http.ResponseWriter, body map[string]any) {
		w.Header().Set("Content-Type", "application/json")
		switch body["type"] {
		case "vaultDetails":
			if body["startTime"] == nil {
				t.Fatalf("vaultPerformance-style request missing startTime: %+v", body)
			}
			_, _ = w.Write([]byte(`{"vaultAddress":"0xVault","pnls":[]}`))
		default:
			t.Fatalf("unexpected request type %q", body["type"])
		}
	})
	cfg := testConfig(srv.URL)
	client, err := newForBaseURL(context.Background(), cfg, srv.URL)
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	raw, err := client.VaultPerformance(context.Background(), "0xVault", 1700000000000, nil)
	if err != nil {
		t.Fatalf("VaultPerformance() unexpected error: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("VaultPerformance() returned invalid JSON: %v", err)
	}
	if decoded["vaultAddress"] != "0xVault" {
		t.Fatalf("VaultPerformance() = %+v", decoded)
	}
}
