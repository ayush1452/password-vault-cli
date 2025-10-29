package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vault-cli/vault/internal/domain"
	"github.com/vault-cli/vault/internal/vault"
)

func TestBoltStore_CreateAndOpenVault(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "vault_test_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	vaultPath := filepath.Join(tempDir, "test.vault")

	// Generate master key
	salt, err := vault.GenerateSalt()
	if err != nil {
		t.Fatalf("Failed to generate salt: %v", err)
	}

	crypto := vault.NewDefaultCryptoEngine()
	masterKey, err := crypto.DeriveKey("test-passphrase", salt)
	if err != nil {
		t.Fatalf("Failed to derive key: %v", err)
	}
	defer vault.Zeroize(masterKey)

	// Create KDF params
	kdfParams := map[string]interface{}{
		"memory":      uint32(1024),
		"iterations":  uint32(1),
		"parallelism": uint8(1),
	}

	// Test vault creation
	store := NewBoltStore()
	err = store.CreateVault(vaultPath, masterKey, kdfParams)
	if err != nil {
		t.Fatalf("Failed to create vault: %v", err)
	}

	// Verify vault file exists and has correct permissions
	info, err := os.Stat(vaultPath)
	if err != nil {
		t.Fatalf("Vault file not created: %v", err)
	}

	if info.Mode().Perm() != 0600 {
		t.Errorf("Incorrect file permissions: got %o, want 0600", info.Mode().Perm())
	}

	// Test opening the vault
	err = store.OpenVault(vaultPath, masterKey)
	if err != nil {
		t.Fatalf("Failed to open vault: %v", err)
	}

	if !store.IsOpen() {
		t.Error("Vault should be open")
	}

	// Test vault metadata
	metadata, err := store.GetVaultMetadata()
	if err != nil {
		t.Fatalf("Failed to get vault metadata: %v", err)
	}

	if metadata.Version != "1.0.0" {
		t.Errorf("Incorrect version: got %s, want 1.0.0", metadata.Version)
	}

	// Test default profile exists
	profiles, err := store.ListProfiles()
	if err != nil {
		t.Fatalf("Failed to list profiles: %v", err)
	}

	if len(profiles) != 1 || profiles[0].Name != "default" {
		t.Error("Default profile not created")
	}

	// Close vault
	err = store.CloseVault()
	if err != nil {
		t.Fatalf("Failed to close vault: %v", err)
	}

	if store.IsOpen() {
		t.Error("Vault should be closed")
	}
}

func TestBoltStore_EntryOperations(t *testing.T) {
	// Setup vault
	tempDir, err := os.MkdirTemp("", "vault_test_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	vaultPath := filepath.Join(tempDir, "test.vault")
	store := NewBoltStore()

	// Create and open vault
	salt, _ := vault.GenerateSalt()
	crypto := vault.NewDefaultCryptoEngine()
	masterKey, _ := crypto.DeriveKey("test-passphrase", salt)
	defer vault.Zeroize(masterKey)

	kdfParams := map[string]interface{}{
		"memory":      uint32(1024),
		"iterations":  uint32(1),
		"parallelism": uint8(1),
	}

	store.CreateVault(vaultPath, masterKey, kdfParams)
	store.OpenVault(vaultPath, masterKey)
	defer store.CloseVault()

	// Test entry creation
	entry := &domain.Entry{
		Name:     "github",
		Username: "user@example.com",
		Secret:   []byte("password123"),
		URL:      "https://github.com",
		Notes:    "Personal GitHub account",
		Tags:     []string{"work", "git"},
	}

	err = store.CreateEntry("default", entry)
	if err != nil {
		t.Fatalf("Failed to create entry: %v", err)
	}

	// Test entry exists
	if !store.EntryExists("default", "github") {
		t.Error("Entry should exist")
	}

	// Test entry retrieval
	retrievedEntry, err := store.GetEntry("default", "github")
	if err != nil {
		t.Fatalf("Failed to get entry: %v", err)
	}

	if retrievedEntry.Name != entry.Name {
		t.Errorf("Name mismatch: got %s, want %s", retrievedEntry.Name, entry.Name)
	}
	if retrievedEntry.Username != entry.Username {
		t.Errorf("Username mismatch: got %s, want %s", retrievedEntry.Username, entry.Username)
	}
	if string(retrievedEntry.Secret) != string(entry.Secret) {
		t.Errorf("Secret mismatch")
	}

	// Test entry listing
	entries, err := store.ListEntries("default", nil)
	if err != nil {
		t.Fatalf("Failed to list entries: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(entries))
	}

	// Note: Listed entries should have secrets cleared for security
	if entries[0].Secret != nil {
		t.Error("Listed entry should not contain secret")
	}

	// Test entry update
	entry.Notes = "Updated notes"
	err = store.UpdateEntry("default", "github", entry)
	if err != nil {
		t.Fatalf("Failed to update entry: %v", err)
	}

	updatedEntry, err := store.GetEntry("default", "github")
	if err != nil {
		t.Fatalf("Failed to get updated entry: %v", err)
	}

	if updatedEntry.Notes != "Updated notes" {
		t.Errorf("Notes not updated: got %s, want %s", updatedEntry.Notes, "Updated notes")
	}

	// Test entry deletion
	err = store.DeleteEntry("default", "github")
	if err != nil {
		t.Fatalf("Failed to delete entry: %v", err)
	}

	if store.EntryExists("default", "github") {
		t.Error("Entry should not exist after deletion")
	}

	// Test getting non-existent entry
	_, err = store.GetEntry("default", "github")
	if err != ErrEntryNotFound {
		t.Errorf("Expected ErrEntryNotFound, got %v", err)
	}
}

func TestBoltStore_ProfileOperations(t *testing.T) {
	// Setup vault
	tempDir, err := os.MkdirTemp("", "vault_test_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	vaultPath := filepath.Join(tempDir, "test.vault")
	store := NewBoltStore()

	// Create and open vault
	salt, _ := vault.GenerateSalt()
	crypto := vault.NewDefaultCryptoEngine()
	masterKey, _ := crypto.DeriveKey("test-passphrase", salt)
	defer vault.Zeroize(masterKey)

	kdfParams := map[string]interface{}{
		"memory":      uint32(1024),
		"iterations":  uint32(1),
		"parallelism": uint8(1),
	}

	store.CreateVault(vaultPath, masterKey, kdfParams)
	store.OpenVault(vaultPath, masterKey)
	defer store.CloseVault()

	// Test profile creation
	err = store.CreateProfile("production", "Production environment")
	if err != nil {
		t.Fatalf("Failed to create profile: %v", err)
	}

	// Test profile exists
	if !store.ProfileExists("production") {
		t.Error("Profile should exist")
	}

	// Test profile retrieval
	profile, err := store.GetProfile("production")
	if err != nil {
		t.Fatalf("Failed to get profile: %v", err)
	}

	if profile.Name != "production" {
		t.Errorf("Name mismatch: got %s, want production", profile.Name)
	}
	if profile.Description != "Production environment" {
		t.Errorf("Description mismatch: got %s, want Production environment", profile.Description)
	}

	// Test profile listing
	profiles, err := store.ListProfiles()
	if err != nil {
		t.Fatalf("Failed to list profiles: %v", err)
	}

	if len(profiles) != 2 { // default + production
		t.Errorf("Expected 2 profiles, got %d", len(profiles))
	}

	// Test creating entry in new profile
	entry := &domain.Entry{
		Name:     "prod-db",
		Username: "admin",
		Secret:   []byte("prod-password"),
		URL:      "db.prod.example.com",
	}

	err = store.CreateEntry("production", entry)
	if err != nil {
		t.Fatalf("Failed to create entry in production profile: %v", err)
	}

	// Verify entry exists in production profile
	if !store.EntryExists("production", "prod-db") {
		t.Error("Entry should exist in production profile")
	}

	// Verify entry doesn't exist in default profile
	if store.EntryExists("default", "prod-db") {
		t.Error("Entry should not exist in default profile")
	}

	// Test profile deletion
	err = store.DeleteProfile("production")
	if err != nil {
		t.Fatalf("Failed to delete profile: %v", err)
	}

	if store.ProfileExists("production") {
		t.Error("Profile should not exist after deletion")
	}

	// Test cannot delete default profile
	err = store.DeleteProfile("default")
	if err == nil {
		t.Error("Should not be able to delete default profile")
	}
}

func TestBoltStore_FilteredListing(t *testing.T) {
	// Setup vault with test entries
	tempDir, err := os.MkdirTemp("", "vault_test_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	vaultPath := filepath.Join(tempDir, "test.vault")
	store := NewBoltStore()

	salt, _ := vault.GenerateSalt()
	crypto := vault.NewDefaultCryptoEngine()
	masterKey, _ := crypto.DeriveKey("test-passphrase", salt)
	defer vault.Zeroize(masterKey)

	kdfParams := map[string]interface{}{
		"memory":      uint32(1024),
		"iterations":  uint32(1),
		"parallelism": uint8(1),
	}

	store.CreateVault(vaultPath, masterKey, kdfParams)
	store.OpenVault(vaultPath, masterKey)
	defer store.CloseVault()

	// Create test entries
	entries := []*domain.Entry{
		{
			Name:     "github",
			Username: "user@example.com",
			Secret:   []byte("password1"),
			Tags:     []string{"work", "git"},
		},
		{
			Name:     "gitlab",
			Username: "user@example.com",
			Secret:   []byte("password2"),
			Tags:     []string{"work", "git"},
		},
		{
			Name:     "personal-email",
			Username: "personal@example.com",
			Secret:   []byte("password3"),
			Tags:     []string{"personal", "email"},
		},
	}

	for _, entry := range entries {
		store.CreateEntry("default", entry)
	}

	// Test search filter
	searchFilter := &domain.Filter{Search: "git"}
	results, err := store.ListEntries("default", searchFilter)
	if err != nil {
		t.Fatalf("Failed to list entries with search filter: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 entries with 'git' in name, got %d", len(results))
	}

	// Test tag filter
	tagFilter := &domain.Filter{Tags: []string{"personal"}}
	results, err = store.ListEntries("default", tagFilter)
	if err != nil {
		t.Fatalf("Failed to list entries with tag filter: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 entry with 'personal' tag, got %d", len(results))
	}

	// Test combined filter
	combinedFilter := &domain.Filter{
		Search: "example.com",
		Tags:   []string{"work"},
	}
	results, err = store.ListEntries("default", combinedFilter)
	if err != nil {
		t.Fatalf("Failed to list entries with combined filter: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 entries matching combined filter, got %d", len(results))
	}
}

func TestBoltStore_ErrorCases(t *testing.T) {
	store := NewBoltStore()

	// Test operations on closed vault
	err := store.CreateEntry("default", &domain.Entry{Name: "test"})
	if err == nil {
		t.Error("Should fail when vault is not open")
	}

	// Test opening non-existent vault
	err = store.OpenVault("/non/existent/path", []byte("key"))
	if err != ErrVaultNotFound {
		t.Errorf("Expected ErrVaultNotFound, got %v", err)
	}

	// Test creating vault that already exists
	tempDir, _ := os.MkdirTemp("", "vault_test_")
	defer os.RemoveAll(tempDir)

	vaultPath := filepath.Join(tempDir, "test.vault")

	// Create file first
	file, _ := os.Create(vaultPath)
	file.Close()

	err = store.CreateVault(vaultPath, []byte("key"), map[string]interface{}{})
	if err != ErrVaultExists {
		t.Errorf("Expected ErrVaultExists, got %v", err)
	}
}

func TestFileLock(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "lock_test_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	lockPath := filepath.Join(tempDir, "test.lock")

	// Test basic locking
	lock1 := NewFileLock(lockPath)
	err = lock1.Lock(1 * time.Second)
	if err != nil {
		t.Fatalf("Failed to acquire lock: %v", err)
	}

	if !lock1.IsLocked() {
		t.Error("Lock should be held")
	}

	// Test lock contention
	lock2 := NewFileLock(lockPath)
	err = lock2.Lock(100 * time.Millisecond)
	if err != ErrLockTimeout {
		t.Errorf("Expected timeout error, got %v", err)
	}

	// Release first lock
	err = lock1.Unlock()
	if err != nil {
		t.Fatalf("Failed to release lock: %v", err)
	}

	if lock1.IsLocked() {
		t.Error("Lock should be released")
	}

	// Now second lock should succeed
	err = lock2.Lock(1 * time.Second)
	if err != nil {
		t.Fatalf("Failed to acquire lock after release: %v", err)
	}

	lock2.Unlock()
}

func TestAtomicWriter(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "atomic_test_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	targetPath := filepath.Join(tempDir, "test.txt")

	// Test successful write
	writer, err := NewAtomicWriter(targetPath)
	if err != nil {
		t.Fatalf("Failed to create atomic writer: %v", err)
	}

	testData := []byte("Hello, World!")
	_, err = writer.Write(testData)
	if err != nil {
		t.Fatalf("Failed to write data: %v", err)
	}

	err = writer.Commit()
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Verify file exists and contains correct data
	data, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("Failed to read target file: %v", err)
	}

	if string(data) != string(testData) {
		t.Errorf("File content mismatch: got %s, want %s", string(data), string(testData))
	}

	// Test abort
	writer2, err := NewAtomicWriter(targetPath + ".2")
	if err != nil {
		t.Fatalf("Failed to create second atomic writer: %v", err)
	}

	writer2.Write([]byte("This should be aborted"))
	err = writer2.Abort()
	if err != nil {
		t.Fatalf("Failed to abort: %v", err)
	}

	// Verify target file doesn't exist
	if _, err := os.Stat(targetPath + ".2"); !os.IsNotExist(err) {
		t.Error("Aborted file should not exist")
	}
}
