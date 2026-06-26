package config

import "testing"

func TestLoad(t *testing.T) {
	cases := []struct {
		name    string
		env     map[string]string
		want    Config
		wantErr string
	}{
		{
			name: "valid mainnet config",
			env: map[string]string{
				"HYPERLIQUID_PRIVATE_KEY": "0x1234567890123456789012345678901234567890123456789012345678901234",
			},
			want: Config{
				PrivateKeyHex: "0x1234567890123456789012345678901234567890123456789012345678901234",
				Testnet:       false,
			},
		},
		{
			name: "testnet flag true is case-insensitive",
			env: map[string]string{
				"HYPERLIQUID_PRIVATE_KEY": "0x1234567890123456789012345678901234567890123456789012345678901234",
				"HYPERLIQUID_TESTNET":     "TRUE",
			},
			want: Config{
				PrivateKeyHex: "0x1234567890123456789012345678901234567890123456789012345678901234",
				Testnet:       true,
			},
		},
		{
			name: "optional addresses are passed through and trimmed",
			env: map[string]string{
				"HYPERLIQUID_PRIVATE_KEY":     "0x1234567890123456789012345678901234567890123456789012345678901234",
				"HYPERLIQUID_ACCOUNT_ADDRESS": "  0xAccount  ",
				"HYPERLIQUID_VAULT_ADDRESS":   "  0xVault  ",
			},
			want: Config{
				PrivateKeyHex:  "0x1234567890123456789012345678901234567890123456789012345678901234",
				AccountAddress: "0xAccount",
				VaultAddress:   "0xVault",
				Testnet:        false,
			},
		},
		{
			name:    "missing private key is an error",
			env:     map[string]string{},
			wantErr: "HYPERLIQUID_PRIVATE_KEY is required",
		},
		{
			name: "malformed private key is an error",
			env: map[string]string{
				"HYPERLIQUID_PRIVATE_KEY": "not-a-key",
			},
			wantErr: "HYPERLIQUID_PRIVATE_KEY must be a 0x-prefixed 64-character hex string",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, key := range []string{
				"HYPERLIQUID_PRIVATE_KEY",
				"HYPERLIQUID_ACCOUNT_ADDRESS",
				"HYPERLIQUID_VAULT_ADDRESS",
				"HYPERLIQUID_TESTNET",
			} {
				t.Setenv(key, "")
			}
			for k, v := range tc.env {
				t.Setenv(k, v)
			}

			got, err := Load()

			if tc.wantErr != "" {
				if err == nil || err.Error() != tc.wantErr {
					t.Fatalf("Load() error = %v, want %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Load() unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("Load() = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestBaseURL(t *testing.T) {
	if (Config{Testnet: false}).BaseURL() != "https://api.hyperliquid.xyz" {
		t.Fatal("mainnet BaseURL mismatch")
	}
	if (Config{Testnet: true}).BaseURL() != "https://api.hyperliquid-testnet.xyz" {
		t.Fatal("testnet BaseURL mismatch")
	}
}
