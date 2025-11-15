package store

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/vault-cli/vault/internal/domain"
	"github.com/vault-cli/vault/internal/vault"
)

// TestStoreIntegration tests comprehensive store operations
func TestStoreIntegration(t *testing.T) {
	tempDir := t.TempDir()
	vaultPath := filepath.Join(tempDir, "integration_test.vault")

	crypto := vault.NewDefaultCryptoEngine()
	passphrase := "integration-test-passphrase"

	// Derive master key from passphrase
	salt, err := vault.GenerateSalt()
	if err != nil {
		t.Fatalf("Failed to generate salt: %v", err)
	}
	masterKey, err := crypto.DeriveKey(passphrase, salt)
	if err != nil {
		t.Fatalf("Failed to derive master key: %v", err)
	}

	// Test initialization
	store := NewBoltStore()
	defer store.CloseVault()

	// Create vault with KDF params
	kdfParams := map[string]interface{}{
		"algorithm": "argon2id",
		"time":      1,
		"memory":    64 * 1024,
		"threads":   4,
	}
	err = store.CreateVault(vaultPath, masterKey, kdfParams)
	if err != nil {
		t.Fatalf("Failed to create vault: %v", err)
	}

	// Open the vault
	err = store.OpenVault(vaultPath, masterKey)
	if err != nil {
		t.Fatalf("Failed to open vault: %v", err)
	}

	// Test adding entries
	entries := []*domain.Entry{
		{
			Name:     "github.com",
			URL:      "https://github.com",
			Username: "testuser",
			Password: []byte("secret123"),
			Notes:    "Development account",
			Tags:     []string{"development", "git"},
		},
		{
			Name:     "gmail.com",
			URL:      "https://gmail.com",
			Username: "test@gmail.com",
			Password: []byte("email-password"),
			Notes:    "Personal email",
			Tags:     []string{"email", "personal"},
		},
		{
			Name:     "aws.amazon.com",
			URL:      "https://aws.amazon.com",
			Username: "aws-user",
			Password: []byte("aws-secret-key"),
			Notes:    "AWS console access",
			Tags:     []string{"cloud", "aws"},
		},
	}

	profile := "default"
	for _, entry := range entries {
		err = store.CreateEntry(profile, entry)
		if err != nil {
			t.Errorf("Failed to add entry %s: %v", entry.Name, err)
		}
	}

	// Test listing entries
	allEntries, err := store.ListEntries(profile, nil)
	if err != nil {
		t.Fatalf("Failed to list entries: %v", err)
	}

	if len(allEntries) != len(entries) {
		t.Errorf("Expected %d entries, got %d", len(entries), len(allEntries))
	}

	// Test getting specific entries
	for _, originalEntry := range entries {
		retrievedEntry, err := store.GetEntry(profile, originalEntry.Name)
		if err != nil {
			t.Errorf("Failed to get entry %s: %v", originalEntry.Name, err)
			continue
		}

		if retrievedEntry.Name != originalEntry.Name {
			t.Errorf("Entry name mismatch: expected %s, got %s", originalEntry.Name, retrievedEntry.Name)
		}
		if retrievedEntry.Username != originalEntry.Username {
			t.Errorf("Username mismatch for %s: expected %s, got %s", originalEntry.Name, originalEntry.Username, retrievedEntry.Username)
		}
		// Password is stored as []byte, use bytes.Equal for comparison
		if !bytes.Equal(retrievedEntry.Password, originalEntry.Password) {
			t.Errorf("Password mismatch for %s", originalEntry.Name)
		}
	}

	// Test updating entries
	updateEntry := &domain.Entry{
		Name:     "github.com",
		URL:      "https://github.com",
		Username: "updated-user",
		Password: []byte("updated-password"),
		Notes:    "Updated development account",
		Tags:     []string{"development", "git", "updated"},
	}

	err = store.UpdateEntry(profile, "github.com", updateEntry)
	if err != nil {
		t.Errorf("Failed to update entry: %v", err)
	}

	// Verify update
	updatedEntry, err := store.GetEntry(profile, "github.com")
	if err != nil {
		t.Errorf("Failed to get updated entry: %v", err)
	} else {
		if updatedEntry.Username != "updated-user" {
			t.Errorf("Entry was not updated properly")
		}
	}

	// Test search functionality
	searchFilter := &domain.Filter{
		Tags: []string{"git"},
	}
	searchResults, err := store.ListEntries(profile, searchFilter)
	if err != nil {
		t.Errorf("Failed to search entries: %v", err)
	}

	if len(searchResults) == 0 {
		t.Error("Search should return results for 'git' tag")
	}

	// Test deleting entries
	err = store.DeleteEntry(profile, "aws.amazon.com")
	if err != nil {
		t.Errorf("Failed to delete entry: %v", err)
	}

	// Verify deletion
	_, err = store.GetEntry(profile, "aws.amazon.com")
	if err == nil {
		t.Error("Entry should have been deleted")
	}

	// Test close and reopen cycle
	err = store.CloseVault()
	if err != nil {
		t.Errorf("Failed to close vault: %v", err)
	}

	// Reopen the vault
	err = store.OpenVault(vaultPath, masterKey)
	if err != nil {
		t.Errorf("Failed to reopen vault: %v", err)
	}

	// Should be able to access entries again
	finalEntries, err := store.ListEntries(profile, nil)
	if err != nil {
		t.Errorf("Failed to list entries after reopen: %v", err)
	}

	if len(finalEntries) != 2 { // Should have 2 entries left after deletion
		t.Errorf("Expected 2 entries after deletion, got %d", len(finalEntries))
	}
}

// TestConcurrentAccess tests concurrent store operations
func TestConcurrentAccess(t *testing.T) {
	tempDir := t.TempDir()
	vaultPath := filepath.Join(tempDir, "concurrent_test.vault")

	crypto := vault.NewDefaultCryptoEngine()
	passphrase := "concurrent-test-passphrase"

	// Derive master key
	salt, err := vault.GenerateSalt()
	if err != nil {
		t.Fatalf("Failed to generate salt: %v", err)
	}
	masterKey, err := crypto.DeriveKey(passphrase, salt)
	if err != nil {
		t.Fatalf("Failed to derive master key: %v", err)
	}

	// Initialize store
	store := NewBoltStore()
	defer store.CloseVault()

	kdfParams := map[string]interface{}{
		"algorithm": "argon2id",
		"time":      1,
		"memory":    64 * 1024,
		"threads":   4,
	}
	err = store.CreateVault(vaultPath, masterKey, kdfParams)
	if err != nil {
		t.Fatalf("Failed to create vault: %v", err)
	}

	err = store.OpenVault(vaultPath, masterKey)
	if err != nil {
		t.Fatalf("Failed to open vault: %v", err)
	}

	// Test concurrent writes
	profile := "default"
	var wg sync.WaitGroup
	numGoroutines := 10
	entriesPerGoroutine := 5

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < entriesPerGoroutine; j++ {
				entry := &domain.Entry{
					Name:     fmt.Sprintf("entry-%d-%d", goroutineID, j),
					Username: fmt.Sprintf("user-%d-%d", goroutineID, j),
					Password: []byte(fmt.Sprintf("pass-%d-%d", goroutineID, j)),
					Notes:    fmt.Sprintf("Concurrent entry %d-%d", goroutineID, j),
				}

				err := store.CreateEntry(profile, entry)
				if err != nil {
					t.Errorf("Failed to add entry %s: %v", entry.Name, err)
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify all entries were added
	entries, err := store.ListEntries(profile, nil)
	if err != nil {
		t.Fatalf("Failed to list entries: %v", err)
	}

	expectedCount := numGoroutines * entriesPerGoroutine
	if len(entries) != expectedCount {
		t.Errorf("Expected %d entries, got %d", expectedCount, len(entries))
	}

	// Test concurrent reads
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < entriesPerGoroutine; j++ {
				entryName := fmt.Sprintf("entry-%d-%d", goroutineID, j)
				_, err := store.GetEntry(profile, entryName)
				if err != nil {
					t.Errorf("Failed to get entry %s: %v", entryName, err)
				}
			}
		}(i)
	}

	wg.Wait()
}

// TestStoreRecovery tests store recovery from corruption
func TestStoreRecovery(t *testing.T) {
	tempDir := t.TempDir()
	vaultPath := filepath.Join(tempDir, "recovery_test.vault")

	crypto := vault.NewDefaultCryptoEngine()
	passphrase := "recovery-test-passphrase"

	// Derive master key
	salt, err := vault.GenerateSalt()
	if err != nil {
		t.Fatalf("Failed to generate salt: %v", err)
	}
	masterKey, err := crypto.DeriveKey(passphrase, salt)
	if err != nil {
		t.Fatalf("Failed to derive master key: %v", err)
	}

	// Create and populate store
	store := NewBoltStore()

	kdfParams := map[string]interface{}{
		"algorithm": "argon2id",
		"time":      1,
		"memory":    64 * 1024,
		"threads":   4,
	}
	err = store.CreateVault(vaultPath, masterKey, kdfParams)
	if err != nil {
		t.Fatalf("Failed to create vault: %v", err)
	}

	err = store.OpenVault(vaultPath, masterKey)
	if err != nil {
		t.Fatalf("Failed to open vault: %v", err)
	}

	// Add test data
	profile := "default"
	testEntry := &domain.Entry{
		Name:     "recovery-test",
		Username: "testuser",
		Password: []byte("testpass"),
		Notes:    "Recovery test entry",
	}

	err = store.CreateEntry(profile, testEntry)
	if err != nil {
		t.Fatalf("Failed to add test entry: %v", err)
	}

	store.CloseVault()

	// Simulate file corruption by truncating
	file, err := os.OpenFile(vaultPath, os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("Failed to open vault file: %v", err)
	}

	err = file.Truncate(100) // Truncate to corrupt the file
	file.Close()
	if err != nil {
		t.Fatalf("Failed to truncate file: %v", err)
	}

	// Try to open corrupted store
	corruptedStore := NewBoltStore()
	err = corruptedStore.OpenVault(vaultPath, masterKey)
	if err == nil {
		corruptedStore.CloseVault()
		t.Error("Should have failed to open corrupted store")
	}

	// Test backup restoration (if backup exists)
	backupPath := vaultPath + ".backup"
	if _, err := os.Stat(backupPath); err == nil {
		// Restore from backup
		err = os.Rename(backupPath, vaultPath)
		if err != nil {
			t.Fatalf("Failed to restore from backup: %v", err)
		}

		// Try to open restored store
		restoredStore := NewBoltStore()
		defer restoredStore.CloseVault()

		err = restoredStore.OpenVault(vaultPath, masterKey)
		if err != nil {
			t.Fatalf("Failed to open restored store: %v", err)
		}

		// Verify data integrity
		entry, err := restoredStore.GetEntry(profile, "recovery-test")
		if err != nil {
			t.Errorf("Failed to get entry from restored store: %v", err)
		} else if entry.Username != "testuser" {
			t.Error("Data corruption detected in restored store")
		}
	}
}

// TestStorePersistence tests data persistence across sessions
func TestStorePersistence(t *testing.T) {
	tempDir := t.TempDir()
	vaultPath := filepath.Join(tempDir, "persistence_test.vault")

	crypto := vault.NewDefaultCryptoEngine()
	passphrase := "persistence-test-passphrase"

	// Derive master key
	salt, err := vault.GenerateSalt()
	if err != nil {
		t.Fatalf("Failed to generate salt: %v", err)
	}
	masterKey, err := crypto.DeriveKey(passphrase, salt)
	if err != nil {
		t.Fatalf("Failed to derive master key: %v", err)
	}

	profile := "default"

	// Session 1: Create and populate store
	{
		store := NewBoltStore()

		kdfParams := map[string]interface{}{
			"algorithm": "argon2id",
			"time":      1,
			"memory":    64 * 1024,
			"threads":   4,
		}
		err := store.CreateVault(vaultPath, masterKey, kdfParams)
		if err != nil {
			t.Fatalf("Failed to create vault: %v", err)
		}

		err = store.OpenVault(vaultPath, masterKey)
		if err != nil {
			t.Fatalf("Failed to open vault: %v", err)
		}

		entries := []*domain.Entry{
			{Name: "persistent1", Username: "user1", Password: []byte("pass1")},
			{Name: "persistent2", Username: "user2", Password: []byte("pass2")},
			{Name: "persistent3", Username: "user3", Password: []byte("pass3")},
		}

		for _, entry := range entries {
			err = store.CreateEntry(profile, entry)
			if err != nil {
				t.Errorf("Failed to add entry %s: %v", entry.Name, err)
			}
		}

		store.CloseVault()
	}

	// Session 2: Reopen and verify data
	{
		store := NewBoltStore()
		defer store.CloseVault()

		err := store.OpenVault(vaultPath, masterKey)
		if err != nil {
			t.Fatalf("Failed to reopen vault: %v", err)
		}

		entries, err := store.ListEntries(profile, nil)
		if err != nil {
			t.Fatalf("Failed to list entries in reopened store: %v", err)
		}

		if len(entries) != 3 {
			t.Errorf("Expected 3 persistent entries, got %d", len(entries))
		}

		// Verify specific entries
		for i := 1; i <= 3; i++ {
			entryName := fmt.Sprintf("persistent%d", i)
			entry, err := store.GetEntry(profile, entryName)
			if err != nil {
				t.Errorf("Failed to get persistent entry %s: %v", entryName, err)
				continue
			}

			expectedUsername := fmt.Sprintf("user%d", i)
			if entry.Username != expectedUsername {
				t.Errorf("Username mismatch for %s: expected %s, got %s", entryName, expectedUsername, entry.Username)
			}
		}
	}
}

// TestStoreMetrics tests performance metrics collection
func TestStoreMetrics(t *testing.T) {
	tempDir := t.TempDir()
	vaultPath := filepath.Join(tempDir, "metrics_test.vault")

	crypto := vault.NewDefaultCryptoEngine()
	passphrase := "metrics-test-passphrase"

	// Derive master key
	salt, err := vault.GenerateSalt()
	if err != nil {
		t.Fatalf("Failed to generate salt: %v", err)
	}
	masterKey, err := crypto.DeriveKey(passphrase, salt)
	if err != nil {
		t.Fatalf("Failed to derive master key: %v", err)
	}

	store := NewBoltStore()
	defer store.CloseVault()

	kdfParams := map[string]interface{}{
		"algorithm": "argon2id",
		"time":      1,
		"memory":    64 * 1024,
		"threads":   4,
	}
	err = store.CreateVault(vaultPath, masterKey, kdfParams)
	if err != nil {
		t.Fatalf("Failed to create vault: %v", err)
	}

	err = store.OpenVault(vaultPath, masterKey)
	if err != nil {
		t.Fatalf("Failed to open vault: %v", err)
	}

	// Measure add operation performance
	profile := "default"
	numEntries := 100
	start := time.Now()

	for i := 0; i < numEntries; i++ {
		entry := &domain.Entry{
			Name:     fmt.Sprintf("perf-entry-%d", i),
			Username: fmt.Sprintf("user-%d", i),
			Password: []byte(fmt.Sprintf("password-%d", i)),
			Notes:    fmt.Sprintf("Performance test entry %d", i),
		}

		err = store.CreateEntry(profile, entry)
		if err != nil {
			t.Errorf("Failed to add entry %d: %v", i, err)
		}
	}

	addDuration := time.Since(start)
	addPerEntry := addDuration / time.Duration(numEntries)

	t.Logf("Add performance: %v total, %v per entry", addDuration, addPerEntry)

	// Measure list operation performance
	start = time.Now()
	entries, err := store.ListEntries(profile, nil)
	listDuration := time.Since(start)

	if err != nil {
		t.Errorf("Failed to list entries: %v", err)
	}

	if len(entries) != numEntries {
		t.Errorf("Expected %d entries, got %d", numEntries, len(entries))
	}

	t.Logf("List performance: %v for %d entries", listDuration, len(entries))

	// Measure get operation performance
	start = time.Now()
	for i := 0; i < 10; i++ { // Test 10 random gets
		entryName := fmt.Sprintf("perf-entry-%d", i*10)
		_, err = store.GetEntry(profile, entryName)
		if err != nil {
			t.Errorf("Failed to get entry %s: %v", entryName, err)
		}
	}
	getDuration := time.Since(start)
	getPerEntry := getDuration / 10

	t.Logf("Get performance: %v total, %v per entry", getDuration, getPerEntry)

	// Performance thresholds (adjust based on requirements)
	if addPerEntry > 10*time.Millisecond {
		t.Logf("Warning: Add operation taking %v per entry (threshold: 10ms)", addPerEntry)
	}

	if listDuration > 100*time.Millisecond {
		t.Logf("Warning: List operation taking %v (threshold: 100ms)", listDuration)
	}

	if getPerEntry > 5*time.Millisecond {
		t.Logf("Warning: Get operation taking %v per entry (threshold: 5ms)", getPerEntry)
	}
}
