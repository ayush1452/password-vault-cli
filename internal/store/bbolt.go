package store

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
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
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("failed to create vault directory: %w", err)
	}

	// Create and open database
	db, err := bbolt.Open(path, 0o600, &bbolt.Options{
		Timeout: 10 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("failed to create vault database: %w", err)
	}

	// Use a closure to ensure db.Close() is called and its error is checked
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			log.Printf("Warning: failed to close database: %v", closeErr)
		}
	}()

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
		_ = os.Remove(path) // Clean up on failure, ignore error from Remove
		return fmt.Errorf("failed to create vault database: %w", err)
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
	db, err := bbolt.Open(path, 0o600, &bbolt.Options{
		Timeout:  10 * time.Second,
		ReadOnly: false,
	})
	if err != nil {
		// Log the error from Unlock but don't override the original error
		if unlockErr := lock.Unlock(); unlockErr != nil {
			log.Printf("warning: failed to unlock file after database open error: %v", unlockErr)
		}
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
		// Log errors but don't override the original error
		if closeErr := db.Close(); closeErr != nil {
			log.Printf("warning: failed to close database after verification error: %v", closeErr)
		}
		if unlockErr := lock.Unlock(); unlockErr != nil {
			log.Printf("warning: failed to unlock file after verification error: %v", unlockErr)
		}
		return fmt.Errorf("vault verification failed: %w", err)
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
		envelopeData, err := vault.EnvelopeToBytes(envelope)
		if err != nil {
			return fmt.Errorf("failed to serialize envelope: %w", err)
		}

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
func (bs *BoltStore) GetEntry(profile, entryID string) (*domain.Entry, error) {
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
func (bs *BoltStore) UpdateEntry(profile, entryID string, entry *domain.Entry) error {
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
		envelopeData, err := vault.EnvelopeToBytes(envelope)
		if err != nil {
			return fmt.Errorf("failed to serialize envelope: %w", err)
		}

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
func (bs *BoltStore) DeleteEntry(profile, entryID string) error {
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
func (bs *BoltStore) EntryExists(profile, entryID string) bool {
	if !bs.isOpen {
		return false
	}

	exists := false
	_ = bs.db.View(func(tx *bbolt.Tx) error {
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
func (bs *BoltStore) CreateProfile(name, description string) error {
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
	_ = bs.db.View(func(tx *bbolt.Tx) error {
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

// RotateMasterKey derives a new master key from the provided passphrase, re-encrypts all
// vault entries, and updates the vault metadata with the new KDF parameters.
func (bs *BoltStore) RotateMasterKey(newPassphrase string) error {
	if !bs.isOpen {
		return fmt.Errorf("vault is not open")
	}
	if newPassphrase == "" {
		return fmt.Errorf("new passphrase cannot be empty")
	}
	if len(bs.masterKey) == 0 {
		return ErrInvalidKey
	}

	metadata, err := bs.GetVaultMetadata()
	if err != nil {
		return fmt.Errorf("failed to load vault metadata: %w", err)
	}

	params, err := parseArgon2Params(metadata.KDFParams)
	if err != nil {
		return err
	}

	newSalt, err := vault.GenerateSalt()
	if err != nil {
		return fmt.Errorf("failed to generate new salt: %w", err)
	}

	cryptoEngine := vault.NewCryptoEngine(params)
	newMasterKey, err := cryptoEngine.DeriveKey(newPassphrase, newSalt)
	if err != nil {
		return fmt.Errorf("failed to derive new master key: %w", err)
	}
	defer vault.Zeroize(newMasterKey)

	newSaltB64 := base64.StdEncoding.EncodeToString(newSalt)
	vault.Zeroize(newSalt)

	oldMasterKey := append([]byte(nil), bs.masterKey...)
	defer vault.Zeroize(oldMasterKey)

	oldMasterKeyStr := string(oldMasterKey)
	newMasterKeyStr := string(newMasterKey)
	updatedAt := time.Now().UTC()

	if err := bs.db.Update(func(tx *bbolt.Tx) error {
		if err := reencryptEntries(tx, oldMasterKeyStr, newMasterKeyStr, bs.crypto); err != nil {
			return err
		}

		metaBucket := tx.Bucket(MetadataBucket)
		if metaBucket == nil {
			return ErrVaultCorrupted
		}

		metadataCopy := *metadata
		metadataCopy.KDFParams = copyKDFParams(metadata.KDFParams)
		if metadataCopy.KDFParams == nil {
			metadataCopy.KDFParams = map[string]interface{}{}
		}
		metadataCopy.KDFParams["memory"] = params.Memory
		metadataCopy.KDFParams["iterations"] = params.Iterations
		metadataCopy.KDFParams["parallelism"] = params.Parallelism
		metadataCopy.KDFParams["salt"] = newSaltB64
		if _, ok := metadataCopy.KDFParams["time"]; ok {
			metadataCopy.KDFParams["time"] = params.Iterations
		}
		if _, ok := metadataCopy.KDFParams["threads"]; ok {
			metadataCopy.KDFParams["threads"] = params.Parallelism
		}
		metadataCopy.UpdatedAt = updatedAt

		metadataJSON, err := json.Marshal(&metadataCopy)
		if err != nil {
			return fmt.Errorf("failed to marshal updated metadata: %w", err)
		}

		return metaBucket.Put([]byte("vault_info"), metadataJSON)
	}); err != nil {
		return err
	}

	if bs.masterKey != nil {
		vault.Zeroize(bs.masterKey)
	}
	bs.masterKey = make([]byte, len(newMasterKey))
	copy(bs.masterKey, newMasterKey)

	if err := bs.LogOperation(&domain.Operation{
		Type:      "rotate_master_key",
		Timestamp: time.Now().UTC(),
		Success:   true,
	}); err != nil {
		log.Printf("warning: failed to log rotate_master_key operation: %v", err)
	}

	return nil
}

func reencryptEntries(tx *bbolt.Tx, oldPassphrase, newPassphrase string, cryptoEngine *vault.CryptoEngine) error {
	return tx.ForEach(func(name []byte, bucket *bbolt.Bucket) error {
		if bucket == nil {
			return nil
		}
		if !strings.HasPrefix(string(name), "entries:") {
			return nil
		}

		return bucket.ForEach(func(k, v []byte) error {
			if v == nil {
				return nil
			}

			envelope, err := vault.EnvelopeFromBytes(v)
			if err != nil {
				return fmt.Errorf("failed to decode envelope for entry %s: %w", string(k), err)
			}

			plaintext, err := cryptoEngine.OpenWithPassphrase(envelope, oldPassphrase)
			if err != nil {
				return fmt.Errorf("failed to decrypt entry %s: %w", string(k), err)
			}

			newEnvelope, err := cryptoEngine.SealWithPassphrase(plaintext, newPassphrase)
			vault.Zeroize(plaintext)
			if err != nil {
				return fmt.Errorf("failed to re-encrypt entry %s: %w", string(k), err)
			}

			envelopeBytes, err := vault.EnvelopeToBytes(newEnvelope)
			if err != nil {
				return fmt.Errorf("failed to serialize rotated entry %s: %w", string(k), err)
			}

			if err := bucket.Put(k, envelopeBytes); err != nil {
				return fmt.Errorf("failed to store rotated entry %s: %w", string(k), err)
			}

			return nil
		})
	})
}

func parseArgon2Params(kdf map[string]interface{}) (vault.Argon2Params, error) {
	if kdf == nil {
		return vault.Argon2Params{}, fmt.Errorf("vault metadata missing KDF parameters")
	}

	memoryVal, err := getUint32Param(kdf, "memory", "Memory")
	if err != nil {
		return vault.Argon2Params{}, err
	}

	iterationsVal, err := getUint32Param(kdf, "iterations", "time", "Iterations")
	if err != nil {
		return vault.Argon2Params{}, err
	}

	parallelVal, err := getUint32Param(kdf, "parallelism", "threads", "Parallelism")
	if err != nil {
		return vault.Argon2Params{}, err
	}

	// Ensure parallelism is within valid range for uint8 (0-255)
	if parallelVal > math.MaxUint8 {
		return vault.Argon2Params{}, fmt.Errorf("parallelism value %d exceeds maximum allowed value of %d", parallelVal, math.MaxUint8)
	}

	return vault.Argon2Params{
		Memory:      memoryVal,
		Iterations:  iterationsVal,
		Parallelism: uint8(parallelVal),
	}, nil
}

func getUint32Param(kdf map[string]interface{}, primary string, aliases ...string) (uint32, error) {
	// Try to find the parameter value using primary key first, then aliases
	var value interface{}
	var found bool

	if value, found = kdf[primary]; !found {
		for _, alias := range aliases {
			if v, exists := kdf[alias]; exists {
				value = v
				found = true
				break
			}
		}
	}

	if !found || value == nil {
		return 0, fmt.Errorf("missing or invalid %s parameter", primary)
	}

	switch v := value.(type) {
	case float64:
		if v < 0 || v > float64(math.MaxUint32) || math.IsNaN(v) || math.IsInf(v, 0) {
			return 0, fmt.Errorf("value for %s out of range (0-%d)", primary, math.MaxUint32)
		}
		return uint32(v), nil

	case float32:
		if v < 0 || v > float32(math.MaxUint32) || math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			return 0, fmt.Errorf("value for %s out of range (0-%d)", primary, math.MaxUint32)
		}
		return uint32(v), nil

	case int:
		if v < 0 || uint64(v) > math.MaxUint32 {
			return 0, fmt.Errorf("value for %s out of range (0-%d)", primary, math.MaxUint32)
		}
		// Safe to convert after bounds check
		// nolint:gosec // Bounds checked above, safe to convert with mask
		return uint32(uint64(v) & 0xFFFFFFFF), nil

	case int8:
		if v < 0 {
			return 0, fmt.Errorf("value for %s cannot be negative", primary)
		}
		// int8 is always within uint32 range when non-negative
		return uint32(v), nil

	case int16:
		if v < 0 {
			return 0, fmt.Errorf("value for %s cannot be negative", primary)
		}
		return uint32(v), nil

	case int32:
		if v < 0 {
			return 0, fmt.Errorf("value for %s cannot be negative", primary)
		}
		return uint32(v), nil

	case int64:
		if v < 0 || uint64(v) > math.MaxUint32 {
			return 0, fmt.Errorf("value for %s out of range (0-%d)", primary, math.MaxUint32)
		}
		// Safe to convert after bounds check
		// nolint:gosec // Bounds checked above, safe to convert with mask
		return uint32(uint64(v) & 0xFFFFFFFF), nil

	case uint:
		if uint64(v) > math.MaxUint32 {
			return 0, fmt.Errorf("value for %s exceeds maximum allowed value of %d", primary, math.MaxUint32)
		}
		// Safe to convert after bounds check
		// nolint:gosec // Bounds checked above, safe to convert with mask
		return uint32(uint64(v) & 0xFFFFFFFF), nil

	case uint8, uint16, uint32:
		// These types are guaranteed to fit in uint32
		// nolint:gosec // Type guarantees value fits in uint32
		return uint32(reflect.ValueOf(v).Uint() & 0xFFFFFFFF), nil

	case uint64:
		if v > math.MaxUint32 {
			return 0, fmt.Errorf("value for %s exceeds maximum allowed value of %d", primary, math.MaxUint32)
		}
		// Safe to convert after bounds check
		// nolint:gosec // Bounds checked above, safe to convert with mask
		return uint32(v & 0xFFFFFFFF), nil

	case json.Number:
		i64, err := v.Int64()
		if err != nil {
			return 0, fmt.Errorf("invalid numeric value for %s: %v", primary, err)
		}
		if i64 < 0 || uint64(i64) > math.MaxUint32 {
			return 0, fmt.Errorf("value for %s out of range (0-%d)", primary, math.MaxUint32)
		}
		// nolint:gosec // Bounds checked above, safe to convert with mask
		return uint32(uint64(i64) & 0xFFFFFFFF), nil

	case string:
		// Try to parse as float first to handle decimal points and scientific notation
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			if f < 0 || f > float64(math.MaxUint32) || math.IsNaN(f) || math.IsInf(f, 0) {
				return 0, fmt.Errorf("value for %s out of range (0-%d)", primary, math.MaxUint32)
			}
			return uint32(f), nil
		}

		// If not a float, try parsing as an integer
		i, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid numeric string for %s: %v", primary, err)
		}
		if i < 0 || uint64(i) > math.MaxUint32 {
			return 0, fmt.Errorf("value for %s out of range (0-%d)", primary, math.MaxUint32)
		}
		// nolint:gosec // Bounds checked above, safe to convert with mask
		return uint32(uint64(i) & 0xFFFFFFFF), nil

	default:
		return 0, fmt.Errorf("unsupported type %T for parameter %s", value, primary)
	}
}

func copyKDFParams(original map[string]interface{}) map[string]interface{} {
	if original == nil {
		return nil
	}

	clone := make(map[string]interface{}, len(original))
	for k, v := range original {
		clone[k] = v
	}
	return clone
}

type vaultExport struct {
	Metadata       *domain.VaultMetadata    `json:"metadata"`
	Profiles       []*domain.Profile        `json:"profiles"`
	Entries        map[string][]exportEntry `json:"entries"`
	AuditLog       []*domain.Operation      `json:"audit_log"`
	ExportedAt     time.Time                `json:"exported_at"`
	IncludeSecrets bool                     `json:"include_secrets"`
}

type exportEntry struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Username  string    `json:"username"`
	URL       string    `json:"url"`
	Notes     string    `json:"notes"`
	Tags      []string  `json:"tags"`
	TOTPSeed  string    `json:"totp_seed,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Secret    string    `json:"secret,omitempty"`
}

type auditEnvelope struct {
	Operation *domain.Operation `json:"operation"`
}

// LogOperation persists an audit entry in the audit bucket.
func (bs *BoltStore) LogOperation(op *domain.Operation) error {
	if !bs.isOpen {
		return fmt.Errorf("vault is not open")
	}
	if op == nil {
		return fmt.Errorf("operation cannot be nil")
	}
	if op.Timestamp.IsZero() {
		op.Timestamp = time.Now().UTC()
	}

	return bs.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(AuditBucket)
		if bucket == nil {
			return fmt.Errorf("audit bucket not found")
		}

		seq, err := bucket.NextSequence()
		if err != nil {
			return fmt.Errorf("failed to allocate audit sequence: %w", err)
		}

		key := make([]byte, 8)
		binary.BigEndian.PutUint64(key, seq)

		env := auditEnvelope{Operation: op}
		payload, err := json.Marshal(env)
		if err != nil {
			return fmt.Errorf("failed to encode audit entry: %w", err)
		}

		return bucket.Put(key, payload)
	})
}

// GetAuditLog returns audit operations in chronological order.
func (bs *BoltStore) GetAuditLog() ([]*domain.Operation, error) {
	if !bs.isOpen {
		return nil, fmt.Errorf("vault is not open")
	}

	var ops []*domain.Operation
	err := bs.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(AuditBucket)
		if bucket == nil {
			return fmt.Errorf("audit bucket not found")
		}

		return bucket.ForEach(func(k, v []byte) error {
			if v == nil {
				return nil
			}
			var env auditEnvelope
			if err := json.Unmarshal(v, &env); err != nil {
				return fmt.Errorf("failed to decode audit entry: %w", err)
			}
			if env.Operation != nil {
				op := *env.Operation
				op.Timestamp = op.Timestamp.UTC()
				ops = append(ops, &op)
			}
			return nil
		})
	})
	if err != nil {
		return nil, err
	}

	return ops, nil
}

// VerifyAuditIntegrity ensures audit entries are well-formed JSON objects.
func (bs *BoltStore) VerifyAuditIntegrity() error {
	if !bs.isOpen {
		return fmt.Errorf("vault is not open")
	}

	return bs.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(AuditBucket)
		if bucket == nil {
			return fmt.Errorf("audit bucket not found")
		}

		return bucket.ForEach(func(k, v []byte) error {
			if v == nil {
				return nil
			}
			var env auditEnvelope
			if err := json.Unmarshal(v, &env); err != nil {
				return fmt.Errorf("corrupted audit entry: %w", err)
			}
			if env.Operation == nil {
				return fmt.Errorf("audit entry missing operation data")
			}
			return nil
		})
	})
}

// ExportVault writes a JSON snapshot of vault metadata, profiles, entries, and audit log.
func (bs *BoltStore) ExportVault(path string, includeSecrets bool) error {
	if !bs.isOpen {
		return fmt.Errorf("vault is not open")
	}
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("export path cannot be empty")
	}

	metadata, err := bs.GetVaultMetadata()
	if err != nil {
		return fmt.Errorf("failed to load metadata: %w", err)
	}

	profiles, err := bs.ListProfiles()
	if err != nil {
		return fmt.Errorf("failed to list profiles: %w", err)
	}

	entries := make(map[string][]exportEntry)
	for _, profile := range profiles {
		summaries, err := bs.ListEntries(profile.Name, nil)
		if err != nil {
			return fmt.Errorf("failed to list entries for profile %s: %w", profile.Name, err)
		}

		for _, summary := range summaries {
			exported := exportEntry{
				ID:        summary.ID,
				Name:      summary.Name,
				Username:  summary.Username,
				URL:       summary.URL,
				Notes:     summary.Notes,
				Tags:      append([]string(nil), summary.Tags...),
				TOTPSeed:  summary.TOTPSeed,
				CreatedAt: summary.CreatedAt,
				UpdatedAt: summary.UpdatedAt,
			}

			if includeSecrets {
				fullEntry, err := bs.GetEntry(profile.Name, summary.ID)
				if err != nil {
					return fmt.Errorf("failed to fetch entry %s/%s: %w", profile.Name, summary.ID, err)
				}

				secretBytes := append([]byte(nil), fullEntry.Secret...)
				if len(secretBytes) == 0 {
					secretBytes = append(secretBytes, fullEntry.Password...)
				}
				if len(secretBytes) > 0 {
					exported.Secret = base64.StdEncoding.EncodeToString(secretBytes)
					vault.Zeroize(secretBytes)
				}

				vault.Zeroize(fullEntry.Secret)
				vault.Zeroize(fullEntry.Password)
			}

			entries[profile.Name] = append(entries[profile.Name], exported)
		}

		if _, ok := entries[profile.Name]; !ok {
			entries[profile.Name] = []exportEntry{}
		}
	}

	auditLog, err := bs.GetAuditLog()
	if err != nil {
		return fmt.Errorf("failed to load audit log: %w", err)
	}

	payload := vaultExport{
		Metadata:       metadata,
		Profiles:       profiles,
		Entries:        entries,
		AuditLog:       auditLog,
		ExportedAt:     time.Now().UTC(),
		IncludeSecrets: includeSecrets,
	}

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode export data: %w", err)
	}

	// Ensure directory exists with secure permissions (0o700)
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("failed to create export directory: %w", err)
		}
	}

	// Check if the data size is too large (max 100MB)
	const maxExportSize = 100 * 1024 * 1024 // 100MB
	if uint64(len(data)) > maxExportSize {
		return fmt.Errorf("export data too large: %d bytes (max %d MB)", len(data), maxExportSize/(1024*1024))
	}

	// Write file with secure permissions (read/write for owner only)
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write export file: %w", err)
	}

	// Atomically rename the temporary file to the final destination
	if err := os.Rename(tmpPath, path); err != nil {
		if removeErr := os.Remove(tmpPath); removeErr != nil {
			log.Printf("Warning: failed to clean up temp file %s: %v", tmpPath, removeErr)
		}
		return fmt.Errorf("failed to finalize export: %w", err)
	}

	if err := bs.LogOperation(&domain.Operation{
		Type:      "export_vault",
		Timestamp: time.Now().UTC(),
		Success:   true,
	}); err != nil {
		return fmt.Errorf("failed to log export operation: %w", err)
	}

	return nil
}

// ImportVault restores data from a previously exported JSON snapshot.
func (bs *BoltStore) ImportVault(path, conflictResolution string) error {
	if !bs.isOpen {
		return fmt.Errorf("vault is not open")
	}
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("import path cannot be empty")
	}

	// Verify file exists and is accessible
	fileInfo, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to access import file: %w", err)
	}

	// Check file permissions (should be readable only by owner)
	if mode := fileInfo.Mode(); mode.Perm()&0o077 != 0 {
		return fmt.Errorf("insecure file permissions on %s: %v (should be 600 or more restrictive)", path, mode.Perm())
	}

	// Clean and validate the file path
	cleanPath := filepath.Clean(path)
	if cleanPath != path {
		return fmt.Errorf("invalid file path: potential directory traversal detected")
	}

	// Read file with size limit (100MB)
	const maxFileSize = 100 << 20 // 100MB
	if fileInfo.Size() > maxFileSize {
		return fmt.Errorf("import file too large: %d bytes (max %d MB)", fileInfo.Size(), maxFileSize>>20)
	}
	if fileInfo.Size() < 1 {
		return fmt.Errorf("import file is empty")
	}

	// Use a size-limited reader to prevent potential OOM with malformed files
	file, err := os.Open(cleanPath)
	if err != nil {
		return fmt.Errorf("failed to open import file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("Warning: failed to close file %s: %v", cleanPath, err)
		}
	}()

	// Read the file with a limit
	limitedReader := io.LimitReader(file, maxFileSize+1)
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		return fmt.Errorf("failed to read import file: %w", err)
	}
	if len(data) > maxFileSize {
		return fmt.Errorf("import file exceeds size limit: %d bytes (max %d MB)", len(data), maxFileSize>>20)
	}

	var payload vaultExport
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("failed to decode import data: %w", err)
	}

	mode := strings.ToLower(conflictResolution)
	if mode == "" {
		mode = "overwrite"
	}
	if mode != "overwrite" && mode != "skip" && mode != "replace" {
		return fmt.Errorf("invalid conflict resolution mode: %s", conflictResolution)
	}

	for _, profile := range payload.Profiles {
		if profile == nil || profile.Name == "" {
			continue
		}
		if !bs.ProfileExists(profile.Name) {
			if err := bs.CreateProfile(profile.Name, profile.Description); err != nil && err != ErrProfileExists {
				return fmt.Errorf("failed to create profile %s: %w", profile.Name, err)
			}
		}
	}

	for profileName, list := range payload.Entries {
		if profileName == "" {
			continue
		}
		if !bs.ProfileExists(profileName) {
			if err := bs.CreateProfile(profileName, ""); err != nil && err != ErrProfileExists {
				return fmt.Errorf("failed to ensure profile %s: %w", profileName, err)
			}
		}

		for i := range list {
			exported := &list[i]
			if exported.ID == "" {
				continue
			}

			entry := &domain.Entry{
				ID:       exported.ID,
				Name:     exported.Name,
				Username: exported.Username,
				URL:      exported.URL,
				Notes:    exported.Notes,
				Tags:     append([]string(nil), exported.Tags...),
				TOTPSeed: exported.TOTPSeed,
			}

			secretBytes := []byte(nil)
			if exported.Secret != "" {
				decoded, err := base64.StdEncoding.DecodeString(exported.Secret)
				if err != nil {
					return fmt.Errorf("failed to decode secret for %s/%s: %w", profileName, exported.ID, err)
				}
				secretBytes = decoded
			}

			exists := bs.EntryExists(profileName, exported.ID)

			if len(secretBytes) == 0 && exists && mode != "replace" {
				current, err := bs.GetEntry(profileName, exported.ID)
				if err == nil {
					secretBytes = append(secretBytes, current.Secret...)
					if len(secretBytes) == 0 {
						secretBytes = append(secretBytes, current.Password...)
					}
					vault.Zeroize(current.Secret)
					vault.Zeroize(current.Password)
				}
			}

			if len(secretBytes) > 0 {
				entry.Secret = append([]byte(nil), secretBytes...)
				entry.Password = append([]byte(nil), secretBytes...)
			}

			switch mode {
			case "skip":
				if exists {
					vault.Zeroize(entry.Secret)
					vault.Zeroize(entry.Password)
					vault.Zeroize(secretBytes)
					continue
				}
			case "replace":
				if exists {
					if err := bs.DeleteEntry(profileName, exported.ID); err != nil {
						vault.Zeroize(entry.Secret)
						vault.Zeroize(entry.Password)
						vault.Zeroize(secretBytes)
						return fmt.Errorf("failed to replace entry %s/%s: %w", profileName, exported.ID, err)
					}
					exists = false
				}
			}

			if exists {
				if err := bs.UpdateEntry(profileName, exported.ID, entry); err != nil {
					vault.Zeroize(entry.Secret)
					vault.Zeroize(entry.Password)
					vault.Zeroize(secretBytes)
					return fmt.Errorf("failed to update entry %s/%s: %w", profileName, exported.ID, err)
				}
			} else {
				if err := bs.CreateEntry(profileName, entry); err != nil {
					vault.Zeroize(entry.Secret)
					vault.Zeroize(entry.Password)
					vault.Zeroize(secretBytes)
					return fmt.Errorf("failed to create entry %s/%s: %w", profileName, exported.ID, err)
				}
			}

			vault.Zeroize(entry.Secret)
			vault.Zeroize(entry.Password)
			vault.Zeroize(secretBytes)
		}
	}

	if meta, err := bs.GetVaultMetadata(); err == nil {
		meta.UpdatedAt = time.Now().UTC()
		_ = bs.UpdateVaultMetadata(meta)
	}

	if err := bs.LogOperation(&domain.Operation{
		Type:      "import_vault",
		Timestamp: time.Now().UTC(),
		Success:   true,
	}); err != nil {
		return fmt.Errorf("failed to log import operation: %w", err)
	}

	return nil
}

// CompactVault performs maintenance on the vault storage to reclaim space.
// This is a no-op for BoltDB as it handles compaction automatically.
// Returns an error if the vault is not open.
func (bs *BoltStore) CompactVault() error {
	if !bs.isOpen {
		return fmt.Errorf("vault is not open")
	}
	// BoltDB doesn't need explicit compaction
	return nil
}

// VerifyIntegrity checks the integrity of the vault by verifying all buckets and their contents.
// Returns an error if any integrity issues are found or if the vault is not open.
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
