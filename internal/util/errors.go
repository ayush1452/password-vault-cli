// Package util provides utility functions and helpers used throughout the password vault.
// It includes common operations, error handling utilities, and other shared functionality.
package util

import (
	"fmt"
	"os"
)

// Exit codes as defined in the architecture
const (
	ExitOK           = 0
	ExitError        = 1
	ExitInvalidInput = 2
	ExitVaultLocked  = 3
	ExitIntegrityErr = 4
)

// ExitWithCode exits the program with the specified code and message
func ExitWithCode(code int, format string, args ...interface{}) {
	if format != "" {
		fmt.Fprintf(os.Stderr, format+"\n", args...)
	}
	os.Exit(code)
}

// HandleError handles errors and exits with appropriate code
func HandleError(err error, context string) {
	if err == nil {
		return
	}

	// Map specific errors to exit codes
	switch err.Error() {
	case "vault is locked by another process":
		ExitWithCode(ExitVaultLocked, "Error: %s - %v", context, err)
	case "vault data is corrupted", "integrity error":
		ExitWithCode(ExitIntegrityErr, "Error: %s - %v\nRun 'vault doctor' to diagnose issues.", context, err)
	default:
		if context != "" {
			ExitWithCode(ExitError, "Error: %s - %v", context, err)
		} else {
			ExitWithCode(ExitError, "Error: %v", err)
		}
	}
}

// WrapError wraps an error with additional context
func WrapError(err error, context string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", context, err)
}
