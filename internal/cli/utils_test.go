package cli

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

// TestWriteString tests the writeString helper function
func TestWriteString(t *testing.T) {
	buf := new(bytes.Buffer)
	
	err := writeString(buf, "test output")
	if err != nil {
		t.Fatalf("writeString failed: %v", err)
	}
	
	if buf.String() != "test output" {
		t.Errorf("Expected 'test output', got '%s'", buf.String())
	}
}

// TestWriteStringError tests error handling
func TestWriteStringError(t *testing.T) {
	errorWriter := &errorWriter{}
	
	err := writeString(errorWriter, "test")
	if err == nil {
		t.Error("writeString should fail with error writer")
	}
}

// errorWriter always returns an error
type errorWriter struct{}

func (ew *errorWriter) Write(p []byte) (n int, err error) {
	return 0, io.ErrShortWrite
}

// TestWriteOutput tests formatted output
func TestWriteOutput(t *testing.T) {
	buf := new(bytes.Buffer)
	
	err := writeOutput(buf, "value: %d", 42)
	if err != nil {
		t.Fatalf("writeOutput failed: %v", err)
	}
	
	if buf.String() != "value: 42" {
		t.Errorf("Expected 'value: 42', got '%s'", buf.String())
	}
}

// TestSecurePrint tests secure printing
func TestSecurePrint(t *testing.T) {
	buf := new(bytes.Buffer)
	
	err := SecurePrint(buf, "password: %s", "secret123")
	if err != nil {
		t.Fatalf("SecurePrint failed: %v", err)
	}
	
	if !strings.Contains(buf.String(), "password:") {
		t.Error("SecurePrint should contain 'password:'")
	}
}

// TestCheckDeferredErr tests error checking
func TestCheckDeferredErr(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		op       string
		cerr     error
		wantNil  bool
	}{
		{"No errors", nil, "test", nil, true},
		{"Original error only", io.EOF, "test", nil, false},
		{"Close error only", nil, "close", io.ErrClosedPipe, false},
		{"Both errors", io.EOF, "close", io.ErrClosedPipe, false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkDeferredErr(tt.err, tt.op, tt.cerr)
			if (result == nil) != tt.wantNil {
				t.Errorf("checkDeferredErr() = %v, wantNil %v", result, tt.wantNil)
			}
		})
	}
}

// TestTruncateString tests string truncation
func TestTruncateString(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		length int
		want   string
	}{
		{"Short string", "hello", 10, "hello"},
		{"Exact length", "12345", 5, "12345"},
		{"Needs truncation", "hello world", 8, "hello..."},
		{"Very short", "a", 5, "a"},
		{"Zero length", "test", 0, ""},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateString(tt.input, tt.length)
			if got != tt.want {
				t.Errorf("truncateString(%q, %d) = %q, want %q", tt.input, tt.length, got, tt.want)
			}
		})
	}
}

// TestIsDebugEnabled tests debug mode detection
func TestIsDebugEnabled(t *testing.T) {
	// Just test that it returns a boolean without erroring
	result := isDebugEnabled()
	_ = result // Use the value to avoid unused variable error
}

// TestClearString tests secure string clearing
func TestClearString(t *testing.T) {
	// This function modifies string in-place, just test it doesn't panic
	testStr := "sensitive data"
	clearString(testStr)
	// Function completes without error
}

// TestSessionFunctions tests session utility functions
func TestIsUnlockedInitialState(t *testing.T) {
	// Session should initially be locked
	if IsUnlocked() {
		t.Error("Session should be locked initially")
	}
}

// TestGetVaultStoreWhenLocked tests getting store when locked
func TestGetVaultStoreWhenLocked(t *testing.T) {
	store := GetVaultStore()
	if store != nil {
		t.Error("GetVaultStore should return nil when locked")
	}
}

// TestRemainingSessionTTL tests TTL calculation
func TestRemainingSessionTTL(t *testing.T) {
	ttl := RemainingSessionTTL()
	if ttl < 0 {
		t.Error("Remaining TTL should not be negative")
	}
}

// TestSessionFilePath tests session file path generation
func TestSessionFilePath(t *testing.T) {
	path := sessionFilePath("/path/to/vault.db")
	if path == "" {
		t.Error("sessionFilePath should return non-empty path")
	}
	if !strings.Contains(path, "session") {
		t.Error("sessionFilePath should contain 'session'")
	}
}

// TestDeriveSessionPassphrase tests passphrase derivation
func TestDeriveSessionPassphrase(t *testing.T) {
	passphrase := deriveSessionPassphrase("/path/to/vault.db")
	if passphrase == "" {
		t.Error("deriveSessionPassphrase should return non-empty string")
	}
	
	// Different paths should give different passphrases
	passphrase2 := deriveSessionPassphrase("/different/path/vault.db")
	if passphrase == passphrase2 {
		t.Error("Different vault paths should produce different passphrases")
	}
}

// TestEnsureVaultDirectory tests directory creation
func TestEnsureVaultDirectory(t *testing.T) {
	tempDir := t.TempDir()
	vaultPath := tempDir + "/subdir/vault.db"
	
	err := EnsureVaultDirectory(vaultPath)
	if err != nil {
		t.Errorf("EnsureVaultDirectory failed: %v", err)
	}
}

// TestLogWarning tests warning logging
func TestLogWarning(t *testing.T) {
	// Just test it doesn't panic
	logWarning("test warning: %s", "message")
}

// TestRefreshSession tests session refresh
func TestRefreshSession(t *testing.T) {
	// Just test it doesn't panic when called on empty session
	RefreshSession()
}
