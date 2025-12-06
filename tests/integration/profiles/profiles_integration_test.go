package profiles_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/vault-cli/vault/internal/domain"
	"github.com/vault-cli/vault/internal/store"
	"github.com/vault-cli/vault/internal/vault"
)

// TestProfileCreation tests creating and managing profiles
func TestProfileCreation(t *testing.T) {
	tempDir := t.TempDir()
	vaultPath := filepath.Join(tempDir, "profiles_test.vault")
	
	// Create vault
	passphrase := "test-profiles"
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
	
	t.Run("Create multiple profiles", func(t *testing.T) {
		profiles := []string{"work", "personal", "production", "development"}
		
		for _, profileName := range profiles {
			// Create profile first
			if err := vaultStore.CreateProfile(profileName, fmt.Sprintf("%s profile", profileName)); err != nil {
				t.Fatalf("Failed to create profile %s: %v", profileName, err)
			}
			
			// Add entry to profile to verify it exists
			entry := &domain.Entry{
				Name:     fmt.Sprintf("%s-entry", profileName),
				Username: "user",
				Password: []byte("pass"),
			}
			
			if err := vaultStore.CreateEntry(profileName, entry); err != nil {
				t.Errorf("Failed to create entry in profile %s: %v", profileName, err)
			}
		}
		
		t.Logf("✓ Created %d profiles successfully", len(profiles))
	})
}

// TestProfileIsolation tests that profiles are isolated from each other
func TestProfileIsolation(t *testing.T) {
	tempDir := t.TempDir()
	vaultPath := filepath.Join(tempDir, "isolation_test.vault")
	
	// Create vault
	passphrase := "test-isolation"
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
	
	t.Run("Entries in one profile don't appear in another", func(t *testing.T) {
		// Create profiles first
		if err := vaultStore.CreateProfile("work", "Work profile"); err != nil {
			t.Fatalf("Failed to create work profile: %v", err)
		}
		if err := vaultStore.CreateProfile("personal", "Personal profile"); err != nil {
			t.Fatalf("Failed to create personal profile: %v", err)
		}
		
		// Add entries to different profiles
		workEntry := &domain.Entry{
			Name:     "work-github",
			Username: "work@company.com",
			Password: []byte("work-pass"),
		}
		
		personalEntry := &domain.Entry{
			Name:     "personal-email",
			Username: "me@personal.com",
			Password: []byte("personal-pass"),
		}
		
		if err := vaultStore.CreateEntry("work", workEntry); err != nil {
			t.Fatalf("Failed to create work entry: %v", err)
		}
		
		if err := vaultStore.CreateEntry("personal", personalEntry); err != nil {
			t.Fatalf("Failed to create personal entry: %v", err)
		}
		
		// List work profile entries
		workEntries, err := vaultStore.ListEntries("work", nil)
		if err != nil {
			t.Fatalf("Failed to list work entries: %v", err)
		}
		
		// Verify only work entry is in work profile
		foundWork := false
		foundPersonal := false
		for _, entry := range workEntries {
			if entry.Name == "work-github" {
				foundWork = true
			}
			if entry.Name == "personal-email" {
				foundPersonal = true
			}
		}
		
		if !foundWork {
			t.Error("Work entry not found in work profile")
		}
		if foundPersonal {
			t.Error("Personal entry should not appear in work profile")
		}
		
		// List personal profile entries
		personalEntries, err := vaultStore.ListEntries("personal", nil)
		if err != nil {
			t.Fatalf("Failed to list personal entries: %v", err)
		}
		
		// Verify only personal entry is in personal profile
		foundWork = false
		foundPersonal = false
		for _, entry := range personalEntries {
			if entry.Name == "work-github" {
				foundWork = true
			}
			if entry.Name == "personal-email" {
				foundPersonal = true
			}
		}
		
		if foundWork {
			t.Error("Work entry should not appear in personal profile")
		}
		if !foundPersonal {
			t.Error("Personal entry not found in personal profile")
		}
		
		t.Log("✓ Profile isolation verified")
	})
	
	t.Run("Same entry name in different profiles", func(t *testing.T) {
		// Create profiles if they don't exist
		if !vaultStore.ProfileExists("work") {
			if err := vaultStore.CreateProfile("work", "Work profile"); err != nil {
				t.Fatalf("Failed to create work profile: %v", err)
			}
		}
		if !vaultStore.ProfileExists("personal") {
			if err := vaultStore.CreateProfile("personal", "Personal profile"); err != nil {
				t.Fatalf("Failed to create personal profile: %v", err)
			}
		}
		
		// Add entries with same name to different profiles
		entry1 := &domain.Entry{
			Name:     "github",
			Username: "work@company.com",
			Password: []byte("work-github-pass"),
		}
		
		entry2 := &domain.Entry{
			Name:     "github",
			Username: "personal@email.com",
			Password: []byte("personal-github-pass"),
		}
		
		if err := vaultStore.CreateEntry("work", entry1); err != nil {
			t.Fatalf("Failed to create work github entry: %v", err)
		}
		
		if err := vaultStore.CreateEntry("personal", entry2); err != nil {
			t.Fatalf("Failed to create personal github entry: %v", err)
		}
		
		// Retrieve from work profile
		workGithub, err := vaultStore.GetEntry("work", "github")
		if err != nil {
			t.Fatalf("Failed to get work github: %v", err)
		}
		
		if workGithub.Username != "work@company.com" {
			t.Errorf("Wrong entry retrieved from work profile: %s", workGithub.Username)
		}
		
		// Retrieve from personal profile
		personalGithub, err := vaultStore.GetEntry("personal", "github")
		if err != nil {
			t.Fatalf("Failed to get personal github: %v", err)
		}
		
		if personalGithub.Username != "personal@email.com" {
			t.Errorf("Wrong entry retrieved from personal profile: %s", personalGithub.Username)
		}
		
		t.Log("✓ Same entry name in different profiles works correctly")
	})
}

// TestProfileOperations tests CRUD operations on profiles
func TestProfileOperations(t *testing.T) {
	tempDir := t.TempDir()
	vaultPath := filepath.Join(tempDir, "operations_test.vault")
	
	// Create vault
	passphrase := "test-operations"
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
	
	t.Run("Add entries to profile", func(t *testing.T) {
		// Create test profile
		if err := vaultStore.CreateProfile("test-profile", "Test profile"); err != nil {
			t.Fatalf("Failed to create test profile: %v", err)
		}
		
		entries := []struct {
			name     string
			username string
		}{
			{"entry1", "user1"},
			{"entry2", "user2"},
			{"entry3", "user3"},
		}
		
		for _, e := range entries {
			entry := &domain.Entry{
				Name:     e.name,
				Username: e.username,
				Password: []byte("pass"),
			}
			
			if err := vaultStore.CreateEntry("test-profile", entry); err != nil {
				t.Errorf("Failed to add entry %s: %v", e.name, err)
			}
		}
		
		// Verify all entries added
		allEntries, err := vaultStore.ListEntries("test-profile", nil)
		if err != nil {
			t.Fatalf("Failed to list entries: %v", err)
		}
		
		if len(allEntries) != len(entries) {
			t.Errorf("Expected %d entries, got %d", len(entries), len(allEntries))
		}
		
		t.Logf("✓ Added %d entries to profile", len(entries))
	})
	
	t.Run("Update entry in profile", func(t *testing.T) {
		// Update entry
		updatedEntry := &domain.Entry{
			Name:     "entry1",
			Username: "updated-user",
			Password: []byte("updated-pass"),
		}
		
		if err := vaultStore.UpdateEntry("test-profile", "entry1", updatedEntry); err != nil {
			t.Fatalf("Failed to update entry: %v", err)
		}
		
		// Verify update
		retrieved, err := vaultStore.GetEntry("test-profile", "entry1")
		if err != nil {
			t.Fatalf("Failed to get updated entry: %v", err)
		}
		
		if retrieved.Username != "updated-user" {
			t.Errorf("Entry not updated: got %s", retrieved.Username)
		}
		
		t.Log("✓ Entry updated successfully")
	})
	
	t.Run("Delete entry from profile", func(t *testing.T) {
		if err := vaultStore.DeleteEntry("test-profile", "entry3"); err != nil {
			t.Fatalf("Failed to delete entry: %v", err)
		}
		
		// Verify deletion
		_, err := vaultStore.GetEntry("test-profile", "entry3")
		if err == nil {
			t.Error("Entry should have been deleted")
		}
		
		t.Log("✓ Entry deleted successfully")
	})
}

// TestProfileWithFilters tests filtering entries within profiles
func TestProfileWithFilters(t *testing.T) {
	tempDir := t.TempDir()
	vaultPath := filepath.Join(tempDir, "filters_test.vault")
	
	// Create vault
	passphrase := "test-filters"
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
	
	t.Run("Filter by tags within profile", func(t *testing.T) {
		// Create filtered profile
		if err := vaultStore.CreateProfile("filtered-profile", "Filtered profile"); err != nil {
			t.Fatalf("Failed to create filtered profile: %v", err)
		}
		
		// Add entries with different tags
		entries := []struct {
			name string
			tags []string
		}{
			{"github", []string{"work", "git"}},
			{"gitlab", []string{"work", "git"}},
			{"email", []string{"personal", "communication"}},
			{"slack", []string{"work", "communication"}},
		}
		
		for _, e := range entries {
			entry := &domain.Entry{
				Name:     e.name,
				Username: "user",
				Password: []byte("pass"),
				Tags:     e.tags,
			}
			
			if err := vaultStore.CreateEntry("filtered-profile", entry); err != nil {
				t.Errorf("Failed to add entry %s: %v", e.name, err)
			}
		}
		
		// Filter by "work" tag
		filter := &domain.Filter{
			Tags: []string{"work"},
		}
		
		workEntries, err := vaultStore.ListEntries("filtered-profile", filter)
		if err != nil {
			t.Fatalf("Failed to list filtered entries: %v", err)
		}
		
		// Should get github, gitlab, slack
		if len(workEntries) != 3 {
			t.Errorf("Expected 3 work entries, got %d", len(workEntries))
		}
		
		t.Logf("✓ Tag filtering works correctly (%d entries found)", len(workEntries))
	})
}

// TestDefaultProfile tests default profile behavior
func TestDefaultProfile(t *testing.T) {
	tempDir := t.TempDir()
	vaultPath := filepath.Join(tempDir, "default_test.vault")
	
	// Create vault
	passphrase := "test-default"
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
	
	t.Run("Default profile exists", func(t *testing.T) {
		// Add entry to default profile
		entry := &domain.Entry{
			Name:     "default-entry",
			Username: "user",
			Password: []byte("pass"),
		}
		
		if err := vaultStore.CreateEntry("default", entry); err != nil {
			t.Fatalf("Failed to add entry to default profile: %v", err)
		}
		
		// Retrieve entry
		retrieved, err := vaultStore.GetEntry("default", "default-entry")
		if err != nil {
			t.Fatalf("Failed to get entry from default profile: %v", err)
		}
		
		if retrieved.Name != "default-entry" {
			t.Error("Entry not found in default profile")
		}
		
		t.Log("✓ Default profile works correctly")
	})
}

// TestProfilePersistence tests that profiles persist across sessions
func TestProfilePersistence(t *testing.T) {
	tempDir := t.TempDir()
	vaultPath := filepath.Join(tempDir, "persistence_test.vault")
	
	// Create vault
	passphrase := "test-persistence"
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
	
	// Session 1: Create profiles and add entries
	{
		vaultStore := store.NewBoltStore()
		if err := vaultStore.CreateVault(vaultPath, masterKey, kdfParamsMap); err != nil {
			t.Fatalf("Failed to create vault: %v", err)
		}
		
		if err := vaultStore.OpenVault(vaultPath, masterKey); err != nil {
			t.Fatalf("Failed to open vault: %v", err)
		}
		
		// Create and add entries to different profiles
		profiles := []string{"work", "personal", "production"}
		for _, profile := range profiles {
			// Create profile
			if err := vaultStore.CreateProfile(profile, fmt.Sprintf("%s profile", profile)); err != nil {
				t.Fatalf("Failed to create profile %s: %v", profile, err)
			}
			
			entry := &domain.Entry{
				Name:     fmt.Sprintf("%s-entry", profile),
				Username: "user",
				Password: []byte("pass"),
			}
			
			if err := vaultStore.CreateEntry(profile, entry); err != nil {
				t.Fatalf("Failed to add entry to %s: %v", profile, err)
			}
		}
		
		vaultStore.CloseVault()
	}
	
	// Session 2: Reopen and verify profiles exist
	{
		vaultStore := store.NewBoltStore()
		if err := vaultStore.OpenVault(vaultPath, masterKey); err != nil {
			t.Fatalf("Failed to reopen vault: %v", err)
		}
		defer vaultStore.CloseVault()
		
		// Verify entries in each profile
		profiles := []string{"work", "personal", "production"}
		for _, profile := range profiles {
			entryName := fmt.Sprintf("%s-entry", profile)
			_, err := vaultStore.GetEntry(profile, entryName)
			if err != nil {
				t.Errorf("Entry not found in %s profile after reopen: %v", profile, err)
			}
		}
		
		t.Log("✓ Profiles persisted across sessions")
	}
}
