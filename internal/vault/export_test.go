package vault

import (
	"testing"

	"github.com/vault-cli/vault/internal/domain"
)

// TestExportVaultEncrypted tests encrypted export functionality
func TestExportVaultEncrypted(t *testing.T) {
	entries := []*domain.Entry{
		{
			Name:     "test-entry",
			Username: "user@example.com",
			Secret:   []byte("secret123"),
			URL:      "https://example.com",
			Tags:     []string{"test"},
			Notes:    "Test notes",
		},
	}
	passphrase := "test-passphrase-123"

	// Test encrypted export
	data, err := ExportVault(entries, passphrase, true)
	if err != nil {
		t.Fatalf("ExportVault failed: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("Export returned empty data")
	}

	// Verify JSON structure
	if !containsString(string(data), "\"encrypted\": true") {
		t.Error("Export should indicate encrypted format")
	}

	if !containsString(string(data), "\"version\"") {
		t.Error("Export should include version field")
	}

	if !containsString(string(data), "\"salt\"") {
		t.Error("Export should include salt field")
	}

	if !containsString(string(data), "\"nonce\"") {
		t.Error("Export should include nonce field")
	}

	if !containsString(string(data), "\"tag\"") {
		t.Error("Export should include tag field for GCM")
	}

	// Verify no plaintext secrets
	if containsString(string(data), "secret123") {
		t.Error("Export should not contain plaintext secrets")
	}

	if containsString(string(data), "user@example.com") {
		t.Error("Export should not contain plaintext usernames")
	}
}

// TestExportVaultPlaintext tests plaintext export functionality
func TestExportVaultPlaintext(t *testing.T) {
	entries := []*domain.Entry{
		{
			Name:     "test-entry",
			Username: "user@example.com",
			Secret:   []byte("secret123"),
			URL:      "https://example.com",
			Tags:     []string{"test", "demo"},
			Notes:    "Test notes",
		},
		{
			Name:     "second-entry",
			Username: "admin",
			Secret:   []byte("admin-pass"),
		},
	}

	// Test plaintext export (passphrase is ignored for plaintext)
	data, err := ExportVault(entries, "", false)
	if err != nil {
		t.Fatalf("ExportVault failed: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("Export returned empty data")
	}

	// Verify JSON structure
	if !containsString(string(data), "\"encrypted\": false") {
		t.Error("Export should indicate plaintext format")
	}

	if !containsString(string(data), "\"entries\"") {
		t.Error("Export should include entries array")
	}

	// Verify entries are in plaintext
	if !containsString(string(data), "test-entry") {
		t.Error("Export should contain entry name")
	}

	if !containsString(string(data), "user@example.com") {
		t.Error("Export should contain username in plaintext")
	}

	if !containsString(string(data), "secret123") {
		t.Error("Export should contain secret in plaintext")
	}

	if !containsString(string(data), "https://example.com") {
		t.Error("Export should contain URL")
	}

	if !containsString(string(data), "test") {
		t.Error("Export should contain tags")
	}
}

// TestImportVaultEncrypted tests encrypted import functionality
func TestImportVaultEncrypted(t *testing.T) {
	entries := []*domain.Entry{
		{
			Name:     "import-test",
			Username: "import@example.com",
			Secret:   []byte("import-secret"),
			URL:      "https://import.com",
			Tags:     []string{"imported"},
			Notes:    "Import test notes",
			TOTPSeed: "TESTSEED123",
		},
	}
	passphrase := "import-passphrase-456"

	// Export first
	exported, err := ExportVault(entries, passphrase, true)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Import back
	imported, err := ImportVault(exported, passphrase)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	if len(imported) != len(entries) {
		t.Errorf("Expected %d entries, got %d", len(entries), len(imported))
	}

	// Verify imported entry
	if imported[0].Name != entries[0].Name {
		t.Errorf("Name mismatch: expected %s, got %s", entries[0].Name, imported[0].Name)
	}

	if imported[0].Username != entries[0].Username {
		t.Errorf("Username mismatch: expected %s, got %s", entries[0].Username, imported[0].Username)
	}

	if string(imported[0].Secret) != string(entries[0].Secret) {
		t.Error("Secret mismatch after import")
	}

	if imported[0].URL != entries[0].URL {
		t.Errorf("URL mismatch: expected %s, got %s", entries[0].URL, imported[0].URL)
	}

	if len(imported[0].Tags) != len(entries[0].Tags) {
		t.Error("Tags count mismatch")
	}

	if imported[0].TOTPSeed != entries[0].TOTPSeed {
		t.Error("TOTP seed mismatch")
	}
}

// TestImportVaultWrongPassphrase tests import with incorrect passphrase
func TestImportVaultWrongPassphrase(t *testing.T) {
	entries := []*domain.Entry{
		{Name: "test", Secret: []byte("secret")},
	}
	correctPass := "correct-passphrase"
	wrongPass := "wrong-passphrase"

	// Export with correct passphrase
	exported, err := ExportVault(entries, correctPass, true)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Try to import with wrong passphrase
	_, err = ImportVault(exported, wrongPass)
	if err == nil {
		t.Error("Import should fail with wrong passphrase")
	}

	if !containsString(err.Error(), "decrypt") {
		t.Errorf("Error should mention decryption: %v", err)
	}
}

// TestImportVaultPlaintext tests plaintext import
func TestImportVaultPlaintext(t *testing.T) {
	entries := []*domain.Entry{
		{
			Name:     "plain-entry",
			Username: "plain@example.com",
			Secret:   []byte("plain-secret"),
		},
	}

	// Export as plaintext
	exported, err := ExportVault(entries, "", false)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Import (no passphrase needed)
	imported, err := ImportVault(exported, "")
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	if len(imported) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(imported))
	}

	if imported[0].Name != "plain-entry" {
		t.Errorf("Name mismatch: %s", imported[0].Name)
	}
}

// TestExportImportMultipleEntries tests export/import with many entries
func TestExportImportMultipleEntries(t *testing.T) {
	entries := make([]*domain.Entry, 100)
	for i := 0; i < 100; i++ {
		entries[i] = &domain.Entry{
			Name:     generateName("entry", i),
			Username: generateName("user", i),
			Secret:   []byte(generateName("secret", i)),
		}
	}

	passphrase := "multi-entry-pass"

	// Export
	exported, err := ExportVault(entries, passphrase, true)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Import
	imported, err := ImportVault(exported, passphrase)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	if len(imported) != len(entries) {
		t.Errorf("Expected %d entries, got %d", len(entries), len(imported))
	}

	// Spot check a few entries
	for i := 0; i < 10; i++ {
		if imported[i].Name != entries[i].Name {
			t.Errorf("Entry %d name mismatch", i)
		}
	}
}

// TestIsEncryptedExport tests encrypted export detection
func TestIsEncryptedExport(t *testing.T) {
	entries := []*domain.Entry{{Name: "test", Secret: []byte("secret")}}

	// Test encrypted
	encrypted, _ := ExportVault(entries, "pass", true)
	if !IsEncryptedExport(encrypted) {
		t.Error("Should detect encrypted export")
	}

	// Test plaintext
	plaintext, _ := ExportVault(entries, "", false)
	if IsEncryptedExport(plaintext) {
		t.Error("Should not detect plaintext as encrypted")
	}
}

// TestExportEmptyEntries tests export with no entries
func TestExportEmptyEntries(t *testing.T) {
	var entries []*domain.Entry

	// Should handle empty list gracefully
	data, err := ExportVault(entries, "pass", true)
	if err != nil {
		t.Fatalf("Export should handle empty entries: %v", err)
	}

	// Import back
	imported, err := ImportVault(data, "pass")
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	if len(imported) != 0 {
		t.Errorf("Expected 0 entries, got %d", len(imported))
	}
}

// Helper functions
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func generateName(prefix string, index int) string {
	return prefix + "-" + itoa(index)
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
