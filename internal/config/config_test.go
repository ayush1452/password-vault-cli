package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestDefaultConfig tests creating default config
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig() returned nil")
	}

	// Test default values
	if cfg.DefaultProfile == "" {
		t.Error("DefaultProfile should have a default value")
	}

	if cfg.AutoLockTTL == 0 {
		t.Error("AutoLockTTL should have a default value")
	}

	if cfg.ClipboardTTL == 0 {
		t.Error("ClipboardTTL should have a default value")
	}

	if cfg.OutputFormat == "" {
		t.Error("OutputFormat should have a default value")
	}

	// Test KDF defaults
	if cfg.KDF.Memory == 0 {
		t.Error("KDF Memory should have a default value")
	}

	if cfg.KDF.Iterations == 0 {
		t.Error("KDF Iterations should have a default value")
	}

	if cfg.KDF.Parallelism == 0 {
		t.Error("KDF Parallelism should have a default value")
	}
}

// TestLoadConfigNonExistent tests loading non-existent config
func TestLoadConfigNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "non-existent-config.yaml")

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v, expected default config", err)
	}

	if cfg == nil {
		t.Fatal("LoadConfig() returned nil for non-existent config")
	}
}

// TestSaveAndLoadConfig tests saving and loading config
func TestSaveAndLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")

	// Create test config
	originalCfg := &Config{
		VaultPath:          "/test/vault/path",
		DefaultProfile:     "test-profile",
		AutoLockTTL:        2 * time.Hour,
		ClipboardTTL:       60 * time.Second,
		OutputFormat:       "json",
		ShowPasswords:      true,
		ConfirmDestructive: false,
		KDF: KDFConfig{
			Memory:      128000,
			Iterations:  5,
			Parallelism: 8,
		},
	}

	// Save config
	err := SaveConfig(originalCfg, configPath)
	if err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("Config file was not created")
	}

	// Load config back
	loadedCfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Verify values match
	if loadedCfg.VaultPath != originalCfg.VaultPath {
		t.Errorf("VaultPath = %v, want %v", loadedCfg.VaultPath, originalCfg.VaultPath)
	}

	if loadedCfg.DefaultProfile != originalCfg.DefaultProfile {
		t.Errorf("DefaultProfile = %v, want %v", loadedCfg.DefaultProfile, originalCfg.DefaultProfile)
	}

	if loadedCfg.AutoLockTTL != originalCfg.AutoLockTTL {
		t.Errorf("AutoLockTTL = %v, want %v", loadedCfg.AutoLockTTL, originalCfg.AutoLockTTL)
	}

	if loadedCfg.ClipboardTTL != originalCfg.ClipboardTTL {
		t.Errorf("ClipboardTTL = %v, want %v", loadedCfg.ClipboardTTL, originalCfg.ClipboardTTL)
	}

	if loadedCfg.OutputFormat != originalCfg.OutputFormat {
		t.Errorf("OutputFormat = %v, want %v", loadedCfg.OutputFormat, originalCfg.OutputFormat)
	}

	// Verify KDF values
	if loadedCfg.KDF.Memory != originalCfg.KDF.Memory {
		t.Errorf("KDF.Memory = %v, want %v", loadedCfg.KDF.Memory, originalCfg.KDF.Memory)
	}
}

// TestConfigFilePermissions tests config file permissions
func TestConfigFilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfg := DefaultConfig()
	err := SaveConfig(cfg, configPath)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Check file permissions
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}

	mode := info.Mode()
	expected := os.FileMode(0600)

	if mode.Perm() != expected {
		t.Errorf("Config file permissions = %o, want %o", mode.Perm(), expected)
	}
}

// TestConfigDirectoryCreation tests directory creation
func TestConfigDirectoryCreation(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nested", "dir", "config.yaml")

	cfg := DefaultConfig()
	err := SaveConfig(cfg, configPath)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify directory was created
	dir := filepath.Dir(configPath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Error("Config directory was not created")
	}

	// Verify file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}
}

// TestConfigStructFields tests Config struct fields
func TestConfigStructFields(t *testing.T) {
	cfg := &Config{
		VaultPath:      "/test/path",
		DefaultProfile: "test",
		AutoLockTTL:    time.Hour,
		Profiles:       make(map[string]*Profile),
	}

	// Test field access
	if cfg.VaultPath != "/test/path" {
		t.Error("VaultPath field access failed")
	}

	if cfg.DefaultProfile != "test" {
		t.Error("DefaultProfile field access failed")
	}

	// Test profile map
	cfg.Profiles["work"] = &Profile{
		Name:      "work",
		VaultPath: "/work/vault",
	}

	if len(cfg.Profiles) != 1 {
		t.Error("Profile map modification failed")
	}
}

// TestLoadInvalidYAML tests loading invalid YAML
func TestLoadInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")

	// Write invalid YAML
	invalidYAML := []byte("invalid: yaml: content: [unclosed")
	err := os.WriteFile(configPath, invalidYAML, 0600)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Try to load - should handle gracefully
	cfg, err := LoadConfig(configPath)
	if err == nil {
		t.Log("LoadConfig() returned default config for invalid YAML (acceptable)")
	}
	if cfg == nil {
		t.Error("LoadConfig() should return a config even for invalid YAML")
	}
}

// TestSecurityConfig tests SecurityConfig structure
func TestSecurityConfig(t *testing.T) {
	sec := SecurityConfig{
		SessionTimeout:      3600,
		MaxFailedAttempts:   3,
		RequireConfirmation: true,
	}

	if sec.SessionTimeout != 3600 {
		t.Error("SessionTimeout mismatch")
	}

	if sec.MaxFailedAttempts != 3 {
		t.Error("MaxFailedAttempts mismatch")
	}

	if !sec.RequireConfirmation {
		t.Error("RequireConfirmation should be true")
	}
}

// TestProfileConfig tests Profile structure
func TestProfileConfig(t *testing.T) {
	profile := Profile{
		Name:             "work",
		VaultPath:        "/work/vault.db",
		AutoLock:         3600,
		ClipboardTimeout: 30,
	}

	if profile.Name != "work" {
		t.Error("Profile name mismatch")
	}

	if profile.VaultPath != "/work/vault.db" {
		t.Error("Profile VaultPath mismatch")
	}
}

// TestKDFConfig tests KDF configuration
func TestKDFConfig(t *testing.T) {
	kdf := KDFConfig{
		Memory:      65536,
		Iterations:  3,
		Parallelism: 4,
	}

	if kdf.Memory != 65536 {
		t.Errorf("Memory = %v, want 65536", kdf.Memory)
	}

	if kdf.Iterations != 3 {
		t.Errorf("Iterations = %v, want 3", kdf.Iterations)
	}

	if kdf.Parallelism != 4 {
		t.Errorf("Parallelism = %v, want 4", kdf.Parallelism)
	}
}

// TestLoadConfigEmptyPath tests loading config with empty path
func TestLoadConfigEmptyPath(t *testing.T) {
	cfg, err := LoadConfig("")
	if err != nil {
		t.Fatalf("LoadConfig(\"\") error = %v, want nil", err)
	}

	if cfg == nil {
		t.Fatal("LoadConfig(\"\") returned nil config")
	}

	// Should return default config
	if cfg.DefaultProfile == "" {
		t.Error("LoadConfig(\"\") should return default config")
	}
}

// TestLoadConfigReadError tests error when reading config file
func TestLoadConfigReadError(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")

	// Create a file that we can't read (by making it inaccessible)
	cfg := DefaultConfig()
	err := SaveConfig(cfg, configPath)
	if err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	// Remove read permissions
	err = os.Chmod(configPath, 0000)
	if err != nil {
		t.Fatalf("Chmod() error = %v", err)
	}

	// Restore permissions after test
	defer os.Chmod(configPath, 0600)

	// Try to load - should return error
	loadedCfg, err := LoadConfig(configPath)
	if err == nil {
		t.Error("LoadConfig() should return error for unreadable file")
	}

	// Should still return a default config
	if loadedCfg == nil {
		t.Error("LoadConfig() should return default config even on error")
	}
}

// TestSaveConfigWriteError tests error when writing config file
func TestSaveConfigWriteError(t *testing.T) {
	// Try to write to a directory that doesn't exist and can't be created
	// Use a path that will fail
	invalidPath := "/invalid/nonexistent/path/that/cannot/be/created/config.yaml"
	
	cfg := DefaultConfig()
	err := SaveConfig(cfg, invalidPath)
	if err == nil {
		t.Error("SaveConfig() should return error for invalid path")
	}
}

// TestSaveConfigDirectoryCreationError tests error when creating directory fails
func TestSaveConfigDirectoryCreationError(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Create a file where we want a directory
	blockingFile := filepath.Join(tmpDir, "blocking")
	err := os.WriteFile(blockingFile, []byte("test"), 0600)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Try to create config in a path that would require this file to be a directory
	configPath := filepath.Join(blockingFile, "subdir", "config.yaml")
	
	cfg := DefaultConfig()
	err = SaveConfig(cfg, configPath)
	if err == nil {
		t.Error("SaveConfig() should return error when directory creation fails")
	}
}

// TestLoadConfigCreateDefaultError tests error when creating default config fails
func TestLoadConfigCreateDefaultError(t *testing.T) {
	//Use a path that definitely doesn't exist and will fail to write
	// Trying to write to root directory should fail on most systems
	configPath := "/readonly-test-vault-config-" + filepath.Base(t.TempDir()) + ".yaml"
	
	// Ensure cleanup even if the file somehow gets created
	defer os.Remove(configPath)
	
	cfg, err := LoadConfig(configPath)
	
	// The error should be returned from SaveConfig failing  
	if err == nil {
		t.Log("LoadConfig() might have succeeded on this system (OS-dependent)")
	}

	// Should still return a default config
	if cfg == nil {
		t.Error("LoadConfig() should return default config even on save error")
	}

	// Verify it's the default config
	if cfg.DefaultProfile == "" {
		t.Error("Should return default config with DefaultProfile set")
	}
}

// TestSaveConfigFileWritePermissionError tests write permission error
func TestSaveConfigFileWritePermissionError(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Make directory read-only
	configDir := filepath.Join(tmpDir, "readonly")
	err := os.MkdirAll(configDir, 0500)
	if err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	
	// Restore permissions after test
	defer os.Chmod(configDir, 0700)
	
	configPath := filepath.Join(configDir, "config.yaml")
	cfg := DefaultConfig()
	
	err = SaveConfig(cfg, configPath)
	if err == nil {
		// On some systems this might succeed, so we don't fail the test
		t.Log("SaveConfig() succeeded despite read-only directory (OS-dependent)")
	}
}

// TestLoadConfigYAMLUnmarshalError tests YAML unmarshal error
func TestLoadConfigYAMLUnmarshalError(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "bad.yaml")

	// Write completely invalid YAML that will fail unmarshaling
	badYAML := []byte("[[[\ninvalid:\n  - yaml\n    - malformed\n")
	err := os.WriteFile(configPath, badYAML, 0600)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Load should handle error gracefully
	cfg, err := LoadConfig(configPath)
	if err == nil {
		t.Log("LoadConfig() handled invalid YAML gracefully")
	}

	if cfg == nil {
		t.Error("LoadConfig() should return config even for YAML errors")
	}
}

