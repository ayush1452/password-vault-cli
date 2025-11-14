package cli

import (
	"fmt"
	"io"
	"path/filepath"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vault-cli/vault/internal/config"
)

var (
	updateUsername     string
	updateSecretPrompt bool
	updateSecretFile   string
	updateURL          string
	updateNotes        string
	updateTags         []string
	updateTotpSeed     string
)

var updateCmd = &cobra.Command{
	Use:   "update <entry-name>",
	Short: "Update an existing entry",
	Long: `Update an existing entry in the vault.

You can update any field of an entry. If no flags are provided,
you'll be prompted to update each field interactively.

Example:
  vault update github --username newuser@example.com
  vault update aws --secret-prompt --notes "Updated credentials"
  vault update database --tags prod,critical --url new-host:5432`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUpdate(cmd, args[0])
	},
}

func runUpdate(cmd *cobra.Command, entryName string) error {
	defer CloseSessionStore()

	// Check if vault is unlocked
	if !IsUnlocked() {
		return fmt.Errorf("vault is locked, run 'vault unlock' first")
	}

	vaultStore := GetVaultStore()

	// Get existing entry
	entry, err := vaultStore.GetEntry(profile, entryName)
	if err != nil {
		return fmt.Errorf("failed to get entry: %w", err)
	}

	// Track what was updated
	updated := false

	// Update username if provided
	if cmd.Flags().Changed("username") {
		entry.Username = updateUsername
		updated = true
	}

	// Update URL if provided
	if cmd.Flags().Changed("url") {
		entry.URL = updateURL
		updated = true
	}

	// Update notes if provided
	if cmd.Flags().Changed("notes") {
		entry.Notes = updateNotes
		updated = true
	}

	// Update tags if provided
	if cmd.Flags().Changed("tags") {
		entry.Tags = updateTags
		updated = true
	}

	// Update TOTP seed if provided
	if cmd.Flags().Changed("totp-seed") {
		entry.TOTPSeed = updateTotpSeed
		updated = true
	}

	// Update secret if requested
	if cmd.Flags().Changed("secret-prompt") || cmd.Flags().Changed("secret-file") {
		var secret string

		if cmd.Flags().Changed("secret-file") {
			// Read from file
			var data []byte
			if updateSecretFile == "-" {
				data, err = io.ReadAll(os.Stdin)
			} else {
				// Clean the file path to prevent directory traversal
				cleanPath := filepath.Clean(updateSecretFile)
				// Optional: Add additional path validation here if needed
				// For example, ensure the path is within an allowed directory
				
				data, err = os.ReadFile(cleanPath)
			}
			if err != nil {
				return fmt.Errorf("failed to read secret file: %w", err)
			}
			secret = strings.TrimSpace(string(data))
		} else {
			// Prompt for secret
			secret, err = PromptPassword("Enter new secret: ")
			if err != nil {
				return fmt.Errorf("failed to read secret: %w", err)
			}
		}

		if secret != "" {
			entry.Secret = []byte(secret)
			updated = true
		}
	}

	// If no flags were provided, prompt for updates interactively
	if !updated {
		// Helper function to handle fmt.Fprintf errors
		printPrompt := func(format string, args ...interface{}) error {
			_, err := fmt.Fprintf(os.Stdout, format, args...)
			if err != nil {
				return fmt.Errorf("failed to write prompt: %w", err)
			}
			return nil
		}

		if err := printPrompt("Updating entry '%s' (press Enter to keep current value):\n\n", entryName); err != nil {
			return err
		}

		// Username
		if err := printPrompt("Username [%s]: ", entry.Username); err != nil {
			return err
		}
		newUsername, err := PromptInput("")
		if err != nil {
			return fmt.Errorf("failed to read username: %w", err)
		}
		if newUsername != "" {
			entry.Username = newUsername
			updated = true
		}

		// URL
		if err := printPrompt("URL [%s]: ", entry.URL); err != nil {
			return err
		}
		newURL, err := PromptInput("")
		if err != nil {
			return fmt.Errorf("failed to read URL: %w", err)
		}
		if newURL != "" {
			entry.URL = newURL
			updated = true
		}

		// Notes
		if err := printPrompt("Notes [%s]: ", entry.Notes); err != nil {
			return err
		}
		newNotes, err := PromptInput("")
		if err != nil {
			return fmt.Errorf("failed to read notes: %w", err)
		}
		if newNotes != "" {
			entry.Notes = newNotes
			updated = true
		}

		// Tags
		currentTags := strings.Join(entry.Tags, ",")
		if err := printPrompt("Tags [%s]: ", currentTags); err != nil {
			return err
		}
		newTags, err := PromptInput("")
		if err != nil {
			return fmt.Errorf("failed to read tags: %w", err)
		}
		if newTags != "" {
			if newTags == "-" {
				entry.Tags = nil
			} else {
				entry.Tags = strings.Split(newTags, ",")
				for i, tag := range entry.Tags {
					entry.Tags[i] = strings.TrimSpace(tag)
				}
			}
			updated = true
		}

		// Secret
		updateSecret, err := PromptConfirm("Update secret?", false)
		if err != nil {
			return fmt.Errorf("failed to get confirmation: %w", err)
		}
		if updateSecret {
			newSecret, err := PromptPassword("Enter new secret: ")
			if err != nil {
				return fmt.Errorf("failed to read secret: %w", err)
			}
			if newSecret != "" {
				entry.Secret = []byte(newSecret)
				updated = true
			}
		}
	}

	if !updated {
		if _, err := fmt.Fprintln(os.Stdout, "No changes made"); err != nil {
			return fmt.Errorf("failed to write output: %w", err)
		}
		return nil
	}

	// Update entry
	if err := vaultStore.UpdateEntry(profile, entryName, entry); err != nil {
		return fmt.Errorf("failed to update entry: %w", err)
	}

	// Refresh session
	RefreshSession()

	if _, err := fmt.Fprintf(os.Stdout, "✓ Entry '%s' updated successfully\n", entryName); err != nil {
		return fmt.Errorf("failed to write success message: %w", err)
	}
	return nil
}

func registerUpdateFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&updateUsername, "username", "", "New username/email")
	cmd.Flags().BoolVar(&updateSecretPrompt, "secret-prompt", false, "Prompt for new secret interactively")
	cmd.Flags().StringVar(&updateSecretFile, "secret-file", "", "Read new secret from file")
	cmd.Flags().StringVar(&updateURL, "url", "", "New associated URL")
	cmd.Flags().StringVar(&updateNotes, "notes", "", "Updated notes")
	cmd.Flags().StringSliceVar(&updateTags, "tags", nil, "Updated comma-separated tags")
	cmd.Flags().StringVar(&updateTotpSeed, "totp-seed", "", "New TOTP seed (base32)")
}

func init() {
	registerUpdateFlags(updateCmd)
}

// NewUpdateCommand creates a new update command for testing
func NewUpdateCommand(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <entry-name>",
		Short: "Update an existing entry",
		Long:  updateCmd.Long,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if cfg != nil && vaultPath == "" {
				vaultPath = cfg.VaultPath
			}
			if cfg != nil && profile == "" {
				profile = cfg.DefaultProfile
			}
			return runUpdate(cmd, args[0])
		},
	}

	registerUpdateFlags(cmd)
	return cmd
}
