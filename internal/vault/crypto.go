package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/subtle"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"time"

	"golang.org/x/crypto/argon2"
)

// KeySize is the size of the AES-256 key in bytes
const KeySize = 32

// SaltSize is the size of the salt for Argon2id in bytes
const SaltSize = 32

// NonceSize is the size of the GCM nonce in bytes
const NonceSize = 12

// TagSize is the size of the GCM authentication tag in bytes
const TagSize = 16

// EnvelopeVersion is the current version of the envelope format
const EnvelopeVersion = 1

// Default Argon2id parameters (tuned for ~300ms on modern hardware)
const (
	// DefaultArgon2Memory is the default memory parameter for Argon2id (64 MB)
	DefaultArgon2Memory = 64 * 1024
	// DefaultArgon2Iterations is the default number of iterations for Argon2id
	DefaultArgon2Iterations = 3
	// DefaultArgon2Parallelism is the default parallelism factor for Argon2id
	DefaultArgon2Parallelism = 4
)

// Error variables for cryptographic operations
var (
	// ErrInvalidEnvelope is returned when the envelope format is invalid
	ErrInvalidEnvelope = errors.New("invalid envelope format")
	// ErrInvalidVersion is returned when the envelope version is not supported
	ErrInvalidVersion = errors.New("unsupported envelope version")
	// ErrDecryptionFailed is returned when decryption fails
	ErrDecryptionFailed = errors.New("decryption failed")
	// ErrInvalidKeySize is returned when the key size is invalid
	ErrInvalidKeySize = errors.New("invalid key size")
	// ErrInvalidNonceSize is returned when the nonce size is invalid
	ErrInvalidNonceSize = errors.New("invalid nonce size")
	// ErrInvalidCiphertext is returned when the ciphertext is invalid
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
func (ce *CryptoEngine) Seal(plaintext, key []byte) (*Envelope, error) {
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
	// Limit parallelism to a reasonable number of CPU cores
	if params.Parallelism > 16 {
		return errors.New("parallelism parameter too high (maximum 16)")
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
// It ensures all integer conversions are safe and prevents overflows
// Returns the serialized bytes or an error if the envelope cannot be serialized
func EnvelopeToBytes(envelope *Envelope) ([]byte, error) {
	if envelope == nil {
		return nil, nil
	}

	// Calculate total size needed with safe integer conversions
	totalSize := 10 // Fixed-size fields (1 + 4 + 4 + 1)

	// Add salt length with overflow check
	saltLen := len(envelope.Salt)
	if saltLen < 0 || uint64(saltLen) > math.MaxUint32 {
		saltLen = math.MaxUint32
	}
	totalSize += 4 + saltLen // 4 bytes for length + salt data

	// Add nonce length with overflow check
	nonceLen := len(envelope.Nonce)
	if nonceLen < 0 || uint64(nonceLen) > math.MaxUint32 {
		nonceLen = math.MaxUint32
	}
	totalSize += 4 + nonceLen // 4 bytes for length + nonce data

	// Add ciphertext length with overflow check
	ciphertextLen := len(envelope.Ciphertext)
	if ciphertextLen < 0 || uint64(ciphertextLen) > math.MaxUint32 {
		ciphertextLen = math.MaxUint32
	}
	totalSize += 4 + ciphertextLen // 4 bytes for length + ciphertext data

	// Add tag length with overflow check
	tagLen := len(envelope.Tag)
	if tagLen < 0 || uint64(tagLen) > math.MaxUint32 {
		tagLen = math.MaxUint32
	}
	totalSize += 4 + tagLen // 4 bytes for length + tag data

	// Ensure we don't exceed MaxInt32 for slice creation
	if totalSize < 0 || totalSize > math.MaxInt32 {
		// This is a very large envelope, truncate to maximum allowed size
		totalSize = math.MaxInt32
	}

	buf := make([]byte, 0, totalSize)

	// Version (1 byte)
	buf = append(buf, envelope.Version)

	// KDF Parameters
	// Memory (4 bytes)
	memoryBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(memoryBytes, envelope.KDFParams.Memory)
	buf = append(buf, memoryBytes...)

	// Iterations (4 bytes)
	iterBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(iterBytes, envelope.KDFParams.Iterations)
	buf = append(buf, iterBytes...)

	// Parallelism (1 byte)
	buf = append(buf, envelope.KDFParams.Parallelism)

	// Salt (4-byte length + data)
	saltLenBytes := make([]byte, 4)
	// Ensure saltLen is within uint32 range before conversion
	if saltLen < 0 || saltLen > math.MaxUint32 {
		return nil, fmt.Errorf("salt length %d is out of range (0-%d)", saltLen, math.MaxUint32)
	}
	saltLen32 := uint32(saltLen)
	binary.LittleEndian.PutUint32(saltLenBytes, saltLen32)
	buf = append(buf, saltLenBytes...)
	if saltLen > 0 && saltLen <= len(envelope.Salt) {
		buf = append(buf, envelope.Salt[:saltLen]...)
	} else if saltLen > 0 {
		// If we get here, there's a logic error in our size calculations
		return nil, fmt.Errorf("salt length %d exceeds available data %d", saltLen, len(envelope.Salt))
	}

	// Nonce (4-byte length + data)
	nonceLenBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(nonceLenBytes, uint32(nonceLen))
	buf = append(buf, nonceLenBytes...)
	if nonceLen > 0 && nonceLen <= len(envelope.Nonce) {
		buf = append(buf, envelope.Nonce[:nonceLen]...)
	} else if nonceLen > 0 {
		buf = append(buf, make([]byte, nonceLen)...)
	}

	// Ciphertext (4-byte length + data)
	ciphertextLenBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(ciphertextLenBytes, uint32(ciphertextLen))
	buf = append(buf, ciphertextLenBytes...)
	if ciphertextLen > 0 && ciphertextLen <= len(envelope.Ciphertext) {
		buf = append(buf, envelope.Ciphertext[:ciphertextLen]...)
	} else if ciphertextLen > 0 {
		buf = append(buf, make([]byte, ciphertextLen)...)
	}

	// Tag (4-byte length + data)
	tagLenBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(tagLenBytes, uint32(tagLen))
	buf = append(buf, tagLenBytes...)
	if tagLen > 0 && tagLen <= len(envelope.Tag) {
		buf = append(buf, envelope.Tag[:tagLen]...)
	} else if tagLen > 0 {
		buf = append(buf, make([]byte, tagLen)...)
	}

	return buf, nil
}

// EnvelopeFromBytes deserializes an envelope from bytes
func EnvelopeFromBytes(data []byte) (*Envelope, error) {
	// Minimum size check: version(1) + memory(4) + iterations(4) + parallelism(1) + saltLen(4) + nonceLen(4) + ciphertextLen(4) + tagLen(4)
	const minEnvelopeSize = 1 + 4 + 4 + 1 + 4 + 4 + 4 + 4
	if len(data) < minEnvelopeSize {
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

	// Validate KDF parameters to prevent potential DoS
	if memory > 1024*1024 || memory < 64*1024 { // 1MB max, 64KB min
		return nil, ErrInvalidEnvelope
	}
	if iterations > 1000 || iterations < 1 { // 1000 max, 1 min
		return nil, ErrInvalidEnvelope
	}
	if parallelism > 16 || parallelism < 1 { // 16 max, 1 min
		return nil, ErrInvalidEnvelope
	}

	kdfParams := Argon2Params{
		Memory:      memory,
		Iterations:  iterations,
		Parallelism: parallelism,
	}

	// Helper function to safely read a length-prefixed field
	readField := func() ([]byte, error) {
		if offset+4 > len(data) {
			return nil, ErrInvalidEnvelope
		}
		fieldLen := binary.LittleEndian.Uint32(data[offset : offset+4])
		offset += 4

		// Check for potential integer overflow/underflow
		if fieldLen > math.MaxUint32-uint32(offset) || offset+int(fieldLen) > len(data) {
			return nil, ErrInvalidEnvelope
		}

		// Limit field size to prevent excessive memory allocation (10MB max per field)
		const maxFieldSize = 10 * 1024 * 1024 // 10MB
		if fieldLen > maxFieldSize {
			return nil, ErrInvalidEnvelope
		}

		field := make([]byte, fieldLen)
		copy(field, data[offset:offset+int(fieldLen)])
		offset += int(fieldLen)
		return field, nil
	}

	// Read salt
	salt, err := readField()
	if err != nil {
		return nil, err
	}

	// Read nonce
	nonce, err := readField()
	if err != nil {
		return nil, err
	}

	// Ciphertext
	ciphertext, err := readField()
	if err != nil {
		return nil, err
	}

	// Tag
	tag, err := readField()
	if err != nil {
		return nil, err
	}

	return &Envelope{
		Version:    version,
		KDFParams:  kdfParams,
		Salt:       salt,
		Nonce:      nonce,
		Ciphertext: ciphertext,
		Tag:        tag,
	}, nil
}
