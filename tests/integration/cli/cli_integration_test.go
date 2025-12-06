package cli_test

import (
	"path/filepath"
	"testing"

	"github.com/vault-cli/vault/internal/domain"
	"github.com/vault-cli/vault/internal/store"
	"github.com/vault-cli/vault/internal/vault"
)

// TestAddGetUpdateDeleteFlow tests the complete CRUD flow
func TestAddGetUpdateDeleteFlow(t *testing.T) {
	tempDir := t.TempDir()
	vaultPath := filepath.Join(tempDir, "crud_test.vault")
	
	// Setup vault
	passphrase := "test-crud-flow"
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
	if err := vaultStore.CreateVault(vaultPath, masterKey, kdfParamsMap); err != nil {
		t.Fatalf("Failed to create vault: %v", err)
	}
	
	if err := vaultStore.OpenVault(vaultPath, masterKey); err != nil {
		t.Fatalf("Failed to open vault: %v", err)
	}
	defer vaultStore.CloseVault()
	
	// Add
	entry := &domain.Entry{
		Name:     "github",
		Username: "user@example.com",
		Password: []byte("initial-password"),
		URL:      "https://github.com",
		Notes:    "Work account",
		Tags:     []string{"work", "git"},
	}
	
	if err := vaultStore.CreateEntry("default", entry); err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	t.Log("✓ Add successful")
	
	// Get
	retrieved, err := vaultStore.GetEntry("default", "github")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if retrieved.Username != "user@example.com" {
		t.Errorf("Get returned wrong data: %s", retrieved.Username)
	}
	t.Log("✓ Get successful")
	
	// Update
	updated := &domain.Entry{
		Name:     "github",
		Username: "newemail@example.com",
		Password: []byte("updated-password"),
		URL:      "https://github.com",
		Notes:    "Updated work account",
		Tags:     []string{"work", "git", "updated"},
	}
	
	if err := vaultStore.UpdateEntry("default", "github", updated); err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	
	retrieved, _ = vaultStore.GetEntry("default", "github")
	if retrieved.Username != "newemail@example.com" {
		t.Error("Update did not persist")
	}
	t.Log("✓ Update successful")
	
	// Delete
	if err := vaultStore.DeleteEntry("default", "github"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	
	_, err = vaultStore.GetEntry("default", "github")
	if err == nil {
		t.Error("Entry should be deleted")
	}
	t.Log("✓ Delete successful")
	
	t.Log("✓ Complete CRUD flow verified")
}

// TestSearchAndFilter tests search and filtering functionality
func TestSearchAndFilter(t *testing.T) {
	tempDir := t.TempDir()
	vaultPath := filepath.Join(tempDir, "search_test.vault")
	
	// Setup vault
	passphrase := "test-search"
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
	
	// Add test data
	entries := []struct {
		name string
		tags []string
	}{
		{"github", []string{"work", "git"}},
		{"gitlab", []string{"work", "git"}},
		{"personal-email", []string{"personal", "email"}},
		{"work-email", []string{"work", "email"}},
	}
	
	for _, e := range entries {
		entry := &domain.Entry{
			Name:     e.name,
			Username: "user",
			Password: []byte("pass"),
			Tags:     e.tags,
		}
		vaultStore.CreateEntry("default", entry)
	}
	
	// Filter by tag
	filter := &domain.Filter{Tags: []string{"work"}}
	workEntries, _ := vaultStore.ListEntries("default", filter)
	
	if len(workEntries) != 3 {
		t.Errorf("Expected 3 work entries, got %d", len(workEntries))
	}
	t.Log("✓ Tag filtering works")
	
	// Search
	searchFilter := &domain.Filter{Search: "git"}
	gitEntries, _ := vaultStore.ListEntries("default", searchFilter)
	
	if len(gitEntries) < 2 {
		t.Errorf("Expected at least 2 git entries, got %d", len(gitEntries))
	}
	t.Log("✓ Search works")
}

// TestErrorHandlingChain tests error propagation
func TestErrorHandlingChain(t *testing.T) {
	tempDir := t.TempDir()
	vaultPath := filepath.Join(tempDir, "error_test.vault")
	
	passphrase := "test-errors"
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
	
	// Access without opening
	_, err := vaultStore.GetEntry("default", "test")
	if err == nil {
		t.Error("Should error when vault not open")
	}
	t.Log("✓ Error on closed vault")
	
	vaultStore.OpenVault(vaultPath, masterKey)
	defer vaultStore.CloseVault()
	
	// Get non-existent entry
	_, err = vaultStore.GetEntry("default", "nonexistent")
	if err == nil {
		t.Error("Should error on non-existent entry")
	}
	t.Log("✓ Error on non-existent entry")
	
	// Add duplicate
	entry := &domain.Entry{Name: "duplicate", Username: "user", Password: []byte("pass")}
	vaultStore.CreateEntry("default", entry)
	
	err = vaultStore.CreateEntry("default", entry)
	if err == nil {
		t.Error("Should error on duplicate entry")
	}
	t.Log("✓ Error on duplicate entry")
	
	// Update non-existent
	err = vaultStore.UpdateEntry("default", "nonexistent", entry)
	if err == nil {
		t.Error("Should error updating non-existent entry")
	}
	t.Log("✓ Error on update non-existent")
	
	// Delete non-existent
	err = vaultStore.DeleteEntry("default", "nonexistent")
	if err == nil {
		t.Error("Should error deleting non-existent entry")
	}
	t.Log("✓ Error on delete non-existent")
}

// TestBulkOperations tests bulk entry operations
func TestBulkOperations(t *testing.T) {
	tempDir := t.TempDir()
	vaultPath := filepath.Join(tempDir, "bulk_test.vault")
	
	passphrase := "test-bulk"
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
	
	// Add 100 entries
	for i := 0; i < 100; i++ {
		entry := &domain.Entry{
			Name:     string(rune('a' + i%26)) + string(rune('0' + i/26)),
			Username: "user",
			Password: []byte("pass"),
		}
		if err := vaultStore.CreateEntry("default", entry); err != nil {
			t.Fatalf("Failed to add entry %d: %v", i, err)
		}
	}
	
	// List all
	entries, _ := vaultStore.ListEntries("default", nil)
	if len(entries) != 100 {
		t.Errorf("Expected 100 entries, got %d", len(entries))
	}
	
	t.Log("✓ Bulk operations successful")
}

// TestTagManagement tests tag operations
func TestTagManagement(t *testing.T) {
	tempDir := t.TempDir()
	vaultPath := filepath.Join(tempDir, "tags_test.vault")
	
	passphrase := "test-tags"
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
	
	// Add entry with tags
	entry := &domain.Entry{
		Name:     "test",
		Username: "user",
		Password: []byte("pass"),
		Tags:     []string{"tag1", "tag2", "tag3"},
	}
	vaultStore.CreateEntry("default", entry)
	
	// Update tags
	updated := &domain.Entry{
		Name:     "test",
		Username: "user",
		Password: []byte("pass"),
		Tags:     []string{"tag2", "tag4"},
	}
	vaultStore.UpdateEntry("default", "test", updated)
	
	retrieved, _ := vaultStore.GetEntry("default", "test")
	if len(retrieved.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(retrieved.Tags))
	}
	
	t.Log("✓ Tag management works")
}
