// internal/hlclient/vault.go
package hlclient

import (
	"context"
	"encoding/json"
	"fmt"
)

// VaultDetails calls the "vaultDetails" /info endpoint. go-hyperliquid does
// not expose this as a typed method, so the response is returned as raw
// JSON for the caller to pass through or inspect.
func (c *Client) VaultDetails(ctx context.Context, vaultAddress string) (json.RawMessage, error) {
	var raw json.RawMessage
	payload := map[string]any{
		"type":         "vaultDetails",
		"vaultAddress": vaultAddress,
	}
	if err := c.postInfo(ctx, payload, &raw); err != nil {
		return nil, fmt.Errorf("failed to fetch vault details for %s: %w", vaultAddress, err)
	}
	return raw, nil
}

// VaultPerformance calls "vaultDetails" with a time range, matching the
// original Python server's vault_details(vault_address, start_time, end_time)
// overload (the underlying Hyperliquid endpoint is the same; supplying a
// time range returns performance/PnL history instead of static details).
func (c *Client) VaultPerformance(
	ctx context.Context,
	vaultAddress string,
	startTime int64,
	endTime *int64,
) (json.RawMessage, error) {
	var raw json.RawMessage
	payload := map[string]any{
		"type":         "vaultDetails",
		"vaultAddress": vaultAddress,
		"startTime":    startTime,
	}
	if endTime != nil {
		payload["endTime"] = *endTime
	}
	if err := c.postInfo(ctx, payload, &raw); err != nil {
		return nil, fmt.Errorf("failed to fetch vault performance for %s: %w", vaultAddress, err)
	}
	return raw, nil
}
