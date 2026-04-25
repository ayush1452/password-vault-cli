# Developer Guide

## Where The Identity Feature Lives

| Area | Files |
| --- | --- |
| CLI surface | `internal/cli/did.go`, `internal/cli/vc.go`, `internal/cli/zk_proof.go`, `internal/cli/identity_helpers.go` |
| Core identity logic | `internal/identity/*` |
| Persistence | `internal/store/store.go`, `internal/store/bbolt.go` |
| Export/import wrapping | `internal/cli/export.go`, `internal/cli/import.go`, `internal/vault/export.go` |
| Operational checks | `internal/cli/status.go`, `internal/cli/doctor.go` |

## Implementation Rules

- DIDs are profile-scoped vault records.
- The DID method is `did:jwk`.
- Keys are P-256.
- Credentials use flat `key=value` claims.
- Credential signing uses deterministic canonical JSON plus ES256.
- Zero-knowledge proofs bind to an explicit verifier challenge.

## Minimum Verification Before Sending A Change

```bash
go test ./internal/identity
go test ./internal/store
go test ./internal/cli -run 'TestDIDCreateAndShowCommand|TestVCIssueAndVerifyCommand|TestZKProofGenerateAndVerifyCommand'
go test ./tests/integration/identity
go test ./tests/e2e -run TestIdentityWorkflow
go test ./tests/security -run 'TestKeyProofRejectsMismatchedChallenge|TestPublicOnlySnapshotOmitsPrivateJWK'
```

## Working Notes

- If you touch snapshot behavior, test both public-only exports and secret-bearing exports.
- If you touch claims or signing input, re-run the tamper and round-trip tests.
- If you touch CLI output or command wiring, verify both stored-artifact and file-based verification paths.
