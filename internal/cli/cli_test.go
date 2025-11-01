package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/vault-cli/vault/internal/config"
	"github.com/vault-cli/vault/internal/domain"
	"github.com/vault-cli/vault/internal/store"
	"github.com/vault-cli/vault/internal/vault"
)

// TestHelper provides common test utilities
type TestHelper struct {
	TempDir    string
	ConfigPath string
	VaultPath  string
	Config     *config.Config
	Salt       []byte // Salt used for key derivation
	Passphrase string // Passphrase used to create the vault
}

// unlockWithSession prepares session state for commands that require an unlocked vault.
func (h *TestHelper) unlockWithSession(t *testing.T) func() {
	t.Helper()

	// Derive master key using stored salt
	crypto := vault.NewDefaultCryptoEngine()
	masterKey, err := crypto.DeriveKey(h.Passphrase, h.Salt)
	if err != nil {
		t.Fatalf("Failed to derive master key: %v", err)
	}

	restore := func() {
		_ = LockVault()
		vault.Zeroize(masterKey)
		sessionManager = nil
		vaultPath = ""
	}

	// Configure session manager with mock store
	s := store.NewBoltStore()
	if err := s.OpenVault(h.VaultPath, masterKey); err != nil {
		restore()
		t.Fatalf("Failed to open vault: %v", err)
	}

	sessionManager = &SessionManager{
		vaultPath:  h.VaultPath,
		vaultStore: s,
		masterKey:  masterKey,
		unlockTime: time.Now(),
		ttl:        time.Hour,
	}

	vaultPath = h.VaultPath

	return restore
}

// NewTestHelper creates a new test helper with temporary directories
func NewTestHelper(t *testing.T) *TestHelper {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	vaultPath := filepath.Join(tempDir, "test.vault")

	cfg := &config.Config{
		VaultPath:      vaultPath,
		DefaultProfile: "default",
		Profiles: map[string]*config.Profile{
			"default": {
				Name:             "default",
				VaultPath:        vaultPath,
				AutoLock:         300,
				ClipboardTimeout: 30,
			},
		},
		Security: config.SecurityConfig{
			SessionTimeout:      1800,
			MaxFailedAttempts:   3,
			RequireConfirmation: true,
		},
	}

	return &TestHelper{
		TempDir:    tempDir,
		ConfigPath: configPath,
		VaultPath:  vaultPath,
		Config:     cfg,
	}
}

// SetupVault initializes a test vault
func (h *TestHelper) SetupVault(t *testing.T, passphrase string) {
	// Store passphrase for later use
	h.Passphrase = passphrase

	// Create store
	s := store.NewBoltStore()

	// Generate salt and derive master key
	salt, err := vault.GenerateSalt()
	if err != nil {
		t.Fatalf("Failed to generate salt: %v", err)
	}

	// Store salt for later use
	h.Salt = salt

	crypto := vault.NewDefaultCryptoEngine()
	masterKey, err := crypto.DeriveKey(passphrase, salt)
	if err != nil {
		t.Fatalf("Failed to derive master key: %v", err)
	}
	defer vault.Zeroize(masterKey)

	// Create KDF params map
	kdfParams := vault.DefaultArgon2Params()
	kdfParamsMap := map[string]interface{}{
		"memory":      kdfParams.Memory,
		"iterations":  kdfParams.Iterations,
		"parallelism": kdfParams.Parallelism,
		"salt":        salt,
	}

	// Create vault
	err = s.CreateVault(h.VaultPath, masterKey, kdfParamsMap)
	if err != nil {
		t.Fatalf("Failed to create vault: %v", err)
	}

	err = s.CloseVault()
	if err != nil {
		t.Fatalf("Failed to close vault: %v", err)
	}
}

// ExecuteCommand executes a CLI command and returns output
func (h *TestHelper) ExecuteCommand(t *testing.T, cmd *cobra.Command, args ...string) (string, string, error) {
	var stdout, stderr bytes.Buffer

	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(args)

	err := cmd.Execute()
	return stdout.String(), stderr.String(), err
}

// TestInitCommand tests the init command
func TestInitCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		passphrase  string
		wantErr     bool
		errContains string
	}{
		{
			name:       "Valid initialization",
			args:       []string{},
			passphrase: "strong-test-passphrase",
			wantErr:    false,
		},
		{
			name:        "Weak passphrase",
			args:        []string{},
			passphrase:  "123",
			wantErr:     true,
			errContains: "passphrase too weak",
		},
		{
			name:        "Empty passphrase",
			args:        []string{},
			passphrase:  "",
			wantErr:     true,
			errContains: "passphrase cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			helper := NewTestHelper(t)

			// Mock passphrase input
			oldStdin := os.Stdin
			defer func() { os.Stdin = oldStdin }()

			// Create init command
			cmd := NewInitCommand(helper.Config)

			// Set environment for testing
			os.Setenv("VAULT_PATH", helper.VaultPath)
			defer os.Unsetenv("VAULT_PATH")

			// Execute command
			stdout, stderr, err := helper.ExecuteCommand(t, cmd, tt.args...)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if !strings.Contains(err.Error(), tt.errContains) && !strings.Contains(stderr, tt.errContains) {
					t.Errorf("Error %v does not contain %v", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v\nStdout: %s\nStderr: %s", err, stdout, stderr)
				return
			}

			// Verify vault was created
			if _, err := os.Stat(helper.VaultPath); os.IsNotExist(err) {
				t.Error("Vault file was not created")
			}
		})
	}
}

// TestAddCommand tests the add command
func TestAddCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		entry       *domain.Entry
		wantErr     bool
		errContains string
	}{
		{
			name: "Valid entry addition",
			args: []string{"test-site"},
			entry: &domain.Entry{
				Name:     "test-site",
				URL:      "https://test.com",
				Username: "testuser",
				Password: []byte("testpass123"),
				Notes:    "Test entry",
			},
			wantErr: false,
		},
		{
			name:        "Missing entry name",
			args:        []string{},
			wantErr:     true,
			errContains: "entry name is required",
		},
		{
			name: "Duplicate entry",
			args: []string{"duplicate"},
			entry: &domain.Entry{
				Name:     "duplicate",
				Username: "user",
				Password: []byte("pass"),
			},
			wantErr:     true,
			errContains: "entry already exists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			helper := NewTestHelper(t)
			helper.SetupVault(t, "test-passphrase")

			// For duplicate test, add entry first
			if tt.name == "Duplicate entry" {
				s := store.NewBoltStore()

				// Derive master key using stored salt
				crypto := vault.NewDefaultCryptoEngine()
				masterKey, _ := crypto.DeriveKey(helper.Passphrase, helper.Salt)
				defer vault.Zeroize(masterKey)

				// Open vault and add entry
				_ = s.OpenVault(helper.VaultPath, masterKey)
				defer s.CloseVault()
				_ = s.CreateEntry("default", tt.entry)
			}

			cmd := NewAddCommand(helper.Config)
			stdout, stderr, err := helper.ExecuteCommand(t, cmd, tt.args...)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if !strings.Contains(err.Error(), tt.errContains) && !strings.Contains(stderr, tt.errContains) {
					t.Errorf("Error %v does not contain %v", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v\nStdout: %s\nStderr: %s", err, stdout, stderr)
			}
		})
	}
}

// TestGetCommand tests the get command
func TestGetCommand(t *testing.T) {
	helper := NewTestHelper(t)
	helper.SetupVault(t, "test-passphrase")

	// Add test entries
	s := store.NewBoltStore()

	// Derive master key using stored salt
	crypto := vault.NewDefaultCryptoEngine()
	masterKey, err := crypto.DeriveKey(helper.Passphrase, helper.Salt)
	if err != nil {
		t.Fatalf("Failed to derive master key: %v", err)
	}
	defer vault.Zeroize(masterKey)

	err = s.OpenVault(helper.VaultPath, masterKey)
	if err != nil {
		t.Fatalf("Failed to open vault: %v", err)
	}
	defer s.CloseVault()

	testEntry := &domain.Entry{
		Name:     "test-entry",
		URL:      "https://example.com",
		Username: "testuser",
		Password: []byte("secret123"),
		Notes:    "Test notes",
	}

	err = s.CreateEntry("default", testEntry)
	if err != nil {
		t.Fatalf("Failed to add test entry: %v", err)
	}

	tests := []struct {
		name        string
		args        []string
		wantErr     bool
		errContains string
		wantOutput  string
	}{
		{
			name:       "Get existing entry",
			args:       []string{"test-entry"},
			wantErr:    false,
			wantOutput: "testuser",
		},
		{
			name:        "Get non-existent entry",
			args:        []string{"non-existent"},
			wantErr:     true,
			errContains: "entry not found",
		},
		{
			name:        "Missing entry name",
			args:        []string{},
			wantErr:     true,
			errContains: "entry name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewGetCommand(helper.Config)
			stdout, stderr, err := helper.ExecuteCommand(t, cmd, tt.args...)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if !strings.Contains(err.Error(), tt.errContains) && !strings.Contains(stderr, tt.errContains) {
					t.Errorf("Error %v does not contain %v", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v\nStdout: %s\nStderr: %s", err, stdout, stderr)
				return
			}

			if tt.wantOutput != "" && !strings.Contains(stdout, tt.wantOutput) {
				t.Errorf("Output %v does not contain %v", stdout, tt.wantOutput)
			}
		})
	}
}

// TestListCommand tests the list command
func TestListCommand(t *testing.T) {
	helper := NewTestHelper(t)
	helper.SetupVault(t, "test-passphrase")

	// Add test entries
	s := store.NewBoltStore()

	// Derive master key using stored salt
	crypto := vault.NewDefaultCryptoEngine()
	masterKey, err := crypto.DeriveKey(helper.Passphrase, helper.Salt)
	if err != nil {
		t.Fatalf("Failed to derive master key: %v", err)
	}
	defer vault.Zeroize(masterKey)

	err = s.OpenVault(helper.VaultPath, masterKey)
	if err != nil {
		t.Fatalf("Failed to open vault: %v", err)
	}
	defer s.CloseVault()

	entries := []*domain.Entry{
		{Name: "entry1", Username: "user1", Password: []byte("pass1")},
		{Name: "entry2", Username: "user2", Password: []byte("pass2")},
		{Name: "entry3", Username: "user3", Password: []byte("pass3")},
	}

	for _, entry := range entries {
		err = s.CreateEntry("default", entry)
		if err != nil {
			t.Fatalf("Failed to add entry %s: %v", entry.Name, err)
		}
	}

	tests := []struct {
		name       string
		args       []string
		wantErr    bool
		wantCount  int
		wantOutput []string
	}{
		{
			name:       "List all entries",
			args:       []string{},
			wantErr:    false,
			wantCount:  3,
			wantOutput: []string{"entry1", "entry2", "entry3"},
		},
		{
			name:       "List with search filter",
			args:       []string{"--search", "entry1"},
			wantErr:    false,
			wantCount:  1,
			wantOutput: []string{"entry1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := helper.unlockWithSession(t)
			t.Cleanup(cleanup)

			cmd := NewListCommand(helper.Config)
			stdout, stderr, err := helper.ExecuteCommand(t, cmd, tt.args...)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v\nStdout: %s\nStderr: %s", err, stdout, stderr)
				return
			}

			for _, expected := range tt.wantOutput {
				if !strings.Contains(stdout, expected) {
					t.Errorf("Output does not contain expected entry: %s", expected)
				}
			}
		})
	}
}

// TestDeleteCommand tests the delete command
func TestDeleteCommand(t *testing.T) {
	helper := NewTestHelper(t)
	helper.SetupVault(t, "test-passphrase")

	// Add test entry
	s := store.NewBoltStore()

	// Derive master key using stored salt
	crypto := vault.NewDefaultCryptoEngine()
	masterKey, err := crypto.DeriveKey(helper.Passphrase, helper.Salt)
	if err != nil {
		t.Fatalf("Failed to derive master key: %v", err)
	}
	defer vault.Zeroize(masterKey)

	err = s.OpenVault(helper.VaultPath, masterKey)
	if err != nil {
		t.Fatalf("Failed to open vault: %v", err)
	}
	defer s.CloseVault()

	testEntry := &domain.Entry{
		Name:     "delete-me",
		Username: "user",
		Password: []byte("pass"),
	}

	err = s.CreateEntry("default", testEntry)
	if err != nil {
		t.Fatalf("Failed to add test entry: %v", err)
	}

	tests := []struct {
		name        string
		args        []string
		wantErr     bool
		errContains string
	}{
		{
			name:    "Delete existing entry",
			args:    []string{"delete-me"},
			wantErr: false,
		},
		{
			name:        "Delete non-existent entry",
			args:        []string{"non-existent"},
			wantErr:     true,
			errContains: "entry not found",
		},
		{
			name:        "Missing entry name",
			args:        []string{},
			wantErr:     true,
			errContains: "entry name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := helper.unlockWithSession(t)
			t.Cleanup(cleanup)

			cmd := NewDeleteCommand(helper.Config)
			stdout, stderr, err := helper.ExecuteCommand(t, cmd, tt.args...)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if !strings.Contains(err.Error(), tt.errContains) && !strings.Contains(stderr, tt.errContains) {
					t.Errorf("Error %v does not contain %v", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v\nStdout: %s\nStderr: %s", err, stdout, stderr)
			}
		})
	}
}

// TestUnlockLockCommands tests unlock and lock commands
func TestUnlockLockCommands(t *testing.T) {
	helper := NewTestHelper(t)
	helper.SetupVault(t, "test-passphrase")

	tests := []struct {
		name        string
		command     string
		args        []string
		wantErr     bool
		errContains string
	}{
		{
			name:    "Unlock vault",
			command: "unlock",
			args:    []string{},
			wantErr: false,
		},
		{
			name:    "Lock vault",
			command: "lock",
			args:    []string{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cmd *cobra.Command

			switch tt.command {
			case "unlock":
				cmd = NewUnlockCommand(helper.Config)
			case "lock":
				cmd = NewLockCommand(helper.Config)
			}

			stdout, stderr, err := helper.ExecuteCommand(t, cmd, tt.args...)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if !strings.Contains(err.Error(), tt.errContains) && !strings.Contains(stderr, tt.errContains) {
					t.Errorf("Error %v does not contain %v", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v\nStdout: %s\nStderr: %s", err, stdout, stderr)
			}
		})
	}
}

// TestCommandValidation tests input validation across commands
func TestCommandValidation(t *testing.T) {
	helper := NewTestHelper(t)

	tests := []struct {
		name        string
		command     string
		args        []string
		wantErr     bool
		errContains string
	}{
		{
			name:        "Add command with invalid characters",
			command:     "add",
			args:        []string{"invalid/name"},
			wantErr:     true,
			errContains: "invalid entry name",
		},
		{
			name:        "Get command with empty name",
			command:     "get",
			args:        []string{""},
			wantErr:     true,
			errContains: "entry name cannot be empty",
		},
		{
			name:        "Delete command with special characters",
			command:     "delete",
			args:        []string{"../../../etc/passwd"},
			wantErr:     true,
			errContains: "invalid entry name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cmd *cobra.Command

			switch tt.command {
			case "add":
				cmd = NewAddCommand(helper.Config)
			case "get":
				cmd = NewGetCommand(helper.Config)
			case "delete":
				cmd = NewDeleteCommand(helper.Config)
			}

			stdout, stderr, err := helper.ExecuteCommand(t, cmd, tt.args...)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if !strings.Contains(err.Error(), tt.errContains) && !strings.Contains(stderr, tt.errContains) {
					t.Errorf("Error %v does not contain %v", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v\nStdout: %s\nStderr: %s", err, stdout, stderr)
			}
		})
	}
}

// TestConfigCommand tests configuration management
func TestConfigCommand(t *testing.T) {
	helper := NewTestHelper(t)
	originalCfg := cfg
	originalCfgFile := cfgFile

	cfg = helper.Config
	cfgFile = helper.ConfigPath

	t.Cleanup(func() {
		cfg = originalCfg
		cfgFile = originalCfgFile
	})

	tests := []struct {
		name        string
		args        []string
		wantErr     bool
		errContains string
		wantOutput  string
	}{
		{
			name:       "Show current config",
			args:       []string{"show"},
			wantErr:    false,
			wantOutput: "default",
		},
		{
			name:       "Set session timeout",
			args:       []string{"set", "session-timeout", "3600"},
			wantErr:    false,
			wantOutput: "Configuration updated",
		},
		{
			name:        "Set invalid timeout",
			args:        []string{"set", "session-timeout", "invalid"},
			wantErr:     true,
			errContains: "invalid timeout value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewConfigCommand(helper.Config)
			stdout, stderr, err := helper.ExecuteCommand(t, cmd, tt.args...)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if !strings.Contains(err.Error(), tt.errContains) && !strings.Contains(stderr, tt.errContains) {
					t.Errorf("Error %v does not contain %v", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v\nStdout: %s\nStderr: %s", err, stdout, stderr)
				return
			}

			if tt.wantOutput != "" && !strings.Contains(stdout, tt.wantOutput) {
				t.Errorf("Output %v does not contain %v", stdout, tt.wantOutput)
			}
		})
	}
}
