package store

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// AtomicWriter handles atomic file operations using temp file + rename
type AtomicWriter struct {
	targetPath string
	tempPath   string
	tempFile   *os.File
}

// NewAtomicWriter creates a new atomic writer for the target path
func NewAtomicWriter(targetPath string) (*AtomicWriter, error) {
	dir := filepath.Dir(targetPath)
	base := filepath.Base(targetPath)

	// Clean and validate the target directory path
	cleanDir := filepath.Clean(dir)
	if cleanDir != dir {
		return nil, fmt.Errorf("invalid directory path: potential directory traversal detected")
	}

	// Validate the base filename to prevent directory traversal
	if strings.Contains(base, "..") || strings.ContainsRune(base, filepath.Separator) {
		return nil, fmt.Errorf("invalid filename: %s", base)
	}

	// Create temporary file in the same directory with a random suffix
	tempPath := filepath.Join(cleanDir, fmt.Sprintf(".%s.tmp.%d.%d", base, os.Getpid(), time.Now().UnixNano()))

	// Ensure the target directory exists with secure permissions
	if err := os.MkdirAll(cleanDir, 0o700); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Create temporary file with secure permissions and exclusive creation
	tempFile, err := os.OpenFile(filepath.Clean(tempPath), os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o600)
	if err != nil {
		// Clean up the temp directory if file creation fails
		_ = os.Remove(tempPath) // Best effort cleanup
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}

	// Ensure the file handle is closed if we return an error
	success := false
	defer func() {
		if !success && tempFile != nil {
			_ = tempFile.Close()
			_ = os.Remove(tempPath)
		}
	}()

	// Mark as successful before returning
	success = true

	return &AtomicWriter{
		targetPath: targetPath,
		tempPath:   tempPath,
		tempFile:   tempFile,
	}, nil
}

// Write writes data to the temporary file
func (aw *AtomicWriter) Write(data []byte) (int, error) {
	if aw.tempFile == nil {
		return 0, fmt.Errorf("writer is closed")
	}
	n, err := aw.tempFile.Write(data)
	if err != nil {
		// Try to clean up, but ignore any error from Abort
		if abortErr := aw.Abort(); abortErr != nil {
			log.Printf("Warning: failed to abort after write error: %v", abortErr)
		}
	}
	return n, err
}

// Commit finalizes the write by syncing and atomically renaming
func (aw *AtomicWriter) Commit() error {
	if aw.tempFile == nil {
		return fmt.Errorf("writer is closed")
	}

	// Sync to ensure data is written to disk
	if err := aw.tempFile.Sync(); err != nil {
		// Try to clean up, but ignore any error from Abort
		if abortErr := aw.Abort(); abortErr != nil {
			log.Printf("Warning: failed to abort after sync error: %v", abortErr)
		}
		return fmt.Errorf("failed to sync temp file: %w", err)
	}

	// Close temp file
	if err := aw.tempFile.Close(); err != nil {
		// Try to clean up, but ignore any error from Abort
		if abortErr := aw.Abort(); abortErr != nil {
			log.Printf("Warning: failed to abort after close error: %v", abortErr)
		}
		return fmt.Errorf("failed to close temp file: %w", err)
	}
	aw.tempFile = nil

	// Atomically rename temp file to target
	if err := os.Rename(aw.tempPath, aw.targetPath); err != nil {
		_ = os.Remove(aw.tempPath) // Clean up on failure, ignore error
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// Abort cancels the write and cleans up the temporary file
func (aw *AtomicWriter) Abort() error {
	var err error

	if aw.tempFile != nil {
		if closeErr := aw.tempFile.Close(); closeErr != nil {
			err = closeErr
		}
		aw.tempFile = nil
	}

	if removeErr := os.Remove(aw.tempPath); removeErr != nil && err == nil {
		err = removeErr
	}

	return err
}

// AtomicWriteFile writes data to a file atomically
func AtomicWriteFile(path string, data []byte) error {
	writer, err := NewAtomicWriter(path)
	if err != nil {
		return err
	}

	if _, err := writer.Write(data); err != nil {
		abortErr := writer.Abort()
		if abortErr != nil {
			log.Printf("Warning: failed to abort atomic writer: %v", abortErr)
		}
		return err
	}

	return writer.Commit()
}

// EnsureFilePermissions ensures the file has secure permissions (0600)
func EnsureFilePermissions(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	// Check if permissions are too permissive
	mode := info.Mode().Perm()
	if mode&0o077 != 0 {
		// Fix permissions to 0600 (owner read/write only)
		return os.Chmod(path, 0o600)
	}

	return nil
}
