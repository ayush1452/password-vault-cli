# Contributing to Password Vault CLI

Thank you for your interest in contributing to the Password Vault CLI! This document provides guidelines and information for contributors.

## 🎯 Project Goals

- **Security First**: Every change must maintain or improve security
- **Simplicity**: Keep the codebase clean and maintainable
- **Performance**: Optimize for speed and resource efficiency
- **Cross-Platform**: Ensure compatibility across operating systems
- **Zero Dependencies**: Minimize external dependencies for security

## 🚀 Getting Started

### Development Environment Setup

1. **Prerequisites**
   ```bash
   # Install Go 1.21 or later
   go version  # Should be 1.21+
   
   # Install development tools
   go install golang.org/x/tools/cmd/goimports@latest
   go install github.com/securecodewarrior/sast-scan@latest
   go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
   ```

2. **Clone and Setup**
   ```bash
   git clone https://github.com/vault-cli/vault.git
   cd vault
   
   # Install dependencies
   go mod download
   
   # Verify setup
   make test
   make build
   ```

3. **IDE Configuration**
   - **VS Code**: Install Go extension, configure goimports on save
   - **GoLand**: Enable Go modules, set up code formatting
   - **Vim/Neovim**: Install vim-go or similar Go plugin

### Project Structure

```
vault/
├── cmd/vault/              # CLI entry point
│   └── main.go
├── internal/               # Private application code
│   ├── cli/               # CLI command implementations
│   │   ├── root.go        # Root command and global flags
│   │   ├── init.go        # Vault initialization
│   │   ├── add.go         # Add entries
│   │   └── ...
│   ├── vault/             # Cryptographic engine
│   │   ├── crypto.go      # Core crypto operations
│   │   ├── envelope.go    # Data envelope format
│   │   └── ...
│   ├── store/             # Storage layer
│   │   ├── store.go       # Storage interface
│   │   ├── bbolt.go       # BoltDB implementation
│   │   └── ...
│   ├── domain/            # Domain models
│   ├── config/            # Configuration management
│   └── util/              # Shared utilities
├── tests/                 # Test suites
│   ├── security_tests.go  # Security scenarios
│   ├── fuzz_tests.go      # Fuzz testing
│   └── acceptance_tests.go # End-to-end tests
├── docs/                  # Documentation
└── Makefile              # Build system
```

## 🔧 Development Workflow

### 1. Issue Creation

Before starting work:

1. **Check existing issues** to avoid duplication
2. **Create an issue** describing the problem or feature
3. **Discuss the approach** with maintainers
4. **Get approval** for significant changes

### 2. Branch Strategy

```bash
# Create feature branch
git checkout -b feature/your-feature-name

# Create bugfix branch
git checkout -b bugfix/issue-description

# Create security fix branch
git checkout -b security/vulnerability-description
```

### 3. Development Process

1. **Write Tests First** (TDD approach)
   ```bash
   # Write failing tests
   make test  # Should fail
   
   # Implement feature
   # ...
   
   # Verify tests pass
   make test  # Should pass
   ```

2. **Code Implementation**
   - Follow Go conventions and idioms
   - Add comprehensive error handling
   - Include security considerations
   - Document public APIs

3. **Security Review**
   ```bash
   # Run security tests
   make test-security
   
   # Run static analysis
   make lint
   
   # Check for vulnerabilities
   make security-scan
   ```

### 4. Testing Requirements

All contributions must include tests:

- **Unit Tests**: Test individual functions/methods
- **Integration Tests**: Test component interactions
- **Security Tests**: Test security scenarios
- **Performance Tests**: Benchmark critical paths

```bash
# Run all tests
make test

# Run specific test suites
make test-unit
make test-integration
make test-security
make test-fuzz

# Generate coverage report
make coverage
```

### 5. Code Quality

#### Code Style

```bash
# Format code
make fmt

# Run linter
make lint

# Fix common issues
make fix
```

#### Documentation

- **Public APIs**: Must have godoc comments
- **Complex Logic**: Inline comments explaining why
- **Security Code**: Detailed security considerations
- **Examples**: Include usage examples

#### Error Handling

```go
// Good: Specific error types
if err != nil {
    return fmt.Errorf("failed to encrypt entry: %w", err)
}

// Bad: Generic errors
if err != nil {
    return err
}
```

## 🛡️ Security Guidelines

### Security-First Development

1. **Threat Modeling**
   - Consider attack vectors for every change
   - Document security assumptions
   - Review cryptographic usage

2. **Secure Coding Practices**
   ```go
   // Always zeroize sensitive data
   defer vault.Zeroize(secretData)
   
   // Use constant-time comparisons
   if !vault.SecureCompare(expected, actual) {
       return errors.New("authentication failed")
   }
   
   // Validate all inputs
   if err := validateEntryName(name); err != nil {
       return fmt.Errorf("invalid entry name: %w", err)
   }
   ```

3. **Cryptographic Requirements**
   - Use only approved algorithms (Argon2id, AES-256-GCM)
   - Generate cryptographically secure random numbers
   - Never reuse nonces or IVs
   - Implement proper key derivation

### Security Testing

```bash
# Run comprehensive security tests
make test-security

# Test with malicious inputs
make test-fuzz

# Validate crypto implementations
make test-crypto

# Check for timing attacks
make test-timing
```

## 📝 Pull Request Process

### 1. Pre-Submission Checklist

- [ ] Tests pass: `make test`
- [ ] Code formatted: `make fmt`
- [ ] Linting clean: `make lint`
- [ ] Security tests pass: `make test-security`
- [ ] Documentation updated
- [ ] CHANGELOG.md updated (if applicable)

### 2. Pull Request Template

```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Security improvement
- [ ] Performance optimization
- [ ] Documentation update

## Security Impact
- [ ] No security impact
- [ ] Improves security
- [ ] Requires security review

## Testing
- [ ] Unit tests added/updated
- [ ] Integration tests added/updated
- [ ] Security tests added/updated
- [ ] Manual testing completed

## Checklist
- [ ] Code follows project style guidelines
- [ ] Self-review completed
- [ ] Documentation updated
- [ ] Tests pass locally
```

### 3. Review Process

1. **Automated Checks**: CI/CD pipeline runs tests
2. **Security Review**: Security-sensitive changes reviewed by maintainers
3. **Code Review**: At least one maintainer approval required
4. **Final Testing**: Manual testing in clean environment

## 🐛 Bug Reports

### Bug Report Template

```markdown
## Bug Description
Clear description of the bug

## Steps to Reproduce
1. Step one
2. Step two
3. Step three

## Expected Behavior
What should happen

## Actual Behavior
What actually happens

## Environment
- OS: [e.g., macOS 13.0]
- Go Version: [e.g., 1.21.0]
- Vault Version: [e.g., 1.0.0]

## Additional Context
Any other relevant information
```

### Security Issues

**⚠️ Do NOT create public issues for security vulnerabilities**

Instead:
1. Email: security@vault-cli.dev
2. Include: Detailed description and reproduction steps
3. Response: We'll respond within 48 hours
4. Disclosure: Coordinated disclosure after fix

## 🎨 Feature Requests

### Feature Request Template

```markdown
## Feature Description
Clear description of the proposed feature

## Use Case
Why is this feature needed?

## Proposed Solution
How should this work?

## Alternatives Considered
Other approaches you've considered

## Security Considerations
Any security implications
```

## 📚 Documentation

### Documentation Standards

1. **README.md**: Keep updated with new features
2. **API Documentation**: Godoc for all public APIs
3. **Architecture Docs**: Update for structural changes
4. **Security Docs**: Document security implications

### Writing Guidelines

- **Clear and Concise**: Easy to understand
- **Examples**: Include practical examples
- **Up-to-Date**: Keep documentation current
- **Security Focus**: Highlight security considerations

## 🏆 Recognition

Contributors are recognized in:

- **CONTRIBUTORS.md**: All contributors listed
- **Release Notes**: Major contributions highlighted
- **GitHub**: Contributor badges and statistics

## 📞 Getting Help

- **GitHub Discussions**: General questions and ideas
- **GitHub Issues**: Bug reports and feature requests
- **Email**: security@vault-cli.dev for security issues
- **Documentation**: Check docs/ directory first

## 🎯 Good First Issues

Look for issues labeled:
- `good-first-issue`: Perfect for newcomers
- `help-wanted`: Community contributions welcome
- `documentation`: Documentation improvements
- `testing`: Test coverage improvements

## 📋 Development Commands

```bash
# Development workflow
make dev-setup          # Setup development environment
make dev-test           # Run tests in development mode
make dev-build          # Build development binary

# Code quality
make fmt                # Format code
make lint               # Run linter
make fix                # Auto-fix issues
make security-scan      # Security static analysis

# Testing
make test               # All tests
make test-unit          # Unit tests only
make test-integration   # Integration tests only
make test-security      # Security tests only
make test-fuzz          # Fuzz tests only
make coverage           # Generate coverage report

# Build and release
make build              # Build binary
make build-all          # Build for all platforms
make release            # Create release artifacts
```

## 🔄 Release Process

1. **Version Bump**: Update version in code and docs
2. **Changelog**: Update CHANGELOG.md
3. **Testing**: Full test suite on all platforms
4. **Security Review**: Final security audit
5. **Release**: Create GitHub release with artifacts

---

Thank you for contributing to Password Vault CLI! Your help makes this project more secure and useful for everyone. 🙏