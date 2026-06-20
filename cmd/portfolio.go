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
			"This is account-private data. Wallet addresses are redacted by default (use --full to reveal).\n" +
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
			value := redactPortfolio(mcpListValue(raw), full, maskBalances)
			return handleResult(cmd, value, nil)
		},
	}
	markMCPQuery(showCmd, "getUserPortfolio")
	showCmd.Flags().BoolVar(&maskBalances, "mask-balances", false, "Mask balance figures in the output")
	showCmd.Flags().BoolVar(&full, "full", false, "Reveal wallet addresses (redacted by default)")

	cmd.AddCommand(showCmd)
	return cmd
}

// redactPortfolio walks a decoded portfolio payload and masks sensitive fields by
// key name. Wallet-address-like fields are masked unless --full; balance-like
// fields are masked only when --mask-balances is set. The response schema is not
// pinned, so redaction is key-name heuristic rather than struct-field based.
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
			case !full && isAddressKey(lk):
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

func isAddressKey(lowerKey string) bool {
	for _, needle := range []string{"address", "walletaddr", "accountid", "private", "secret", "apikey", "mnemonic", "seedphrase"} {
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
