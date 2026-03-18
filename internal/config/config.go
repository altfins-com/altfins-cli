package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

const (
	DefaultAppName = "af"
	DefaultBaseURL = "https://altfins.com"
)

type Settings struct {
	APIKey  string `json:"api_key,omitempty" mapstructure:"api_key"`
	BaseURL string `json:"base_url,omitempty" mapstructure:"base_url"`
}

type Resolved struct {
	Settings
	AuthSource string `json:"authSource"`
	HasAPIKey  bool   `json:"hasApiKey"`
}

type Manager struct {
	path string
}

func NewManager(appName string) (*Manager, error) {
	path, err := DefaultPath(appName)
	if err != nil {
		return nil, err
	}
	return &Manager{path: path}, nil
}

func NewManagerAt(path string) *Manager {
	return &Manager{path: path}
}

func DefaultPath(appName string) (string, error) {
	if strings.TrimSpace(appName) == "" {
		appName = DefaultAppName
	}
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return filepath.Join(configDir, appName, "config.json"), nil
}

func (m *Manager) Path() string {
	return m.path
}

func (m *Manager) Load() (Settings, error) {
	v := viper.New()
	v.SetConfigFile(m.path)
	v.SetConfigType("json")
	v.SetDefault("base_url", DefaultBaseURL)
	v.BindEnv("api_key", "ALTFINS_API_KEY")
	v.BindEnv("base_url", "ALTFINS_BASE_URL")
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		var lookupErr viper.ConfigFileNotFoundError
		if !errors.As(err, &lookupErr) && !os.IsNotExist(err) {
			return Settings{}, fmt.Errorf("read config: %w", err)
		}
	}

	var out Settings
	if err := v.Unmarshal(&out); err != nil {
		return Settings{}, fmt.Errorf("parse config: %w", err)
	}
	out.APIKey = strings.TrimSpace(out.APIKey)
	out.BaseURL = strings.TrimSpace(out.BaseURL)
	if out.BaseURL == "" {
		out.BaseURL = DefaultBaseURL
	}
	return out, nil
}

func (m *Manager) Resolve() (Resolved, error) {
	settings, err := m.Load()
	if err != nil {
		return Resolved{}, err
	}

	source := ""
	if envKey, ok := os.LookupEnv("ALTFINS_API_KEY"); ok && strings.TrimSpace(envKey) != "" {
		source = "env"
	} else if settings.APIKey != "" {
		source = "config"
	}

	return Resolved{
		Settings:   settings,
		AuthSource: source,
		HasAPIKey:  source != "",
	}, nil
}

func (m *Manager) SaveAPIKey(apiKey string) error {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return fmt.Errorf("api key cannot be empty")
	}

	current, err := m.loadStored()
	if err != nil {
		return err
	}
	current.APIKey = apiKey
	if current.BaseURL == "" {
		current.BaseURL = DefaultBaseURL
	}
	return m.writeStored(current)
}

func (m *Manager) ClearAPIKey() error {
	current, err := m.loadStored()
	if err != nil {
		return err
	}
	current.APIKey = ""
	if strings.TrimSpace(current.BaseURL) == "" || current.BaseURL == DefaultBaseURL {
		if err := os.Remove(m.path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove config file: %w", err)
		}
		return nil
	}
	return m.writeStored(current)
}

func (m *Manager) loadStored() (Settings, error) {
	data, err := os.ReadFile(m.path)
	if err != nil {
		if os.IsNotExist(err) {
			return Settings{BaseURL: DefaultBaseURL}, nil
		}
		return Settings{}, fmt.Errorf("read config file: %w", err)
	}
	var out Settings
	if err := json.Unmarshal(data, &out); err != nil {
		return Settings{}, fmt.Errorf("decode config file: %w", err)
	}
	if strings.TrimSpace(out.BaseURL) == "" {
		out.BaseURL = DefaultBaseURL
	}
	return out, nil
}

func (m *Manager) writeStored(settings Settings) error {
	if err := os.MkdirAll(filepath.Dir(m.path), 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(m.path, data, 0o600); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}
	return nil
}
