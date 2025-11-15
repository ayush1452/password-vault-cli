// Package config handles the configuration management for the password vault.
// It provides functionality to load, save, and manage application configuration.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the vault configuration
type Config struct {
	VaultPath          string              `yaml:"vault_path"`
	DefaultProfile     string              `yaml:"default_profile"`
	Profiles           map[string]*Profile `yaml:"profiles"`
	AutoLockTTL        time.Duration       `yaml:"auto_lock_ttl"`
	ClipboardTTL       time.Duration       `yaml:"clipboard_ttl"`
	OutputFormat       string              `yaml:"output_format"`
	ShowPasswords      bool                `yaml:"show_passwords"`
	ConfirmDestructive bool                `yaml:"confirm_destructive"`
	KDF                KDFConfig           `yaml:"kdf"`
	Security           SecurityConfig      `yaml:"security"`
}

// Profile represents a vault profile configuration
type Profile struct {
	Name             string `yaml:"name"`
	VaultPath        string `yaml:"vault_path"`
	AutoLock         int    `yaml:"auto_lock"`
	ClipboardTimeout int    `yaml:"clipboard_timeout"`
}

// SecurityConfig represents security-related configuration
type SecurityConfig struct {
	SessionTimeout      int  `yaml:"session_timeout"`
	MaxFailedAttempts   int  `yaml:"max_failed_attempts"`
	RequireConfirmation bool `yaml:"require_confirmation"`
}

// KDFConfig represents KDF parameters for new vaults
type KDFConfig struct {
	Memory      uint32 `yaml:"memory"`
	Iterations  uint32 `yaml:"iterations"`
	Parallelism uint8  `yaml:"parallelism"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	home, _ := os.UserHomeDir()
	vaultPath := filepath.Join(home, ".local", "share", "vault", "vault.db")
	return &Config{
		VaultPath:      vaultPath,
		DefaultProfile: "default",
		Profiles: map[string]*Profile{
			"default": {
				Name:             "default",
				VaultPath:        vaultPath,
				AutoLock:         3600,
				ClipboardTimeout: 30,
			},
		},
		AutoLockTTL:        time.Hour,
		ClipboardTTL:       30 * time.Second,
		OutputFormat:       "table",
		ShowPasswords:      false,
		ConfirmDestructive: true,
		KDF: KDFConfig{
			Memory:      65536, // 64 MB
			Iterations:  3,
			Parallelism: 4,
		},
		Security: SecurityConfig{
			SessionTimeout:      1800,
			MaxFailedAttempts:   3,
			RequireConfirmation: true,
		},
	}
}

// LoadConfig loads configuration from file or returns default
func LoadConfig(configPath string) (*Config, error) {
	cfg := DefaultConfig()

	if configPath == "" {
		return cfg, nil
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Create default config file
		if err := SaveConfig(cfg, configPath); err != nil {
			return cfg, fmt.Errorf("failed to create default config: %w", err)
		}
		return cfg, nil
	}

	// Clean the file path to prevent directory traversal
	cleanPath := filepath.Clean(configPath)

	// Optional: Add additional path validation here if needed
	// For example, ensure the path is within an allowed directory

	// Read config file
	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return cfg, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return cfg, fmt.Errorf("failed to parse config file: %w", err)
	}

	return cfg, nil
}

// SaveConfig saves configuration to file
func SaveConfig(cfg *Config, configPath string) error {
	// Clean the file path to prevent directory traversal
	cleanPath := filepath.Clean(configPath)

	// Optional: Add additional path validation here if needed
	// For example, ensure the path is within an allowed directory

	// Create directory if it doesn't exist
	dir := filepath.Dir(cleanPath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file using cleaned path
	if err := os.WriteFile(cleanPath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
