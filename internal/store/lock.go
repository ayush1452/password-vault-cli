package store

import (
	"errors"
	"os"
	"path/filepath"
	"time"
)

// Error variables for file locking operations
var (
	// ErrLockTimeout is returned when a lock cannot be acquired within the specified timeout
	ErrLockTimeout = errors.New("lock acquisition timeout")
	// ErrLockExists is returned when a lock file already exists
	ErrLockExists = errors.New("lock file exists")
	// ErrLockNotHeld is returned when attempting to release a lock that isn't held
	ErrLockNotHeld = errors.New("lock not held")
)

// FileLock represents a file lock
type FileLock struct {
	path     string
	lockFile *os.File
	locked   bool
}

// NewFileLock creates a new file lock for the given path
func NewFileLock(vaultPath string) *FileLock {
	lockPath := vaultPath + ".lock"
	return &FileLock{
		path:   lockPath,
		locked: false,
	}
}

// Lock acquires the file lock with a timeout
func (fl *FileLock) Lock(timeout time.Duration) error {
	if fl.locked {
		return errors.New("lock already held")
	}

	// Create lock file directory if it doesn't exist
	dir := filepath.Dir(fl.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	// Try to acquire lock with timeout
	start := time.Now()
	for {
		// Try to create exclusive lock file
		file, err := os.OpenFile(fl.path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			// Successfully created lock file
			fl.lockFile = file
			fl.locked = true

			// Write process ID to lock file
			if _, err := file.WriteString(string(rune(os.Getpid()))); err != nil {
				fl.Unlock() // Clean up on error
				return err
			}

			return platformLock(file)
		}

		// Check if timeout exceeded
		if time.Since(start) > timeout {
			return ErrLockTimeout
		}

		// Check if lock file is stale
		if fl.isLockStale() {
			// Remove stale lock file and try again
			os.Remove(fl.path)
			continue
		}

		// Wait before retrying
		time.Sleep(100 * time.Millisecond)
	}
}

// Unlock releases the file lock
func (fl *FileLock) Unlock() error {
	if !fl.locked {
		return ErrLockNotHeld
	}

	var err error
	if fl.lockFile != nil {
		// Release platform-specific lock
		if unlockErr := platformUnlock(fl.lockFile); unlockErr != nil {
			err = unlockErr
		}

		// Close file
		if closeErr := fl.lockFile.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
		fl.lockFile = nil
	}

	// Remove lock file
	if removeErr := os.Remove(fl.path); removeErr != nil && err == nil {
		err = removeErr
	}

	fl.locked = false
	return err
}

// IsLocked returns true if the lock is currently held
func (fl *FileLock) IsLocked() bool {
	return fl.locked
}

// isLockStale checks if the lock file is from a dead process
func (fl *FileLock) isLockStale() bool {
	// Check if lock file exists
	info, err := os.Stat(fl.path)
	if err != nil {
		return false
	}

	// Consider lock stale if it's older than 5 minutes
	// This is a simple heuristic; in production, you might want to
	// check if the process is actually running
	return time.Since(info.ModTime()) > 5*time.Minute
}
