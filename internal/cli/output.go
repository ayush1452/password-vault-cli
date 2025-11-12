package cli

import (
	"fmt"
	"io"
	"log"
)

// writeOutput is a helper function to write output with error checking
func writeOutput(w io.Writer, format string, args ...interface{}) error {
	_, err := fmt.Fprintf(w, format, args...)
	if err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}
	return nil
}

// checkDeferredErr checks and logs errors from deferred function calls.
// It's designed to be used with named return values in functions.
// Example: defer checkDeferredErr(&err, "CloseSessionStore", CloseSessionStore())
func checkDeferredErr(err *error, op string, cerr error) {
	if cerr != nil {
		log.Printf("Warning: error in deferred %s: %v", op, cerr)
		if *err == nil {
			*err = fmt.Errorf("%s: %w", op, cerr)
		}
	}
}
