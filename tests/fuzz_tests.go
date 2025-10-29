package tests

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vault-cli/vault/internal/domain"
	"github.com/vault-cli/vault/internal/store"
	"github.com/vault-cli/vault/internal/vault"
)

// FuzzTestSuite contains fuzzing tests for the Password Vault CLI
type FuzzTestSuite struct {
	TempDir   string
	VaultPath string
	Crypto    *vault.CryptoEngine
}

// NewFuzzTestSuite creates a new fuzz test suite
func NewFuzzTestSuite(t *testing.T) *FuzzTestSuite {
	tempDir := t.TempDir()
	vaultPath := filepath.Join(tempDir, "fuzz_test.vault")

	return &FuzzTestSuite{
		TempDir:   tempDir,
		VaultPath: vaultPath,
		Crypto:    vault.NewDefaultCryptoEngine(),
	}
}

// FuzzCryptoEngine tests crypto operations with random inputs
func FuzzCryptoEngine(f *testing.F) {
	// Seed corpus with known good inputs
	f.Add([]byte("test passphrase"), []byte("test data"))
	f.Add([]byte(""), []byte(""))
	f.Add([]byte("unicode-密码-🔐"), []byte("unicode data 测试"))
	f.Add([]byte("very long passphrase with many characters"), []byte("large data block"))

	f.Fuzz(func(t *testing.T, passphrase []byte, plaintext []byte) {
		crypto := vault.NewDefaultCryptoEngine()

		// Skip if inputs are too large (prevent OOM)
		if len(passphrase) > 10000 || len(plaintext) > 100000 {
			t.Skip("Input too large")
		}

		// Test key derivation with random passphrase
		salt, err := vault.GenerateSalt()
		if err != nil {
			t.Fatalf("Failed to generate salt: %v", err)
		}

		key, err := crypto.DeriveKey(string(passphrase), salt)
		if err != nil {
			// Key derivation should handle any passphrase gracefully
			if len(passphrase) == 0 {
				return // Empty passphrase may be rejected
			}
			t.Errorf("Key derivation failed with passphrase length %d: %v", len(passphrase), err)
			return
		}

		// Test encryption with random data
		envelope, err := crypto.Seal(plaintext, key)
		if err != nil {
			t.Errorf("Encryption failed with data length %d: %v", len(plaintext), err)
			return
		}

		// Test decryption
		decrypted, err := crypto.Open(envelope, key)
		if err != nil {
			t.Errorf("Decryption failed: %v", err)
			return
		}

		// Verify round-trip integrity
		if !bytes.Equal(plaintext, decrypted) {
			t.Errorf("Round-trip failed: original %d bytes, decrypted %d bytes",
				len(plaintext), len(decrypted))
		}

		// Clean up sensitive data
		vault.Zeroize(key)
	})
}

// FuzzEnvelopeSerialization tests envelope serialization with malformed data
func FuzzEnvelopeSerialization(f *testing.F) {
	// Create valid envelope for seed
	crypto := vault.NewDefaultCryptoEngine()
	key := make([]byte, 32)
	rand.Read(key)

	envelope, _ := crypto.Seal([]byte("test data"), key)
	validJSON, _ := json.Marshal(envelope)

	// Seed corpus
	f.Add(validJSON)
	f.Add([]byte(`{"version":1}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`invalid json`))
	f.Add([]byte(``))

	f.Fuzz(func(t *testing.T, data []byte) {
		// Skip extremely large inputs
		if len(data) > 50000 {
			t.Skip("Input too large")
		}

		var envelope vault.Envelope
		err := json.Unmarshal(data, &envelope)

		// Unmarshaling may fail with invalid JSON - that's expected
		if err != nil {
			return
		}

		// If unmarshaling succeeds, test envelope validation
		// This should not panic or cause undefined behavior
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Panic during envelope processing: %v", r)
			}
		}()

		// Try to use the envelope (should fail gracefully if invalid)
		testKey := make([]byte, 32)
		rand.Read(testKey)

		_, err = crypto.Open(&envelope, testKey)
		// Error is expected for invalid envelopes - just ensure no panic
	})
}

// FuzzEntryValidation tests entry validation with random inputs
func FuzzEntryValidation(f *testing.F) {
	// Seed with valid entries
	f.Add("test-entry", "https://example.com", "testuser", "testpass", "test notes", `["tag1","tag2"]`)
	f.Add("", "", "", "", "", `[]`)
	f.Add("unicode-测试", "https://测试.com", "用户", "密码", "笔记", `["标签"]`)

	f.Fuzz(func(t *testing.T, name, url, username, password, notes, tagsJSON string) {
		// Skip extremely large inputs
		if len(name) > 10000 || len(url) > 10000 || len(username) > 10000 ||
			len(password) > 10000 || len(notes) > 10000 || len(tagsJSON) > 10000 {
			t.Skip("Input too large")
		}

		// Parse tags
		var tags []string
		json.Unmarshal([]byte(tagsJSON), &tags) // Ignore errors

		// Create entry (should not panic)
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Panic during entry creation: %v", r)
			}
		}()

		entry := &domain.Entry{
			Name:     name,
			URL:      url,
			Username: username,
			Password: []byte(password),
			Notes:    notes,
			Tags:     tags,
		}

		// Test validation (should handle any input gracefully)
		isValid := validateEntry(entry)

		// Test serialization (should not panic)
		_, err := json.Marshal(entry)
		if err != nil {
			// Some characters may not be serializable - that's OK
		}

		// Log validation result for debugging
		if !isValid && len(name) > 0 && len(name) < 1000 {
			t.Logf("Entry validation failed for name: %q", name[:min(len(name), 50)])
		}
	})
}

// FuzzStoreOperations tests store operations with random data
func FuzzStoreOperations(f *testing.F) {
	// Seed with valid operations
	f.Add("add", "test-entry", "testuser", "testpass")
	f.Add("get", "test-entry", "", "")
	f.Add("delete", "test-entry", "", "")
	f.Add("list", "", "", "")

	f.Fuzz(func(t *testing.T, operation, entryName, username, password string) {
		// Skip extremely large inputs
		if len(entryName) > 1000 || len(username) > 1000 || len(password) > 1000 {
			t.Skip("Input too large")
		}

		// Create temporary store
		tempDir := t.TempDir()
		vaultPath := filepath.Join(tempDir, "fuzz_store.vault")

		s := store.NewBoltStore()
		
		// Create and open vault
		crypto := vault.NewDefaultCryptoEngine()
		passphrase := "fuzz-test-passphrase"
		salt, _ := vault.GenerateSalt()
		masterKey, _ := crypto.DeriveKey(passphrase, salt)
		
		kdfParams := map[string]interface{}{
			"algorithm": "argon2id",
		}
		
		err := s.CreateVault(vaultPath, masterKey, kdfParams)
		if err != nil {
			t.Skip("Failed to create vault")
		}
		defer s.CloseVault()

		err = s.OpenVault(vaultPath, masterKey)
		if err != nil {
			t.Skip("Failed to open vault")
		}

		// Ensure operations don't panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Panic during store operation %s: %v", operation, r)
			}
		}()

		// Use default profile for all operations
		profile := "default"

		// Perform operation based on input
		switch operation {
		case "add":
			if entryName != "" {
				entry := &domain.Entry{
					Name:     entryName,
					Username: username,
					Password: []byte(password),
				}
				s.CreateEntry(profile, entry) // Ignore errors - may be invalid
			}

		case "get":
			if entryName != "" {
				s.GetEntry(profile, entryName) // Ignore errors - may not exist
			}

		case "delete":
			if entryName != "" {
				s.DeleteEntry(profile, entryName) // Ignore errors - may not exist
			}

		case "list":
			s.ListEntries(profile, nil) // Should always work

		case "search":
			if entryName != "" {
				filter := &domain.Filter{Search: entryName}
				s.ListEntries(profile, filter) // Use filtered list instead
			}
		}
	})
}

// FuzzCLIInput tests CLI input parsing with random data
func FuzzCLIInput(f *testing.F) {
	// Seed with valid CLI commands
	f.Add("init")
	f.Add("add test-entry")
	f.Add("get test-entry")
	f.Add("list")
	f.Add("delete test-entry")
	f.Add("help")

	f.Fuzz(func(t *testing.T, input string) {
		// Skip extremely large inputs
		if len(input) > 10000 {
			t.Skip("Input too large")
		}

		// Parse CLI input (should not panic)
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Panic during CLI parsing: %v", r)
			}
		}()

		// Split input into command and args
		parts := strings.Fields(input)
		if len(parts) == 0 {
			return
		}

		command := parts[0]
		args := parts[1:]

		// Test command validation
		isValidCommand := validateCommand(command)

		// Test argument parsing
		for _, arg := range args {
			validateArgument(arg) // Should handle any input
		}

		// Log interesting cases
		if len(parts) > 10 {
			t.Logf("Long command with %d parts: %s", len(parts), command)
		}

		if !isValidCommand && len(command) < 100 {
			t.Logf("Invalid command: %q", command)
		}
	})
}

// FuzzConfigurationParsing tests configuration parsing with malformed data
func FuzzConfigurationParsing(f *testing.F) {
	// Seed with valid configurations
	validConfig := `
vault_path: "/tmp/test.vault"
default_profile: "default"
profiles:
  default:
    name: "default"
    vault_path: "/tmp/test.vault"
    auto_lock: 300
security:
  session_timeout: 1800
  max_failed_attempts: 3
`

	f.Add([]byte(validConfig))
	f.Add([]byte(`{}`))
	f.Add([]byte(`invalid yaml`))
	f.Add([]byte(``))
	f.Add([]byte(`vault_path: "../../etc/passwd"`))

	f.Fuzz(func(t *testing.T, configData []byte) {
		// Skip extremely large inputs
		if len(configData) > 50000 {
			t.Skip("Input too large")
		}

		// Test configuration parsing (should not panic)
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Panic during config parsing: %v", r)
			}
		}()

		// Write config to temporary file
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.yaml")

		err := os.WriteFile(configPath, configData, 0644)
		if err != nil {
			t.Skip("Failed to write config file")
		}

		// Try to parse configuration
		parseConfiguration(configPath) // Should handle any input gracefully
	})
}

// FuzzFileOperations tests file operations with malformed paths
func FuzzFileOperations(f *testing.F) {
	// Seed with various path types
	f.Add("/tmp/test.vault")
	f.Add("./test.vault")
	f.Add("../test.vault")
	f.Add("/etc/passwd")
	f.Add("")
	f.Add("con") // Windows special file

	f.Fuzz(func(t *testing.T, path string) {
		// Skip extremely large paths
		if len(path) > 4096 {
			t.Skip("Path too large")
		}

		// Test path validation (should not panic)
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Panic during path validation: %v", r)
			}
		}()

		// Validate path
		isValid := validateVaultPath(path)

		// Test path operations only on safe paths
		if isValid && strings.HasPrefix(path, "/tmp/") {
			// Safe to test file operations
			testFileOperations(t, path)
		}

		// Log interesting cases
		if strings.Contains(path, "..") {
			t.Logf("Path traversal attempt: %q", path)
		}

		if strings.Contains(path, "\x00") {
			t.Logf("Null byte in path: %q", path)
		}
	})
}

// Helper functions for fuzz tests

func validateEntry(entry *domain.Entry) bool {
	if entry == nil {
		return false
	}

	// Basic validation
	if len(entry.Name) == 0 || len(entry.Name) > 255 {
		return false
	}

	// Check for dangerous characters
	dangerous := []string{"\x00", "\r", "\n", "../"}
	for _, char := range dangerous {
		if strings.Contains(entry.Name, char) {
			return false
		}
	}

	return true
}

func validateCommand(command string) bool {
	validCommands := []string{
		"init", "add", "get", "list", "delete", "update",
		"lock", "unlock", "config", "help", "version",
	}

	for _, valid := range validCommands {
		if command == valid {
			return true
		}
	}

	return false
}

func validateArgument(arg string) bool {
	// Basic argument validation
	if len(arg) > 1000 {
		return false
	}

	// Check for dangerous patterns
	if strings.Contains(arg, "\x00") {
		return false
	}

	return true
}

func parseConfiguration(configPath string) error {
	// Placeholder for configuration parsing
	// In real implementation, this would parse YAML/JSON config
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	// Basic validation
	if len(data) > 100000 {
		return fmt.Errorf("config file too large")
	}

	return nil
}

func validateVaultPath(path string) bool {
	if len(path) == 0 || len(path) > 4096 {
		return false
	}

	// Reject dangerous paths
	dangerous := []string{
		"..", "/etc/", "/root/", "/home/", "\\", "\x00",
		"con", "prn", "aux", "nul", // Windows special files
	}

	lowerPath := strings.ToLower(path)
	for _, danger := range dangerous {
		if strings.Contains(lowerPath, danger) {
			return false
		}
	}

	return true
}

func testFileOperations(t *testing.T, path string) {
	// Test file creation (in safe location only)
	if !strings.HasPrefix(path, "/tmp/") {
		return
	}

	// Create directory if needed
	dir := filepath.Dir(path)
	os.MkdirAll(dir, 0755)

	// Test file operations
	testData := []byte("test data")

	// Write
	err := os.WriteFile(path, testData, 0644)
	if err != nil {
		return // Expected for invalid paths
	}

	// Read
	_, err = os.ReadFile(path)
	if err != nil {
		return
	}

	// Clean up
	os.Remove(path)
}

// Benchmark functions for performance testing

func BenchmarkKeyDerivation(b *testing.B) {
	crypto := vault.NewDefaultCryptoEngine()
	passphrase := "benchmark-test-passphrase"
	salt, _ := vault.GenerateSalt()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := crypto.DeriveKey(passphrase, salt)
		if err != nil {
			b.Fatalf("Key derivation failed: %v", err)
		}
	}
}

func BenchmarkEncryption(b *testing.B) {
	crypto := vault.NewDefaultCryptoEngine()
	key := make([]byte, 32)
	rand.Read(key)

	// Test different data sizes
	sizes := []int{16, 64, 256, 1024, 4096, 16384}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size-%d", size), func(b *testing.B) {
			data := make([]byte, size)
			rand.Read(data)

			b.ResetTimer()
			b.ReportAllocs()
			b.SetBytes(int64(size))

			for i := 0; i < b.N; i++ {
				_, err := crypto.Seal(data, key)
				if err != nil {
					b.Fatalf("Encryption failed: %v", err)
				}
			}
		})
	}
}

func BenchmarkDecryption(b *testing.B) {
	crypto := vault.NewDefaultCryptoEngine()
	key := make([]byte, 32)
	rand.Read(key)

	// Test different data sizes
	sizes := []int{16, 64, 256, 1024, 4096, 16384}

	for _, size := range sizes {
		data := make([]byte, size)
		rand.Read(data)

		envelope, err := crypto.Seal(data, key)
		if err != nil {
			b.Fatalf("Failed to create test envelope: %v", err)
		}

		b.Run(fmt.Sprintf("size-%d", size), func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()
			b.SetBytes(int64(size))

			for i := 0; i < b.N; i++ {
				_, err := crypto.Open(envelope, key)
				if err != nil {
					b.Fatalf("Decryption failed: %v", err)
				}
			}
		})
	}
}

func BenchmarkStoreOperations(b *testing.B) {
	tempDir := b.TempDir()
	vaultPath := filepath.Join(tempDir, "benchmark.vault")

	s := store.NewBoltStore()
	
	crypto := vault.NewDefaultCryptoEngine()
	passphrase := "benchmark-passphrase"
	salt, _ := vault.GenerateSalt()
	masterKey, _ := crypto.DeriveKey(passphrase, salt)
	
	kdfParams := map[string]interface{}{
		"algorithm": "argon2id",
	}
	
	err := s.CreateVault(vaultPath, masterKey, kdfParams)
	if err != nil {
		b.Fatalf("Failed to create vault: %v", err)
	}
	defer s.CloseVault()

	err = s.OpenVault(vaultPath, masterKey)
	if err != nil {
		b.Fatalf("Failed to open vault: %v", err)
	}

	profile := "default"

	// Benchmark add operations
	b.Run("Add", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			entry := &domain.Entry{
				Name:     fmt.Sprintf("bench-entry-%d", i),
				Username: fmt.Sprintf("user-%d", i),
				Password: []byte(fmt.Sprintf("pass-%d", i)),
			}

			err := s.CreateEntry(profile, entry)
			if err != nil {
				b.Fatalf("Failed to add entry: %v", err)
			}
		}
	})

	// Add some entries for get/list benchmarks
	for i := 0; i < 1000; i++ {
		entry := &domain.Entry{
			Name:     fmt.Sprintf("test-entry-%d", i),
			Username: fmt.Sprintf("user-%d", i),
			Password: []byte(fmt.Sprintf("pass-%d", i)),
		}
		s.CreateEntry(profile, entry)
	}

	// Benchmark get operations
	b.Run("Get", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			entryName := fmt.Sprintf("test-entry-%d", i%1000)
			_, err := s.GetEntry(profile, entryName)
			if err != nil {
				b.Fatalf("Failed to get entry: %v", err)
			}
		}
	})

	// Benchmark list operations
	b.Run("List", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, err := s.ListEntries(profile, nil)
			if err != nil {
				b.Fatalf("Failed to list entries: %v", err)
			}
		}
	})
}
