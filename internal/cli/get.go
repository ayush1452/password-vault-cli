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
		return runGet(cmd, args[0])
	},
}

func init() {
	getCmd.Flags().StringVar(&field, "field", "secret", "Field to retrieve (secret|username|url|notes)")
	getCmd.Flags().BoolVar(&copy, "copy", false, "Copy to clipboard instead of displaying")
	getCmd.Flags().BoolVar(&show, "show", false, "Show secret in terminal (security warning)")
}

// NewGetCommand creates a new get command for testing
func NewGetCommand(cfg *config.Config) *cobra.Command {
	var cmd = &cobra.Command{
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
			return runGet(cmd, args[0])
		},
	}

	cmd.Flags().StringVar(&field, "field", "secret", "Field to retrieve (secret|username|url|notes)")
	cmd.Flags().BoolVar(&copy, "copy", false, "Copy to clipboard instead of displaying")
	cmd.Flags().BoolVar(&show, "show", false, "Show secret in terminal (security warning)")

	return cmd
}

func runGet(cmd *cobra.Command, entryName string) error {
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
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Field '%s' is empty for entry '%s'\n", field, entryName); err != nil {
			return fmt.Errorf("failed to write output: %w", err)
		}
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

		// Get the output writer once
		out := cmd.OutOrStdout()
		
		// Helper function to write output with error checking
		writeOutput := func(format string, args ...interface{}) error {
			_, err := fmt.Fprintf(out, format, args...)
			return err
		}

		// Write the success message
		if err := writeOutput("✓ %s for '%s' copied to clipboard", strings.Title(field), entryName); err != nil {
			return fmt.Errorf("failed to write output: %w", err)
		}

		// Add timeout info if sensitive
		if sensitive {
			if err := writeOutput(" (clears in %v)", timeout); err != nil {
				return fmt.Errorf("failed to write timeout info: %w", err)
			}
		}

		// Add newline
		if _, err := fmt.Fprintln(out); err != nil {
			return fmt.Errorf("failed to write newline: %w", err)
		}
	} else {
		// Display in terminal
		out := cmd.OutOrStdout()
		
		if sensitive {
			if _, err := fmt.Fprintln(out, "⚠️  WARNING: Displaying secret in terminal"); err != nil {
				return fmt.Errorf("failed to write warning: %w", err)
			}
		}
		
		if _, err := fmt.Fprintf(out, "%s: %s\n", strings.Title(field), value); err != nil {
			return fmt.Errorf("failed to write %s: %w", field, err)
		}
	}

	// Show additional info if verbose
	if verbose && field == "secret" {
		// Get the output writer once
		out := cmd.OutOrStdout()
		
		// Helper function to write output with error checking
		writeOutput := func(format string, args ...interface{}) error {
			_, err := fmt.Fprintf(out, format, args...)
			return err
		}

		// Write entry details with error checking
		if err := writeOutput("\nEntry details:\n"); err != nil {
			return fmt.Errorf("failed to write entry details header: %w", err)
		}

		if err := writeOutput("  Name: %s\n", entry.Name); err != nil {
			return fmt.Errorf("failed to write entry name: %w", err)
		}

		if entry.Username != "" {
			if err := writeOutput("  Username: %s\n", entry.Username); err != nil {
				return fmt.Errorf("failed to write username: %w", err)
			}
		}

		if entry.URL != "" {
			if err := writeOutput("  URL: %s\n", entry.URL); err != nil {
				return fmt.Errorf("failed to write URL: %w", err)
			}
		}

		if len(entry.Tags) > 0 {
			if err := writeOutput("  Tags: %v\n", entry.Tags); err != nil {
				return fmt.Errorf("failed to write tags: %w", err)
			}
		}

		if entry.Notes != "" {
			if err := writeOutput("  Notes: %s\n", entry.Notes); err != nil {
				return fmt.Errorf("failed to write notes: %w", err)
			}
		}

		createdAt := entry.CreatedAt.Format("2006-01-02 15:04:05")
		if err := writeOutput("  Created: %s\n", createdAt); err != nil {
			return fmt.Errorf("failed to write creation time: %w", err)
		}

		updatedAt := entry.UpdatedAt.Format("2006-01-02 15:04:05")
		if err := writeOutput("  Updated: %s\n", updatedAt); err != nil {
			return fmt.Errorf("failed to write update time: %w", err)
		}
	}

	return nil
}
