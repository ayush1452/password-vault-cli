// Package clipboard provides cross-platform clipboard operations for the password vault.
// It allows secure copying of sensitive data to the system clipboard with automatic clearing.
package clipboard

import (
	"fmt"
	"log"
	"time"

	"github.com/atotto/clipboard"
)

// ClipboardInterface defines the interface for clipboard operations
type ClipboardInterface interface {
	WriteAll(text string) error
	ReadAll() (string, error)
}

// realClipboard wraps the actual clipboard package
type realClipboard struct{}

func (r *realClipboard) WriteAll(text string) error {
	return clipboard.WriteAll(text)
}

func (r *realClipboard) ReadAll() (string, error) {
	return clipboard.ReadAll()
}

// defaultClipboard is the default clipboard implementation
var defaultClipboard ClipboardInterface = &realClipboard{}

// CopyWithTimeout copies text to clipboard and clears it after timeout
func CopyWithTimeout(text string, timeout time.Duration) error {
	return copyWithTimeoutInternal(text, timeout, defaultClipboard)
}

// copyWithTimeoutInternal is the internal implementation that accepts a clipboard interface
func copyWithTimeoutInternal(text string, timeout time.Duration, cb ClipboardInterface) error {
	// Copy to clipboard
	if err := cb.WriteAll(text); err != nil {
		return fmt.Errorf("failed to copy to clipboard: %w", err)
	}

	// Start goroutine to clear clipboard after timeout
	go func() {
		time.Sleep(timeout)

		// Check if clipboard still contains our text before clearing
		current, err := cb.ReadAll()
		if err == nil && current == text {
			// Best effort clear, log any errors
			if err := cb.WriteAll(""); err != nil {
				log.Printf("warning: failed to clear clipboard: %v", err)
			}
		}
	}()

	return nil
}

// IsAvailable returns true if clipboard functionality is available
func IsAvailable() bool {
	// Try to read from clipboard to test availability
	_, err := clipboard.ReadAll()
	return err == nil
}

// Clear clears the clipboard
func Clear() error {
	return clipboard.WriteAll("")
}

// Copy copies text to the clipboard without a timeout
func Copy(text string) error {
	return clipboard.WriteAll(text)
}

// CanRead checks if the clipboard can be read
func CanRead() bool {
	_, err := clipboard.ReadAll()
	return err == nil
}

// Read reads the current content of the clipboard
func Read() (string, error) {
	return clipboard.ReadAll()
}
