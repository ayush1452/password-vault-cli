package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

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

	// Helper function to write output with error checking
	writeOutput := func(w io.Writer, format string, args ...interface{}) error {
		_, err := fmt.Fprintf(w, format, args...)
		if err != nil {
			return fmt.Errorf("failed to write output: %w", err)
		}
		return nil
	}

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
			if err := writeOutput(out, "No entries found matching the filter criteria\n"); err != nil {
				return err
			}
		} else {
			if err := writeOutput(out, "No entries found in profile '%s'\n", profile); err != nil {
				return err
			}
			if err := writeOutput(out, "Use 'vault add <name>' to create your first entry\n"); err != nil {
				return err
			}
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

	// Helper function to write to tabwriter with error checking
	writeOutput := func(format string, args ...interface{}) error {
		_, err := fmt.Fprintf(w, format, args...)
		if err != nil {
			return fmt.Errorf("failed to write output: %w", err)
		}
		return nil
	}

	// Write table header
	if err := writeOutput("NAME\n"); err != nil {
		return err
	}
	if err := writeOutput("----\n"); err != nil {
		return err
	}

	// Write table rows
	for _, entry := range entries {
		if err := writeOutput("%s\n", entry.Name); err != nil {
			return err
		}
	}

	// Write summary
	if err := writeOutput("\nFound %d entries in profile '%s'\n", len(entries), profile); err != nil {
		return err
	}

	return nil
}

func outputEntriesTableLong(out io.Writer, entries []*domain.Entry) error {
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	defer w.Flush()

	// Helper function to write to tabwriter with error checking
	writeOutput := func(format string, args ...interface{}) error {
		_, err := fmt.Fprintf(w, format, args...)
		if err != nil {
			return fmt.Errorf("failed to write output: %w", err)
		}
		return nil
	}

	// Write table header
	headers := []string{"NAME", "USERNAME", "TAGS", "UPDATED_AT"}
	headerLine := strings.Join(headers, "\t") + "\n"
	separator := strings.Repeat("-", 4) + "\t" +
		strings.Repeat("-", 8) + "\t" +
		strings.Repeat("-", 4) + "\t" +
		strings.Repeat("-", 10) + "\n"

	if err := writeOutput(headerLine); err != nil {
		return err
	}
	if err := writeOutput(separator); err != nil {
		return err
	}

	// Write table rows
	for _, entry := range entries {
		tags := strings.Join(entry.Tags, ",")
		if len(tags) > 40 {
			tags = tags[:37] + "..."
		}

		username := entry.Username
		if len(username) > 24 {
			username = username[:21] + "..."
		}

		updatedAt := entry.UpdatedAt.Format("2006-01-02 15:04")
		if err := writeOutput("%s\t%s\t%s\t%s\n",
			entry.Name,
			username,
			tags,
			updatedAt,
		); err != nil {
			return err
		}
	}

	// Write summary
	if err := writeOutput("\nFound %d entries in profile '%s'\n", len(entries), profile); err != nil {
		return err
	}

	return nil
}

func outputEntriesJSON(out io.Writer, entries []*domain.Entry) error {
	// Create output structure without secrets
	type EntryOutput struct {
		Name      string   `json:"name"`
		Username  string   `json:"username,omitempty"`
		URL       string   `json:"url,omitempty"`
		Tags      []string `json:"tags,omitempty"`
		Notes     string   `json:"notes,omitempty"`
		CreatedAt string   `json:"createdAt"`
		UpdatedAt string   `json:"updatedAt"`
	}

	var output []EntryOutput

	for _, entry := range entries {
		output = append(output, EntryOutput{
			Name:      entry.Name,
			Username:  entry.Username,
			URL:       entry.URL,
			Tags:      entry.Tags,
			Notes:     entry.Notes,
			CreatedAt: entry.CreatedAt.Format(time.RFC3339),
			UpdatedAt: entry.UpdatedAt.Format(time.RFC3339),
		})
	}

	// Encode to JSON with indentation
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(output); err != nil {
		return fmt.Errorf("failed to encode entries to JSON: %w", err)
	}

	return nil
}
