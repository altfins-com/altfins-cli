package cmd

import (
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/altfins-com/altfins-cli/internal/app"
)

func newAuthCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage altFINS API key configuration",
	}

	var apiKey string
	setCmd := &cobra.Command{
		Use:   "set",
		Short: "Save an altFINS API key into local config",
		RunE: func(cmd *cobra.Command, args []string) error {
			factory, err := factoryFor(cmd)
			if err != nil {
				return err
			}
			if apiKey == "" {
				form := huh.NewForm(
					huh.NewGroup(
						huh.NewInput().
							Title("altFINS API Key").
							Description("Stored locally in ~/.config/af/config.json with 0600 permissions").
							EchoMode(huh.EchoModePassword).
							Value(&apiKey),
					),
				)
				if err := form.Run(); err != nil {
					return err
				}
			}
			if err := factory.Config.SaveAPIKey(apiKey); err != nil {
				return err
			}
			return handleResult(cmd, map[string]any{
				"status":        "saved",
				"path":          factory.Config.Path(),
				"apiKeyPreview": app.MaskSecret(apiKey),
			}, nil)
		},
	}
	setCmd.Flags().StringVar(&apiKey, "api-key", "", "API key to save")

	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show current auth configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			factory, err := factoryFor(cmd)
			if err != nil {
				return err
			}
			resolved, err := factory.ResolveConfig()
			if err != nil {
				return err
			}
			return handleResult(cmd, map[string]any{
				"path":          factory.Config.Path(),
				"authSource":    resolved.AuthSource,
				"hasApiKey":     resolved.HasAPIKey,
				"apiKeyPreview": app.MaskSecret(resolved.APIKey),
				"baseURL":       resolved.BaseURL,
			}, nil)
		},
	}

	clearCmd := &cobra.Command{
		Use:   "clear",
		Short: "Remove locally stored API key from config",
		RunE: func(cmd *cobra.Command, args []string) error {
			factory, err := factoryFor(cmd)
			if err != nil {
				return err
			}
			if err := factory.Config.ClearAPIKey(); err != nil {
				return err
			}
			return handleResult(cmd, map[string]any{
				"status": "cleared",
				"path":   factory.Config.Path(),
			}, nil)
		},
	}

	cmd.AddCommand(setCmd, statusCmd, clearCmd)
	return cmd
}
