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
	sample := func() []map[string]any {
		return []map[string]any{
			{"symbol": "ETH", "walletAddress": "0xABCDEF0123456789", "balance": "12.5", "exchange": "MetaMask"},
		}
	}

	// Default: wallet address redacted, balance preserved.
	def := redactPortfolio(sample(), false, false).([]map[string]any)[0]
	if def["walletAddress"] == "0xABCDEF0123456789" {
		t.Error("wallet address must be redacted by default")
	}
	if def["walletAddress"] == "" {
		t.Error("redacted wallet address should be masked, not removed")
	}
	if def["balance"] != "12.5" {
		t.Errorf("balance should be preserved by default, got %v", def["balance"])
	}
	if def["symbol"] != "ETH" {
		t.Errorf("non-sensitive symbol must be preserved, got %v", def["symbol"])
	}

	// --full: wallet address revealed.
	full := redactPortfolio(sample(), true, false).([]map[string]any)[0]
	if full["walletAddress"] != "0xABCDEF0123456789" {
		t.Errorf("--full must reveal the wallet address, got %v", full["walletAddress"])
	}

	// --mask-balances: balance hidden.
	masked := redactPortfolio(sample(), false, true).([]map[string]any)[0]
	if masked["balance"] != "***" {
		t.Errorf("--mask-balances must hide the balance, got %v", masked["balance"])
	}
}
