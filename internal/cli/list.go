package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/vault-cli/vault/internal/config"
	"github.com/vault-cli/vault/internal/domain"
)

var (
	listTags   []string
	search     string
	outputJSON bool
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List entries in the vault",
	Long: `List all entries in the current profile, with optional filtering.

You can filter entries by tags or search for entries containing specific text
in their name, username, or URL.

Example:
  vault list                           # List all entries
  vault list --tags work,git           # List entries with 'work' or 'git' tags
  vault list --search github          # List entries containing 'github'
  vault list --json                   # Output in JSON format`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runList()
	},
}

func init() {
	listCmd.Flags().StringSliceVar(&listTags, "tags", nil, "Filter by tags")
	listCmd.Flags().StringVar(&search, "search", "", "Search in name, username, and URL")
	listCmd.Flags().BoolVar(&outputJSON, "json", false, "Output in JSON format")
}

// NewListCommand creates a new list command for testing
func NewListCommand(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List entries in the vault",
		Long: `List all entries in the current profile, with optional filtering.

You can filter entries by tags or search for entries containing specific text
in their name, username, or URL.

Example:
  vault list                           # List all entries
  vault list --tags work,git           # List entries with 'work' or 'git' tags
  vault list --search github          # List entries containing 'github'
  vault list --json                   # Output in JSON format`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if cfg != nil && vaultPath == "" {
				vaultPath = cfg.VaultPath
			}
			if cfg != nil && profile == "" {
				profile = cfg.DefaultProfile
			}
			return runList()
		},
	}

	cmd.Flags().StringSliceVar(&listTags, "tags", nil, "Filter by tags")
	cmd.Flags().StringVar(&search, "search", "", "Search in name, username, and URL")
	cmd.Flags().BoolVar(&outputJSON, "json", false, "Output in JSON format")

	return cmd
}

func runList() error {
	defer CloseSessionStore()

	// Check if vault is unlocked
	if !IsUnlocked() {
		return fmt.Errorf("vault is locked, run 'vault unlock' first")
	}

	vaultStore := GetVaultStore()

	// Create filter
	filter := &domain.Filter{
		Tags:   listTags,
		Search: search,
	}

	// Get entries
	entries, err := vaultStore.ListEntries(profile, filter)
	if err != nil {
		return fmt.Errorf("failed to list entries: %w", err)
	}

	// Refresh session
	RefreshSession()

	if len(entries) == 0 {
		if filter.Search != "" || len(filter.Tags) > 0 {
			fmt.Println("No entries found matching the filter criteria")
		} else {
			fmt.Printf("No entries found in profile '%s'\n", profile)
			fmt.Println("Use 'vault add <name>' to create your first entry")
		}
		return nil
	}

	// Sort entries by name
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})

	// Output based on format
	if outputJSON {
		return outputEntriesJSON(entries)
	}

	return outputEntriesTable(entries)
}

func outputEntriesTable(entries []*domain.Entry) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	// Header
	fmt.Fprintf(w, "NAME\tUSERNAME\tURL\tTAGS\tUPDATED\n")
	fmt.Fprintf(w, "----\t--------\t---\t----\t-------\n")

	// Entries
	for _, entry := range entries {
		tags := strings.Join(entry.Tags, ",")
		if len(tags) > 30 {
			tags = tags[:27] + "..."
		}

		url := entry.URL
		if len(url) > 40 {
			url = url[:37] + "..."
		}

		username := entry.Username
		if len(username) > 20 {
			username = username[:17] + "..."
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			entry.Name,
			username,
			url,
			tags,
			entry.UpdatedAt.Format("2006-01-02"),
		)
	}

	fmt.Printf("\nFound %d entries in profile '%s'\n", len(entries), profile)
	return nil
}

func outputEntriesJSON(entries []*domain.Entry) error {
	// Create output structure without secrets
	type EntryOutput struct {
		Name      string   `json:"name"`
		Username  string   `json:"username"`
		URL       string   `json:"url"`
		Notes     string   `json:"notes"`
		Tags      []string `json:"tags"`
		CreatedAt string   `json:"created_at"`
		UpdatedAt string   `json:"updated_at"`
	}

	var output []EntryOutput
	for _, entry := range entries {
		output = append(output, EntryOutput{
			Name:      entry.Name,
			Username:  entry.Username,
			URL:       entry.URL,
			Notes:     entry.Notes,
			Tags:      entry.Tags,
			CreatedAt: entry.CreatedAt.Format("2006-01-02T15:04:05Z"),
			UpdatedAt: entry.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}
