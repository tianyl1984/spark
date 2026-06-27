package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// DefaultPort is used when config.json does not specify a port.
const DefaultPort = 8010

// Config holds the runtime configuration loaded from $HOME/.spark/config.json.
type Config struct {
	// Port is the TCP port the webhook server listens on.
	Port int `json:"port"`
	// Secret is the GitHub webhook secret used to verify request signatures.
	// When empty, signature verification is skipped.
	Secret string `json:"secret"`
}

// Dir returns the spark configuration directory: $HOME/.spark.
func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, ".spark"), nil
}

// Load reads and parses $HOME/.spark/config.json.
func Load() (*Config, error) {
	dir, err := Dir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "config.json")

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}

	cfg := &Config{Port: DefaultPort}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	if cfg.Port == 0 {
		cfg.Port = DefaultPort
	}
	return cfg, nil
}
