package store

import (
	"errors"

	"github.com/vault-cli/vault/internal/domain"
)

// Error variables for vault store operations
var (
	// ErrVaultNotFound is returned when the specified vault does not exist
	ErrVaultNotFound = errors.New("vault not found")
	// ErrVaultExists is returned when attempting to create a vault that already exists
	ErrVaultExists = errors.New("vault already exists")
	// ErrEntryNotFound is returned when the specified entry does not exist
	ErrEntryNotFound = errors.New("entry not found")
	// ErrEntryExists is returned when attempting to create an entry that already exists
	ErrEntryExists = errors.New("entry already exists")
	// ErrProfileNotFound is returned when the specified profile does not exist
	ErrProfileNotFound = errors.New("profile not found")
	// ErrProfileExists is returned when attempting to create a profile that already exists
	ErrProfileExists = errors.New("profile already exists")
	// ErrVaultLocked is returned when the vault is locked by another process
	ErrVaultLocked = errors.New("vault is locked by another process")
	// ErrVaultCorrupted is returned when the vault data is corrupted or invalid
	ErrVaultCorrupted = errors.New("vault data is corrupted")
	// ErrInvalidKey is returned when the provided master key is invalid
	ErrInvalidKey = errors.New("invalid master key")
	// ErrTransactionFailed is returned when a database transaction fails
	ErrTransactionFailed = errors.New("transaction failed")
)

// VaultStore defines the interface for vault storage operations
type VaultStore interface {
	// Vault lifecycle
	CreateVault(path string, masterKey []byte, kdfParams map[string]interface{}) error
	OpenVault(path string, masterKey []byte) error
	CloseVault() error
	IsOpen() bool

	// Entry operations
	CreateEntry(profile string, entry *domain.Entry) error
	GetEntry(profile, entryID string) (*domain.Entry, error)
	ListEntries(profile string, filter *domain.Filter) ([]*domain.Entry, error)
	UpdateEntry(profile, entryID string, entry *domain.Entry) error
	DeleteEntry(profile, entryID string) error
	EntryExists(profile, entryID string) bool

	// Profile operations
	CreateProfile(name, description string) error
	GetProfile(name string) (*domain.Profile, error)
	ListProfiles() ([]*domain.Profile, error)
	DeleteProfile(name string) error
	ProfileExists(name string) bool

	// Metadata operations
	GetVaultMetadata() (*domain.VaultMetadata, error)
	UpdateVaultMetadata(metadata *domain.VaultMetadata) error

	// Audit operations
	LogOperation(op *domain.Operation) error
	GetAuditLog() ([]*domain.Operation, error)
	VerifyAuditIntegrity() error

	// Backup/restore operations
	ExportVault(path string, includeSecrets bool) error
	ImportVault(path, conflictResolution string) error

	// Maintenance operations
	CompactVault() error
	VerifyIntegrity() error

	// Security operations
	RotateMasterKey(newPassphrase string) error
}
