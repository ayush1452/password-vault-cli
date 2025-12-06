// Package domain defines the core data structures and interfaces for the password vault.
// It contains the main business entities and their relationships.
package domain

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Entry represents a password entry in the vault
type Entry struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Username  string    `json:"username"`
	Secret    []byte    `json:"secret"`
	Password  []byte    `json:"password"` // Alias for Secret for backwards compatibility
	URL       string    `json:"url"`
	Notes     string    `json:"notes"`
	Tags      []string  `json:"tags"`
	TOTPSeed  string    `json:"totp_seed,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Profile represents a vault profile
type Profile struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

// VaultMetadata represents vault metadata
type VaultMetadata struct {
	Version   string                 `json:"version"`
	KDFParams map[string]interface{} `json:"kdf_params"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at,omitempty"`
	FileHMAC  string                 `json:"file_hmac,omitempty"` // HMAC of vault file for integrity checking
}

// Filter represents entry filtering options
type Filter struct {
	Search       string   `json:"search"`
	Tags         []string `json:"tags"`
	SearchTokens []string `json:"search_tokens"`
}

// Operation represents an audit log operation
type Operation struct {
	Type      string    `json:"type"`
	Profile   string    `json:"profile"`
	EntryID   string    `json:"entry_id"`
	Timestamp time.Time `json:"timestamp"`
	Success   bool      `json:"success"`
}

// Config represents the vault configuration
type Config struct {
	VaultPath          string        `yaml:"vault_path"`
	DefaultProfile     string        `yaml:"default_profile"`
	AutoLockTTL        time.Duration `yaml:"auto_lock_ttl"`
	ClipboardTTL       time.Duration `yaml:"clipboard_ttl"`
	OutputFormat       string        `yaml:"output_format"`
	ShowPasswords      bool          `yaml:"show_passwords"`
	ConfirmDestructive bool          `yaml:"confirm_destructive"`
	KDF                KDFConfig     `yaml:"kdf"`
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
	return &Config{
		VaultPath:          filepath.Join(home, ".local", "share", "vault", "vault.db"),
		DefaultProfile:     "default",
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
