package cli

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/vault-cli/vault/internal/config"
	"github.com/vault-cli/vault/internal/identity"
)

// NewDID creates the DID command group.
func NewDID(cfg *config.Config) *cobra.Command {
	didCmd := &cobra.Command{
		Use:   "did",
		Short: "Manage decentralized identifiers",
	}

	didCmd.AddCommand(newDIDCreateCmd(cfg))
	didCmd.AddCommand(newDIDListCmd(cfg))
	didCmd.AddCommand(newDIDShowCmd(cfg))
	return didCmd
}

func newDIDCreateCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a local did:jwk identity",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			applyConfigDefaults(cfg)
			defer func() {
				err = checkDeferredErr(err, "CloseSessionStore", CloseSessionStore())
			}()

			if !IsUnlocked() {
				return fmt.Errorf("vault is locked, run 'vault unlock' first")
			}

			outputJSON, _ := cmd.Flags().GetBool("json")
			exportPath, _ := cmd.Flags().GetString("export")
			vaultStore := GetVaultStore()

			if vaultStore.IdentityExists(activeProfile(), args[0]) {
				return fmt.Errorf("identity '%s' already exists in profile '%s'", args[0], activeProfile())
			}

			record, err := identity.GenerateIdentity(args[0], time.Now().UTC())
			if err != nil {
				return err
			}
			if err := vaultStore.CreateIdentity(activeProfile(), record); err != nil {
				return fmt.Errorf("failed to store identity: %w", err)
			}
			RefreshSession()
			maybeLogIdentityOperation(vaultStore, "did_create", record.Name)

			documentJSON, err := record.PublicDocumentJSON()
			if err != nil {
				return err
			}
			if exportPath != "" {
				if err := writeArtifactFile(exportPath, documentJSON); err != nil {
					return err
				}
			}

			if outputJSON {
				return writePrettyJSON(cmd.OutOrStdout(), documentJSON)
			}

			if err := writeOutput(cmd.OutOrStdout(), "✓ DID '%s' created in profile '%s'\n", record.Name, activeProfile()); err != nil {
				return err
			}
			if err := writeOutput(cmd.OutOrStdout(), "DID: %s\n", record.DID); err != nil {
				return err
			}
			if exportPath != "" {
				return writeOutput(cmd.OutOrStdout(), "Public DID document exported to %s\n", exportPath)
			}
			return nil
		},
	}
	cmd.Flags().Bool("json", false, "Output the public DID document as JSON")
	cmd.Flags().String("export", "", "Write the public DID document to a file")
	return cmd
}

func newDIDListCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List local DIDs in the active profile",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			applyConfigDefaults(cfg)
			defer func() {
				err = checkDeferredErr(err, "CloseSessionStore", CloseSessionStore())
			}()

			if !IsUnlocked() {
				return fmt.Errorf("vault is locked, run 'vault unlock' first")
			}

			outputJSON, _ := cmd.Flags().GetBool("json")
			vaultStore := GetVaultStore()
			records, err := vaultStore.ListIdentities(activeProfile())
			if err != nil {
				return fmt.Errorf("failed to list identities: %w", err)
			}
			sortIdentities(records)
			RefreshSession()

			if outputJSON {
				publicRecords := make([]*identity.IdentityRecord, 0, len(records))
				for _, record := range records {
					publicRecords = append(publicRecords, record.PublicCopy())
				}
				data, err := json.Marshal(publicRecords)
				if err != nil {
					return err
				}
				return writePrettyJSON(cmd.OutOrStdout(), data)
			}

			if len(records) == 0 {
				return writeOutput(cmd.OutOrStdout(), "No DIDs found in profile '%s'\n", activeProfile())
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			defer func() {
				if flushErr := w.Flush(); flushErr != nil {
					logWarning("Failed to flush DID list output: %v", flushErr)
				}
			}()
			if err := writeOutput(w, "NAME\tDID\tCREATED\n"); err != nil {
				return err
			}
			if err := writeOutput(w, "----\t---\t-------\n"); err != nil {
				return err
			}
			for _, record := range records {
				if err := writeOutput(w, "%s\t%s\t%s\n", record.Name, record.DID, record.CreatedAt.Format(time.RFC3339)); err != nil {
					return err
				}
			}
			return writeOutput(cmd.OutOrStdout(), "\nFound %d DID(s) in profile '%s'\n", len(records), activeProfile())
		},
	}
	cmd.Flags().Bool("json", false, "Output identities as JSON")
	return cmd
}

func newDIDShowCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <name>",
		Short: "Show the public DID document for a local identity",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			applyConfigDefaults(cfg)
			defer func() {
				err = checkDeferredErr(err, "CloseSessionStore", CloseSessionStore())
			}()

			if !IsUnlocked() {
				return fmt.Errorf("vault is locked, run 'vault unlock' first")
			}

			outputJSON, _ := cmd.Flags().GetBool("json")
			vaultStore := GetVaultStore()
			record, err := vaultStore.GetIdentity(activeProfile(), args[0])
			if err != nil {
				return fmt.Errorf("failed to get identity: %w", err)
			}
			RefreshSession()

			documentJSON, err := record.PublicDocumentJSON()
			if err != nil {
				return err
			}
			if outputJSON {
				return writePrettyJSON(cmd.OutOrStdout(), documentJSON)
			}

			if err := writeOutput(cmd.OutOrStdout(), "Name: %s\n", record.Name); err != nil {
				return err
			}
			if err := writeOutput(cmd.OutOrStdout(), "DID: %s\n", record.DID); err != nil {
				return err
			}
			return writeOutput(cmd.OutOrStdout(), "Verification Method: %s\n", record.VerificationMethodID)
		},
	}
	cmd.Flags().Bool("json", false, "Output the public DID document as JSON")
	return cmd
}
