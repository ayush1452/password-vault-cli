package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vault-cli/vault/internal/domain"
	"github.com/vault-cli/vault/internal/store"
	"github.com/vault-cli/vault/internal/vault"
)

func main() {
	fmt.Println("Password Vault CLI - Storage Layer Demo")
	fmt.Println("=======================================")

	// Create temporary directory for demo
	tempDir, err := os.MkdirTemp("", "vault_storage_demo_")
	if err != nil {
		log.Printf("Failed to create temp directory: %v", err)
		return
	}

	vaultPath := filepath.Join(tempDir, "demo.vault")
	fmt.Printf("Demo vault: %s\n", vaultPath)

	// Only set up cleanup after we know the temp directory was created successfully
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			log.Printf("Warning: failed to remove temp directory: %v", err)
		}
	}()

	// Demo 1: Create and initialize vault
	fmt.Println("\n1. Creating New Vault")
	fmt.Println("   " + strings.Repeat("─", 40))

	// Generate master key from passphrase
	passphrase := "my-secure-master-passphrase-123"
	salt, err := vault.GenerateSalt()
	if err != nil {
		log.Printf("Failed to generate salt: %v", err)
		return
	}

	crypto := vault.NewDefaultCryptoEngine()
	masterKey, err := crypto.DeriveKey(passphrase, salt)
	if err != nil {
		log.Printf("Failed to derive key: %v", err)
		return
	}
	defer vault.Zeroize(masterKey)

	// Create KDF parameters
	kdfParams := map[string]interface{}{
		"memory":      uint32(1024), // 1 MB for demo
		"iterations":  uint32(1),
		"parallelism": uint8(1),
	}

	// Create vault store
	vaultStore := store.NewBoltStore()

	err = vaultStore.CreateVault(vaultPath, masterKey, kdfParams)
	if err != nil {
		log.Printf("Failed to create vault: %v", err)
		return
	}
	fmt.Println("   ✓ Vault created successfully")

	// Demo 2: Open vault
	fmt.Println("\n2. Opening Vault")
	fmt.Println("   " + strings.Repeat("─", 40))

	err = vaultStore.OpenVault(vaultPath, masterKey)
	if err != nil {
		log.Printf("Failed to open vault: %v", err)
		return
	}
	fmt.Println("   ✓ Vault opened successfully")
	defer func() {
		if err := vaultStore.CloseVault(); err != nil {
			log.Printf("Warning: failed to close vault: %v", err)
		}
	}()

	// Demo 3: Get vault metadata
	fmt.Println("\n3. Vault Metadata")
	fmt.Println("   " + strings.Repeat("─", 40))

	metadata, err := vaultStore.GetVaultMetadata()
	if err != nil {
		log.Printf("Failed to get metadata: %v", err)
		return
	}
	fmt.Printf("   Version: %s\n", metadata.Version)
	fmt.Printf("   Created: %s\n", metadata.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("   KDF Memory: %v KB\n", metadata.KDFParams["memory"])
	fmt.Printf("   KDF Iterations: %v\n", metadata.KDFParams["iterations"])

	// Demo 4: Profile management
	fmt.Println("\n4. Profile Management")
	fmt.Println("   " + strings.Repeat("─", 40))

	// List existing profiles
	profiles, err := vaultStore.ListProfiles()
	if err != nil {
		log.Printf("Failed to list profiles: %v", err)
		return
	}
	fmt.Printf("   Initial profiles: %d\n", len(profiles))
	for _, p := range profiles {
		fmt.Printf("   - %s: %s\n", p.Name, p.Description)
	}

	// Create new profiles
	err = vaultStore.CreateProfile("work", "Work-related passwords")
	if err != nil {
		log.Printf("Failed to create work profile: %v", err)
		return
	}
	fmt.Println("   ✓ Created 'work' profile")

	err = vaultStore.CreateProfile("personal", "Personal accounts")
	if err != nil {
		log.Printf("Failed to create personal profile: %v", err)
		return
	}
	fmt.Println("   ✓ Created 'personal' profile")

	// List all profiles
	profiles, err = vaultStore.ListProfiles()
	if err != nil {
		log.Printf("Failed to list profiles: %v", err)
		return
	}
	fmt.Printf("   Total profiles: %d\n", len(profiles))

	// Demo 5: Entry creation and storage
	fmt.Println("\n5. Creating Password Entries")
	fmt.Println("   " + strings.Repeat("─", 40))

	// Create sample entries
	entries := []*domain.Entry{
		{
			Name:      "github",
			Username:  "john.doe@example.com",
			Secret:    []byte("super-secret-password-123"),
			URL:       "https://github.com",
			Notes:     "Personal GitHub account",
			Tags:      []string{"development", "git"},
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		},
		{
			Name:      "aws-console",
			Username:  "admin@company.com",
			Secret:    []byte("AWS-P@ssw0rd-2024"),
			URL:       "https://console.aws.amazon.com",
			Notes:     "AWS production console access",
			Tags:      []string{"work", "cloud", "production"},
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		},
		{
			Name:      "database",
			Username:  "db_admin",
			Secret:    []byte("PostgreSQL-Secret-Key"),
			URL:       "postgres://localhost:5432/mydb",
			Notes:     "Production database credentials",
			Tags:      []string{"work", "database"},
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		},
	}

	// Store entries in work profile
	for _, entry := range entries[:2] {
		err = vaultStore.CreateEntry("work", entry)
		if err != nil {
			log.Printf("Failed to create entry %s: %v", entry.Name, err)
			continue
		}
		fmt.Printf("   ✓ Created entry: %s (%s)\n", entry.Name, entry.URL)
	}

	// Store entry in default profile
	githubEntry := &domain.Entry{
		Name:      "personal-github",
		Username:  "personal@email.com",
		Secret:    []byte("personal-github-token"),
		URL:       "https://github.com",
		Notes:     "Personal projects",
		Tags:      []string{"personal", "git"},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	err = vaultStore.CreateEntry("default", githubEntry)
	if err != nil {
		log.Printf("Failed to create personal entry: %v", err)
		return
	}
	fmt.Println("   ✓ Created entry: personal-github (default profile)")

	// Demo 6: Entry retrieval
	fmt.Println("\n6. Retrieving Entries")
	fmt.Println("   " + strings.Repeat("─", 40))

	// Get specific entry
	retrievedEntry, err := vaultStore.GetEntry("work", "github")
	if err != nil {
		log.Printf("Failed to retrieve entry: %v", err)
		return
	}
	fmt.Printf("   Entry: %s\n", retrievedEntry.Name)
	fmt.Printf("   Username: %s\n", retrievedEntry.Username)
	fmt.Printf("   URL: %s\n", retrievedEntry.URL)
	fmt.Printf("   Secret: %s (decrypted)\n", string(retrievedEntry.Secret))
	fmt.Printf("   Tags: %v\n", retrievedEntry.Tags)

	// Demo 7: List all entries
	fmt.Println("\n7. Listing All Entries")
	fmt.Println("   " + strings.Repeat("─", 40))

	// List entries in work profile
	workEntries, err := vaultStore.ListEntries("work", nil)
	if err != nil {
		log.Printf("Failed to list work entries: %v", err)
		return
	}
	fmt.Printf("   Work profile (%d entries):\n", len(workEntries))
	for _, e := range workEntries {
		fmt.Printf("   - %s: %s (%s)\n", e.Name, e.Username, e.URL)
	}

	// List entries in default profile
	defaultEntries, err := vaultStore.ListEntries("default", nil)
	if err != nil {
		log.Printf("Failed to list default entries: %v", err)
		return
	}
	fmt.Printf("   Default profile (%d entries):\n", len(defaultEntries))
	for _, e := range defaultEntries {
		fmt.Printf("   - %s: %s (%s)\n", e.Name, e.Username, e.URL)
	}

	// Demo 8: Entry update
	fmt.Println("\n8. Updating Entry")
	fmt.Println("   " + strings.Repeat("─", 40))

	// Update entry
	retrievedEntry.Notes = "Updated: Personal GitHub account with 2FA enabled"
	retrievedEntry.Tags = append(retrievedEntry.Tags, "2fa-enabled")
	retrievedEntry.UpdatedAt = time.Now().UTC()

	err = vaultStore.UpdateEntry("work", "github", retrievedEntry)
	if err != nil {
		log.Printf("Failed to update entry: %v", err)
		return
	}
	fmt.Println("   ✓ Entry updated successfully")

	// Verify update
	updated, err := vaultStore.GetEntry("work", "github")
	if err != nil {
		log.Printf("Failed to retrieve updated entry: %v", err)
		return
	}
	fmt.Printf("   New notes: %s\n", updated.Notes)
	fmt.Printf("   New tags: %v\n", updated.Tags)

	// Demo 9: Entry filtering
	fmt.Println("\n9. Filtering Entries")
	fmt.Println("   " + strings.Repeat("─", 40))

	// Filter by tags
	filter := &domain.Filter{
		Tags: []string{"development"},
	}
	filtered, err := vaultStore.ListEntries("work", filter)
	if err != nil {
		log.Printf("Failed to filter entries: %v", err)
		return
	}
	fmt.Printf("   Entries with 'development' tag: %d\n", len(filtered))
	for _, e := range filtered {
		fmt.Printf("   - %s\n", e.Name)
	}

	// Demo 10: Entry deletion
	fmt.Println("\n10. Deleting Entry")
	fmt.Println("   " + strings.Repeat("─", 40))

	// Check if entry exists
	if vaultStore.EntryExists("work", "database") {
		fmt.Println("   Entry 'database' exists")
	}

	// Delete entry
	err = vaultStore.DeleteEntry("work", "database")
	if err != nil {
		log.Printf("Failed to delete entry: %v", err)
		return
	}
	fmt.Println("   ✓ Entry 'database' deleted")

	// Verify deletion
	if !vaultStore.EntryExists("work", "database") {
		fmt.Println("   ✓ Entry no longer exists")
	}

	// Demo 11: Profile deletion
	fmt.Println("\n11. Profile Management (Deletion)")
	fmt.Println("   " + strings.Repeat("─", 40))

	// Delete personal profile (it's empty)
	err = vaultStore.DeleteProfile("personal")
	if err != nil {
		log.Printf("Failed to delete profile: %v", err)
		return
	}
	fmt.Println("   ✓ Profile 'personal' deleted")

	profiles, err = vaultStore.ListProfiles()
	if err != nil {
		log.Printf("Failed to list profiles: %v", err)
		return
	}
	fmt.Printf("   Remaining profiles: %d\n", len(profiles))
	for _, p := range profiles {
		fmt.Printf("   - %s\n", p.Name)
	}

	// Demo 12: Vault integrity check
	fmt.Println("\n12. Integrity Verification")
	fmt.Println("   " + strings.Repeat("─", 40))

	err = vaultStore.VerifyIntegrity()
	if err != nil {
		log.Printf("   ✗ Integrity check failed: %v\n", err)
	} else {
		fmt.Println("   ✓ Vault integrity verified")
	}

	// Demo 13: Close vault
	fmt.Println("\n13. Closing Vault")
	fmt.Println("   " + strings.Repeat("─", 40))

	err = vaultStore.CloseVault()
	if err != nil {
		log.Printf("Failed to close vault: %v", err)
		return
	}
	fmt.Println("   ✓ Vault closed successfully")

	if !vaultStore.IsOpen() {
		fmt.Println("   ✓ Vault is locked")
	}

	// Summary
	fmt.Println("\n" + strings.Repeat("═", 50))
	fmt.Println("✓ All storage operations completed successfully!")
	fmt.Println(strings.Repeat("═", 50))
	fmt.Println("\nKey Features Demonstrated:")
	fmt.Println("  • Vault creation and initialization")
	fmt.Println("  • Secure entry storage with encryption")
	fmt.Println("  • Profile management")
	fmt.Println("  • CRUD operations (Create, Read, Update, Delete)")
	fmt.Println("  • Entry filtering and search")
	fmt.Println("  • Integrity verification")
	fmt.Println("  • Secure vault locking")
	fmt.Printf("\nVault location: %s\n", vaultPath)
	fmt.Println("(will be automatically cleaned up)")
}
