package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/vault-cli/vault/internal/config"
	"github.com/vault-cli/vault/internal/vault"
)

var (
	exportEncrypted      bool
	exportPlaintext      bool
	exportPath           string
	exportIncludeSecrets bool
	exportPassphrase     string
)

// NewExport creates a new export command
func NewExport(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
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

	cmd.Flags().BoolVar(&exportEncrypted, "encrypted", false, "Export in encrypted format")
	cmd.Flags().BoolVar(&exportPlaintext, "plaintext", false, "Export in plaintext format")
	cmd.Flags().StringVar(&exportPath, "path", "", "Export file path (required)")
	cmd.Flags().BoolVar(&exportIncludeSecrets, "include-secrets", false, "Include secrets in export (requires confirmation)")
	cmd.Flags().StringVar(&exportPassphrase, "passphrase", "", "Passphrase for encrypted export")
	cmd.MarkFlagRequired("path")

	return cmd
}

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

	vaultStore := GetVaultStore()

	// Confirm for plaintext export only when secrets are included.
	if exportPlaintext && exportIncludeSecrets {
		confirmed, err := PromptConfirm(
			"WARNING: Exporting a PLAINTEXT vault snapshot with secrets. Passwords and DID private keys will be visible. Continue?",
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

	tmpFile, err := os.CreateTemp("", "vault-export-*.json")
	if err != nil {
		return fmt.Errorf("failed to create temporary export file: %w", err)
	}
	tmpPath := tmpFile.Name()
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary export file: %w", err)
	}
	defer func() {
		if removeErr := os.Remove(tmpPath); removeErr != nil && !os.IsNotExist(removeErr) {
			logWarning("Failed to remove temporary export file %s: %v", tmpPath, removeErr)
		}
	}()

	if err := vaultStore.ExportVault(tmpPath, exportIncludeSecrets); err != nil {
		return fmt.Errorf("failed to export vault snapshot: %w", err)
	}

	rawSnapshot, err := os.ReadFile(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to read temporary export snapshot: %w", err)
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

	data := rawSnapshot
	if exportEncrypted {
		data, err = vault.EncryptExportData(rawSnapshot, passphrase)
		if err != nil {
			return fmt.Errorf("failed to encrypt export data: %w", err)
		}
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

	fmt.Printf("✓ Exported vault snapshot to %s (%s format)\n", exportPath, format)
	if exportEncrypted {
		fmt.Println("Keep your export passphrase safe - you'll need it to import this backup")
	} else if exportIncludeSecrets {
		fmt.Println("WARNING: This file contains unencrypted secrets, including DID private keys. Store it securely!")
	} else {
		fmt.Println("Snapshot exported without secret material")
	}

	// Close session store to release lock file
	if err := CloseSessionStore(); err != nil {
		logWarning("Failed to close session store after export: %v", err)
	}

	return nil
}
