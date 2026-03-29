package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/vault-cli/vault/internal/config"
	"github.com/vault-cli/vault/internal/domain"
	"github.com/vault-cli/vault/internal/identity"
	"github.com/vault-cli/vault/internal/store"
)

func applyConfigDefaults(cfg *config.Config) {
	if cfg == nil {
		return
	}
	if vaultPath == "" {
		vaultPath = cfg.VaultPath
	}
	if profile == "" && cfg.DefaultProfile != "" {
		profile = cfg.DefaultProfile
	}
}

func activeProfile() string {
	if profile == "" {
		return "default"
	}
	return profile
}

func parseCredentialClaims(values []string) ([]identity.CredentialClaim, error) {
	claims := make([]identity.CredentialClaim, 0, len(values))
	for _, raw := range values {
		parts := strings.SplitN(raw, "=", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" {
			return nil, fmt.Errorf("invalid claim %q, expected key=value", raw)
		}
		claims = append(claims, identity.CredentialClaim{
			Name:  parts[0],
			Value: parts[1],
		})
	}
	return identity.NormalizeClaims(claims)
}

func writeArtifactFile(path string, data []byte) error {
	cleanPath := filepath.Clean(path)
	if err := os.MkdirAll(filepath.Dir(cleanPath), 0o700); err != nil {
		return fmt.Errorf("create artifact directory: %w", err)
	}
	if err := os.WriteFile(cleanPath, data, 0o600); err != nil {
		return fmt.Errorf("write artifact file: %w", err)
	}
	return nil
}

func writePrettyJSON(out io.Writer, data []byte) error {
	var buf bytes.Buffer
	if err := json.Indent(&buf, data, "", "  "); err != nil {
		if _, err := out.Write(data); err != nil {
			return fmt.Errorf("write JSON output: %w", err)
		}
		_, err := fmt.Fprintln(out)
		return err
	}
	if _, err := buf.WriteTo(out); err != nil {
		return fmt.Errorf("write pretty JSON: %w", err)
	}
	_, err := fmt.Fprintln(out)
	return err
}

func sortIdentities(records []*identity.IdentityRecord) {
	sort.Slice(records, func(i, j int) bool {
		return strings.ToLower(records[i].Name) < strings.ToLower(records[j].Name)
	})
}

func sortCredentials(records []*identity.CredentialRecord) {
	sort.Slice(records, func(i, j int) bool {
		return strings.ToLower(records[i].ID) < strings.ToLower(records[j].ID)
	})
}

func resolveSubjectReference(vaultStore store.VaultStore, subject string) (string, error) {
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return "", fmt.Errorf("subject cannot be empty")
	}
	if vaultStore != nil && vaultStore.IdentityExists(activeProfile(), subject) {
		record, err := vaultStore.GetIdentity(activeProfile(), subject)
		if err != nil {
			return "", err
		}
		return record.DID, nil
	}
	if strings.HasPrefix(subject, "did:") {
		return subject, nil
	}
	return "", fmt.Errorf("subject must be a local DID name or raw DID")
}

func resolveDIDReference(ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", fmt.Errorf("DID reference cannot be empty")
	}
	if IsUnlocked() {
		vaultStore := GetVaultStore()
		if vaultStore != nil && vaultStore.IdentityExists(activeProfile(), ref) {
			record, err := vaultStore.GetIdentity(activeProfile(), ref)
			if err != nil {
				return "", err
			}
			return record.DID, nil
		}
	}
	if strings.HasPrefix(ref, "did:") {
		return ref, nil
	}
	data, err := os.ReadFile(filepath.Clean(ref))
	if err != nil {
		return "", fmt.Errorf("resolve DID reference: %w", err)
	}
	var document identity.DIDDocument
	if err := json.Unmarshal(data, &document); err != nil {
		return "", fmt.Errorf("parse DID document: %w", err)
	}
	if !strings.HasPrefix(document.ID, "did:") {
		return "", fmt.Errorf("DID document is missing a valid DID")
	}
	return document.ID, nil
}

func maybeLogIdentityOperation(vaultStore store.VaultStore, operationType, resourceID string) {
	if vaultStore == nil {
		return
	}
	if err := vaultStore.LogOperation(&domain.Operation{
		Type:      operationType,
		Profile:   activeProfile(),
		EntryID:   resourceID,
		Timestamp: time.Now().UTC(),
		Success:   true,
	}); err != nil {
		logWarning("Failed to log %s operation: %v", operationType, err)
	}
}
