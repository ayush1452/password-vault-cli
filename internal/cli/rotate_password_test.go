package cli

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vault-cli/vault/internal/config"
	"github.com/vault-cli/vault/internal/crypto"
	"github.com/vault-cli/vault/internal/domain"
)

// rotatePasswordVaultStore is a minimal implementation of VaultStore for testing
type rotatePasswordVaultStore struct {
	entries map[string]*domain.Entry
}

func (m *rotatePasswordVaultStore) GetEntry(profile, id string) (*domain.Entry, error) {
	if entry, exists := m.entries[id]; exists {
		// Return a copy to prevent test interference
		entryCopy := *entry
		return &entryCopy, nil
	}
	return nil, errors.New("entry not found")
}

func (m *rotatePasswordVaultStore) UpdateEntry(profile, id string, entry *domain.Entry) error {
	if _, exists := m.entries[id]; !exists {
		return errors.New("entry not found")
	}
	// Create a copy to prevent test interference
	entryCopy := *entry
	m.entries[id] = &entryCopy
	return nil
}

func (m *rotatePasswordVaultStore) IsOpen() bool { return true }

// Add other required store.VaultStore methods with empty implementations
func (m *rotatePasswordVaultStore) CreateVault(path string, masterKey []byte, kdfParams map[string]interface{}) error {
	return nil
}
func (m *rotatePasswordVaultStore) OpenVault(path string, masterKey []byte) error         { return nil }
func (m *rotatePasswordVaultStore) CloseVault() error                                     { return nil }
func (m *rotatePasswordVaultStore) CreateEntry(profile string, entry *domain.Entry) error { return nil }
func (m *rotatePasswordVaultStore) ListEntries(profile string, filter *domain.Filter) ([]*domain.Entry, error) {
	return nil, nil
}
func (m *rotatePasswordVaultStore) DeleteEntry(profile, id string) error             { return nil }
func (m *rotatePasswordVaultStore) EntryExists(profile, id string) bool              { return false }
func (m *rotatePasswordVaultStore) CreateProfile(name, description string) error     { return nil }
func (m *rotatePasswordVaultStore) GetProfile(name string) (*domain.Profile, error)  { return nil, nil }
func (m *rotatePasswordVaultStore) ListProfiles() ([]*domain.Profile, error)         { return nil, nil }
func (m *rotatePasswordVaultStore) DeleteProfile(name string) error                  { return nil }
func (m *rotatePasswordVaultStore) ProfileExists(name string) bool                   { return false }
func (m *rotatePasswordVaultStore) GetVaultMetadata() (*domain.VaultMetadata, error) { return nil, nil }
func (m *rotatePasswordVaultStore) UpdateVaultMetadata(metadata *domain.VaultMetadata) error {
	return nil
}
func (m *rotatePasswordVaultStore) LogOperation(op *domain.Operation) error            { return nil }
func (m *rotatePasswordVaultStore) GetAuditLog() ([]*domain.Operation, error)          { return nil, nil }
func (m *rotatePasswordVaultStore) VerifyAuditIntegrity() error                        { return nil }
func (m *rotatePasswordVaultStore) ExportVault(path string, includeSecrets bool) error { return nil }
func (m *rotatePasswordVaultStore) ImportVault(path string, conflictResolution string) error {
	return nil
}
func (m *rotatePasswordVaultStore) CompactVault() error                        { return nil }
func (m *rotatePasswordVaultStore) VerifyIntegrity() error                     { return nil }
func (m *rotatePasswordVaultStore) RotateMasterKey(newPassphrase string) error { return nil }

// TestRotatePasswordCommand tests the rotate password command
func TestRotatePasswordCommand(t *testing.T) {
	// Create a test entry
	testEntry := &domain.Entry{
		ID:        "test-entry",
		Name:      "test-entry",
		Password:  []byte("old-password"),
		CreatedAt: time.Now().Add(-24 * time.Hour),
		UpdatedAt: time.Now().Add(-1 * time.Hour),
	}

	// Create a mock vault store
	mockStore := &rotatePasswordVaultStore{
		entries: map[string]*domain.Entry{
			"test-entry": testEntry,
		},
	}

	// Save original functions to restore later
	originalCopyToClipboard := copyToClipboard
	originalClipboardAvailable := clipboardIsAvailable
	reader := &testReader{}
	crypto.SetRandomSource(reader)
	t.Cleanup(func() {
		crypto.SetRandomSource(nil)
	})

	// Setup mock session manager
	originalManager := sessionManager
	sessionManager = &SessionManager{
		vaultStore: mockStore,
		masterKey:  []byte("test-master-key"),
		unlockTime: time.Now(),
		ttl:        30 * time.Minute,
	}

	tests := []struct {
		name                 string
		args                 []string
		setup                func()
		expectError          bool
		expectOutput         string
		expectCopy           bool
		expectShow           bool
		expectedLength       int
		expectedClipboardTTL time.Duration
	}{
		{
			name: "rotate with default options",
			args: []string{"test-entry"},
			setup: func() {
				// Reset the entry before each test
				mockStore.entries["test-entry"] = &domain.Entry{
					ID:        "test-entry",
					Name:      "test-entry",
					Password:  []byte("old-password"),
					CreatedAt: time.Now().Add(-24 * time.Hour),
					UpdatedAt: time.Now().Add(-1 * time.Hour),
				}
			},
			expectError:    false,
			expectOutput:   "✓ Password rotated successfully\n",
			expectCopy:     false,
			expectedLength: 20,
		},
		{
			name: "rotate with show flag",
			args: []string{"test-entry", "--show"},
			setup: func() {
				mockStore.entries["test-entry"] = &domain.Entry{
					ID:        "test-entry",
					Name:      "test-entry",
					Password:  []byte("old-password"),
					CreatedAt: time.Now().Add(-24 * time.Hour),
					UpdatedAt: time.Now().Add(-1 * time.Hour),
				}
			},
			expectError:    false,
			expectOutput:   "",
			expectCopy:     false,
			expectShow:     true,
			expectedLength: 20,
		},
		{
			name: "rotate with copy flag",
			args: []string{"test-entry", "--copy", "--ttl", "30"},
			setup: func() {
				mockStore.entries["test-entry"] = &domain.Entry{
					ID:        "test-entry",
					Name:      "test-entry",
					Password:  []byte("old-password"),
					CreatedAt: time.Now().Add(-24 * time.Hour),
					UpdatedAt: time.Now().Add(-1 * time.Hour),
				}
			},
			expectError:          false,
			expectOutput:         "✓ Password rotated and copied to clipboard (clears in 30s)\n",
			expectCopy:           true,
			expectedLength:       20,
			expectedClipboardTTL: 30 * time.Second,
		},
		{
			name: "rotate with copy default ttl",
			args: []string{"test-entry", "--copy"},
			setup: func() {
				mockStore.entries["test-entry"] = &domain.Entry{
					ID:        "test-entry",
					Name:      "test-entry",
					Password:  []byte("old-password"),
					CreatedAt: time.Now().Add(-24 * time.Hour),
					UpdatedAt: time.Now().Add(-1 * time.Hour),
				}
			},
			expectError:          false,
			expectOutput:         "✓ Password rotated and copied to clipboard (clears in 45s)\n",
			expectCopy:           true,
			expectedLength:       20,
			expectedClipboardTTL: 45 * time.Second,
		},
		{
			name: "rotate with custom length",
			args: []string{"test-entry", "--length", "32"},
			setup: func() {
				mockStore.entries["test-entry"] = &domain.Entry{
					ID:        "test-entry",
					Name:      "test-entry",
					Password:  []byte("old-password"),
					CreatedAt: time.Now().Add(-24 * time.Hour),
					UpdatedAt: time.Now().Add(-1 * time.Hour),
				}
			},
			expectError:    false,
			expectOutput:   "✓ Password rotated successfully\n",
			expectCopy:     false,
			expectedLength: 32,
		},
		{
			name:         "rotate non-existent entry",
			args:         []string{"non-existent"},
			setup:        func() {},
			expectError:  true,
			expectOutput: "failed to get entry: entry not found",
			expectCopy:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test
			if tt.setup != nil {
				tt.setup()
			}

			// Setup clipboard spy
			clipSpy := &clipboardSpy{}

			// Setup clipboard
			copyToClipboard = clipSpy.copy
			clipboardIsAvailable = func() bool { return true }

			// Create command with mock store
			cmd := NewRotatePasswordCommand(&config.Config{ClipboardTTL: 45 * time.Second})
			stdout := &bytes.Buffer{}
			cmd.SetOut(stdout)
			cmd.SetErr(stdout)
			cmd.SetArgs(tt.args)

			// Execute command
			err := cmd.Execute()

			// Verify results
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, stdout.String(), tt.expectOutput)
				return
			}

			assert.NoError(t, err)

			var entry *domain.Entry
			if len(tt.args) > 0 && tt.args[0] == "test-entry" {
				entry, _ = mockStore.GetEntry("", "test-entry")
			}

			if tt.expectShow {
				if assert.NotNil(t, entry) {
					assert.Equal(t, string(entry.Password)+"\n", stdout.String())
				}
			} else if tt.expectOutput != "" {
				assert.Equal(t, tt.expectOutput, stdout.String())
			}

			if entry != nil {
				assert.NotEqual(t, "old-password", string(entry.Password))
				if tt.expectedLength > 0 {
					assert.Len(t, entry.Password, tt.expectedLength)
				}
				assert.True(t, time.Since(entry.UpdatedAt) < time.Second)
			}

			if tt.expectCopy {
				assert.True(t, clipSpy.called)
				if tt.expectedClipboardTTL > 0 {
					assert.Equal(t, tt.expectedClipboardTTL, clipSpy.ttl)
				}
				if entry != nil {
					assert.Equal(t, string(entry.Password), clipSpy.secret)
				}
			} else {
				assert.False(t, clipSpy.called)
			}
		})
	}

	// Restore original functions
	sessionManager = originalManager
	copyToClipboard = originalCopyToClipboard
	clipboardIsAvailable = originalClipboardAvailable
}
