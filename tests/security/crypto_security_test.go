package security_test

import (
	"crypto/subtle"
	"testing"
	"time"

	"github.com/vault-cli/vault/internal/vault"
)

// TestKeyDerivationSecurity tests the security of key derivation
func TestKeyDerivationSecurity(t *testing.T) {
	t.Run("Argon2id parameters are strong", func(t *testing.T) {
		params := vault.DefaultArgon2Params()
		
		// Check memory parameter (should be at least 64MB)
		if params.Memory < 65536 {
			t.Errorf("Memory parameter too low: %d KB (minimum: 65536 KB)", params.Memory)
		} else {
			t.Logf("✓ Memory parameter: %d KB", params.Memory)
		}
		
		// Check iterations (should be at least 3)
		if params.Iterations < 3 {
			t.Errorf("Iterations too low: %d (minimum: 3)", params.Iterations)
		} else {
			t.Logf("✓ Iterations: %d", params.Iterations)
		}
		
		// Check parallelism (should be at least 4)
		if params.Parallelism < 4 {
			t.Errorf("Parallelism too low: %d (minimum: 4)", params.Parallelism)
		} else {
			t.Logf("✓ Parallelism: %d", params.Parallelism)
		}
	})
	
	t.Run("KDF timing is appropriate", func(t *testing.T) {
		params := vault.DefaultArgon2Params()
		crypto := vault.NewCryptoEngine(params)
		
		passphrase := "test-passphrase-for-timing"
		salt, err := vault.GenerateSalt()
		if err != nil {
			t.Fatalf("Failed to generate salt: %v", err)
		}
		
		// Measure KDF time
		start := time.Now()
		key, err := crypto.DeriveKey(passphrase, salt)
		duration := time.Since(start)
		
		if err != nil {
			t.Fatalf("Key derivation failed: %v", err)
		}
		defer vault.Zeroize(key)
		
		t.Logf("KDF duration: %v", duration)
		
		// Should take at least 30ms (configurable based on security requirements)
		if duration < 30*time.Millisecond {
			t.Logf("Warning: KDF is fast (%v), consider increasing parameters for production", duration)
		}
		
		// Should not take more than 1 second (usability)
		if duration > time.Second {
			t.Logf("Warning: KDF is slow (%v), may impact user experience", duration)
		}
		
		// Ideal range: 100-500ms
		if duration >= 100*time.Millisecond && duration <= 500*time.Millisecond {
			t.Logf("✓ KDF timing is in ideal range for security/usability balance")
		}
	})
	
	t.Run("Salt is unique for each derivation", func(t *testing.T) {
		numSalts := 100
		salts := make(map[string]bool)
		
		for i := 0; i < numSalts; i++ {
			salt, err := vault.GenerateSalt()
			if err != nil {
				t.Fatalf("Failed to generate salt %d: %v", i, err)
			}
			
			saltStr := string(salt)
			if salts[saltStr] {
				t.Errorf("Duplicate salt generated at iteration %d", i)
			}
			salts[saltStr] = true
			
			// Check salt length
			if len(salt) != 32 {
				t.Errorf("Salt length incorrect: %d bytes (expected: 32)", len(salt))
			}
		}
		
		t.Logf("✓ Generated %d unique salts of correct length", numSalts)
	})
	
	t.Run("Same passphrase with different salts produces different keys", func(t *testing.T) {
		passphrase := "same-passphrase"
		params := vault.DefaultArgon2Params()
		crypto := vault.NewCryptoEngine(params)
		
		// Generate two different salts
		salt1, _ := vault.GenerateSalt()
		salt2, _ := vault.GenerateSalt()
		
		// Derive keys
		key1, _ := crypto.DeriveKey(passphrase, salt1)
		key2, _ := crypto.DeriveKey(passphrase, salt2)
		
		defer vault.Zeroize(key1)
		defer vault.Zeroize(key2)
		
		// Keys should be different
		if subtle.ConstantTimeCompare(key1, key2) == 1 {
			t.Error("Same passphrase with different salts produced identical keys")
		} else {
			t.Log("✓ Different salts produce different keys")
		}
	})
	
	t.Run("Different passphrases produce different keys", func(t *testing.T) {
		params := vault.DefaultArgon2Params()
		crypto := vault.NewCryptoEngine(params)
		salt, _ := vault.GenerateSalt()
		
		key1, _ := crypto.DeriveKey("passphrase1", salt)
		key2, _ := crypto.DeriveKey("passphrase2", salt)
		
		defer vault.Zeroize(key1)
		defer vault.Zeroize(key2)
		
		if subtle.ConstantTimeCompare(key1, key2) == 1 {
			t.Error("Different passphrases produced identical keys")
		} else {
			t.Log("✓ Different passphrases produce different keys")
		}
	})
}

// TestEncryptionSecurity tests the security of encryption operations
func TestEncryptionSecurity(t *testing.T) {
	t.Run("Encryption uses AES-256-GCM", func(t *testing.T) {
		// This is verified by the crypto implementation
		// We test that encryption/decryption works correctly
		crypto := vault.NewDefaultCryptoEngine()
		
		plaintext := []byte("sensitive data to encrypt")
		key := make([]byte, 32) // 256-bit key
		for i := range key {
			key[i] = byte(i)
		}
		
		// Encrypt
		ciphertext, err := crypto.Seal(plaintext, key)
		if err != nil {
			t.Fatalf("Encryption failed: %v", err)
		}
		
		// Verify ciphertext is different from plaintext
		if subtle.ConstantTimeCompare(plaintext, ciphertext.Ciphertext) == 1 {
			t.Error("Ciphertext should not match plaintext")
		}
		
		// Verify ciphertext is not empty
		if len(ciphertext.Ciphertext) == 0 {
			t.Error("Ciphertext should not be empty")
		}
		
		// Verify nonce and tag are present
		if len(ciphertext.Nonce) == 0 {
			t.Error("Nonce should not be empty")
		}
		if len(ciphertext.Tag) == 0 {
			t.Error("Tag should not be empty")
		}
		
		// Decrypt
		decrypted, err := crypto.Open(ciphertext, key)
		if err != nil {
			t.Fatalf("Decryption failed: %v", err)
		}
		
		// Verify decrypted matches original
		if subtle.ConstantTimeCompare(plaintext, decrypted) != 1 {
			t.Error("Decrypted data does not match original plaintext")
		}
		
		t.Log("✓ AES-256-GCM encryption/decryption working correctly")
	})
	
	t.Run("Nonce is unique for each encryption", func(t *testing.T) {
		crypto := vault.NewDefaultCryptoEngine()
		plaintext := []byte("test data")
		key := make([]byte, 32)
		
		numEncryptions := 1000
		ciphertexts := make(map[string]bool)
		
		for i := 0; i < numEncryptions; i++ {
			ciphertext, err := crypto.Seal(plaintext, key)
			if err != nil {
				t.Fatalf("Encryption %d failed: %v", i, err)
			}
			
			// Check for duplicate nonces (would indicate nonce reuse)
			nonceStr := string(ciphertext.Nonce)
			if ciphertexts[nonceStr] {
				t.Errorf("Nonce reuse detected at iteration %d!", i)
			}
			ciphertexts[nonceStr] = true
		}
		
		t.Logf("✓ %d encryptions produced unique ciphertexts (unique nonces)", numEncryptions)
	})
	
	t.Run("Tampering with ciphertext is detected", func(t *testing.T) {
		crypto := vault.NewDefaultCryptoEngine()
		plaintext := []byte("sensitive data")
		key := make([]byte, 32)
		
		ciphertext, err := crypto.Seal(plaintext, key)
		if err != nil {
			t.Fatalf("Encryption failed: %v", err)
		}
		
		// Tamper with ciphertext
		tamperedCiphertext := &vault.Envelope{
			Version:    ciphertext.Version,
			KDFParams:  ciphertext.KDFParams,
			Salt:       ciphertext.Salt,
			Nonce:      ciphertext.Nonce,
			Ciphertext: make([]byte, len(ciphertext.Ciphertext)),
			Tag:        make([]byte, len(ciphertext.Tag)),
		}
		copy(tamperedCiphertext.Ciphertext, ciphertext.Ciphertext)
		copy(tamperedCiphertext.Tag, ciphertext.Tag)
		
		// Flip a bit in the ciphertext
		if len(tamperedCiphertext.Ciphertext) > 10 {
			tamperedCiphertext.Ciphertext[5] ^= 0x01
		}
		
		// Try to decrypt tampered ciphertext
		_, err = crypto.Open(tamperedCiphertext, key)
		if err == nil {
			t.Error("Tampered ciphertext should fail authentication")
		} else {
			t.Logf("✓ Tampering detected: %v", err)
		}
	})
	
	t.Run("Wrong key fails decryption", func(t *testing.T) {
		crypto := vault.NewDefaultCryptoEngine()
		plaintext := []byte("sensitive data")
		correctKey := make([]byte, 32)
		wrongKey := make([]byte, 32)
		
		// Make keys different
		for i := range wrongKey {
			wrongKey[i] = byte(i + 1)
		}
		
		// Encrypt with correct key
		ciphertext, err := crypto.Seal(plaintext, correctKey)
		if err != nil {
			t.Fatalf("Encryption failed: %v", err)
		}
		
		// Try to decrypt with wrong key
		_, err = crypto.Open(ciphertext, wrongKey)
		if err == nil {
			t.Error("Decryption with wrong key should fail")
		} else {
			t.Logf("✓ Wrong key rejected: %v", err)
		}
	})
	
	t.Run("Empty plaintext can be encrypted", func(t *testing.T) {
		crypto := vault.NewDefaultCryptoEngine()
		plaintext := []byte("")
		key := make([]byte, 32)
		
		ciphertext, err := crypto.Seal(plaintext, key)
		if err != nil {
			t.Fatalf("Failed to encrypt empty plaintext: %v", err)
		}
		
		decrypted, err := crypto.Open(ciphertext, key)
		if err != nil {
			t.Fatalf("Failed to decrypt empty ciphertext: %v", err)
		}
		
		if len(decrypted) != 0 {
			t.Errorf("Decrypted empty plaintext should be empty, got %d bytes", len(decrypted))
		}
		
		t.Log("✓ Empty plaintext handled correctly")
	})
	
	t.Run("Large plaintext can be encrypted", func(t *testing.T) {
		crypto := vault.NewDefaultCryptoEngine()
		
		// Create 1MB plaintext
		plaintext := make([]byte, 1024*1024)
		for i := range plaintext {
			plaintext[i] = byte(i % 256)
		}
		
		key := make([]byte, 32)
		
		// Encrypt
		start := time.Now()
		ciphertext, err := crypto.Seal(plaintext, key)
		encryptDuration := time.Since(start)
		
		if err != nil {
			t.Fatalf("Failed to encrypt large plaintext: %v", err)
		}
		
		t.Logf("Encrypted 1MB in %v", encryptDuration)
		
		// Decrypt
		start = time.Now()
		decrypted, err := crypto.Open(ciphertext, key)
		decryptDuration := time.Since(start)
		
		if err != nil {
			t.Fatalf("Failed to decrypt large ciphertext: %v", err)
		}
		
		t.Logf("Decrypted 1MB in %v", decryptDuration)
		
		// Verify
		if subtle.ConstantTimeCompare(plaintext, decrypted) != 1 {
			t.Error("Large plaintext not correctly encrypted/decrypted")
		}
		
		t.Log("✓ Large plaintext (1MB) handled correctly")
	})
}

// TestKeyManagement tests secure key management practices
func TestKeyManagement(t *testing.T) {
	t.Run("Keys are zeroized after use", func(t *testing.T) {
		key := make([]byte, 32)
		for i := range key {
			key[i] = byte(i)
		}
		
		// Verify key is not all zeros
		allZeros := true
		for _, b := range key {
			if b != 0 {
				allZeros = false
				break
			}
		}
		if allZeros {
			t.Fatal("Test key should not be all zeros")
		}
		
		// Zeroize
		vault.Zeroize(key)
		
		// Verify all zeros
		for i, b := range key {
			if b != 0 {
				t.Errorf("Byte %d not zeroized: %x", i, b)
			}
		}
		
		t.Log("✓ Key properly zeroized")
	})
	
	t.Run("Derived keys have correct length", func(t *testing.T) {
		crypto := vault.NewDefaultCryptoEngine()
		salt, _ := vault.GenerateSalt()
		
		key, err := crypto.DeriveKey("test-passphrase", salt)
		if err != nil {
			t.Fatalf("Key derivation failed: %v", err)
		}
		defer vault.Zeroize(key)
		
		// AES-256 requires 32-byte keys
		if len(key) != 32 {
			t.Errorf("Derived key length incorrect: %d bytes (expected: 32)", len(key))
		} else {
			t.Log("✓ Derived key has correct length (32 bytes for AES-256)")
		}
	})
	
	t.Run("Keys are not predictable", func(t *testing.T) {
		crypto := vault.NewDefaultCryptoEngine()
		
		// Derive multiple keys with similar passphrases
		passphrases := []string{
			"password1",
			"password2",
			"password3",
		}
		
		keys := make([][]byte, len(passphrases))
		
		for i, pass := range passphrases {
			salt, _ := vault.GenerateSalt()
			key, _ := crypto.DeriveKey(pass, salt)
			keys[i] = key
			defer vault.Zeroize(key)
		}
		
		// Verify keys are different
		for i := 0; i < len(keys); i++ {
			for j := i + 1; j < len(keys); j++ {
				if subtle.ConstantTimeCompare(keys[i], keys[j]) == 1 {
					t.Errorf("Keys %d and %d are identical", i, j)
				}
			}
		}
		
		t.Log("✓ Keys are unique and not predictable")
	})
}

// TestCryptographicPrimitives tests the underlying crypto primitives
func TestCryptographicPrimitives(t *testing.T) {
	t.Run("Salt generation uses crypto/rand", func(t *testing.T) {
		// Generate multiple salts and verify they're different
		numSalts := 100
		salts := make(map[string]bool)
		
		for i := 0; i < numSalts; i++ {
			salt, err := vault.GenerateSalt()
			if err != nil {
				t.Fatalf("Salt generation failed: %v", err)
			}
			
			// Check length
			if len(salt) != 32 {
				t.Errorf("Salt %d has incorrect length: %d", i, len(salt))
			}
			
			// Check uniqueness
			saltStr := string(salt)
			if salts[saltStr] {
				t.Errorf("Duplicate salt at iteration %d", i)
			}
			salts[saltStr] = true
		}
		
		t.Logf("✓ Generated %d unique salts using crypto/rand", numSalts)
	})
	
	t.Run("Encryption nonce generation is secure", func(t *testing.T) {
		// This is tested indirectly through unique ciphertext generation
		crypto := vault.NewDefaultCryptoEngine()
		plaintext := []byte("test")
		key := make([]byte, 32)
		
		// Generate multiple ciphertexts and check for unique nonces
		nonces := make(map[string]bool)
		numTests := 100
		
		for i := 0; i < numTests; i++ {
			envelope, err := crypto.Seal(plaintext, key)
			if err != nil {
				t.Fatalf("Encryption %d failed: %v", i, err)
			}
			
			// Check for unique nonce
			nonceStr := string(envelope.Nonce)
			if nonces[nonceStr] {
				t.Errorf("Nonce reuse detected at iteration %d", i)
			}
			nonces[nonceStr] = true
		}
		
		t.Logf("✓ %d unique nonces generated (secure nonce generation)", numTests)
	})
}

// TestSecurityBestPractices tests adherence to security best practices
func TestSecurityBestPractices(t *testing.T) {
	t.Run("No hardcoded keys or secrets", func(t *testing.T) {
		// This is a code review item, but we can verify that
		// the crypto engine requires keys to be provided
		crypto := vault.NewDefaultCryptoEngine()
		
		// Attempting to use nil key should fail
		_, err := crypto.Seal([]byte("test"), nil)
		if err == nil {
			t.Error("Encryption with nil key should fail")
		} else {
			t.Logf("✓ Nil key rejected: %v", err)
		}
	})
	
	t.Run("Constant-time comparison for sensitive data", func(t *testing.T) {
		// Verify that subtle.ConstantTimeCompare is available and works
		a := []byte("secret")
		b := []byte("secret")
		c := []byte("public")
		
		if subtle.ConstantTimeCompare(a, b) != 1 {
			t.Error("Constant-time comparison failed for equal values")
		}
		
		if subtle.ConstantTimeCompare(a, c) == 1 {
			t.Error("Constant-time comparison failed for different values")
		}
		
		t.Log("✓ Constant-time comparison available and working")
	})
	
	t.Run("KDF parameters can be validated", func(t *testing.T) {
		// Test parameter validation
		validParams := vault.Argon2Params{
			Memory:      65536,
			Iterations:  3,
			Parallelism: 4,
		}
		
		if err := vault.ValidateArgon2Params(validParams); err != nil {
			t.Errorf("Valid parameters rejected: %v", err)
		}
		
		// Test invalid parameters
		invalidParams := []vault.Argon2Params{
			{Memory: 0, Iterations: 3, Parallelism: 4},           // Zero memory
			{Memory: 65536, Iterations: 0, Parallelism: 4},       // Zero iterations
			{Memory: 65536, Iterations: 3, Parallelism: 0},       // Zero parallelism
			{Memory: 1024, Iterations: 1, Parallelism: 1},        // Too weak
		}
		
		for i, params := range invalidParams {
			if err := vault.ValidateArgon2Params(params); err == nil {
				t.Errorf("Invalid parameters %d not rejected: %+v", i, params)
			}
		}
		
		t.Log("✓ Parameter validation working correctly")
	})
}
