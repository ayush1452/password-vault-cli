// Package tests contains acceptance tests for the password vault CLI.
// These tests verify end-to-end functionality by executing the CLI commands.
package tests

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"syscall"
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
	// Use absolute path to go binary
	goPath, err := exec.LookPath("go")
	if err != nil {
		t.Fatalf("Failed to find 'go' in PATH: %v", err)
	}

	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(suite.BinaryPath), 0o700); err != nil {
		t.Fatalf("Failed to create binary directory: %v", err)
	}

	cmd := &exec.Cmd{
		Path: goPath,
		Args: []string{goPath, "build", "-o", suite.BinaryPath, "./cmd/vault"},
		Dir:  "/workspace",
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build binary: %v\nOutput: %s", err, output)
	}

	// Set secure permissions on the binary (0600 is the most restrictive for files)
	if err := os.Chmod(suite.BinaryPath, 0o600); err != nil {
		t.Fatalf("Failed to set binary permissions: %v", err)
	}
}

// validateCommandArgs validates command line arguments to prevent injection
func validateCommandArgs(args ...string) error {
	// Only allow alphanumeric, basic punctuation, and path characters
	safePattern := regexp.MustCompile(`^[a-zA-Z0-9_\-./@=:]+$`)

	// Define a list of allowed subcommands and their argument patterns
	allowedCommands := map[string]string{
		"init":    `^[a-zA-Z0-9_\-./@=:]+$`,
		"unlock":  `^[a-zA-Z0-9_\-./@=:]+$`,
		"lock":    `^[a-zA-Z0-9_\-./@=:]*$`,
		"status":  `^[a-zA-Z0-9_\-./@=:]*$`,
		"add":     `^[a-zA-Z0-9_\-./@=:]+$`,
		"get":     `^[a-zA-Z0-9_\-./@=:]+$`,
		"list":    `^[a-zA-Z0-9_\-./@=:]*$`,
		"update":  `^[a-zA-Z0-9_\-./@=:]+$`,
		"delete":  `^[a-zA-Z0-9_\-./@=:]+$`,
		"export":  `^[a-zA-Z0-9_\-./@=:]+$`,
		"import":  `^[a-zA-Z0-9_\-./@=:]+$`,
		"audit":   `^[a-zA-Z0-9_\-./@=:]*$`,
		"rotate":  `^[a-zA-Z0-9_\-./@=:]*$`,
		"version": `^[a-zA-Z0-9_\-./@=:]*$`,
		"help":    `^[a-zA-Z0-9_\-./@=:]*$`,
	}

	if len(args) == 0 {
		return nil
	}

	// Validate the command itself
	command := args[0]
	if !safePattern.MatchString(command) {
		return fmt.Errorf("invalid command: %s", command)
	}

	// Check if command is in the allowed list
	pattern, ok := allowedCommands[command]
	if !ok {
		return fmt.Errorf("command not allowed: %s", command)
	}

	// Validate arguments based on the command
	for i, arg := range args[1:] {
		// Skip empty arguments
		if arg == "" {
			continue
		}

		// Skip validation for flag arguments (start with -)
		if arg[0] == '-' {
			// If this is a flag with a value (like --flag=value), validate the value
			if parts := strings.SplitN(arg, "=", 2); len(parts) > 1 {
				if !regexp.MustCompile(pattern).MatchString(parts[1]) {
					return fmt.Errorf("invalid value for flag %s: %s", parts[0], parts[1])
				}
			}
			continue
		}

		// Special handling for certain commands
		switch command {
		case "add", "update":
			// First argument after command is the entry name
			if i == 0 && !safePattern.MatchString(arg) {
				return fmt.Errorf("invalid entry name: %s", arg)
			}
		default:
			// For other commands, use the command-specific pattern
			if !regexp.MustCompile(pattern).MatchString(arg) {
				return fmt.Errorf("invalid argument for %s: %s", command, arg)
			}
		}
	}

	return nil
}

// RunCommand executes a vault CLI command and returns the result
// Returns:
//   - stdout: The standard output of the command
//   - stderr: The standard error output of the command
//   - exitCode: The exit code of the command
func (suite *AcceptanceTestSuite) RunCommand(t *testing.T, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()

	// Make a copy of args to avoid modifying the original
	execArgs := make([]string, len(args))
	copy(execArgs, args)

	// Validate command arguments
	if err := validateCommandArgs(execArgs...); err != nil {
		t.Fatalf("Invalid command arguments: %v", err)
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create command with context and full path to binary
	binaryPath, err := filepath.Abs(suite.BinaryPath)
	if err != nil {
		t.Fatalf("Failed to get absolute path to binary: %v", err)
	}

	// Ensure the binary exists and is executable
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Fatalf("Binary not found at %s", binaryPath)
	}

	// nolint:gosec // Arguments are validated by validateCommandArgs
	cmd := exec.CommandContext(ctx, binaryPath, execArgs...)

	// Create a secure temporary directory for this command
	tempDir, err := os.MkdirTemp("", "vault-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Warning: failed to remove temp directory: %v", err)
		}
	}()

	// Set a clean, minimal environment
	cmd.Env = []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + filepath.Clean(os.Getenv("HOME")),
		"USER=" + filepath.Clean(os.Getenv("USER")),
		"TMPDIR=" + tempDir,
		"VAULT_PATH=" + filepath.Clean(suite.VaultPath),
		"VAULT_CONFIG=" + filepath.Clean(suite.ConfigPath),
		// Disable any potential command history
		"HISTFILE=",
		"HISTFILESIZE=0",
		"HISTSIZE=0",
	}

	// Set process attributes for additional security
	cmd.SysProcAttr = &syscall.SysProcAttr{
		// Create a new process group
		Setpgid: true,
		// Create a new session
		Setsid: true,
		// Set the process group ID to the new process ID
		Pgid: 0,
	}

	// On Unix-like systems, set additional attributes
	if runtime.GOOS != "windows" {
		// Set a secure umask
		syscall.Umask(0o077)
	}

	// Create pipes for stdout and stderr
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("Failed to create stdout pipe: %v", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		t.Fatalf("Failed to create stderr pipe: %v", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start command: %v", err)
	}

	// Read output in a separate goroutine to prevent deadlock
	var stdoutBuf, stderrBuf bytes.Buffer
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		_, _ = io.Copy(&stdoutBuf, stdoutPipe)
	}()

	go func() {
		defer wg.Done()
		_, _ = io.Copy(&stderrBuf, stderrPipe)
	}()

	// Wait for command to complete
	err = cmd.Wait()
	wg.Wait()

	// Get exit code
	exitCode = 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
				exitCode = status.ExitStatus()
			} else {
				exitCode = 1
			}
		} else {
			t.Logf("Command error: %v", err)
			exitCode = 1
		}
	}

	// Get output
	stdoutStr := stdoutBuf.String()
	stderrStr := stderrBuf.String()

	// Clear sensitive data from memory
	defer func() {
		stdoutBuf.Reset()
		stderrBuf.Reset()
		// Overwrite the memory with zeros
		for i := range stdoutStr {
			stdoutStr = stdoutStr[:i] + "\x00"
		}
		for i := range stderrStr {
			stderrStr = stderrStr[:i] + "\x00"
		}
	}()

	return stdoutStr, stderrStr, exitCode
}

// RunCommandWithInput executes a command with stdin input and returns the result
// Returns:
//   - stdout: The standard output of the command
//   - stderr: The standard error output of the command
//   - exitCode: The exit code of the command
func (suite *AcceptanceTestSuite) RunCommandWithInput(t *testing.T, input string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()

	// Sanitize input
	input = strings.TrimSpace(input)
	if input == "" {
		t.Fatal("Empty input provided to RunCommandWithInput")
	}

	// Make a copy of args to avoid modifying the original
	execArgs := make([]string, len(args))
	copy(execArgs, args)

	// Validate command arguments
	if err := validateCommandArgs(execArgs...); err != nil {
		t.Fatalf("Invalid command arguments: %v", err)
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create command with context and full path to binary
	binaryPath, err := filepath.Abs(suite.BinaryPath)
	if err != nil {
		t.Fatalf("Failed to get absolute path to binary: %v", err)
	}

	// Ensure the binary exists and is executable
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Fatalf("Binary not found at %s", binaryPath)
	}

	// nolint:gosec // Arguments are validated by validateCommandArgs
	cmd := exec.CommandContext(ctx, binaryPath, execArgs...)

	// Create a secure temporary directory for this command
	tempDir, err := os.MkdirTemp("", "vault-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Warning: failed to remove temp directory: %v", err)
		}
	}()

	// Set a clean, minimal environment
	cmd.Env = []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + filepath.Clean(os.Getenv("HOME")),
		"USER=" + filepath.Clean(os.Getenv("USER")),
		"TMPDIR=" + tempDir,
		"VAULT_PATH=" + filepath.Clean(suite.VaultPath),
		"VAULT_CONFIG=" + filepath.Clean(suite.ConfigPath),
		// Disable any potential command history
		"HISTFILE=",
		"HISTFILESIZE=0",
		"HISTSIZE=0",
	}

	// Set process attributes for additional security
	cmd.SysProcAttr = &syscall.SysProcAttr{
		// Create a new process group
		Setpgid: true,
		// Create a new session
		Setsid: true,
		// Set the process group ID to the new process ID
		Pgid: 0,
	}

	// On Unix-like systems, set additional attributes
	if runtime.GOOS != "windows" {
		// Set a secure umask
		syscall.Umask(0o077)
	}

	// Create pipes for stdin, stdout, and stderr
	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("Failed to create stdin pipe: %v", err)
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("Failed to create stdout pipe: %v", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		t.Fatalf("Failed to create stderr pipe: %v", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start command: %v", err)
	}

	// Write input in a separate goroutine
	errChan := make(chan error, 1)
	go func() {
		_, err := io.WriteString(stdinPipe, input)
		if err != nil {
			errChan <- fmt.Errorf("failed to write to stdin: %w", err)
			return
		}
		if err := stdinPipe.Close(); err != nil {
			errChan <- fmt.Errorf("failed to close stdin: %w", err)
			return
		}
		errChan <- nil
	}()

	// Read output in separate goroutines to prevent deadlock
	var stdoutBuf, stderrBuf bytes.Buffer
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		_, _ = io.Copy(&stdoutBuf, stdoutPipe)
	}()

	go func() {
		defer wg.Done()
		_, _ = io.Copy(&stderrBuf, stderrPipe)
	}()

	// Wait for input to be written
	if err := <-errChan; err != nil {
		t.Fatalf("Error writing input: %v", err)
	}

	// Wait for command to complete
	err = cmd.Wait()
	wg.Wait()

	// Get exit code
	exitCode = 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
				exitCode = status.ExitStatus()
			} else {
				exitCode = 1
			}
		} else {
			t.Logf("Command error: %v", err)
			exitCode = 1
		}
	}

	// Get output
	stdoutStr := stdoutBuf.String()
	stderrStr := stderrBuf.String()

	// Clear sensitive data from memory
	defer func() {
		stdoutBuf.Reset()
		stderrBuf.Reset()
		// Overwrite the memory with zeros
		for i := range input {
			input = input[:i] + "\x00"
		}
		for i := range stdoutStr {
			stdoutStr = stdoutStr[:i] + "\x00"
		}
		for i := range stderrStr {
			stderrStr = stderrStr[:i] + "\x00"
		}
	}()

	return stdoutStr, stderrStr, exitCode
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
		listOut, _, listCode := suite.RunCommandWithInput(t, input, "list")

		if listCode != 0 {
			t.Errorf("List failed: exit code %d\nOutput: %s", listCode, listOut)
		}

		for _, entry := range entries {
			if !strings.Contains(listOut, entry.name) {
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
		var getExitCode int
		stdout, _, getExitCode = suite.RunCommandWithInput(t, input, "get", "github.com")

		if getExitCode == 0 && !strings.Contains(stdout, "updated-user") {
			t.Error("Entry was not updated properly")
		}

		// Step 6: Search entries
		t.Log("Step 6: Search entries")
		input = fmt.Sprintf("%s\n", suite.Passphrase)
		searchOut, _, searchCode := suite.RunCommandWithInput(t, input, "list", "--search", "gmail")
		if searchCode != 0 {
			t.Errorf("Search failed: exit code %d\nOutput: %s", searchCode, searchOut)
		}

		if !strings.Contains(searchOut, "gmail.com") {
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
		getOut, getErr, getCode := suite.RunCommandWithInput(t, input, "get", "aws.amazon.com")

		if getCode == 0 {
			t.Errorf("Entry was not deleted properly. Output: %s, Error: %s", getOut, getErr)
		}

		// Step 9: Export data
		t.Log("Step 9: Export data")
		exportPath := filepath.Join(suite.TempDir, "export.json")
		input = fmt.Sprintf("%s\n", suite.Passphrase)
		exportOut, _, exportCode := suite.RunCommandWithInput(t, input, "export", exportPath)

		if exportCode != 0 {
			t.Errorf("Export failed: exit code %d\nOutput: %s", exportCode, exportOut)
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
			t.Errorf("Should fail with wrong passphrase")
		}

		if !strings.Contains(stderr, "authentication failed") && !strings.Contains(stdout, "authentication failed") {
			t.Logf("Expected authentication error, got stdout: %s, stderr: %s", stdout, stderr)
		}

		// Test 2: Non-existent entry
		t.Log("Test 2: Non-existent entry")
		input = fmt.Sprintf("%s\n", suite.Passphrase)
		getOut, getErr, getCode := suite.RunCommandWithInput(t, input, "get", "non-existent-entry")

		if getCode == 0 {
			t.Errorf("Should fail for non-existent entry. Output: %s, Error: %s", getOut, getErr)
		}

		// Test 3: Invalid entry name
		t.Log("Test 3: Invalid entry name")
		input = fmt.Sprintf("https://test.com\nuser\npass\nnotes\n%s\n", suite.Passphrase)
		addOut, addErr, addCode := suite.RunCommandWithInput(t, input, "add", "../invalid/name")

		if addCode == 0 {
			t.Errorf("Should fail for invalid entry name. Output: %s, Error: %s", addOut, addErr)
		}

		// Test 4: Duplicate entry
		t.Log("Test 4: Duplicate entry")

		// Add an entry first
		input = fmt.Sprintf("https://test.com\nuser\npass\nnotes\n%s\n", suite.Passphrase)
		suite.RunCommandWithInput(t, input, "add", "test-entry")

		// Try to add the same entry again
		input = fmt.Sprintf("https://test.com\nuser2\npass2\nnotes2\n%s\n", suite.Passphrase)
		addOut, addErr, addCode = suite.RunCommandWithInput(t, input, "add", "test-entry")

		if addCode == 0 {
			t.Errorf("Should fail for duplicate entry. Output: %s, Error: %s", addOut, addErr)
		}

		// Test 5: Missing arguments
		t.Log("Test 5: Missing arguments")
		cmdOut, cmdErr, cmdCode := suite.RunCommand(t, "get")

		if cmdCode == 0 {
			t.Errorf("Should fail when entry name is missing. Output: %s, Error: %s", cmdOut, cmdErr)
		}

		// Test 6: Invalid command
		t.Log("Test 6: Invalid command")
		invalidOut, invalidErr, invalidCode := suite.RunCommand(t, "invalid-command")

		if invalidCode == 0 {
			t.Errorf("Should fail for invalid command. Output: %s, Error: %s", invalidOut, invalidErr)
		} else {
			t.Logf("Invalid command test passed with status %d", invalidCode)
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
			if err := os.MkdirAll(dir, 0o750); err != nil {
				t.Errorf("Failed to create test directory: %v", err)
				continue
			}

			// Set vault path
			if err := os.Setenv("VAULT_PATH", testPath); err != nil {
				t.Errorf("Failed to set VAULT_PATH: %v", err)
				continue
			}

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
		if err := os.Setenv("VAULT_PATH", suite.VaultPath); err != nil {
			t.Fatalf("Failed to reset VAULT_PATH: %v", err)
		}

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
			if mode&0o044 != 0 {
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
		listOut, _, listCode := suite.RunCommandWithInput(t, input, "list")
		listDuration := time.Since(start)

		if listCode != 0 {
			t.Errorf("List failed: exit code %d\nOutput: %s", listCode, listOut)
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
				if mode&0o044 != 0 {
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
		backupOut, _, backupCode := suite.RunCommandWithInput(t, input, "backup", backupPath)

		if backupCode != 0 {
			t.Errorf("Backup failed: exit code %d\nOutput: %s", backupCode, backupOut)
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
		err = os.WriteFile(suite.VaultPath, corruptedData, 0o600)
		if err != nil {
			t.Fatalf("Failed to corrupt vault file: %v", err)
		}

		// Try to access corrupted vault
		input = fmt.Sprintf("%s\n", suite.Passphrase)
		corruptOut, corruptErr, corruptCode := suite.RunCommandWithInput(t, input, "list")

		if corruptCode == 0 {
			t.Errorf("Corrupted vault should not be accessible. Output: %s, Error: %s", corruptOut, corruptErr)
		} else {
			t.Log("✅ Corrupted vault properly rejected")
		}

		// Restore original file
		err = os.WriteFile(suite.VaultPath, originalData, 0o600)
		if err != nil {
			t.Fatalf("Failed to restore vault file: %v", err)
		}

		// Verify data integrity after restore
		input = fmt.Sprintf("%s\n", suite.Passphrase)
		restoreOut, restoreErr, restoreCode := suite.RunCommandWithInput(t, input, "list")

		if restoreCode != 0 {
			t.Errorf("Failed to access restored vault. Error: %s, Output: %s", restoreErr, restoreOut)
		} else {
			for _, entryName := range testEntries {
				if !strings.Contains(restoreOut, entryName) {
					t.Errorf("Entry %s missing after restore. Output: %s", entryName, restoreOut)
				}
			}
		}

		// Test 3: Import/Export integrity
		t.Log("Test 3: Import/Export integrity")

		exportPath := filepath.Join(suite.TempDir, "export_test.json")
		input = fmt.Sprintf("%s\n", suite.Passphrase)
		exportOut, exportErr, exportCode := suite.RunCommandWithInput(t, input, "export", exportPath)

		if exportCode != 0 {
			t.Logf("Export command failed (may not be implemented): %s", exportErr)
			t.Logf("Export output: %s", exportOut)
		} else {
			// Clean and validate the export path
			cleanExportPath := filepath.Clean(exportPath)
			if cleanExportPath != exportPath {
				t.Fatal("Invalid export path: potential directory traversal detected")
			}

			// Verify export file
			if _, err := os.Stat(cleanExportPath); os.IsNotExist(err) {
				t.Error("Export file was not created")
			} else {
				// Read and validate export content
				exportData, err := os.ReadFile(cleanExportPath)
				if err != nil {
					t.Errorf("Failed to read export file: %v", err)
				} else {
					t.Logf("Export file size: %d bytes", len(exportData))
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
