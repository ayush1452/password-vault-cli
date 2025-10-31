# Local Working Guide

## Prerequisites
- **Go toolchain**: Ensure Go 1.21+ is installed and accessible on your PATH.
- **Vault binaries**: Build the CLI once per code change with `go build -o vault ./cmd/vault`.
- **Data location**: The vault database lives at `~/.local/share/vault/vault.db`. Session state is cached in `~/.local/share/vault/vault.db.session`.

## Initial Setup
- **Build**: `go build -o vault ./cmd/vault`
- **Create vault**: `./vault init`
  - When prompted, enter and confirm a strong master passphrase.
  - The command writes metadata (including KDF parameters and salt) into `vault.db`.

## Daily Workflow
- **Unlock**: `./vault unlock`
  - This derives the master key using metadata from `vault.db`.
  - On success, the session cache `vault.db.session` is written and future commands can reuse it.
- **Add entries**:
  - `./vault add <name> --username <user> --secret-prompt`
  - Secrets are encrypted with the master key and stored in BoltDB.
- **Update entries**:
  - `./vault update <name> --username <new-user> --url <https://...> --notes "..." --tags team1,prod --secret-prompt`
  - Flags mirror those on `add` (`--secret-file`, `--totp-seed`, etc.). If no flags are passed, the command walks you through interactive prompts.
- **List entries**:
  - `./vault list`
  - Use filters such as `--search <text>` or `--tags work,git` as needed.
- **Retrieve entry**:
  - `./vault get <name> [--field username|secret|url|notes]`
  - Add `--show` to print secrets or rely on clipboard copy (default for secrets).
- **Update / delete**:
  - `./vault update <name>` and follow prompts.
  - `./vault delete <name>` removes an entry (honors `confirm_destructive` config).

## Session Persistence
- **Auto-lock TTL**: Default is 1 hour (`unlock --ttl <duration>` overrides).
- **Reusing sessions**: As long as `vault.db.session` exists and TTL has not expired, subsequent commands skip passphrase prompts.
- **After reboot**:
  - Delete the stale session file if desired: `rm ~/.local/share/vault/vault.db.session`
  - Run `./vault unlock` again to regenerate the session cache.

## Troubleshooting
- **Vault locked errors**: Run `./vault unlock` or ensure the session file exists within TTL.
- **Decryption failed**: Regenerate the vault or ensure the master passphrase matches the metadata. Removing `vault.db.session` can help clear corrupted cache state.
- **Start fresh**:
  - Remove database and session: `rm ~/.local/share/vault/vault.db ~/.local/share/vault/vault.db.lock ~/.local/share/vault/vault.db.session`
  - Rerun `./vault init` followed by `./vault unlock`.
- **Update workflow smoke test**:
  - `printf 'newSecret\n' | script -q /dev/null ./vault update <name> --username new.user --url https://new.example.com --notes "Rotated" --tags newTag --secret-prompt`
  - `./vault get <name> --show`
  - `./vault list`
  - Remove the session file (`rm ~/.local/share/vault/vault.db.session`) and run `./vault unlock` → `./vault list` to confirm persistence.

## Helpful Tips
- **Close session explicitly**: `./vault lock` zeroizes the in-memory key and removes the session file.
- **Config tweaks**: Use `./vault config set auto_lock_ttl 30m` or other keys defined in `docs/USAGE.md` to customize behavior.
- **Security hygiene**: Run `./vault doctor` periodically to verify integrity and review audit logs via `./vault audit list`.
