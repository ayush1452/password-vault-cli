# Binary Artifact Handbook

New to the project and wondering what the extra files at the repository root are (`vault`, `cli`, `crypto`, `storage`)? This guide explains **what each binary does**, **how it is built**, **when to use it**, and **what happens if you delete it**.

> All binaries sit in the project root (next to `go.mod`) and are excluded from Git by `.gitignore`. Deleting them never harms source code—you can rebuild at any time.

## 1. Cheat Sheet

| Binary | Build Command | What it does | When to run it | Safe to delete? |
| --- | --- | --- | --- | --- |
| `vault` | `go build -o vault ./cmd/vault` | The primary CLI you use every day to manage the password vault. | Whenever you interact with the real vault (init, add, get, list, etc.). | ✅ Yes. Rebuild when needed. |
| `cli` | `go build -o cli ./examples/cli` | A scripted demo that showcases basic CLI flows using a temporary vault. | When demoing the app to someone or testing your environment quickly. | ✅ Yes. Rebuild for future demos. |
| `crypto` | `go build -o crypto ./examples/crypto` | A sandbox executable for experimenting with Argon2id, AES-GCM, etc. | When you want to benchmark or inspect the cryptographic primitives. | ✅ Yes. Delete after experimentation. |
| `storage` | `go build -o storage ./examples/storage` | Exercises the BoltDB storage layer without invoking the full CLI. | When testing storage operations or debugging DB behavior. | ✅ Yes. Regenerate on demand. |


---

## 2. Understanding the Shared Traits

- Built for **your** OS/architecture; binaries from Windows/macOS/Linux are not interchangeable.
- Deleting a binary simply removes the compiled executable—not the Go source.
- Rebuild anytime with the associated `go build` command (Go 1.21+ recommended).
- To see command-line flags or versions, run `./<binary> --help` or `./<binary> --version` (where supported).


---

## 3. Deep Dive by Binary

### 3.1 `vault` — Main CLI

- **Purpose**: This is the application you install or ship. It handles initialization, unlocking, entry CRUD, audits, etc.
- **Source code**: `cmd/vault/main.go` plus the `internal/` packages.
- **Build it**:

```bash
go build -o vault ./cmd/vault
```

- **Example day-to-day commands**:

```bash
./vault init
./vault unlock --ttl 1h
./vault status --json
./vault list --profile personal --search github
./vault get github --copy
./vault lock
```

- **Distribution tip**: `sudo mv vault /usr/local/bin/` (or add to `PATH`) for system-wide use.
- **If deleted**: Only the executable disappears. Recreate it with the same build command.


### 3.2 `cli` — Scripted Demo Runner

- **Purpose**: Automates a walkthrough of the CLI for tutorials or quick smoke checks. It spins up a temp directory, builds a throwaway vault, and runs a sequence of commands.
- **Source code**: `examples/cli/main.go`.
- **Build it**:

```bash
go build -o cli ./examples/cli
```

- **What running it looks like**:

```bash
./cli
# Sample output (trimmed)
Password Vault CLI - Complete Demo
Demo vault: /tmp/vault_cli_demo_123/demo.vault
1. Building vault binary...
2. Display version
   Command: vault --version
   Output: vault version 1.0.0
...
```

- **When to use**: Show teammates how the CLI behaves, verify integrations after refactors, or generate automated tutorial output.
- **If deleted**: No core functionality lost. Rebuild if you need the demo again.


### 3.3 `crypto` — Cryptography Playground

- **Purpose**: Lets developers experiment with low-level cryptographic routines without touching real vault data.
- **Source code**: `examples/crypto/main.go` (may include benchmarks, comparisons, etc.).
- **Build it**:

```bash
go build -o crypto ./examples/crypto
```

- **Possible usage patterns**:

```bash
./crypto --benchmark-kdf
./crypto --inspect-envelope demo.envelope
./crypto --compare-engines
```

*(Flags depend on how the example is implemented; consult the file before running.)*

- **If deleted**: Merely removes the exploratory binary; the main application remains unaffected.


### 3.4 `storage` — Storage-Layer Exerciser

- **Purpose**: Focuses on BoltDB interactions (creating, listing, migrating vault data) outside the full CLI context.
- **Source code**: `examples/storage/main.go`.
- **Build it**:

```bash
go build -o storage ./examples/storage
```

- **Typical commands**:

```bash
./storage --init demo.db
./storage --insert demo.db default github
./storage --list demo.db
./storage --stats demo.db
```

*(Arguments depend on the example implementation—check the source for available flags.)*

- **If deleted**: Only removes the helper. Rebuild whenever you need to inspect storage behavior again.


---

## 4. Frequently Asked Questions

- **Do I commit these binaries?** No. They are intentionally ignored by Git.
- **Can I have all binaries present simultaneously?** Absolutely. They serve different niches and don’t conflict.
- **How do I clean up?** Just delete the binaries (`rm ./vault ./cli ...`). Source code stays intact.
- **What if a binary feels outdated?** Re-run the respective `go build` command to refresh it from the latest source.


---

With this handbook, a newcomer can immediately identify each binary, understand its role, reproduce it, and decide whether it should live on their machine. If you add more sample executables later, follow the same pattern: note the source file, build command, purpose, example usage, and deletion impact.
