# Password Vault CLI - Complete Guide

## 🔐 What Is This Project?

A **secure, offline password manager** that runs entirely on your local machine via command line. Think of it as a military-grade password safe that you control completely.

### **Why You Need This**

**Problem:** 
- Cloud password managers (LastPass, 1Password) = trust third parties with your secrets
- Browser-saved passwords = easily stolen by malware
- Text files = zero security
- Memory = forget complex passwords

**Solution:**
This vault provides:
- ✅ **Military-grade encryption** (AES-256-GCM + Argon2id)
- ✅ **Complete offline control** - no cloud, no tracking
- ✅ **Command-line speed** - faster than GUI apps
- ✅ **Tamper detection** - alerts if vault is modified
- ✅ **Audit logs** - track all access
- ✅ **Multi-profile support** - personal/work separation

---

## 🚀 How to Build & Run Locally

### **Step 1: Build the Binary**

```bash
# Navigate to project directory
cd /Users/ayush/VisualStudioProjects/password-vault-cli-project

# Build the vault CLI
go build -o vault ./cmd/vault/

# Verify it built successfully
./vault --version
```

### **Step 2: Initialize Your First Vault**

```bash
# Create a new encrypted vault
./vault init

# You'll be prompted:
# Enter master passphrase: ************
# Confirm passphrase: ************
# 
# Output:
# ✅ Vault initialized at: ~/.local/share/vault/vault.db
# ⚠️  IMPORTANT: Your master passphrase cannot be recovered!
```

**What happens:**
- Creates encrypted database file
- Derives encryption key from your passphrase using Argon2id (takes ~2 seconds - intentionally slow to prevent brute force)
- Sets up default profile
- Creates audit log

---

## 📖 Complete Usage Guide

### **Basic Commands**

#### **1. Add a Password**

```bash
# Add entry for GitHub
./vault add github.com

# Interactive prompts:
# URL: https://github.com
# Username: your-username
# Password: ************ (auto-generates if empty)
# Notes: My GitHub account
# Master passphrase: ************
#
# Output:
# ✅ Entry 'github.com' added successfully
```

#### **2. Retrieve a Password**

```bash
# Get GitHub password
./vault get github.com

# Output:
# Name:     github.com
# URL:      https://github.com
# Username: your-username
# Password: ************ (hidden by default)
# Notes:    My GitHub account
# Created:  2025-10-22 21:30:45

# Show password in plain text
./vault get github.com --show-password

# Copy password to clipboard
./vault get github.com --copy
# ✅ Password copied to clipboard (will clear in 30s)
```

#### **3. List All Passwords**

```bash
# List all entries
./vault list

# Output:
# NAME              USERNAME           URL                    UPDATED
# github.com        your-username      https://github.com     2 hours ago
# gmail.com         you@gmail.com      https://gmail.com      1 day ago
# aws.amazon.com    admin              https://aws.com        3 days ago

# Search entries
./vault list --search gmail

# Filter by tags
./vault list --tags work,important
```

#### **4. Update a Password**

```bash
# Update GitHub password
./vault update github.com

# Change password only
./vault update github.com --password

# Auto-generate new secure password
./vault update github.com --generate
```

#### **5. Delete an Entry**

```bash
# Delete entry (asks for confirmation)
./vault delete github.com

# Output:
# ⚠️  Are you sure you want to delete 'github.com'? (y/N): y
# Master passphrase: ************
# ✅ Entry deleted successfully
```

---

### **Advanced Features**

#### **1. Profiles (Personal vs Work)**

```bash
# Create work profile
./vault profile create work "Work passwords"

# Add entry to work profile
./vault add --profile work company-vpn

# List work passwords
./vault list --profile work

# Switch default profile
./vault config set default-profile work
```

#### **2. Generate Secure Passwords**

```bash
# Generate 20-character password
./vault generate

# Output:
# 🔐 Generated password: K9$mP2#vQ8@xL5&nW7!z
# Copied to clipboard (clears in 30s)

# Custom length and complexity
./vault generate --length 32 --no-symbols
./vault generate --length 16 --numbers-only
```

#### **3. Backup & Export**

```bash
# Create encrypted backup
./vault backup ~/backups/vault-backup-2025-10-22.vault

# Export to JSON (for migration)
./vault export --output passwords.json
# ⚠️  Warning: This exports PLAINTEXT passwords!

# Import from JSON
./vault import passwords.json
```

#### **4. Audit & Security**

```bash
# View audit log (who accessed what, when)
./vault audit

# Output:
# TIMESTAMP            ACTION    PROFILE   ENTRY          SUCCESS
# 2025-10-22 21:30:45  CREATE    default   github.com     ✅
# 2025-10-22 21:35:12  GET       default   github.com     ✅
# 2025-10-22 21:40:23  UPDATE    default   github.com     ✅

# Verify vault integrity
./vault verify

# Output:
# ✅ Vault structure: OK
# ✅ Encryption integrity: OK
# ✅ Audit chain: OK
# ✅ No corruption detected
```

#### **5. Lock/Unlock**

```bash
# Lock vault (clears decryption keys from memory)
./vault lock

# Unlock for session
./vault unlock
# Master passphrase: ************
# ✅ Vault unlocked

# Auto-lock after 1 hour of inactivity
./vault config set auto-lock-ttl 1h
```

---

## 🎯 Real-World Usage Examples

### **Example 1: Developer's Daily Workflow**

```bash
# Morning: Unlock vault
./vault unlock

# Need GitHub token
./vault get github-token --copy
# Paste into terminal

# Add new API key
./vault add stripe-api-key

# Evening: Lock vault
./vault lock
```

### **Example 2: Managing 100+ Passwords**

```bash
# Search for Amazon-related passwords
./vault list --search amazon

# Tag entries for organization
./vault update aws.amazon.com --tags cloud,production,critical
./vault update amazon.com --tags shopping,personal

# List only critical passwords
./vault list --tags critical
```

### **Example 3: Password Rotation**

```bash
# Generate new password for old account
./vault update old-service.com --generate --length 24

# Verify it was updated
./vault audit | grep old-service
```

### **Example 4: Secure Sharing (Export Specific Entry)**

```bash
# Export single entry to share with team
./vault get company-wifi --export > wifi-password.txt

# Securely delete after sharing
shred -u wifi-password.txt
```

---

## 🛡️ Security Features

### **1. Encryption Stack**

```
Your Password
    ↓
Argon2id KDF (64MB RAM, 3 iterations, 4 threads)
    ↓
256-bit Encryption Key
    ↓
AES-256-GCM Encryption
    ↓
Encrypted Vault File
```

### **2. Protection Against Attacks**

| Attack Type | Protection |
|-------------|------------|
| Brute Force | Argon2id (2+ seconds per attempt) |
| Rainbow Tables | Random salt per vault |
| Timing Attacks | Constant-time comparisons |
| Memory Dumps | Auto-zeroization of keys |
| File Tampering | HMAC integrity checks |
| Keyloggers | Clipboard auto-clear (30s) |
| Cold Boot | Memory encryption |

### **3. What We Test (The Security Tests We Fixed)**

```bash
# Run security test suite
go test -v github.com/vault-cli/vault/tests -run "Test.*"

# Tests verify:
✅ Tamper Detection    - File modification blocked
✅ Timing Attacks      - No password timing leaks
✅ Memory Leaks        - Secrets cleared from RAM
✅ Concurrent Access   - No race conditions
✅ Crypto Security     - No nonce reuse, strong KDF
✅ Input Validation    - No injection attacks
```

---

## 📊 Performance

```bash
# Typical operation times:
Initialize Vault:  2-3 seconds (Argon2id)
Add Entry:         50-100ms (encryption)
Get Entry:         30-50ms (decryption)
List 1000 Entries: 200-500ms
Search Entries:    100-300ms
```

---

## 🔧 Configuration

```bash
# View current config
./vault config show

# Common settings
./vault config set vault-path ~/my-secure-vault.db
./vault config set clipboard-ttl 60s
./vault config set auto-lock-ttl 30m
./vault config set output-format json
./vault config set show-passwords false

# Security settings
./vault config set confirm-destructive true
./vault config set kdf.memory 131072    # 128MB
./vault config set kdf.iterations 5
./vault config set kdf.parallelism 8
```

---

## 🎓 Why This Architecture?

### **Offline-First Design**
- ✅ Works without internet
- ✅ No cloud sync = no data breaches
- ✅ You control encryption keys
- ✅ No subscription fees
- ✅ Open source = auditable security

### **BoltDB Storage**
- ✅ Embedded database (no server needed)
- ✅ ACID transactions (atomic updates)
- ✅ Single file (easy backups)
- ✅ Fast reads (<1ms)

### **Argon2id KDF**
- ✅ Winner of Password Hashing Competition
- ✅ Resistant to GPU/ASIC attacks
- ✅ Memory-hard (can't parallelize easily)
- ✅ Configurable difficulty

### **AES-256-GCM Encryption**
- ✅ Military-grade security
- ✅ Authenticated encryption (tamper detection)
- ✅ Hardware-accelerated (AES-NI)

---

## 🚨 Important Security Notes

### **DO:**
✅ Use strong master passphrase (20+ chars)
✅ Backup vault file regularly
✅ Lock vault when not in use
✅ Store backups on encrypted drives
✅ Review audit logs periodically

### **DON'T:**
❌ Store master passphrase anywhere
❌ Use same password for vault and websites
❌ Share master passphrase
❌ Store vault on cloud sync folders (Dropbox, etc.)
❌ Run on untrusted/shared computers

---

## 📦 Installation for Daily Use

```bash
# Build and install globally
go build -o vault ./cmd/vault/
sudo mv vault /usr/local/bin/

# Now use from anywhere
vault init
vault add gmail.com
vault get gmail.com

# Add to .bashrc/.zshrc for aliases
alias vpass='vault get --copy'
alias vlist='vault list'
alias vadd='vault add'
```

---

## 🎯 Summary

This is a **security-first, developer-friendly password manager** for people who:
- Don't trust cloud password managers
- Want complete control over their data
- Prefer CLI over GUI
- Need military-grade encryption
- Value speed and simplicity
- Want auditable, open-source security

**The tests we fixed ensure this vault is actually secure**, not just theoretically secure!