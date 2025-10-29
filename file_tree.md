# Password Vault CLI - File Structure

## Project Directory Structure

```
password-vault-cli/
├── cmd/
│   └── vault/
│       ├── main.go                    # Application entry point and version info
│       └── root.go                    # Root command setup and global flags
├── internal/
│   ├── cli/                          # CLI command implementations
│   │   ├── add.go                    # Add entry command
│   │   ├── audit.go                  # Audit log commands  
│   │   ├── config.go                 # Configuration commands
│   │   ├── delete.go                 # Delete entry command
│   │   ├── doctor.go                 # Security health check
│   │   ├── export.go                 # Export vault command
│   │   ├── get.go                    # Get entry command
│   │   ├── import.go                 # Import vault command
│   │   ├── init.go                   # Initialize vault command
│   │   ├── list.go                   # List entries command
│   │   ├── lock.go                   # Lock vault command
│   │   ├── profiles.go               # Profile management commands
│   │   ├── rotate.go                 # Master key rotation command
│   │   ├── unlock.go                 # Unlock vault command
│   │   ├── update.go                 # Update entry command
│   │   └── utils.go                  # CLI utility functions
│   ├── vault/                        # Core vault operations
│   │   ├── manager.go                # VaultManager implementation
│   │   ├── session.go                # Session management
│   │   ├── crypto.go                 # CryptoProvider implementation  
│   │   ├── derive.go                 # Key derivation functions
│   │   ├── envelope.go               # Encryption envelope format
│   │   └── keyring.go                # Master key management
│   ├── store/                        # Storage layer
│   │   ├── store.go                  # VaultStore interface
│   │   ├── bbolt.go                  # BoltDB implementation
│   │   ├── lock.go                   # File locking utilities
│   │   ├── txn.go                    # Transaction management
│   │   └── migration.go              # Database schema migrations
│   ├── domain/                       # Domain models and validation
│   │   ├── models.go                 # Entry, Profile, Operation structs
│   │   ├── validate.go               # Input validation functions
│   │   ├── filter.go                 # Entry filtering logic
│   │   └── errors.go                 # Domain-specific errors
│   ├── audit/                        # Audit logging system
│   │   ├── logger.go                 # AuditLogger interface
│   │   ├── hmac_chain.go             # HMAC chain implementation
│   │   ├── operations.go             # Operation type definitions
│   │   └── verify.go                 # Integrity verification
│   ├── clipboard/                    # Clipboard integration
│   │   ├── clipboard.go              # ClipboardManager interface
│   │   ├── manager.go                # Cross-platform implementation
│   │   ├── platform_unix.go          # Unix-specific clipboard code
│   │   ├── platform_windows.go       # Windows-specific clipboard code
│   │   └── platform_darwin.go        # macOS-specific clipboard code
│   ├── config/                       # Configuration management
│   │   ├── config.go                 # Configuration struct and loading
│   │   ├── defaults.go               # Default configuration values
│   │   ├── paths.go                  # Platform-specific paths
│   │   └── validation.go             # Configuration validation
│   └── util/                         # Shared utilities
│       ├── errors.go                 # Error handling utilities
│       ├── zeroize.go                # Memory zeroization functions
│       ├── random.go                 # Cryptographic random generation
│       ├── files.go                  # File operation utilities
│       └── platform.go               # Platform detection and utils
├── pkg/                              # Public API (if needed for extensions)
│   └── vault/
│       └── client.go                 # Public vault client interface
├── docs/                             # Documentation
│   ├── ARCHITECTURE.md               # System architecture document
│   ├── SECURITY.md                   # Security model and threat analysis
│   ├── API.md                        # CLI command reference
│   ├── CONFIGURATION.md              # Configuration guide
│   └── DEVELOPMENT.md                # Development setup and guidelines
├── tests/                            # Test files
│   ├── integration/                  # Integration test suites
│   │   ├── vault_test.go             # End-to-end vault operations
│   │   ├── cli_test.go               # CLI command testing
│   │   ├── crypto_test.go            # Cryptographic operation tests
│   │   └── scenarios/                # Test scenario definitions
│   ├── fixtures/                     # Test data and fixtures
│   │   ├── test_vectors.json         # Cryptographic test vectors
│   │   ├── sample_vault.db           # Sample vault for testing
│   │   └── config_samples/           # Sample configuration files
│   └── mocks/                        # Generated mock interfaces
│       ├── mock_store.go             # Mock VaultStore
│       ├── mock_crypto.go            # Mock CryptoProvider
│       └── mock_clipboard.go         # Mock ClipboardManager
├── scripts/                          # Build and development scripts
│   ├── build.sh                      # Cross-platform build script
│   ├── test.sh                       # Test execution script
│   ├── lint.sh                       # Code linting script
│   ├── generate-mocks.sh             # Mock generation script
│   └── release.sh                    # Release packaging script
├── deployments/                      # Deployment configurations
│   ├── docker/                       # Docker build files (for CI)
│   │   ├── Dockerfile                # Multi-stage build container
│   │   └── docker-compose.yml        # Development environment
│   └── ci/                           # CI/CD configurations
│       ├── github-actions.yml        # GitHub Actions workflow
│       ├── test-matrix.yml           # Cross-platform test matrix
│       └── security-scan.yml         # Security scanning configuration
├── examples/                         # Usage examples and demos
│   ├── quickstart.md                 # Getting started guide
│   ├── advanced-usage.md             # Advanced features guide
│   ├── scripts/                      # Example automation scripts
│   │   ├── backup-vault.sh           # Automated backup script
│   │   ├── bulk-import.sh            # Bulk entry import script
│   │   └── security-audit.sh         # Security audit script
│   └── configs/                      # Example configuration files
│       ├── minimal.yaml              # Minimal configuration
│       ├── secure.yaml               # High-security configuration
│       └── development.yaml          # Development configuration
├── .github/                          # GitHub-specific files
│   ├── workflows/                    # GitHub Actions workflows
│   │   ├── ci.yml                    # Continuous integration
│   │   ├── security.yml              # Security scanning
│   │   └── release.yml               # Release automation
│   ├── ISSUE_TEMPLATE/               # Issue templates
│   │   ├── bug_report.md             # Bug report template
│   │   ├── feature_request.md        # Feature request template
│   │   └── security_issue.md         # Security issue template
│   └── PULL_REQUEST_TEMPLATE.md      # Pull request template
├── go.mod                            # Go module definition
├── go.sum                            # Go module checksums
├── Makefile                          # Build automation
├── README.md                         # Project overview and quick start
├── LICENSE                           # MIT license
├── CHANGELOG.md                      # Version history and changes
├── CONTRIBUTING.md                   # Contribution guidelines
├── SECURITY.md                       # Security policy and reporting
└── .gitignore                        # Git ignore patterns
```

## Key File Descriptions

### Core Application Files

**cmd/vault/main.go**
- Application entry point with version embedding
- Global flag parsing and configuration loading
- Graceful shutdown handling and cleanup

**internal/vault/manager.go**
- Central VaultManager implementing core business logic
- Session management and security policy enforcement
- Coordination between storage, crypto, and audit layers

**internal/store/bbolt.go**
- BoltDB storage implementation with encryption
- Atomic transaction management and rollback
- Profile-based bucket organization and indexing

**internal/vault/crypto.go**
- Argon2id key derivation and AEAD encryption
- Secure random generation and nonce management
- Memory zeroization and cryptographic utilities

### CLI Command Structure

Each command file in `internal/cli/` follows a consistent pattern:
- Command definition with Cobra framework
- Input validation and secure prompting
- Business logic delegation to VaultManager
- Output formatting and error handling
- Integration with clipboard and audit systems

### Security-Critical Files

**internal/util/zeroize.go**
- Memory clearing functions for sensitive data
- Compiler optimization prevention
- Platform-specific secure memory handling

**internal/audit/hmac_chain.go**
- Tamper-evident audit log implementation
- HMAC chain computation and verification
- Operation serialization and integrity checking

**internal/clipboard/platform_*.go**
- Platform-specific clipboard implementations
- Auto-clear timer management
- Security warning and capability detection

### Configuration and Documentation

**docs/SECURITY.md**
- Comprehensive threat model analysis
- Cryptographic parameter recommendations
- Security best practices and limitations

**examples/configs/**
- Sample configurations for different use cases
- Security parameter tuning guidelines
- Platform-specific optimization examples

### Testing Infrastructure

**tests/integration/**
- End-to-end workflow testing
- Cross-platform compatibility verification
- Security property validation (fuzzing, corruption tests)

**tests/fixtures/test_vectors.json**
- Cryptographic test vectors for validation
- Known-good encryption/decryption pairs
- KDF parameter verification data

## Build and Release Structure

### Cross-Platform Builds
The build system produces static binaries for:
- Linux: amd64, arm64 (glibc and musl variants)
- macOS: amd64, arm64 (universal binary)
- Windows: amd64, arm64 (.exe with embedded manifest)

### Release Artifacts
- Compressed binaries with version embedding
- Checksums and digital signatures
- Installation scripts for package managers
- Documentation bundles (man pages, completion scripts)

### Development Workflow
- Pre-commit hooks for formatting and linting
- Automated testing on multiple platforms
- Security scanning and dependency auditing
- Mock generation for unit testing isolation