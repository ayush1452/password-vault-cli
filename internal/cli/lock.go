package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/vault-cli/vault/internal/config"
)

var lockCmd = &cobra.Command{
	Use:   "lock",
	Short: "Lock the vault",
	Long: `Lock the vault and clear the master key from memory.

After locking, you'll need to unlock the vault again with your
master passphrase to access your entries.

Example:
  vault lock`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runLock()
	},
}

// NewLockCommand creates a new lock command for testing
func NewLockCommand(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lock",
		Short: "Lock the vault",
		Long: `Lock the vault and clear the master key from memory.

After locking, you'll need to unlock the vault again with your
master passphrase to access your entries.

Example:
  vault lock`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLock()
		},
	}

	return cmd
}

func runLock() error {
	if !IsUnlocked() {
		if err := writeOutput(os.Stdout, "Vault is already locked\n"); err != nil {
			return fmt.Errorf("failed to write status: %w", err)
		}
		return nil
	}

	if err := LockVault(); err != nil {
		return fmt.Errorf("failed to lock vault: %w", err)
	}

	if err := writeOutput(os.Stdout, "✓ Vault locked successfully\n"); err != nil {
		return fmt.Errorf("failed to write success message: %w", err)
	}
	return nil
}
