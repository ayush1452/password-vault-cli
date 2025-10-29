# Security Analysis - Password Vault CLI

## Threat Model

### Assets Protected
- **Master passphrase**: The primary secret that protects all vault data
- **Stored passwords**: User credentials and sensitive information
- **Metadata**: Entry names, URLs, notes, and other associated data
- **Audit logs**: Historical record of vault operations

### Threat Actors
- **Malicious software**: Malware, keyloggers, memory scrapers on user's system
- **Physical attackers**: Individuals with physical access to storage media
- **Forensic analysts**: Law enforcement or corporate investigators
- **Insider threats**: Users with legitimate system access attempting unauthorized data access

### Attack Vectors
1. **Memory attacks**: RAM dumps, cold boot attacks, process memory inspection
2. **Storage attacks**: Direct file system access, backup recovery, deleted file recovery
3. **Side-channel attacks**: Timing analysis, power analysis, cache analysis
4. **Brute force attacks**: Dictionary attacks on master passphrase
5. **Implementation attacks**: Buffer overflows, format string bugs, logic errors
6. **Social engineering**: Tricking users into revealing passphrases or vault contents

## Cryptographic Design

### Key Derivation Function (KDF)
**Algorithm**: Argon2id (RFC 9106)
**Rationale**: 
- Memory-hard function resistant to ASIC/GPU attacks
- Combines data-dependent (Argon2i) and data-independent (Argon2d) access patterns
- Winner of Password Hashing Competition (PHC)
- Recommended by OWASP and security experts

**Parameters**:
- **Memory**: 64 MB (default) - Configurable from 1 MB to 1 GB
- **Iterations**: 3 (default) - Configurable from 1 to 100
- **Parallelism**: 4 (default) - Configurable from 1 to 255
- **Output length**: 32 bytes (256 bits)
- **Salt length**: 32 bytes (256 bits)

**Tuning Guidelines**:
- Target 200-500ms on user's hardware
- Increase memory first, then iterations for better security
- Consider user experience vs. security trade-offs
- Use `vault config tune-kdf` to automatically optimize parameters

### Authenticated Encryption
**Algorithm**: AES-256-GCM (Galois/Counter Mode)
**Rationale**:
- NIST-approved AEAD (Authenticated Encryption with Associated Data)
- Provides both confidentiality and integrity
- Constant-time implementation available in Go standard library
- Widely analyzed and trusted

**Parameters**:
- **Key size**: 256 bits (derived from Argon2id)
- **Nonce size**: 96 bits (12 bytes) - Randomly generated per record
- **Tag size**: 128 bits (16 bytes) - Authentication tag

**Nonce Strategy**:
- Cryptographically secure random generation (crypto/rand)
- Unique per encrypted record
- Never reused with the same key
- Collision probability negligible with 96-bit nonces

### Envelope Format
```
Version (1 byte) || KDF_Params (9 bytes) || Salt_Len (4 bytes) || Salt || 
Nonce_Len (4 bytes) || Nonce || Ciphertext_Len (4 bytes) || Ciphertext || 
Tag_Len (4 bytes) || Tag
```

**Versioning**: Enables cryptographic agility and future upgrades
**Self-describing**: Contains all parameters needed for decryption
**Integrity**: Entire envelope protected by AEAD authentication

## Security Properties

### Confidentiality
- **At rest**: All sensitive data encrypted with AES-256-GCM
- **In memory**: Secrets zeroized after use (best effort)
- **In transit**: No network communication (local-only design)

### Integrity
- **Cryptographic**: AEAD provides tamper detection
- **File-level**: Atomic writes prevent partial updates
- **Audit trail**: HMAC-chained log detects unauthorized modifications

### Availability
- **Backup**: Encrypted export/import functionality
- **Recovery**: Master key rotation preserves data access
- **Locking**: File locking prevents concurrent corruption

### Authentication
- **Passphrase-based**: User proves knowledge of master passphrase
- **Key derivation**: Argon2id makes brute force computationally expensive
- **No bypass**: No backdoors or recovery mechanisms

## Security Assumptions

### What We Protect Against
✅ **Offline attacks**: Stolen vault files cannot be decrypted without passphrase  
✅ **Brute force**: Argon2id makes password cracking expensive  
✅ **Tampering**: AEAD and audit logs detect unauthorized modifications  
✅ **Partial writes**: Atomic operations prevent corruption  
✅ **Weak randomness**: Uses OS cryptographically secure RNG  

### What We Cannot Protect Against
❌ **Memory attacks**: RAM dumps may reveal decrypted secrets  
❌ **Keyloggers**: Master passphrase entry can be captured  
❌ **Malicious OS**: Compromised kernel can access all data  
❌ **Physical access**: Unencrypted swap files, hibernation images  
❌ **Side channels**: Timing, power, electromagnetic analysis  
❌ **Rubber hose cryptanalysis**: Physical coercion of users  

### Trust Boundaries
- **Trusted**: CPU, hardware RNG, Go runtime, crypto libraries
- **Untrusted**: File system, operating system, other processes
- **User responsibility**: Secure passphrase selection and protection

## Implementation Security

### Memory Management
```go
// Secure memory clearing
func Zeroize(data []byte) {
    for i := range data {
        data[i] = 0
    }
}
```
**Limitations**: 
- Compiler optimizations may eliminate zeroization
- Go GC may move/copy data before clearing
- CPU caches and swap files not addressed

### Constant-Time Operations
```go
// Prevents timing attacks on comparisons
func SecureCompare(a, b []byte) bool {
    return subtle.ConstantTimeCompare(a, b) == 1
}
```

### Error Handling
- **Fail closed**: Decryption failures return errors, never partial data
- **Information leakage**: Error messages don't reveal internal state
- **Resource cleanup**: Defer statements ensure cleanup on errors

### File Operations
```go
// Atomic write pattern
tmpFile := vaultPath + ".tmp"
write(tmpFile, data)
fsync(tmpFile)
rename(tmpFile, vaultPath)
```

## Operational Security

### File Permissions
- **Vault files**: 0600 (owner read/write only)
- **Config files**: 0600 (owner read/write only)
- **Temporary files**: 0600, deleted immediately after use

### Process Isolation
- **Single process**: No daemon or background services
- **File locking**: Prevents concurrent vault access
- **Clean exit**: Zeroizes memory on termination signals

### Audit Logging
- **HMAC chain**: Cryptographically linked audit entries
- **Tamper detection**: `vault audit-log --verify` detects modifications
- **Minimal logging**: Only metadata logged, never secrets

## Configuration Security

### Default Parameters
All defaults chosen for security over performance:
```yaml
kdf:
  memory: 65536      # 64 MB
  iterations: 3      # ~300ms on modern hardware
  parallelism: 4     # CPU cores
clipboard:
  timeout: 30        # Auto-clear after 30 seconds
vault:
  auto_lock: 300     # Lock after 5 minutes idle
```

### Parameter Validation
- **KDF memory**: 1 MB minimum, 1 GB maximum
- **KDF iterations**: 1 minimum, 100 maximum
- **Parallelism**: 1-255 (uint8 range)
- **Timeouts**: Positive integers only

## Compliance and Standards

### Cryptographic Standards
- **NIST SP 800-132**: Password-Based Key Derivation
- **RFC 9106**: Argon2 specification
- **NIST SP 800-38D**: GCM mode specification
- **FIPS 140-2**: Approved cryptographic modules (Go crypto)

### Best Practices
- **OWASP**: Password storage cheat sheet compliance
- **SANS**: Secure coding practices
- **CWE**: Common weakness enumeration mitigation

## Security Testing

### Automated Tests
- **Unit tests**: All cryptographic functions
- **Integration tests**: End-to-end encryption/decryption
- **Fuzz testing**: Invalid input handling
- **Property tests**: Cryptographic properties verification

### Manual Testing
- **Penetration testing**: Simulated attacks on implementation
- **Code review**: Security-focused source code analysis
- **Threat modeling**: Systematic attack scenario analysis

### Test Vectors
```go
// Known answer tests for reproducibility
func TestKnownVectors(t *testing.T) {
    // Test with fixed inputs to ensure consistency
    passphrase := "test123"
    salt := [32]byte{0, 1, 2, ...} // Fixed salt
    
    key := deriveKey(passphrase, salt)
    expected := "a1b2c3..." // Known expected output
    
    assert.Equal(t, expected, hex.EncodeToString(key))
}
```

## Incident Response

### Vulnerability Disclosure
1. **Report**: security@vault-project.org
2. **Timeline**: 90-day coordinated disclosure
3. **Severity**: CVSS v3.1 scoring
4. **Mitigation**: Immediate patch for critical issues

### Compromise Response
1. **Immediate**: Change master passphrase
2. **Short-term**: Rotate all stored passwords
3. **Long-term**: Migrate to new vault file

### Monitoring
- **File integrity**: Monitor vault file modifications
- **Access patterns**: Unusual unlock times or locations
- **Failed attempts**: Multiple incorrect passphrase entries

## Future Considerations

### Cryptographic Agility
- **Algorithm updates**: Envelope versioning supports migration
- **Key rotation**: Master passphrase rotation preserves data
- **Quantum resistance**: Post-quantum algorithms when standardized

### Hardware Security
- **TPM integration**: Hardware-backed key storage
- **Secure enclaves**: Intel SGX, ARM TrustZone support
- **Hardware tokens**: FIDO2, PIV card integration

### Enhanced Protection
- **Memory encryption**: Intel TME, AMD SME support
- **Secure deletion**: Platform-specific secure erase
- **Anti-forensics**: Plausible deniability features

## Conclusion

The Password Vault CLI implements defense-in-depth security with:
- **Strong cryptography**: Industry-standard algorithms and parameters
- **Secure implementation**: Memory safety and constant-time operations
- **Operational security**: File permissions and audit logging
- **Threat awareness**: Clear documentation of limitations

Users must understand that no software can protect against all threats. The vault provides strong protection against offline attacks and tampering, but cannot defend against compromised systems or coerced users.

For maximum security:
1. Use a strong, unique master passphrase
2. Keep the system and vault software updated
3. Use full-disk encryption on storage devices
4. Avoid using the vault on compromised systems
5. Regularly audit vault access logs

**Security is a process, not a product. Stay vigilant.**