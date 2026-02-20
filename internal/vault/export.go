package vault

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/vault-cli/vault/internal/domain"
)

// ExportFormat represents the export file format
type ExportFormat struct {
	Version   string        `json:"version"`
	Encrypted bool          `json:"encrypted"`
	Salt      string        `json:"salt,omitempty"`
	Nonce     string        `json:"nonce,omitempty"`
	Tag       string        `json:"tag,omitempty"`
	Data      string        `json:"data,omitempty"`
	Entries   []ExportEntry `json:"entries,omitempty"`
}

// ExportEntry represents an entry in plaintext export
type ExportEntry struct {
	Name     string   `json:"name"`
	Username string   `json:"username"`
	Secret   string   `json:"secret"`
	URL      string   `json:"url"`
	Tags     []string `json:"tags"`
	Notes    string   `json:"notes"`
	TOTPSeed string   `json:"totp_seed,omitempty"`
}

// ExportVault exports entries to JSON format
func ExportVault(entries []*domain.Entry, passphrase string, encrypted bool) ([]byte, error) {
	if encrypted {
		return exportEncrypted(entries, passphrase)
	}
	return exportPlaintext(entries)
}

// ImportVault imports entries from JSON format
func ImportVault(data []byte, passphrase string) ([]*domain.Entry, error) {
	var format ExportFormat
	if err := json.Unmarshal(data, &format); err != nil {
		return nil, fmt.Errorf("invalid export file format: %w", err)
	}

	if format.Encrypted {
		return importEncrypted(data, passphrase)
	}
	return importPlaintext(data)
}

func exportEncrypted(entries []*domain.Entry, passphrase string) ([]byte, error) {
	// Convert entries to export format
	exportEntries := make([]ExportEntry, len(entries))
	for i, entry := range entries {
		exportEntries[i] = ExportEntry{
			Name:     entry.Name,
			Username: entry.Username,
			Secret:   string(entry.Secret),
			URL:      entry.URL,
			Tags:     entry.Tags,
			Notes:    entry.Notes,
			TOTPSeed: entry.TOTPSeed,
		}
	}

	// Marshal to JSON
	plainData, err := json.Marshal(exportEntries)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal entries: %w", err)
	}

	// Create crypto engine
	crypto := NewDefaultCryptoEngine()

	// Encrypt data with passphrase
	envelope, err := crypto.SealWithPassphrase(plainData, passphrase)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt data: %w", err)
	}

	// Create export format
	format := ExportFormat{
		Version:   "1.0",
		Encrypted: true,
		Salt:      base64.StdEncoding.EncodeToString(envelope.Salt),
		Nonce:     base64.StdEncoding.EncodeToString(envelope.Nonce),
		Tag:       base64.StdEncoding.EncodeToString(envelope.Tag),
		Data:      base64.StdEncoding.EncodeToString(envelope.Ciphertext),
	}

	return json.MarshalIndent(format, "", "  ")
}

func exportPlaintext(entries []*domain.Entry) ([]byte, error) {
	exportEntries := make([]ExportEntry, len(entries))
	for i, entry := range entries {
		exportEntries[i] = ExportEntry{
			Name:     entry.Name,
			Username: entry.Username,
			Secret:   string(entry.Secret),
			URL:      entry.URL,
			Tags:     entry.Tags,
			Notes:    entry.Notes,
			TOTPSeed: entry.TOTPSeed,
		}
	}

	format := ExportFormat{
		Version:   "1.0",
		Encrypted: false,
		Entries:   exportEntries,
	}

	return json.MarshalIndent(format, "", "  ")
}

func importEncrypted(data []byte, passphrase string) ([]*domain.Entry, error) {
	var format ExportFormat
	if err := json.Unmarshal(data, &format); err != nil {
		return nil, fmt.Errorf("invalid export file: %w", err)
	}

	// Decode salt, nonce, tag, and ciphertext
	salt, err := base64.StdEncoding.DecodeString(format.Salt)
	if err != nil {
		return nil, fmt.Errorf("invalid salt: %w", err)
	}

	nonce, err := base64.StdEncoding.DecodeString(format.Nonce)
	if err != nil {
		return nil, fmt.Errorf("invalid nonce: %w", err)
	}

	tag, err := base64.StdEncoding.DecodeString(format.Tag)
	if err != nil {
		return nil, fmt.Errorf("invalid tag: %w", err)
	}

	ciphertext, err := base64.StdEncoding.DecodeString(format.Data)
	if err != nil {
		return nil, fmt.Errorf("invalid encrypted data: %w", err)
	}

	// Create envelope
	envelope := &Envelope{
		Version:    1,
		KDFParams:  DefaultArgon2Params(),
		Salt:       salt,
		Nonce:      nonce,
		Tag:        tag,
		Ciphertext: ciphertext,
	}

	// Create crypto engine
	crypto := NewDefaultCryptoEngine()

	// Decrypt data with passphrase
	plainData, err := crypto.OpenWithPassphrase(envelope, passphrase)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data (wrong passphrase?): %w", err)
	}

	// Unmarshal entries
	var exportEntries []ExportEntry
	if err := json.Unmarshal(plainData, &exportEntries); err != nil {
		return nil, fmt.Errorf("invalid entry data: %w", err)
	}

	return convertExportEntries(exportEntries), nil
}

func importPlaintext(data []byte) ([]*domain.Entry, error) {
	var format ExportFormat
	if err := json.Unmarshal(data, &format); err != nil {
		return nil, fmt.Errorf("invalid export file: %w", err)
	}

	return convertExportEntries(format.Entries), nil
}

func convertExportEntries(exportEntries []ExportEntry) []*domain.Entry {
	entries := make([]*domain.Entry, len(exportEntries))
	for i, e := range exportEntries {
		entries[i] = &domain.Entry{
			Name:     e.Name,
			Username: e.Username,
			Secret:   []byte(e.Secret),
			URL:      e.URL,
			Tags:     e.Tags,
			Notes:    e.Notes,
			TOTPSeed: e.TOTPSeed,
		}
	}
	return entries
}
