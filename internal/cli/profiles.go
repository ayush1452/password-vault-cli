package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/vault-cli/vault/internal/config"
)

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
		return runProfilesList()
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

func runProfilesList() error {
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

	if len(profiles) == 0 {
		fmt.Println("No profiles found")
		return nil
	}

	// Sort profiles by name
	sort.Slice(profiles, func(i, j int) bool {
		return profiles[i].Name < profiles[j].Name
	})

	if outputJSON {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(profiles)
	}

	// Table output
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintf(w, "NAME\tDESCRIPTION\tCREATED\tDEFAULT\n")
	fmt.Fprintf(w, "----\t-----------\t-------\t-------\n")

	for _, profile := range profiles {
		isDefault := ""
		if profile.Name == cfg.DefaultProfile {
			isDefault = "✓"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			profile.Name,
			profile.Description,
			profile.CreatedAt.Format("2006-01-02"),
			isDefault,
		)
	}

	fmt.Printf("\nFound %d profiles\n", len(profiles))
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
		fmt.Println("Profile deletion cancelled")
		return nil
	}

	// Delete profile
	if err := vaultStore.DeleteProfile(name); err != nil {
		return fmt.Errorf("failed to delete profile: %w", err)
	}

	// Refresh session
	RefreshSession()

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

	fmt.Printf("✓ Default profile set to '%s'\n", name)
	return nil
}
