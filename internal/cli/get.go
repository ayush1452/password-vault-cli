package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vault-cli/vault/internal/clipboard"
	"github.com/vault-cli/vault/internal/config"
)

var (
	field string
	copy  bool
	show  bool
)

var getCmd = &cobra.Command{
	Use:   "get <entry-name>",
	Short: "Get an entry from the vault",
	Long: `Get an entry from the vault and display or copy the specified field.

By default, the secret is copied to clipboard and cleared after 30 seconds.
Use --show to display the secret in the terminal (not recommended).
Use --field to specify which field to retrieve.

Example:
  vault get github                    # Copy secret to clipboard
  vault get github --show             # Display secret in terminal
  vault get github --field username   # Copy username to clipboard
  vault get github --field url --show # Display URL in terminal`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runGet(args[0])
	},
}

func init() {
	getCmd.Flags().StringVar(&field, "field", "secret", "Field to retrieve (secret|username|url|notes)")
	getCmd.Flags().BoolVar(&copy, "copy", false, "Copy to clipboard instead of displaying")
	getCmd.Flags().BoolVar(&show, "show", false, "Show secret in terminal (security warning)")
}

// NewGetCommand creates a new get command for testing
func NewGetCommand(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <entry-name>",
		Short: "Get an entry from the vault",
		Long: `Get an entry from the vault and display or copy the specified field.

By default, the secret is copied to clipboard and cleared after 30 seconds.
Use --show to display the secret in the terminal (not recommended).
Use --field to specify which field to retrieve.

Example:
  vault get github                    # Copy secret to clipboard
  vault get github --show             # Display secret in terminal
  vault get github --field username   # Copy username to clipboard
  vault get github --field url --show # Display URL in terminal`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if cfg != nil && vaultPath == "" {
				vaultPath = cfg.VaultPath
			}
			if cfg != nil && profile == "" {
				profile = cfg.DefaultProfile
			}
			return runGet(args[0])
		},
	}

	cmd.Flags().StringVar(&field, "field", "secret", "Field to retrieve (secret|username|url|notes)")
	cmd.Flags().BoolVar(&copy, "copy", false, "Copy to clipboard instead of displaying")
	cmd.Flags().BoolVar(&show, "show", false, "Show secret in terminal (security warning)")

	return cmd
}

func runGet(entryName string) error {
	defer CloseSessionStore()

	// Check if vault is unlocked
	if !IsUnlocked() {
		return fmt.Errorf("vault is locked, run 'vault unlock' first")
	}

	vaultStore := GetVaultStore()

	// Get entry
	entry, err := vaultStore.GetEntry(profile, entryName)
	if err != nil {
		return fmt.Errorf("failed to get entry: %w", err)
	}

	// Refresh session
	RefreshSession()

	// Get the requested field
	var value string
	var sensitive bool

	switch strings.ToLower(field) {
	case "secret", "password":
		value = string(entry.Secret)
		sensitive = true
	case "username", "user":
		value = entry.Username
	case "url":
		value = entry.URL
	case "notes":
		value = entry.Notes
	default:
		return fmt.Errorf("invalid field: %s (valid: secret, username, url, notes)", field)
	}

	if value == "" {
		fmt.Printf("Field '%s' is empty for entry '%s'\n", field, entryName)
		return nil
	}

	// Handle output based on flags and field sensitivity
	if sensitive && !show && !copy {
		// Default behavior for secrets: copy to clipboard
		copy = true
	}

	if copy || (!show && sensitive) {
		// Copy to clipboard
		if !clipboard.IsAvailable() {
			return fmt.Errorf("clipboard not available, use --show to display in terminal")
		}

		timeout := cfg.ClipboardTTL
		if err := clipboard.CopyWithTimeout(value, timeout); err != nil {
			return fmt.Errorf("failed to copy to clipboard: %w", err)
		}

		fmt.Printf("✓ %s for '%s' copied to clipboard", strings.Title(field), entryName)
		if sensitive {
			fmt.Printf(" (clears in %v)", timeout)
		}
		fmt.Println()
	} else {
		// Display in terminal
		if sensitive {
			fmt.Println("⚠️  WARNING: Displaying secret in terminal")
		}
		fmt.Printf("%s: %s\n", strings.Title(field), value)
	}

	// Show additional info if verbose
	if verbose && field == "secret" {
		fmt.Printf("\nEntry details:\n")
		fmt.Printf("  Name: %s\n", entry.Name)
		if entry.Username != "" {
			fmt.Printf("  Username: %s\n", entry.Username)
		}
		if entry.URL != "" {
			fmt.Printf("  URL: %s\n", entry.URL)
		}
		if len(entry.Tags) > 0 {
			fmt.Printf("  Tags: %v\n", entry.Tags)
		}
		if entry.Notes != "" {
			fmt.Printf("  Notes: %s\n", entry.Notes)
		}
		fmt.Printf("  Created: %s\n", entry.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("  Updated: %s\n", entry.UpdatedAt.Format("2006-01-02 15:04:05"))
	}

	return nil
}
