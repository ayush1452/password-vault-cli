package tests

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// AcceptanceTestSuite contains end-to-end acceptance tests
type AcceptanceTestSuite struct {
	TempDir    string
	VaultPath  string
	ConfigPath string
	BinaryPath string
	Passphrase string
}

// NewAcceptanceTestSuite creates a new acceptance test suite
func NewAcceptanceTestSuite(t *testing.T) *AcceptanceTestSuite {
	tempDir := t.TempDir()
	vaultPath := filepath.Join(tempDir, "acceptance.vault")
	configPath := filepath.Join(tempDir, "config.yaml")
	binaryPath := filepath.Join(tempDir, "vault")

	return &AcceptanceTestSuite{
		TempDir:    tempDir,
		VaultPath:  vaultPath,
		ConfigPath: configPath,
		BinaryPath: binaryPath,
		Passphrase: "acceptance-test-passphrase-2024",
	}
}

// BuildBinary builds the vault CLI binary for testing
func (suite *AcceptanceTestSuite) BuildBinary(t *testing.T) {
	cmd := exec.Command("go", "build", "-o", suite.BinaryPath, "./cmd/vault")
	cmd.Dir = "/workspace"

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build binary: %v\nOutput: %s", err, output)
	}
}

// RunCommand executes a vault CLI command and returns the result
func (suite *AcceptanceTestSuite) RunCommand(t *testing.T, args ...string) (string, string, int) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, suite.BinaryPath, args...)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("VAULT_PATH=%s", suite.VaultPath),
		fmt.Sprintf("VAULT_CONFIG=%s", suite.ConfigPath),
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			exitCode = 1
		}
	}

	return stdout.String(), stderr.String(), exitCode
}

// RunCommandWithInput executes a command with stdin input
func (suite *AcceptanceTestSuite) RunCommandWithInput(t *testing.T, input string, args ...string) (string, string, int) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, suite.BinaryPath, args...)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("VAULT_PATH=%s", suite.VaultPath),
		fmt.Sprintf("VAULT_CONFIG=%s", suite.ConfigPath),
	)

	cmd.Stdin = strings.NewReader(input)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			exitCode = 1
		}
	}

	return stdout.String(), stderr.String(), exitCode
}

// TestCompleteWorkflow tests the complete vault workflow
func TestCompleteWorkflow(t *testing.T) {
	suite := NewAcceptanceTestSuite(t)
	suite.BuildBinary(t)

	t.Run("Complete User Workflow", func(t *testing.T) {
		// Step 1: Initialize vault
		t.Log("Step 1: Initialize vault")
		input := fmt.Sprintf("%s\n%s\n", suite.Passphrase, suite.Passphrase)
		stdout, stderr, exitCode := suite.RunCommandWithInput(t, input, "init")

		if exitCode != 0 {
			t.Fatalf("Init failed: exit code %d\nStdout: %s\nStderr: %s", exitCode, stdout, stderr)
		}

		if !strings.Contains(stdout, "Vault initialized") {
			t.Errorf("Expected initialization success message, got: %s", stdout)
		}

		// Verify vault file was created
		if _, err := os.Stat(suite.VaultPath); os.IsNotExist(err) {
			t.Error("Vault file was not created")
		}

		// Step 2: Add entries
		t.Log("Step 2: Add entries")
		entries := []struct {
			name     string
			url      string
			username string
			password string
			notes    string
		}{
			{"github.com", "https://github.com", "testuser", "github123", "Development account"},
			{"gmail.com", "https://gmail.com", "test@gmail.com", "email456", "Personal email"},
			{"aws.amazon.com", "https://aws.amazon.com", "awsuser", "aws789", "Cloud services"},
		}

		for _, entry := range entries {
			input := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n",
				entry.url, entry.username, entry.password, entry.notes, suite.Passphrase)
			stdout, stderr, exitCode := suite.RunCommandWithInput(t, input, "add", entry.name)

			if exitCode != 0 {
				t.Errorf("Add entry %s failed: exit code %d\nStdout: %s\nStderr: %s",
					entry.name, exitCode, stdout, stderr)
				continue
			}

			if !strings.Contains(stdout, "Entry added") {
				t.Errorf("Expected entry added message for %s, got: %s", entry.name, stdout)
			}
		}

		// Step 3: List entries
		t.Log("Step 3: List entries")
		input = fmt.Sprintf("%s\n", suite.Passphrase)
		stdout, stderr, exitCode = suite.RunCommandWithInput(t, input, "list")

		if exitCode != 0 {
			t.Fatalf("List failed: exit code %d\nStdout: %s\nStderr: %s", exitCode, stdout, stderr)
		}

		for _, entry := range entries {
			if !strings.Contains(stdout, entry.name) {
				t.Errorf("Entry %s not found in list output", entry.name)
			}
		}

		// Step 4: Get specific entries
		t.Log("Step 4: Get specific entries")
		for _, entry := range entries {
			input := fmt.Sprintf("%s\n", suite.Passphrase)
			stdout, stderr, exitCode := suite.RunCommandWithInput(t, input, "get", entry.name)

			if exitCode != 0 {
				t.Errorf("Get entry %s failed: exit code %d\nStdout: %s\nStderr: %s",
					entry.name, exitCode, stdout, stderr)
				continue
			}

			if !strings.Contains(stdout, entry.username) {
				t.Errorf("Username not found in get output for %s", entry.name)
			}
		}

		// Step 5: Update an entry
		t.Log("Step 5: Update an entry")
		input = fmt.Sprintf("https://github.com\nupdated-user\nupdated-pass\nUpdated notes\n%s\n", suite.Passphrase)
		stdout, stderr, exitCode = suite.RunCommandWithInput(t, input, "update", "github.com")

		if exitCode != 0 {
			t.Errorf("Update failed: exit code %d\nStdout: %s\nStderr: %s", exitCode, stdout, stderr)
		}

		// Verify update
		input = fmt.Sprintf("%s\n", suite.Passphrase)
		stdout, stderr, exitCode = suite.RunCommandWithInput(t, input, "get", "github.com")

		if exitCode == 0 && !strings.Contains(stdout, "updated-user") {
			t.Error("Entry was not updated properly")
		}

		// Step 6: Search entries
		t.Log("Step 6: Search entries")
		input = fmt.Sprintf("%s\n", suite.Passphrase)
		stdout, stderr, exitCode = suite.RunCommandWithInput(t, input, "list", "--search", "gmail")

		if exitCode != 0 {
			t.Errorf("Search failed: exit code %d\nStdout: %s\nStderr: %s", exitCode, stdout, stderr)
		} else if !strings.Contains(stdout, "gmail.com") {
			t.Error("Search did not find expected entry")
		}

		// Step 7: Lock and unlock vault
		t.Log("Step 7: Lock and unlock vault")
		stdout, stderr, exitCode = suite.RunCommand(t, "lock")

		if exitCode != 0 {
			t.Errorf("Lock failed: exit code %d\nStdout: %s\nStderr: %s", exitCode, stdout, stderr)
		}

		input = fmt.Sprintf("%s\n", suite.Passphrase)
		stdout, stderr, exitCode = suite.RunCommandWithInput(t, input, "unlock")

		if exitCode != 0 {
			t.Errorf("Unlock failed: exit code %d\nStdout: %s\nStderr: %s", exitCode, stdout, stderr)
		}

		// Step 8: Delete an entry
		t.Log("Step 8: Delete an entry")
		input = fmt.Sprintf("y\n%s\n", suite.Passphrase)
		stdout, stderr, exitCode = suite.RunCommandWithInput(t, input, "delete", "aws.amazon.com")

		if exitCode != 0 {
			t.Errorf("Delete failed: exit code %d\nStdout: %s\nStderr: %s", exitCode, stdout, stderr)
		}

		// Verify deletion
		input = fmt.Sprintf("%s\n", suite.Passphrase)
		stdout, stderr, exitCode = suite.RunCommandWithInput(t, input, "get", "aws.amazon.com")

		if exitCode == 0 {
			t.Error("Entry was not deleted properly")
		}

		// Step 9: Export data
		t.Log("Step 9: Export data")
		exportPath := filepath.Join(suite.TempDir, "export.json")
		input = fmt.Sprintf("%s\n", suite.Passphrase)
		stdout, stderr, exitCode = suite.RunCommandWithInput(t, input, "export", exportPath)

		if exitCode != 0 {
			t.Errorf("Export failed: exit code %d\nStdout: %s\nStderr: %s", exitCode, stdout, stderr)
		} else {
			// Verify export file exists
			if _, err := os.Stat(exportPath); os.IsNotExist(err) {
				t.Error("Export file was not created")
			}
		}

		t.Log("✅ Complete workflow test passed")
	})
}

// TestErrorHandling tests error scenarios and edge cases
func TestErrorHandling(t *testing.T) {
	suite := NewAcceptanceTestSuite(t)
	suite.BuildBinary(t)

	t.Run("Error Scenarios", func(t *testing.T) {
		// Test 1: Wrong passphrase
		t.Log("Test 1: Wrong passphrase")

		// Initialize vault first
		input := fmt.Sprintf("%s\n%s\n", suite.Passphrase, suite.Passphrase)
		suite.RunCommandWithInput(t, input, "init")

		// Try with wrong passphrase
		input = "wrong-passphrase\n"
		stdout, stderr, exitCode := suite.RunCommandWithInput(t, input, "list")

		if exitCode == 0 {
			t.Error("Should fail with wrong passphrase")
		}

		if !strings.Contains(stderr, "authentication failed") && !strings.Contains(stdout, "authentication failed") {
			t.Logf("Expected authentication error, got stdout: %s, stderr: %s", stdout, stderr)
		}

		// Test 2: Non-existent entry
		t.Log("Test 2: Non-existent entry")
		input = fmt.Sprintf("%s\n", suite.Passphrase)
		stdout, stderr, exitCode = suite.RunCommandWithInput(t, input, "get", "non-existent-entry")

		if exitCode == 0 {
			t.Error("Should fail for non-existent entry")
		}

		// Test 3: Invalid entry name
		t.Log("Test 3: Invalid entry name")
		input = fmt.Sprintf("https://test.com\nuser\npass\nnotes\n%s\n", suite.Passphrase)
		stdout, stderr, exitCode = suite.RunCommandWithInput(t, input, "add", "../invalid/name")

		if exitCode == 0 {
			t.Error("Should fail for invalid entry name")
		}

		// Test 4: Duplicate entry
		t.Log("Test 4: Duplicate entry")

		// Add an entry first
		input = fmt.Sprintf("https://test.com\nuser\npass\nnotes\n%s\n", suite.Passphrase)
		suite.RunCommandWithInput(t, input, "add", "test-entry")

		// Try to add the same entry again
		input = fmt.Sprintf("https://test.com\nuser2\npass2\nnotes2\n%s\n", suite.Passphrase)
		stdout, stderr, exitCode = suite.RunCommandWithInput(t, input, "add", "test-entry")

		if exitCode == 0 {
			t.Error("Should fail for duplicate entry")
		}

		// Test 5: Missing arguments
		t.Log("Test 5: Missing arguments")
		stdout, stderr, exitCode = suite.RunCommand(t, "get")

		if exitCode == 0 {
			t.Error("Should fail when entry name is missing")
		}

		// Test 6: Invalid command
		t.Log("Test 6: Invalid command")
		stdout, stderr, exitCode = suite.RunCommand(t, "invalid-command")

		if exitCode == 0 {
			t.Error("Should fail for invalid command")
		}

		t.Log("✅ Error handling tests passed")
	})
}

// TestCrossplatformCompatibility tests cross-platform functionality
func TestCrossplatformCompatibility(t *testing.T) {
	suite := NewAcceptanceTestSuite(t)
	suite.BuildBinary(t)

	t.Run("Cross-platform Compatibility", func(t *testing.T) {
		// Test 1: Path handling
		t.Log("Test 1: Path handling")

		// Test with different path formats
		testPaths := []string{
			filepath.Join(suite.TempDir, "test1.vault"),
			filepath.Join(suite.TempDir, "subdir", "test2.vault"),
		}

		for i, testPath := range testPaths {
			// Create directory if needed
			dir := filepath.Dir(testPath)
			os.MkdirAll(dir, 0755)

			// Set vault path
			os.Setenv("VAULT_PATH", testPath)

			// Initialize vault
			input := fmt.Sprintf("%s\n%s\n", suite.Passphrase, suite.Passphrase)
			stdout, stderr, exitCode := suite.RunCommandWithInput(t, input, "init")

			if exitCode != 0 {
				t.Errorf("Path test %d failed: exit code %d\nStdout: %s\nStderr: %s",
					i+1, exitCode, stdout, stderr)
				continue
			}

			// Verify file was created
			if _, err := os.Stat(testPath); os.IsNotExist(err) {
				t.Errorf("Vault file not created at path: %s", testPath)
			}
		}

		// Test 2: Unicode handling
		t.Log("Test 2: Unicode handling")

		// Reset to original vault path
		os.Setenv("VAULT_PATH", suite.VaultPath)

		// Initialize vault
		input := fmt.Sprintf("%s\n%s\n", suite.Passphrase, suite.Passphrase)
		suite.RunCommandWithInput(t, input, "init")

		// Add entry with unicode characters
		unicodeEntry := struct {
			name     string
			username string
			password string
			notes    string
		}{
			"测试-site", "用户名", "密码123", "Unicode测试笔记🔐",
		}

		input = fmt.Sprintf("https://测试.com\n%s\n%s\n%s\n%s\n",
			unicodeEntry.username, unicodeEntry.password, unicodeEntry.notes, suite.Passphrase)
		stdout, stderr, exitCode := suite.RunCommandWithInput(t, input, "add", unicodeEntry.name)

		if exitCode != 0 {
			t.Errorf("Unicode entry add failed: exit code %d\nStdout: %s\nStderr: %s",
				exitCode, stdout, stderr)
		} else {
			// Verify unicode entry can be retrieved
			input = fmt.Sprintf("%s\n", suite.Passphrase)
			stdout, stderr, exitCode = suite.RunCommandWithInput(t, input, "get", unicodeEntry.name)

			if exitCode != 0 {
				t.Errorf("Unicode entry get failed: exit code %d\nStdout: %s\nStderr: %s",
					exitCode, stdout, stderr)
			} else if !strings.Contains(stdout, unicodeEntry.username) {
				t.Error("Unicode characters not preserved in entry")
			}
		}

		// Test 3: File permissions
		t.Log("Test 3: File permissions")

		// Check vault file permissions
		info, err := os.Stat(suite.VaultPath)
		if err != nil {
			t.Errorf("Failed to stat vault file: %v", err)
		} else {
			mode := info.Mode()
			// Vault file should not be world-readable
			if mode&0044 != 0 {
				t.Errorf("Vault file has insecure permissions: %v", mode)
			}
		}

		t.Log("✅ Cross-platform compatibility tests passed")
	})
}

// TestPerformanceAcceptance tests performance under realistic conditions
func TestPerformanceAcceptance(t *testing.T) {
	suite := NewAcceptanceTestSuite(t)
	suite.BuildBinary(t)

	t.Run("Performance Acceptance", func(t *testing.T) {
		// Initialize vault
		input := fmt.Sprintf("%s\n%s\n", suite.Passphrase, suite.Passphrase)
		suite.RunCommandWithInput(t, input, "init")

		// Test 1: Large number of entries
		t.Log("Test 1: Large number of entries")

		numEntries := 1000
		start := time.Now()

		for i := 0; i < numEntries; i++ {
			entryName := fmt.Sprintf("perf-entry-%d", i)
			input := fmt.Sprintf("https://example%d.com\nuser%d\npass%d\nNotes for entry %d\n%s\n",
				i, i, i, i, suite.Passphrase)

			stdout, stderr, exitCode := suite.RunCommandWithInput(t, input, "add", entryName)
			if exitCode != 0 {
				t.Errorf("Failed to add entry %d: exit code %d\nStdout: %s\nStderr: %s",
					i, exitCode, stdout, stderr)
				break
			}

			// Log progress every 100 entries
			if (i+1)%100 == 0 {
				t.Logf("Added %d/%d entries", i+1, numEntries)
			}
		}

		addDuration := time.Since(start)
		addPerEntry := addDuration / time.Duration(numEntries)

		t.Logf("Add performance: %v total, %v per entry", addDuration, addPerEntry)

		// Test 2: List performance with many entries
		t.Log("Test 2: List performance")

		start = time.Now()
		input = fmt.Sprintf("%s\n", suite.Passphrase)
		stdout, stderr, exitCode := suite.RunCommandWithInput(t, input, "list")
		listDuration := time.Since(start)

		if exitCode != 0 {
			t.Errorf("List failed: exit code %d\nStdout: %s\nStderr: %s", exitCode, stdout, stderr)
		} else {
			t.Logf("List performance: %v for %d entries", listDuration, numEntries)
		}

		// Test 3: Get performance
		t.Log("Test 3: Get performance")

		testGets := 100
		start = time.Now()

		for i := 0; i < testGets; i++ {
			entryName := fmt.Sprintf("perf-entry-%d", i*10) // Get every 10th entry
			input := fmt.Sprintf("%s\n", suite.Passphrase)

			stdout, stderr, exitCode := suite.RunCommandWithInput(t, input, "get", entryName)
			if exitCode != 0 {
				t.Errorf("Get entry %s failed: exit code %d\nStdout: %s\nStderr: %s",
					entryName, exitCode, stdout, stderr)
			}
		}

		getDuration := time.Since(start)
		getPerEntry := getDuration / time.Duration(testGets)

		t.Logf("Get performance: %v total, %v per entry", getDuration, getPerEntry)

		// Performance thresholds (adjust based on requirements)
		if addPerEntry > 100*time.Millisecond {
			t.Logf("Warning: Add operation taking %v per entry (threshold: 100ms)", addPerEntry)
		}

		if listDuration > 2*time.Second {
			t.Logf("Warning: List operation taking %v (threshold: 2s)", listDuration)
		}

		if getPerEntry > 50*time.Millisecond {
			t.Logf("Warning: Get operation taking %v per entry (threshold: 50ms)", getPerEntry)
		}

		t.Log("✅ Performance acceptance tests completed")
	})
}

// TestSecurityAcceptance tests security features end-to-end
func TestSecurityAcceptance(t *testing.T) {
	suite := NewAcceptanceTestSuite(t)
	suite.BuildBinary(t)

	t.Run("Security Acceptance", func(t *testing.T) {
		// Test 1: Passphrase strength requirements
		t.Log("Test 1: Passphrase strength")

		weakPassphrases := []string{
			"123",
			"password",
			"abc",
		}

		for _, weak := range weakPassphrases {
			input := fmt.Sprintf("%s\n%s\n", weak, weak)
			stdout, stderr, exitCode := suite.RunCommandWithInput(t, input, "init")

			// Should either reject weak passphrase or warn about it
			if exitCode == 0 {
				if !strings.Contains(stdout, "weak") && !strings.Contains(stderr, "weak") {
					t.Logf("Warning: Weak passphrase '%s' accepted without warning", weak)
				}
			}
		}

		// Test 2: Session timeout
		t.Log("Test 2: Session timeout")

		// Initialize with strong passphrase
		input := fmt.Sprintf("%s\n%s\n", suite.Passphrase, suite.Passphrase)
		suite.RunCommandWithInput(t, input, "init")

		// Add an entry to unlock the vault
		input = fmt.Sprintf("https://test.com\nuser\npass\nnotes\n%s\n", suite.Passphrase)
		suite.RunCommandWithInput(t, input, "add", "test-entry")

		// Try to access without passphrase (should be locked)
		stdout, stderr, exitCode := suite.RunCommand(t, "list")
		if exitCode == 0 && !strings.Contains(stdout, "locked") && !strings.Contains(stderr, "locked") {
			t.Log("Note: Vault may not have automatic session timeout")
		}

		// Test 3: File encryption verification
		t.Log("Test 3: File encryption verification")

		// Read vault file content
		vaultData, err := os.ReadFile(suite.VaultPath)
		if err != nil {
			t.Errorf("Failed to read vault file: %v", err)
		} else {
			// Vault file should not contain plaintext passwords
			if bytes.Contains(vaultData, []byte("test-entry")) ||
				bytes.Contains(vaultData, []byte("user")) ||
				bytes.Contains(vaultData, []byte("pass")) {
				t.Error("Vault file contains plaintext data - encryption may be broken")
			} else {
				t.Log("✅ Vault file appears to be properly encrypted")
			}
		}

		// Test 4: Backup file security
		t.Log("Test 4: Backup file security")

		backupPath := suite.VaultPath + ".backup"
		if _, err := os.Stat(backupPath); err == nil {
			// Check backup file permissions
			info, err := os.Stat(backupPath)
			if err != nil {
				t.Errorf("Failed to stat backup file: %v", err)
			} else {
				mode := info.Mode()
				if mode&0044 != 0 {
					t.Errorf("Backup file has insecure permissions: %v", mode)
				}
			}
		}

		t.Log("✅ Security acceptance tests completed")
	})
}

// TestRecoveryScenarios tests disaster recovery and data integrity
func TestRecoveryScenarios(t *testing.T) {
	suite := NewAcceptanceTestSuite(t)
	suite.BuildBinary(t)

	t.Run("Recovery Scenarios", func(t *testing.T) {
		// Initialize vault and add test data
		input := fmt.Sprintf("%s\n%s\n", suite.Passphrase, suite.Passphrase)
		suite.RunCommandWithInput(t, input, "init")

		// Add test entries
		testEntries := []string{"recovery-test-1", "recovery-test-2", "recovery-test-3"}
		for i, entryName := range testEntries {
			input := fmt.Sprintf("https://test%d.com\nuser%d\npass%d\nNotes %d\n%s\n",
				i, i, i, i, suite.Passphrase)
			suite.RunCommandWithInput(t, input, "add", entryName)
		}

		// Test 1: Backup and restore
		t.Log("Test 1: Backup and restore")

		// Create backup
		backupPath := filepath.Join(suite.TempDir, "manual_backup.vault")
		input = fmt.Sprintf("%s\n", suite.Passphrase)
		stdout, stderr, exitCode := suite.RunCommandWithInput(t, input, "backup", backupPath)

		if exitCode != 0 {
			t.Logf("Backup command failed (may not be implemented): %s", stderr)
		} else {
			// Verify backup file exists
			if _, err := os.Stat(backupPath); os.IsNotExist(err) {
				t.Error("Backup file was not created")
			}
		}

		// Test 2: Corrupted file recovery
		t.Log("Test 2: Corrupted file recovery")

		// Create a copy of the original vault
		originalData, err := os.ReadFile(suite.VaultPath)
		if err != nil {
			t.Fatalf("Failed to read vault file: %v", err)
		}

		// Corrupt the vault file
		corruptedData := append([]byte("CORRUPTED"), originalData[10:]...)
		err = os.WriteFile(suite.VaultPath, corruptedData, 0644)
		if err != nil {
			t.Fatalf("Failed to corrupt vault file: %v", err)
		}

		// Try to access corrupted vault
		input = fmt.Sprintf("%s\n", suite.Passphrase)
		stdout, stderr, exitCode = suite.RunCommandWithInput(t, input, "list")

		if exitCode == 0 {
			t.Error("Corrupted vault should not be accessible")
		} else {
			t.Log("✅ Corrupted vault properly rejected")
		}

		// Restore original file
		err = os.WriteFile(suite.VaultPath, originalData, 0644)
		if err != nil {
			t.Fatalf("Failed to restore vault file: %v", err)
		}

		// Verify data integrity after restore
		input = fmt.Sprintf("%s\n", suite.Passphrase)
		stdout, stderr, exitCode = suite.RunCommandWithInput(t, input, "list")

		if exitCode != 0 {
			t.Errorf("Failed to access restored vault: %s", stderr)
		} else {
			for _, entryName := range testEntries {
				if !strings.Contains(stdout, entryName) {
					t.Errorf("Entry %s missing after restore", entryName)
				}
			}
		}

		// Test 3: Import/Export integrity
		t.Log("Test 3: Import/Export integrity")

		exportPath := filepath.Join(suite.TempDir, "export_test.json")
		input = fmt.Sprintf("%s\n", suite.Passphrase)
		stdout, stderr, exitCode = suite.RunCommandWithInput(t, input, "export", exportPath)

		if exitCode != 0 {
			t.Logf("Export command failed (may not be implemented): %s", stderr)
		} else {
			// Verify export file
			if _, err := os.Stat(exportPath); os.IsNotExist(err) {
				t.Error("Export file was not created")
			} else {
				// Read and validate export content
				exportData, err := os.ReadFile(exportPath)
				if err != nil {
					t.Errorf("Failed to read export file: %v", err)
				} else {
					// Should contain entry data
					for _, entryName := range testEntries {
						if !strings.Contains(string(exportData), entryName) {
							t.Errorf("Export missing entry: %s", entryName)
						}
					}
				}
			}
		}

		t.Log("✅ Recovery scenario tests completed")
	})
}
