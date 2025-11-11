package cli

import (
	"fmt"
	"io"
	"os"
)

// writeOutput is a helper function to write output with error checking
func writeOutput(w io.Writer, format string, args ...interface{}) error {
	_, err := fmt.Fprintf(w, format, args...)
	if err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}
	return nil
}

// printStatus is a convenience wrapper for writing to stdout with error checking
func printStatus(format string, args ...interface{}) error {
	return writeOutput(os.Stdout, format, args...)
}
