package cli

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
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

// logWarning logs a warning message with proper error handling
func logWarning(format string, args ...interface{}) {
	if err := log.Output(2, fmt.Sprintf("WARNING: "+format, args...)); err != nil {
		// If we can't log, there's not much we can do, so we'll just print to stderr
		fmt.Fprintf(os.Stderr, "WARNING: Failed to log warning: %v\n", err)
	}
}

// IsUnlocked returns true if the vault is currently unlocked
func IsUnlocked() bool {
	ensureSessionRestored()

	if sessionManager == nil {
		return false
	}

	// Check if session has expired
	if time.Since(sessionManager.unlockTime) > sessionManager.ttl {
		// Session expired, lock vault
		if err := LockVault(); err != nil {
			logWarning("Failed to lock vault: %v", err)
		}
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
			logWarning("Error opening vault store: %v", err)
			// Try to clear the session if we can't open the vault
			if clearErr := LockVault(); clearErr != nil {
				logWarning("Failed to clear session after open error: %v", clearErr)
			}
			return nil
		}
		sessionManager.vaultStore = vaultStore
	}

	return sessionManager.vaultStore
}

// UnlockVault unlocks the vault with the given passphrase
func UnlockVault(vaultPath, passphrase string, ttl time.Duration) error {
	// Validate input parameters
	if vaultPath == "" {
		return errors.New("vault path cannot be empty")
	}
	if passphrase == "" {
		return errors.New("passphrase cannot be empty")
	}
	if ttl <= 0 {
		return fmt.Errorf("invalid TTL duration: %v", ttl)
	}

	// Check if vault exists
	if _, err := os.Stat(vaultPath); os.IsNotExist(err) {
		return fmt.Errorf("vault not found at %s: %w", vaultPath, err)
	}

	log.Printf("Attempting to unlock vault at: %s", vaultPath)

	// Create and open a temporary store to load metadata
	tempStore := store.NewBoltStore()
	defer func() {
		if tempStore != nil {
			if closeErr := tempStore.CloseVault(); closeErr != nil {
				logWarning("Failed to close temporary vault store: %v", closeErr)
			}
		}
	}()

	// Use a dummy key for metadata access
	dummyKey := make([]byte, 32)
	if err := tempStore.OpenVault(vaultPath, dummyKey); err != nil {
		return fmt.Errorf("failed to open vault for metadata: %w", err)
	}

	metadata, err := tempStore.GetVaultMetadata()
	if err != nil {
		return fmt.Errorf("failed to load vault metadata: %w", err)
	}

	// Close the temporary store
	if err := tempStore.CloseVault(); err != nil {
		log.Printf("Warning: failed to close temporary vault store: %v", err)
	}
	tempStore = nil

	// Derive the actual master key
	params, salt, err := metadataToKeyParams(metadata)
	if err != nil {
		return fmt.Errorf("invalid vault metadata: %w", err)
	}

	crypto := vault.NewCryptoEngine(params)
	masterKey, err := crypto.DeriveKey(passphrase, salt)
	if err != nil {
		return fmt.Errorf("failed to derive master key: %w", err)
	}
	defer vault.Zeroize(masterKey)

	// Create and open the actual store with the derived key
	vaultStore := store.NewBoltStore()
	if err := vaultStore.OpenVault(vaultPath, masterKey); err != nil {
		if _, err := fmt.Fprintf(os.Stderr, "Failed to open vault with derived key: %v\n", err); err != nil {
			// If we can't even print to stderr, we're in a bad state
			panic(err)
		}
		return fmt.Errorf("failed to open vault with derived key: %w", err)
	}

	// Create new session
	sessionManager = &SessionManager{
		vaultPath:  vaultPath,
		vaultStore: vaultStore,
		masterKey:  masterKey,
		unlockTime: time.Now(),
		ttl:        ttl,
	}

	// Persist the session
	if err := persistSession(); err != nil {
		if closeErr := vaultStore.CloseVault(); closeErr != nil {
			log.Printf("Failed to close vault after session persistence error: %v", closeErr)
		}
		sessionManager = nil
		return fmt.Errorf("failed to persist session: %w", err)
	}

	// Close the store as we'll open it on demand
	if err := CloseSessionStore(); err != nil {
		logWarning("Failed to close session store after unlock: %v", err)
	}

	log.Printf("Successfully unlocked vault at: %s", vaultPath)
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
	if removeErr := os.Remove(sessionFile); removeErr != nil && !os.IsNotExist(removeErr) {
		// If there's an error removing the file and it's not because it doesn't exist,
		// combine it with any existing error
		if err == nil {
			err = fmt.Errorf("failed to remove session file: %w", removeErr)
		} else {
			err = fmt.Errorf("%w; failed to remove session file: %v", err, removeErr)
		}
	}
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
	return os.MkdirAll(dir, 0o700)
}

// persistSession saves the current session state to disk in a secure manner.
// It creates a session file containing an encrypted version of the master key.
// The function ensures proper file permissions and atomic write operations.
func persistSession() error {
	// Early return if there's no active session or TTL is invalid
	if sessionManager == nil {
		log.Print("No active session to persist")
		return nil
	}

	if sessionManager.ttl <= 0 {
		log.Print("Skipping session persistence: invalid TTL")
		return nil
	}

	path := sessionFilePath(sessionManager.vaultPath)
	log.Printf("Persisting session to: %s", path)

	// Ensure the directory exists with secure permissions
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("failed to create session directory: %w", err)
	}

	// Create a temporary file for atomic write
	tempFile, err := os.CreateTemp(dir, ".vault-session-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary session file: %w", err)
	}
	tempPath := tempFile.Name()
	defer func() {
		// Check for errors when closing the temp file
		if closeErr := tempFile.Close(); closeErr != nil {
			log.Printf("Warning: failed to close temporary file %s: %v", tempPath, closeErr)
		}
		// Only try to remove if the file still exists
		if _, err := os.Stat(tempPath); err == nil {
			if removeErr := os.Remove(tempPath); removeErr != nil {
				log.Printf("Warning: failed to remove temporary file %s: %v", tempPath, removeErr)
			}
		}
	}()

	// Encrypt the master key
	crypto := vault.NewDefaultCryptoEngine()
	sessionPassphrase := deriveSessionPassphrase(sessionManager.vaultPath)
	envelope, err := crypto.SealWithPassphrase(sessionManager.masterKey, sessionPassphrase)
	if err != nil {
		return fmt.Errorf("failed to encrypt master key: %w", err)
	}

	// Prepare session data
	envelopeBytes, err := vault.EnvelopeToBytes(envelope)
	if err != nil {
		return fmt.Errorf("failed to serialize master key envelope: %w", err)
	}

	data := sessionFileData{
		VaultPath:         sessionManager.vaultPath,
		UnlockTime:        sessionManager.unlockTime,
		TTLSeconds:        int64(sessionManager.ttl / time.Second),
		MasterKeyEnvelope: envelopeBytes,
	}

	// Serialize and write to temporary file
	serialized, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to serialize session data: %w", err)
	}

	// Write with sync to ensure data is flushed to disk
	if _, err := tempFile.Write(serialized); err != nil {
		return fmt.Errorf("failed to write session data: %w", err)
	}

	// Ensure data is written to disk
	if err := tempFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync session data to disk: %w", err)
	}

	// Close the file before renaming (required on Windows)
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file: %w", err)
	}

	// Atomic rename to final location
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("failed to commit session file: %w", err)
	}

	// Set restrictive permissions on the final file
	if err := os.Chmod(path, 0o600); err != nil {
		log.Printf("Warning: failed to set permissions on session file: %v", err)
		// Not a fatal error, but log it
	}

	log.Printf("Successfully persisted session to: %s", path)
	return nil
}

func ensureSessionRestored() {
	if sessionManager != nil {
		return
	}

	if vaultPath == "" {
		log.Printf("No vault path set, skipping session restore")
		return
	}

	path := sessionFilePath(vaultPath)
	data, err := loadSessionFile(path)
	if err != nil {
		log.Printf("Failed to load session file %s: %v", path, err)
		return
	}
	if data == nil {
		log.Printf("No session data found in %s", path)
		return
	}

	ttl := time.Duration(data.TTLSeconds) * time.Second
	if ttl <= 0 {
		log.Printf("Invalid TTL in session file, removing: %s", path)
		if err := os.Remove(path); err != nil {
			log.Printf("Failed to remove invalid session file: %v", err)
		}
		return
	}

	if time.Since(data.UnlockTime) > ttl {
		log.Printf("Session expired, removing session file: %s", path)
		if err := os.Remove(path); err != nil {
			log.Printf("Failed to remove expired session file: %v", err)
		}
		return
	}

	envelope, err := vault.EnvelopeFromBytes(data.MasterKeyEnvelope)
	if err != nil {
		log.Printf("Failed to parse master key envelope: %v", err)
		if err := os.Remove(path); err != nil {
			log.Printf("Failed to remove invalid session file: %v", err)
		}
		return
	}

	crypto := vault.NewDefaultCryptoEngine()
	sessionPassphrase := deriveSessionPassphrase(data.VaultPath)
	masterKey, err := crypto.OpenWithPassphrase(envelope, sessionPassphrase)
	if err != nil {
		log.Printf("Failed to decrypt master key: %v", err)
		if err := os.Remove(path); err != nil {
			log.Printf("Failed to remove unreadable session file: %v", err)
		}
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
	if vaultStore := GetVaultStore(); vaultStore != nil {
		if err := CloseSessionStore(); err != nil {
			log.Printf("Failed to release vault store after restore: %v", err)
		}
	}
}

func loadSessionFile(path string) (*sessionFileData, error) {
	// Clean the file path to prevent directory traversal
	cleanPath := filepath.Clean(path)

	// Optional: Add additional path validation here if needed
	// For example, ensure the path is within an allowed directory

	content, err := os.ReadFile(cleanPath)
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
		if removeErr := os.Remove(path); removeErr != nil && !os.IsNotExist(removeErr) {
			log.Printf("Warning: failed to remove session file %s: %v", path, removeErr)
		}
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
