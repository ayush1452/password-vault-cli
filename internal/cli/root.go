package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/vault-cli/vault/internal/config"
	"github.com/vault-cli/vault/internal/util"
)

var (
	cfgFile   string
	vaultPath string
	profile   string
	verbose   bool
	cfg       *config.Config
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "vault",
	Short: "A secure, local-only password manager",
	Long: `Vault is a secure, local-only password manager that uses strong cryptography
to protect your credentials. All data is encrypted locally and never transmitted
over the network.

Features:
- AES-256-GCM encryption with Argon2id key derivation
- Profile-based organization (dev, prod, personal, etc.)
- Secure clipboard integration with auto-clear
- Comprehensive audit logging
- Cross-platform support (macOS, Linux, Windows)`,
	Version: "1.0.0",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Load configuration
		var err error
		cfg, err = config.LoadConfig(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Set vault path from config if not provided
		if vaultPath == "" {
			vaultPath = cfg.VaultPath
		}

		// Set profile from config if not provided
		if profile == "" {
			profile = cfg.DefaultProfile
		}

		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/vault/config.yaml)")
	rootCmd.PersistentFlags().StringVar(&vaultPath, "vault", "", "vault database path")
	rootCmd.PersistentFlags().StringVar(&profile, "profile", "", "vault profile to use")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	// Add all subcommands
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(unlockCmd)
	rootCmd.AddCommand(lockCmd)
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(profilesCmd)
	rootCmd.AddCommand(exportCmd)
	rootCmd.AddCommand(importCmd)
	rootCmd.AddCommand(rotateMasterKeyCmd)
	rootCmd.AddCommand(auditLogCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(passgenCmd)
	rootCmd.AddCommand(NewRotatePasswordCommand(cfg))
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag
		return
	}

	// Find home directory
	home, err := os.UserHomeDir()
	if err != nil {
		util.ExitWithCode(util.ExitError, "Failed to get home directory: %v", err)
	}

	// Search config in home directory with name ".vault" (without extension)
	configDir := filepath.Join(home, ".config", "vault")
	cfgFile = filepath.Join(configDir, "config.yaml")
}
