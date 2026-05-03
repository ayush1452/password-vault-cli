package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/vault-cli/vault/internal/config"
	"github.com/vault-cli/vault/internal/identity"
)

// NewVC creates the verifiable credential command group.
func NewVC(cfg *config.Config) *cobra.Command {
	vcCmd := &cobra.Command{
		Use:   "vc",
		Short: "Issue and inspect verifiable credentials",
	}

	vcCmd.AddCommand(newVCIssueCmd(cfg))
	vcCmd.AddCommand(newVCListCmd(cfg))
	vcCmd.AddCommand(newVCShowCmd(cfg))
	vcCmd.AddCommand(newVCVerifyCmd(cfg))
	return vcCmd
}

func newVCIssueCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "issue <credential-id>",
		Short: "Issue a signed verifiable credential",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			applyConfigDefaults(cfg)
			defer func() {
				err = checkDeferredErr(err, "CloseSessionStore", CloseSessionStore())
			}()

			if !IsUnlocked() {
				return fmt.Errorf("vault is locked, run 'vault unlock' first")
			}

			issuerName, _ := cmd.Flags().GetString("issuer")
			subjectRef, _ := cmd.Flags().GetString("subject")
			typeFlags, _ := cmd.Flags().GetStringSlice("type")
			claimFlags, _ := cmd.Flags().GetStringSlice("claim")
			expiresRaw, _ := cmd.Flags().GetString("expires")
			outputJSON, _ := cmd.Flags().GetBool("json")
			exportPath, _ := cmd.Flags().GetString("export")

			if strings.TrimSpace(issuerName) == "" {
				return fmt.Errorf("issuer is required")
			}
			if strings.TrimSpace(subjectRef) == "" {
				return fmt.Errorf("subject is required")
			}

			vaultStore := GetVaultStore()
			issuerRecord, err := vaultStore.GetIdentity(activeProfile(), issuerName)
			if err != nil {
				return fmt.Errorf("failed to get issuer identity: %w", err)
			}
			if issuerRecord.PrivateJWK.D == "" {
				return fmt.Errorf("issuer '%s' does not have private key material", issuerName)
			}

			subject, err := resolveSubjectReference(vaultStore, subjectRef)
			if err != nil {
				return err
			}
			claims, err := parseCredentialClaims(claimFlags)
			if err != nil {
				return err
			}

			var expiresAt *time.Time
			if strings.TrimSpace(expiresRaw) != "" {
				parsed, err := time.Parse(time.RFC3339, expiresRaw)
				if err != nil {
					return fmt.Errorf("invalid expiration time: %w", err)
				}
				expiresAt = &parsed
			}

			record, err := identity.IssueCredential(issuerRecord, args[0], subject, typeFlags, claims, expiresAt, time.Now().UTC())
			if err != nil {
				return err
			}
			if err := vaultStore.CreateCredential(activeProfile(), record); err != nil {
				return fmt.Errorf("failed to store credential: %w", err)
			}
			RefreshSession()
			maybeLogIdentityOperation(vaultStore, "vc_issue", record.ID)

			payload, err := identity.MarshalCredentialJSON(record)
			if err != nil {
				return err
			}
			if exportPath != "" {
				if err := writeArtifactFile(exportPath, payload); err != nil {
					return err
				}
			}
			if outputJSON {
				return writePrettyJSON(cmd.OutOrStdout(), payload)
			}

			if err := writeOutput(cmd.OutOrStdout(), "✓ Credential '%s' issued in profile '%s'\n", record.ID, activeProfile()); err != nil {
				return err
			}
			if err := writeOutput(cmd.OutOrStdout(), "Issuer: %s\n", record.IssuerDID); err != nil {
				return err
			}
			if err := writeOutput(cmd.OutOrStdout(), "Subject: %s\n", record.Subject); err != nil {
				return err
			}
			if exportPath != "" {
				return writeOutput(cmd.OutOrStdout(), "Credential exported to %s\n", exportPath)
			}
			return nil
		},
	}
	cmd.Flags().String("issuer", "", "Local DID name to use as the issuer")
	cmd.Flags().String("subject", "", "Local DID name or raw DID for the credential subject")
	cmd.Flags().StringSlice("type", nil, "Additional VC type values")
	cmd.Flags().StringSlice("claim", nil, "Flat credential claim in key=value form")
	cmd.Flags().String("expires", "", "Optional RFC3339 expiration time")
	cmd.Flags().Bool("json", false, "Output the signed credential as JSON")
	cmd.Flags().String("export", "", "Write the signed credential to a file")
	return cmd
}

func newVCListCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List credentials in the active profile",
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
			records, err := vaultStore.ListCredentials(activeProfile())
			if err != nil {
				return fmt.Errorf("failed to list credentials: %w", err)
			}
			sortCredentials(records)
			RefreshSession()

			if outputJSON {
				payload, err := json.Marshal(records)
				if err != nil {
					return err
				}
				return writePrettyJSON(cmd.OutOrStdout(), payload)
			}

			if len(records) == 0 {
				return writeOutput(cmd.OutOrStdout(), "No credentials found in profile '%s'\n", activeProfile())
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			defer func() {
				if flushErr := w.Flush(); flushErr != nil {
					logWarning("Failed to flush VC list output: %v", flushErr)
				}
			}()
			if err := writeOutput(w, "ID\tISSUER\tSUBJECT\tISSUED_AT\n"); err != nil {
				return err
			}
			if err := writeOutput(w, "--\t------\t-------\t---------\n"); err != nil {
				return err
			}
			for _, record := range records {
				if err := writeOutput(w, "%s\t%s\t%s\t%s\n", record.ID, record.IssuerDID, record.Subject, record.IssuedAt.Format(time.RFC3339)); err != nil {
					return err
				}
			}
			return writeOutput(cmd.OutOrStdout(), "\nFound %d credential(s) in profile '%s'\n", len(records), activeProfile())
		},
	}
	cmd.Flags().Bool("json", false, "Output credentials as JSON")
	return cmd
}

func newVCShowCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <credential-id>",
		Short: "Show a stored credential",
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
			record, err := vaultStore.GetCredential(activeProfile(), args[0])
			if err != nil {
				return fmt.Errorf("failed to get credential: %w", err)
			}
			RefreshSession()

			if outputJSON {
				payload, err := identity.MarshalCredentialJSON(record)
				if err != nil {
					return err
				}
				return writePrettyJSON(cmd.OutOrStdout(), payload)
			}

			if err := writeOutput(cmd.OutOrStdout(), "ID: %s\n", record.ID); err != nil {
				return err
			}
			if err := writeOutput(cmd.OutOrStdout(), "Issuer: %s\n", record.IssuerDID); err != nil {
				return err
			}
			if err := writeOutput(cmd.OutOrStdout(), "Subject: %s\n", record.Subject); err != nil {
				return err
			}
			if err := writeOutput(cmd.OutOrStdout(), "Types: %s\n", strings.Join(record.Types, ", ")); err != nil {
				return err
			}
			for _, claim := range record.Claims {
				if err := writeOutput(cmd.OutOrStdout(), "Claim %s=%s\n", claim.Name, claim.Value); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().Bool("json", false, "Output the full signed credential as JSON")
	return cmd
}

func newVCVerifyCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify [credential-id]",
		Short: "Verify a stored or exported credential",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			applyConfigDefaults(cfg)
			defer func() {
				err = checkDeferredErr(err, "CloseSessionStore", CloseSessionStore())
			}()

			filePath, _ := cmd.Flags().GetString("file")
			var record *identity.CredentialRecord

			if strings.TrimSpace(filePath) != "" {
				data, err := os.ReadFile(filePath)
				if err != nil {
					return fmt.Errorf("failed to read credential file: %w", err)
				}
				record, err = identity.ParseCredentialJSON(data)
				if err != nil {
					return err
				}
			} else {
				if len(args) != 1 {
					return fmt.Errorf("credential ID is required unless --file is used")
				}
				if !IsUnlocked() {
					return fmt.Errorf("vault is locked, run 'vault unlock' first")
				}
				vaultStore := GetVaultStore()
				record, err = vaultStore.GetCredential(activeProfile(), args[0])
				if err != nil {
					return fmt.Errorf("failed to get credential: %w", err)
				}
				RefreshSession()
				maybeLogIdentityOperation(vaultStore, "vc_verify", record.ID)
			}

			if err := identity.VerifyCredential(record); err != nil {
				return err
			}

			return writeOutput(cmd.OutOrStdout(), "✓ Credential '%s' verified successfully\n", record.ID)
		},
	}
	cmd.Flags().String("file", "", "Path to a signed credential JSON file")
	return cmd
}
