# Repository File Tree

This is the current high-level structure of the repository after adding the decentralized identity feature.

```text
.
├── cmd/
│   └── vault/
│       └── main.go
├── docs/
│   ├── ARCHITECTURE.md
│   ├── BINARY_ARTIFACTS.md
│   ├── BENCHMARK_PUBLISHING.md
│   ├── SECURITY_VALIDATION.md
│   ├── TESTING_BENCHMARKS.md
│   └── USAGE.md
├── examples/
│   ├── README.md
│   ├── cli/main.go
│   ├── crypto/main.go
│   └── storage/main.go
├── internal/
│   ├── cli/
│   │   ├── add.go
│   │   ├── audit.go
│   │   ├── config.go
│   │   ├── delete.go
│   │   ├── did.go
│   │   ├── doctor.go
│   │   ├── export.go
│   │   ├── get.go
│   │   ├── identity_helpers.go
│   │   ├── import.go
│   │   ├── init.go
│   │   ├── list.go
│   │   ├── lock.go
│   │   ├── passgen.go
│   │   ├── profiles.go
│   │   ├── root.go
│   │   ├── rotate.go
│   │   ├── rotate_password.go
│   │   ├── session.go
│   │   ├── status.go
│   │   ├── unlock.go
│   │   ├── update.go
│   │   ├── vc.go
│   │   └── zk_proof.go
│   ├── clipboard/
│   ├── config/
│   ├── crypto/
│   ├── domain/
│   ├── identity/
│   │   ├── canonical.go
│   │   ├── credential.go
│   │   ├── did.go
│   │   ├── json.go
│   │   ├── proof.go
│   │   ├── types.go
│   │   └── identity_test.go
│   ├── store/
│   │   ├── bbolt.go
│   │   ├── store.go
│   │   └── store_test.go
│   ├── util/
│   └── vault/
│       ├── README.md
│       ├── crypto.go
│       ├── export.go
│       ├── export_test.go
│       ├── meta.go
│       └── query.go
├── tests/
│   ├── README.md
│   ├── TEST_STRATEGY.md
│   ├── UNIT_TESTS.md
│   ├── benchmarks/
│   ├── e2e/
│   │   ├── complete_workflow_test.go
│   │   ├── identity_workflow_test.go
│   │   ├── password_rotation_test.go
│   │   └── session_management_test.go
│   ├── fuzz/
│   ├── integration/
│   │   ├── audit/
│   │   ├── cli/
│   │   ├── clipboard/
│   │   ├── config/
│   │   ├── doctor/
│   │   ├── export_import/
│   │   ├── identity/
│   │   ├── profiles/
│   │   └── store/
│   ├── manual/
│   └── security/
│       ├── attack_scenarios_test.go
│       ├── crypto_security_test.go
│       ├── identity_security_test.go
│       ├── permission_security_test.go
│       └── session_security_test.go
├── CI-WORKFLOW.md
├── DEVELOPER_GUIDE.md
├── IMPROVEMENTS.md
├── LOCAL_WORKING.md
├── New_Readme.md
├── README.md
├── SECURITY.md
├── TESTING.md
└── system_design.md
```

## Key Files Added For Decentralized Identity

- `internal/identity/*` implements DID generation, credential signing, and proof verification.
- `internal/cli/did.go`, `internal/cli/vc.go`, and `internal/cli/zk_proof.go` expose the new CLI commands.
- `internal/store/bbolt.go` and `internal/store/store.go` now manage `dids:<profile>` and `credentials:<profile>` buckets.
- `tests/integration/identity`, `tests/e2e/identity_workflow_test.go`, and `tests/security/identity_security_test.go` cover the new feature from multiple angles.
