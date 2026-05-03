# Password Vault CLI Quick Guide

Password Vault CLI stores passwords and local identity artifacts in one encrypted vault.

## What It Does

- manages password entries offline
- organizes data by profile
- creates local `did:jwk` identifiers
- issues signed verifiable credentials
- generates and verifies challenge-based zero-knowledge proofs of DID key possession

## Fast Start

```bash
go build -o vault ./cmd/vault
./vault init
./vault unlock
./vault add github --username myuser --secret-prompt
./vault did create issuer
./vault did create subject
./vault vc issue badge --issuer issuer --subject subject --type EmployeeCredential --claim role=backend
./vault zk-proof --did issuer --challenge demo-1 --output proof.json
./vault zk-proof verify --did issuer --challenge demo-1 --proof proof.json
```

## Important Limits

- identity support is local-only
- only `did:jwk` is implemented
- claims are flat `key=value` pairs
- the proof flow demonstrates key possession, not claim disclosure
