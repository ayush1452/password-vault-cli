package vault

import (
	"bytes"
	"crypto/rand"
	"testing"
	"time"
)

func TestGenerateSalt(t *testing.T) {
	salt1, err := GenerateSalt()
	if err != nil {
		t.Fatalf("Failed to generate salt: %v", err)
	}

	if len(salt1) != SaltSize {
		t.Errorf("Expected salt size %d, got %d", SaltSize, len(salt1))
	}

	salt2, err := GenerateSalt()
	if err != nil {
		t.Fatalf("Failed to generate second salt: %v", err)
	}

	if bytes.Equal(salt1, salt2) {
		t.Error("Generated salts should be different")
	}
}

func TestGenerateNonce(t *testing.T) {
	nonce1, err := GenerateNonce()
	if err != nil {
		t.Fatalf("Failed to generate nonce: %v", err)
	}

	if len(nonce1) != NonceSize {
		t.Errorf("Expected nonce size %d, got %d", NonceSize, len(nonce1))
	}

	nonce2, err := GenerateNonce()
	if err != nil {
		t.Fatalf("Failed to generate second nonce: %v", err)
	}

	if bytes.Equal(nonce1, nonce2) {
		t.Error("Generated nonces should be different")
	}
}

func TestDeriveKey(t *testing.T) {
	engine := NewDefaultCryptoEngine()

	// Test vector with known inputs
	passphrase := "test-passphrase-123"
	salt := make([]byte, SaltSize)
	for i := range salt {
		salt[i] = byte(i % 256)
	}

	key1, err := engine.DeriveKey(passphrase, salt)
	if err != nil {
		t.Fatalf("Failed to derive key: %v", err)
	}

	if len(key1) != KeySize {
		t.Errorf("Expected key size %d, got %d", KeySize, len(key1))
	}

	// Same inputs should produce same key
	key2, err := engine.DeriveKey(passphrase, salt)
	if err != nil {
		t.Fatalf("Failed to derive key second time: %v", err)
	}

	if !bytes.Equal(key1, key2) {
		t.Error("Same inputs should produce same key")
	}

	// Different passphrase should produce different key
	key3, err := engine.DeriveKey("different-passphrase", salt)
	if err != nil {
		t.Fatalf("Failed to derive key with different passphrase: %v", err)
	}

	if bytes.Equal(key1, key3) {
		t.Error("Different passphrase should produce different key")
	}

	// Different salt should produce different key
	salt2 := make([]byte, SaltSize)
	rand.Read(salt2)
	key4, err := engine.DeriveKey(passphrase, salt2)
	if err != nil {
		t.Fatalf("Failed to derive key with different salt: %v", err)
	}

	if bytes.Equal(key1, key4) {
		t.Error("Different salt should produce different key")
	}
}

func TestSealOpen(t *testing.T) {
	engine := NewDefaultCryptoEngine()

	// Generate key
	salt, err := GenerateSalt()
	if err != nil {
		t.Fatalf("Failed to generate salt: %v", err)
	}

	key, err := engine.DeriveKey("test-passphrase", salt)
	if err != nil {
		t.Fatalf("Failed to derive key: %v", err)
	}
	defer Zeroize(key)

	// Test data
	plaintext := []byte("This is a secret message that needs to be encrypted!")

	// Seal
	envelope, err := engine.Seal(plaintext, key)
	if err != nil {
		t.Fatalf("Failed to seal: %v", err)
	}

	// Verify envelope structure
	if envelope.Version != EnvelopeVersion {
		t.Errorf("Expected version %d, got %d", EnvelopeVersion, envelope.Version)
	}
	if len(envelope.Nonce) != NonceSize {
		t.Errorf("Expected nonce size %d, got %d", NonceSize, len(envelope.Nonce))
	}
	if len(envelope.Tag) != TagSize {
		t.Errorf("Expected tag size %d, got %d", TagSize, len(envelope.Tag))
	}
	if len(envelope.Ciphertext) == 0 {
		t.Error("Ciphertext should not be empty")
	}

	// Open
	decrypted, err := engine.Open(envelope, key)
	if err != nil {
		t.Fatalf("Failed to open: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Error("Decrypted text does not match original")
	}
}

func TestSealOpenWithPassphrase(t *testing.T) {
	engine := NewDefaultCryptoEngine()

	plaintext := []byte("Secret data for passphrase test")
	passphrase := "my-secure-passphrase-456"

	// Seal with passphrase
	envelope, err := engine.SealWithPassphrase(plaintext, passphrase)
	if err != nil {
		t.Fatalf("Failed to seal with passphrase: %v", err)
	}

	// Verify salt is present
	if envelope.Salt == nil || len(envelope.Salt) != SaltSize {
		t.Error("Envelope should contain salt")
	}

	// Open with passphrase
	decrypted, err := engine.OpenWithPassphrase(envelope, passphrase)
	if err != nil {
		t.Fatalf("Failed to open with passphrase: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Error("Decrypted text does not match original")
	}

	// Wrong passphrase should fail
	_, err = engine.OpenWithPassphrase(envelope, "wrong-passphrase")
	if err == nil {
		t.Error("Wrong passphrase should fail")
	}
}

func TestTamperDetection(t *testing.T) {
	engine := NewDefaultCryptoEngine()

	plaintext := []byte("Data that should detect tampering")
	passphrase := "tamper-test-passphrase"

	envelope, err := engine.SealWithPassphrase(plaintext, passphrase)
	if err != nil {
		t.Fatalf("Failed to seal: %v", err)
	}

	// Test tampering with ciphertext
	originalCiphertext := make([]byte, len(envelope.Ciphertext))
	copy(originalCiphertext, envelope.Ciphertext)

	envelope.Ciphertext[0] ^= 1 // Flip one bit
	_, err = engine.OpenWithPassphrase(envelope, passphrase)
	if err == nil {
		t.Error("Tampered ciphertext should fail decryption")
	}

	// Restore ciphertext
	copy(envelope.Ciphertext, originalCiphertext)

	// Test tampering with tag
	envelope.Tag[0] ^= 1 // Flip one bit in tag
	_, err = engine.OpenWithPassphrase(envelope, passphrase)
	if err == nil {
		t.Error("Tampered tag should fail decryption")
	}
}

func TestZeroize(t *testing.T) {
	data := []byte("sensitive data that should be zeroed")
	original := make([]byte, len(data))
	copy(original, data)

	Zeroize(data)

	// Check that all bytes are zero
	for i, b := range data {
		if b != 0 {
			t.Errorf("Byte at index %d not zeroed: %d", i, b)
		}
	}

	// Ensure original data was different
	allZero := true
	for _, b := range original {
		if b != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("Original data was already all zeros, test invalid")
	}
}

func TestZeroizeString(t *testing.T) {
	str := "sensitive string data"
	original := str

	ZeroizeString(&str)

	if str != "" {
		t.Error("String should be empty after zeroization")
	}

	if original == "" {
		t.Error("Original string was empty, test invalid")
	}
}

func TestSecureCompare(t *testing.T) {
	data1 := []byte("same data")
	data2 := []byte("same data")
	data3 := []byte("different data")

	if !SecureCompare(data1, data2) {
		t.Error("Same data should compare equal")
	}

	if SecureCompare(data1, data3) {
		t.Error("Different data should not compare equal")
	}

	if SecureCompare(data1, []byte("same")) {
		t.Error("Different length data should not compare equal")
	}
}

func TestValidateArgon2Params(t *testing.T) {
	// Valid params
	validParams := Argon2Params{
		Memory:      64 * 1024,
		Iterations:  3,
		Parallelism: 4,
	}

	if err := ValidateArgon2Params(validParams); err != nil {
		t.Errorf("Valid params should pass validation: %v", err)
	}

	// Invalid memory (too low)
	invalidParams := validParams
	invalidParams.Memory = 512
	if err := ValidateArgon2Params(invalidParams); err == nil {
		t.Error("Low memory should fail validation")
	}

	// Invalid memory (too high)
	invalidParams = validParams
	invalidParams.Memory = 2 * 1024 * 1024
	if err := ValidateArgon2Params(invalidParams); err == nil {
		t.Error("High memory should fail validation")
	}

	// Invalid iterations (too low)
	invalidParams = validParams
	invalidParams.Iterations = 0
	if err := ValidateArgon2Params(invalidParams); err == nil {
		t.Error("Zero iterations should fail validation")
	}

	// Invalid parallelism (too low)
	invalidParams = validParams
	invalidParams.Parallelism = 0
	if err := ValidateArgon2Params(invalidParams); err == nil {
		t.Error("Zero parallelism should fail validation")
	}
}

func TestBenchmarkKDF(t *testing.T) {
	params := DefaultArgon2Params()
	passphrase := "benchmark-test-passphrase"
	salt, err := GenerateSalt()
	if err != nil {
		t.Fatalf("Failed to generate salt: %v", err)
	}

	duration := BenchmarkKDF(params, passphrase, salt)

	if duration <= 0 {
		t.Error("Benchmark duration should be positive")
	}

	// Should typically take at least 50ms with default params
	if duration < 50*time.Millisecond {
		t.Logf("Warning: KDF took only %v, might be too fast", duration)
	}
}

func TestEnvelopeSerialization(t *testing.T) {
	engine := NewDefaultCryptoEngine()

	plaintext := []byte("Test data for serialization")
	passphrase := "serialization-test-passphrase"

	// Create envelope
	envelope, err := engine.SealWithPassphrase(plaintext, passphrase)
	if err != nil {
		t.Fatalf("Failed to create envelope: %v", err)
	}

	// Serialize
	data := EnvelopeToBytes(envelope)
	if len(data) == 0 {
		t.Error("Serialized data should not be empty")
	}

	// Deserialize
	envelope2, err := EnvelopeFromBytes(data)
	if err != nil {
		t.Fatalf("Failed to deserialize envelope: %v", err)
	}

	// Compare envelopes
	if envelope2.Version != envelope.Version {
		t.Error("Version mismatch after serialization")
	}
	if !bytes.Equal(envelope2.Salt, envelope.Salt) {
		t.Error("Salt mismatch after serialization")
	}
	if !bytes.Equal(envelope2.Nonce, envelope.Nonce) {
		t.Error("Nonce mismatch after serialization")
	}
	if !bytes.Equal(envelope2.Ciphertext, envelope.Ciphertext) {
		t.Error("Ciphertext mismatch after serialization")
	}
	if !bytes.Equal(envelope2.Tag, envelope.Tag) {
		t.Error("Tag mismatch after serialization")
	}

	// Verify it can still decrypt
	decrypted, err := engine.OpenWithPassphrase(envelope2, passphrase)
	if err != nil {
		t.Fatalf("Failed to decrypt deserialized envelope: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Error("Decrypted data mismatch after serialization")
	}
}

func TestKnownTestVectors(t *testing.T) {
	// Test with known inputs to ensure consistency
	params := Argon2Params{
		Memory:      1024, // Small for test
		Iterations:  1,
		Parallelism: 1,
	}

	engine := NewCryptoEngine(params)

	passphrase := "test123"
	salt := make([]byte, SaltSize)
	// Fill salt with predictable pattern
	for i := range salt {
		salt[i] = byte(i)
	}

	key1, err := engine.DeriveKey(passphrase, salt)
	if err != nil {
		t.Fatalf("Failed to derive key: %v", err)
	}

	// Derive again with same inputs
	key2, err := engine.DeriveKey(passphrase, salt)
	if err != nil {
		t.Fatalf("Failed to derive key second time: %v", err)
	}

	if !bytes.Equal(key1, key2) {
		t.Error("Same inputs should always produce same key")
	}

	// Test encryption with fixed nonce (for reproducibility in tests)
	plaintext := []byte("Hello, World!")

	// We can't easily test with fixed nonce in the current API,
	// but we can test that encryption/decryption is consistent
	envelope, err := engine.Seal(plaintext, key1)
	if err != nil {
		t.Fatalf("Failed to seal: %v", err)
	}

	decrypted, err := engine.Open(envelope, key1)
	if err != nil {
		t.Fatalf("Failed to open: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Error("Round-trip encryption failed")
	}
}

// Benchmark tests
func BenchmarkDeriveKey(b *testing.B) {
	engine := NewDefaultCryptoEngine()
	passphrase := "benchmark-passphrase"
	salt, _ := GenerateSalt()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key, _ := engine.DeriveKey(passphrase, salt)
		Zeroize(key)
	}
}

func BenchmarkSeal(b *testing.B) {
	engine := NewDefaultCryptoEngine()
	salt, _ := GenerateSalt()
	key, _ := engine.DeriveKey("test", salt)
	defer Zeroize(key)

	plaintext := make([]byte, 1024) // 1KB
	rand.Read(plaintext)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Seal(plaintext, key)
	}
}

func BenchmarkOpen(b *testing.B) {
	engine := NewDefaultCryptoEngine()
	salt, _ := GenerateSalt()
	key, _ := engine.DeriveKey("test", salt)
	defer Zeroize(key)

	plaintext := make([]byte, 1024) // 1KB
	rand.Read(plaintext)

	envelope, _ := engine.Seal(plaintext, key)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Open(envelope, key)
	}
}
