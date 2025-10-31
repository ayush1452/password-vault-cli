package cli

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"time"

	"github.com/vault-cli/vault/internal/domain"
	"github.com/vault-cli/vault/internal/store"
	"github.com/vault-cli/vault/internal/vault"
)

// SessionManager handles vault session state
type SessionManager struct {
	vaultPath  string
	vaultStore store.VaultStore
	masterKey  []byte
	unlockTime time.Time
	ttl        time.Duration
}

type sessionFileData struct {
	VaultPath         string    `json:"vault_path"`
	UnlockTime        time.Time `json:"unlock_time"`
	TTLSeconds        int64     `json:"ttl_seconds"`
	MasterKeyEnvelope []byte    `json:"master_key_envelope"`
}

var sessionManager *SessionManager

// IsUnlocked returns true if the vault is currently unlocked
func IsUnlocked() bool {
	ensureSessionRestored()

	if sessionManager == nil {
		return false
	}

	// Check if session has expired
	if time.Since(sessionManager.unlockTime) > sessionManager.ttl {
		// Session expired, lock vault
		LockVault()
		return false
	}

	return sessionManager.masterKey != nil
}

// GetVaultStore returns the current vault store if unlocked
func GetVaultStore() store.VaultStore {
	ensureSessionRestored()

	if !IsUnlocked() {
		return nil
	}

	if sessionManager.vaultStore == nil || !sessionManager.vaultStore.IsOpen() {
		vaultStore := store.NewBoltStore()
		if err := vaultStore.OpenVault(sessionManager.vaultPath, sessionManager.masterKey); err != nil {
			fmt.Printf("Warning: failed to open vault store from session: %v\n", err)
			ClearPersistedSession()
			return nil
		}
		sessionManager.vaultStore = vaultStore
	}

	return sessionManager.vaultStore
}

// UnlockVault unlocks the vault with the given passphrase
func UnlockVault(vaultPath, passphrase string, ttl time.Duration) error {
	// Check if vault exists
	if _, err := os.Stat(vaultPath); os.IsNotExist(err) {
		return fmt.Errorf("vault not found at %s", vaultPath)
	}

	// Create store
	vaultStore := store.NewBoltStore()

	// Open the vault temporarily to load metadata
	if err := vaultStore.OpenVault(vaultPath, make([]byte, 32)); err != nil {
		return fmt.Errorf("failed to open vault for metadata: %w", err)
	}

	metadata, err := vaultStore.GetVaultMetadata()
	if err != nil {
		vaultStore.CloseVault()
		return fmt.Errorf("failed to load vault metadata: %w", err)
	}

	// Close the temporary store opened with a dummy key
	vaultStore.CloseVault()

	// Recreate store for proper opening
	vaultStore = store.NewBoltStore()

	params, salt, err := metadataToKeyParams(metadata)
	if err != nil {
		return fmt.Errorf("invalid vault metadata: %w", err)
	}
	crypto := vault.NewCryptoEngine(params)
	masterKey, err := crypto.DeriveKey(passphrase, salt)
	if err != nil {
		return fmt.Errorf("failed to derive master key: %w", err)
	}

	if err := vaultStore.OpenVault(vaultPath, masterKey); err != nil {
		vault.Zeroize(masterKey)
		return fmt.Errorf("failed to open vault: %w", err)
	}

	sessionManager = &SessionManager{
		vaultPath:  vaultPath,
		vaultStore: vaultStore,
		masterKey:  masterKey,
		unlockTime: time.Now(),
		ttl:        ttl,
	}

	if err := persistSession(); err != nil {
		vaultStore.CloseVault()
		vault.Zeroize(masterKey)
		sessionManager = nil
		return fmt.Errorf("failed to persist session: %w", err)
	}

	CloseSessionStore()

	return nil
}

// LockVault locks the vault and clears the session
func LockVault() error {
	if sessionManager == nil {
		return nil
	}

	sessionFile := sessionFilePath(sessionManager.vaultPath)

	var err error
	if closeErr := CloseSessionStore(); closeErr != nil {
		err = closeErr
	}

	if sessionManager.masterKey != nil {
		vault.Zeroize(sessionManager.masterKey)
	}

	sessionManager = nil
	os.Remove(sessionFile)
	return err
}

// RefreshSession updates the session unlock time
func RefreshSession() {
	if sessionManager != nil {
		sessionManager.unlockTime = time.Now()
		if err := persistSession(); err != nil {
			fmt.Printf("Warning: failed to persist session: %v\n", err)
		}
	}
}

// GetSessionInfo returns information about the current session
func GetSessionInfo() (bool, time.Duration, error) {
	ensureSessionRestored()

	if sessionManager == nil {
		return false, 0, nil
	}

	elapsed := time.Since(sessionManager.unlockTime)
	remaining := sessionManager.ttl - elapsed

	if remaining <= 0 {
		return false, 0, nil
	}

	return true, remaining, nil
}

// RemainingSessionTTL returns the remaining TTL without triggering unlock side effects.
func RemainingSessionTTL() time.Duration {
	ensureSessionRestored()

	if sessionManager == nil || sessionManager.ttl <= 0 {
		return 0
	}

	remaining := sessionManager.ttl - time.Since(sessionManager.unlockTime)
	if remaining < 0 {
		return 0
	}

	return remaining
}

// EnsureVaultDirectory creates the vault directory if it doesn't exist
func EnsureVaultDirectory(vaultPath string) error {
	dir := filepath.Dir(vaultPath)
	return os.MkdirAll(dir, 0700)
}

func persistSession() error {
	if sessionManager == nil {
		return nil
	}

	if sessionManager.ttl <= 0 {
		return nil
	}

	path := sessionFilePath(sessionManager.vaultPath)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	crypto := vault.NewDefaultCryptoEngine()
	sessionPassphrase := deriveSessionPassphrase(sessionManager.vaultPath)
	envelope, err := crypto.SealWithPassphrase(sessionManager.masterKey, sessionPassphrase)
	if err != nil {
		return err
	}

	data := sessionFileData{
		VaultPath:         sessionManager.vaultPath,
		UnlockTime:        sessionManager.unlockTime,
		TTLSeconds:        int64(sessionManager.ttl / time.Second),
		MasterKeyEnvelope: vault.EnvelopeToBytes(envelope),
	}

	serialized, err := json.Marshal(data)
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, serialized, 0600); err != nil {
		return err
	}

	return nil
}

func ensureSessionRestored() {
	if sessionManager != nil {
		return
	}

	if vaultPath == "" {
		return
	}

	path := sessionFilePath(vaultPath)
	data, err := loadSessionFile(path)
	if err != nil {
		return
	}
	if data == nil {
		return
	}

	ttl := time.Duration(data.TTLSeconds) * time.Second
	if ttl <= 0 {
		os.Remove(path)
		return
	}

	if time.Since(data.UnlockTime) > ttl {
		os.Remove(path)
		return
	}

	envelope, err := vault.EnvelopeFromBytes(data.MasterKeyEnvelope)
	if err != nil {
		os.Remove(path)
		return
	}

	crypto := vault.NewDefaultCryptoEngine()
	sessionPassphrase := deriveSessionPassphrase(data.VaultPath)
	masterKey, err := crypto.OpenWithPassphrase(envelope, sessionPassphrase)
	if err != nil {
		os.Remove(path)
		return
	}

	sessionManager = &SessionManager{
		vaultPath:  data.VaultPath,
		vaultStore: nil,
		masterKey:  masterKey,
		unlockTime: data.UnlockTime,
		ttl:        ttl,
	}

	// Verify the master key by opening and immediately closing the vault.
	if store := GetVaultStore(); store != nil {
		if err := CloseSessionStore(); err != nil {
			fmt.Printf("Warning: failed to release vault store after restore: %v\n", err)
		}
	}
}

func loadSessionFile(path string) (*sessionFileData, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var data sessionFileData
	if err := json.Unmarshal(content, &data); err != nil {
		return nil, err
	}

	return &data, nil
}

func sessionFilePath(vaultPath string) string {
	if vaultPath == "" {
		return ""
	}
	return vaultPath + ".session"
}

func deriveSessionPassphrase(vaultPath string) string {
	username := "unknown"
	if currentUser, err := user.Current(); err == nil && currentUser != nil {
		username = currentUser.Username
	}

	hostname := "localhost"
	if host, err := os.Hostname(); err == nil {
		hostname = host
	}

	return fmt.Sprintf("vault-session:%s:%s:%s", hostname, username, vaultPath)
}

// CloseSessionStore closes the active vault store without clearing the session.
func CloseSessionStore() error {
	if sessionManager == nil {
		return nil
	}

	if sessionManager.vaultStore == nil {
		return nil
	}

	if !sessionManager.vaultStore.IsOpen() {
		sessionManager.vaultStore = nil
		return nil
	}

	if err := sessionManager.vaultStore.CloseVault(); err != nil {
		return err
	}

	sessionManager.vaultStore = nil
	return nil
}

// ClearPersistedSession removes the session file and resets in-memory state.
func ClearPersistedSession() {
	if sessionManager == nil {
		return
	}

	path := sessionFilePath(sessionManager.vaultPath)
	if path != "" {
		os.Remove(path)
	}

	if err := CloseSessionStore(); err != nil {
		fmt.Printf("Warning: failed to close vault store while clearing session: %v\n", err)
	}

	if sessionManager.masterKey != nil {
		vault.Zeroize(sessionManager.masterKey)
	}

	sessionManager = nil
}

func metadataToKeyParams(metadata *domain.VaultMetadata) (vault.Argon2Params, []byte, error) {
	if metadata == nil {
		return vault.Argon2Params{}, nil, fmt.Errorf("vault metadata missing")
	}

	params := vault.Argon2Params{}
	var ok bool

	memoryVal, ok := metadata.KDFParams["memory"].(float64)
	if !ok {
		return vault.Argon2Params{}, nil, fmt.Errorf("missing or invalid memory parameter")
	}
	params.Memory = uint32(memoryVal)

	iterationsVal, ok := metadata.KDFParams["iterations"].(float64)
	if !ok {
		return vault.Argon2Params{}, nil, fmt.Errorf("missing or invalid iterations parameter")
	}
	params.Iterations = uint32(iterationsVal)

	parallelVal, ok := metadata.KDFParams["parallelism"].(float64)
	if !ok {
		return vault.Argon2Params{}, nil, fmt.Errorf("missing or invalid parallelism parameter")
	}
	params.Parallelism = uint8(parallelVal)

	saltBase64, ok := metadata.KDFParams["salt"].(string)
	if !ok {
		return vault.Argon2Params{}, nil, fmt.Errorf("missing or invalid salt")
	}

	saltBytes, err := base64.StdEncoding.DecodeString(saltBase64)
	if err != nil {
		return vault.Argon2Params{}, nil, fmt.Errorf("failed to decode salt: %w", err)
	}

	return params, saltBytes, nil
}
