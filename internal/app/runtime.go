package app

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/altfins-com/altfins-cli/internal/altfins"
	"github.com/altfins-com/altfins-cli/internal/config"
)

type RootOptions struct {
	Output  string
	DryRun  bool
	NoColor bool
	Fields  []string
}

type Factory struct {
	Options RootOptions
	Stdout  io.Writer
	Stderr  io.Writer
	Config  *config.Manager
}

type AuthRequiredError struct {
	Message string
}

func (e *AuthRequiredError) Error() string {
	if strings.TrimSpace(e.Message) != "" {
		return e.Message
	}
	return "altFINS API key is required"
}

func (e *AuthRequiredError) ExitCode() int {
	return 3
}

// ConfirmationRequiredExitCode is the process exit code returned when a mutating
// command needs an explicit confirmation/force flag that was not provided in a
// non-interactive context. It is exposed as `confirmationExitCode` in
// `af commands -o json` so the metadata and the real exit code cannot drift.
const ConfirmationRequiredExitCode = 5

// ConfirmationRequiredError is returned when a mutating command needs explicit
// confirmation (a confirmation flag such as --yes, or --force) that was not
// provided in a non-interactive context. It carries a dedicated exit code so
// autonomous agents can distinguish "pass a confirmation flag" from a generic
// failure.
type ConfirmationRequiredError struct {
	Command string
	Flags   []string
	Message string
}

func (e *ConfirmationRequiredError) Error() string {
	if strings.TrimSpace(e.Message) != "" {
		return e.Message
	}
	if strings.TrimSpace(e.Command) != "" {
		return fmt.Sprintf("%s requires explicit confirmation", e.Command)
	}
	return "command requires explicit confirmation"
}

func (e *ConfirmationRequiredError) ExitCode() int {
	return ConfirmationRequiredExitCode
}

func NewFactory(opts RootOptions, stdout, stderr io.Writer) (*Factory, error) {
	manager, err := config.NewManager(config.DefaultAppName)
	if err != nil {
		return nil, err
	}
	return &Factory{
		Options: opts,
		Stdout:  stdout,
		Stderr:  stderr,
		Config:  manager,
	}, nil
}

func (f *Factory) ResolveConfig() (config.Resolved, error) {
	return f.Config.Resolve()
}

func (f *Factory) NewClient() (*altfins.Client, error) {
	resolved, err := f.ResolveConfig()
	if err != nil {
		return nil, err
	}
	if !resolved.HasAPIKey && !f.Options.DryRun {
		return nil, &AuthRequiredError{Message: "altFINS API key not configured. Run `af auth set` or export ALTFINS_API_KEY."}
	}
	return altfins.NewClient(altfins.ClientConfig{
		BaseURL:    resolved.BaseURL,
		APIKey:     resolved.APIKey,
		AuthSource: resolved.AuthSource,
		DryRun:     f.Options.DryRun,
	}), nil
}

// NewMCPClient builds the MCP (JSON-RPC) client for MCP-backed commands
// (calendar, portfolio). It mirrors NewClient: it shares the stored API key and
// fails closed with an AuthRequiredError (exit 3) when no key is configured and
// the command is not a dry-run.
func (f *Factory) NewMCPClient() (*altfins.MCPClient, error) {
	resolved, err := f.ResolveConfig()
	if err != nil {
		return nil, err
	}
	if !resolved.HasAPIKey && !f.Options.DryRun {
		return nil, &AuthRequiredError{Message: "altFINS API key not configured. Run `af auth set` or export ALTFINS_API_KEY."}
	}
	return altfins.NewMCPClient(altfins.MCPClientConfig{
		URL:        resolved.MCPURL,
		APIKey:     resolved.APIKey,
		AuthSource: resolved.AuthSource,
		DryRun:     f.Options.DryRun,
	}), nil
}

func (f *Factory) WriteOutput(data any) error {
	return WriteOutput(f.Stdout, data, f.Options.Output, f.Options.Fields)
}

func (f *Factory) HandleCommandResult(data any, err error) error {
	if err == nil {
		return f.WriteOutput(data)
	}
	if dryRun, ok := altfins.IsDryRun(err); ok {
		enc := json.NewEncoder(f.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(dryRun.Preview)
	}
	return err
}

func MaskSecret(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if len(value) <= 8 {
		return strings.Repeat("*", len(value))
	}
	return value[:4] + strings.Repeat("*", len(value)-8) + value[len(value)-4:]
}

func ParseCSV(value string) []string {
	raw := strings.Split(value, ",")
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		item = strings.TrimSpace(item)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

func FormatError(err error) string {
	return fmt.Sprintf("Error: %v", err)
}
