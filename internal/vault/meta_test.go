package vault

import (
	"encoding/base64"
	"testing"
	"time"

	"github.com/vault-cli/vault/internal/domain"
)

// TestDecodeMetadataInfo tests metadata decoding
func TestDecodeMetadataInfo(t *testing.T) {
	// Create test metadata with valid KDF params
	salt := make([]byte, 32)
	for i := range salt {
		salt[i] = byte(i)
	}
	saltB64 := base64.StdEncoding.EncodeToString(salt)

	kdfParams := map[string]interface{}{
		"memory":      uint32(65536),
		"iterations":  uint32(3),
		"parallelism": uint8(4),
		"salt":        saltB64,
	}

	metadata := &domain.VaultMetadata{
		CreatedAt: time.Now(),
		KDFParams: kdfParams,
	}

	info, decodedSalt, err := DecodeMetadataInfo(metadata)
	if err != nil {
		t.Fatalf("DecodeMetadataInfo() error = %v", err)
	}

	if info == nil {
		t.Fatal("DecodeMetadataInfo() returned nil info")
	}

	if info.Cipher != "AES-256-GCM" {
		t.Errorf("Expected cipher AES-256-GCM, got %s", info.Cipher)
	}

	if info.KDF.Memory != 65536 {
		t.Errorf("Expected memory 65536, got %d", info.KDF.Memory)
	}

	if info.KDF.Iterations != 3 {
		t.Errorf("Expected iterations 3, got %d", info.KDF.Iterations)
	}

	if info.KDF.Parallelism != 4 {
		t.Errorf("Expected parallelism 4, got %d", info.KDF.Parallelism)
	}

	if len(decodedSalt) != 32 {
		t.Errorf("Expected salt length 32, got %d", len(decodedSalt))
	}

	if info.SaltLength != 32 {
		t.Errorf("Expected salt length 32 in info, got %d", info.SaltLength)
	}
}

// TestDecodeMetadataInfoNil tests nil metadata
func TestDecodeMetadataInfoNil(t *testing.T) {
	_, _, err := DecodeMetadataInfo(nil)
	if err == nil {
		t.Error("DecodeMetadataInfo(nil) should return error")
	}
}

// TestDecodeMetadataInfoMissingKDFParams tests missing KDF params
func TestDecodeMetadataInfoMissingKDFParams(t *testing.T) {
	metadata := &domain.VaultMetadata{
		CreatedAt: time.Now(),
		KDFParams: nil,
	}

	_, _, err := DecodeMetadataInfo(metadata)
	if err == nil {
		t.Error("DecodeMetadataInfo() should fail with nil KDFParams")
	}
}

// TestDecodeMetadataInfoInvalidSalt tests invalid base64 salt
func TestDecodeMetadataInfoInvalidSalt(t *testing.T) {
	kdfParams := map[string]interface{}{
		"memory":      uint32(65536),
		"iterations":  uint32(3),
		"parallelism": uint8(4),
		"salt":        "invalid-base64!!!",
	}

	metadata := &domain.VaultMetadata{
		CreatedAt: time.Now(),
		KDFParams: kdfParams,
	}

	_, _, err := DecodeMetadataInfo(metadata)
	if err == nil {
		t.Error("DecodeMetadataInfo() should fail with invalid base64 salt")
	}
}

// TestDecodeKDFParamsFloat64Values tests KDF params with float64 values
func TestDecodeKDFParamsFloat64Values(t *testing.T) {
	// Simulate JSON unmarshaling that converts numbers to float64
	salt := make([]byte, 32)
	saltB64 := base64.StdEncoding.EncodeToString(salt)

	kdfParams := map[string]interface{}{
		"memory":      float64(65536),
		"iterations":  float64(3),
		"parallelism": float64(4),
		"salt":        saltB64,
	}

	metadata := &domain.VaultMetadata{
		CreatedAt: time.Now(),
		KDFParams: kdfParams,
	}

	info, _, err := DecodeMetadataInfo(metadata)
	if err != nil {
		t.Fatalf("DecodeMetadataInfo() with float64 values error = %v", err)
	}

	if info.KDF.Memory != 65536 {
		t.Errorf("Expected memory 65536 from float64, got %d", info.KDF.Memory)
	}
}

// TestDecodeMetadataInfoMissingSalt tests missing salt in KDF params
func TestDecodeMetadataInfoMissingSalt(t *testing.T) {
	kdfParams := map[string]interface{}{
		"memory":      uint32(65536),
		"iterations":  uint32(3),
		"parallelism": uint8(4),
		// salt missing - will be empty string which is valid base64 (decodes to empty byte slice)
	}

	metadata := &domain.VaultMetadata{
		CreatedAt: time.Now(),
		KDFParams: kdfParams,
	}

	// Empty salt is valid (empty string is valid base64), just results in empty byte slice
	info, salt, err := DecodeMetadataInfo(metadata)
	if err != nil {
		t.Fatalf("DecodeMetadataInfo() error = %v", err)
	}
	if len(salt) != 0 {
		t.Errorf("Expected empty salt, got length %d", len(salt))
	}
	if info.SaltLength != 0 {
		t.Errorf("Expected SaltLength 0, got %d", info.SaltLength)
	}
}

// TestDecodeMetadataInfoValidFormats tests various valid formats
func TestDecodeMetadataInfoValidFormats(t *testing.T) {
	tests := []struct {
		name      string
		memory    interface{}
		iter      interface{}
		parallel  interface{}
		wantError bool
	}{
		{"uint32", uint32(65536), uint32(3), uint8(4), false},
		{"int", int(65536), int(3), int(4), false},
		{"float64", float64(65536), float64(3), float64(4), false},
	}

	salt := make([]byte, 32)
	saltB64 := base64.StdEncoding.EncodeToString(salt)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kdfParams := map[string]interface{}{
				"memory":      tt.memory,
				"iterations":  tt.iter,
				"parallelism": tt.parallel,
				"salt":        saltB64,
			}

			metadata := &domain.VaultMetadata{
				CreatedAt: time.Now(),
				KDFParams: kdfParams,
			}

			info, _, err := DecodeMetadataInfo(metadata)
			if (err != nil) != tt.wantError {
				t.Errorf("DecodeMetadataInfo() error = %v, wantError %v", err, tt.wantError)
			}
			if err == nil && info == nil {
				t.Error("DecodeMetadataInfo() returned nil info without error")
			}
		})
	}
}
