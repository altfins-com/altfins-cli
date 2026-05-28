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
	var setForce bool
	setCmd := &cobra.Command{
		Use:   "set",
		Short: "Save an altFINS API key into local config",
		RunE: func(cmd *cobra.Command, args []string) error {
			factory, err := factoryFor(cmd)
			if err != nil {
				return err
			}

			// A key already stored in config must not be silently replaced.
			// An env-provided key (AuthSource "env") is not a stored key.
			resolved, err := factory.ResolveConfig()
			if err != nil {
				return err
			}
			keyAlreadyStored := resolved.AuthSource == "config"

			if factory.Options.DryRun {
				return handleResult(cmd, map[string]any{
					"dryRun":          true,
					"action":          "auth set",
					"path":            factory.Config.Path(),
					"executed":        false,
					"wouldReplaceKey": keyAlreadyStored,
					"forceRequired":   keyAlreadyStored && !setForce,
				}, nil)
			}

			if keyAlreadyStored && !setForce {
				return &app.ConfirmationRequiredError{
					Command: "auth set",
					Flags:   []string{"--force"},
					Message: "an API key is already stored in config; pass --force to replace it",
				}
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
				"replaced":      keyAlreadyStored,
			}, nil)
		},
	}
	setCmd.Flags().StringVar(&apiKey, "api-key", "", "API key to save")
	setCmd.Flags().BoolVar(&setForce, "force", false, "Replace an existing stored API key instead of failing")
	annotateSafety(setCmd, safetyMeta{
		OperationType:     OpLocalWrite,
		MutatesLocalState: true,
		DryRunSupported:   true,
		ForceSupported:    true,
	})

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
	markLocalRead(statusCmd)

	var clearYes bool
	clearCmd := &cobra.Command{
		Use:   "clear",
		Short: "Remove locally stored API key from config",
		RunE: func(cmd *cobra.Command, args []string) error {
			factory, err := factoryFor(cmd)
			if err != nil {
				return err
			}

			if factory.Options.DryRun {
				return handleResult(cmd, map[string]any{
					"dryRun":   true,
					"action":   "auth clear",
					"path":     factory.Config.Path(),
					"executed": false,
				}, nil)
			}

			if !clearYes {
				if isInteractiveStdin(cmd) {
					confirm := false
					form := huh.NewForm(
						huh.NewGroup(
							huh.NewConfirm().
								Title("Clear the stored altFINS API key?").
								Affirmative("Yes").
								Negative("No").
								Value(&confirm),
						),
					)
					if err := form.Run(); err != nil {
						return err
					}
					if !confirm {
						return handleResult(cmd, map[string]any{
							"status": "aborted",
							"path":   factory.Config.Path(),
						}, nil)
					}
				} else {
					return &app.ConfirmationRequiredError{
						Command: "auth clear",
						Flags:   []string{"--yes", "-y"},
						Message: "auth clear permanently removes the stored API key; pass --yes/-y to confirm in non-interactive mode",
					}
				}
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
	clearCmd.Flags().BoolVarP(&clearYes, "yes", "y", false, "Confirm removal without an interactive prompt")
	annotateSafety(clearCmd, safetyMeta{
		OperationType:        OpLocalWrite,
		MutatesLocalState:    true,
		DryRunSupported:      true,
		ConfirmationRequired: true,
		ConfirmationFlags:    []string{"--yes", "-y"},
	})

	cmd.AddCommand(setCmd, statusCmd, clearCmd)
	return cmd
}
