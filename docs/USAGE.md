# Password Vault CLI - Usage Guide

Complete guide to using the Password Vault CLI for secure password management.

## 📋 Table of Contents

- [Getting Started](#getting-started)
- [Basic Commands](#basic-commands)
- [Advanced Features](#advanced-features)
- [Profile Management](#profile-management)
- [Security Features](#security-features)
- [Configuration](#configuration)
- [Tips and Best Practices](#tips-and-best-practices)
- [Troubleshooting](#troubleshooting)

## 🚀 Getting Started

### First Time Setup

1. **Initialize Your Vault**
   ```bash
   vault init
   ```
   - Creates a new encrypted vault
   - Prompts for master passphrase
   - Sets up default configuration

2. **Configure Security Parameters** (Optional)
   ```bash
   vault init --kdf-memory 256 --kdf-iterations 5 --kdf-parallelism 8
   ```
   - `--kdf-memory`: Memory usage in MB (default: 128)
   - `--kdf-iterations`: Time cost (default: 3)
   - `--kdf-parallelism`: CPU cores to use (default: 4)

3. **Unlock Your Vault**
   ```bash
   vault unlock
   ```
   - Prompts for master passphrase
   - Unlocks vault for current session

### Basic Workflow

```bash
# 1. Unlock vault
vault unlock --ttl 2h

# 2. Add your first password
vault add github --username myuser --url https://github.com

# 3. Retrieve password (copies to clipboard)
vault get github --copy

# 4. List all entries
vault list

# 5. Lock vault when done
vault lock
```

## 🔧 Basic Commands

### Vault Management

#### Initialize Vault
```bash
# Basic initialization
vault init

# With custom parameters
vault init --kdf-memory 256 --kdf-iterations 5 --force

# Specify vault location
vault --vault /path/to/vault.db init
```

#### Unlock/Lock Vault
```bash
# Unlock with default timeout (30 minutes)
vault unlock

# Unlock with custom timeout
vault unlock --ttl 2h
vault unlock --ttl 30m
vault unlock --ttl 3600  # seconds

# Lock vault manually
vault lock

# Check vault status
vault status

# Machine-readable status
vault status --json
```

**What `vault status` shows**
- **Vault path**: Location of the active `vault.db` file.
- **Cipher/KDF metadata**: `AES-256-GCM` plus Argon2id parameters (memory, iterations, parallelism, salt length).
- **Entry statistics**: Entry count and most recent `updated_at` timestamp (visible only when the session is unlocked).
- **Session state**: Reports `locked` or `unlocked`, including remaining TTL when unlocked. `Remaining TTL` is in seconds in JSON output to simplify scripting.

Use `vault status --json` when integrating with scripts or monitoring systems.

### Entry Management

#### Add Entries

**Interactive Mode** (Recommended)
```bash
vault add github
# Prompts for:
# - Username: myuser
# - Password: [hidden input]
# - URL: https://github.com
# - Notes: My GitHub account
# - Tags: development,git
```

**Command Line Mode**
```bash
# Basic entry
vault add github --username myuser --url https://github.com

# With password prompt
vault add github --username myuser --secret-prompt

# With password from file
vault add github --username myuser --secret-file ~/.secrets/github

# With all details
vault add github \
  --username myuser \
  --url https://github.com \
  --notes "My GitHub account" \
  --tags "development,git" \
  --secret-prompt
```

**TOTP Support**
```bash
# Add entry with TOTP seed
vault add github --username myuser --totp-seed "JBSWY3DPEHPK3PXP"

# Generate TOTP code
vault get github --totp
```

#### Retrieve Entries

**Basic Retrieval**
```bash
# Copy password to clipboard (default)
vault get github

# Display entry details (without password)
vault get github --show

# Copy specific field
vault get github --field username --copy
vault get github --field url --copy

# Show password in terminal (use with caution)
vault get github --field password --show
```

**Field Options**
- `password` (default)
- `username`
- `url`
- `notes`
- `totp` (if configured)

#### List Entries

**Basic Listing**
```bash
# List all entries
vault list

# List with details
vault list --show

# JSON output
vault list --json
```

**Filtering**
```bash
# Search by name
vault list --search github

# Filter by tags
vault list --tags development
vault list --tags "development,git"

# Combined filters
vault list --search git --tags development
```

**Output Formats**
```bash
# Table format (default)
vault list

# JSON format
vault list --json

# CSV format
vault list --csv
```

#### Update Entries

```bash
# Interactive update
vault update github

# Update specific fields
vault update github --username newuser
vault update github --url https://github.com/newuser
vault update github --notes "Updated account"

# Rotate password
vault update github --rotate-secret

# Update tags
vault update github --tags "development,git,updated"
```

#### Rotate Passwords for Existing Entries

Use `vault rotate` when you need to regenerate a stored password without re-entering metadata.

```bash
# Rotate using default settings (20-character password, no output)
vault rotate github

# Show the new password instead of copying it
vault rotate github --show

# Copy to clipboard and clear after 30 seconds
vault rotate github --copy --ttl 30

# Generate a longer secret
vault rotate github --length 32
```

Command details:

- **`--length`** sets the generated password length (default `20`).
- **`--show`** prints the new password to stdout (mutually exclusive with `--copy`).
- **`--copy`** copies the new password to the system clipboard; combine with `--ttl` to control the clear timeout.
- **`--ttl`** accepts seconds; use `-1` to fall back to the configured default (`config.Config.ClipboardTTL`, 45s by default).
- Errors clearly report when an entry is missing or the clipboard is unavailable.

#### Delete Entries

```bash
# Delete with confirmation
vault delete github

# Force delete (no confirmation)
vault delete github --yes

# Delete multiple entries
vault delete github gitlab bitbucket --yes
```

## 🎯 Advanced Features

### Password Generation

Use `vault passgen` to create strong passwords or passphrases without storing them in the vault. This command is powered by `internal/cli/passgen.go` and the entropy helpers in `internal/crypto/entropy.go`.

#### Quick Examples

```bash
# Generate a 20-character password (default alnumsym charset)
vault passgen

# Generate a 32-character alphanumeric password
vault passgen --length 32 --charset alnum

# Generate a 5-word diceware-style passphrase
vault passgen --words 5

# Copy a password to clipboard for 45 seconds
vault passgen --copy --ttl 45
```

#### Flags

- **`--length <n>`** (default: `20`)
  Length of the generated password. Ignored when `--words` is supplied.
- **`--words <n>`**
  Generate a Diceware-style passphrase with `<n>` hyphenated adjective–noun words. Cannot be combined with `--length` or `--charset`.
- **`--charset <alpha|alnum|alnumsym>`** (default: `alnumsym`)
  Character set used for password generation when `--length` is active.
- **`--copy`**
  Copy the generated output to the clipboard using `internal/clipboard/clipboard.go`. Falls back to printing when omitted.
- **`--ttl <seconds>`**
  How long the clipboard entry remains before clearing. Use `-1` (default) to fall back to `config.Config.ClipboardTTL`, otherwise specify seconds.

If clipboard integration is unavailable, the command exits with an error when `--copy` is used. Invalid flag combinations and non-positive lengths/word counts return non-zero exit codes.

### Profile Management

Profiles help organize passwords by environment or purpose.

#### Create and Manage Profiles

```bash
# List profiles
vault profiles list

# Create new profile
vault profiles create production "Production environment passwords"
vault profiles create personal "Personal accounts"
vault profiles create development "Development and testing"

# Set default profile
vault profiles set-default production

# Delete profile
vault profiles delete old-profile

# Rename profile
vault profiles rename dev development
```

#### Using Profiles

```bash
# Add entry to specific profile
vault --profile production add prod-db --username admin

# List entries in profile
vault --profile production list

# Switch default profile
vault profiles set-default production

# Copy entry between profiles
vault copy github --from personal --to development
```

### Import and Export

#### Export Vault

**Encrypted Export** (Recommended)
```bash
# Export entire vault
vault export --encrypted backup.vault

# Export specific profile
vault export --encrypted backup.vault --profile production

# Export with compression
vault export --encrypted backup.vault.gz --compress
```

**Plaintext Export** (Use with caution)
```bash
# Export to JSON (requires double confirmation)
vault export --plaintext backup.json --include-secrets

# Export without passwords
vault export --plaintext entries.json
```

#### Import Vault

```bash
# Import from encrypted backup
vault import backup.vault

# Handle conflicts
vault import backup.vault --conflict skip      # Skip existing
vault import backup.vault --conflict overwrite # Overwrite existing
vault import backup.vault --conflict duplicate # Create duplicates

# Import to specific profile
vault import backup.vault --profile imported
```

### Security Operations

#### Master Key Rotation

```bash
# Rotate master passphrase
vault rotate-master-key

# Verify rotation
vault doctor --check-encryption
```

#### Audit and Monitoring

```bash
# View audit log
vault audit-log

# Verify audit log integrity
vault audit-log --verify

# Export audit log
vault audit-log --export audit.log

# Show recent operations
vault audit-log --since "2023-01-01"
vault audit-log --last 50
```

#### Security Health Check

```bash
# Run all security checks
vault doctor

# Specific checks
vault doctor --check-permissions
vault doctor --check-encryption
vault doctor --check-kdf-params
vault doctor --check-integrity

# Fix issues automatically
vault doctor --fix
```

## ⚙️ Configuration

### Configuration File

**Location:**
- Linux/macOS: `~/.config/vault/config.yaml`
- Windows: `%APPDATA%\vault\config.yaml`

**Example Configuration:**
```yaml
# Vault settings
vault_path: ~/.local/share/vault/vault.db
default_profile: default

# Security settings
security:
  session_timeout: 1800        # 30 minutes
  clipboard_timeout: 30        # 30 seconds
  max_failed_attempts: 3
  require_confirmation: true
  auto_lock_on_idle: true

# KDF parameters
kdf:
  memory: 128                   # MB
  iterations: 3
  parallelism: 4

# Profiles
profiles:
  default:
    name: default
    vault_path: ~/.local/share/vault/vault.db
    auto_lock: 300             # 5 minutes
    clipboard_timeout: 30
  
  production:
    name: production
    vault_path: ~/.local/share/vault/prod.db
    auto_lock: 60              # 1 minute
    clipboard_timeout: 10      # 10 seconds

# UI settings
ui:
  show_passwords: false
  confirm_deletions: true
  use_colors: true
```

### Environment Variables

```bash
# Override default vault path
export VAULT_PATH=/path/to/vault.db

# Override config file location
export VAULT_CONFIG=/path/to/config.yaml

# Set default profile
export VAULT_PROFILE=production

# Disable clipboard integration
export VAULT_NO_CLIPBOARD=1

# Enable debug logging
export VAULT_DEBUG=1
```

### Command Line Configuration

```bash
# View current configuration
vault config show

# Get specific setting
vault config get session_timeout

# Set configuration value
vault config set session_timeout 3600
vault config set clipboard_timeout 60

# Reset to defaults
vault config reset

# Show configuration file path
vault config path
```

## 💡 Tips and Best Practices

### Security Best Practices

1. **Strong Master Passphrase**
   ```bash
   # Use a long, unique passphrase
   # Consider using a passphrase generator
   vault init  # Choose 6+ random words
   ```

2. **Regular Backups**
   ```bash
   # Create encrypted backups regularly
   vault export --encrypted "backup-$(date +%Y%m%d).vault"
   
   # Store backups securely (different location)
   cp backup-*.vault /secure/backup/location/
   ```

3. **Session Management**
   ```bash
   # Use appropriate session timeouts
   vault unlock --ttl 30m  # For quick tasks
   vault unlock --ttl 2h   # For longer work sessions
   
   # Always lock when done
   vault lock
   ```

4. **Profile Organization**
   ```bash
   # Separate sensitive environments
   vault profiles create production
   vault profiles create development
   vault profiles create personal
   ```

### Performance Tips

1. **Optimize KDF Parameters**
   ```bash
   # Balance security vs. performance
   vault init --kdf-memory 64 --kdf-iterations 2  # Faster
   vault init --kdf-memory 256 --kdf-iterations 5 # More secure
   ```

2. **Use Appropriate Timeouts**
   ```bash
   # Longer sessions for active work
   vault unlock --ttl 4h
   
   # Shorter timeouts for sensitive profiles
   vault --profile production unlock --ttl 15m
   ```

### Workflow Optimization

1. **Aliases and Scripts**
   ```bash
   # Add to ~/.bashrc or ~/.zshrc
   alias vu='vault unlock --ttl 2h'
   alias vl='vault lock'
   alias vg='vault get'
   alias va='vault add'
   alias vls='vault list'
   ```

2. **Integration with Other Tools**
   ```bash
   # Use with password managers
   vault get github --field password | pbcopy
   
   # Integration with scripts
   PASSWORD=$(vault get database --field password --show)
   mysql -u admin -p"$PASSWORD" mydb
   ```

## 🔍 Troubleshooting

### Common Issues

#### "Vault is locked" Error
```bash
# Solution: Unlock the vault
vault unlock

# Check vault status
vault status
```

#### "Wrong passphrase" Error
```bash
# Ensure you're using the correct passphrase
# Check if vault file exists and is readable
ls -la ~/.local/share/vault/

# Try with different vault path
vault --vault /path/to/vault.db unlock
```

#### "Permission denied" Error
```bash
# Check file permissions
ls -la ~/.local/share/vault/vault.db

# Fix permissions
chmod 600 ~/.local/share/vault/vault.db
```

#### Clipboard Not Working
```bash
# Check clipboard integration
vault get test --copy --debug

# Disable clipboard if needed
vault get test --show

# Set environment variable
export VAULT_NO_CLIPBOARD=1
```

### Performance Issues

#### Slow Key Derivation
```bash
# Check current KDF parameters
vault doctor --check-kdf-params

# Reduce parameters for faster unlocking
vault rotate-master-key --kdf-memory 64 --kdf-iterations 2
```

#### Large Vault Performance
```bash
# Use filtering for large vaults
vault list --search term
vault list --tags specific-tag

# Consider splitting into profiles
vault profiles create archive
```

### Recovery Procedures

#### Forgot Master Passphrase
```
Unfortunately, there is no way to recover a forgotten master passphrase.
This is by design for security reasons.

Recovery options:
1. Restore from encrypted backup (if you remember the backup passphrase)
2. Start fresh with a new vault (all data will be lost)
```

#### Corrupted Vault File
```bash
# Check vault integrity
vault doctor --check-integrity

# Restore from backup
vault import backup.vault --conflict overwrite

# If no backup exists, the vault cannot be recovered
```

#### Lost Configuration
```bash
# Reset to default configuration
vault config reset

# Recreate configuration
vault config set session_timeout 1800
vault config set clipboard_timeout 30
```

### Debug Mode

```bash
# Enable debug logging
export VAULT_DEBUG=1
vault unlock

# Check log files
tail -f ~/.local/share/vault/debug.log

# Verbose output
vault --verbose get github
```

### Getting Help

```bash
# Built-in help
vault --help
vault add --help

# Version information
vault --version

# Configuration information
vault doctor --verbose
```

---

For more advanced usage and development information, see:
- [Architecture Documentation](ARCHITECTURE.md)
- [Security Model](../SECURITY.md)
- [Contributing Guide](../CONTRIBUTING.md)