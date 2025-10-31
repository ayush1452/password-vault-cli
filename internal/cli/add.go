package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vault-cli/vault/internal/config"
	"github.com/vault-cli/vault/internal/domain"
)

var (
	username     string
	secretPrompt bool
	secretFile   string
	url          string
	notes        string
	tags         []string
	totpSeed     string
)

var addCmd = &cobra.Command{
	Use:   "add <entry-name>",
	Short: "Add a new entry to the vault",
	Long: `Add a new entry to the vault with the specified name.

You can provide the secret via interactive prompt (--secret-prompt),
from a file (--secret-file), or it will be prompted automatically.

Example:
  vault add github --username user@example.com --secret-prompt
  vault add aws --username admin --secret-file secret.txt --url https://console.aws.amazon.com
  vault add database --username dbuser --notes "Production database" --tags prod,db`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAdd(args[0])
	},
}

func init() {
	addCmd.Flags().StringVar(&username, "username", "", "Username/email")
	addCmd.Flags().BoolVar(&secretPrompt, "secret-prompt", false, "Prompt for secret interactively")
	addCmd.Flags().StringVar(&secretFile, "secret-file", "", "Read secret from file")
	addCmd.Flags().StringVar(&url, "url", "", "Associated URL")
	addCmd.Flags().StringVar(&notes, "notes", "", "Additional notes")
	addCmd.Flags().StringSliceVar(&tags, "tags", nil, "Comma-separated tags")
	addCmd.Flags().StringVar(&totpSeed, "totp-seed", "", "TOTP seed (base32)")
}

// NewAddCommand creates a new add command for testing
func NewAddCommand(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <entry-name>",
		Short: "Add a new entry to the vault",
		Long: `Add a new entry to the vault with the specified name.

You can provide the secret via interactive prompt (--secret-prompt),
from a file (--secret-file), or it will be prompted automatically.

Example:
  vault add github --username user@example.com --secret-prompt
  vault add aws --username admin --secret-file secret.txt --url https://console.aws.amazon.com
  vault add database --username dbuser --notes "Production database" --tags prod,db`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if cfg != nil && vaultPath == "" {
				vaultPath = cfg.VaultPath
			}
			if cfg != nil && profile == "" {
				profile = cfg.DefaultProfile
			}
			return runAdd(args[0])
		},
	}

	cmd.Flags().StringVar(&username, "username", "", "Username/email")
	cmd.Flags().BoolVar(&secretPrompt, "secret-prompt", false, "Prompt for secret interactively")
	cmd.Flags().StringVar(&secretFile, "secret-file", "", "Read secret from file")
	cmd.Flags().StringVar(&url, "url", "", "Associated URL")
	cmd.Flags().StringVar(&notes, "notes", "", "Additional notes")
	cmd.Flags().StringSliceVar(&tags, "tags", nil, "Comma-separated tags")
	cmd.Flags().StringVar(&totpSeed, "totp-seed", "", "TOTP seed (base32)")

	return cmd
}

func runAdd(entryName string) error {
	defer CloseSessionStore()

	// Check if vault is unlocked
	if !IsUnlocked() {
		return fmt.Errorf("vault is locked, run 'vault unlock' first")
	}

	vaultStore := GetVaultStore()

	// Check if entry already exists
	if vaultStore.EntryExists(profile, entryName) {
		return fmt.Errorf("entry '%s' already exists in profile '%s'", entryName, profile)
	}

	// Get secret
	var secret string
	var err error

	if secretFile != "" {
		// Read from file
		var data []byte
		if secretFile == "-" {
			data, err = io.ReadAll(os.Stdin)
		} else {
			data, err = os.ReadFile(secretFile)
		}
		if err != nil {
			return fmt.Errorf("failed to read secret file: %w", err)
		}
		secret = strings.TrimSpace(string(data))
	} else if secretPrompt || secret == "" {
		// Prompt for secret
		secret, err = PromptPassword("Enter secret: ")
		if err != nil {
			return fmt.Errorf("failed to read secret: %w", err)
		}
	}

	if secret == "" {
		return fmt.Errorf("secret cannot be empty")
	}

	// Prompt for username if not provided
	if username == "" {
		username, err = PromptInput("Username (optional): ")
		if err != nil {
			return fmt.Errorf("failed to read username: %w", err)
		}
	}

	// Create entry
	entry := &domain.Entry{
		Name:     entryName,
		Username: username,
		Secret:   []byte(secret),
		URL:      url,
		Notes:    notes,
		Tags:     tags,
	}

	// Add TOTP seed if provided
	if totpSeed != "" {
		entry.TOTPSeed = totpSeed
	}

	// Save entry
	if err := vaultStore.CreateEntry(profile, entry); err != nil {
		return fmt.Errorf("failed to create entry: %w", err)
	}

	// Refresh session
	RefreshSession()

	fmt.Printf("✓ Entry '%s' added successfully to profile '%s'\n", entryName, profile)

	if verbose {
		fmt.Printf("Details:\n")
		fmt.Printf("  Username: %s\n", username)
		fmt.Printf("  URL: %s\n", url)
		fmt.Printf("  Tags: %v\n", tags)
		if notes != "" {
			fmt.Printf("  Notes: %s\n", notes)
		}
	}

	return nil
}
