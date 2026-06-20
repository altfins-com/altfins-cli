package cmd

import (
	"encoding/json"
	"testing"
)

func TestPortfolioShowDryRunBuildsToolCall(t *testing.T) {
	isolatedConfig(t)
	t.Setenv("ALTFINS_API_KEY", "TEST-DUMMY-KEY")
	out, err := runCLI(t, nil, "--dry-run", "portfolio", "show", "-o", "json")
	if err != nil {
		t.Fatalf("portfolio show --dry-run failed: %v (out=%s)", err, out)
	}
	var preview mcpPreview
	if err := json.Unmarshal([]byte(out), &preview); err != nil {
		t.Fatalf("parse dry-run preview: %v (out=%s)", err, out)
	}
	if preview.Body.Params.Name != "getUserPortfolio" {
		t.Errorf("tool name = %q, want getUserPortfolio", preview.Body.Params.Name)
	}
	if len(preview.Body.Params.Arguments) != 0 {
		t.Errorf("getUserPortfolio takes no arguments, got %v", preview.Body.Params.Arguments)
	}
	if preview.Headers["X-Api-Key"] != "redacted" {
		t.Errorf("X-Api-Key header = %q, want redacted", preview.Headers["X-Api-Key"])
	}
}

func TestPortfolioRedaction(t *testing.T) {
	const addr = "0xABCDEF0123456789"
	const secret = "sk-supersecretapikey"
	sample := func() []map[string]any {
		return []map[string]any{
			{"symbol": "ETH", "walletAddress": addr, "balance": "12.5", "exchange": "MetaMask", "apiKey": secret},
		}
	}

	// Default: wallet address + secret redacted, balance/symbol preserved.
	def := redactPortfolio(sample(), false, false).([]map[string]any)[0]
	if def["walletAddress"] == addr {
		t.Error("wallet address must be redacted by default")
	}
	if def["walletAddress"] == "" {
		t.Error("redacted wallet address should be masked, not removed")
	}
	if def["apiKey"] == secret {
		t.Error("apiKey must be masked by default")
	}
	if def["balance"] != "12.5" {
		t.Errorf("balance should be preserved by default, got %v", def["balance"])
	}
	if def["symbol"] != "ETH" {
		t.Errorf("non-sensitive symbol must be preserved, got %v", def["symbol"])
	}

	// --full: wallet address revealed, but secrets stay masked (CRITICAL).
	full := redactPortfolio(sample(), true, false).([]map[string]any)[0]
	if full["walletAddress"] != addr {
		t.Errorf("--full must reveal the wallet address, got %v", full["walletAddress"])
	}
	if full["apiKey"] == secret {
		t.Error("--full must NOT reveal secrets/API keys — security regression")
	}

	// --mask-balances: balance hidden.
	masked := redactPortfolio(sample(), false, true).([]map[string]any)[0]
	if masked["balance"] != "***" {
		t.Errorf("--mask-balances must hide the balance, got %v", masked["balance"])
	}
}

func TestPortfolioRedactionRecurses(t *testing.T) {
	const addr = "bc1qexampleaddress0000"
	nested := map[string]any{
		"exchanges": []any{
			map[string]any{"name": "Binance", "wallets": []any{
				map[string]any{"address": addr, "privateKey": "PRIV", "balance": "3.0"},
			}},
		},
	}
	out := redactPortfolio(nested, true, false).(map[string]any)
	exchanges := out["exchanges"].([]any)
	wallet := exchanges[0].(map[string]any)["wallets"].([]any)[0].(map[string]any)
	if wallet["address"] != addr {
		t.Errorf("--full should reveal nested address, got %v", wallet["address"])
	}
	if wallet["privateKey"] == "PRIV" {
		t.Error("nested privateKey must always be masked, even with --full")
	}
}

func TestPortfolioMasksUnknownStringsByDefault(t *testing.T) {
	// Unverified portfolio shape: any unrecognized string leaf (PII like email /
	// accountNumber) must be masked by default and revealed only with --full.
	sample := func() []map[string]any {
		return []map[string]any{{
			"symbol":        "ETH", // safe display field -> shown
			"email":         "user@example.com",
			"accountNumber": "ACC-12345678",
			"balance":       "5.0", // balance -> shown
		}}
	}
	def := redactPortfolio(sample(), false, false).([]map[string]any)[0]
	if def["email"] == "user@example.com" {
		t.Error("unknown string field 'email' must be masked by default")
	}
	if def["accountNumber"] == "ACC-12345678" {
		t.Error("'accountNumber' (account-class) must be masked by default")
	}
	if def["symbol"] != "ETH" {
		t.Errorf("safe display field 'symbol' must stay visible, got %v", def["symbol"])
	}
	if def["balance"] != "5.0" {
		t.Errorf("balance must stay visible by default, got %v", def["balance"])
	}

	full := redactPortfolio(sample(), true, false).([]map[string]any)[0]
	if full["email"] != "user@example.com" {
		t.Errorf("--full must reveal the unknown 'email' field, got %v", full["email"])
	}
}
