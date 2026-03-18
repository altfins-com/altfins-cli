package app

import (
	"path/filepath"
	"testing"

	"github.com/altfins-com/altfins-cli/internal/config"
)

func TestFactoryNewClientAllowsDryRunWithoutAPIKey(t *testing.T) {
	t.Setenv("ALTFINS_API_KEY", "")

	factory := &Factory{
		Options: RootOptions{DryRun: true},
		Config:  config.NewManagerAt(filepath.Join(t.TempDir(), "config.json")),
	}

	client, err := factory.NewClient()
	if err != nil {
		t.Fatalf("expected dry-run client without api key, got error: %v", err)
	}
	if client == nil {
		t.Fatal("expected client instance")
	}
}

func TestFactoryNewClientRequiresAPIKeyOutsideDryRun(t *testing.T) {
	t.Setenv("ALTFINS_API_KEY", "")

	factory := &Factory{
		Options: RootOptions{},
		Config:  config.NewManagerAt(filepath.Join(t.TempDir(), "config.json")),
	}

	_, err := factory.NewClient()
	if err == nil {
		t.Fatal("expected auth error without api key")
	}
	if _, ok := err.(*AuthRequiredError); !ok {
		t.Fatalf("expected auth required error, got %T", err)
	}
}
