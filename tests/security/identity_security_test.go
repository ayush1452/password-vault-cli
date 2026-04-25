package security_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vault-cli/vault/internal/identity"
	"github.com/vault-cli/vault/internal/store"
	"github.com/vault-cli/vault/internal/vault"
)

func TestKeyProofRejectsMismatchedChallenge(t *testing.T) {
	record, err := identity.GenerateIdentity("issuer", time.Unix(1700000000, 0))
	if err != nil {
		t.Fatalf("failed to generate identity: %v", err)
	}

	proof, err := identity.GenerateKeyPossessionProof(record, "expected-challenge", time.Unix(1700000100, 0))
	if err != nil {
		t.Fatalf("failed to generate proof: %v", err)
	}

	if err := identity.VerifyKeyPossessionProof(record.DID, proof, "other-challenge"); err == nil {
		t.Fatalf("expected mismatched challenge verification to fail")
	}
}

func TestPublicOnlySnapshotOmitsPrivateJWK(t *testing.T) {
	tempDir := t.TempDir()
	vaultPath := filepath.Join(tempDir, "security_identity.vault")
	exportPath := filepath.Join(tempDir, "public_snapshot.json")

	salt, err := vault.GenerateSalt()
	if err != nil {
		t.Fatalf("failed to generate salt: %v", err)
	}
	crypto := vault.NewDefaultCryptoEngine()
	masterKey, err := crypto.DeriveKey("security-passphrase", salt)
	if err != nil {
		t.Fatalf("failed to derive key: %v", err)
	}
	defer vault.Zeroize(masterKey)

	params := vault.DefaultArgon2Params()
	kdfParams := map[string]interface{}{
		"memory":      params.Memory,
		"iterations":  params.Iterations,
		"parallelism": params.Parallelism,
		"salt":        salt,
	}

	vaultStore := store.NewBoltStore()
	if err := vaultStore.CreateVault(vaultPath, masterKey, kdfParams); err != nil {
		t.Fatalf("failed to create vault: %v", err)
	}
	if err := vaultStore.OpenVault(vaultPath, masterKey); err != nil {
		t.Fatalf("failed to open vault: %v", err)
	}
	defer vaultStore.CloseVault()

	record, err := identity.GenerateIdentity("issuer", time.Unix(1700000000, 0))
	if err != nil {
		t.Fatalf("failed to generate identity: %v", err)
	}
	if err := vaultStore.CreateIdentity("default", record); err != nil {
		t.Fatalf("failed to store identity: %v", err)
	}
	if err := vaultStore.ExportVault(exportPath, false); err != nil {
		t.Fatalf("failed to export public-only snapshot: %v", err)
	}

	data, err := os.ReadFile(exportPath)
	if err != nil {
		t.Fatalf("failed to read snapshot: %v", err)
	}
	if strings.Contains(string(data), "\"private_jwk\"") {
		t.Fatalf("public-only snapshot should omit private_jwk")
	}
}
