package store

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ValidateVaultPath validates that a vault path is safe and doesn't contain directory traversal attempts
func ValidateVaultPath(path string) error {
	if path == "" {
		return fmt.Errorf("vault path cannot be empty")
	}

	// Explicitly reject paths with ".." to prevent directory traversal
	// This is a security-critical check
	if strings.Contains(path, "..") {
		return fmt.Errorf("path contains directory traversal: %s", path)
	}

	// Get absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	// Prevent access to system directories
	systemDirs := []string{
		"/etc",
		"/sys",
		"/proc",
		"/dev",
		"/boot",
		"/root",
		"C:\\Windows",
		"C:\\Program Files",
		"/System",
		"/Library/System",
	}

	for _, sysDir := range systemDirs {
		if strings.HasPrefix(absPath, sysDir) {
			return fmt.Errorf("cannot create vault in system directory: %s", sysDir)
		}
	}

	return nil
}
