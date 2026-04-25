package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/vault-cli/vault/internal/config"
	"github.com/vault-cli/vault/internal/identity"
)

// NewZKProof creates the key-possession proof command.
func NewZKProof(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "zk-proof",
		Short: "Generate a zero-knowledge key-possession proof",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			applyConfigDefaults(cfg)
			defer func() {
				err = checkDeferredErr(err, "CloseSessionStore", CloseSessionStore())
			}()

			if !IsUnlocked() {
				return fmt.Errorf("vault is locked, run 'vault unlock' first")
			}

			didName, _ := cmd.Flags().GetString("did")
			challenge, _ := cmd.Flags().GetString("challenge")
			outputPath, _ := cmd.Flags().GetString("output")
			if strings.TrimSpace(didName) == "" {
				return fmt.Errorf("did is required")
			}
			if strings.TrimSpace(challenge) == "" {
				return fmt.Errorf("challenge is required")
			}

			vaultStore := GetVaultStore()
			record, err := vaultStore.GetIdentity(activeProfile(), didName)
			if err != nil {
				return fmt.Errorf("failed to get DID identity: %w", err)
			}
			if record.PrivateJWK.D == "" {
				return fmt.Errorf("identity '%s' does not have private key material", didName)
			}

			proof, err := identity.GenerateKeyPossessionProof(record, challenge, time.Now().UTC())
			if err != nil {
				return err
			}
			payload, err := json.Marshal(proof)
			if err != nil {
				return fmt.Errorf("marshal key proof: %w", err)
			}
			if outputPath != "" {
				if err := writeArtifactFile(outputPath, payload); err != nil {
					return err
				}
			}

			RefreshSession()
			maybeLogIdentityOperation(vaultStore, "zk_proof_generate", record.Name)
			return writePrettyJSON(cmd.OutOrStdout(), payload)
		},
	}
	cmd.Flags().String("did", "", "Local DID name to use for proof generation")
	cmd.Flags().String("challenge", "", "Verifier-supplied challenge")
	cmd.Flags().String("output", "", "Write the generated proof JSON to a file")
	cmd.AddCommand(newZKProofVerifyCmd(cfg))
	return cmd
}

func newZKProofVerifyCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify a zero-knowledge proof against a DID and challenge",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			applyConfigDefaults(cfg)
			defer func() {
				err = checkDeferredErr(err, "CloseSessionStore", CloseSessionStore())
			}()

			didRef, _ := cmd.Flags().GetString("did")
			challenge, _ := cmd.Flags().GetString("challenge")
			proofPath, _ := cmd.Flags().GetString("proof")
			if strings.TrimSpace(didRef) == "" {
				return fmt.Errorf("did is required")
			}
			if strings.TrimSpace(challenge) == "" {
				return fmt.Errorf("challenge is required")
			}
			if strings.TrimSpace(proofPath) == "" {
				return fmt.Errorf("proof is required")
			}

			did, err := resolveDIDReference(didRef)
			if err != nil {
				return err
			}

			var data []byte
			if proofPath == "-" {
				data, err = io.ReadAll(os.Stdin)
			} else {
				data, err = os.ReadFile(proofPath)
			}
			if err != nil {
				return fmt.Errorf("failed to read proof input: %w", err)
			}

			var proof identity.KeyPossessionProof
			if err := json.Unmarshal(data, &proof); err != nil {
				return fmt.Errorf("failed to parse proof JSON: %w", err)
			}
			if err := identity.VerifyKeyPossessionProof(did, &proof, challenge); err != nil {
				return err
			}

			if IsUnlocked() {
				maybeLogIdentityOperation(GetVaultStore(), "zk_proof_verify", did)
			}
			return writeOutput(cmd.OutOrStdout(), "✓ Zero-knowledge proof verified successfully for %s\n", did)
		},
	}
	cmd.Flags().String("did", "", "Raw did:jwk, local DID name, or path to an exported DID document")
	cmd.Flags().String("challenge", "", "Verifier-supplied challenge")
	cmd.Flags().String("proof", "", "Path to a proof JSON file or '-' for stdin")
	return cmd
}
