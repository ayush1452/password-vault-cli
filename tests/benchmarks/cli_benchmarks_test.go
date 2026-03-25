package benchmarks_test

import (
	"path/filepath"
	"testing"

	"github.com/vault-cli/vault/internal/domain"
	"github.com/vault-cli/vault/internal/store"
	"github.com/vault-cli/vault/internal/vault"
)

// BenchmarkVaultInit benchmarks vault initialization
func BenchmarkVaultInit(b *testing.B) {
	tempDir := b.TempDir()

	for i := 0; i < b.N; i++ {
		vaultPath := filepath.Join(tempDir, "bench_init.vault")

		passphrase := "benchmark-passphrase"
		crypto := vault.NewDefaultCryptoEngine()
		salt, _ := vault.GenerateSalt()
		masterKey, _ := crypto.DeriveKey(passphrase, salt)

		kdfParams := vault.DefaultArgon2Params()
		kdfParamsMap := map[string]interface{}{
			"memory":      kdfParams.Memory,
			"iterations":  kdfParams.Iterations,
			"parallelism": kdfParams.Parallelism,
			"salt":        salt,
		}

		vaultStore := store.NewBoltStore()
		vaultStore.CreateVault(vaultPath, masterKey, kdfParamsMap)
		vaultStore.CloseVault()
		vault.Zeroize(masterKey)
	}
}

// BenchmarkVaultUnlock benchmarks vault unlocking
func BenchmarkVaultUnlock(b *testing.B) {
	tempDir := b.TempDir()
	vaultPath := filepath.Join(tempDir, "bench_unlock.vault")

	// Setup
	passphrase := "benchmark-passphrase"
	crypto := vault.NewDefaultCryptoEngine()
	salt, _ := vault.GenerateSalt()
	masterKey, _ := crypto.DeriveKey(passphrase, salt)

	kdfParams := vault.DefaultArgon2Params()
	kdfParamsMap := map[string]interface{}{
		"memory":      kdfParams.Memory,
		"iterations":  kdfParams.Iterations,
		"parallelism": kdfParams.Parallelism,
		"salt":        salt,
	}

	vaultStore := store.NewBoltStore()
	vaultStore.CreateVault(vaultPath, masterKey, kdfParamsMap)
	vaultStore.CloseVault()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		vaultStore.OpenVault(vaultPath, masterKey)
		vaultStore.CloseVault()
	}

	vault.Zeroize(masterKey)
}

// BenchmarkEntryAdd benchmarks adding entries
func BenchmarkEntryAdd(b *testing.B) {
	tempDir := b.TempDir()
	vaultPath := filepath.Join(tempDir, "bench_add.vault")

	// Setup
	passphrase := "benchmark-passphrase"
	crypto := vault.NewDefaultCryptoEngine()
	salt, _ := vault.GenerateSalt()
	masterKey, _ := crypto.DeriveKey(passphrase, salt)
	defer vault.Zeroize(masterKey)

	kdfParams := vault.DefaultArgon2Params()
	kdfParamsMap := map[string]interface{}{
		"memory":      kdfParams.Memory,
		"iterations":  kdfParams.Iterations,
		"parallelism": kdfParams.Parallelism,
		"salt":        salt,
	}

	vaultStore := store.NewBoltStore()
	vaultStore.CreateVault(vaultPath, masterKey, kdfParamsMap)
	vaultStore.OpenVault(vaultPath, masterKey)
	defer vaultStore.CloseVault()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		entry := &domain.Entry{
			Name:     string(rune('a' + i%26)),
			Username: "user",
			Password: []byte("password"),
		}
		vaultStore.CreateEntry("default", entry)
	}
}

// BenchmarkEntryGet benchmarks retrieving entries
func BenchmarkEntryGet(b *testing.B) {
	tempDir := b.TempDir()
	vaultPath := filepath.Join(tempDir, "bench_get.vault")

	// Setup
	passphrase := "benchmark-passphrase"
	crypto := vault.NewDefaultCryptoEngine()
	salt, _ := vault.GenerateSalt()
	masterKey, _ := crypto.DeriveKey(passphrase, salt)
	defer vault.Zeroize(masterKey)

	kdfParams := vault.DefaultArgon2Params()
	kdfParamsMap := map[string]interface{}{
		"memory":      kdfParams.Memory,
		"iterations":  kdfParams.Iterations,
		"parallelism": kdfParams.Parallelism,
		"salt":        salt,
	}

	vaultStore := store.NewBoltStore()
	vaultStore.CreateVault(vaultPath, masterKey, kdfParamsMap)
	vaultStore.OpenVault(vaultPath, masterKey)
	defer vaultStore.CloseVault()

	// Add test entry
	entry := &domain.Entry{Name: "test", Username: "user", Password: []byte("pass")}
	vaultStore.CreateEntry("default", entry)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		vaultStore.GetEntry("default", "test")
	}
}

// BenchmarkEntryList benchmarks listing entries
func BenchmarkEntryList(b *testing.B) {
	tempDir := b.TempDir()
	vaultPath := filepath.Join(tempDir, "bench_list.vault")

	// Setup
	passphrase := "benchmark-passphrase"
	crypto := vault.NewDefaultCryptoEngine()
	salt, _ := vault.GenerateSalt()
	masterKey, _ := crypto.DeriveKey(passphrase, salt)
	defer vault.Zeroize(masterKey)

	kdfParams := vault.DefaultArgon2Params()
	kdfParamsMap := map[string]interface{}{
		"memory":      kdfParams.Memory,
		"iterations":  kdfParams.Iterations,
		"parallelism": kdfParams.Parallelism,
		"salt":        salt,
	}

	vaultStore := store.NewBoltStore()
	vaultStore.CreateVault(vaultPath, masterKey, kdfParamsMap)
	vaultStore.OpenVault(vaultPath, masterKey)
	defer vaultStore.CloseVault()

	// Add 100 entries
	for i := 0; i < 100; i++ {
		entry := &domain.Entry{
			Name:     string(rune('a'+i%26)) + string(rune('0'+i/26)),
			Username: "user",
			Password: []byte("pass"),
		}
		vaultStore.CreateEntry("default", entry)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		vaultStore.ListEntries("default", nil)
	}
}

// BenchmarkEntryUpdate benchmarks updating entries
func BenchmarkEntryUpdate(b *testing.B) {
	tempDir := b.TempDir()
	vaultPath := filepath.Join(tempDir, "bench_update.vault")

	// Setup
	passphrase := "benchmark-passphrase"
	crypto := vault.NewDefaultCryptoEngine()
	salt, _ := vault.GenerateSalt()
	masterKey, _ := crypto.DeriveKey(passphrase, salt)
	defer vault.Zeroize(masterKey)

	kdfParams := vault.DefaultArgon2Params()
	kdfParamsMap := map[string]interface{}{
		"memory":      kdfParams.Memory,
		"iterations":  kdfParams.Iterations,
		"parallelism": kdfParams.Parallelism,
		"salt":        salt,
	}

	vaultStore := store.NewBoltStore()
	vaultStore.CreateVault(vaultPath, masterKey, kdfParamsMap)
	vaultStore.OpenVault(vaultPath, masterKey)
	defer vaultStore.CloseVault()

	// Add test entry
	entry := &domain.Entry{Name: "test", Username: "user", Password: []byte("pass")}
	vaultStore.CreateEntry("default", entry)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		updated := &domain.Entry{Name: "test", Username: "updated", Password: []byte("newpass")}
		vaultStore.UpdateEntry("default", "test", updated)
	}
}

// BenchmarkEntryDelete benchmarks deleting entries
func BenchmarkEntryDelete(b *testing.B) {
	tempDir := b.TempDir()
	vaultPath := filepath.Join(tempDir, "bench_delete.vault")

	// Setup
	passphrase := "benchmark-passphrase"
	crypto := vault.NewDefaultCryptoEngine()
	salt, _ := vault.GenerateSalt()
	masterKey, _ := crypto.DeriveKey(passphrase, salt)
	defer vault.Zeroize(masterKey)

	kdfParams := vault.DefaultArgon2Params()
	kdfParamsMap := map[string]interface{}{
		"memory":      kdfParams.Memory,
		"iterations":  kdfParams.Iterations,
		"parallelism": kdfParams.Parallelism,
		"salt":        salt,
	}

	vaultStore := store.NewBoltStore()
	vaultStore.CreateVault(vaultPath, masterKey, kdfParamsMap)
	vaultStore.OpenVault(vaultPath, masterKey)
	defer vaultStore.CloseVault()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		entry := &domain.Entry{
			Name:     string(rune('a' + i%26)),
			Username: "user",
			Password: []byte("pass"),
		}
		vaultStore.CreateEntry("default", entry)
		b.StartTimer()

		vaultStore.DeleteEntry("default", entry.Name)
	}
}
