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

	rawSnapshot := data
	if vault.IsEncryptedExport(data) {
		rawSnapshot, err = vault.DecryptExportData(data, passphrase)
		if err != nil {
			return fmt.Errorf("failed to decrypt import: %w", err)
		}
	}

	vaultStore := GetVaultStore()

	tmpFile, err := os.CreateTemp("", "vault-import-*.json")
	if err != nil {
		return fmt.Errorf("failed to create temporary import file: %w", err)
	}
	tmpPath := tmpFile.Name()
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary import file: %w", err)
	}
	defer func() {
		if removeErr := os.Remove(tmpPath); removeErr != nil && !os.IsNotExist(removeErr) {
			logWarning("Failed to remove temporary import file %s: %v", tmpPath, removeErr)
		}
	}()

	if err := os.WriteFile(tmpPath, rawSnapshot, 0600); err != nil {
		return fmt.Errorf("failed to stage import snapshot: %w", err)
	}

	if err := vaultStore.ImportVault(tmpPath, importConflict); err != nil {
		return fmt.Errorf("failed to import vault snapshot: %w", err)
	}

	// Refresh session
	RefreshSession()

	// Close session store to release lock file
	if err := CloseSessionStore(); err != nil {
		logWarning("Failed to close session store after import: %v", err)
	}

	fmt.Println("✓ Vault snapshot imported successfully")

	return nil
}
