// internal/tools/account_test.go
package tools

import (
	"context"
	"errors"
	"testing"

	hyperliquid "github.com/sonirico/go-hyperliquid"
)

type fakeAccountClient struct {
	state          *hyperliquid.UserState
	err            error
	accountAddress string
	gotAddress     string
	gotDex         string
}

func (f *fakeAccountClient) UserState(ctx context.Context, address, dex string) (*hyperliquid.UserState, error) {
	f.gotAddress = address
	f.gotDex = dex
	if f.err != nil {
		return nil, f.err
	}
	return f.state, nil
}

func (f *fakeAccountClient) AccountAddress() string { return f.accountAddress }

func sampleUserState() *hyperliquid.UserState {
	return &hyperliquid.UserState{
		AssetPositions: []hyperliquid.AssetPosition{
			{Type: "oneWay", Position: hyperliquid.Position{Coin: "BTC", Szi: "0.1"}},
		},
		MarginSummary: hyperliquid.MarginSummary{
			AccountValue:    "1000",
			TotalMarginUsed: "100",
		},
		Withdrawable: "900",
	}
}

func TestGetAccountInfo(t *testing.T) {
	client := &fakeAccountClient{accountAddress: "0xConfigured", state: sampleUserState()}

	_, out, err := getAccountInfoHandler(client)(context.Background(), nil, GetAccountInfoArgs{})
	if err != nil {
		t.Fatalf("handler unexpected error: %v", err)
	}
	if client.gotAddress != "0xConfigured" {
		t.Fatalf("expected default address to be used, got %q", client.gotAddress)
	}
	if out.Data == nil {
		t.Fatal("expected non-empty account data in output")
	}
}

func TestGetAccountInfo_ExplicitAddress(t *testing.T) {
	client := &fakeAccountClient{accountAddress: "0xConfigured", state: sampleUserState()}

	_, _, err := getAccountInfoHandler(client)(context.Background(), nil, GetAccountInfoArgs{UserAddress: "0xOther", Dex: "spot"})
	if err != nil {
		t.Fatalf("handler unexpected error: %v", err)
	}
	if client.gotAddress != "0xOther" || client.gotDex != "spot" {
		t.Fatalf("expected explicit address/dex to be passed through, got (%q, %q)", client.gotAddress, client.gotDex)
	}
}

func TestGetAccountInfo_Error(t *testing.T) {
	client := &fakeAccountClient{accountAddress: "0xConfigured", err: errors.New("network down")}

	_, _, err := getAccountInfoHandler(client)(context.Background(), nil, GetAccountInfoArgs{})
	if err == nil {
		t.Fatal("expected handler to surface the client error")
	}
}

func TestGetPositions(t *testing.T) {
	client := &fakeAccountClient{accountAddress: "0xConfigured", state: sampleUserState()}

	_, out, err := getPositionsHandler(client)(context.Background(), nil, GetPositionsArgs{})
	if err != nil {
		t.Fatalf("handler unexpected error: %v", err)
	}
	if len(out.Data) != 1 || out.Data[0].Position.Coin != "BTC" {
		t.Fatalf("getPositions output = %+v, want one BTC position", out)
	}
}

func TestGetBalance(t *testing.T) {
	client := &fakeAccountClient{accountAddress: "0xConfigured", state: sampleUserState()}

	_, out, err := getBalanceHandler(client)(context.Background(), nil, GetBalanceArgs{})
	if err != nil {
		t.Fatalf("handler unexpected error: %v", err)
	}
	if out.Data.AccountValue != "1000" || out.Data.Withdrawable != "900" {
		t.Fatalf("getBalance output = %+v", out)
	}
}
