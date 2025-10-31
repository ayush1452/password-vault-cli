# Vault Cryptography Module

This module implements the core cryptographic operations for the Password Vault CLI, providing secure encryption, key derivation, and data integrity protection.

## Features

- **Argon2id Key Derivation**: Memory-hard function resistant to GPU/ASIC attacks
- **AES-256-GCM Encryption**: Authenticated encryption with integrity protection
- **Secure Memory Management**: Automatic zeroization of sensitive data
- **Tamper Detection**: Cryptographic integrity verification
- **Configurable Parameters**: Tunable security/performance trade-offs

## Quick Start

```go
package main

import (
    "github.com/vault-cli/vault/internal/vault"
)

func main() {
    // Create crypto engine
    engine := vault.NewDefaultCryptoEngine()
    
    // Encrypt data with passphrase
    data := []byte("secret password")
    passphrase := "my-master-passphrase"
    
    envelope, err := engine.SealWithPassphrase(data, passphrase)
    if err != nil {
        panic(err)
    }
    
    // Decrypt data
    decrypted, err := engine.OpenWithPassphrase(envelope, passphrase)
    if err != nil {
        panic(err)
    }
    
    // decrypted now contains "secret password"
}
```

## API Reference

### Core Types

#### `CryptoEngine`
The main interface for cryptographic operations.

```go
engine := vault.NewDefaultCryptoEngine()
engine := vault.NewCryptoEngine(customParams)
```

#### `Envelope`
Encrypted data container with all necessary metadata.

```go
type Envelope struct {
    Version    uint8         // Format version
    KDFParams  Argon2Params  // Key derivation parameters
    Salt       []byte        // Random salt for KDF
    Nonce      []byte        // Random nonce for encryption
    Ciphertext []byte        // Encrypted data
    Tag        []byte        // Authentication tag
}
```

#### `Argon2Params`
Key derivation function parameters.

```go
type Argon2Params struct {
    Memory      uint32 // Memory usage in KB
    Iterations  uint32 // Number of iterations
    Parallelism uint8  // Degree of parallelism
}
```

### Key Functions

#### Key Derivation
```go
// Derive key from passphrase and salt
key, err := engine.DeriveKey(passphrase, salt)
defer vault.Zeroize(key) // Always clean up
```

#### Encryption
```go
// Encrypt with existing key
envelope, err := engine.Seal(plaintext, key)

// Encrypt with passphrase (generates new salt)
envelope, err := engine.SealWithPassphrase(plaintext, passphrase)
```

#### Decryption
```go
// Decrypt with existing key
plaintext, err := engine.Open(envelope, key)

// Decrypt with passphrase
plaintext, err := engine.OpenWithPassphrase(envelope, passphrase)
```

#### Utilities
```go
// Generate cryptographically secure random values
salt, err := vault.GenerateSalt()
nonce, err := vault.GenerateNonce()

// Secure memory operations
vault.Zeroize(sensitiveData)
vault.ZeroizeString(&sensitiveString)
isEqual := vault.SecureCompare(data1, data2)

// Parameter validation and tuning
err := vault.ValidateArgon2Params(params)
duration := vault.BenchmarkKDF(params, passphrase, salt)
tunedParams, err := vault.TuneArgon2Params(targetDuration, testPassphrase)
```

#### Serialization
```go
// Convert envelope to bytes for storage
data := vault.EnvelopeToBytes(envelope)

// Convert bytes back to envelope
envelope, err := vault.EnvelopeFromBytes(data)
```

## Security Considerations

### Key Derivation (Argon2id)
- **Default parameters**: 64 MB memory, 3 iterations, 4 parallelism (~300ms)
- **Tuning**: Use `TuneArgon2Params()` to optimize for your hardware
- **Security**: Increase memory first, then iterations for better protection

### Encryption (AES-256-GCM)
- **Algorithm**: NIST-approved authenticated encryption
- **Key size**: 256 bits (32 bytes)
- **Nonce**: 96 bits, randomly generated per operation
- **Tag**: 128 bits for authentication

### Memory Safety
- **Zeroization**: Always call `Zeroize()` on sensitive data
- **Limitations**: Go GC may copy data before clearing
- **Best practice**: Minimize lifetime of decrypted data

### Nonce Management
- **Uniqueness**: Each encryption uses a fresh random nonce
- **Size**: 96 bits provides excellent collision resistance
- **Generation**: Uses `crypto/rand` for cryptographic security

## Error Handling

The module defines specific error types for different failure modes:

```go
var (
    ErrInvalidEnvelope    = errors.New("invalid envelope format")
    ErrInvalidVersion     = errors.New("unsupported envelope version")
    ErrDecryptionFailed   = errors.New("decryption failed")
    ErrInvalidKeySize     = errors.New("invalid key size")
    ErrInvalidNonceSize   = errors.New("invalid nonce size")
    ErrInvalidCiphertext  = errors.New("invalid ciphertext")
)
```

### Common Error Scenarios

1. **Wrong passphrase**: Returns `ErrDecryptionFailed`
2. **Tampered data**: Returns `ErrDecryptionFailed`
3. **Invalid parameters**: Returns validation error
4. **Corrupted envelope**: Returns `ErrInvalidEnvelope`

## Performance

### Benchmarks (on modern hardware)
- **Key derivation**: ~300ms (default parameters)
- **Encryption**: ~1μs per KB
- **Decryption**: ~1μs per KB

### Tuning Guidelines
- **Interactive use**: 200-500ms key derivation
- **Server use**: 1-2s key derivation acceptable
- **Memory**: More memory = better security vs. GPU attacks
- **Iterations**: More iterations = better security vs. CPU attacks

## Testing

Run the test suite:
```bash
go test -v ./internal/vault/
```

Run benchmarks:
```bash
go test -bench=. ./internal/vault/
```

The test suite includes:
- Unit tests for all functions
- Known answer tests for reproducibility
- Tamper detection tests
- Serialization round-trip tests
- Performance benchmarks

## Examples

See the examples directory for complete demonstrations:
- `examples/crypto/` - Cryptography features demo
- `examples/storage/` - Storage layer demo
- `examples/cli/` - CLI integration demo

Run demos with:
```bash
make demo-crypto    # Cryptography demo
make demo-storage   # Storage layer demo
make demo-all       # Run all demos
```

## Dependencies

- `golang.org/x/crypto/argon2`: Argon2id key derivation
- `crypto/aes`: AES block cipher
- `crypto/cipher`: GCM authenticated encryption
- `crypto/rand`: Cryptographically secure random number generation
- `crypto/subtle`: Constant-time operations

## Compliance

This implementation follows:
- **NIST SP 800-132**: Password-based key derivation
- **RFC 9106**: Argon2 specification  
- **NIST SP 800-38D**: GCM mode specification
- **OWASP**: Password storage guidelines