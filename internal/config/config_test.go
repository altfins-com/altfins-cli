package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveUsesEnvOverConfig(t *testing.T) {
	t.Setenv("ALTFINS_API_KEY", "env-secret")
	manager := NewManagerAt(filepath.Join(t.TempDir(), "config.json"))
	if err := manager.SaveAPIKey("config-secret"); err != nil {
		t.Fatalf("save api key: %v", err)
	}

	resolved, err := manager.Resolve()
	if err != nil {
		t.Fatalf("resolve config: %v", err)
	}
	if got, want := resolved.APIKey, "env-secret"; got != want {
		t.Fatalf("api key mismatch: got %q want %q", got, want)
	}
	if got, want := resolved.AuthSource, "env"; got != want {
		t.Fatalf("auth source mismatch: got %q want %q", got, want)
	}
}

func TestSaveAPIKeyUsesStrictPermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "af", "config.json")
	manager := NewManagerAt(path)

	if err := manager.SaveAPIKey("stored-secret"); err != nil {
		t.Fatalf("save api key: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat config file: %v", err)
	}
	if got, want := info.Mode().Perm(), os.FileMode(0o600); got != want {
		t.Fatalf("file permissions mismatch: got %v want %v", got, want)
	}
}
