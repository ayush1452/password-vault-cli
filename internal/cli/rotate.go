package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var rotateMasterKeyCmd = &cobra.Command{
	Use:   "rotate-master-key",
	Short: "Rotate the master passphrase",
	Long: `Rotate the master passphrase while preserving all vault data.

This process:
1. Prompts for the current master passphrase
2. Prompts for a new master passphrase
3. Re-encrypts all vault data with the new key
4. Updates the vault metadata

This operation cannot be undone, so ensure you remember the new passphrase.

Example:
  vault rotate-master-key`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runRotateMasterKey()
	},
}

func runRotateMasterKey() error {
	// Check if vault is unlocked
	if !IsUnlocked() {
		return fmt.Errorf("vault is locked, run 'vault unlock' first")
	}

	// TODO: Implement master key rotation
	return fmt.Errorf("master key rotation is not yet implemented")
}
