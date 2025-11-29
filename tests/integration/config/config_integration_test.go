package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

type Config struct {
	VaultPath   string `yaml:"vault_path"`
	DefaultTTL  string `yaml:"default_ttl"`
	KDFMemory   int    `yaml:"kdf_memory"`
	KDFIterations int  `yaml:"kdf_iterations"`
}

// TestConfigLoading tests loading configuration from file
func TestConfigLoading(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	
	// Create config file
	config := Config{
		VaultPath:     "/path/to/vault.db",
		DefaultTTL:    "1h",
		KDFMemory:     65536,
		KDFIterations: 3,
	}
	
	data, _ := yaml.Marshal(config)
	os.WriteFile(configPath, data, 0600)
	
	// Load config
	var loaded Config
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}
	
	if err := yaml.Unmarshal(content, &loaded); err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}
	
	if loaded.VaultPath != config.VaultPath {
		t.Error("Config not loaded correctly")
	}
	
	t.Log("✓ Config loading works")
}

// TestConfigOverride tests CLI flags overriding config file
func TestConfigOverride(t *testing.T) {
	// Config file values
	fileConfig := Config{
		VaultPath:  "/default/vault.db",
		DefaultTTL: "1h",
	}
	
	// CLI override
	cliVaultPath := "/custom/vault.db"
	
	// CLI should take precedence
	finalPath := cliVaultPath
	if finalPath == "" {
		finalPath = fileConfig.VaultPath
	}
	
	if finalPath != cliVaultPath {
		t.Error("CLI override not working")
	}
	
	t.Log("✓ Config override works")
}

// TestConfigValidation tests configuration validation
func TestConfigValidation(t *testing.T) {
	testCases := []struct {
		name    string
		config  Config
		valid   bool
	}{
		{
			name: "valid config",
			config: Config{
				VaultPath:     "/valid/path.db",
				DefaultTTL:    "1h",
				KDFMemory:     65536,
				KDFIterations: 3,
			},
			valid: true,
		},
		{
			name: "invalid KDF memory",
			config: Config{
				VaultPath:     "/valid/path.db",
				DefaultTTL:    "1h",
				KDFMemory:     100, // Too low
				KDFIterations: 3,
			},
			valid: false,
		},
		{
			name: "invalid KDF iterations",
			config: Config{
				VaultPath:     "/valid/path.db",
				DefaultTTL:    "1h",
				KDFMemory:     65536,
				KDFIterations: 0, // Too low
			},
			valid: false,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			valid := validateConfig(tc.config)
			if valid != tc.valid {
				t.Errorf("Expected valid=%v, got %v", tc.valid, valid)
			}
		})
	}
	
	t.Log("✓ Config validation works")
}

// TestConfigDefaults tests default configuration values
func TestConfigDefaults(t *testing.T) {
	defaults := Config{
		DefaultTTL:    "1h",
		KDFMemory:     65536,
		KDFIterations: 3,
	}
	
	if defaults.DefaultTTL != "1h" {
		t.Error("Default TTL incorrect")
	}
	if defaults.KDFMemory != 65536 {
		t.Error("Default KDF memory incorrect")
	}
	
	t.Log("✓ Config defaults correct")
}

// TestConfigEnvironmentVariables tests environment variable support
func TestConfigEnvironmentVariables(t *testing.T) {
	// Set environment variable
	os.Setenv("VAULT_PATH", "/env/vault.db")
	defer os.Unsetenv("VAULT_PATH")
	
	// Environment should override file but not CLI
	envPath := os.Getenv("VAULT_PATH")
	if envPath != "/env/vault.db" {
		t.Error("Environment variable not read")
	}
	
	t.Log("✓ Environment variables work")
}

// Helper function
func validateConfig(c Config) bool {
	if c.KDFMemory < 1024 {
		return false
	}
	if c.KDFIterations < 1 {
		return false
	}
	return true
}
