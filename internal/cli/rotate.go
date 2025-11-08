package cli

import (
	"crypto/subtle"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/vault-cli/vault/internal/vault"
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

	// Get the current vault store
	vaultStore := GetVaultStore()
	if vaultStore == nil {
		return fmt.Errorf("no active vault store")
	}

	// Prompt for current passphrase to verify identity
	fmt.Println("Verifying current passphrase...")
	currentPassphrase, err := PromptPassword("Enter current passphrase: ")
	if err != nil {
		return fmt.Errorf("failed to read current passphrase: %w", err)
	}

	if currentPassphrase == "" {
		return fmt.Errorf("passphrase cannot be empty")
	}

	metadata, err := vaultStore.GetVaultMetadata()
	if err != nil {
		return fmt.Errorf("failed to load vault metadata: %w", err)
	}

	params, salt, err := metadataToKeyParams(metadata)
	if err != nil {
		return fmt.Errorf("invalid vault metadata: %w", err)
	}

	cryptoEngine := vault.NewCryptoEngine(params)
	derivedKey, err := cryptoEngine.DeriveKey(currentPassphrase, salt)
	if err != nil {
		return fmt.Errorf("failed to derive master key: %w", err)
	}
	defer vault.Zeroize(derivedKey)

	if sessionManager == nil || sessionManager.masterKey == nil || subtle.ConstantTimeCompare(derivedKey, sessionManager.masterKey) != 1 {
		return fmt.Errorf("incorrect current passphrase")
	}

	// Prompt for new passphrase
	fmt.Println("\nSet a new master passphrase")
	newPassphrase, err := PromptPassword("Enter new passphrase: ")
	if err != nil {
		return fmt.Errorf("failed to read new passphrase: %w", err)
	}

	if newPassphrase == "" {
		return fmt.Errorf("new passphrase cannot be empty")
	}

	// Confirm new passphrase
	confirmPassphrase, err := PromptPassword("Confirm new passphrase: ")
	if err != nil {
		return fmt.Errorf("failed to confirm new passphrase: %w", err)
	}

	if newPassphrase != confirmPassphrase {
		return fmt.Errorf("passphrases do not match")
	}

	// Perform the key rotation
	fmt.Println("\nRotating master key...")

	if err := vaultStore.RotateMasterKey(newPassphrase); err != nil {
		return fmt.Errorf("failed to rotate master key: %w", err)
	}

	// Lock the vault to force re-authentication with the new passphrase
	if err := LockVault(); err != nil {
		return fmt.Errorf("failed to lock vault after rotation: %w", err)
	}

	fmt.Println("\n✅ Master key rotation completed successfully!")
	fmt.Println("Please unlock the vault with your new passphrase using 'vault unlock'")

	return nil
}
