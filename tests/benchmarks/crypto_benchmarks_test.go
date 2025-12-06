package benchmarks_test

import (
	"testing"

	"github.com/vault-cli/vault/internal/vault"
)

// BenchmarkKeyDerivation benchmarks Argon2id key derivation
func BenchmarkKeyDerivation(b *testing.B) {
	passphrase := "benchmark-passphrase"
	crypto := vault.NewDefaultCryptoEngine()
	salt, _ := vault.GenerateSalt()
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		key, _ := crypto.DeriveKey(passphrase, salt)
		vault.Zeroize(key)
	}
}

// BenchmarkKeyDerivationParallel benchmarks parallel key derivation
func BenchmarkKeyDerivationParallel(b *testing.B) {
	passphrase := "benchmark-passphrase"
	crypto := vault.NewDefaultCryptoEngine()
	salt, _ := vault.GenerateSalt()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			key, _ := crypto.DeriveKey(passphrase, salt)
			vault.Zeroize(key)
		}
	})
}

// BenchmarkSaltGeneration benchmarks salt generation
func BenchmarkSaltGeneration(b *testing.B) {
	for i := 0; i < b.N; i++ {
		vault.GenerateSalt()
	}
}

// BenchmarkZeroization benchmarks memory zeroization
func BenchmarkZeroization(b *testing.B) {
	data := make([]byte, 32)
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		vault.Zeroize(data)
	}
}

// BenchmarkKeyDerivationWithDifferentParams benchmarks different KDF parameters
func BenchmarkKeyDerivationMemory64MB(b *testing.B) {
	passphrase := "benchmark"
	params := vault.Argon2Params{Memory: 65536, Iterations: 3, Parallelism: 4}
	crypto := vault.NewCryptoEngine(params)
	salt, _ := vault.GenerateSalt()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key, _ := crypto.DeriveKey(passphrase, salt)
		vault.Zeroize(key)
	}
}

func BenchmarkKeyDerivationMemory128MB(b *testing.B) {
	passphrase := "benchmark"
	params := vault.Argon2Params{Memory: 131072, Iterations: 3, Parallelism: 4}
	crypto := vault.NewCryptoEngine(params)
	salt, _ := vault.GenerateSalt()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key, _ := crypto.DeriveKey(passphrase, salt)
		vault.Zeroize(key)
	}
}

func BenchmarkKeyDerivationIterations5(b *testing.B) {
	passphrase := "benchmark"
	params := vault.Argon2Params{Memory: 65536, Iterations: 5, Parallelism: 4}
	crypto := vault.NewCryptoEngine(params)
	salt, _ := vault.GenerateSalt()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key, _ := crypto.DeriveKey(passphrase, salt)
		vault.Zeroize(key)
	}
}
