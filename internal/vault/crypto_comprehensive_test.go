package vault

import (
	"bytes"
	"crypto/rand"
	"strings"
	"testing"
	"time"
)

// TestGenerateSaltComprehensive tests salt generation with comprehensive scenarios
func TestGenerateSaltComprehensive(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"Generate salt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			salt1, err := GenerateSalt()
			if err != nil {
				t.Errorf("GenerateSalt() error = %v", err)
				return
			}

			salt2, err := GenerateSalt()
			if err != nil {
				t.Errorf("GenerateSalt() error = %v", err)
				return
			}

			// Check salt size
			if len(salt1) != SaltSize {
				t.Errorf("GenerateSalt() salt size = %d, want %d", len(salt1), SaltSize)
			}

			// Check salts are different (extremely unlikely to be same)
			if bytes.Equal(salt1, salt2) {
				t.Error("GenerateSalt() generated identical salts")
			}

			// Check salt is not all zeros
			allZeros := make([]byte, SaltSize)
			if bytes.Equal(salt1, allZeros) {
				t.Error("GenerateSalt() generated all-zero salt")
			}
		})
	}
}

// TestGenerateNonceComprehensive tests nonce generation with comprehensive scenarios
func TestGenerateNonceComprehensive(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"Generate nonce"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nonce1, err := GenerateNonce()
			if err != nil {
				t.Errorf("GenerateNonce() error = %v", err)
				return
			}

			nonce2, err := GenerateNonce()
			if err != nil {
				t.Errorf("GenerateNonce() error = %v", err)
				return
			}

			// Check nonce size
			if len(nonce1) != NonceSize {
				t.Errorf("GenerateNonce() nonce size = %d, want %d", len(nonce1), NonceSize)
			}

			// Check nonces are different
			if bytes.Equal(nonce1, nonce2) {
				t.Error("GenerateNonce() generated identical nonces")
			}

			// Check nonce is not all zeros
			allZeros := make([]byte, NonceSize)
			if bytes.Equal(nonce1, allZeros) {
				t.Error("GenerateNonce() generated all-zero nonce")
			}
		})
	}
}

// TestCryptoEngine_DeriveKey tests key derivation with various scenarios
func TestCryptoEngine_DeriveKey(t *testing.T) {
	ce := NewDefaultCryptoEngine()
	salt, _ := GenerateSalt()

	tests := []struct {
		name       string
		passphrase string
		salt       []byte
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "Valid passphrase and salt",
			passphrase: "test-passphrase-123",
			salt:       salt,
			wantErr:    false,
		},
		{
			name:       "Empty passphrase",
			passphrase: "",
			salt:       salt,
			wantErr:    false, // Should still work but produce weak key
		},
		{
			name:       "Long passphrase",
			passphrase: strings.Repeat("a", 1000),
			salt:       salt,
			wantErr:    false,
		},
		{
			name:       "Invalid salt size",
			passphrase: "test-passphrase",
			salt:       make([]byte, 16), // Wrong size
			wantErr:    true,
			errMsg:     "invalid salt size",
		},
		{
			name:       "Nil salt",
			passphrase: "test-passphrase",
			salt:       nil,
			wantErr:    true,
			errMsg:     "invalid salt size",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := ce.DeriveKey(tt.passphrase, tt.salt)

			if tt.wantErr {
				if err == nil {
					t.Errorf("DeriveKey() expected error but got none")
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("DeriveKey() error = %v, want error containing %v", err, tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("DeriveKey() unexpected error = %v", err)
				return
			}

			// Check key size
			if len(key) != KeySize {
				t.Errorf("DeriveKey() key size = %d, want %d", len(key), KeySize)
			}

			// Test deterministic behavior - same inputs should produce same key
			key2, err := ce.DeriveKey(tt.passphrase, tt.salt)
			if err != nil {
				t.Errorf("DeriveKey() second call error = %v", err)
				return
			}

			if !bytes.Equal(key, key2) {
				t.Error("DeriveKey() is not deterministic")
			}

			// Test different salt produces different key
			if len(tt.salt) == SaltSize {
				differentSalt, _ := GenerateSalt()
				key3, err := ce.DeriveKey(tt.passphrase, differentSalt)
				if err != nil {
					t.Errorf("DeriveKey() with different salt error = %v", err)
					return
				}

				if bytes.Equal(key, key3) {
					t.Error("DeriveKey() produced same key with different salt")
				}
			}
		})
	}
}

// TestCryptoEngine_KeyDerivationTiming tests timing characteristics
func TestCryptoEngine_KeyDerivationTiming(t *testing.T) {
	ce := NewDefaultCryptoEngine()
	salt, _ := GenerateSalt()
	passphrase := "test-passphrase-for-timing"

	start := time.Now()
	_, err := ce.DeriveKey(passphrase, salt)
	duration := time.Since(start)

	if err != nil {
		t.Errorf("DeriveKey() timing test error = %v", err)
		return
	}

	// Key derivation should take reasonable time (50ms to 2s)
	if duration < 50*time.Millisecond {
		t.Logf("Warning: Key derivation took only %v, might be too fast", duration)
	}
	if duration > 2*time.Second {
		t.Logf("Warning: Key derivation took %v, might be too slow", duration)
	}

	t.Logf("Key derivation took %v", duration)
}

// TestCryptoEngine_SealAndOpen tests encryption and decryption
func TestCryptoEngine_SealAndOpen(t *testing.T) {
	ce := NewDefaultCryptoEngine()
	key := make([]byte, KeySize)
	rand.Read(key)

	tests := []struct {
		name      string
		plaintext []byte
		key       []byte
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "Valid plaintext and key",
			plaintext: []byte("Hello, World! This is a test message."),
			key:       key,
			wantErr:   false,
		},
		{
			name:      "Empty plaintext",
			plaintext: []byte(""),
			key:       key,
			wantErr:   false,
		},
		{
			name:      "Large plaintext",
			plaintext: bytes.Repeat([]byte("A"), 10000),
			key:       key,
			wantErr:   false,
		},
		{
			name:      "Invalid key size",
			plaintext: []byte("test"),
			key:       make([]byte, 16), // Wrong size
			wantErr:   true,
			errMsg:    "invalid key size",
		},
		{
			name:      "Nil key",
			plaintext: []byte("test"),
			key:       nil,
			wantErr:   true,
			errMsg:    "invalid key size",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test Seal
			envelope, err := ce.Seal(tt.plaintext, tt.key)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Seal() expected error but got none")
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Seal() error = %v, want error containing %v", err, tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("Seal() unexpected error = %v", err)
				return
			}

			// Validate envelope structure
			if envelope.Version != EnvelopeVersion {
				t.Errorf("Seal() envelope version = %d, want %d", envelope.Version, EnvelopeVersion)
			}
			if len(envelope.Nonce) != NonceSize {
				t.Errorf("Seal() nonce size = %d, want %d", len(envelope.Nonce), NonceSize)
			}
			if len(envelope.Tag) != TagSize {
				t.Errorf("Seal() tag size = %d, want %d", len(envelope.Tag), TagSize)
			}

			// Test Open
			decrypted, err := ce.Open(envelope, tt.key)
			if err != nil {
				t.Errorf("Open() error = %v", err)
				return
			}

			if !bytes.Equal(decrypted, tt.plaintext) {
				t.Errorf("Open() decrypted = %v, want %v", decrypted, tt.plaintext)
			}
		})
	}
}

// TestCryptoEngine_OpenInvalidData tests decryption with invalid data
func TestCryptoEngine_OpenInvalidData(t *testing.T) {
	ce := NewDefaultCryptoEngine()
	key := make([]byte, KeySize)
	rand.Read(key)

	plaintext := []byte("test message")
	envelope, _ := ce.Seal(plaintext, key)

	tests := []struct {
		name     string
		envelope *Envelope
		key      []byte
		wantErr  bool
		errMsg   string
	}{
		{
			name: "Invalid version",
			envelope: &Envelope{
				Version:    99,
				KDFParams:  envelope.KDFParams,
				Nonce:      envelope.Nonce,
				Ciphertext: envelope.Ciphertext,
				Tag:        envelope.Tag,
			},
			key:     key,
			wantErr: true,
			errMsg:  "unsupported envelope version",
		},
		{
			name: "Invalid nonce size",
			envelope: &Envelope{
				Version:    envelope.Version,
				KDFParams:  envelope.KDFParams,
				Nonce:      make([]byte, 8), // Wrong size
				Ciphertext: envelope.Ciphertext,
				Tag:        envelope.Tag,
			},
			key:     key,
			wantErr: true,
			errMsg:  "invalid nonce size",
		},
		{
			name: "Invalid tag size",
			envelope: &Envelope{
				Version:    envelope.Version,
				KDFParams:  envelope.KDFParams,
				Nonce:      envelope.Nonce,
				Ciphertext: envelope.Ciphertext,
				Tag:        make([]byte, 8), // Wrong size
			},
			key:     key,
			wantErr: true,
			errMsg:  "invalid tag size",
		},
		{
			name: "Corrupted ciphertext",
			envelope: &Envelope{
				Version:    envelope.Version,
				KDFParams:  envelope.KDFParams,
				Nonce:      envelope.Nonce,
				Ciphertext: append(envelope.Ciphertext, 0xFF), // Corrupt data
				Tag:        envelope.Tag,
			},
			key:     key,
			wantErr: true,
			errMsg:  "decryption failed",
		},
		{
			name: "Corrupted tag",
			envelope: &Envelope{
				Version:    envelope.Version,
				KDFParams:  envelope.KDFParams,
				Nonce:      envelope.Nonce,
				Ciphertext: envelope.Ciphertext,
				Tag:        append([]byte{0xFF}, envelope.Tag[1:]...), // Corrupt tag
			},
			key:     key,
			wantErr: true,
			errMsg:  "decryption failed",
		},
		{
			name:     "Wrong key size",
			envelope: envelope,
			key:      make([]byte, 16), // Wrong key size
			wantErr:  true,
			errMsg:   "invalid key size",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ce.Open(tt.envelope, tt.key)

			if !tt.wantErr {
				if err != nil {
					t.Errorf("Open() unexpected error = %v", err)
				}
				return
			}

			if err == nil {
				t.Errorf("Open() expected error but got none")
				return
			}

			if !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Open() error = %v, want error containing %v", err, tt.errMsg)
			}
		})
	}
}

// TestCryptoEngine_SealWithPassphrase tests passphrase-based encryption
func TestCryptoEngine_SealWithPassphrase(t *testing.T) {
	ce := NewDefaultCryptoEngine()

	tests := []struct {
		name       string
		plaintext  []byte
		passphrase string
		wantErr    bool
	}{
		{
			name:       "Valid passphrase encryption",
			plaintext:  []byte("Secret message"),
			passphrase: "strong-passphrase-123",
			wantErr:    false,
		},
		{
			name:       "Empty passphrase",
			plaintext:  []byte("Secret message"),
			passphrase: "",
			wantErr:    false, // Should work but be weak
		},
		{
			name:       "Unicode passphrase",
			plaintext:  []byte("Secret message"),
			passphrase: "пароль-密码-🔐",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			envelope, err := ce.SealWithPassphrase(tt.plaintext, tt.passphrase)

			if tt.wantErr {
				if err == nil {
					t.Errorf("SealWithPassphrase() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("SealWithPassphrase() unexpected error = %v", err)
				return
			}

			// Validate envelope has salt
			if len(envelope.Salt) != SaltSize {
				t.Errorf("SealWithPassphrase() salt size = %d, want %d", len(envelope.Salt), SaltSize)
			}

			// Test decryption
			decrypted, err := ce.OpenWithPassphrase(envelope, tt.passphrase)
			if err != nil {
				t.Errorf("OpenWithPassphrase() error = %v", err)
				return
			}

			if !bytes.Equal(decrypted, tt.plaintext) {
				t.Errorf("OpenWithPassphrase() decrypted = %v, want %v", decrypted, tt.plaintext)
			}

			// Test wrong passphrase fails
			_, err = ce.OpenWithPassphrase(envelope, tt.passphrase+"wrong")
			if err == nil {
				t.Error("OpenWithPassphrase() with wrong passphrase should fail")
			}
		})
	}
}

// TestValidateArgon2ParamsComprehensive tests parameter validation with comprehensive scenarios
func TestValidateArgon2ParamsComprehensive(t *testing.T) {
	tests := []struct {
		name    string
		params  Argon2Params
		wantErr bool
		errMsg  string
	}{
		{
			name:    "Valid default params",
			params:  DefaultArgon2Params(),
			wantErr: false,
		},
		{
			name: "Memory too low",
			params: Argon2Params{
				Memory:      512, // Too low
				Iterations:  3,
				Parallelism: 4,
			},
			wantErr: true,
			errMsg:  "memory parameter too low",
		},
		{
			name: "Memory too high",
			params: Argon2Params{
				Memory:      2 * 1024 * 1024, // Too high
				Iterations:  3,
				Parallelism: 4,
			},
			wantErr: true,
			errMsg:  "memory parameter too high",
		},
		{
			name: "Iterations too low",
			params: Argon2Params{
				Memory:      64 * 1024,
				Iterations:  0, // Too low
				Parallelism: 4,
			},
			wantErr: true,
			errMsg:  "iterations parameter too low",
		},
		{
			name: "Iterations too high",
			params: Argon2Params{
				Memory:      64 * 1024,
				Iterations:  101, // Too high
				Parallelism: 4,
			},
			wantErr: true,
			errMsg:  "iterations parameter too high",
		},
		{
			name: "Parallelism too low",
			params: Argon2Params{
				Memory:      64 * 1024,
				Iterations:  3,
				Parallelism: 0, // Too low
			},
			wantErr: true,
			errMsg:  "parallelism parameter too low",
		},
		{
			name: "Parallelism too high",
			params: Argon2Params{
				Memory:      64 * 1024,
				Iterations:  3,
				Parallelism: 255, // Too high (max allowed 254)
			},
			wantErr: true,
			errMsg:  "parallelism parameter too high",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateArgon2Params(tt.params)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateArgon2Params() expected error but got none")
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateArgon2Params() error = %v, want error containing %v", err, tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("ValidateArgon2Params() unexpected error = %v", err)
			}
		})
	}
}

// TestZeroizeComprehensive tests secure memory clearing with comprehensive scenarios
func TestZeroizeComprehensive(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "Zero byte slice",
			data: []byte{1, 2, 3, 4, 5},
		},
		{
			name: "Zero empty slice",
			data: []byte{},
		},
		{
			name: "Zero nil slice",
			data: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := make([]byte, len(tt.data))
			copy(original, tt.data)

			Zeroize(tt.data)

			// Check all bytes are zero
			for i, b := range tt.data {
				if b != 0 {
					t.Errorf("Zeroize() byte at index %d = %v, want 0", i, b)
				}
			}
		})
	}
}

// TestSecureCompareComprehensive tests constant-time comparison with comprehensive scenarios
func TestSecureCompareComprehensive(t *testing.T) {
	tests := []struct {
		name string
		a    []byte
		b    []byte
		want bool
	}{
		{
			name: "Equal slices",
			a:    []byte{1, 2, 3, 4},
			b:    []byte{1, 2, 3, 4},
			want: true,
		},
		{
			name: "Different slices same length",
			a:    []byte{1, 2, 3, 4},
			b:    []byte{1, 2, 3, 5},
			want: false,
		},
		{
			name: "Different lengths",
			a:    []byte{1, 2, 3},
			b:    []byte{1, 2, 3, 4},
			want: false,
		},
		{
			name: "Empty slices",
			a:    []byte{},
			b:    []byte{},
			want: true,
		},
		{
			name: "Nil slices",
			a:    nil,
			b:    nil,
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SecureCompare(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("SecureCompare() = %v, want %v", got, tt.want)
			}
		})
	}
}
