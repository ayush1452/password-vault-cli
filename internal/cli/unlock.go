package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/vault-cli/vault/internal/config"
)

var (
	unlockTTL time.Duration
)

var unlockCmd = &cobra.Command{
	Use:   "unlock",
	Short: "Unlock the vault",
	Long: `Unlock the vault with your master passphrase.

The vault will remain unlocked for the specified duration (default 1 hour).
After the timeout, you'll need to unlock it again.

Example:
  vault unlock
  vault unlock --ttl 30m
  vault unlock --ttl 2h`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUnlock()
	},
}

func init() {
	unlockCmd.Flags().DurationVar(&unlockTTL, "ttl", time.Hour, "Auto-lock timeout")
}

// NewUnlockCommand creates a new unlock command for testing
func NewUnlockCommand(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unlock",
		Short: "Unlock the vault",
		Long: `Unlock the vault with your master passphrase.

The vault will remain unlocked for the specified duration (default 1 hour).
After the timeout, you'll need to unlock it again.

Example:
  vault unlock
  vault unlock --ttl 30m
  vault unlock --ttl 2h`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if cfg != nil && vaultPath == "" {
				vaultPath = cfg.VaultPath
			}
			return runUnlock()
		},
	}

	cmd.Flags().DurationVar(&unlockTTL, "ttl", time.Hour, "Auto-lock timeout")

	return cmd
}

func runUnlock() error {
	// Check if already unlocked
	if IsUnlocked() {
		unlocked, remaining, _ := GetSessionInfo()
		if unlocked {
			fmt.Printf("Vault is already unlocked (expires in %v)\n", remaining.Round(time.Second))
			return nil
		}
	}

	// Prompt for passphrase
	passphrase, err := PromptPassword("Enter master passphrase: ")
	if err != nil {
		return fmt.Errorf("failed to read passphrase: %w", err)
	}

	// Unlock vault
	if err := UnlockVault(vaultPath, passphrase, unlockTTL); err != nil {
		return fmt.Errorf("failed to unlock vault: %w", err)
	}

	fmt.Printf("✓ Vault unlocked successfully\n")
	fmt.Printf("Auto-lock timeout: %v\n", unlockTTL)

	return nil
}
