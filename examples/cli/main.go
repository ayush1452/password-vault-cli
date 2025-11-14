package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

func main() {
	fmt.Println("Password Vault CLI - Complete Demo")
	fmt.Println("==================================")

	// Create temporary directory for demo
	tempDir, err := os.MkdirTemp("", "vault_cli_demo_")
	if err != nil {
		log.Fatal("Failed to create temp directory:", err)
	}
	defer os.RemoveAll(tempDir)

	vaultPath := filepath.Join(tempDir, "demo.vault")
	configPath := filepath.Join(tempDir, "config.yaml")

	fmt.Printf("Demo vault: %s\n", vaultPath)
	fmt.Printf("Demo config: %s\n", configPath)

	// Build the vault binary
	fmt.Println("\n1. Building vault binary...")
	vaultBinaryPath := filepath.Join(tempDir, "vault")

	// Use absolute path to the Go binary for security
	goPath, err := exec.LookPath("go")
	if err != nil {
		log.Printf("Could not find 'go' in PATH: %v", err)
		return
	}

	buildCmd := &exec.Cmd{
		Path: goPath,
		Args: []string{"go", "build", "-o", vaultBinaryPath, "./cmd/vault"},
		Dir:  "/workspace",
		SysProcAttr: &syscall.SysProcAttr{
			Setpgid: true, // Prevent child processes from being killed when parent is killed
		},
	}

	output, err := buildCmd.CombinedOutput()
	if err != nil {
		log.Printf("Build output: %s\nFailed to build vault binary: %v", output, err)
		return
	}

	// Ensure the binary has secure permissions
	if err := os.Chmod(vaultBinaryPath, 0o600); err != nil {
		log.Printf("Failed to set binary permissions: %v", err)
		return
	}

	fmt.Println("✓ Vault binary built successfully")

	vaultBinary := filepath.Join(tempDir, "vault")

	// Demo commands
	commands := []struct {
		name string
		args []string
		desc string
	}{
		{
			name: "Show help",
			args: []string{"--help"},
			desc: "Display main help",
		},
		{
			name: "Show version",
			args: []string{"--version"},
			desc: "Display version information",
		},
		{
			name: "Initialize vault",
			args: []string{"init", "--vault", vaultPath, "--kdf-memory", "1024", "--kdf-iterations", "1"},
			desc: "Create a new vault with demo KDF parameters",
		},
		{
			name: "Doctor check (locked)",
			args: []string{"doctor", "--vault", vaultPath},
			desc: "Run security checks on locked vault",
		},
		{
			name: "List commands",
			args: []string{"--help"},
			desc: "Show available commands",
		},
	}

	for i, cmd := range commands {
		fmt.Printf("\n%d. %s\n", i+2, cmd.desc)
		fmt.Printf("Command: vault %s\n", joinArgs(cmd.args))

		// Secure command execution with proper argument handling
		execCmd := &exec.Cmd{
			Path: vaultBinary,
			Args: append([]string{filepath.Base(vaultBinary)}, cmd.args...),
			Dir:  tempDir,
			SysProcAttr: &syscall.SysProcAttr{
				Setpgid: true, // Prevent child processes from being killed when parent is killed
			},
		}

		// For init command, we need to provide input
		if cmd.name == "Initialize vault" {
			execCmd.Stdin = nil // In a real demo, this would be handled securely
			fmt.Println("Note: This would normally prompt for master passphrase")
		}

		output, err := execCmd.CombinedOutput()
		if err != nil {
			fmt.Printf("Command failed: %v\n", err)
		}
		fmt.Printf("Output:\n%s\n", string(output))

		time.Sleep(500 * time.Millisecond) // Brief pause between commands
	}

	fmt.Println("\n✓ CLI demo completed!")
	fmt.Printf("Vault binary available at: %s\n", vaultBinary)
	fmt.Println("\nTo continue testing manually:")
	fmt.Printf("1. cd %s\n", tempDir)
	fmt.Printf("2. ./vault init --vault demo.vault\n")
	fmt.Printf("3. ./vault unlock --vault demo.vault\n")
	fmt.Printf("4. ./vault add github --username user@example.com --secret-prompt\n")
	fmt.Printf("5. ./vault list --vault demo.vault\n")
	fmt.Printf("6. ./vault get github --vault demo.vault\n")
}

func joinArgs(args []string) string {
	result := ""
	for i, arg := range args {
		if i > 0 {
			result += " "
		}
		result += arg
	}
	return result
}
