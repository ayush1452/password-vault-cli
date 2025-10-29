# Password Vault CLI - System Design

## Implementation Approach

The Password Vault CLI will be implemented as a secure, local-only password management system using Go with the following key components:

1. **Cryptographic Foundation** - Argon2id key derivation with AES-256-GCM authenticated encryption
2. **Storage Engine** - BoltDB embedded database with record-level encryption and atomic operations  
3. **CLI Interface** - Cobra framework providing intuitive command structure with secure input handling
4. **Profile System** - Namespace-based organization for different environments (dev/prod/personal)
5. **Audit Trail** - HMAC-chained operation logging for tamper detection and compliance
6. **Cross-Platform Support** - Native binaries for macOS, Linux, and Windows (amd64/arm64)

### Critical Requirements Addressed

- **Zero Trust Security**: All data encrypted at rest, filesystem treated as hostile
- **Memory Safety**: Immediate zeroization of sensitive data, no plaintext secrets in memory
- **Atomic Operations**: Crash-safe writes using temporary files and atomic rename
- **Concurrency Control**: File-based locking preventing concurrent access corruption
- **Tamper Detection**: HMAC chain verification and AEAD tag validation

### Technology Stack

- **Language**: Go 1.21+ for memory safety and cross-platform support
- **CLI Framework**: Cobra for command structure and flag parsing
- **Database**: BoltDB for embedded, ACID-compliant key-value storage
- **Cryptography**: Standard library crypto/aes + golang.org/x/crypto/argon2
- **Platform Integration**: Cross-platform clipboard and file system APIs

## Main User-UI Interaction Patterns

### 1. Initial Setup Workflow
```bash
# Create new vault with custom KDF parameters
vault init --kdf-memory 65536 --kdf-iterations 3
# Prompts for master passphrase with confirmation
# Creates vault.db with 0600 permissions
```

### 2. Daily Usage Pattern  
```bash
# Unlock vault for session
vault unlock --ttl 3600

# Add new credentials
vault add github --username user@example.com --secret-prompt --url https://github.com --tags work,git

# Retrieve and copy password
vault get github --copy
# Password copied to clipboard, auto-clears in 30s

# Search and list entries
vault list --tag work --search git

# Lock vault when done
vault lock
```

### 3. Profile Management
```bash
# Create environment-specific profiles
vault profiles create production
vault profiles create development

# Work with specific profiles
vault --profile production add prod-db --username admin --secret-prompt
vault --profile development add dev-api --username test --secret-file ./api-key.txt
```

### 4. Backup and Security Operations
```bash
# Export encrypted backup
vault export --encrypted backup-2024.vault

# Rotate master key periodically  
vault rotate-master-key
# Prompts for current and new passphrase, re-encrypts all data

# Verify audit trail integrity
vault audit-log --verify

# Security health check
vault doctor
```

## Data Structures and Interfaces Overview

The system uses a layered architecture with clear interface boundaries:

### Core Interfaces
- **VaultStore**: Database operations with profile namespacing
- **CryptoProvider**: Key derivation and authenticated encryption  
- **AuditLogger**: Tamper-evident operation logging
- **ClipboardManager**: Secure clipboard integration with auto-clear

### Key Data Types
- **Entry**: Password record with metadata (username, URL, notes, tags, TOTP)
- **Envelope**: Encrypted storage format {version, nonce, ciphertext, tag}
- **Operation**: Audit log entry with HMAC chain linkage
- **VaultMetadata**: Vault configuration and KDF parameters

### Security Boundaries
- **Application Layer**: Business logic, session management, input validation
- **Security Layer**: Cryptographic operations, key management, memory zeroization
- **Storage Layer**: Encrypted persistence, atomic transactions, file locking
- **Platform Layer**: OS integration, clipboard, file permissions

## Program Call Flow Overview

### Entry Addition Flow
1. CLI validates input and prompts for secret
2. VaultManager checks session validity and generates entry ID
3. CryptoProvider generates unique nonce and encrypts entry JSON
4. BoltStore saves encrypted envelope to profile bucket
5. AuditLogger records operation with HMAC chain update
6. Memory zeroization clears sensitive data

### Key Rotation Flow  
1. Verify current master key with existing vault metadata
2. Derive new master key from new passphrase with fresh salt
3. Begin atomic transaction for re-encryption process
4. Iterate through all profiles and entries, re-encrypting with new key
5. Update vault metadata with new salt and KDF parameters
6. Commit transaction and update session with new key
7. Zeroize old and new master keys from memory

### Audit Verification Flow
1. Load complete operation history from audit bucket
2. Retrieve chain key (encrypted with master key)
3. Recompute HMAC chain from genesis operation
4. Compare computed HMACs with stored values
5. Report any breaks in chain integrity

## Database ER Diagram Overview

The BoltDB storage uses a bucket-based hierarchy:

```
vault.db
├── metadata/           # Vault configuration and KDF parameters
├── profiles/          # Profile definitions and metadata  
├── entries:default/   # Default profile entry storage
├── entries:prod/      # Production profile entry storage
├── audit/            # HMAC-chained operation log
└── config/           # Runtime configuration settings
```

### Key Relationships
- **Profiles** contain multiple **Entries** (1:N)
- **Operations** reference **Profiles** and **Entries** (N:1, N:1)  
- **Envelopes** store encrypted **Entry** data (1:1)
- **Metadata** defines **KDF Parameters** for key derivation (1:1)

### Data Integrity
- Each entry encrypted with unique nonce and authenticated with AEAD
- Audit operations linked via HMAC chain preventing tampering
- Vault metadata includes integrity checksums for corruption detection

## Unclear Aspects and Assumptions

### Assumptions Made
1. **Performance Target**: Key derivation ~300ms on modern hardware (tunable)
2. **Memory Requirements**: 64MB+ available for default Argon2id parameters  
3. **File System**: POSIX-compliant with atomic rename support
4. **Clipboard Security**: Best-effort clearing, OS may retain copies in swap/memory dumps
5. **Concurrent Access**: Single-user model, file locking prevents corruption

### Areas Requiring Clarification
1. **TOTP Implementation**: Generate codes in CLI or just store seeds for external apps?
2. **Import Conflict Resolution**: Detailed merge strategies for duplicate entries during import
3. **Export Key Derivation**: Use same master key or prompt for separate export passphrase?
4. **Profile Audit Isolation**: Should each profile have independent audit chains?
5. **Recovery Procedures**: Behavior when audit chain corrupted but entries remain valid?

### Security Considerations Needing Review
1. **Memory Protection**: Additional measures against memory dumps (mlock, guard pages)?
2. **Timing Attacks**: Constant-time operations for passphrase verification?
3. **Hardware Integration**: Future TPM/Secure Enclave support for key storage?
4. **Backup Security**: Encrypted export format and key management strategy?
5. **Emergency Access**: Master key recovery options without compromising security?

### Platform-Specific Concerns
1. **Windows**: File locking behavior and permission model differences
2. **macOS**: Keychain integration and clipboard security policies  
3. **Linux**: Distribution-specific clipboard managers and security contexts
4. **Mobile**: Future iOS/Android support considerations for data format compatibility