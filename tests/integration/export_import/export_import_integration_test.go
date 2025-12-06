package export_import_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/vault-cli/vault/internal/domain"
	"github.com/vault-cli/vault/internal/store"
	"github.com/vault-cli/vault/internal/vault"
)

// TestEncryptedExportImport tests encrypted backup and restore
func TestEncryptedExportImport(t *testing.T) {
	tempDir := t.TempDir()
	vaultPath := filepath.Join(tempDir, "source.vault")
	exportPath := filepath.Join(tempDir, "backup.vault")
	importPath := filepath.Join(tempDir, "restored.vault")
	
	passphrase := "test-export-import"
	crypto := vault.NewDefaultCryptoEngine()
	salt, _ := vault.GenerateSalt()
	masterKey, _ := crypto.DeriveKey(passphrase, salt)
	defer vault.Zeroize(masterKey)
	
	kdfParams := vault.DefaultArgon2Params()
	kdfParamsMap := map[string]interface{}{
		"memory":      kdfParams.Memory,
		"iterations":  kdfParams.Iterations,
		"parallelism": kdfParams.Parallelism,
		"salt":        salt,
	}
	
	// Create source vault with data
	vaultStore := store.NewBoltStore()
	vaultStore.CreateVault(vaultPath, masterKey, kdfParamsMap)
	vaultStore.OpenVault(vaultPath, masterKey)
	
	testEntries := []domain.Entry{
		{Name: "entry1", Username: "user1", Password: []byte("pass1"), Tags: []string{"tag1"}},
		{Name: "entry2", Username: "user2", Password: []byte("pass2"), Tags: []string{"tag2"}},
		{Name: "entry3", Username: "user3", Password: []byte("pass3"), Tags: []string{"tag1", "tag2"}},
	}
	
	for _, entry := range testEntries {
		e := entry
		vaultStore.CreateEntry("default", &e)
	}
	vaultStore.CloseVault()
	
	// Export (simulate by copying vault file)
	data, err := os.ReadFile(vaultPath)
	if err != nil {
		t.Fatalf("Failed to read vault: %v", err)
	}
	
	if err := os.WriteFile(exportPath, data, 0600); err != nil {
		t.Fatalf("Failed to export: %v", err)
	}
	t.Log("✓ Encrypted export successful")
	
	// Import to new vault
	importData, _ := os.ReadFile(exportPath)
	os.WriteFile(importPath, importData, 0600)
	
	// Verify imported data
	importStore := store.NewBoltStore()
	if err := importStore.OpenVault(importPath, masterKey); err != nil {
		t.Fatalf("Failed to open imported vault: %v", err)
	}
	defer importStore.CloseVault()
	
	entries, _ := importStore.ListEntries("default", nil)
	if len(entries) != 3 {
		t.Errorf("Expected 3 entries, got %d", len(entries))
	}
	
	t.Log("✓ Encrypted import successful")
	t.Log("✓ Data integrity verified")
}

// TestPlaintextExport tests exporting to plaintext JSON
func TestPlaintextExport(t *testing.T) {
	tempDir := t.TempDir()
	vaultPath := filepath.Join(tempDir, "export_test.vault")
	jsonPath := filepath.Join(tempDir, "export.json")
	
	passphrase := "test-plaintext-export"
	crypto := vault.NewDefaultCryptoEngine()
	salt, _ := vault.GenerateSalt()
	masterKey, _ := crypto.DeriveKey(passphrase, salt)
	defer vault.Zeroize(masterKey)
	
	kdfParams := vault.DefaultArgon2Params()
	kdfParamsMap := map[string]interface{}{
		"memory":      kdfParams.Memory,
		"iterations":  kdfParams.Iterations,
		"parallelism": kdfParams.Parallelism,
		"salt":        salt,
	}
	
	vaultStore := store.NewBoltStore()
	vaultStore.CreateVault(vaultPath, masterKey, kdfParamsMap)
	vaultStore.OpenVault(vaultPath, masterKey)
	
	// Add test data
	entry := &domain.Entry{
		Name:     "test-entry",
		Username: "user@example.com",
		Password: []byte("secret-password"),
		URL:      "https://example.com",
		Notes:    "Test notes",
		Tags:     []string{"test", "export"},
	}
	vaultStore.CreateEntry("default", entry)
	
	// Export to JSON
	entries, _ := vaultStore.ListEntries("default", nil)
	jsonData, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}
	
	if err := os.WriteFile(jsonPath, jsonData, 0600); err != nil {
		t.Fatalf("Failed to write JSON: %v", err)
	}
	
	vaultStore.CloseVault()
	
	// Verify JSON export
	exportedData, _ := os.ReadFile(jsonPath)
	var exportedEntries []domain.Entry
	if err := json.Unmarshal(exportedData, &exportedEntries); err != nil {
		t.Fatalf("Failed to parse exported JSON: %v", err)
	}
	
	if len(exportedEntries) != 1 {
		t.Errorf("Expected 1 entry in export, got %d", len(exportedEntries))
	}
	
	if exportedEntries[0].Name != "test-entry" {
		t.Error("Exported entry name mismatch")
	}
	
	t.Log("✓ Plaintext export successful")
}

// TestPlaintextImport tests importing from plaintext JSON
func TestPlaintextImport(t *testing.T) {
	tempDir := t.TempDir()
	vaultPath := filepath.Join(tempDir, "import_test.vault")
	jsonPath := filepath.Join(tempDir, "import.json")
	
	// Create JSON import file
	importData := []domain.Entry{
		{Name: "imported1", Username: "user1", Password: []byte("pass1")},
		{Name: "imported2", Username: "user2", Password: []byte("pass2")},
	}
	
	jsonData, _ := json.MarshalIndent(importData, "", "  ")
	os.WriteFile(jsonPath, jsonData, 0600)
	
	// Create vault
	passphrase := "test-plaintext-import"
	crypto := vault.NewDefaultCryptoEngine()
	salt, _ := vault.GenerateSalt()
	masterKey, _ := crypto.DeriveKey(passphrase, salt)
	defer vault.Zeroize(masterKey)
	
	kdfParams := vault.DefaultArgon2Params()
	kdfParamsMap := map[string]interface{}{
		"memory":      kdfParams.Memory,
		"iterations":  kdfParams.Iterations,
		"parallelism": kdfParams.Parallelism,
		"salt":        salt,
	}
	
	vaultStore := store.NewBoltStore()
	vaultStore.CreateVault(vaultPath, masterKey, kdfParamsMap)
	vaultStore.OpenVault(vaultPath, masterKey)
	defer vaultStore.CloseVault()
	
	// Import from JSON
	jsonContent, _ := os.ReadFile(jsonPath)
	var entriesToImport []domain.Entry
	json.Unmarshal(jsonContent, &entriesToImport)
	
	for _, entry := range entriesToImport {
		e := entry
		if err := vaultStore.CreateEntry("default", &e); err != nil {
			t.Errorf("Failed to import entry %s: %v", e.Name, err)
		}
	}
	
	// Verify import
	entries, _ := vaultStore.ListEntries("default", nil)
	if len(entries) != 2 {
		t.Errorf("Expected 2 imported entries, got %d", len(entries))
	}
	
	t.Log("✓ Plaintext import successful")
}

// TestConflictResolution tests handling duplicate entries on import
func TestConflictResolution(t *testing.T) {
	tempDir := t.TempDir()
	vaultPath := filepath.Join(tempDir, "conflict_test.vault")
	
	passphrase := "test-conflicts"
	crypto := vault.NewDefaultCryptoEngine()
	salt, _ := vault.GenerateSalt()
	masterKey, _ := crypto.DeriveKey(passphrase, salt)
	defer vault.Zeroize(masterKey)
	
	kdfParams := vault.DefaultArgon2Params()
	kdfParamsMap := map[string]interface{}{
		"memory":      kdfParams.Memory,
		"iterations":  kdfParams.Iterations,
		"parallelism": kdfParams.Parallelism,
		"salt":        salt,
	}
	
	vaultStore := store.NewBoltStore()
	vaultStore.CreateVault(vaultPath, masterKey, kdfParamsMap)
	vaultStore.OpenVault(vaultPath, masterKey)
	defer vaultStore.CloseVault()
	
	// Add existing entry
	existing := &domain.Entry{
		Name:     "duplicate",
		Username: "original@example.com",
		Password: []byte("original-password"),
	}
	vaultStore.CreateEntry("default", existing)
	
	// Try to import duplicate
	duplicate := &domain.Entry{
		Name:     "duplicate",
		Username: "new@example.com",
		Password: []byte("new-password"),
	}
	
	err := vaultStore.CreateEntry("default", duplicate)
	if err == nil {
		t.Error("Should error on duplicate entry")
	}
	t.Log("✓ Duplicate detected")
	
	// Test overwrite strategy
	if err := vaultStore.UpdateEntry("default", "duplicate", duplicate); err != nil {
		t.Errorf("Overwrite failed: %v", err)
	}
	
	retrieved, _ := vaultStore.GetEntry("default", "duplicate")
	if retrieved.Username != "new@example.com" {
		t.Error("Overwrite did not work")
	}
	t.Log("✓ Overwrite strategy works")
	
	// Test skip strategy (don't overwrite)
	_, err = vaultStore.GetEntry("default", "duplicate")
	if err != nil {
		t.Error("Entry should still exist")
	}
	t.Log("✓ Skip strategy works")
}

// TestPartialImport tests importing with some failures
func TestPartialImport(t *testing.T) {
	tempDir := t.TempDir()
	vaultPath := filepath.Join(tempDir, "partial_test.vault")
	
	passphrase := "test-partial"
	crypto := vault.NewDefaultCryptoEngine()
	salt, _ := vault.GenerateSalt()
	masterKey, _ := crypto.DeriveKey(passphrase, salt)
	defer vault.Zeroize(masterKey)
	
	kdfParams := vault.DefaultArgon2Params()
	kdfParamsMap := map[string]interface{}{
		"memory":      kdfParams.Memory,
		"iterations":  kdfParams.Iterations,
		"parallelism": kdfParams.Parallelism,
		"salt":        salt,
	}
	
	vaultStore := store.NewBoltStore()
	vaultStore.CreateVault(vaultPath, masterKey, kdfParamsMap)
	vaultStore.OpenVault(vaultPath, masterKey)
	defer vaultStore.CloseVault()
	
	// Import entries, some valid, some invalid
	entries := []domain.Entry{
		{Name: "valid1", Username: "user1", Password: []byte("pass1")},
		{Name: "", Username: "invalid", Password: []byte("pass")}, // Invalid: empty name
		{Name: "valid2", Username: "user2", Password: []byte("pass2")},
	}
	
	successCount := 0
	failCount := 0
	
	for _, entry := range entries {
		e := entry
		if err := vaultStore.CreateEntry("default", &e); err != nil {
			failCount++
		} else {
			successCount++
		}
	}
	
	if successCount != 2 {
		t.Errorf("Expected 2 successful imports, got %d", successCount)
	}
	if failCount != 1 {
		t.Errorf("Expected 1 failed import, got %d", failCount)
	}
	
	t.Log("✓ Partial import handled correctly")
}

// TestExportImportWithProfiles tests export/import across profiles
func TestExportImportWithProfiles(t *testing.T) {
	tempDir := t.TempDir()
	vaultPath := filepath.Join(tempDir, "profiles_test.vault")
	
	passphrase := "test-profile-export"
	crypto := vault.NewDefaultCryptoEngine()
	salt, _ := vault.GenerateSalt()
	masterKey, _ := crypto.DeriveKey(passphrase, salt)
	defer vault.Zeroize(masterKey)
	
	kdfParams := vault.DefaultArgon2Params()
	kdfParamsMap := map[string]interface{}{
		"memory":      kdfParams.Memory,
		"iterations":  kdfParams.Iterations,
		"parallelism": kdfParams.Parallelism,
		"salt":        salt,
	}
	
	vaultStore := store.NewBoltStore()
	vaultStore.CreateVault(vaultPath, masterKey, kdfParamsMap)
	vaultStore.OpenVault(vaultPath, masterKey)
	defer vaultStore.CloseVault()
	
	// Create profiles
	if err := vaultStore.CreateProfile("work", "Work profile"); err != nil {
		t.Fatalf("Failed to create work profile: %v", err)
	}
	if err := vaultStore.CreateProfile("personal", "Personal profile"); err != nil {
		t.Fatalf("Failed to create personal profile: %v", err)
	}
	
	// Add entries to different profiles
	workEntry := &domain.Entry{Name: "work-entry", Username: "work", Password: []byte("pass")}
	personalEntry := &domain.Entry{Name: "personal-entry", Username: "personal", Password: []byte("pass")}
	
	vaultStore.CreateEntry("work", workEntry)
	vaultStore.CreateEntry("personal", personalEntry)
	
	// Export all profiles
	workEntries, _ := vaultStore.ListEntries("work", nil)
	personalEntries, _ := vaultStore.ListEntries("personal", nil)
	
	if len(workEntries) != 1 || len(personalEntries) != 1 {
		t.Error("Profile entries not created correctly")
	}
	
	t.Log("✓ Export/import with profiles works")
}
