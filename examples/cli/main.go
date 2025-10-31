package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
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
	buildCmd := exec.Command("go", "build", "-o", filepath.Join(tempDir, "vault"), "./cmd/vault")
	buildCmd.Dir = "/workspace"
	if output, err := buildCmd.CombinedOutput(); err != nil {
		log.Printf("Build output: %s", output)
		log.Fatal("Failed to build vault binary:", err)
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

		execCmd := exec.Command(vaultBinary, cmd.args...)
		execCmd.Dir = tempDir

		// For init command, we need to provide input
		if cmd.name == "Initialize vault" {
			execCmd.Stdin = nil // We'll need to handle this differently in a real demo
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
		if len(arg) == 0 || arg[0] != '-' {
			result += arg
		} else {
			result += arg
		}
	}
	return result
}
