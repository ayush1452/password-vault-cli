package store_test

import (
	"encoding/base64"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vault-cli/vault/internal/domain"
	"github.com/vault-cli/vault/internal/store"
	"github.com/vault-cli/vault/internal/vault"
)

// TestBoltStoreRotateMasterKey verifies the master key rotation functionality of the BoltStore.
// It tests the following scenarios:
// 1. Creating a new vault with an initial master key
// 2. Adding test data to the vault
// 3. Rotating the master key to a new passphrase
// 4. Verifying the data can still be accessed with the new key
func TestBoltStoreRotateMasterKey(t *testing.T) {
	tempDir := t.TempDir()
	vaultPath := filepath.Join(tempDir, "vault.db")

	params := vault.DefaultArgon2Params()
	cryptoEngine := vault.NewCryptoEngine(params)

	salt, err := vault.GenerateSalt()
	if err != nil {
		t.Fatalf("failed to generate salt: %v", err)
	}

	masterKey, err := cryptoEngine.DeriveKey("old-pass", salt)
	if err != nil {
		t.Fatalf("failed to derive initial key: %v", err)
	}
	defer vault.Zeroize(masterKey)

	oldKeyCopy := make([]byte, len(masterKey))
	copy(oldKeyCopy, masterKey)
	defer vault.Zeroize(oldKeyCopy)

	kdfParamsMap := map[string]interface{}{
		"memory":      params.Memory,
		"iterations":  params.Iterations,
		"parallelism": params.Parallelism,
		"salt":        salt,
	}

	creator := store.NewBoltStore()
	if err := creator.CreateVault(vaultPath, masterKey, kdfParamsMap); err != nil {
		t.Fatalf("failed to create vault: %v", err)
	}

	bs := store.NewBoltStore()
	if err := bs.OpenVault(vaultPath, masterKey); err != nil {
		t.Fatalf("failed to open vault: %v", err)
	}

	initialMetadata, err := bs.GetVaultMetadata()
	if err != nil {
		t.Fatalf("failed to get metadata: %v", err)
	}

	initialSaltB64, ok := initialMetadata.KDFParams["salt"].(string)
	if !ok {
		t.Fatalf("metadata salt was not a base64 string")
	}

	entry := &domain.Entry{
		ID:        "example",
		Name:      "example",
		Username:  "user",
		Password:  []byte("initial-secret"),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	if err := bs.CreateEntry("default", entry); err != nil {
		t.Fatalf("failed to create entry: %v", err)
	}

	if err := bs.RotateMasterKey("new-pass"); err != nil {
		t.Fatalf("rotate master key failed: %v", err)
	}

	rotatedEntry, err := bs.GetEntry("default", entry.ID)
	if err != nil {
		t.Fatalf("failed to fetch entry after rotation: %v", err)
	}
	if string(rotatedEntry.Password) != "initial-secret" {
		t.Fatalf("password mismatch after rotation, got %s", string(rotatedEntry.Password))
	}

	rotatedMetadata, err := bs.GetVaultMetadata()
	if err != nil {
		t.Fatalf("failed to get rotated metadata: %v", err)
	}

	rotatedSaltB64, ok := rotatedMetadata.KDFParams["salt"].(string)
	if !ok {
		t.Fatalf("rotated salt not a base64 string")
	}
	if rotatedSaltB64 == initialSaltB64 {
		t.Fatalf("salt was not updated during rotation")
	}

	rotatedSalt, err := base64.StdEncoding.DecodeString(rotatedSaltB64)
	if err != nil {
		t.Fatalf("failed to decode rotated salt: %v", err)
	}

	rotatedParams := vault.Argon2Params{
		Memory:      uint32(rotatedMetadata.KDFParams["memory"].(float64)),
		Iterations:  uint32(rotatedMetadata.KDFParams["iterations"].(float64)),
		Parallelism: uint8(rotatedMetadata.KDFParams["parallelism"].(float64)),
	}

	rotatedCrypto := vault.NewCryptoEngine(rotatedParams)
	rotatedMasterKey, err := rotatedCrypto.DeriveKey("new-pass", rotatedSalt)
	if err != nil {
		t.Fatalf("failed to derive rotated master key: %v", err)
	}
	defer vault.Zeroize(rotatedMasterKey)

	if err := bs.CloseVault(); err != nil {
		t.Fatalf("failed to close vault after rotation: %v", err)
	}

	bsNew := store.NewBoltStore()
	if err := bsNew.OpenVault(vaultPath, rotatedMasterKey); err != nil {
		t.Fatalf("failed to open with rotated key: %v", err)
	}
	defer func() {
		if err := bsNew.CloseVault(); err != nil {
			t.Logf("Warning: failed to close new vault: %v", err)
		}
	}()

	fetched, err := bsNew.GetEntry("default", entry.ID)
	if err != nil {
		t.Fatalf("failed to fetch entry with rotated key: %v", err)
	}
	if string(fetched.Password) != "initial-secret" {
		t.Fatalf("fetched entry password mismatch: %s", string(fetched.Password))
	}

	bsOld := store.NewBoltStore()
	if err := bsOld.OpenVault(vaultPath, oldKeyCopy); err != nil {
		t.Fatalf("old key open should succeed for comparison: %v", err)
	}
	defer func() {
		if err := bsOld.CloseVault(); err != nil {
			t.Logf("Warning: failed to close old vault: %v", err)
		}
	}()

	if _, err := bsOld.GetEntry("default", entry.ID); err == nil {
		t.Fatalf("expected decrypt failure when using old master key")
	} else if !strings.Contains(err.Error(), "failed to decrypt entry") {
		t.Fatalf("unexpected error when using old master key: %v", err)
	}
}
