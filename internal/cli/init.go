package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/vault-cli/vault/internal/config"
	"github.com/vault-cli/vault/internal/store"
	"github.com/vault-cli/vault/internal/vault"
)

var (
	kdfMemory      uint32
	kdfIterations  uint32
	kdfParallelism uint8
	force          bool
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new vault",
	Long: `Initialize a new vault with a master passphrase and KDF parameters.

The vault will be created with strong cryptographic defaults:
- Argon2id key derivation function
- AES-256-GCM authenticated encryption
- Secure file permissions (0600)

Example:
  vault init
  vault init --kdf-memory 128 --kdf-iterations 5
  vault init --vault /path/to/vault.db --force`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runInit()
	},
}

func init() {
	initCmd.Flags().Uint32Var(&kdfMemory, "kdf-memory", 65536, "Memory parameter for Argon2id (KB)")
	initCmd.Flags().Uint32Var(&kdfIterations, "kdf-iterations", 3, "Time parameter for Argon2id")
	initCmd.Flags().Uint8Var(&kdfParallelism, "kdf-parallelism", 4, "Parallelism parameter for Argon2id")
	initCmd.Flags().BoolVar(&force, "force", false, "Overwrite existing vault")
}

// NewInitCommand creates a new init command for testing
func NewInitCommand(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new vault",
		Long: `Initialize a new vault with a master passphrase and KDF parameters.

The vault will be created with strong cryptographic defaults:
- Argon2id key derivation function
- AES-256-GCM authenticated encryption
- Secure file permissions (0600)

Example:
  vault init
  vault init --kdf-memory 128 --kdf-iterations 5
  vault init --vault /path/to/vault.db --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if cfg != nil && vaultPath == "" {
				vaultPath = cfg.VaultPath
			}
			return runInit()
		},
	}

	cmd.Flags().Uint32Var(&kdfMemory, "kdf-memory", 65536, "Memory parameter for Argon2id (KB)")
	cmd.Flags().Uint32Var(&kdfIterations, "kdf-iterations", 3, "Time parameter for Argon2id")
	cmd.Flags().Uint8Var(&kdfParallelism, "kdf-parallelism", 4, "Parallelism parameter for Argon2id")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing vault")

	return cmd
}

func runInit() error {
	// Expand vault path
	if err := EnsureVaultDirectory(vaultPath); err != nil {
		return fmt.Errorf("failed to create vault directory: %w", err)
	}

	// Check if vault already exists
	if _, err := os.Stat(vaultPath); err == nil && !force {
		return fmt.Errorf("vault already exists at %s (use --force to overwrite)", vaultPath)
	}

	// Validate KDF parameters
	kdfParams := vault.Argon2Params{
		Memory:      kdfMemory,
		Iterations:  kdfIterations,
		Parallelism: kdfParallelism,
	}

	if err := vault.ValidateArgon2Params(kdfParams); err != nil {
		return fmt.Errorf("invalid KDF parameters: %w", err)
	}

	// Prompt for master passphrase
	fmt.Println("Creating new vault...")
	fmt.Println("Choose a strong master passphrase. This will be used to encrypt all your data.")

	passphrase, err := PromptPasswordConfirm("Enter master passphrase: ")
	if err != nil {
		return fmt.Errorf("failed to get passphrase: %w", err)
	}

	if len(passphrase) < 8 {
		return fmt.Errorf("passphrase must be at least 8 characters long")
	}

	// Generate salt
	salt, err := vault.GenerateSalt()
	if err != nil {
		return fmt.Errorf("failed to generate salt: %w", err)
	}

	// Derive master key
	crypto := vault.NewCryptoEngine(kdfParams)
	masterKey, err := crypto.DeriveKey(passphrase, salt)
	if err != nil {
		return fmt.Errorf("failed to derive master key: %w", err)
	}
	defer vault.Zeroize(masterKey)

	// Create KDF params map
	kdfParamsMap := map[string]interface{}{
		"memory":      kdfParams.Memory,
		"iterations":  kdfParams.Iterations,
		"parallelism": kdfParams.Parallelism,
		"salt":        salt,
	}

	// Create vault
	vaultStore := store.NewBoltStore()
	if err := vaultStore.CreateVault(vaultPath, masterKey, kdfParamsMap); err != nil {
		return fmt.Errorf("failed to create vault: %w", err)
	}

	fmt.Printf("✓ Vault created successfully at %s\n", vaultPath)
	fmt.Printf("KDF Parameters:\n")
	fmt.Printf("  Memory: %d KB\n", kdfMemory)
	fmt.Printf("  Iterations: %d\n", kdfIterations)
	fmt.Printf("  Parallelism: %d\n", kdfParallelism)
	fmt.Printf("\nUse 'vault unlock' to start using your vault.\n")

	return nil
}
