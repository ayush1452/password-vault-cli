package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/vault-cli/vault/internal/config"
)

// NewProfiles creates a new profiles command
func NewProfiles(cfg *config.Config) *cobra.Command {
	profilesCmd := &cobra.Command{
		Use:   "profiles",
		Short: "Manage vault profiles",
		Long: `Manage vault profiles for organizing entries by environment or category.

Profiles allow you to separate entries into different namespaces, such as
'work', 'personal', 'production', 'development', etc.

Example:
  vault profiles list                    # List all profiles
  vault profiles create production       # Create production profile
  vault profiles delete old-project     # Delete a profile
  vault profiles set-default work       # Set default profile`,
	}

	profilesCmd.AddCommand(newProfilesListCmd())
	profilesCmd.AddCommand(newProfilesCreateCmd())
	profilesCmd.AddCommand(newProfilesDeleteCmd())
	profilesCmd.AddCommand(newProfilesRenameCmd())
	profilesCmd.AddCommand(newProfilesSetDefaultCmd())

	return profilesCmd
}

func newProfilesListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all profiles",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProfilesList(cmd.OutOrStdout())
		},
	}
}

var profileCreateDescription string

func newProfilesCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Use flag value, don't prompt if empty (for non-interactive use)
			desc, _ := cmd.Flags().GetString("description")
			return runProfilesCreateNonInteractive(args[0], desc)
		},
	}
	cmd.Flags().StringVar(&profileCreateDescription, "description", "", "Profile description")
	return cmd
}

var profileDeleteYes bool

func newProfilesDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			yes, _ := cmd.Flags().GetBool("yes")
			return runProfilesDeleteWithFlag(args[0], yes)
		},
	}
	cmd.Flags().BoolVar(&profileDeleteYes, "yes", false, "Skip confirmation prompt")
	return cmd
}

func newProfilesRenameCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rename <old-name> <new-name>",
		Short: "Rename a profile",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProfilesRename(args[0], args[1])
		},
	}
}

func newProfilesSetDefaultCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set-default <name>",
		Short: "Set the default profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProfilesSetDefault(args[0])
		},
	}
}

var profilesCmd = &cobra.Command{
	Use:   "profiles",
	Short: "Manage vault profiles",
	Long: `Manage vault profiles for organizing entries by environment or category.

Profiles allow you to separate entries into different namespaces, such as
'work', 'personal', 'production', 'development', etc.

Example:
  vault profiles list                    # List all profiles
  vault profiles create production       # Create production profile
  vault profiles delete old-project     # Delete a profile
  vault profiles set-default work       # Set default profile`,
}

var profilesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all profiles",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runProfilesList(cmd.OutOrStdout())
	},
}

var profilesCreateCmd = &cobra.Command{
	Use:   "create <name> [description]",
	Short: "Create a new profile",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		description := ""
		if len(args) > 1 {
			description = args[1]
		}
		return runProfilesCreate(args[0], description)
	},
}

var profilesDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runProfilesDelete(args[0])
	},
}

var profilesRenameCmd = &cobra.Command{
	Use:   "rename <old-name> <new-name>",
	Short: "Rename a profile",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runProfilesRename(args[0], args[1])
	},
}

var profilesSetDefaultCmd = &cobra.Command{
	Use:   "set-default <name>",
	Short: "Set the default profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runProfilesSetDefault(args[0])
	},
}

func init() {
	profilesCmd.AddCommand(profilesListCmd)
	profilesCmd.AddCommand(profilesCreateCmd)
	profilesCmd.AddCommand(profilesDeleteCmd)
	profilesCmd.AddCommand(profilesRenameCmd)
	profilesCmd.AddCommand(profilesSetDefaultCmd)
}

func runProfilesList(out io.Writer) error {
	// Helper function to write output with error checking
	writeOutput := func(w io.Writer, format string, args ...interface{}) error {
		_, err := fmt.Fprintf(w, format, args...)
		if err != nil {
			return fmt.Errorf("failed to write output: %w", err)
		}
		return nil
	}

	// Check if vault is unlocked
	if !IsUnlocked() {
		return fmt.Errorf("vault is locked, run 'vault unlock' first")
	}

	vaultStore := GetVaultStore()

	// Get profiles
	profiles, err := vaultStore.ListProfiles()
	if err != nil {
		return fmt.Errorf("failed to list profiles: %w", err)
	}

	// Refresh session
	RefreshSession()

	// Close session store to release lock file
	if err := CloseSessionStore(); err != nil {
		logWarning("Failed to close session store after profile list: %v", err)
	}

	if len(profiles) == 0 {
		if err := writeOutput(out, "No profiles found\n"); err != nil {
			return err
		}
		return nil
	}

	// Sort profiles by name
	sort.Slice(profiles, func(i, j int) bool {
		return profiles[i].Name < profiles[j].Name
	})

	if outputJSON {
		// Close session store to release lock file
		if err := CloseSessionStore(); err != nil {
			logWarning("Failed to close session store after profile list: %v", err)
		}
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(profiles); err != nil {
			return fmt.Errorf("failed to encode profiles to JSON: %w", err)
		}
		return nil
	}

	// Table output
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	defer func() {
		if err := w.Flush(); err != nil {
			log.Printf("warning: failed to flush tabwriter: %v\n", err)
		}
	}()

	// Write table header
	headers := []string{"NAME", "DESCRIPTION", "CREATED", "DEFAULT"}
	headerLine := strings.Join(headers, "\t") + "\n"
	separator := strings.Repeat("-", 4) + "\t" +
		strings.Repeat("-", 11) + "\t" +
		strings.Repeat("-", 7) + "\t" +
		strings.Repeat("-", 6) + "\n"

	if err := writeOutput(w, "%s", headerLine); err != nil {
		return fmt.Errorf("failed to write table header: %w", err)
	}
	if err := writeOutput(w, "%s", separator); err != nil {
		return fmt.Errorf("failed to write header separator: %w", err)
	}

	// Write table rows
	for _, profile := range profiles {
		isDefault := ""
		if profile.Name == cfg.DefaultProfile {
			isDefault = "✓"
		}

		if err := writeOutput(w, "%s\t%s\t%s\t%s\n",
			profile.Name,
			profile.Description,
			profile.CreatedAt.Format("2006-01-02"),
			isDefault,
		); err != nil {
			return fmt.Errorf("failed to write profile '%s' (name: %s, description: %s, created_at: %s, is_default: %s): %w", profile.Name, profile.Name, profile.Description, profile.CreatedAt.Format("2006-01-02"), isDefault, err)
		}
	}

	// Write summary
	if err := writeOutput(os.Stdout, "\nFound %d profiles\n", len(profiles)); err != nil {
		return fmt.Errorf("failed to write summary for %d profiles: %w", len(profiles), err)
	}

	// Close session store to release lock file (already closed above, but safe to call again)
	if err := CloseSessionStore(); err != nil {
		logWarning("Failed to close session store after profile list: %v", err)
	}

	return nil
}

func runProfilesCreate(name, description string) error {
	// Check if vault is unlocked
	if !IsUnlocked() {
		return fmt.Errorf("vault is locked, run 'vault unlock' first")
	}

	vaultStore := GetVaultStore()

	// Check if profile already exists
	if vaultStore.ProfileExists(name) {
		return fmt.Errorf("profile '%s' already exists", name)
	}

	// Prompt for description if not provided
	if description == "" {
		var err error
		description, err = PromptInput(fmt.Sprintf("Description for profile '%s' (optional): ", name))
		if err != nil {
			return fmt.Errorf("failed to read description: %w", err)
		}
	}

	// Create profile
	if err := vaultStore.CreateProfile(name, description); err != nil {
		return fmt.Errorf("failed to create profile: %w", err)
	}

	// Refresh session
	RefreshSession()

	// Close session store to release lock file
	if err := CloseSessionStore(); err != nil {
		logWarning("Failed to close session store after profile creation: %v", err)
	}

	fmt.Printf("✓ Profile '%s' created successfully\n", name)
	return nil
}

func runProfilesDelete(name string) error {
	// Check if vault is unlocked
	if !IsUnlocked() {
		return fmt.Errorf("vault is locked, run 'vault unlock' first")
	}

	if name == "default" {
		return fmt.Errorf("cannot delete the default profile")
	}

	vaultStore := GetVaultStore()

	// Check if profile exists
	if !vaultStore.ProfileExists(name) {
		return fmt.Errorf("profile '%s' does not exist", name)
	}

	// Get entries count for confirmation
	entries, err := vaultStore.ListEntries(name, nil)
	if err != nil {
		return fmt.Errorf("failed to check profile entries: %w", err)
	}

	// Confirm deletion
	if len(entries) > 0 {
		fmt.Printf("⚠️  Profile '%s' contains %d entries\n", name, len(entries))
	}

	confirmed, err := PromptConfirm(fmt.Sprintf("Delete profile '%s'?", name), false)
	if err != nil {
		return fmt.Errorf("failed to get confirmation: %w", err)
	}

	if !confirmed {
		fmt.Println("Profile deletion canceled")
		return nil
	}

	// Delete profile
	if err := vaultStore.DeleteProfile(name); err != nil {
		return fmt.Errorf("failed to delete profile: %w", err)
	}

	// Refresh session
	RefreshSession()

	// Close session store to release lock file
	if err := CloseSessionStore(); err != nil {
		logWarning("Failed to close session store after profile deletion: %v", err)
	}

	fmt.Printf("✓ Profile '%s' deleted successfully\n", name)
	return nil
}

func runProfilesRename(oldName, newName string) error {
	// This would require more complex implementation in the storage layer
	// For now, return an error indicating it's not implemented
	return fmt.Errorf("profile renaming is not yet implemented")
}

func runProfilesSetDefault(name string) error {
	// Check if vault is unlocked
	if !IsUnlocked() {
		return fmt.Errorf("vault is locked, run 'vault unlock' first")
	}

	vaultStore := GetVaultStore()

	// Check if profile exists
	if !vaultStore.ProfileExists(name) {
		return fmt.Errorf("profile '%s' does not exist", name)
	}

	// Update configuration
	cfg.DefaultProfile = name

	// Save configuration
	if err := config.SaveConfig(cfg, cfgFile); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	// Close session store to release lock file
	if err := CloseSessionStore(); err != nil {
		logWarning("Failed to close session store after setting default profile: %v", err)
	}

	fmt.Printf("✓ Default profile set to '%s'\n", name)
	return nil
}

// runProfilesCreateNonInteractive creates a profile without prompting for description
func runProfilesCreateNonInteractive(name, description string) error {
	// Check if vault is unlocked
	if !IsUnlocked() {
		return fmt.Errorf("vault is locked, run 'vault unlock' first")
	}

	vaultStore := GetVaultStore()

	// Check if profile already exists
	if vaultStore.ProfileExists(name) {
		return fmt.Errorf("profile '%s' already exists", name)
	}

	// Create profile (don't prompt for description - use what's provided)
	if err := vaultStore.CreateProfile(name, description); err != nil {
		return fmt.Errorf("failed to create profile: %w", err)
	}

	// Refresh session
	RefreshSession()

	// Close session store to release lock file
	if err := CloseSessionStore(); err != nil {
		logWarning("Failed to close session store after profile creation: %v", err)
	}

	fmt.Printf("✓ Profile '%s' created successfully\n", name)
	return nil
}

// runProfilesDeleteWithFlag deletes a profile with optional --yes flag
func runProfilesDeleteWithFlag(name string, skipConfirm bool) error {
	// Check if vault is unlocked
	if !IsUnlocked() {
		return fmt.Errorf("vault is locked, run 'vault unlock' first")
	}

	if name == "default" {
		return fmt.Errorf("cannot delete the default profile")
	}

	vaultStore := GetVaultStore()

	// Check if profile exists
	if !vaultStore.ProfileExists(name) {
		return fmt.Errorf("profile '%s' does not exist", name)
	}

	// Get entries count for confirmation
	entries, err := vaultStore.ListEntries(name, nil)
	if err != nil {
		return fmt.Errorf("failed to check profile entries: %w", err)
	}

	// Confirm deletion unless --yes flag is used
	if !skipConfirm {
		if len(entries) > 0 {
			fmt.Printf("⚠️  Profile '%s' contains %d entries\n", name, len(entries))
		}

		confirmed, err := PromptConfirm(fmt.Sprintf("Delete profile '%s'?", name), false)
		if err != nil {
			return fmt.Errorf("failed to get confirmation: %w", err)
		}

		if !confirmed {
			fmt.Println("Profile deletion canceled")
			return nil
		}
	}

	// Delete profile
	if err := vaultStore.DeleteProfile(name); err != nil {
		return fmt.Errorf("failed to delete profile: %w", err)
	}

	// Refresh session
	RefreshSession()

	// Close session store to release lock file
	if err := CloseSessionStore(); err != nil {
		logWarning("Failed to close session store after profile deletion: %v", err)
	}

	fmt.Printf("✓ Profile '%s' deleted successfully\n", name)
	return nil
}
