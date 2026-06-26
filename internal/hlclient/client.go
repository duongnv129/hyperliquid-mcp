package hlclient

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	hyperliquid "github.com/sonirico/go-hyperliquid"

	"github.com/duongnv129/hyperliquid-mcp/internal/config"
)

// Client wraps a sonirico/go-hyperliquid Exchange/Info pair with the
// additional operations the MCP tools need (bracket orders, cancel-all,
// vault queries, server time) that the underlying SDK does not expose
// directly.
type Client struct {
	exchange       *hyperliquid.Exchange
	info           *hyperliquid.Info
	httpClient     *http.Client
	baseURL        string
	accountAddress string
}

// New constructs a Client from configuration, deriving the trading account
// address from the private key when HYPERLIQUID_ACCOUNT_ADDRESS is unset.
func New(ctx context.Context, cfg config.Config) (*Client, error) {
	return newForBaseURL(ctx, cfg, cfg.BaseURL())
}

// newForBaseURL is the testable constructor: it takes the base URL
// explicitly so tests can point it at an httptest.Server.
func newForBaseURL(ctx context.Context, cfg config.Config, baseURL string) (client *Client, err error) {
	defer func() {
		if r := recover(); r != nil {
			client = nil
			err = fmt.Errorf("failed to initialize hyperliquid client: %v", r)
		}
	}()

	privateKey, perr := crypto.HexToECDSA(strings.TrimPrefix(cfg.PrivateKeyHex, "0x"))
	if perr != nil {
		return nil, fmt.Errorf("invalid private key: %w", perr)
	}

	pub, ok := privateKey.Public().(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("invalid private key: could not derive public key")
	}
	derivedAddress := crypto.PubkeyToAddress(*pub).Hex()

	accountAddress := cfg.AccountAddress
	if accountAddress == "" {
		accountAddress = derivedAddress
	}

	exchange := hyperliquid.NewExchange(
		ctx,
		privateKey,
		baseURL,
		nil, // Meta fetched automatically
		cfg.VaultAddress,
		accountAddress,
		nil, // SpotMeta fetched automatically
		nil, // PerpDexs fetched automatically
	)

	return &Client{
		exchange:       exchange,
		info:           exchange.Info(),
		httpClient:     &http.Client{Timeout: 30 * time.Second},
		baseURL:        baseURL,
		accountAddress: accountAddress,
	}, nil
}

// AccountAddress returns the trading account address (configured, or
// derived from the private key if HYPERLIQUID_ACCOUNT_ADDRESS was unset).
func (c *Client) AccountAddress() string {
	return c.accountAddress
}

// ServerTimeMillis returns the current time as Unix milliseconds, matching
// the original Python server's get_server_time (a local clock read, not an
// API call).
func (c *Client) ServerTimeMillis() int64 {
	return time.Now().UnixMilli()
}

// AssetIndex resolves a coin symbol (e.g. "SOL") to its asset index, using
// the metadata go-hyperliquid already cached at construction time.
func (c *Client) AssetIndex(coin string) (int, bool) {
	return c.info.CoinToAsset(coin)
}

// CoinForAsset resolves an asset index back to its coin symbol by looking
// it up in the exchange's universe metadata.
func (c *Client) CoinForAsset(ctx context.Context, asset int) (string, error) {
	meta, err := c.info.Meta(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to fetch meta: %w", err)
	}
	if asset < 0 || asset >= len(meta.Universe) {
		return "", fmt.Errorf("asset index %d out of range (0-%d)", asset, len(meta.Universe)-1)
	}
	return meta.Universe[asset].Name, nil
}

// postInfo issues a raw POST to {baseURL}/info for endpoints go-hyperliquid
// does not expose as a typed method (recentTrades, vaultDetails,
// vaultPerformance), decoding the JSON response into out.
func (c *Client) postInfo(ctx context.Context, payload map[string]any, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/info", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request %q failed: %w", payload["type"], err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response for %q: %w", payload["type"], err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("request %q failed with status %d: %s", payload["type"], resp.StatusCode, string(respBody))
	}

	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("failed to decode response for %q: %w", payload["type"], err)
	}
	return nil
}
