package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/altfins-com/altfins-cli/internal/app"
)

// portfolioDefaultColumns is the curated default table/csv column set per FINAL.md.
// The output layer keeps only those present; address columns are appended only
// when --full. Kept small so the unverified portfolio shape degrades gracefully.
var portfolioDefaultColumns = []string{
	"exchange", "exchangeName", "wallet", "walletName", "source",
	"symbol", "ticker", "asset", "currency", "name",
	"balance", "amount", "quantity", "value",
}

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
			"This is account-private data. By default only known-safe display fields are shown; any\n" +
			"unrecognized field (and wallet addresses) is masked because the response shape is not pinned.\n" +
			"Use --full to reveal addresses and unrecognized fields (secrets such as API keys, private keys,\n" +
			"and mnemonics are ALWAYS masked, even with --full). Pass --mask-balances to hide balance figures,\n" +
			"or `--fields symbol` to project to symbols only.",
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
			// Default table view (no explicit --fields): project to the curated
			// columns; address columns only appear with --full.
			if tableMode && len(factory.Options.Fields) == 0 {
				cols := portfolioDefaultColumns
				if full {
					cols = append(append([]string(nil), cols...), "address", "walletAddress", "account")
				}
				value = applyDefaultColumns(value, factory.Options.Output, nil, cols)
			}
			return handleResult(cmd, value, nil)
		},
	}
	markMCPQuery(showCmd, "getUserPortfolio")
	showCmd.Flags().BoolVar(&maskBalances, "mask-balances", false, "Mask balance figures in the output")
	showCmd.Flags().BoolVar(&full, "full", false, "Reveal wallet addresses and unrecognized fields (secrets stay masked)")

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

// redactPortfolio walks a decoded portfolio payload and masks sensitive fields.
// Because the response schema is not pinned (real-PII, not sampled), it uses a
// default-deny model for scalar STRING leaves: a string is shown only if its key
// is a known-safe display field or a balance; anything else is masked by default
// and revealed only with --full. Secrets (API keys, private keys, mnemonics,
// passwords) are masked unconditionally, even with --full. Wallet addresses are
// masked unless --full. Numbers and booleans pass through unless their key is
// secret/address-class.
func redactPortfolio(value any, full, maskBalances bool) any {
	return redactPortfolioValue("", value, full, maskBalances)
}

// redactPortfolioValue is the key-aware walker. parentKey is the lower-cased map
// key under which value sits; it is carried into array elements so that scalars
// nested inside arrays (e.g. {"walletAddresses":[...]}, {"emails":[...]}) are
// classified by their parent key rather than leaking because the key was lost.
func redactPortfolioValue(parentKey string, value any, full, maskBalances bool) any {
	switch v := value.(type) {
	case []map[string]any:
		out := make([]map[string]any, 0, len(v))
		for _, item := range v {
			if m, ok := redactPortfolioValue(parentKey, item, full, maskBalances).(map[string]any); ok {
				out = append(out, m)
			}
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, child := range v {
			out[key] = redactPortfolioValue(strings.ToLower(key), child, full, maskBalances)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, child := range v {
			out[i] = redactPortfolioValue(parentKey, child, full, maskBalances)
		}
		return out
	default:
		if parentKey == "" {
			return value
		}
		return redactScalar(parentKey, value, full, maskBalances)
	}
}

// redactScalar applies the masking policy to a single scalar (string/number/bool).
func redactScalar(lowerKey string, child any, full, maskBalances bool) any {
	switch {
	case isAlwaysSecretKey(lowerKey):
		return app.MaskSecret(scalarString(child))
	case isWalletAddressKey(lowerKey):
		if full {
			return child
		}
		return app.MaskSecret(scalarString(child))
	case isBalanceKey(lowerKey):
		if maskBalances {
			return "***"
		}
		return child
	}
	// Unknown leaf: numbers/bools pass through; strings are default-deny.
	if s, ok := child.(string); ok {
		if isPortfolioSafeLeafKey(lowerKey) || full {
			return child
		}
		return app.MaskSecret(s)
	}
	return child
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
	for _, needle := range []string{"address", "addr", "account", "pubkey", "publickey", "iban", "hash"} {
		if strings.Contains(lowerKey, needle) {
			return true
		}
	}
	return lowerKey == "wallet"
}

func isBalanceKey(lowerKey string) bool {
	for _, needle := range []string{"balance", "amount", "quantity"} {
		if strings.Contains(lowerKey, needle) {
			return true
		}
	}
	return lowerKey == "qty"
}

// isPortfolioSafeLeafKey is the allowlist of clearly-non-PII string fields shown
// by default. Anything not on it (and not a balance/known class) is masked by
// default for the unverified-shape PII command.
func isPortfolioSafeLeafKey(lowerKey string) bool {
	for _, needle := range []string{"symbol", "ticker", "asset", "currency", "exchange", "network", "chain", "market", "price", "change", "percent", "volume", "marketcap"} {
		if strings.Contains(lowerKey, needle) {
			return true
		}
	}
	switch lowerKey {
	case "name", "coinname", "id", "coinid", "type", "status", "rank", "cap", "source":
		return true
	}
	return false
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
