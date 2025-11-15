package cli

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/vault-cli/vault/internal/config"
)

var unlockTTL time.Duration

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
	// Use the shared writeOutput function for consistent error handling
	writeOutput := func(format string, args ...interface{}) error {
		return writeOutput(os.Stdout, format, args...)
	}

	// Check if already unlocked
	if IsUnlocked() {
		unlocked, remaining, err := GetSessionInfo()
		if err != nil {
			log.Printf("Warning: failed to get session info: %v", err)
			// Continue with unlock process if we can't get session info
		} else if unlocked {
			if err := writeOutput("Vault is already unlocked (expires in %v)\n", remaining.Round(time.Second)); err != nil {
				log.Printf("Warning: failed to write status: %v", err)
			}
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

	// Write success message and timeout info
	errs := []error{
		writeOutput("✓ Vault unlocked successfully\n"),
		writeOutput("Auto-lock timeout: %v\n", unlockTTL),
	}

	// Return the first error if any occurred
	for _, err := range errs {
		if err != nil {
			return fmt.Errorf("error during unlock: %w", err)
		}
	}

	return nil
}
