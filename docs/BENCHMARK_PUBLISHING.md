# Benchmark Publishing System

## Overview

This document describes the automated benchmark publishing and regression-checking system implemented in the CI/CD pipeline.

## Architecture

The system consists of:
- **4 Helper Scripts** (`scripts/`) for modular operations
- **Updated CI Workflow** (`.github/workflows/ci.yml`) with integrated regression checking
- **GitHub Pages Deployment** with atomic updates

## Folder Structure

The published GitHub Pages site has the following structure:

```
https://<user>.github.io/password-vault-cli/
├── index.html                    # Root index with navigation
├── .nojekyll                     # Disable Jekyll processing
├── previews/                     # All preview benchmarks
│   ├── index.html               # Preview listing
│   ├── main/                    # Latest validated main branch baseline
│   │   └── benchmark.txt
│   ├── pr-123/                  # Pull request preview
│   │   └── benchmark.txt
│   └── feature-branch/          # Feature branch preview
│       └── benchmark.txt
├── latest/                       # Canonical baseline for regression checks
│   └── benchmark.txt
└── old/                          # Archived baselines
    ├── index.html               # Archive listing
    ├── 2025-11-21T15-30-00Z/   # Timestamped archive
    │   └── benchmark.txt
    └── 2025-11-21T14-00-00Z/
        └── benchmark.txt
```

## Workflow Logic

### 1. Benchmark Execution
- Runs on every push to `main`/`develop` and all pull requests
- Executes benchmarks on `./internal/crypto`, `./internal/vault`, `./tests`
- Generates `benchmark.txt` with results

### 2. Staging (`benchmark-stage.sh`)
- Computes safe folder name from PR number or branch name
- Creates `/tmp/pages/previews/<safe-name>/` structure
- Exports environment variables for downstream steps

### 3. Regression Check (`benchmark-regression.sh`)
- Fetches baselines from gh-pages branch
- Selects correct baseline:
  - **PR to main** → compare against `previews/main/`
  - **Push to main** → compare against previous `previews/main/`
  - **Other PRs/branches** → compare against `latest/`
- Runs `benchstat` comparison
- Fails if regression > 10% (configurable via `REGRESSION_THRESHOLD`)

### 4. Conditional Promotion (`benchmark-promote.sh`)
- **Only runs if regression check passes**
- Archives current `latest/` to `old/<timestamp>/`
- For main branch pushes:
  - Promotes to `previews/main/`
  - Promotes to `latest/`
- For other branches:
  - Promotes to `latest/` only
- Cleans up old archives (keeps last 10)

### 5. Index Generation (`generate-index.sh`)
- Creates `index.html` at root with navigation
- Creates `previews/index.html` listing all previews
- Creates `old/index.html` listing all archives
- Uses clean, modern HTML with embedded CSS

### 6. GitHub Pages Deployment
- Uses official GitHub Actions:
  - `actions/configure-pages@v4`
  - `actions/upload-pages-artifact@v3`
  - `actions/deploy-pages@v4`
- Atomic deployment (all-or-nothing)
- Concurrent deployments prevented via concurrency group

## Configuration

### Environment Variables

Set in `.github/workflows/ci.yml`:

```yaml
env:
  REGRESSION_THRESHOLD: '10'  # Percentage threshold for regression (default: 10%)
  MAX_OLD_ARCHIVES: '10'      # Number of old archives to retain (default: 10)
```

### Permissions

The benchmarks job requires:
```yaml
permissions:
  contents: write      # For git operations
  pages: write         # For Pages deployment
  id-token: write      # For OIDC authentication
```

## Usage

### For Developers

**Creating a Pull Request:**
1. Push your branch and create a PR
2. CI automatically runs benchmarks
3. Regression check compares against appropriate baseline
4. Preview published to `previews/pr-<number>/`
5. PR comment includes link to preview and regression status

**Merging to Main:**
1. After PR approval and merge
2. CI runs benchmarks on main branch
3. Regression check compares against previous `previews/main/`
4. If passed:
   - Old `latest/` archived to `old/<timestamp>/`
   - New benchmark promoted to `previews/main/` and `latest/`

### For Reviewers

**Viewing Benchmark Results:**
1. Visit `https://<user>.github.io/password-vault-cli/`
2. Navigate to:
   - **Latest** → Most recent validated baseline
   - **Previews** → All PR and branch previews
   - **Archives** → Historical baselines

**Interpreting Regression Status:**
- ✅ **No Regression** → Performance is stable or improved
- ⚠️ **Regression Detected** → Performance degraded >10%
  - Preview still published for inspection
  - Baselines NOT updated
  - CI job fails

## Edge Cases

### No Previous Baseline
- First run or after baseline deletion
- Regression check skipped
- Benchmark promoted directly to baselines
- Logged as "first run"

### Concurrent PRs
- Multiple PRs targeting main run simultaneously
- Each gets its own preview folder
- Last successful PR to complete overwrites baselines
- No race conditions due to atomic deployment

### Regression Failure
- Preview published to `previews/<name>/` for inspection
- Baselines (`latest/`, `previews/main/`) NOT updated
- CI job fails with clear error message
- PR comment indicates regression detected

### No Benchmarks Found
- CI job fails gracefully
- Error message: "No benchmarks found"
- Baselines NOT overwritten
- No promotion occurs

## Maintenance

### Cleaning Up Old Archives
Automatic cleanup runs during promotion:
- Keeps last 10 archives (configurable)
- Deletes older archives automatically
- Sorted by timestamp (newest first)

### Manual Cleanup
If needed, manually delete archives from gh-pages branch:
```bash
git checkout gh-pages
rm -rf old/<timestamp>
git add -A
git commit -m "Clean up old archives"
git push origin gh-pages
```

### Updating Regression Threshold
Edit `.github/workflows/ci.yml`:
```yaml
env:
  REGRESSION_THRESHOLD: '15'  # Increase to 15%
```

## Troubleshooting

### Pages Not Deploying
1. Check repository settings → Pages → Source is "GitHub Actions"
2. Verify permissions in workflow file
3. Check CI logs for deployment errors

### Regression Check Always Fails
1. Verify baseline exists on gh-pages branch
2. Check benchstat output in CI logs
3. Adjust `REGRESSION_THRESHOLD` if needed

### Preview Not Found (404)
1. Wait a few minutes for Pages deployment
2. Check CI logs for deployment success
3. Verify preview folder exists in `/tmp/pages/previews/`

### Scripts Not Executable
Run locally:
```bash
chmod +x scripts/*.sh
git add scripts/
git commit -m "Make scripts executable"
git push
```

## Benefits

✅ **Automated Regression Detection** → Prevents performance degradation  
✅ **Intelligent Baseline Selection** → Context-aware comparisons  
✅ **Historical Archiving** → Trend analysis over time  
✅ **User-Friendly Navigation** → Clean index pages  
✅ **Atomic Deployments** → No broken states  
✅ **Configurable Thresholds** → Flexible for different needs  
✅ **Edge Case Handling** → Robust error handling  

## Future Enhancements

- [ ] Benchmark visualization with charts
- [ ] Trend analysis dashboard
- [ ] Slack/Discord notifications for regressions
- [ ] Benchmark comparison across multiple PRs
- [ ] Custom HTML templates for benchmark pages
