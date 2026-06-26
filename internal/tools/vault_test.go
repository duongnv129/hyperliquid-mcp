// internal/tools/vault_test.go
package tools

import (
	"context"
	"encoding/json"
	"testing"
)

type fakeVaultClient struct {
	details         json.RawMessage
	performance     json.RawMessage
	gotVaultAddress string
	gotStartTime    int64
	err             error
}

func (f *fakeVaultClient) VaultDetails(ctx context.Context, vaultAddress string) (json.RawMessage, error) {
	f.gotVaultAddress = vaultAddress
	return f.details, f.err
}

func (f *fakeVaultClient) VaultPerformance(ctx context.Context, vaultAddress string, startTime int64, endTime *int64) (json.RawMessage, error) {
	f.gotVaultAddress, f.gotStartTime = vaultAddress, startTime
	return f.performance, f.err
}

func TestVaultDetailsHandler(t *testing.T) {
	client := &fakeVaultClient{details: json.RawMessage(`{"name":"Test Vault"}`)}

	_, out, err := vaultDetailsHandler(client)(context.Background(), nil, VaultDetailsArgs{VaultAddress: "0xVault"})
	if err != nil {
		t.Fatalf("handler unexpected error: %v", err)
	}
	if client.gotVaultAddress != "0xVault" {
		t.Fatalf("expected vault address to be passed through, got %q", client.gotVaultAddress)
	}
	if string(out.Data) != `{"name":"Test Vault"}` {
		t.Fatalf("vaultDetails output = %s", out.Data)
	}
}

func TestVaultPerformanceHandler(t *testing.T) {
	client := &fakeVaultClient{performance: json.RawMessage(`{"vaultAddress":"0xVault","pnls":[]}`)}

	_, out, err := vaultPerformanceHandler(client)(context.Background(), nil, VaultPerformanceArgs{
		VaultAddress: "0xVault", StartTime: 1700000000000,
	})
	if err != nil {
		t.Fatalf("handler unexpected error: %v", err)
	}
	if client.gotStartTime != 1700000000000 {
		t.Fatalf("expected startTime to be passed through, got %d", client.gotStartTime)
	}
	if string(out.Data) != `{"vaultAddress":"0xVault","pnls":[]}` {
		t.Fatalf("vaultPerformance output = %s", out.Data)
	}
}
