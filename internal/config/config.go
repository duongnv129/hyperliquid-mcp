package config

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
)

const (
	mainnetBaseURL = "https://api.hyperliquid.xyz"
	testnetBaseURL = "https://api.hyperliquid-testnet.xyz"
)

var privateKeyPattern = regexp.MustCompile(`^0x[0-9a-fA-F]{64}$`)

// Config holds Hyperliquid MCP server configuration loaded from environment
// variables, matching the original Python server's mcp.json `env` block.
type Config struct {
	PrivateKeyHex  string
	AccountAddress string
	VaultAddress   string
	Testnet        bool
}

// Load reads and validates configuration from environment variables.
func Load() (Config, error) {
	pk := strings.TrimSpace(os.Getenv("HYPERLIQUID_PRIVATE_KEY"))
	if pk == "" {
		return Config{}, errors.New("HYPERLIQUID_PRIVATE_KEY is required")
	}
	if !strings.HasPrefix(pk, "0x") {
		pk = "0x" + pk
	}
	if !privateKeyPattern.MatchString(pk) {
		return Config{}, fmt.Errorf("HYPERLIQUID_PRIVATE_KEY must be a 64-character hex string")
	}

	testnet := strings.EqualFold(strings.TrimSpace(os.Getenv("HYPERLIQUID_TESTNET")), "true")

	return Config{
		PrivateKeyHex:  pk,
		AccountAddress: strings.TrimSpace(os.Getenv("HYPERLIQUID_ACCOUNT_ADDRESS")),
		VaultAddress:   strings.TrimSpace(os.Getenv("HYPERLIQUID_VAULT_ADDRESS")),
		Testnet:        testnet,
	}, nil
}

// BaseURL returns the Hyperliquid REST API base URL for this configuration.
func (c Config) BaseURL() string {
	if c.Testnet {
		return testnetBaseURL
	}
	return mainnetBaseURL
}
