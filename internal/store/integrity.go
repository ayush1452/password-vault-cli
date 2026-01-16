package store

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"

	"go.etcd.io/bbolt"

	"github.com/vault-cli/vault/internal/domain"
	"github.com/vault-cli/vault/internal/vault"
)

// ComputeVaultFileHMAC computes HMAC-SHA256 of the entire vault file
func ComputeVaultFileHMAC(vaultPath string, masterKey []byte) (string, error) {
	// Read vault file
	data, err := os.ReadFile(vaultPath)
	if err != nil {
		return "", fmt.Errorf("failed to read vault file: %w", err)
	}

	// Compute HMAC using master key
	hmacBytes := vault.ComputeHMAC(data, masterKey)
	
	// Encode to base64 for storage
	return base64.StdEncoding.EncodeToString(hmacBytes), nil
}

// VerifyVaultFileHMAC verifies the HMAC of the vault file
func VerifyVaultFileHMAC(vaultPath string, masterKey []byte, expectedHMAC string) error {
	// Read vault file
	data, err := os.ReadFile(vaultPath)
	if err != nil {
		return fmt.Errorf("failed to read vault file: %w", err)
	}

	// Decode expected HMAC
	expectedHMACBytes, err := base64.StdEncoding.DecodeString(expectedHMAC)
	if err != nil {
		return fmt.Errorf("failed to decode HMAC: %w", err)
	}

	// Verify HMAC
	if !vault.VerifyHMAC(data, masterKey, expectedHMACBytes) {
		return ErrVaultCorrupted
	}

	return nil
}

// UpdateVaultFileHMAC updates the HMAC in the vault metadata after modifications
func (bs *BoltStore) UpdateVaultFileHMAC() error {
	if !bs.isOpen {
		return fmt.Errorf("vault is not open")
	}

	// Compute new HMAC
	hmac, err := ComputeVaultFileHMAC(bs.path, bs.masterKey)
	if err != nil {
		return fmt.Errorf("failed to compute HMAC: %w", err)
	}

	// Update metadata
	return bs.db.Update(func(tx *bbolt.Tx) error {
		metaBucket := tx.Bucket(MetadataBucket)
		if metaBucket == nil {
			return ErrVaultCorrupted
		}

		// Get current metadata
		metadataJSON := metaBucket.Get([]byte("vault_info"))
		if metadataJSON == nil {
			return ErrVaultCorrupted
		}

		var metadata domain.VaultMetadata
		if err := json.Unmarshal(metadataJSON, &metadata); err != nil {
			return err
		}

		// Update HMAC
		metadata.FileHMAC = hmac

		// Save updated metadata
		updatedJSON, err := json.Marshal(metadata)
		if err != nil {
			return err
		}

		return metaBucket.Put([]byte("vault_info"), updatedJSON)
	})
}
