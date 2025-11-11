package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/vault-cli/vault/internal/config"
	"github.com/vault-cli/vault/internal/crypto"
	"github.com/vault-cli/vault/internal/domain"
)

// NewRotatePasswordCommand creates a new command for rotating passwords
func NewRotatePasswordCommand(cfg *config.Config) *cobra.Command {
	var (
		length     int
		copyToClip bool
		ttl        int
		show       bool
	)

	cmd := &cobra.Command{
		Use:   "rotate NAME",
		Short: "Regenerate password for an existing entry",
		Long: `Regenerates the password for an existing entry while preserving other metadata.

This command generates a new secure password for the specified entry and updates it
in the vault. The original creation timestamp is preserved, but the updated_at timestamp
is updated to the current time.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRotatePassword(cmd, args[0], length, copyToClip, ttl, show, cfg)
		},
	}

	// Add flags
	cmd.Flags().IntVarP(&length, "length", "l", 20, "Length of the new password")
	cmd.Flags().BoolVarP(&copyToClip, "copy", "c", false, "Copy password to clipboard")
	cmd.Flags().IntVar(&ttl, "ttl", -1, "Time in seconds before clipboard is cleared (-1 uses config default)")
	cmd.Flags().BoolVarP(&show, "show", "s", false, "Show the new password in output")

	// Mark flags as mutually exclusive where needed
	cmd.MarkFlagsMutuallyExclusive("copy", "show")

	return cmd
}

func runRotatePassword(cmd *cobra.Command, name string, length int, copyToClip bool, ttl int, show bool, cfg *config.Config) error {
	// Check if vault is unlocked
	if !IsUnlocked() {
		return fmt.Errorf("vault is locked, run 'vault unlock' first")
	}

	// Get the vault store
	vaultStore := GetVaultStore()

	// Generate new password using the same logic as passgen
	newPassword, err := crypto.GeneratePassword(length, crypto.CharsetAlnumSym)
	if err != nil {
		return fmt.Errorf("failed to generate password: %w", err)
	}

	// Get the current entry
	entry, err := vaultStore.GetEntry("", name) // Empty string for profile uses default
	if err != nil {
		return fmt.Errorf("failed to get entry: %w", err)
	}

	// Create a copy of the entry with the new password
	updatedEntry := &domain.Entry{
		ID:        entry.ID,
		Name:      entry.Name,
		Username:  entry.Username,
		Password:  []byte(newPassword),
		URL:       entry.URL,
		Notes:     entry.Notes,
		Tags:      entry.Tags,
		TOTPSeed:  entry.TOTPSeed,
		CreatedAt: entry.CreatedAt,
		UpdatedAt: time.Now(),
	}

	// Save the updated entry
	err = vaultStore.UpdateEntry("", name, updatedEntry) // Empty string for profile uses default
	if err != nil {
		return fmt.Errorf("failed to update entry: %w", err)
	}

	// Handle output and clipboard
	if show {
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), newPassword); err != nil {
			return fmt.Errorf("failed to write password to output: %w", err)
		}
	}

	if copyToClip {
		ttlDuration := time.Duration(ttl) * time.Second
		if ttl == -1 {
			ttlDuration = cfg.ClipboardTTL
		}

		if err := copyToClipboard(newPassword, ttlDuration); err != nil {
			return fmt.Errorf("failed to copy to clipboard: %w", err)
		}
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "✓ Password rotated and copied to clipboard (clears in %s)\n", ttlDuration.Round(time.Second)); err != nil {
			return fmt.Errorf("failed to write status to output: %w", err)
		}
	} else if !show {
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), "✓ Password rotated successfully"); err != nil {
			return fmt.Errorf("failed to write status to output: %w", err)
		}
	}

	return nil
}
