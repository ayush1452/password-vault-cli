# Developer Guide: Password Vault CLI

This guide provides essential information for developers working on the Password Vault CLI project, focusing on common issues and their solutions in the CI/CD pipeline.

## Table of Contents
1. [CI/CD Pipeline Overview](#cicd-pipeline-overview)
2. [Common Issues and Solutions](#common-issues-and-solutions)
   - [Linting Issues](#linting-issues)
   - [Static Analysis Issues](#static-analysis-issues)
   - [Security Scanning Issues](#security-scanning-issues)
   - [Testing Issues](#testing-issues)
3. [Local Development Setup](#local-development-setup)
4. [Best Practices](#best-practices)
5. [Troubleshooting](#troubleshooting)

## CI/CD Pipeline Overview

The project uses GitHub Actions for CI/CD with the following main stages:

1. **Linting**: Runs `golangci-lint` to check code style and common issues
2. **Static Analysis**: Runs `errcheck`, `staticcheck`, and other static analysis tools
3. **Security Scanning**: Uses CodeQL and gosec to identify security vulnerabilities
4. **Testing**: Runs unit, integration, and fuzz tests
5. **Build**: Builds the application for different platforms

## Common Issues and Solutions

### Linting Issues

#### 1. File Formatting (`gofumpt`)
- **Issue**: "File is not properly formatted"
- **Solution**:
  ```bash
  # Fix formatting issues
  make fmt
  
  # Or manually with gofumpt
  gofumpt -w .
  ```

#### 2. Unused Imports
- **Issue**: "imported and not used"
- **Solution**:
  - Remove unused imports
  - Or use blank identifier `_` if the import is for side effects
  ```go
  import (
      _ "github.com/example/package" // For side effects
  )
  ```

### Static Analysis Issues

#### 1. Unhandled Errors (`errcheck`)
- **Issue**: Error return values not checked
- **Solution**:
  - Always handle errors explicitly
  - Use `//nolint:errcheck` only when you're certain it's safe to ignore
  - For cleanup operations, log the error:
  ```go
  if err := resource.Cleanup(); err != nil {
      log.Printf("Warning: cleanup failed: %v", err)
  }
  ```

#### 2. Type Assertions Without Error Handling
- **Issue**: Type assertions without checking `ok` value
- **Solution**:
  ```go
  // Bad
  value := someInterface.(SomeType)
  
  // Good
  value, ok := someInterface.(SomeType)
  if !ok {
      return fmt.Errorf("expected SomeType, got %T", someInterface)
  }
  ```

### Security Scanning Issues

#### 1. Hardcoded Credentials
- **Issue**: Potential hardcoded credentials found
- **Solution**:
  - Use environment variables or secure configuration management
  - Never commit sensitive data to version control

#### 2. Insecure Random Number Generation
- **Issue**: Using `math/rand` instead of `crypto/rand`
- **Solution**:
  ```go
  // Bad
  import "math/rand"
  
  // Good
  import "crypto/rand"
  ```

### Testing Issues

#### 1. Fuzz Test Failures
- **Issue**: Fuzz tests failing due to unhandled errors
- **Solution**:
  - Ensure all error conditions are handled in test code
  - Use `t.Fatalf` for test setup failures
  - Clean up resources with `defer`

#### 2. Race Conditions
- **Issue**: Data races detected
- **Solution**:
  - Use mutexes for shared state
  - Run tests with `-race` flag locally
  ```bash
  go test -race ./...
  ```

## Local Development Setup

### Prerequisites
- Go 1.24.10 or later
- `golangci-lint` for local linting
- `gofumpt` for code formatting
- `errcheck` for static analysis
- `staticcheck` for advanced static analysis
- `gosec` for security scanning

### Complete Setup Instructions
1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/password-vault-cli.git
   cd password-vault-cli
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Install development tools:
   ```bash
   # Install linters and formatters
   go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.55.2
   go install mvdan.cc/gofumpt@latest
   
   # Install static analysis tools
   go install github.com/kisielk/errcheck@latest
   go install honnef.co/go/tools/cmd/staticcheck@latest
   go install github.com/securego/gosec/v2/cmd/gosec@latest
   ```

### Comprehensive Local Testing Commands

#### Running Tests
```bash
# Run all tests
make test

# Run tests with race detector
make test-race

# Run only unit tests
go test -short ./...

# Run tests with coverage
make coverage

# Run fuzz tests
go test -fuzz=FuzzEntryValidation -fuzztime=30s ./tests

# Run benchmark tests
go test -run=^$ -bench=. -benchmem ./tests
```

#### Linting and Static Analysis
```bash
# Run all linters (same as CI)
make lint

# Run specific linters
golangci-lint run --enable=errcheck ./...
golangci-lint run --enable=staticcheck ./...

# Check for unused code
staticcheck ./...

# Check for unchecked errors
errcheck ./...

# Check for security issues
gosec ./...
```

#### Code Formatting
```bash
# Format all Go files
gofumpt -w .

# Check formatting without changing files
gofumpt -l .

# Format imports
goimports -w .
```

#### Build and Run
```bash
# Build the application
make build

# Install to $GOPATH/bin
make install

# Run with debug logging
VAULT_DEBUG=true ./password-vault-cli
```

#### Pre-commit Hook (Recommended)
Create a `.git/hooks/pre-commit` file with:
```bash
#!/bin/sh
set -e

echo "Running pre-commit checks..."

echo "- Formatting code..."
gofumpt -l -w .

echo "- Running linters..."
golangci-lint run --fast

echo "- Running tests..."
go test -short ./...

echo "All checks passed!"
```
Then make it executable:
```bash
chmod +x .git/hooks/pre-commit
```

## Best Practices

### Error Handling
- Always handle errors explicitly
- Provide meaningful error messages
- Log errors when appropriate
- Use `errors.Wrap`/`fmt.Errorf("%w", err)` for context

### Testing
- Write table-driven tests for multiple test cases
- Use test helpers to reduce duplication
- Test error conditions
- Use `t.Cleanup` for test cleanup

### Security
- Never log sensitive data
- Use `vault.Zeroize` to clear sensitive data from memory
- Follow the principle of least privilege

## Troubleshooting

### CI Pipeline Failing
1. Check the specific job that's failing
2. Look for error messages in the logs
3. Reproduce locally using the same commands
4. Check for environment differences

### Common Problems
1. **Import Cycle**
   - Restructure packages to avoid circular dependencies
   - Consider using interfaces for package boundaries

2. **Race Conditions**
   - Run tests with `-race` flag
   - Use proper synchronization primitives

3. **Memory Leaks**
   - Use `defer` for resource cleanup
   - Be careful with goroutines and channels

### Getting Help
- Check the project's issue tracker
- Review the code of conduct
- Follow the pull request template when submitting changes
