// Package config handles configuration loading and management.
package config

import (
	"os"
	"path/filepath"
	"time"

	"github.com/commatea/ComX-Bridge/pkg/core"
	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
)

// Default config file locations.
var configPaths = []string{
	"./config.yaml",
	"./config.yml",
	"./comx.yaml",
	"./comx.yml",
	"~/.config/comx/config.yaml",
	"/etc/comx/config.yaml",
}

// Load loads configuration from file.
func Load(path string) (*core.Config, error) {
	// If path is specified, use it directly
	if path != "" {
		return loadFile(path)
	}

	// Try default paths
	for _, p := range configPaths {
		// Expand home directory
		if p[0] == '~' {
			home, err := os.UserHomeDir()
			if err == nil {
				p = filepath.Join(home, p[2:])
			}
		}

		if _, err := os.Stat(p); err == nil {
			return loadFile(p)
		}
	}

	// Return default config if no file found
	return DefaultConfig(), nil
}

// loadFile loads configuration from a specific file.
func loadFile(path string) (*core.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg core.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	if err := Validate(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Validate validates the configuration.
func Validate(cfg *core.Config) error {
	validate := validator.New()
	return validate.Struct(cfg)
}

// Save saves configuration to file.
func Save(path string, cfg *core.Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// DefaultConfig returns a default configuration.
func DefaultConfig() *core.Config {
	return &core.Config{
		Gateways: []core.GatewayConfig{},
		Plugins: core.PluginConfig{
			Directory: "./plugins",
			AutoLoad:  true,
			Sandbox:   true,
		},
		AI: core.AIConfig{
			Enabled: false,
			Sidecar: core.SidecarConfig{
				Address: "localhost:50051",
				Timeout: 30 * time.Second,
			},
			Features: core.AIFeatures{
				AnomalyDetection: false,
				ProtocolAnalysis: false,
				AutoOptimize:     false,
				CodeGeneration:   false,
			},
		},
		Logging: core.LoggingConfig{
			Level:  "info",
			Format: "text",
			Output: "stdout",
		},
		Metrics: core.MetricsConfig{
			Enabled:  false,
			Endpoint: "/metrics",
			Interval: 10 * time.Second,
		},
	}
}
