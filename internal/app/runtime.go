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
	if !resolved.HasAPIKey {
		return nil, &AuthRequiredError{Message: "altFINS API key not configured. Run `af auth set` or export ALTFINS_API_KEY."}
	}
	return altfins.NewClient(altfins.ClientConfig{
		BaseURL:    resolved.BaseURL,
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
