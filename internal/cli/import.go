package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var importConflict string

var importCmd = &cobra.Command{
	Use:   "import <file>",
	Short: "Import vault data",
	Long: `Import vault data from a backup file.

You can import from encrypted vault exports or plaintext JSON files.
Conflict resolution determines what happens when entries already exist.

Example:
  vault import backup.vault
  vault import backup.json --conflict overwrite
  vault import backup.vault --conflict skip`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runImport(args[0])
	},
}

func init() {
	importCmd.Flags().StringVar(&importConflict, "conflict", "skip", "Conflict resolution (skip|overwrite|duplicate)")
}

func runImport(filePath string) error {
	// Check if vault is unlocked
	if !IsUnlocked() {
		return fmt.Errorf("vault is locked, run 'vault unlock' first")
	}

	// TODO: Implement import functionality
	return fmt.Errorf("import functionality is not yet implemented")
}
