package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	DefaultFormat = "table"

	EnvProfile   = "MF_PROFILE"
	EnvFormat    = "MF_FORMAT"
	EnvNoInput   = "MF_NO_INPUT"
	EnvDebug     = "MF_DEBUG"
	EnvConfigDir = "MF_CONFIG_DIR"

	configFile = "config.yaml"
)

type Config struct {
	Version       int                `yaml:"version"`
	ActiveProfile string             `yaml:"active_profile"`
	Defaults      Defaults           `yaml:"defaults"`
	Profiles      map[string]Profile `yaml:"profiles"`
}

type Defaults struct {
	Format string `yaml:"format"`
}

type Profile struct {
	ClientID string   `yaml:"client_id"`
	OfficeID string   `yaml:"office_id,omitempty"`
	Scopes   []string `yaml:"scopes,omitempty"`
}

// DefaultConfigDir returns the default configuration directory.
func DefaultConfigDir() string {
	if d := os.Getenv(EnvConfigDir); d != "" {
		return d
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".config", "mf")
	}
	return filepath.Join(home, ".config", "mf")
}

// Load reads the configuration file.
func Load() (*Config, error) {
	dir := DefaultConfigDir()
	path := filepath.Join(dir, configFile)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return defaultConfig(), nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return &cfg, nil
}

// Save writes the configuration file.
func (c *Config) Save() error {
	dir := DefaultConfigDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	return os.WriteFile(filepath.Join(dir, configFile), data, 0600)
}

func defaultConfig() *Config {
	return &Config{
		Version:  1,
		Defaults: Defaults{Format: DefaultFormat},
		Profiles: make(map[string]Profile),
	}
}

// EnvOr returns the environment variable value or the fallback.
func EnvOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// IsNoInput returns whether non-interactive mode is enabled.
func IsNoInput() bool {
	return os.Getenv(EnvNoInput) == "1"
}
