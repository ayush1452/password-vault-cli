package cli

import (
	"fmt"

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
		fmt.Println("Vault is already locked")
		return nil
	}

	if err := LockVault(); err != nil {
		return fmt.Errorf("failed to lock vault: %w", err)
	}

	fmt.Println("✓ Vault locked successfully")
	return nil
}
