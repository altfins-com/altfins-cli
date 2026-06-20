package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/altfins-com/altfins-cli/internal/app"
)

func newPortfolioCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "portfolio",
		Short: "Authenticated user's crypto portfolio",
	}

	var maskBalances bool
	var full bool

	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Show the authenticated user's portfolio holdings",
		Long: "Show the authenticated user's crypto portfolio (holdings across connected exchanges and wallets).\n\n" +
			"This is account-private data. Wallet addresses are hidden by default (use --full to reveal).\n" +
			"Secrets (API keys, private keys, mnemonics) are ALWAYS masked, even with --full.\n" +
			"Pass --mask-balances to hide balance figures, or use `--fields symbol` to project to symbols only.",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := mcpClientFor(cmd)
			if err != nil {
				return err
			}
			var raw json.RawMessage
			if callErr := client.CallTool(cmd.Context(), "getUserPortfolio", map[string]any{}, &raw); callErr != nil {
				return handleResult(cmd, nil, callErr)
			}
			factory, err := factoryFor(cmd)
			if err != nil {
				return err
			}
			value := redactPortfolio(mcpListValue(raw), full, maskBalances)

			tableMode := isTableMode(factory.Options.Output)
			if tableMode && isEmptyResult(value) {
				fmt.Fprintln(factory.Stdout, "No connected exchanges or wallets. Connect an exchange in the altFINS web app to see your holdings.")
				return nil
			}
			// In the default table view (no explicit --fields), hide wallet-address
			// fields entirely unless --full; JSON/JSONL keep them masked.
			if tableMode && len(factory.Options.Fields) == 0 && !full {
				value = dropAddressKeys(value)
			}
			return handleResult(cmd, value, nil)
		},
	}
	markMCPQuery(showCmd, "getUserPortfolio")
	showCmd.Flags().BoolVar(&maskBalances, "mask-balances", false, "Mask balance figures in the output")
	showCmd.Flags().BoolVar(&full, "full", false, "Reveal wallet addresses (hidden by default; secrets stay masked)")

	cmd.AddCommand(showCmd)
	return cmd
}

func isTableMode(output string) bool {
	switch strings.ToLower(strings.TrimSpace(output)) {
	case "", "table", "csv":
		return true
	default:
		return false
	}
}

func isEmptyResult(value any) bool {
	switch v := value.(type) {
	case []map[string]any:
		return len(v) == 0
	case []any:
		return len(v) == 0
	case map[string]any:
		return len(v) == 0
	case nil:
		return true
	default:
		return false
	}
}

// redactPortfolio walks a decoded portfolio payload and masks sensitive fields by
// key name. Secrets (API keys, private keys, mnemonics, passwords) are ALWAYS
// masked, even with --full. Wallet-address-like fields are masked unless --full.
// Balance-like fields are masked only when --mask-balances is set. The response
// schema is not pinned (real-PII, not sampled), so redaction is key-name heuristic.
func redactPortfolio(value any, full, maskBalances bool) any {
	switch v := value.(type) {
	case []map[string]any:
		out := make([]map[string]any, 0, len(v))
		for _, item := range v {
			redacted := redactPortfolio(item, full, maskBalances)
			if m, ok := redacted.(map[string]any); ok {
				out = append(out, m)
			} else {
				out = append(out, map[string]any{"value": redacted})
			}
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, child := range v {
			lk := strings.ToLower(key)
			switch {
			case isAlwaysSecretKey(lk):
				out[key] = app.MaskSecret(scalarString(child))
			case !full && isWalletAddressKey(lk):
				out[key] = app.MaskSecret(scalarString(child))
			case maskBalances && isBalanceKey(lk):
				out[key] = "***"
			default:
				out[key] = redactPortfolio(child, full, maskBalances)
			}
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, child := range v {
			out[i] = redactPortfolio(child, full, maskBalances)
		}
		return out
	default:
		return value
	}
}

// dropAddressKeys removes wallet-address-like fields outright (rather than masking
// them). Used for the default table view so addresses are hidden, not shown masked.
func dropAddressKeys(value any) any {
	switch v := value.(type) {
	case []map[string]any:
		out := make([]map[string]any, 0, len(v))
		for _, item := range v {
			if m, ok := dropAddressKeys(item).(map[string]any); ok {
				out = append(out, m)
			}
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, child := range v {
			if isWalletAddressKey(strings.ToLower(key)) {
				continue
			}
			out[key] = dropAddressKeys(child)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, child := range v {
			out[i] = dropAddressKeys(child)
		}
		return out
	default:
		return value
	}
}

// isAlwaysSecretKey matches credential-bearing fields that must never be shown,
// even with --full.
func isAlwaysSecretKey(lowerKey string) bool {
	for _, needle := range []string{"private", "secret", "apikey", "api_key", "mnemonic", "seedphrase", "seed_phrase", "password", "passphrase", "pwd"} {
		if strings.Contains(lowerKey, needle) {
			return true
		}
	}
	switch lowerKey {
	case "pass", "token", "key":
		return true
	}
	return false
}

// isWalletAddressKey matches wallet-address-like fields, masked by default and
// revealed by --full. Errs toward over-redaction (a privacy control).
func isWalletAddressKey(lowerKey string) bool {
	for _, needle := range []string{"address", "addr", "accountid", "pubkey", "publickey", "iban"} {
		if strings.Contains(lowerKey, needle) {
			return true
		}
	}
	switch lowerKey {
	case "wallet", "account", "hash":
		return true
	}
	return false
}

func isBalanceKey(lowerKey string) bool {
	for _, needle := range []string{"balance", "amount", "quantity"} {
		if strings.Contains(lowerKey, needle) {
			return true
		}
	}
	return lowerKey == "qty"
}

func scalarString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", typed)
	}
}
