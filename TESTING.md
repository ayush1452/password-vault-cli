# Testing Guide

This repository uses package-level unit tests, store and CLI integration tests, end-to-end CLI workflows, and security-focused scenarios.

## Test Layout

| Area | Location | Focus |
| --- | --- | --- |
| Unit tests | `internal/*/*_test.go` | Package behavior in isolation |
| Integration tests | `tests/integration/*` | Store, CLI, config, audit, export/import, and identity flows |
| End-to-end tests | `tests/e2e/*` | Real binary workflows |
| Security tests | `tests/security/*` | Tampering, challenge mismatch, export secrecy, session and file-security scenarios |
| Benchmarks | `tests/benchmarks/*` | Performance tracking |
| Fuzz tests | `tests/fuzz/*` | Input hardening |

## Identity Feature Coverage

The decentralized identity MVP adds coverage in four layers.

### Unit coverage

`internal/identity/identity_test.go` validates:

- DID generation and DID document encoding
- deterministic canonical JSON
- credential sign and verify
- tampered credential rejection
- proof generation and verification
- wrong challenge rejection
- wrong DID rejection
- record serialization round trips

### Store coverage

`internal/store/store_test.go` validates:

- identity CRUD
- credential CRUD
- profile isolation
- snapshot export/import for identities and credentials
- omission of private JWKs from public-only snapshots

### CLI coverage

`internal/cli/identity_commands_test.go` validates:

- `vault did create`
- `vault did show`
- `vault vc issue`
- `vault vc verify`
- `vault zk-proof`
- `vault zk-proof verify`

### Integration, e2e, and security coverage

- `tests/integration/identity/identity_integration_test.go`
- `tests/e2e/identity_workflow_test.go`
- `tests/security/identity_security_test.go`

These cover full workflows, exported artifacts, and challenge-binding/security behavior.

## Recommended Commands

Run the new feature-focused suites:

```bash
go test ./internal/identity
go test ./internal/store
go test ./internal/cli -run 'TestDIDCreateAndShowCommand|TestVCIssueAndVerifyCommand|TestZKProofGenerateAndVerifyCommand'
go test ./tests/integration/identity
go test ./tests/e2e -run TestIdentityWorkflow
go test ./tests/security -run 'TestKeyProofRejectsMismatchedChallenge|TestPublicOnlySnapshotOmitsPrivateJWK'
```

Run broader suites:

```bash
go test ./...
go test -race ./...
go test ./tests/benchmarks -run '^$' -bench .
go test ./tests/fuzz -fuzz . -fuzztime=30s
```

## Writing New Tests

- Prefer table-driven tests for input validation and signature/proof edge cases.
- Keep identity claims flat in tests because nested JSON claims are intentionally out of scope.
- When testing exports, cover both public-only snapshots and secret-bearing snapshots.
- When testing proofs, always include wrong-challenge and wrong-DID cases.

## Expected Verification For Identity Changes

Any change to `internal/identity`, `internal/store`, or the identity CLI commands should include:

- unit coverage in `internal/identity`
- a store or CLI test for the code path you changed
- at least one exported-artifact or security-oriented assertion when signatures, proofs, or snapshots are involved
