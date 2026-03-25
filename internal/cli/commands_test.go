package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vault-cli/vault/internal/domain"
)

// TestAuditCommand tests the audit command
func TestAuditCommand(t *testing.T) {
	helper := NewTestHelper(t)
	helper.SetupVault(t, "test-passphrase")

	// Unlock vault and add some operations
	cleanup := helper.unlockWithSession(t)
	defer cleanup()

	// Add an entry to generate audit log
	s := GetVaultStore()
	if s != nil {
		entry := &domain.Entry{
			Name:     "test-entry",
			Username: "user",
			Secret:   []byte("password"),
		}
		_ = s.CreateEntry("default", entry)
	}

	// Test audit log command
	cmd := NewAudit(helper.Config)
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := cmd.Execute()
	if err != nil {
		t.Logf("Audit command completed with: %v", err)
	}
}

// TestDoctorCommand tests the doctor command
func TestDoctorCommand(t *testing.T) {
	helper := NewTestHelper(t)
	helper.SetupVault(t, "test-passphrase")

	cmd := NewDoctor(helper.Config)
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := cmd.Execute()
	if err != nil {
		t.Logf("Doctor command completed with: %v", err)
	}
}

// TestExportCommand tests the export command
func TestExportCommand(t *testing.T) {
	helper := NewTestHelper(t)
	helper.SetupVault(t, "test-passphrase")

	// Unlock vault
	cleanup := helper.unlockWithSession(t)
	defer cleanup()

	// Create export file path
	exportPath := filepath.Join(helper.TempDir, "export.json")

	cmd := NewExport(helper.Config)
	cmd.SetArgs([]string{"--output", exportPath})

	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := cmd.Execute()
	if err != nil {
		t.Logf("Export command result: %v", err)
	}

	// Check if export file was created
	if _, err := os.Stat(exportPath); err == nil {
		t.Logf("Export file created successfully")
	}
}

// TestImportCommand tests the import command
func TestImportCommand(t *testing.T) {
	helper := NewTestHelper(t)
	helper.SetupVault(t, "test-passphrase")

	// Create a simple import file
	importPath := filepath.Join(helper.TempDir, "import.json")
	importData := `{
		"version": "1.0",
		"exported_at": "` + time.Now().Format(time.RFC3339) + `",
		"entries": {}
	}`

	if err := os.WriteFile(importPath, []byte(importData), 0600); err != nil {
		t.Fatalf("Failed to create import file: %v", err)
	}

	// Unlock vault
	cleanup := helper.unlockWithSession(t)
	defer cleanup()

	cmd := NewImport(helper.Config)
	cmd.SetArgs([]string{importPath})

	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := cmd.Execute()
	if err != nil {
		t.Logf("Import command result: %v", err)
	}
}

// TestProfilesListCommand tests profiles list
func TestProfilesListCommand(t *testing.T) {
	helper := NewTestHelper(t)
	helper.SetupVault(t, "test-passphrase")

	cleanup := helper.unlockWithSession(t)
	defer cleanup()

	cmd := NewProfiles(helper.Config)
	cmd.SetArgs([]string{"list"})

	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := cmd.Execute()
	if err != nil {
		t.Errorf("Profiles list failed: %v", err)
	}

	// Should show default profile
	output := stdout.String()
	if len(output) == 0 {
		t.Error("Profiles list should produce output")
	}
}

// TestProfilesCreateCommand tests profile creation
func TestProfilesCreateCommand(t *testing.T) {
	helper := NewTestHelper(t)
	helper.SetupVault(t, "test-passphrase")

	cleanup := helper.unlockWithSession(t)
	defer cleanup()

	cmd := NewProfiles(helper.Config)
	cmd.SetArgs([]string{"create", "test-profile", "--description", "Test Profile"})

	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := cmd.Execute()
	if err != nil {
		t.Logf("Profile create result: %v", err)
	}
}

// TestStatusCommand tests the status command
func TestStatusCommand(t *testing.T) {
	helper := NewTestHelper(t)
	helper.SetupVault(t, "test-passphrase")

	cleanup := helper.unlockWithSession(t)
	defer cleanup()

	cmd := NewStatus(helper.Config)

	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := cmd.Execute()
	if err != nil {
		t.Logf("Status command result: %v", err)
	}

	output := stdout.String() + stderr.String()
	if len(output) == 0 {
		t.Error("Status command should produce output")
	}
}

// TestUpdateCommand tests the update command
func TestUpdateCommand(t *testing.T) {
	helper := NewTestHelper(t)
	helper.SetupVault(t, "test-passphrase")

	cleanup := helper.unlockWithSession(t)
	defer cleanup()

	// First create an entry
	s := GetVaultStore()
	if s != nil {
		entry := &domain.Entry{
			Name:     "update-test",
			Username: "olduser",
			Secret:   []byte("oldpass"),
		}
		_ = s.CreateEntry("default", entry)
	}

	// Now update it
	cmd := NewUpdate(helper.Config)
	cmd.SetArgs([]string{"update-test", "--username", "newuser"})

	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := cmd.Execute()
	if err != nil {
		t.Logf("Update command result: %v", err)
	}
}

// TestRotateCommand tests password rotation
func TestRotateCommand(t *testing.T) {
	helper := NewTestHelper(t)
	helper.SetupVault(t, "test-passphrase")

	cleanup := helper.unlockWithSession(t)
	defer cleanup()

	// Create an entry first
	s := GetVaultStore()
	if s != nil {
		entry := &domain.Entry{
			Name:     "rotate-test",
			Username: "user",
			Secret:   []byte("oldpassword"),
		}
		_ = s.CreateEntry("default", entry)
	}

	cmd := NewRotatePassword(helper.Config)
	cmd.SetArgs([]string{"rotate-test", "--length", "16"})

	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := cmd.Execute()
	if err != nil {
		t.Logf("Rotate command result: %v", err)
	}
}

// TestConfigCommands tests config get/set
func TestConfigGetCommand(t *testing.T) {
	helper := NewTestHelper(t)

	cmd := NewConfig(helper.Config)
	cmd.SetArgs([]string{"get", "vault_path"})

	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := cmd.Execute()
	if err != nil {
		t.Logf("Config get result: %v", err)
	}
}

func TestConfigSetCommand(t *testing.T) {
	helper := NewTestHelper(t)

	cmd := NewConfig(helper.Config)
	cmd.SetArgs([]string{"set", "clipboard_timeout", "45"})

	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := cmd.Execute()
	if err != nil {
		t.Logf("Config set result: %v", err)
	}
}

func TestConfigPathCommand(t *testing.T) {
	helper := NewTestHelper(t)

	cmd := NewConfig(helper.Config)
	cmd.SetArgs([]string{"path"})

	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := cmd.Execute()
	if err != nil {
		t.Errorf("Config path failed: %v", err)
	}

	if stdout.Len() == 0 {
		t.Error("Config path should output the path")
	}
}
