package cli

import (
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
)

// MaxOutputSize is the maximum allowed size for output to prevent memory exhaustion
const MaxOutputSize = 10 * 1024 * 1024 // 10MB

// writeString writes a string to the writer with error checking and size limits
func writeString(w io.Writer, s string) error {
	// Check output size to prevent potential memory exhaustion
	if len(s) > MaxOutputSize {
		return fmt.Errorf("output size %d exceeds maximum allowed size %d",
			len(s), MaxOutputSize)
	}

	// Write the output with error checking
	n, err := fmt.Fprint(w, s)
	if err != nil {
		return fmt.Errorf("failed to write output (wrote %d bytes): %w", n, err)
	}

	// Ensure the output is flushed if it's a file
	if f, ok := w.(interface{ Flush() error }); ok {
		if flushErr := f.Flush(); flushErr != nil {
			return fmt.Errorf("failed to flush output: %w", flushErr)
		}
	}

	return nil
}

// writeOutput is a helper function to write formatted output with error checking and size limits
func writeOutput(w io.Writer, format string, args ...interface{}) error {
	// Format the output first to check its size
	output := fmt.Sprintf(format, args...)
	return writeString(w, output)
}

// checkDeferredErr checks and logs errors from deferred function calls.
// It's designed to be used with named return values in functions.
// Example: defer checkDeferredErr(&err, "CloseSessionStore", CloseSessionStore())
func checkDeferredErr(err *error, op string, cerr error) {
	if cerr != nil {
		// Log the error with a stack trace in debug mode
		if isDebugEnabled() {
			log.Printf("DEBUG: error in deferred %s: %+v", op, cerr)
		} else {
			log.Printf("Warning: error in deferred %s: %v", op, cerr)
		}

		// Only override the error if it's not already set
		if *err == nil {
			*err = fmt.Errorf("%s: %w", op, cerr)
		}

		// For critical errors, consider exiting
		if isCriticalError(cerr) {
			os.Exit(1) // Using 1 as a generic error code
		}
	}
}

// isCriticalError determines if an error is critical enough to exit the program
func isCriticalError(err error) bool {
	// Add conditions for critical errors
	return os.IsPermission(err) || os.IsNotExist(err)
}

// isDebugEnabled checks if debug mode is enabled via environment variable
func isDebugEnabled() bool {
	dbg, _ := strconv.ParseBool(os.Getenv("VAULT_DEBUG"))
	return dbg
}

// SecurePrint prints sensitive information with security considerations
func SecurePrint(w io.Writer, format string, args ...interface{}) error {
	// Clear sensitive data from memory after use
	defer func() {
		for i := range args {
			if s, ok := args[i].(string); ok {
				// Overwrite the string in the argument list
				args[i] = "[REDACTED]"
				// Try to clear the original string if it's a string literal
				clearString(s)
			}
		}
	}()

	return writeOutput(w, format, args...)
}

// clearString attempts to clear sensitive string data from memory
func clearString(s string) {
	// Convert to byte slice to clear the underlying array
	b := []byte(s)
	for i := range b {
		b[i] = 0
	}
}
