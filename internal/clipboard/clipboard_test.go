package clipboard

import (
	"errors"
	"testing"
	"time"
)

// mockClipboard is a mock implementation for testing
type mockClipboard struct {
	data         string
	writeError   error
	readError    error
	shouldFail   bool
	failOnClear  bool
}

func (m *mockClipboard) WriteAll(text string) error {
	if m.shouldFail && text != "" {
		return m.writeError
	}
	if m.failOnClear && text == "" {
		return m.writeError
	}
	m.data = text
	return nil
}

func (m *mockClipboard) ReadAll() (string, error) {
	if m.readError != nil {
		return "", m.readError
	}
	return m.data, nil
}

// TestIsAvailable tests clipboard availability detection
func TestIsAvailable(t *testing.T) {
	// IsAvailable should not panic
	available := IsAvailable()

	// We can't guarantee clipboard is available in all test environments
	// but the function should at least return without error
	t.Logf("Clipboard available: %v", available)
}

// TestCopyBasic tests basic copy functionality
func TestCopyBasic(t *testing.T) {
	if !IsAvailable() {
		t.Skip("Clipboard not available in test environment")
	}

	testData := "test-clipboard-data"

	err := Copy(testData)
	if err != nil {
		t.Fatalf("Copy() error = %v", err)
	}

	// Verify copy succeeded (if we can read back)
	if CanRead() {
		content, err := Read()
		if err != nil {
			t.Fatalf("Read() error = %v", err)
		}

		if content != testData {
			t.Errorf("Read() = %v, want %v", content, testData)
		}
	}
}

// TestCopyWithTimeout tests copy with automatic timeout
func TestCopyWithTimeout(t *testing.T) {
	if !IsAvailable() {
		t.Skip("Clipboard not available in test environment")
	}

	testData := "timeout-test-data"
	timeout := 100 * time.Millisecond

	err := CopyWithTimeout(testData, timeout)
	if err != nil {
		t.Fatalf("CopyWithTimeout() error = %v", err)
	}

	// Immediately check content
	if CanRead() {
		content, _ := Read()
		if content != testData {
			t.Error("Content should be available immediately after copy")
		}
	}

	// Wait for timeout + buffer
	time.Sleep(timeout + 50*time.Millisecond)

	// Content may or may not be cleared depending on implementation
	// This test just ensures no panic occurs
}

// TestClear tests clipboard clearing
func TestClear(t *testing.T) {
	if !IsAvailable() {
		t.Skip("Clipboard not available in test environment")
	}

	// First copy something
	err := Copy("clear-test-data")
	if err != nil {
		t.Fatalf("Copy() error = %v", err)
	}

	// Clear clipboard
	err = Clear()
	if err != nil {
		t.Fatalf("Clear() error = %v", err)
	}

	// Verify content is cleared (if we can read)
	if CanRead() {
		content, _ := Read()
		if content != "" {
			t.Logf("Note: Clipboard not empty after clear (platform dependent): %v", content)
		}
	}
}

// TestCopyEmpty tests copying empty string
func TestCopyEmpty(t *testing.T) {
	if !IsAvailable() {
		t.Skip("Clipboard not available in test environment")
	}

	err := Copy("")
	if err != nil {
		t.Fatalf("Copy() with empty string error = %v", err)
	}
}

// TestCopyLargeContent tests copying large content
func TestCopyLargeContent(t *testing.T) {
	if !IsAvailable() {
		t.Skip("Clipboard not available in test environment")
	}

	// Create a large string
	largeData := make([]byte, 10000)
	for i := range largeData {
		largeData[i] = byte('a' + (i % 26))
	}

	err := Copy(string(largeData))
	if err != nil {
		t.Fatalf("Copy() large content error = %v", err)
	}
}

// TestCopyUnicode tests copying unicode content
func TestCopyUnicode(t *testing.T) {
	if !IsAvailable() {
		t.Skip("Clipboard not available in test environment")
	}

	unicodeData := "Hello 世界 🔒 Пароль"

	err := Copy(unicodeData)
	if err != nil {
		t.Fatalf("Copy() unicode error = %v", err)
	}

	if CanRead() {
		content, err := Read()
		if err != nil {
			t.Fatalf("Read() error = %v", err)
		}

		if content != unicodeData {
			t.Errorf("Unicode content mismatch: got %v, want %v", content, unicodeData)
		}
	}
}

// TestMultipleCopies tests multiple sequential copies
func TestMultipleCopies(t *testing.T) {
	if !IsAvailable() {
		t.Skip("Clipboard not available in test environment")
	}

	tests := []string{
		"first-copy",
		"second-copy",
		"third-copy",
	}

	for _, data := range tests {
		err := Copy(data)
		if err != nil {
			t.Fatalf("Copy(%v) error = %v", data, err)
		}

		if CanRead() {
			content, _ := Read()
			if content != data {
				t.Errorf("After Copy(%v), Read() = %v", data, content)
			}
		}
	}
}

// TestConcurrentCopy tests concurrent clipboard operations
func TestConcurrentCopy(t *testing.T) {
	if !IsAvailable() {
		t.Skip("Clipboard not available in test environment")
	}

	done := make(chan bool)

	// Launch multiple goroutines
	for i := 0; i < 5; i++ {
		go func(id int) {
			data := generateData("concurrent", id)
			err := Copy(data)
			if err != nil {
				t.Errorf("Concurrent Copy(%d) error = %v", id, err)
			}
			done <- true
		}(i)
	}

	// Wait for all to complete
	for i := 0; i < 5; i++ {
		<-done
	}
}

// TestTimeoutCancellation tests that timeout can be cancelled
func TestTimeoutCancellation(t *testing.T) {
	if !IsAvailable() {
		t.Skip("Clipboard not available in test environment")
	}

	// Copy with a long timeout
	err := CopyWithTimeout("cancel-test", 10*time.Second)
	if err != nil {
		t.Fatalf("CopyWithTimeout() error = %v", err)
	}

	// Immediately clear (should cancel timeout)
	err = Clear()
	if err != nil {
		t.Fatalf("Clear() error = %v", err)
	}

	// No panic should occur
}

// TestSpecialCharacters tests copying special characters
func TestSpecialCharacters(t *testing.T) {
	if !IsAvailable() {
		t.Skip("Clipboard not available in test environment")
	}

	specialChars := []string{
		"password!@#$%^&*()",
		"line1\nline2\nline3",
		"tab\tseparated\tvalues",
		"quotes\"and'apostrophes",
		"backslash\\test",
	}

	for _, data := range specialChars {
		err := Copy(data)
		if err != nil {
			t.Errorf("Copy(%q) error = %v", data, err)
		}
	}
}

// TestIsNotAvailable tests behavior when clipboard is not available
func TestIsNotAvailable(t *testing.T) {
	// This test verifies that functions handle unavailable clipboard gracefully
	// In a real scenario where clipboard is not available

	if IsAvailable() {
		t.Skip("Clipboard is available, cannot test unavailable scenario")
	}

	// Should not panic
	err := Copy("test")
	if err == nil {
		t.Error("Copy() should error when clipboard is not available")
	}
}

// Helper functions
func generateData(prefix string, id int) string {
	return prefix + "-" + itoa(id)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf) - 1
	for n > 0 {
		buf[i] = byte('0' + n%10)
		n /= 10
		i--
	}
	return string(buf[i+1:])
}

// TestCopyWithTimeoutWriteError tests error handling when initial WriteAll fails
func TestCopyWithTimeoutWriteError(t *testing.T) {
	mock := &mockClipboard{
		shouldFail: true,
		writeError: errors.New("mock write error"),
	}

	err := copyWithTimeoutInternal("test data", 100*time.Millisecond, mock)
	if err == nil {
		t.Error("Expected error when WriteAll fails, got nil")
	}

	if err != nil && !errors.Is(err, mock.writeError) {
		// Check if error message contains our mock error
		expectedMsg := "failed to copy to clipboard"
		if len(err.Error()) == 0 || err.Error()[:len(expectedMsg)] != expectedMsg {
			t.Errorf("Expected error to contain '%s', got: %v", expectedMsg, err)
		}
	}
}

// TestCopyWithTimeoutClearError tests error logging when clear fails in timeout goroutine
func TestCopyWithTimeoutClearError(t *testing.T) {
	testData := "test-data-for-clear-error"
	
	mock := &mockClipboard{
		failOnClear: true,
		writeError:  errors.New("mock clear error"),
	}

	// This should succeed initially
	err := copyWithTimeoutInternal(testData, 50*time.Millisecond, mock)
	if err != nil {
		t.Fatalf("Expected no error on initial copy, got: %v", err)
	}

	// Verify data was written
	if mock.data != testData {
		t.Errorf("Expected data to be '%s', got '%s'", testData, mock.data)
	}

	// Wait for timeout to trigger the clear attempt (which will fail and log)
	time.Sleep(100 * time.Millisecond)

	// The goroutine should have attempted to clear and hit the error path
	// Since we can't easily capture log output, we just verify that no panic occurred
	// and the test completes successfully
}

// TestCopyWithTimeoutClearSuccess tests successful clear after timeout
func TestCopyWithTimeoutClearSuccess(t *testing.T) {
	testData := "test-data-for-clear-success"
	
	mock := &mockClipboard{}

	err := copyWithTimeoutInternal(testData, 50*time.Millisecond, mock)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify data was written
	if mock.data != testData {
		t.Errorf("Expected data to be '%s', got '%s'", testData, mock.data)
	}

	// Wait for timeout to trigger clear
	time.Sleep(100 * time.Millisecond)

	// Verify data was cleared
	if mock.data != "" {
		t.Errorf("Expected data to be cleared, got '%s'", mock.data)
	}
}

// TestCopyWithTimeoutDataChanged tests that clear doesn't happen if data changed
func TestCopyWithTimeoutDataChanged(t *testing.T) {
	testData := "original-data"
	
	mock := &mockClipboard{}

	err := copyWithTimeoutInternal(testData, 50*time.Millisecond, mock)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Change the clipboard data before timeout
	mock.data = "different-data"

	// Wait for timeout
	time.Sleep(100 * time.Millisecond)

	// Verify data was NOT cleared (because it changed)
	if mock.data != "different-data" {
		t.Errorf("Expected data to remain 'different-data', got '%s'", mock.data)
	}
}

// TestCopyWithTimeoutReadError tests behavior when ReadAll fails during timeout check
func TestCopyWithTimeoutReadError(t *testing.T) {
	testData := "test-data"
	
	mock := &mockClipboard{
		readError: errors.New("mock read error"),
	}

	err := copyWithTimeoutInternal(testData, 50*time.Millisecond, mock)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Wait for timeout - the goroutine should handle read error gracefully
	time.Sleep(100 * time.Millisecond)

	// No panic should occur
}

