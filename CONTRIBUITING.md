# Contributing

## Development Areas

Important directories:

- `internal/cli`
- `internal/store`
- `internal/identity`
- `internal/vault`
- `tests`

## Expectations For Identity Changes

If you change the decentralized identity feature, include:

- code changes in the relevant CLI, store, or identity package
- unit tests for the new behavior
- integration, e2e, or security coverage when signatures, proofs, or snapshots change
- documentation updates in the repo markdown files

## Minimum Test Commands

```bash
go test ./internal/identity
go test ./internal/store
go test ./internal/cli -run 'TestDIDCreateAndShowCommand|TestVCIssueAndVerifyCommand|TestZKProofGenerateAndVerifyCommand'
go test ./tests/integration/identity
go test ./tests/e2e -run TestIdentityWorkflow
go test ./tests/security -run 'TestKeyProofRejectsMismatchedChallenge|TestPublicOnlySnapshotOmitsPrivateJWK'
```
