package identity_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/vault-cli/vault/internal/identity"
	"github.com/vault-cli/vault/internal/store"
	"github.com/vault-cli/vault/internal/vault"
)

func TestIdentityRoundTripAcrossStore(t *testing.T) {
	tempDir := t.TempDir()
	vaultPath := filepath.Join(tempDir, "identity_integration.vault")

	salt, err := vault.GenerateSalt()
	if err != nil {
		t.Fatalf("failed to generate salt: %v", err)
	}

	crypto := vault.NewDefaultCryptoEngine()
	masterKey, err := crypto.DeriveKey("integration-passphrase", salt)
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

	issuer, err := identity.GenerateIdentity("issuer", time.Unix(1700000000, 0))
	if err != nil {
		t.Fatalf("failed to generate issuer: %v", err)
	}
	if err := vaultStore.CreateIdentity("default", issuer); err != nil {
		t.Fatalf("failed to create issuer: %v", err)
	}

	credential, err := identity.IssueCredential(
		issuer,
		"access-cred",
		issuer.DID,
		[]string{"AccessCredential"},
		[]identity.CredentialClaim{{Name: "scope", Value: "read"}},
		nil,
		time.Unix(1700000100, 0),
	)
	if err != nil {
		t.Fatalf("failed to issue credential: %v", err)
	}
	if err := vaultStore.CreateCredential("default", credential); err != nil {
		t.Fatalf("failed to store credential: %v", err)
	}

	fetchedIdentity, err := vaultStore.GetIdentity("default", "issuer")
	if err != nil {
		t.Fatalf("failed to fetch issuer: %v", err)
	}
	if fetchedIdentity.DID != issuer.DID {
		t.Fatalf("issuer DID mismatch")
	}

	fetchedCredential, err := vaultStore.GetCredential("default", "access-cred")
	if err != nil {
		t.Fatalf("failed to fetch credential: %v", err)
	}
	if err := identity.VerifyCredential(fetchedCredential); err != nil {
		t.Fatalf("stored credential should verify: %v", err)
	}
}
