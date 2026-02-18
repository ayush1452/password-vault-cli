package domain

import (
	"os"
	"testing"
	"time"
)

// TestEntryFields tests Entry field access
func TestEntryFields(t *testing.T) {
	now := time.Now().UTC()
	entry := Entry{
		ID:        "test-id",
		Name:      "test-name",
		Username:  "test-user",
		Secret:    []byte("test-secret"),
		URL:       "https://test.com",
		Tags:      []string{"tag1", "tag2"},
		Notes:     "test notes",
		TOTPSeed:  "TESTSEED",
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Test field access
	if entry.ID != "test-id" {
		t.Errorf("ID = %v, want test-id", entry.ID)
	}

	if entry.Name != "test-name" {
		t.Errorf("Name = %v, want test-name", entry.Name)
	}

	if entry.Username != "test-user" {
		t.Errorf("Username = %v, want test-user", entry.Username)
	}

	if string(entry.Secret) != "test-secret" {
		t.Error("Secret mismatch")
	}

	if entry.URL != "https://test.com" {
		t.Errorf("URL = %v, want https://test.com", entry.URL)
	}

	if len(entry.Tags) != 2 {
		t.Errorf("Tags length = %d, want 2", len(entry.Tags))
	}

	if entry.Notes != "test notes" {
		t.Errorf("Notes = %v, want test notes", entry.Notes)
	}

	if entry.TOTPSeed != "TESTSEED" {
		t.Errorf("TOTPSeed = %v, want TESTSEED", entry.TOTPSeed)
	}
}

// TestEntryMinimalFields tests entry with minimal fields
func TestEntryMinimalFields(t *testing.T) {
	entry := Entry{
		Name:   "minimal",
		Secret: []byte("secret"),
	}

	if entry.Name != "minimal" {
		t.Error("Name mismatch")
	}

	if string(entry.Secret) != "secret" {
		t.Error("Secret mismatch")
	}

	// Optional fields should be zero values
	if entry.Username != "" {
		t.Error("Username should be empty")
	}

	if entry.URL != "" {
		t.Error("URL should be empty")
	}

	if len(entry.Tags) != 0 {
		t.Error("Tags should be empty")
	}
}

// TestProfileFields tests Profile field access
func TestProfileFields(t *testing.T) {
	now := time.Now().UTC()
	profile := Profile{
		Name:        "work",
		Description: "Work accounts",
		CreatedAt:   now,
	}

	if profile.Name != "work" {
		t.Errorf("Name = %v, want work", profile.Name)
	}

	if profile.Description != "Work accounts" {
		t.Errorf("Description = %v, want Work accounts", profile.Description)
	}

	if profile.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
}

// TestVaultMetadata tests VaultMetadata structure
func TestVaultMetadata(t *testing.T) {
	now := time.Now().UTC()
	meta := VaultMetadata{
		Version:   "1.0",
		CreatedAt: now,
		UpdatedAt: now.Add(time.Hour),
		KDFParams: map[string]interface{}{
			"memory":      65536,
			"iterations":  3,
			"parallelism": 4,
		},
	}

	if meta.Version != "1.0" {
		t.Errorf("Version = %v, want 1.0", meta.Version)
	}

	if meta.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}

	if !meta.UpdatedAt.After(meta.CreatedAt) {
		t.Error("UpdatedAt should be after CreatedAt")
	}

	if len(meta.KDFParams) == 0 {
		t.Error("KDFParams should not be empty")
	}
}

// TestFilter tests Filter structure
func TestFilter(t *testing.T) {
	filter := Filter{
		Search:       "github",
		Tags:         []string{"work", "personal"},
		SearchTokens: []string{"git", "hub"},
	}

	if filter.Search != "github" {
		t.Errorf("Search = %v, want github", filter.Search)
	}

	if len(filter.Tags) != 2 {
		t.Errorf("Tags length = %d, want 2", len(filter.Tags))
	}

	if len(filter.SearchTokens) != 2 {
		t.Errorf("SearchTokens length = %d, want 2", len(filter.SearchTokens))
	}
}

// TestOperation tests Operation structure
func TestOperation(t *testing.T) {
	now := time.Now().UTC()
	op := Operation{
		Type:      "CREATE",
		Profile:   "default",
		EntryID:   "entry-1",
		Timestamp: now,
		Success:   true,
	}

	if op.Type != "CREATE" {
		t.Errorf("Type = %v, want CREATE", op.Type)
	}

	if op.Profile != "default" {
		t.Errorf("Profile = %v, want default", op.Profile)
	}

	if !op.Success {
		t.Error("Success should be true")
	}
}

// TestDefaultConfig tests default configuration
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig() returned nil")
	}

	if cfg.DefaultProfile != "default" {
		t.Errorf("DefaultProfile = %v, want default", cfg.DefaultProfile)
	}

	if cfg.AutoLockTTL == 0 {
		t.Error("AutoLockTTL should have a value")
	}

	if cfg.ClipboardTTL == 0 {
		t.Error("ClipboardTTL should have a value")
	}

	if cfg.KDF.Memory == 0 {
		t.Error("KDF Memory should have a value")
	}

	if cfg.KDF.Iterations == 0 {
		t.Error("KDF Iterations should have a value")
	}

	if cfg.KDF.Parallelism == 0 {
		t.Error("KDF Parallelism should have a value")
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

// TestConfigFieldModification tests modifying config fields
func TestConfigFieldModification(t *testing.T) {
	cfg := DefaultConfig()

	// Modify fields
	cfg.DefaultProfile = "custom"
	cfg.ShowPasswords = true
	cfg.AutoLockTTL = 2 * time.Hour

	if cfg.DefaultProfile != "custom" {
		t.Error("DefaultProfile modification failed")
	}

	if !cfg.ShowPasswords {
		t.Error("ShowPasswords modification failed")
	}

	if cfg.AutoLockTTL != 2*time.Hour {
		t.Error("AutoLockTTL modification failed")
	}
}

// TestEntryTagsManipulation tests tag array manipulation
func TestEntryTagsManipulation(t *testing.T) {
	entry := Entry{
		Name:   "test",
		Secret: []byte("secret"),
		Tags:   []string{"tag1"},
	}

	// Add tag
	entry.Tags = append(entry.Tags, "tag2")

	if len(entry.Tags) != 2 {
		t.Errorf("Tags length = %d, want 2", len(entry.Tags))
	}

	// Check contains
	found := false
	for _, tag := range entry.Tags {
		if tag == "tag2" {
			found = true
			break
		}
	}

	if !found {
		t.Error("tag2 not found after append")
	}
}

// TestPasswordFieldAlias tests Password field as alias for Secret
func TestPasswordFieldAlias(t *testing.T) {
	entry := Entry{
		Name:     "test",
		Password: []byte("password-via-alias"),
	}

	// Both fields should exist
	if string(entry.Password) != "password-via-alias" {
		t.Error("Password field mismatch")
	}
}

// TestLoadConfig tests loading configuration
func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() string
		cleanup func(string)
		wantErr bool
	}{
		{
			name: "Empty config path returns default",
			setup: func() string {
				return ""
			},
			cleanup: func(s string) {},
			wantErr: false,
		},
		{
			name: "Non-existent file creates default",
			setup: func() string {
				return "/tmp/test-vault-config-nonexistent.yaml"
			},
			cleanup: func(path string) {
				os.Remove(path)
			},
			wantErr: false,
		},
		{
			name: "Valid config file",
			setup: func() string {
				path := "/tmp/test-vault-config-valid.yaml"
				cfg := DefaultConfig()
				cfg.DefaultProfile = "test-profile"
				SaveConfig(cfg, path)
				return path
			},
			cleanup: func(path string) {
				os.Remove(path)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup()
			defer tt.cleanup(path)

			cfg, err := LoadConfig(path)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
			if cfg == nil {
				t.Error("LoadConfig() returned nil config")
			}
		})
	}
}

// TestLoadConfigReadError tests read error handling
func TestLoadConfigReadError(t *testing.T) {
	// Create a file with no read permissions
	path := "/tmp/test-vault-config-noread.yaml"
	os.WriteFile(path, []byte("test"), 0o000)
	defer os.Remove(path)

	_, err := LoadConfig(path)
	if err == nil {
		t.Error("LoadConfig() should fail with unreadable file")
	}
}

// TestLoadConfigInvalidYAML tests YAML parsing error
func TestLoadConfigInvalidYAML(t *testing.T) {
	path := "/tmp/test-vault-config-invalid.yaml"
	os.WriteFile(path, []byte("invalid: yaml: content: ["), 0o600)
	defer os.Remove(path)

	_, err := LoadConfig(path)
	if err == nil {
		t.Error("LoadConfig() should fail with invalid YAML")
	}
}

// TestLoadConfigValidYAML tests successful YAML parsing
func TestLoadConfigValidYAML(t *testing.T) {
	path := "/tmp/test-vault-config-custom.yaml"
	cfg := DefaultConfig()
	cfg.DefaultProfile = "custom-profile"
	cfg.AutoLockTTL = 2 * time.Hour
	cfg.ClipboardTTL = 60 * time.Second

	SaveConfig(cfg, path)
	defer os.Remove(path)

	loaded, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if loaded.DefaultProfile != "custom-profile" {
		t.Errorf("DefaultProfile = %v, want custom-profile", loaded.DefaultProfile)
	}

	if loaded.AutoLockTTL != 2*time.Hour {
		t.Errorf("AutoLockTTL = %v, want 2h", loaded.AutoLockTTL)
	}
}

// TestSaveConfig tests saving configuration
func TestSaveConfig(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		cfg     *Config
		wantErr bool
	}{
		{
			name:    "Valid save",
			path:    "/tmp/test-vault-save-valid.yaml",
			cfg:     DefaultConfig(),
			wantErr: false,
		},
		{
			name:    "Save to subdirectory",
			path:    "/tmp/test-vault-subdir/config.yaml",
			cfg:     DefaultConfig(),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer os.RemoveAll("/tmp/test-vault-save-valid.yaml")
			defer os.RemoveAll("/tmp/test-vault-subdir")

			err := SaveConfig(tt.cfg, tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("SaveConfig() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				// Verify file exists
				if _, err := os.Stat(tt.path); os.IsNotExist(err) {
					t.Error("SaveConfig() did not create file")
				}
			}
		})
	}
}

// TestSaveConfigInvalidPath tests error handling for invalid paths
func TestSaveConfigInvalidPath(t *testing.T) {
	cfg := DefaultConfig()

	// Try to save to a path where we can't create directories
	err := SaveConfig(cfg, "/root/forbidden/config.yaml")
	if err == nil {
		t.Error("SaveConfig() should fail with forbidden path")
	}
}

// TestConfigRoundTrip tests save and load cycle
func TestConfigRoundTrip(t *testing.T) {
	path := "/tmp/test-vault-roundtrip.yaml"
	defer os.Remove(path)

	// Create custom config
	original := DefaultConfig()
	original.DefaultProfile = "roundtrip-test"
	original.ShowPasswords = true
	original.ConfirmDestructive = false
	original.OutputFormat = "json"
	original.KDF.Memory = 131072
	original.KDF.Iterations = 5

	// Save
	if err := SaveConfig(original, path); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	// Load
	loaded, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Compare
	if loaded.DefaultProfile != original.DefaultProfile {
		t.Errorf("DefaultProfile = %v, want %v", loaded.DefaultProfile, original.DefaultProfile)
	}
	if loaded.ShowPasswords != original.ShowPasswords {
		t.Errorf("ShowPasswords = %v, want %v", loaded.ShowPasswords, original.ShowPasswords)
	}
	if loaded.ConfirmDestructive != original.ConfirmDestructive {
		t.Errorf("ConfirmDestructive = %v, want %v", loaded.ConfirmDestructive, original.ConfirmDestructive)
	}
	if loaded.OutputFormat != original.OutputFormat {
		t.Errorf("OutputFormat = %v, want %v", loaded.OutputFormat, original.OutputFormat)
	}
	if loaded.KDF.Memory != original.KDF.Memory {
		t.Errorf("KDF.Memory = %v, want %v", loaded.KDF.Memory, original.KDF.Memory)
	}
	if loaded.KDF.Iterations != original.KDF.Iterations {
		t.Errorf("KDF.Iterations = %v, want %v", loaded.KDF.Iterations, original.KDF.Iterations)
	}
}

// TestConfigPathCleaning tests that paths are properly cleaned
func TestConfigPathCleaning(t *testing.T) {
	path := "/tmp/../tmp/test-vault-cleaned.yaml"
	cfg := DefaultConfig()

	err := SaveConfig(cfg, path)
	defer os.Remove("/tmp/test-vault-cleaned.yaml")

	if err != nil {
		t.Errorf("SaveConfig() with path traversal error = %v", err)
	}

	// Verify the cleaned path was used
	if _, err := os.Stat("/tmp/test-vault-cleaned.yaml"); os.IsNotExist(err) {
		t.Error("File not created at expected cleaned path")
	}
}

// TestEntryPasswordBackwardCompatibility tests Password field compatibility
func TestEntryPasswordBackwardCompatibility(t *testing.T) {
	secret := []byte("test-secret")

	// Test setting Secret field
	entry1 := Entry{
		Name:   "test1",
		Secret: secret,
	}

	// Both should be accessible
	if string(entry1.Secret) != string(secret) {
		t.Error("Secret field mismatch")
	}

	// Test setting Password field
	entry2 := Entry{
		Name:     "test2",
		Password: secret,
	}

	if string(entry2.Password) != string(secret) {
		t.Error("Password field mismatch")
	}
}

// TestVaultMetadataWithFileHMAC tests FileHMAC field
func TestVaultMetadataWithFileHMAC(t *testing.T) {
	meta := VaultMetadata{
		Version:   "1.0",
		CreatedAt: time.Now(),
		FileHMAC:  "abc123def456",
	}

	if meta.FileHMAC != "abc123def456" {
		t.Errorf("FileHMAC = %v, want abc123def456", meta.FileHMAC)
	}
}

// TestConfigAllFields tests all Config fields
func TestConfigAllFields(t *testing.T) {
	cfg := &Config{
		VaultPath:          "/custom/path",
		DefaultProfile:     "work",
		AutoLockTTL:        2 * time.Hour,
		ClipboardTTL:       45 * time.Second,
		OutputFormat:       "json",
		ShowPasswords:      true,
		ConfirmDestructive: false,
		KDF: KDFConfig{
			Memory:      131072,
			Iterations:  4,
			Parallelism: 8,
		},
	}

	if cfg.VaultPath != "/custom/path" {
		t.Error("VaultPath mismatch")
	}
	if cfg.DefaultProfile != "work" {
		t.Error("DefaultProfile mismatch")
	}
	if cfg.AutoLockTTL != 2*time.Hour {
		t.Error("AutoLockTTL mismatch")
	}
	if cfg.ClipboardTTL != 45*time.Second {
		t.Error("ClipboardTTL mismatch")
	}
	if cfg.OutputFormat != "json" {
		t.Error("OutputFormat mismatch")
	}
	if !cfg.ShowPasswords {
		t.Error("ShowPasswords mismatch")
	}
	if cfg.ConfirmDestructive {
		t.Error("ConfirmDestructive mismatch")
	}
	if cfg.KDF.Memory != 131072 {
		t.Error("KDF.Memory mismatch")
	}
	if cfg.KDF.Iterations != 4 {
		t.Error("KDF.Iterations mismatch")
	}
	if cfg.KDF.Parallelism != 8 {
		t.Error("KDF.Parallelism mismatch")
	}
}
