# Password Vault CLI

A secure, offline, open-source CLI password vault with local-only encryption, designed for developers and security-conscious users.

## 🔐 Features

- **Local-Only**: No network connectivity, sync, or telemetry
- **Strong Encryption**: Argon2id KDF + AES-256-GCM AEAD
- **Profile Support**: Organize passwords by environment (dev/prod/personal)
- **Cross-Platform**: Works on macOS, Linux, and Windows
- **Secure by Design**: Zero trust architecture, tamper detection
- **CLI Interface**: Fast, scriptable command-line interface
- **Clipboard Integration**: Auto-clearing clipboard support
- **Audit Logging**: Tamper-evident operation history
- **Export/Import**: Encrypted backup and restore

## 🚀 Quick Start

### Installation

#### From Source (Recommended)

```bash
# Clone the repository
git clone https://github.com/vault-cli/vault.git
cd vault

# Build the binary
make build

# Install to system PATH
sudo make install
```

#### Prerequisites

- Go 1.21 or later
- Git

### Basic Usage

```bash
# Initialize a new vault
vault init

# Unlock the vault
vault unlock

# Add your first password
vault add github --username myuser --url https://github.com

# Retrieve a password (copies to clipboard)
vault get github --copy

# List all entries
vault list

# Lock the vault when done
vault lock
```

## 📖 Documentation

- [Usage Guide](docs/USAGE.md) - Detailed command examples
- [Architecture](docs/ARCHITECTURE.md) - System design and components
- [Security Model](SECURITY.md) - Threat model and cryptographic choices
- [Contributing](CONTRIBUTING.md) - Development setup and guidelines

## 🛡️ Security

This password vault is designed with security as the primary concern:

- **Argon2id KDF**: Memory-hard key derivation (~300ms default)
- **AES-256-GCM**: NIST-approved authenticated encryption
- **Unique Nonces**: CSPRNG-generated, never reused
- **File Permissions**: Vault files locked to 0600 (owner-only)
- **Memory Protection**: Sensitive data zeroized after use
- **Tamper Detection**: Cryptographic integrity verification

### Security Validation

Run the security test suite:

```bash
make test-security
make test-fuzz
make test-acceptance
```

See [SECURITY_VALIDATION.md](docs/SECURITY_VALIDATION.md) for detailed test results.

## 🔧 Commands

| Command | Description |
|---------|-------------|
| `init` | Initialize a new vault |
| `unlock` | Unlock vault for access |
| `lock` | Lock vault and clear session |
| `add <entry>` | Add new password entry |
| `get <entry>` | Retrieve password entry |
| `list` | List all entries |
| `update <entry>` | Update existing entry |
| `delete <entry>` | Delete entry |
| `profiles` | Manage profiles |
| `export` | Export encrypted backup |
| `import` | Import from backup |
| `doctor` | Run security health checks |
| `config` | Manage configuration |

## 📁 Configuration

Default configuration locations:

- **Linux/macOS**: `~/.config/vault/config.yaml`
- **Windows**: `%APPDATA%\vault\config.yaml`

Example configuration:

```yaml
vault_path: ~/.local/share/vault/vault.db
default_profile: default
session_timeout: 1800  # 30 minutes
clipboard_timeout: 30  # 30 seconds
profiles:
  default:
    name: default
    auto_lock: 300
  production:
    name: production
    auto_lock: 60
```

## 🏗️ Development

### Build from Source

```bash
# Clone repository
git clone https://github.com/vault-cli/vault.git
cd vault

# Install dependencies
go mod download

# Run tests
make test

# Build binary
make build

# Run security tests
make test-security

# Run demo examples
make demo-crypto     # Cryptography features
make demo-storage    # Storage layer
make demo-all        # All demos
```

### Project Structure

```
.
├── cmd/vault/           # CLI entry point
├── internal/
│   ├── cli/            # Command implementations
│   ├── vault/          # Cryptographic engine
│   ├── store/          # Storage layer
│   ├── domain/         # Data models
│   ├── config/         # Configuration
│   └── util/           # Utilities
├── tests/              # Test suites
├── docs/               # Documentation
└── Makefile           # Build system
```

## 🧪 Testing

Comprehensive test suite with >90% coverage:

```bash
# Run all tests
make test

# Run specific test suites
make test-unit          # Unit tests
make test-integration   # Integration tests
make test-security      # Security scenarios
make test-fuzz          # Fuzz testing
make test-acceptance    # End-to-end workflows

# Generate coverage report
make coverage
```

## 📊 Performance

Typical performance characteristics:

- **Key Derivation**: ~300ms (tunable)
- **Encryption**: <1ms per entry
- **Database Operations**: <5ms per operation
- **Startup Time**: <100ms

## 🔒 Threat Model

This vault protects against:

- ✅ Unauthorized vault access
- ✅ Data exfiltration from disk
- ✅ Memory dumps and swap files
- ✅ Brute force attacks
- ✅ Dictionary attacks
- ✅ File tampering
- ✅ Timing attacks

**Out of Scope:**
- Physical access to unlocked system
- Keyloggers and screen capture
- OS-level privilege escalation
- Hardware attacks (cold boot, etc.)

## 📝 License

MIT License - see [LICENSE](LICENSE) for details.

## 🤝 Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for:

- Development setup
- Code style guidelines
- Testing requirements
- Security considerations
- Pull request process

## 🆘 Support

- **Issues**: [GitHub Issues](https://github.com/vault-cli/vault/issues)
- **Security**: Report security issues to security@vault-cli.dev
- **Documentation**: [docs/](docs/) directory

## 🎯 Roadmap

- [ ] TOTP/2FA support
- [ ] Hardware security key integration
- [ ] Mobile companion app
- [ ] Browser extension
- [ ] Team sharing (encrypted)
- [ ] Backup to cloud storage

## ⚡ Quick Examples

```bash
# Initialize and setup
vault init --kdf-memory 128 --kdf-iterations 5
vault unlock --ttl 2h

# Add entries with different methods
vault add github --username user@example.com --secret-prompt
vault add aws --username admin --secret-file ~/.aws/secret
vault add db --username root --url postgres://localhost:5432

# Organize with profiles
vault profiles create production "Production environment"
vault --profile production add prod-db --username admin

# Search and retrieve
vault list --tags work,development
vault get github --field password --copy
vault get aws --show  # Display without copying

# Maintenance
vault doctor  # Security health check
vault export --encrypted backup.vault
vault rotate-master-key
```

---

**⚠️ Security Notice**: Always keep your master passphrase secure and create regular encrypted backups. This tool stores passwords locally and cannot recover lost master passphrases.