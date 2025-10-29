package clipboard

import (
	"fmt"
	"time"

	"github.com/atotto/clipboard"
)

// CopyWithTimeout copies text to clipboard and clears it after timeout
func CopyWithTimeout(text string, timeout time.Duration) error {
	// Copy to clipboard
	if err := clipboard.WriteAll(text); err != nil {
		return fmt.Errorf("failed to copy to clipboard: %w", err)
	}

	// Start goroutine to clear clipboard after timeout
	go func() {
		time.Sleep(timeout)

		// Check if clipboard still contains our text before clearing
		current, err := clipboard.ReadAll()
		if err == nil && current == text {
			clipboard.WriteAll("")
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
