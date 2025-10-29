# Password Vault CLI - Examples

This directory contains demonstration programs showcasing different features of the Password Vault CLI.

## Available Examples

### 1. Cryptography Demo (`crypto/`)
Demonstrates the core cryptographic operations:
- Argon2id key derivation
- AES-256-GCM encryption/decryption
- Secure memory handling
- Tamper detection
- Serialization/deserialization
- KDF parameter tuning

**Run:**
```bash
make demo-crypto
# or
go run ./examples/crypto/
```

### 2. Storage Layer Demo (`storage/`)
Demonstrates the storage layer functionality:
- Vault creation and initialization
- Profile management
- Entry CRUD operations
- Secure entry storage with encryption
- Entry filtering and search
- Integrity verification
- Vault locking

**Run:**
```bash
make demo-storage
# or
go run ./examples/storage/
```

### 3. CLI Demo (`cli/`)
Demonstrates CLI integration:
- Building the vault binary
- Running vault commands
- Command-line interface testing
- Automated vault operations

**Run:**
```bash
make demo-cli
# or
go run ./examples/cli/
```

## Running All Demos

To run all demos in sequence:
```bash
make demo-all
```

## Demo Features

Each demo:
- ✅ Creates temporary directories (auto-cleanup)
- ✅ Uses lightweight parameters for fast execution
- ✅ Demonstrates best practices
- ✅ Includes detailed console output
- ✅ Handles errors gracefully

## Educational Purpose

These examples are designed to help developers:
- Understand the vault's architecture
- Learn how to use the internal APIs
- See security best practices in action
- Test features interactively
- Contribute to the project

## Note

The demos use reduced security parameters (lower memory, iterations) for fast execution. **Production usage should use the default secure parameters.**

---

For more information, see:
- [Main README](../README.md)
- [Architecture Documentation](../docs/ARCHITECTURE.md)
- [Security Model](../SECURITY.md)
- [Contributing Guide](../CONTRIBUTING.md)
