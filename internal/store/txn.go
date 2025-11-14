package store

import (
	"fmt"
	"os"
	"path/filepath"
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

	// Create temporary file in the same directory
	tempPath := filepath.Join(dir, fmt.Sprintf(".%s.tmp.%d", base, os.Getpid()))

	// Create directory if it doesn't exist
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Create temporary file with secure permissions
	tempFile, err := os.OpenFile(tempPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o600)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}

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
		_ = aw.Abort() // Clean up on write error
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
		_ = aw.Abort() // Clean up on sync error
		return fmt.Errorf("failed to sync temp file: %w", err)
	}

	// Close temp file
	if err := aw.tempFile.Close(); err != nil {
		_ = aw.Abort() // Clean up on close error
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
		writer.Abort()
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
