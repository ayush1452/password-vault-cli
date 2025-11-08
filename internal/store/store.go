package store

import (
	"errors"

	"github.com/vault-cli/vault/internal/domain"
)

var (
	ErrVaultNotFound     = errors.New("vault not found")
	ErrVaultExists       = errors.New("vault already exists")
	ErrEntryNotFound     = errors.New("entry not found")
	ErrEntryExists       = errors.New("entry already exists")
	ErrProfileNotFound   = errors.New("profile not found")
	ErrProfileExists     = errors.New("profile already exists")
	ErrVaultLocked       = errors.New("vault is locked by another process")
	ErrVaultCorrupted    = errors.New("vault data is corrupted")
	ErrInvalidKey        = errors.New("invalid master key")
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
	GetEntry(profile string, entryID string) (*domain.Entry, error)
	ListEntries(profile string, filter *domain.Filter) ([]*domain.Entry, error)
	UpdateEntry(profile string, entryID string, entry *domain.Entry) error
	DeleteEntry(profile string, entryID string) error
	EntryExists(profile string, entryID string) bool

	// Profile operations
	CreateProfile(name string, description string) error
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
	ImportVault(path string, conflictResolution string) error

	// Maintenance operations
	CompactVault() error
	VerifyIntegrity() error

	// Security operations
	RotateMasterKey(newPassphrase string) error
}
