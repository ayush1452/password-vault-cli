# System Design

## Purpose

The system is a local-only vault that stores encrypted password data and, now, profile-scoped decentralized identity artifacts.

## Core Flows

### Password flow

1. Initialize the vault and derive the master key from the passphrase.
2. Unlock the vault for a session.
3. Encrypt entry records and store them in `entries:<profile>`.
4. Retrieve or rotate secrets only while the session is unlocked.

### Identity flow

1. Unlock the vault.
2. Create a local DID with `vault did create <name>`.
3. Store the encrypted DID record in `dids:<profile>`.
4. Issue a signed credential with `vault vc issue ...`.
5. Store the credential record in `credentials:<profile>`.
6. Generate a proof for a challenge with `vault zk-proof`.
7. Verify the proof with `vault zk-proof verify`.

## Data Boundaries

### Encrypted at rest

- password entries
- DID records, including private JWK material
- credential records

### Publicly exportable artifacts

- DID documents
- signed verifiable credentials
- zero-knowledge proof JSON generated for a verifier challenge

## Storage Layout

```text
metadata
profiles
audit
entries:<profile>
dids:<profile>
credentials:<profile>
```

## Identity Design Choices

- DID method is fixed to `did:jwk`.
- P-256 keys are used for the MVP.
- Credentials are signed with ES256 over canonical JSON.
- Claims are stored as sorted flat `key=value` pairs.
- Proofs demonstrate DID key possession only.

## Operational Hooks

- `status` surfaces DID and credential counts for the active profile.
- `doctor` validates identity buckets and verifies stored credentials.
- snapshot export/import includes identity data alongside passwords and audit history.
- audit logging records DID creation, credential issuance, and proof generation/verification.

## Explicit Non-Goals

- remote resolution of DIDs
- blockchain anchoring
- credential revocation infrastructure
- selective disclosure over credential claims
- nested claim objects in the CLI
