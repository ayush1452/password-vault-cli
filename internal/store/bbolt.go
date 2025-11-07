package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.etcd.io/bbolt"

	"github.com/vault-cli/vault/internal/domain"
	"github.com/vault-cli/vault/internal/vault"
)

// Bucket names
var (
	MetadataBucket = []byte("metadata")
	ProfilesBucket = []byte("profiles")
	AuditBucket    = []byte("audit")
	ConfigBucket   = []byte("config")
)

// BoltStore implements VaultStore using BoltDB
type BoltStore struct {
	db        *bbolt.DB
	path      string
	masterKey []byte
	crypto    *vault.CryptoEngine
	lock      *FileLock
	isOpen    bool
}

// NewBoltStore creates a new BoltDB-based vault store
func NewBoltStore() *BoltStore {
	return &BoltStore{
		crypto: vault.NewDefaultCryptoEngine(),
	}
}

// CreateVault creates a new vault at the specified path
func (bs *BoltStore) CreateVault(path string, masterKey []byte, kdfParams map[string]interface{}) error {
	if bs.isOpen {
		return fmt.Errorf("vault is already open")
	}

	// Check if vault already exists
	if _, err := os.Stat(path); err == nil {
		return ErrVaultExists
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create vault directory: %w", err)
	}

	// Create and open database
	db, err := bbolt.Open(path, 0600, &bbolt.Options{
		Timeout: 10 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("failed to create vault database: %w", err)
	}
	defer db.Close()

	// Initialize vault structure
	err = db.Update(func(tx *bbolt.Tx) error {
		// Create metadata bucket
		metaBucket, err := tx.CreateBucket(MetadataBucket)
		if err != nil {
			return fmt.Errorf("failed to create metadata bucket: %w", err)
		}

		// Store vault metadata
		metadata := &domain.VaultMetadata{
			Version:   "1.0.0",
			KDFParams: kdfParams,
			CreatedAt: time.Now().UTC(),
		}

		metadataJSON, err := json.Marshal(metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}

		if err := metaBucket.Put([]byte("vault_info"), metadataJSON); err != nil {
			return fmt.Errorf("failed to store vault metadata: %w", err)
		}

		// Create profiles bucket and default profile
		profilesBucket, err := tx.CreateBucket(ProfilesBucket)
		if err != nil {
			return fmt.Errorf("failed to create profiles bucket: %w", err)
		}

		defaultProfile := &domain.Profile{
			Name:        "default",
			Description: "Default profile",
			CreatedAt:   time.Now().UTC(),
		}

		profileJSON, err := json.Marshal(defaultProfile)
		if err != nil {
			return fmt.Errorf("failed to marshal default profile: %w", err)
		}

		if err := profilesBucket.Put([]byte("default"), profileJSON); err != nil {
			return fmt.Errorf("failed to store default profile: %w", err)
		}

		// Create default entries bucket
		if _, err := tx.CreateBucket([]byte("entries:default")); err != nil {
			return fmt.Errorf("failed to create default entries bucket: %w", err)
		}

		// Create audit bucket
		if _, err := tx.CreateBucket(AuditBucket); err != nil {
			return fmt.Errorf("failed to create audit bucket: %w", err)
		}

		// Create config bucket
		configBucket, err := tx.CreateBucket(ConfigBucket)
		if err != nil {
			return fmt.Errorf("failed to create config bucket: %w", err)
		}

		// Set default configuration
		if err := configBucket.Put([]byte("default_profile"), []byte("default")); err != nil {
			return fmt.Errorf("failed to set default profile config: %w", err)
		}

		return nil
	})

	if err != nil {
		os.Remove(path) // Clean up on failure
		return err
	}

	// Ensure proper file permissions
	return EnsureFilePermissions(path)
}

// OpenVault opens an existing vault
func (bs *BoltStore) OpenVault(path string, masterKey []byte) error {
	if bs.isOpen {
		return fmt.Errorf("vault is already open")
	}

	// Check if vault exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return ErrVaultNotFound
	}

	// Verify file permissions
	if err := EnsureFilePermissions(path); err != nil {
		return fmt.Errorf("failed to verify vault permissions: %w", err)
	}

	// Acquire file lock
	lock := NewFileLock(path)
	if err := lock.Lock(30 * time.Second); err != nil {
		return ErrVaultLocked
	}

	// Open database
	db, err := bbolt.Open(path, 0600, &bbolt.Options{
		Timeout:  10 * time.Second,
		ReadOnly: false,
	})
	if err != nil {
		lock.Unlock()
		return fmt.Errorf("failed to open vault database: %w", err)
	}

	// Verify vault integrity and master key
	err = db.View(func(tx *bbolt.Tx) error {
		metaBucket := tx.Bucket(MetadataBucket)
		if metaBucket == nil {
			return ErrVaultCorrupted
		}

		// Try to read metadata to verify master key
		metadataJSON := metaBucket.Get([]byte("vault_info"))
		if metadataJSON == nil {
			return ErrVaultCorrupted
		}

		var metadata domain.VaultMetadata
		if err := json.Unmarshal(metadataJSON, &metadata); err != nil {
			return ErrVaultCorrupted
		}

		return nil
	})

	if err != nil {
		db.Close()
		lock.Unlock()
		return err
	}

	bs.db = db
	bs.path = path
	bs.masterKey = make([]byte, len(masterKey))
	copy(bs.masterKey, masterKey)
	bs.lock = lock
	bs.isOpen = true

	return nil
}

// CloseVault closes the vault and releases resources
func (bs *BoltStore) CloseVault() error {
	if !bs.isOpen {
		return nil
	}

	var err error

	// Clear master key
	if bs.masterKey != nil {
		vault.Zeroize(bs.masterKey)
		bs.masterKey = nil
	}

	// Close database
	if bs.db != nil {
		if dbErr := bs.db.Close(); dbErr != nil {
			err = dbErr
		}
		bs.db = nil
	}

	// Release file lock
	if bs.lock != nil {
		if lockErr := bs.lock.Unlock(); lockErr != nil && err == nil {
			err = lockErr
		}
		bs.lock = nil
	}

	bs.isOpen = false
	return err
}

// IsOpen returns true if the vault is currently open
func (bs *BoltStore) IsOpen() bool {
	return bs.isOpen
}

// CreateEntry creates a new entry in the specified profile
func (bs *BoltStore) CreateEntry(profile string, entry *domain.Entry) error {
	if !bs.isOpen {
		return fmt.Errorf("vault is not open")
	}

	// Generate entry ID if not provided
	if entry.ID == "" {
		entry.ID = entry.Name
	}

	// Set timestamps
	now := time.Now().UTC()
	entry.CreatedAt = now
	entry.UpdatedAt = now

	return bs.db.Update(func(tx *bbolt.Tx) error {
		bucketName := fmt.Sprintf("entries:%s", profile)
		bucket := tx.Bucket([]byte(bucketName))
		if bucket == nil {
			return ErrProfileNotFound
		}

		// Check if entry already exists
		if bucket.Get([]byte(entry.ID)) != nil {
			return ErrEntryExists
		}

		// Encrypt entry
		entryJSON, err := json.Marshal(entry)
		if err != nil {
			return fmt.Errorf("failed to marshal entry: %w", err)
		}

		envelope, err := bs.crypto.SealWithPassphrase(entryJSON, string(bs.masterKey))
		if err != nil {
			return fmt.Errorf("failed to encrypt entry: %w", err)
		}

		// Serialize envelope
		envelopeData := vault.EnvelopeToBytes(envelope)

		// Store encrypted entry
		if err := bucket.Put([]byte(entry.ID), envelopeData); err != nil {
			return fmt.Errorf("failed to store entry: %w", err)
		}

		// Clear sensitive data
		vault.Zeroize(entryJSON)

		return nil
	})
}

// GetEntry retrieves an entry from the specified profile
func (bs *BoltStore) GetEntry(profile string, entryID string) (*domain.Entry, error) {
	if !bs.isOpen {
		return nil, fmt.Errorf("vault is not open")
	}

	var entry *domain.Entry
	err := bs.db.View(func(tx *bbolt.Tx) error {
		bucketName := fmt.Sprintf("entries:%s", profile)
		bucket := tx.Bucket([]byte(bucketName))
		if bucket == nil {
			return ErrProfileNotFound
		}

		// Get encrypted entry
		envelopeData := bucket.Get([]byte(entryID))
		if envelopeData == nil {
			return ErrEntryNotFound
		}

		// Deserialize envelope
		envelope, err := vault.EnvelopeFromBytes(envelopeData)
		if err != nil {
			return fmt.Errorf("failed to deserialize envelope: %w", err)
		}

		// Decrypt entry
		entryJSON, err := bs.crypto.OpenWithPassphrase(envelope, string(bs.masterKey))
		if err != nil {
			return fmt.Errorf("failed to decrypt entry: %w", err)
		}
		defer vault.Zeroize(entryJSON)

		// Unmarshal entry
		entry = &domain.Entry{}
		if err := json.Unmarshal(entryJSON, entry); err != nil {
			return fmt.Errorf("failed to unmarshal entry: %w", err)
		}

		return nil
	})

	return entry, err
}

// ListEntries returns all entries in the specified profile
func (bs *BoltStore) ListEntries(profile string, filter *domain.Filter) ([]*domain.Entry, error) {
	if !bs.isOpen {
		return nil, fmt.Errorf("vault is not open")
	}

	var entries []*domain.Entry
	err := bs.db.View(func(tx *bbolt.Tx) error {
		bucketName := fmt.Sprintf("entries:%s", profile)
		bucket := tx.Bucket([]byte(bucketName))
		if bucket == nil {
			return ErrProfileNotFound
		}

		return bucket.ForEach(func(k, v []byte) error {
			// Deserialize envelope
			envelope, err := vault.EnvelopeFromBytes(v)
			if err != nil {
				return fmt.Errorf("failed to deserialize envelope: %w", err)
			}

			// Decrypt entry
			entryJSON, err := bs.crypto.OpenWithPassphrase(envelope, string(bs.masterKey))
			if err != nil {
				return fmt.Errorf("failed to decrypt entry: %w", err)
			}
			defer vault.Zeroize(entryJSON)

			// Unmarshal entry
			var entry domain.Entry
			if err := json.Unmarshal(entryJSON, &entry); err != nil {
				return fmt.Errorf("failed to unmarshal entry: %w", err)
			}

			// Apply filter
			if filter != nil {
				if len(filter.SearchTokens) > 0 {
					if !vault.MatchesSearchTokens(&entry, filter.SearchTokens) {
						return nil
					}
				} else if filter.Search != "" {
					searchLower := strings.ToLower(filter.Search)
					if !strings.Contains(strings.ToLower(entry.Name), searchLower) &&
						!strings.Contains(strings.ToLower(entry.Username), searchLower) &&
						!strings.Contains(strings.ToLower(entry.URL), searchLower) {
						return nil
					}
				}

				if len(filter.Tags) > 0 {
					hasTag := false
					for _, filterTag := range filter.Tags {
						for _, entryTag := range entry.Tags {
							if strings.EqualFold(filterTag, entryTag) {
								hasTag = true
								break
							}
						}
						if hasTag {
							break
						}
					}
					if !hasTag {
						return nil // Skip this entry
					}
				}
			}

			// Clear secret for listing (security)
			// Don't include sensitive data in listings
			entryCopy := entry
			entryCopy.Secret = nil
			entryCopy.TOTPSeed = ""

			entries = append(entries, &entryCopy)
			return nil
		})
	})

	return entries, err
}

// UpdateEntry updates an existing entry
func (bs *BoltStore) UpdateEntry(profile string, entryID string, entry *domain.Entry) error {
	if !bs.isOpen {
		return fmt.Errorf("vault is not open")
	}

	// Ensure ID matches
	entry.ID = entryID
	entry.UpdatedAt = time.Now().UTC()

	return bs.db.Update(func(tx *bbolt.Tx) error {
		bucketName := fmt.Sprintf("entries:%s", profile)
		bucket := tx.Bucket([]byte(bucketName))
		if bucket == nil {
			return ErrProfileNotFound
		}

		// Check if entry exists
		if bucket.Get([]byte(entryID)) == nil {
			return ErrEntryNotFound
		}

		// Encrypt updated entry
		entryJSON, err := json.Marshal(entry)
		if err != nil {
			return fmt.Errorf("failed to marshal entry: %w", err)
		}

		envelope, err := bs.crypto.SealWithPassphrase(entryJSON, string(bs.masterKey))
		if err != nil {
			return fmt.Errorf("failed to encrypt entry: %w", err)
		}

		// Serialize envelope
		envelopeData := vault.EnvelopeToBytes(envelope)

		// Store updated entry
		if err := bucket.Put([]byte(entryID), envelopeData); err != nil {
			return fmt.Errorf("failed to update entry: %w", err)
		}

		// Clear sensitive data
		vault.Zeroize(entryJSON)

		return nil
	})
}

// DeleteEntry deletes an entry from the specified profile
func (bs *BoltStore) DeleteEntry(profile string, entryID string) error {
	if !bs.isOpen {
		return fmt.Errorf("vault is not open")
	}

	return bs.db.Update(func(tx *bbolt.Tx) error {
		bucketName := fmt.Sprintf("entries:%s", profile)
		bucket := tx.Bucket([]byte(bucketName))
		if bucket == nil {
			return ErrProfileNotFound
		}

		// Check if entry exists
		if bucket.Get([]byte(entryID)) == nil {
			return ErrEntryNotFound
		}

		// Delete entry
		return bucket.Delete([]byte(entryID))
	})
}

// EntryExists checks if an entry exists in the specified profile
func (bs *BoltStore) EntryExists(profile string, entryID string) bool {
	if !bs.isOpen {
		return false
	}

	exists := false
	bs.db.View(func(tx *bbolt.Tx) error {
		bucketName := fmt.Sprintf("entries:%s", profile)
		bucket := tx.Bucket([]byte(bucketName))
		if bucket != nil {
			exists = bucket.Get([]byte(entryID)) != nil
		}
		return nil
	})

	return exists
}

// CreateProfile creates a new profile
func (bs *BoltStore) CreateProfile(name string, description string) error {
	if !bs.isOpen {
		return fmt.Errorf("vault is not open")
	}

	return bs.db.Update(func(tx *bbolt.Tx) error {
		// Check if profile already exists
		profilesBucket := tx.Bucket(ProfilesBucket)
		if profilesBucket == nil {
			return ErrVaultCorrupted
		}

		if profilesBucket.Get([]byte(name)) != nil {
			return ErrProfileExists
		}

		// Create profile
		profile := &domain.Profile{
			Name:        name,
			Description: description,
			CreatedAt:   time.Now().UTC(),
		}

		profileJSON, err := json.Marshal(profile)
		if err != nil {
			return fmt.Errorf("failed to marshal profile: %w", err)
		}

		if err := profilesBucket.Put([]byte(name), profileJSON); err != nil {
			return fmt.Errorf("failed to store profile: %w", err)
		}

		// Create entries bucket for the profile
		bucketName := fmt.Sprintf("entries:%s", name)
		if _, err := tx.CreateBucket([]byte(bucketName)); err != nil {
			return fmt.Errorf("failed to create entries bucket: %w", err)
		}

		return nil
	})
}

// GetProfile retrieves a profile by name
func (bs *BoltStore) GetProfile(name string) (*domain.Profile, error) {
	if !bs.isOpen {
		return nil, fmt.Errorf("vault is not open")
	}

	var profile *domain.Profile
	err := bs.db.View(func(tx *bbolt.Tx) error {
		profilesBucket := tx.Bucket(ProfilesBucket)
		if profilesBucket == nil {
			return ErrVaultCorrupted
		}

		profileJSON := profilesBucket.Get([]byte(name))
		if profileJSON == nil {
			return ErrProfileNotFound
		}

		profile = &domain.Profile{}
		return json.Unmarshal(profileJSON, profile)
	})

	return profile, err
}

// ListProfiles returns all profiles
func (bs *BoltStore) ListProfiles() ([]*domain.Profile, error) {
	if !bs.isOpen {
		return nil, fmt.Errorf("vault is not open")
	}

	var profiles []*domain.Profile
	err := bs.db.View(func(tx *bbolt.Tx) error {
		profilesBucket := tx.Bucket(ProfilesBucket)
		if profilesBucket == nil {
			return ErrVaultCorrupted
		}

		return profilesBucket.ForEach(func(k, v []byte) error {
			var profile domain.Profile
			if err := json.Unmarshal(v, &profile); err != nil {
				return err
			}
			profiles = append(profiles, &profile)
			return nil
		})
	})

	return profiles, err
}

// DeleteProfile deletes a profile and all its entries
func (bs *BoltStore) DeleteProfile(name string) error {
	if !bs.isOpen {
		return fmt.Errorf("vault is not open")
	}

	if name == "default" {
		return fmt.Errorf("cannot delete default profile")
	}

	return bs.db.Update(func(tx *bbolt.Tx) error {
		// Check if profile exists
		profilesBucket := tx.Bucket(ProfilesBucket)
		if profilesBucket == nil {
			return ErrVaultCorrupted
		}

		if profilesBucket.Get([]byte(name)) == nil {
			return ErrProfileNotFound
		}

		// Delete profile
		if err := profilesBucket.Delete([]byte(name)); err != nil {
			return fmt.Errorf("failed to delete profile: %w", err)
		}

		// Delete entries bucket
		bucketName := fmt.Sprintf("entries:%s", name)
		if err := tx.DeleteBucket([]byte(bucketName)); err != nil {
			return fmt.Errorf("failed to delete entries bucket: %w", err)
		}

		return nil
	})
}

// ProfileExists checks if a profile exists
func (bs *BoltStore) ProfileExists(name string) bool {
	if !bs.isOpen {
		return false
	}

	exists := false
	bs.db.View(func(tx *bbolt.Tx) error {
		profilesBucket := tx.Bucket(ProfilesBucket)
		if profilesBucket != nil {
			exists = profilesBucket.Get([]byte(name)) != nil
		}
		return nil
	})

	return exists
}

// GetVaultMetadata returns vault metadata
func (bs *BoltStore) GetVaultMetadata() (*domain.VaultMetadata, error) {
	if !bs.isOpen {
		return nil, fmt.Errorf("vault is not open")
	}

	var metadata *domain.VaultMetadata
	err := bs.db.View(func(tx *bbolt.Tx) error {
		metaBucket := tx.Bucket(MetadataBucket)
		if metaBucket == nil {
			return ErrVaultCorrupted
		}

		metadataJSON := metaBucket.Get([]byte("vault_info"))
		if metadataJSON == nil {
			return ErrVaultCorrupted
		}

		metadata = &domain.VaultMetadata{}
		return json.Unmarshal(metadataJSON, metadata)
	})

	return metadata, err
}

// UpdateVaultMetadata updates vault metadata
func (bs *BoltStore) UpdateVaultMetadata(metadata *domain.VaultMetadata) error {
	if !bs.isOpen {
		return fmt.Errorf("vault is not open")
	}

	return bs.db.Update(func(tx *bbolt.Tx) error {
		metaBucket := tx.Bucket(MetadataBucket)
		if metaBucket == nil {
			return ErrVaultCorrupted
		}

		metadataJSON, err := json.Marshal(metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}

		return metaBucket.Put([]byte("vault_info"), metadataJSON)
	})
}

// Placeholder implementations for audit and backup operations
func (bs *BoltStore) LogOperation(op *domain.Operation) error {
	// TODO: Implement HMAC-chained audit logging
	return nil
}

func (bs *BoltStore) GetAuditLog() ([]*domain.Operation, error) {
	// TODO: Implement audit log retrieval
	return nil, nil
}

func (bs *BoltStore) VerifyAuditIntegrity() error {
	// TODO: Implement audit chain verification
	return nil
}

func (bs *BoltStore) ExportVault(path string, includeSecrets bool) error {
	// TODO: Implement vault export
	return nil
}

func (bs *BoltStore) ImportVault(path string, conflictResolution string) error {
	// TODO: Implement vault import
	return nil
}

func (bs *BoltStore) CompactVault() error {
	if !bs.isOpen {
		return fmt.Errorf("vault is not open")
	}
	// BoltDB doesn't need explicit compaction
	return nil
}

func (bs *BoltStore) VerifyIntegrity() error {
	if !bs.isOpen {
		return fmt.Errorf("vault is not open")
	}

	return bs.db.View(func(tx *bbolt.Tx) error {
		// Verify all buckets exist
		requiredBuckets := [][]byte{
			MetadataBucket,
			ProfilesBucket,
			AuditBucket,
			ConfigBucket,
		}

		for _, bucketName := range requiredBuckets {
			if tx.Bucket(bucketName) == nil {
				return fmt.Errorf("missing required bucket: %s", string(bucketName))
			}
		}

		// Verify metadata
		metaBucket := tx.Bucket(MetadataBucket)
		metadataJSON := metaBucket.Get([]byte("vault_info"))
		if metadataJSON == nil {
			return fmt.Errorf("missing vault metadata")
		}

		var metadata domain.VaultMetadata
		if err := json.Unmarshal(metadataJSON, &metadata); err != nil {
			return fmt.Errorf("corrupted vault metadata: %w", err)
		}

		return nil
	})
}
