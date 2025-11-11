package main

import (
	"fmt"
	"log"
	"time"

	"github.com/vault-cli/vault/internal/vault"
)

func main() {
	fmt.Println("Password Vault CLI - Cryptography Demo")
	fmt.Println("=====================================")

	// Create crypto engine with default parameters
	engine := vault.NewDefaultCryptoEngine()

	// Demo 1: Key derivation
	fmt.Println("\n1. Key Derivation Demo")
	passphrase := "my-super-secure-passphrase-123"
	salt, err := vault.GenerateSalt()
	if err != nil {
		log.Fatal("Failed to generate salt:", err)
	}

	start := time.Now()
	key, err := engine.DeriveKey(passphrase, salt)
	if err != nil {
		log.Fatal("Failed to derive key:", err)
	}
	duration := time.Since(start)

	fmt.Printf("Key derivation took: %v\n", duration)
	fmt.Printf("Derived key length: %d bytes\n", len(key))
	defer vault.Zeroize(key) // Always clean up keys

	// Demo 2: Encryption/Decryption with passphrase
	fmt.Println("\n2. Encryption/Decryption Demo")
	secretData := []byte("This is my secret password: admin123!")

	fmt.Printf("Original data: %s\n", string(secretData))

	// Encrypt
	envelope, err := engine.SealWithPassphrase(secretData, passphrase)
	if err != nil {
		log.Fatal("Failed to encrypt:", err)
	}

	fmt.Printf("Encrypted envelope version: %d\n", envelope.Version)
	fmt.Printf("Salt length: %d bytes\n", len(envelope.Salt))
	fmt.Printf("Nonce length: %d bytes\n", len(envelope.Nonce))
	fmt.Printf("Ciphertext length: %d bytes\n", len(envelope.Ciphertext))
	fmt.Printf("Tag length: %d bytes\n", len(envelope.Tag))

	// Decrypt
	decrypted, err := engine.OpenWithPassphrase(envelope, passphrase)
	if err != nil {
		log.Fatal("Failed to decrypt:", err)
	}

	fmt.Printf("Decrypted data: %s\n", string(decrypted))

	// Demo 3: Tamper detection
	fmt.Println("\n3. Tamper Detection Demo")

	// Create a copy and tamper with it
	tamperedEnvelope := *envelope
	tamperedEnvelope.Ciphertext = make([]byte, len(envelope.Ciphertext))
	copy(tamperedEnvelope.Ciphertext, envelope.Ciphertext)
	tamperedEnvelope.Ciphertext[0] ^= 1 // Flip one bit

	_, err = engine.OpenWithPassphrase(&tamperedEnvelope, passphrase)
	if err != nil {
		fmt.Printf("✓ Tamper detection worked: %v\n", err)
	} else {
		fmt.Println("✗ Tamper detection failed!")
	}

	// Demo 4: Serialization
	fmt.Println("\n4. Serialization Demo")

	serialized := vault.EnvelopeToBytes(envelope)
	fmt.Printf("Serialized envelope size: %d bytes\n", len(serialized))

	deserialized, err := vault.EnvelopeFromBytes(serialized)
	if err != nil {
		log.Fatal("Failed to deserialize:", err)
	}

	// Verify it still works
	decrypted2, err := engine.OpenWithPassphrase(deserialized, passphrase)
	if err != nil {
		log.Fatal("Failed to decrypt deserialized:", err)
	}

	fmt.Printf("✓ Serialization successful: %s\n", string(decrypted2))

	// Demo 5: KDF parameter tuning
	fmt.Println("\n5. KDF Parameter Tuning Demo")

	targetDuration := 250 * time.Millisecond
	tunedParams, err := vault.TuneArgon2Params(targetDuration, "test-passphrase")
	if err != nil {
		log.Fatal("Failed to tune parameters:", err)
	}

	fmt.Printf("Tuned parameters:\n")
	fmt.Printf("  Memory: %d KB\n", tunedParams.Memory)
	fmt.Printf("  Iterations: %d\n", tunedParams.Iterations)
	fmt.Printf("  Parallelism: %d\n", tunedParams.Parallelism)

	// Test the tuned parameters
	testSalt, err := vault.GenerateSalt()
	if err != nil {
		log.Fatal("Failed to generate salt:", err)
	}
	actualDuration := vault.BenchmarkKDF(tunedParams, "test-passphrase", testSalt)
	fmt.Printf("Actual duration with tuned params: %v\n", actualDuration)

	fmt.Println("\n✓ All crypto demos completed successfully!")
}
