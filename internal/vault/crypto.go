package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/subtle"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/argon2"
)

const (
	// Crypto constants
	KeySize   = 32 // AES-256 key size
	SaltSize  = 32 // Salt size for Argon2id
	NonceSize = 12 // GCM nonce size
	TagSize   = 16 // GCM tag size

	// Envelope version
	EnvelopeVersion = 1

	// Default Argon2id parameters (tuned for ~300ms on modern hardware)
	DefaultArgon2Memory      = 64 * 1024 // 64 MB
	DefaultArgon2Iterations  = 3
	DefaultArgon2Parallelism = 4
)

var (
	ErrInvalidEnvelope   = errors.New("invalid envelope format")
	ErrInvalidVersion    = errors.New("unsupported envelope version")
	ErrDecryptionFailed  = errors.New("decryption failed")
	ErrInvalidKeySize    = errors.New("invalid key size")
	ErrInvalidNonceSize  = errors.New("invalid nonce size")
	ErrInvalidCiphertext = errors.New("invalid ciphertext")
)

// Argon2Params holds the key derivation parameters
type Argon2Params struct {
	Memory      uint32 `json:"memory"`
	Iterations  uint32 `json:"iterations"`
	Parallelism uint8  `json:"parallelism"`
}

// DefaultArgon2Params returns the default Argon2id parameters
func DefaultArgon2Params() Argon2Params {
	return Argon2Params{
		Memory:      DefaultArgon2Memory,
		Iterations:  DefaultArgon2Iterations,
		Parallelism: DefaultArgon2Parallelism,
	}
}

// Envelope represents the encrypted data structure
type Envelope struct {
	Version    uint8        `json:"version"`
	KDFParams  Argon2Params `json:"kdf_params"`
	Salt       []byte       `json:"salt"`
	Nonce      []byte       `json:"nonce"`
	Ciphertext []byte       `json:"ciphertext"`
	Tag        []byte       `json:"tag"`
}

// CryptoEngine handles all cryptographic operations
type CryptoEngine struct {
	params Argon2Params
}

// NewCryptoEngine creates a new crypto engine with specified parameters
func NewCryptoEngine(params Argon2Params) *CryptoEngine {
	return &CryptoEngine{
		params: params,
	}
}

// NewDefaultCryptoEngine creates a new crypto engine with default parameters
func NewDefaultCryptoEngine() *CryptoEngine {
	return NewCryptoEngine(DefaultArgon2Params())
}

// GenerateSalt creates a cryptographically secure random salt
func GenerateSalt() ([]byte, error) {
	salt := make([]byte, SaltSize)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}
	return salt, nil
}

// GenerateNonce creates a cryptographically secure random nonce
func GenerateNonce() ([]byte, error) {
	nonce := make([]byte, NonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}
	return nonce, nil
}

// DeriveKey derives a key from a passphrase using Argon2id
func (ce *CryptoEngine) DeriveKey(passphrase string, salt []byte) ([]byte, error) {
	if len(salt) != SaltSize {
		return nil, fmt.Errorf("invalid salt size: expected %d, got %d", SaltSize, len(salt))
	}

	start := time.Now()
	key := argon2.IDKey(
		[]byte(passphrase),
		salt,
		ce.params.Iterations,
		ce.params.Memory,
		ce.params.Parallelism,
		KeySize,
	)
	duration := time.Since(start)

	// Log timing for security analysis (should be 200-500ms)
	if duration < 100*time.Millisecond {
		fmt.Printf("Warning: Key derivation took only %v, consider increasing parameters\n", duration)
	} else if duration > 1*time.Second {
		fmt.Printf("Warning: Key derivation took %v, consider decreasing parameters\n", duration)
	}

	return key, nil
}

// Seal encrypts plaintext using AES-256-GCM
func (ce *CryptoEngine) Seal(plaintext []byte, key []byte) (*Envelope, error) {
	if len(key) != KeySize {
		return nil, ErrInvalidKeySize
	}

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate nonce
	nonce, err := GenerateNonce()
	if err != nil {
		return nil, err
	}

	// Encrypt the plaintext
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	// Split ciphertext and tag
	if len(ciphertext) < TagSize {
		return nil, ErrInvalidCiphertext
	}

	actualCiphertext := ciphertext[:len(ciphertext)-TagSize]
	tag := ciphertext[len(ciphertext)-TagSize:]

	envelope := &Envelope{
		Version:    EnvelopeVersion,
		KDFParams:  ce.params,
		Salt:       nil, // Salt is set separately for vault-level operations
		Nonce:      nonce,
		Ciphertext: actualCiphertext,
		Tag:        tag,
	}

	return envelope, nil
}

// Open decrypts ciphertext using AES-256-GCM
func (ce *CryptoEngine) Open(envelope *Envelope, key []byte) ([]byte, error) {
	if len(key) != KeySize {
		return nil, ErrInvalidKeySize
	}

	if envelope.Version != EnvelopeVersion {
		return nil, ErrInvalidVersion
	}

	if len(envelope.Nonce) != NonceSize {
		return nil, ErrInvalidNonceSize
	}

	if len(envelope.Tag) != TagSize {
		return nil, fmt.Errorf("invalid tag size: expected %d, got %d", TagSize, len(envelope.Tag))
	}

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Reconstruct the sealed data (ciphertext + tag)
	sealedData := make([]byte, len(envelope.Ciphertext)+len(envelope.Tag))
	copy(sealedData, envelope.Ciphertext)
	copy(sealedData[len(envelope.Ciphertext):], envelope.Tag)

	// Decrypt
	plaintext, err := gcm.Open(nil, envelope.Nonce, sealedData, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	return plaintext, nil
}

// SealWithPassphrase encrypts plaintext with a passphrase (generates new salt)
func (ce *CryptoEngine) SealWithPassphrase(plaintext []byte, passphrase string) (*Envelope, error) {
	salt, err := GenerateSalt()
	if err != nil {
		return nil, err
	}

	key, err := ce.DeriveKey(passphrase, salt)
	if err != nil {
		return nil, err
	}
	defer Zeroize(key)

	envelope, err := ce.Seal(plaintext, key)
	if err != nil {
		return nil, err
	}

	envelope.Salt = salt
	return envelope, nil
}

// OpenWithPassphrase decrypts ciphertext with a passphrase
func (ce *CryptoEngine) OpenWithPassphrase(envelope *Envelope, passphrase string) ([]byte, error) {
	if envelope.Salt == nil {
		return nil, errors.New("envelope missing salt")
	}

	// Use the KDF parameters from the envelope
	engine := NewCryptoEngine(envelope.KDFParams)
	key, err := engine.DeriveKey(passphrase, envelope.Salt)
	if err != nil {
		return nil, err
	}
	defer Zeroize(key)

	return engine.Open(envelope, key)
}

// Zeroize securely clears a byte slice
func Zeroize(data []byte) {
	for i := range data {
		data[i] = 0
	}
}

// ZeroizeString securely clears a string by converting to bytes and zeroing
func ZeroizeString(s *string) {
	if s == nil {
		return
	}
	// Convert string to byte slice and zero it
	b := []byte(*s)
	Zeroize(b)
	*s = ""
}

// SecureCompare performs constant-time comparison of two byte slices
func SecureCompare(a, b []byte) bool {
	return subtle.ConstantTimeCompare(a, b) == 1
}

// ValidateArgon2Params validates Argon2id parameters
func ValidateArgon2Params(params Argon2Params) error {
	if params.Memory < 1024 {
		return errors.New("memory parameter too low (minimum 1024 KB)")
	}
	if params.Memory > 1024*1024 {
		return errors.New("memory parameter too high (maximum 1 GB)")
	}
	if params.Iterations < 1 {
		return errors.New("iterations parameter too low (minimum 1)")
	}
	if params.Iterations > 100 {
		return errors.New("iterations parameter too high (maximum 100)")
	}
	if params.Parallelism < 1 {
		return errors.New("parallelism parameter too low (minimum 1)")
	}
	if params.Parallelism >= 255 {
		return errors.New("parallelism parameter too high (maximum 254)")
	}
	return nil
}

// BenchmarkKDF measures the time taken for key derivation with given parameters
func BenchmarkKDF(params Argon2Params, passphrase string, salt []byte) time.Duration {
	start := time.Now()
	argon2.IDKey(
		[]byte(passphrase),
		salt,
		params.Iterations,
		params.Memory,
		params.Parallelism,
		KeySize,
	)
	return time.Since(start)
}

// TuneArgon2Params automatically tunes Argon2id parameters for target duration
func TuneArgon2Params(targetDuration time.Duration, testPassphrase string) (Argon2Params, error) {
	salt, err := GenerateSalt()
	if err != nil {
		return Argon2Params{}, err
	}

	params := DefaultArgon2Params()

	// Start with default and adjust
	duration := BenchmarkKDF(params, testPassphrase, salt)

	// Adjust iterations to get close to target
	if duration < targetDuration {
		// Need to increase difficulty
		for duration < targetDuration && params.Iterations < 10 {
			params.Iterations++
			duration = BenchmarkKDF(params, testPassphrase, salt)
		}
		// If still too fast, increase memory
		for duration < targetDuration && params.Memory < 128*1024 {
			params.Memory += 8 * 1024
			duration = BenchmarkKDF(params, testPassphrase, salt)
		}
	} else if duration > targetDuration*2 {
		// Too slow, decrease difficulty
		for duration > targetDuration*2 && params.Iterations > 1 {
			params.Iterations--
			duration = BenchmarkKDF(params, testPassphrase, salt)
		}
		// If still too slow, decrease memory
		for duration > targetDuration*2 && params.Memory > 8*1024 {
			params.Memory -= 8 * 1024
			duration = BenchmarkKDF(params, testPassphrase, salt)
		}
	}

	return params, nil
}

// EnvelopeToBytes serializes an envelope to bytes for storage
func EnvelopeToBytes(envelope *Envelope) []byte {
	// Simple binary format: version(1) + kdf_params(9) + salt_len(4) + salt + nonce_len(4) + nonce + ciphertext_len(4) + ciphertext + tag_len(4) + tag
	buf := make([]byte, 0, 1+9+4+len(envelope.Salt)+4+len(envelope.Nonce)+4+len(envelope.Ciphertext)+4+len(envelope.Tag))

	// Version
	buf = append(buf, envelope.Version)

	// KDF params (9 bytes: memory(4) + iterations(4) + parallelism(1))
	memBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(memBytes, envelope.KDFParams.Memory)
	buf = append(buf, memBytes...)

	iterBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(iterBytes, envelope.KDFParams.Iterations)
	buf = append(buf, iterBytes...)

	buf = append(buf, envelope.KDFParams.Parallelism)

	// Salt
	saltLenBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(saltLenBytes, uint32(len(envelope.Salt)))
	buf = append(buf, saltLenBytes...)
	buf = append(buf, envelope.Salt...)

	// Nonce
	nonceLenBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(nonceLenBytes, uint32(len(envelope.Nonce)))
	buf = append(buf, nonceLenBytes...)
	buf = append(buf, envelope.Nonce...)

	// Ciphertext
	ciphertextLenBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(ciphertextLenBytes, uint32(len(envelope.Ciphertext)))
	buf = append(buf, ciphertextLenBytes...)
	buf = append(buf, envelope.Ciphertext...)

	// Tag
	tagLenBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(tagLenBytes, uint32(len(envelope.Tag)))
	buf = append(buf, tagLenBytes...)
	buf = append(buf, envelope.Tag...)

	return buf
}

// EnvelopeFromBytes deserializes an envelope from bytes
func EnvelopeFromBytes(data []byte) (*Envelope, error) {
	if len(data) < 1+9+4+4+4+4 { // Minimum size
		return nil, ErrInvalidEnvelope
	}

	offset := 0

	// Version
	version := data[offset]
	offset++

	if version != EnvelopeVersion {
		return nil, ErrInvalidVersion
	}

	// KDF params
	memory := binary.LittleEndian.Uint32(data[offset : offset+4])
	offset += 4
	iterations := binary.LittleEndian.Uint32(data[offset : offset+4])
	offset += 4
	parallelism := data[offset]
	offset++

	kdfParams := Argon2Params{
		Memory:      memory,
		Iterations:  iterations,
		Parallelism: parallelism,
	}

	// Salt
	if offset+4 > len(data) {
		return nil, ErrInvalidEnvelope
	}
	saltLen := binary.LittleEndian.Uint32(data[offset : offset+4])
	offset += 4
	if offset+int(saltLen) > len(data) {
		return nil, ErrInvalidEnvelope
	}
	salt := make([]byte, saltLen)
	copy(salt, data[offset:offset+int(saltLen)])
	offset += int(saltLen)

	// Nonce
	if offset+4 > len(data) {
		return nil, ErrInvalidEnvelope
	}
	nonceLen := binary.LittleEndian.Uint32(data[offset : offset+4])
	offset += 4
	if offset+int(nonceLen) > len(data) {
		return nil, ErrInvalidEnvelope
	}
	nonce := make([]byte, nonceLen)
	copy(nonce, data[offset:offset+int(nonceLen)])
	offset += int(nonceLen)

	// Ciphertext
	if offset+4 > len(data) {
		return nil, ErrInvalidEnvelope
	}
	ciphertextLen := binary.LittleEndian.Uint32(data[offset : offset+4])
	offset += 4
	if offset+int(ciphertextLen) > len(data) {
		return nil, ErrInvalidEnvelope
	}
	ciphertext := make([]byte, ciphertextLen)
	copy(ciphertext, data[offset:offset+int(ciphertextLen)])
	offset += int(ciphertextLen)

	// Tag
	if offset+4 > len(data) {
		return nil, ErrInvalidEnvelope
	}
	tagLen := binary.LittleEndian.Uint32(data[offset : offset+4])
	offset += 4
	if offset+int(tagLen) > len(data) {
		return nil, ErrInvalidEnvelope
	}
	tag := make([]byte, tagLen)
	copy(tag, data[offset:offset+int(tagLen)])

	return &Envelope{
		Version:    version,
		KDFParams:  kdfParams,
		Salt:       salt,
		Nonce:      nonce,
		Ciphertext: ciphertext,
		Tag:        tag,
	}, nil
}
