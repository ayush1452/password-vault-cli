package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/vault-cli/vault/internal/config"
)

var deleteYes bool

var deleteCmd = &cobra.Command{
	Use:   "delete <entry-name>",
	Short: "Delete an entry from the vault",
	Long: `Delete an entry from the vault permanently.

This action cannot be undone. You will be prompted for confirmation
unless you use the --yes flag.

Example:
  vault delete old-account
  vault delete temp-entry --yes`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDelete(args[0])
	},
}

func init() {
	deleteCmd.Flags().BoolVar(&deleteYes, "yes", false, "Skip confirmation prompt")
}

// NewDeleteCommand creates a new delete command for testing
func NewDeleteCommand(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <entry-name>",
		Short: "Delete an entry from the vault",
		Long: `Delete an entry from the vault permanently.

This action cannot be undone. You will be prompted for confirmation
unless you use the --yes flag.

Example:
  vault delete old-account
  vault delete temp-entry --yes`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if cfg != nil && vaultPath == "" {
				vaultPath = cfg.VaultPath
			}
			if cfg != nil && profile == "" {
				profile = cfg.DefaultProfile
			}
			return runDelete(args[0])
		},
	}

	cmd.Flags().BoolVar(&deleteYes, "yes", false, "Skip confirmation prompt")

	return cmd
}

func runDelete(entryName string) error {
	// Helper function to write output with error checking
	writeOutput := func(w io.Writer, format string, args ...interface{}) error {
		_, err := fmt.Fprintf(w, format, args...)
		if err != nil {
			return fmt.Errorf("failed to write output: %w", err)
		}
		return nil
	}

	// Check if vault is unlocked
	if !IsUnlocked() {
		return fmt.Errorf("vault is locked, run 'vault unlock' first")
	}

	vaultStore := GetVaultStore()

	// Check if entry exists
	if !vaultStore.EntryExists(profile, entryName) {
		return fmt.Errorf("entry '%s' does not exist in profile '%s'", entryName, profile)
	}

	// Confirm deletion unless --yes flag is used
	if !deleteYes {
		confirmMsg := fmt.Sprintf("Delete entry '%s' from profile '%s'?", entryName, profile)
		confirmed, err := PromptConfirm(confirmMsg, false)
		if err != nil {
			return fmt.Errorf("failed to get confirmation: %w", err)
		}

		if !confirmed {
			if err := writeOutput(os.Stdout, "Entry deletion cancelled\n"); err != nil {
				return err
			}
			return nil
		}
	}

	// Delete entry
	if err := vaultStore.DeleteEntry(profile, entryName); err != nil {
		return fmt.Errorf("failed to delete entry: %w", err)
	}

	// Refresh session
	RefreshSession()

	if err := writeOutput(os.Stdout, "✓ Entry '%s' deleted successfully from profile '%s'\n", entryName, profile); err != nil {
		return err
	}
	return nil
}
