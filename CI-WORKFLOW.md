# CI/CD Pipeline (GitHub Actions)

This project uses GitHub Actions to run a multi-stage CI pipeline that enforces code quality, builds artifacts, runs tests and benchmarks, scans for security issues and notifies maintainers on failures.

**Workflow file location:** `.github/workflows/ci.yml`

---

## Summary / Quick view



**Trigger events**

* `push` to `main` or `develop` (only when Go sources or module files change)
* `pull_request` targeting `main` or `develop`
* `workflow_dispatch` for manual runs

**High-level stages (visual):**

```
                                   ┌────────────────────┐
                                   │    code-quality    │
                                   │ (lint, vet, gosec) │
                                   └─────────┬──────────┘
                                             │
               ┌─────────────────────────────┴────────────────────────────┐
               ▼                                                          ▼
      ┌────────────────────┐                                  ┌────────────────────┐
      │       build        │                                  │    security-scan   │
      │ (Linux / macOS /   │                                  │       (Trivy)      │
      │   Windows matrix)  │                                  └───────────┬────────┘
      └─────────┬──────────┘                                              │
                │                                                         │
                ▼                                                         ▼
      ┌────────────────────┐                                        (artifact SARIF)
      │        test        │                                              │
      │ (unit / race / cov)│                                              │
      └─────────┬──────────┘                                              │
                │                                                         │
      ┌─────────┴─────--───┐                                              │
      ▼                    ▼                                              │
┌────────────────────┐  ┌────────────────────┐                            │
│      coverage      │  │     benchmarks     │                            │
│      (Codecov)     │  │  (go test -bench)  │                            │
└────────────────────┘  └─────────┬──────────┘                            │
                                  │                                       │
                                  │                                       │
                     ┌────────────┴──────────────┐                        │
                     ▼                           ▼                        │
          ┌──────────────────────┐     ┌────────────────────┐             │ 
          │   performance-check  │     │       notify       │             |
          │      (benchstat)     │     │  (GitHub Issue)    │◄────────────┘      
          └──────────────────────┘     └────────────────────┘



Parallel / Independent:
  ┌───────────────────────────┐
  │     CodeQL Analysis       │
  └───────────────────────────┘
```

**Critical path:** `code-quality` → `build` → `test` → `benchmarks` → `performance-check`
**Parallel jobs:** `codeql-analysis` (runs independently); `security-scan` runs after `code-quality` but in parallel to `build`.

---

# Full job-by-job explanation (what, why, how)

> For each job below I document: purpose, steps, artifacts, where to look on failure and common fixes.

---

## Global environment & defaults (top of workflow)

**Environment variables defined in workflow:**

```yaml
env:
  GO_VERSION: '1.24.10'
  GOTOOLCHAIN: local
  BINARY_NAME: 'vault'
  CACHE_VERSION: 'v1'
  COVERAGE_FILE: 'coverage.out'
  BENCHMARK_FILE: 'benchmark.out'
  LINT_TIMEOUT: '5m'
  GOLANGCI_LINT_VERSION: 'v2.3.1'
  GOSEC_VERSION: 'v2.19.0'
```

**Notes & recommendations**

* These are the defaults available to all jobs. If you need different Go versions for a matrix, override `GO_VERSION` per job.
* `GOLANGCI_LINT_VERSION` in your workflow is provided as an env var used in the install script. Verify that the tag you expect exists (many golangci-lint releases look like `v1.64.0` — confirm the value you want).
* Keep `GOSEC_VERSION` pinned for reproducible scans.

---

## Job: `code-quality` (Matrix: formatting & linting / static analysis / security-scan)

**Purpose:** enforce formatting, linting and static checks before letting code progress.

**Key steps**

1. checkout code (full history)
2. setup-go (`actions/setup-go@v4`) using `GO_VERSION`
3. cache Go module files (~ `~/go/pkg/mod`, `~/.cache/go-build`)
4. install developer tooling:

   * `goimports`, `gofumpt`, `staticcheck`, `gosec`, `golangci-lint`, `errcheck`
5. run format checks (gofmt/goimports/gofumpt)
6. run `golangci-lint` and `staticcheck`
7. run `errcheck` (error-handling checks)

**Artifacts & outputs**

* Linter output logs (visible in Actions UI)
* `coverage-report` artifact uploaded only from the `format-lint` matrix run

**Common failures & fixes**

* **`errcheck` flags ignored `_ = fn()`**

  * Cause: you run `errcheck -blank` and code intentionally uses `_ = ...` to ignore errors.
  * Fix: either handle the error in code or add `// nolint:errcheck` with a comment where ignoring is intentional; or remove `-blank` from `errcheck` invocation in the CI to relax the check.
* **golangci-lint misconfiguration or unknown version**

  * Cause: mismatched `GOLANGCI_LINT_VERSION` or install script failure.
  * Fix: pin to an official release, e.g. `v1.64.0` (confirm on the project). Use `golangci-lint --version` to debug.
* **formatting failures**

  * Run locally: `gofmt -s -l .`, `goimports -w .`, `gofumpt -w .` or run `make fmt` if available.

**Security-specific checks (matrix includes `security-scan`)**

* `gosec` is installed and run — the job returns SARIF data if failures found (helpful for GitHub Security tab).

---

## Job: `build`

**Purpose:** cross-compile the CLI binary for multiple platforms defined in the matrix (linux, macos, windows).

**Key steps**

1. checkout repo
2. set up Go (same version)
3. get dependencies (`go mod download`)
4. build binary with `-ldflags` embedding metadata:

   * `Version`, `BuildDate`, `CommitHash`, `BuildOS`
5. upload compiled artifacts (`actions/upload-artifact@v4`)

**Artifacts**

* `bin/vault-*-*` artifacts per platform

**Common failures & fixes**

* **Missing dependencies**: `go mod download` fails — run `go mod tidy` locally to fix; ensure the repo contains `go.sum`.
* **CGO issues or cross-compile problems**: Set `CGO_ENABLED=0` when you don't need cgo (already in the workflow).
* **Build breaks on macOS/Windows matrix**: check the `platform` values and use correct `GOOS/GOARCH` mapping. If building for Mac Apple Silicon, include `darwin/arm64` matrix entry if desired.

**Recommendations**

* Consider adding `strip`/`upx` step only for release builds to reduce binary size (optional).
* Use deterministic artifacts names that include `commit` or `tag` for release automation.

---

## Job: `test`

**Purpose:** run unit/integration tests, with `-race` detection and coverage measurement.

**Key steps**

1. checkout code
2. setup go for the matrix version(s)
3. `go mod tidy` & `go mod download`
4. `go build` then `go test -race -coverprofile=coverage.out -covermode=atomic ./...`
5. verify built binary by running `./vault --version` (smoke check)

**Artifacts**

* `coverage.out` (uploaded by the `coverage` job via artifact)

**Common failures & fixes**

* **Race detector failures**: your tests access shared state unsafely. Fix by isolating tests and adding `sync.Mutex` locks or context scoping.
* **Test flakes**: increase test timeouts or add retries for network-dependent tests (or mark as integration and skip in CI).
* **Missing test data**: ensure test resources are committed or mocked.

**Recommendations**

* Split long integration tests into a separate job (optionally guarded by a label or manual trigger).
* If tests require secrets (e.g., cloud tests), run them only on a protected branch or with ephemeral credentials.

---

## Job: `codeql-analysis`

**Purpose:** perform CodeQL security analysis of the repo.

**Key steps**

* init CodeQL with `github/codeql-action/init@v3` and run `github/codeql-action/analyze@v3`.
* uses queries: `security-extended`, `security-and-quality`

**Notes**

* Runs in parallel to other jobs to reduce wall time.
* Outputs appear under the Security tab in GitHub if configured.

**Common failures & fixes**

* **Unexpected warnings**: review findings in the Security tab. Not every alert is exploitable — triage and mark false positives if needed.
* **Analysis failures**: ensure `go.mod` loads and `go` version matches your codebase.

---

## Job: `coverage`

**Purpose:** upload `coverage.out` to Codecov for reporting.

**Key steps**

* download coverage artifact from `test`
* use `codecov/codecov-action@v3` to upload

**Secrets**

* `CODECOV_TOKEN` (if uploading for private repos or if configured to require a token). If not required, `fail_ci_if_error: false` is set.

**Common failures & fixes**

* **Upload fails**: check `CODECOV_TOKEN`, file path (coverage file present), and network issues.
* Use `go tool cover -func=coverage.out` locally to validate the file format.

---

## Job: `benchmarks`

**Purpose:** run benchmarks and record results for regression checks.

**Key steps**

* run `go test -bench=. -benchmem -count=3 ./... | tee benchmark.txt`
* upload results to `benchmark-action/github-action-benchmark@v1` (auto push configured)

**Common failures & fixes**

* **Noisy CI environment** → benchmark variance. Fix by:

  * Increase `-count` (e.g., 5)
  * Run benchmarks in larger runners or dedicated performance runners
  * Run multiple repeats and consider statistical tests

**Recommendations**

* Use a dedicated performance runner (self-hosted) if you need stable results.
* Make the threshold conservative to avoid false positives.

---

## Job: `performance-check`

**Purpose:** compare current benchmarks against historical baseline and fail on meaningful regressions.

**Key details**

* downloads previous benchmark data (artifact) and compares via `benchstat`
* fails pipeline on regression over threshold (configured to `10%` in your script)

**Common failures & fixes**

* **No baseline found**: store baseline on first successful run, or accept that first run will just create baseline.
* **False positives**: widen the threshold or require multiple failing runs before hard-failing.

---

## Job: `security-scan` (Trivy)

**Purpose:** filesystem vulnerability scan with Trivy and upload SARIF to the Security tab.

**Key steps**

* run `aquasecurity/trivy-action@master`
* produce `trivy-results.sarif`
* upload SARIF file with `github/codeql-action/upload-sarif@v3`
* artifacts: `trivy-results.sarif`

**Common failures & fixes**

* **Trivy DB download timeouts**: add caching for `~/.cache/trivy` or increase job timeout.
* **Long scan times**: scoping scan paths or using `--skip-dirs` helps.
* **False positives**: triage or pin Trivy version and update DB regularly.

---

## Job: `notify` (Pipeline Failure Notification)

**Purpose:** create a GitHub issue automatically when pipeline fails on pushes (not PRs).

**Key points**

* Only runs when `failure()` and `github.event_name == 'push'`
* Uses `actions/github-script@v6` to create a GitHub issue that contains summary and links to the run.

**Common pitfalls & fixes**

* **Duplicates**: you might get duplicate CI issue creation on repeated failures. Consider querying for existing open CI issues and append a comment instead of creating a new issue.
* **Permission issues**: ensure the default `GITHUB_TOKEN` has the necessary `issues` permissions (it usually does).

---

# Where artifacts and logs live

* **Build artifacts**: under the run > `Artifacts` (`bin/*` files). The `build` job uploads them.
* **Coverage**: uploaded as `coverage-report` artifact from `code-quality` (only the `format-lint` matrix run currently).
* **Benchmark**: `benchmark.txt` is used by benchmark action and stored as artifacts in that action.
* **Security SARIFs**: `security-scan.sarif` (gosec) and `trivy-results.sarif` (Trivy) are added to the Security tab in GitHub and also uploaded as artifacts.
* **Job logs**: accessible from the Actions run page (click the job > steps).

---

# Wire-up / triggers and linking

* **Workflow file link:** `.github/workflows/ci.yml` in this repo.
* **Security tab SARIF links:** SARIF uploads produce entries in GitHub Security > Code scanning.
* **Failed run link:** each created issue includes a link to the workflow run:

  ```
  https://github.com/<owner>/<repo>/actions/runs/<runId>
  ```
* **Notifications and Slack:** not configured by default. You can add a Slack notification step or use an external alerting action.

---

# Troubleshooting & common errors (practical recipes)

### 1) `errcheck` failing on `_ = fn()`

* **Fix A (CI quick):** remove `-blank` from errcheck flags in `ci.yml`:

  ```diff
  - errcheck -blank -asserts -ignorepkg=bytes,io/ioutil,os -ignoretests ./...
  + errcheck -asserts -ignorepkg=bytes,io/ioutil,os -ignoretests ./...
  ```
* **Fix B (code):** explicitly handle or intentionally annotate with `// nolint:errcheck` and an explanatory comment.

### 2) `golangci-lint` install or version issues

* Ensure your `GOLANGCI_LINT_VERSION` matches an actual release tag (eg. `v1.64.8`). Use `golangci-lint --version` to verify in logs. If install script fails, pin to a specific download URL or use `actions/setup-go` with `go install` approach.

### 3) Cache misses or slow `go mod download`

* Use stable cache key format:

  ```yaml
  key: ${{ runner.os }}-go-${{ env.GO_VERSION }}-${{ hashFiles('**/go.sum') }}
  ```
* If cache invalidation occurs often, ensure `go.sum` changes only when dependencies change.

### 4) Codecov upload failing

* Ensure `CODECOV_TOKEN` secret is set (if required). Use `fail_ci_if_error: false` to prevent pipeline from failing while debugging.

### 5) Benchmark flakiness

* Increase iterations `-count=5` and use benchstat rules with tolerance.
* Consider running benchmarks on a dedicated self-hosted runner.

### 6) Trivy timeouts

* Add caching for trivy DB:

  ```yaml
  - uses: actions/cache@v3
    with:
      path: ~/.cache/trivy
      key: trivy-db-${{ github.ref_name }}-${{ github.run_id }}
  ```
* Increase `timeout-minutes` on the job.

---

# Security & secrets

* **Required repository secrets**:

  * `CODECOV_TOKEN` (if used)
  * any cloud credentials for integration tests (prefer ephemeral and use `environment` protection rules)
* **GITHUB_TOKEN**: used for issue creation and artifact interactions (default token scope is generally fine).
* **Don’t** store long-lived or production secrets in plain text in the workflow.

---

# Recommended workflow improvements (small practical edits)

1. **Fix errcheck noise** (recommended quick unblock):
   Replace `errcheck -blank ...` with `errcheck -asserts -ignorepkg=bytes,io/ioutil,os -ignoretests ./...` (remove `-blank`).

2. **Pin golangci-lint to a stable release**: validate `GOLANGCI_LINT_VERSION`. Example:

   ```yaml
   GOLANGCI_LINT_VERSION: 'v1.64.8'
   ```

   and modify install script accordingly.

3. **Cache golangci-lint and other binaries** to avoid repeatedly downloading tools:

   * Either install via `go install` where possible, or cache `$(go env GOPATH)/bin`.

4. **Make coverage artifact unconditional**: upload coverage even if lint step fails (helps post-mortem).

5. **Add retry wrapper** for flaky network steps (e.g., `go get`, `go mod download`, `gosec` DB updates).

6. **Add `paths-ignore`** for CI triggers if non-code docs change (already limited to `**.go` and go.mod files — good).

7. **Add annotations for false positives**: standardize `// nolint:errcheck // reason:` comments.

---

# How to run and debug locally

**Run lints locally**

```bash
# install the tools locally
go install honnef.co/go/tools/cmd/staticcheck@latest
go install github.com/kisielk/errcheck@latest
golangci-lint run --timeout=5m ./...
```

**Run tests locally**

```bash
go test -v -race ./...
```

**Run benchmarks locally**

```bash
go test -bench=. -benchmem -count=3 ./...
```

**Run a single GitHub Actions job locally (approx)**

* Use `act` (third-party tool) or a Docker-based runner. See `./scripts/ci-local.sh` if you add one to the repo.

---

# Versioning & update policy (recommended)

* **Go**: pinned to `1.24.10` in env. Update policy: Monthly review and upgrade major versions on a quarterly cadence.
* **gosec**: `v2.19.0` (pinned in `env`) — update monthly or as needed for critical fixes.
* **golangci-lint**: pin to a specific `v1.x` release. Update monthly.
* **CodeQL**: action is maintained by GitHub; keep action version `@v3`.
* **Trivy**: pin to a release and refresh DB periodically.

---

# Metrics & alerts (what to watch)

* **Pipeline duration** — target < 30 minutes
* **Test coverage** — track via Codecov; alert on drop > 5%
* **Benchmark regression** — alert on >10% regression (configurable)
* **New security vulnerabilities** — triage immediately; high & critical first
* **Build failure rate** — investigate if > 20%

---

# Maintenance checklist

* Monthly: update linter & scanner versions, refresh Trivy DB
* Quarterly: update Go minor/patch versions and validate all jobs
* Annually: review entire CI design and tooling

---

# Appendices

## Example: small, exact `errcheck` CI fix (copy-paste)

Replace in `.github/workflows/ci.yml`:

```diff
- errcheck -blank -asserts -ignorepkg=bytes,io/ioutil,os -ignoretests ./...
+ # Do not enforce blank-assignment checks in CI; handle those in code or via nolint
+ errcheck -asserts -ignorepkg=bytes,io/ioutil,os -ignoretests ./...
```

## Example: check & pin `golangci-lint` install

In the install step change to explicit install (safer):

```yaml
- name: Install golangci-lint
  run: |
    curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.64.8
```

*(Replace `v1.64.8` with your chosen pinned version.)*

---

# Closing / Next steps I can do for you

I can:

* Generate a **ready-to-commit README snippet** that exactly replaces your existing CI section (I prepared one above that you can drop in).
* Produce an updated **`.github/workflows/ci.yml`** with the suggested fixes (errcheck change, pinned golangci-lint, caching improvements, improved matrix for `darwin/arm64`) and return a `git diff` you can apply.
* Create `./scripts/ci-local.sh` to reproduce CI steps locally (helpful for debugging).
* Add a small workflow change to prevent duplicate `notify` issues (check for existing open CI issues before creating).

Which of these would you like me to produce now? I can generate the updated workflow file (diff) or the exact README markdown ready to paste — pick one and I’ll produce it.
