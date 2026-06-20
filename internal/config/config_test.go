package config

import (
	"os"
	"path/filepath"
	"runtime"
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

func TestClearAPIKeyPreservesCustomMCPURL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"api_key":"k","mcp_url":"https://staging.mcp.example/mcp"}`), 0o600); err != nil {
		t.Fatalf("seed config: %v", err)
	}
	manager := NewManagerAt(path)
	if err := manager.ClearAPIKey(); err != nil {
		t.Fatalf("clear api key: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("config file must be kept when a custom mcp_url is set: %v", err)
	}
	settings, err := manager.Load()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if settings.MCPURL != "https://staging.mcp.example/mcp" {
		t.Errorf("custom mcp_url must survive auth clear, got %q", settings.MCPURL)
	}
	if settings.APIKey != "" {
		t.Errorf("api key must be cleared, got %q", settings.APIKey)
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
	if runtime.GOOS == "windows" {
		return
	}
	if got, want := info.Mode().Perm(), os.FileMode(0o600); got != want {
		t.Fatalf("file permissions mismatch: got %v want %v", got, want)
	}
}
