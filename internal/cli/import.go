package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/vault-cli/vault/internal/config"
	"github.com/vault-cli/vault/internal/vault"
)

var (
	importPath       string
	importConflict   string
	importPassphrase string
)

// NewImport creates a new import command
func NewImport(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import vault data",
		Long: `Import vault data from a backup file.

You can import from encrypted vault exports or plaintext JSON files.
Conflict resolution determines what happens when entries already exist:
  - skip: Skip existing entries (default)
  - overwrite: Replace existing entries
  - fail: Error if any entry exists

Example:
  vault import --path backup.json --passphrase "backup-pass"
  vault import --path backup.json --conflict overwrite
  vault import --path backup.json --conflict skip`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runImport()
		},
	}

	cmd.Flags().StringVar(&importPath, "path", "", "Import file path (required)")
	cmd.Flags().StringVar(&importConflict, "conflict", "skip", "Conflict resolution (skip|overwrite|fail)")
	cmd.Flags().StringVar(&importPassphrase, "passphrase", "", "Passphrase for encrypted import")
	cmd.MarkFlagRequired("path")

	return cmd
}

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import vault data",
	Long: `Import vault data from a backup file.

You can import from encrypted vault exports or plaintext JSON files.
Conflict resolution determines what happens when entries already exist:
  - skip: Skip existing entries (default)
  - overwrite: Replace existing entries
  - fail: Error if any entry exists

Example:
  vault import --path backup.json --passphrase "backup-pass"
  vault import --path backup.json --conflict overwrite
  vault import --path backup.json --conflict skip`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runImport()
	},
}

func init() {
	importCmd.Flags().StringVar(&importPath, "path", "", "Import file path (required)")
	importCmd.Flags().StringVar(&importConflict, "conflict", "skip", "Conflict resolution (skip|overwrite|fail)")
	importCmd.Flags().StringVar(&importPassphrase, "passphrase", "", "Passphrase for encrypted import")
	importCmd.MarkFlagRequired("path")
}

func runImport() error {
	// Check if vault is unlocked
	if !IsUnlocked() {
		return fmt.Errorf("vault is locked, run 'vault unlock' first")
	}

	// Validate conflict strategy
	if importConflict != "skip" && importConflict != "overwrite" && importConflict != "fail" {
		return fmt.Errorf("invalid conflict resolution: %s (must be skip, overwrite, or fail)", importConflict)
	}

	// Ensure profile is set
	if profile == "" {
		profile = "default"
	}

	// Read import file
	data, err := os.ReadFile(importPath)
	if err != nil {
		return fmt.Errorf("failed to read import file: %w", err)
	}

	// Get passphrase if needed
	var passphrase string
	if importPassphrase != "" {
		passphrase = importPassphrase
	} else {
		// Try to detect if file is encrypted by checking for "encrypted": true
		if vault.IsEncryptedExport(data) {
			passphrase, err = PromptPassword("Enter passphrase for encrypted import: ")
			if err != nil {
				return fmt.Errorf("failed to read passphrase: %w", err)
			}
		}
	}

	// Import entries
	entries, err := vault.ImportVault(data, passphrase)
	if err != nil {
		return fmt.Errorf("failed to import vault: %w", err)
	}

	if len(entries) == 0 {
		return fmt.Errorf("no entries found in import file")
	}

	// Get vault store
	vaultStore := GetVaultStore()

	// Import entries with conflict resolution
	imported := 0
	skipped := 0
	overwritten := 0

	for _, entry := range entries {
		exists := vaultStore.EntryExists(profile, entry.Name)

		if exists {
			switch importConflict {
			case "fail":
				return fmt.Errorf("entry '%s' already exists (use --conflict skip or --conflict overwrite)", entry.Name)
			case "skip":
				skipped++
				continue
			case "overwrite":
				if err := vaultStore.UpdateEntry(profile, entry.Name, entry); err != nil {
					return fmt.Errorf("failed to overwrite entry '%s': %w", entry.Name, err)
				}
				overwritten++
			}
		} else {
			if err := vaultStore.CreateEntry(profile, entry); err != nil {
				return fmt.Errorf("failed to create entry '%s': %w", entry.Name, err)
			}
			imported++
		}
	}

	// Refresh session
	RefreshSession()

	// Success message
	fmt.Printf("✓ Import completed:\n")
	if imported > 0 {
		fmt.Printf("  - %d new entries imported\n", imported)
	}
	if overwritten > 0 {
		fmt.Printf("  - %d entries overwritten\n", overwritten)
	}
	if skipped > 0 {
		fmt.Printf("  - %d entries skipped (already exist)\n", skipped)
	}
	fmt.Printf("Total: %d entries processed\n", len(entries))

	return nil
}
