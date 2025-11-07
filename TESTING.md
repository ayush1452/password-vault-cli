# Testing Guide for Vault CLI

This document outlines the testing strategy, best practices, and guidelines for the Vault CLI project.

## Table of Contents
1. [Test Structure](#test-structure)
2. [List Command Examples](#list-command-examples)
3. [Test Scenarios](#test-scenarios)
4. [Writing Tests](#writing-tests)
5. [Test Helpers](#test-helpers)
6. [Best Practices](#best-practices)
7. [Running Tests](#running-tests)
8. [Troubleshooting](#troubleshooting)

## Test Structure

### Unit Tests
- Located in `internal/*/` directories
- Follow the pattern `*_test.go`
- Use standard library `testing` package
- Table-driven tests are encouraged
- Example: `TestListCommandDetailed` in `internal/cli/cli_test.go`

### Integration Tests
- End-to-end tests for CLI commands
- Test actual command execution
- Located in `cmd/*_test.go`
- Test command-line flags, arguments, and output

## List Command Guide for Beginners

The `list` command helps you view your saved password entries. Here's everything you need to know to get started.

### Basic Commands

#### 1. View All Entries
```bash
# See a simple list of all your saved entries
vault list
```

Example output:
```
NAME
----
aws-prod
github
gitlab

Found 3 entries in profile 'default'
```

#### 2. View Detailed Information
```bash
# See more details about each entry
vault list --long
```

Example output:
```
Found 3 entries in profile 'default'
NAME      USERNAME    TAGS           UPDATED_AT
----      --------    ----           ----------
aws-prod  awsuser     work,aws,prod  2025-11-07 15:30
github    testuser    work,git       2025-11-07 15:30
gitlab    gitlabuser  work,git       2025-11-07 15:30
```

#### 3. Get Output as JSON
```bash
# Useful for scripts or processing the output
vault list --json

# Example output:
# {
#   "entries": [
#     {
#       "name": "github",
#       "username": "testuser",
#       "url": "github.com",
#       "tags": ["work", "git"],
#       "updated_at": "2025-11-07T15:30:00Z"
#     },
#     ...
#   ],
#   "count": 3
# }
```

### Finding Specific Entries

#### 1. Filter by Tags
```bash
# Find entries with a specific tag
vault list --tags work

# Find entries with either tag (OR logic)
vault list --tags work,personal
```

#### 2. Search Entries
```bash
# Search in name, username, or URL
vault list --search git

# Example output (would match 'github' and 'gitlab'):
# NAME
# ----
# github
# gitlab
```

#### 3. Combine Filters
```bash
# Find work-related git accounts (AND logic)
vault list --tags work --search git --long

# Example output:
# NAME    USERNAME    TAGS      UPDATED_AT
# ----    --------    ----      ----------
# github  testuser    work,git  2025-11-07 15:30
# gitlab  gitlabuser  work,git  2025-11-07 15:30
```

### Pro Tips
- Use `--long` to see more details about each entry
- Combine multiple tags with commas (no spaces)
- The search looks in name, username, and URL fields
- Use `--json` when you need to process the output with scripts

### Common Use Cases

#### 1. Find All Work Accounts
```bash
vault list --tags work
```

#### 2. Find a Specific Service
```bash
# Find all GitHub related accounts
vault list --search github
```

#### 3. Get Full Details for an Entry
```bash
# First find the entry name
vault list
# Then get details
vault list --search github --long
```

## Test Scenarios

### List Command Tests

#### Basic Functionality
- [x] List all entries
- [x] List with detailed output
- [x] List with JSON output

#### Filtering
- [x] Filter by single tag
- [x] Filter by multiple tags (OR logic)
- [x] Search by text
- [x] Combine tag and search filters (AND logic)

#### Edge Cases
- [x] List with non-existent tag
- [x] List with empty search term
- [x] List with special characters in search
- [x] List with very long entry names

#### Output Validation
- [x] Verify table formatting
- [x] Verify JSON output structure
- [x] Verify error messages
- [x] Verify no sensitive data leakage

## Writing Tests

### Test Naming Conventions
- Use `TestFunctionName` for basic tests
- Use `TestFunctionName_Scenario` for specific scenarios
- For table-driven tests, use descriptive names that explain the test case

### Table-Driven Tests
Example structure:
```go
func TestFunction(t *testing.T) {
    tests := []struct {
        name        string
        input       string
        expected    string
        expectError bool
    }{
        {
            name:        "valid input",
            input:       "test",
            expected:    "expected output",
            expectError: false,
        },
        // more test cases...
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

## Test Scenarios

### List Command Tests

#### Basic Functionality
- List all entries
- List with no entries
- List with specific tag filter
- List with multiple tag filters (OR logic)
- List with search term
- List with multiple search terms (fuzzy AND matching)
- List with long format output
- List with JSON output

#### Edge Cases
- List with non-existent tag
- List with empty search term
- List with special characters in search
- List with very long entry names
- List with maximum number of entries

#### Output Validation
- Verify table formatting
- Verify JSON output structure
- Verify error messages
- Verify no sensitive data leakage

### Recent Changes and Fixes

#### Output Handling Fix (2023-11-06)
- **Issue**: Tests were failing to capture "No entries found" messages
- **Root Cause**: Direct `fmt.Println` usage instead of using command's output writer
- **Fix**: Updated to use `fmt.Fprintln(cmd.OutOrStdout(), ...)`
- **Files Affected**: `internal/cli/list.go`
- **Test Impact**: Fixed `List_with_non-existent_tag` test case

#### JSON Output Testing
- **Improvement**: Enhanced JSON output validation
- **Approach**: Parse JSON and validate structure/values instead of string matching
- **Benefits**: More robust tests, less fragile to whitespace/formatting changes

#### Test Coverage Improvements
- Added tests for edge cases in list command
- Improved test assertions with descriptive error messages
- Added debug logging for troubleshooting test failures

## Best Practices

1. **Test Organization**
   - Group related tests using subtests
   - Use descriptive test names
   - Keep tests focused and independent

2. **Test Data**
   - Use consistent test data across tests
   - Clean up test data after tests
   - Use test helpers for common setup/teardown

3. **Assertions**
   - Use `t.Helper()` in test helpers
   - Provide clear error messages
   - Test both success and failure cases

4. **Performance**
   - Use `t.Parallel()` for independent tests
   - Keep tests fast and focused
   - Use build tags for integration tests

## Running Tests

### Run All Tests
```bash
go test -v ./...
```

### Run Specific Test
```bash
go test -v -run TestName ./...
```

### Run Tests with Race Detector
```bash
go test -race -v ./...
```

### Run Tests with Coverage
```bash
go test -coverprofile=coverage.out ./...
```

## Troubleshooting

### Common Issues

#### Test Fails with "No entries found"
- **Cause**: Output might be going to stderr instead of stdout
- **Solution**: Check both stdout and stderr in test assertions

#### JSON Test Failures
- **Cause**: Whitespace or formatting differences
- **Solution**: Parse JSON and validate structure instead of string matching

#### Vault Locking Issues
- **Cause**: Multiple tests trying to access vault simultaneously
- **Solution**: Use `unlockWithSession` helper and ensure proper cleanup

### Debugging Tests

#### Enable Verbose Output
```bash
go test -v ./...
```

#### Debug Specific Test
```go
t.Logf("Debug: %v", variable)
t.Logf("Stdout: %q", stdout)
t.Logf("Stderr: %q", stderr)
```

#### Print Command Output
```go
fmt.Printf("Output: %s\n", output)
```

## Adding New Tests

1. Identify the functionality to test
2. Create a new test function or add to existing table-driven test
3. Include edge cases and error conditions
4. Update this document if adding new test patterns or conventions

## Writing Tests

### Table-Driven Tests
Use table-driven tests for testing multiple scenarios:

```go
func TestSomething(t *testing.T) {
    tests := []struct {
        name        string
        input       string
        expected    string
        expectError bool
    } {
        {
            name:        "valid input",
            input:       "test",
            expected:    "test",
            expectError: false,
        },
        // more test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := SomeFunction(tt.input)
            if tt.expectError {
                require.Error(t, err)
                return
            }
            require.NoError(t, err)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

### Testing CLI Commands
Use the `TestHelper` to test CLI commands:

```go
func TestCommand(t *testing.T) {
    helper := NewTestHelper(t)
    defer helper.Cleanup()
    
    cmd := NewCommand()
    stdout, stderr, err := helper.ExecuteCommand(t, cmd, "arg1", "--flag", "value")
    
    require.NoError(t, err)
    assert.Contains(t, stdout, "expected output")
    assert.Empty(t, stderr)
}
```

## Test Helpers

### TestHelper
Provides common test utilities:
- `NewTestHelper(t *testing.T)`: Create a new test helper
- `ExecuteCommand()`: Execute a command and capture output
- `CreateTestFile()`: Create a temporary test file
- `TempDir()`: Create a temporary directory

### Common Assertions
Use these patterns for assertions:

```go
// Check for errors
require.NoError(t, err)  // Fails immediately
assert.NoError(t, err)   // Continues test

// Check values
assert.Equal(t, expected, actual)
assert.Contains(t, stringValue, "substring")
assert.True(t, condition)

// Check JSON
var data SomeStruct
err := json.Unmarshal(jsonData, &data)
require.NoError(t, err)
assert.Equal(t, "expected", data.Field)
```

## Best Practices

### 1. Test Structure
- Keep tests focused and simple
- Test one thing per test case
- Use descriptive test names
- Group related tests with subtests

### 2. Test Data
- Use constants for expected values
- Create test data in setup functions
- Clean up resources using `t.Cleanup()`

### 3. Error Handling
- Always check errors in tests
- Test both success and error cases
- Verify error messages when testing error cases

### 4. Performance
- Use `t.Parallel()` for independent tests
- Avoid unnecessary sleeps or delays
- Use test helpers to reduce code duplication

### 5. Maintainability
- Keep tests up-to-date with code changes
- Document complex test cases
- Add comments for non-obvious test logic

## Running Tests

### Run All Tests
```bash
go test -v ./...
```

### Run Specific Test
```bash
go test -v -run TestName ./...
```

### Run Tests with Coverage
```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Run Tests with Race Detection
```bash
go test -race ./...
```

## Troubleshooting

### Common Issues

#### 1. Vault Locked
If you see "vault is locked" errors:
- Ensure tests properly unlock the vault
- Use `unlockWithSession` helper
- Check for test cleanup issues

#### 2. Test Data Conflicts
- Use unique test data for each test
- Clean up after tests
- Use `t.Cleanup()` for resource cleanup

#### 3. Flaky Tests
- Avoid time-based tests when possible
- Use test doubles for external services
- Make tests independent of each other

### Debugging Tests
1. Run with `-v` for verbose output
2. Use `t.Log()` for debug information
3. Check both stdout and stderr in test output
4. Use `-run` to isolate failing tests

## Adding New Tests

1. Create a new test file or add to an existing one
2. Follow the existing patterns
3. Add test cases for:
   - Success scenarios
   - Error conditions
   - Edge cases
   - Invalid inputs
4. Run tests locally before committing
5. Update this document if adding new test patterns

## Code Review

When reviewing test code, check for:
- Test coverage of edge cases
- Proper error handling
- Resource cleanup
- Test independence
- Clear assertions
- No test-only code in production files
