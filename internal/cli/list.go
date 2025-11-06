package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/vault-cli/vault/internal/config"
	"github.com/vault-cli/vault/internal/domain"
	"github.com/vault-cli/vault/internal/vault"
)

var (
	listTags   []string
	search     string
	outputJSON bool
	listLong   bool
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List entries in the vault",
	Long: `List all entries in the current profile, with optional filtering.

You can filter entries by tags or search for entries containing specific text
in their name, username, or URL. The --search flag supports fuzzy token
matching using '+' as an AND separator (e.g. 'aws+prod').

Example:
  vault list                           # List all entries
  vault list --tags work,git           # List entries with 'work' or 'git' tags
  vault list --search github          # List entries containing 'github'
  vault list --json                   # Output in JSON format
  vault list --long                   # Show detailed output with additional columns`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runList(cmd)
	},
}

func init() {
	listCmd.Flags().StringSliceVar(&listTags, "tags", nil, "Filter by tags")
	listCmd.Flags().StringVar(&search, "search", "", "Search in name, username, and URL")
	listCmd.Flags().BoolVar(&outputJSON, "json", false, "Output in JSON format")
	listCmd.Flags().BoolVar(&listLong, "long", false, "Show detailed output with additional columns")
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
  vault list --json                   # Output in JSON format
  vault list --long                   # Show detailed output with additional columns`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if cfg != nil && vaultPath == "" {
				vaultPath = cfg.VaultPath
			}
			if cfg != nil && profile == "" {
				profile = cfg.DefaultProfile
			}
			return runList(cmd)
		},
	}

	cmd.Flags().StringSliceVar(&listTags, "tags", nil, "Filter by tags")
	cmd.Flags().StringVar(&search, "search", "", "Search in name, username, and URL")
	cmd.Flags().BoolVar(&outputJSON, "json", false, "Output in JSON format")
	cmd.Flags().BoolVar(&listLong, "long", false, "Show detailed output with additional columns")

	return cmd
}

func runList(cmd *cobra.Command) error {
	defer CloseSessionStore()
	defer func() {
		listTags = nil
		search = ""
		outputJSON = false
		listLong = false
	}()

	out := cmd.OutOrStdout()

	// Check if vault is unlocked
	if !IsUnlocked() {
		return fmt.Errorf("vault is locked, run 'vault unlock' first")
	}

	vaultStore := GetVaultStore()

	// Create filter
	filter := &domain.Filter{
		Tags:         listTags,
		Search:       search,
		SearchTokens: vault.ParseSearchTokens(search),
	}

	// Get entries
	entries, err := vaultStore.ListEntries(profile, filter)
	if err != nil {
		return fmt.Errorf("failed to list entries: %w", err)
	}

	// Refresh session
	RefreshSession()

	if len(entries) == 0 {
		out := cmd.OutOrStdout()
		if filter.Search != "" || len(filter.Tags) > 0 {
			fmt.Fprintln(out, "No entries found matching the filter criteria")
		} else {
			fmt.Fprintf(out, "No entries found in profile '%s'\n", profile)
			fmt.Fprintln(out, "Use 'vault add <name>' to create your first entry")
		}
		return nil
	}

	// Sort entries by name
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})

	// Output based on format
	if outputJSON {
		return outputEntriesJSON(out, entries)
	}

	return outputEntriesTable(out, entries)
}

func outputEntriesTable(out io.Writer, entries []*domain.Entry) error {
	if listLong {
		return outputEntriesTableLong(out, entries)
	}

	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintf(w, "NAME\n")
	fmt.Fprintf(w, "----\n")

	for _, entry := range entries {
		fmt.Fprintf(w, "%s\n", entry.Name)
	}

	fmt.Fprintf(out, "\nFound %d entries in profile '%s'\n", len(entries), profile)
	return nil
}

func outputEntriesTableLong(out io.Writer, entries []*domain.Entry) error {
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintf(w, "NAME\tUSERNAME\tTAGS\tUPDATED_AT\n")
	fmt.Fprintf(w, "----\t--------\t----\t----------\n")

	for _, entry := range entries {
		tags := strings.Join(entry.Tags, ",")
		if len(tags) > 40 {
			tags = tags[:37] + "..."
		}

		username := entry.Username
		if len(username) > 24 {
			username = username[:21] + "..."
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			entry.Name,
			username,
			tags,
			entry.UpdatedAt.Format("2006-01-02 15:04"),
		)
	}

	fmt.Fprintf(out, "\nFound %d entries in profile '%s'\n", len(entries), profile)
	return nil
}

func outputEntriesJSON(out io.Writer, entries []*domain.Entry) error {
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

	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}
