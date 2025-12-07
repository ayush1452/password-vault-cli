package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/vault-cli/vault/internal/vault"
)

var (
	exportEncrypted      bool
	exportPlaintext      bool
	exportPath           string
	exportIncludeSecrets bool
	exportPassphrase     string
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export vault data",
	Long: `Export vault data for backup or migration purposes.

You can export in encrypted format (recommended) or plaintext format.
Encrypted exports require a passphrase and can be safely stored anywhere.
Plaintext exports are unencrypted and should only be used in secure locations.

Example:
  vault export --path backup.json --encrypted --passphrase "backup-pass"
  vault export --path backup.json --plaintext --include-secrets`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runExport()
	},
}

func init() {
	exportCmd.Flags().BoolVar(&exportEncrypted, "encrypted", false, "Export in encrypted format")
	exportCmd.Flags().BoolVar(&exportPlaintext, "plaintext", false, "Export in plaintext format")
	exportCmd.Flags().StringVar(&exportPath, "path", "", "Export file path (required)")
	exportCmd.Flags().BoolVar(&exportIncludeSecrets, "include-secrets", false, "Include secrets in export (requires confirmation)")
	exportCmd.Flags().StringVar(&exportPassphrase, "passphrase", "", "Passphrase for encrypted export")
	exportCmd.MarkFlagRequired("path")
}

func runExport() error {
	// Check if vault is unlocked
	if !IsUnlocked() {
		return fmt.Errorf("vault is locked, run 'vault unlock' first")
	}

	// Validate flags
	if !exportEncrypted && !exportPlaintext {
		exportEncrypted = true // Default to encrypted
	}

	if exportEncrypted && exportPlaintext {
		return fmt.Errorf("cannot use both --encrypted and --plaintext")
	}

	// Ensure profile is set
	if profile == "" {
		profile = "default"
	}

	// Get vault store
	vaultStore := GetVaultStore()

	// Get all entries for the profile
	entries, err := vaultStore.ListEntries(profile, nil)
	if err != nil {
		return fmt.Errorf("failed to list entries: %w", err)
	}

	if len(entries) == 0 {
		return fmt.Errorf("no entries to export in profile '%s'", profile)
	}

	// Confirm for plaintext export
	if exportPlaintext {
		if !exportIncludeSecrets {
			return fmt.Errorf("plaintext export requires --include-secrets flag for safety")
		}

		confirmed, err := PromptConfirm(
			fmt.Sprintf("WARNING: Exporting %d entries in PLAINTEXT format. Secrets will be visible. Continue?", len(entries)),
			false,
		)
		if err != nil {
			return fmt.Errorf("failed to get confirmation: %w", err)
		}
		if !confirmed {
			fmt.Println("Export canceled")
			return nil
		}
	}

	// Get passphrase for encrypted export
	var passphrase string
	if exportEncrypted {
		if exportPassphrase != "" {
			passphrase = exportPassphrase
		} else {
			passphrase, err = PromptPassword("Enter passphrase for encrypted export: ")
			if err != nil {
				return fmt.Errorf("failed to read passphrase: %w", err)
			}

			// Confirm passphrase
			confirmPass, err := PromptPassword("Confirm passphrase: ")
			if err != nil {
				return fmt.Errorf("failed to read confirmation: %w", err)
			}

			if passphrase != confirmPass {
				return fmt.Errorf("passphrases do not match")
			}
		}
	}

	// Export entries
	data, err := vault.ExportVault(entries, passphrase, exportEncrypted)
	if err != nil {
		return fmt.Errorf("failed to export vault: %w", err)
	}

	// Write to file
	if err := os.WriteFile(exportPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write export file: %w", err)
	}

	// Success message
	format := "encrypted"
	if exportPlaintext {
		format = "plaintext"
	}

	fmt.Printf("✓ Exported %d entries to %s (%s format)\n", len(entries), exportPath, format)
	if exportEncrypted {
		fmt.Println("Keep your export passphrase safe - you'll need it to import this backup")
	} else {
		fmt.Println("WARNING: This file contains unencrypted secrets. Store it securely!")
	}

	return nil
}
