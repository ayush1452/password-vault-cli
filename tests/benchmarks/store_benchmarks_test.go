package benchmarks_test

import (
	"path/filepath"
	"testing"

	"github.com/vault-cli/vault/internal/domain"
	"github.com/vault-cli/vault/internal/store"
	"github.com/vault-cli/vault/internal/vault"
)

// BenchmarkStoreCreate benchmarks store creation
func BenchmarkStoreCreate(b *testing.B) {
	tempDir := b.TempDir()

	passphrase := "benchmark"
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
	vaultPath := filepath.Join(tempDir, "bench.vault")
	vaultStore.CreateVault(vaultPath, masterKey, kdfParamsMap)
	vaultStore.OpenVault(vaultPath, masterKey)
	defer vaultStore.CloseVault()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		entry := &domain.Entry{
			Name:     string(rune('a' + i%26)),
			Username: "user",
			Password: []byte("pass"),
		}
		vaultStore.CreateEntry("default", entry)
	}
}

// BenchmarkStoreRead benchmarks store reads
func BenchmarkStoreRead(b *testing.B) {
	tempDir := b.TempDir()

	passphrase := "benchmark"
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
	vaultPath := filepath.Join(tempDir, "bench.vault")
	vaultStore.CreateVault(vaultPath, masterKey, kdfParamsMap)
	vaultStore.OpenVault(vaultPath, masterKey)
	defer vaultStore.CloseVault()

	entry := &domain.Entry{Name: "test", Username: "user", Password: []byte("pass")}
	vaultStore.CreateEntry("default", entry)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		vaultStore.GetEntry("default", "test")
	}
}

// BenchmarkStoreUpdate benchmarks store updates
func BenchmarkStoreUpdate(b *testing.B) {
	tempDir := b.TempDir()

	passphrase := "benchmark"
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
	vaultPath := filepath.Join(tempDir, "bench.vault")
	vaultStore.CreateVault(vaultPath, masterKey, kdfParamsMap)
	vaultStore.OpenVault(vaultPath, masterKey)
	defer vaultStore.CloseVault()

	entry := &domain.Entry{Name: "test", Username: "user", Password: []byte("pass")}
	vaultStore.CreateEntry("default", entry)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		updated := &domain.Entry{Name: "test", Username: "updated", Password: []byte("newpass")}
		vaultStore.UpdateEntry("default", "test", updated)
	}
}

// BenchmarkStoreDelete benchmarks store deletions
func BenchmarkStoreDelete(b *testing.B) {
	tempDir := b.TempDir()

	passphrase := "benchmark"
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
	vaultPath := filepath.Join(tempDir, "bench.vault")
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

// BenchmarkStoreList benchmarks listing entries
func BenchmarkStoreListSmall(b *testing.B) {
	benchmarkStoreList(b, 10)
}

func BenchmarkStoreListMedium(b *testing.B) {
	benchmarkStoreList(b, 100)
}

func BenchmarkStoreListLarge(b *testing.B) {
	benchmarkStoreList(b, 1000)
}

func benchmarkStoreList(b *testing.B, numEntries int) {
	tempDir := b.TempDir()

	passphrase := "benchmark"
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
	vaultPath := filepath.Join(tempDir, "bench.vault")
	vaultStore.CreateVault(vaultPath, masterKey, kdfParamsMap)
	vaultStore.OpenVault(vaultPath, masterKey)
	defer vaultStore.CloseVault()

	// Add entries
	for i := 0; i < numEntries; i++ {
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
