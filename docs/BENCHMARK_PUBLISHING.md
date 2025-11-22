# Benchmark Publishing System

Automated benchmark publishing and regression-checking system for the password-vault-cli project.

## Overview

This system automatically runs performance benchmarks, detects regressions, and publishes results to GitHub Pages with organized navigation and version history.

### Key Features

- ✅ **Automated Execution**: Benchmarks run on every PR commit and main branch push
- ✅ **Smart Regression Detection**: Different strategies for PRs vs main branch
- ✅ **Version History**: Archives last 10 versions per context
- ✅ **Organized Structure**: Separate folders for master and pull requests
- ✅ **GitHub Pages**: Published with navigation indexes
- ✅ **PR Integration**: Automatic comments with benchmark links

---

## Architecture

### Directory Structure

```
previews/
├── master/
│   ├── latest/          # Current main branch benchmarks
│   └── old/
│       └── <timestamp>/ # Archived versions (keep last 10)
└── pull-requests/
    └── pr-N/
        ├── latest/      # Current PR benchmarks
        └── old/
            └── <timestamp>/ # Archived commits (keep last 10)
```

### Regression Strategy

| Context | Baseline | Action | Purpose |
|---------|----------|--------|---------|
| **PR Commits** | Previous commit in same PR | Informational | Track incremental changes |
| **Main Branch** | Current master baseline | **BLOCKS** | Protect main from regressions |

---

## Workflow Logic

### Pull Request Flow

1. **Run Benchmarks**: Execute on `./internal/crypto`, `./internal/vault`, `./tests`
2. **Stage Results**: Detect PR context, set path to `previews/pull-requests/pr-N/`
3. **Regression Check**: Compare vs previous commit (if exists)
   - ⚠️ Regression detected → Warning (informational only)
   - ✅ No regression → Pass
   - ℹ️ First run → Skip check
4. **Archive & Publish**:
   - Archive current `latest/` → `old/<timestamp>/`
   - Publish new results → `latest/`
   - Cleanup: Keep last 10 archives
5. **Generate Indexes**: Create navigation pages
6. **Deploy**: Push to `gh-pages` branch
7. **Comment**: Post PR comment with links and status

### Main Branch Flow

1. **Run Benchmarks**: Execute on configured packages
2. **Stage Results**: Set path to `previews/master/`
3. **Regression Check**: Compare vs current master baseline
   - ❌ Regression detected → **FAIL CI** (blocks)
   - ✅ No regression → Continue
   - ℹ️ First run → Skip check
4. **Archive & Publish**:
   - Archive current `latest/` → `old/<timestamp>/`
   - Publish new results → `latest/`
   - Cleanup: Keep last 10 archives
5. **Generate Indexes**: Create navigation pages
6. **Deploy**: Push to `gh-pages` branch

---

## Components

### Scripts

#### `benchmark-stage.sh`
**Purpose**: Stage benchmark results into proper directory structure

**Logic**:
- Detects PR vs main branch context
- Extracts PR number from GitHub event
- Sets target path: `previews/master/` or `previews/pull-requests/pr-N/`
- Creates staging directory with results
- Exports environment variables for downstream steps

**Environment Variables Exported**:
- `IS_MAIN_BRANCH`: `true` if push to main
- `IS_PR`: `true` if pull request
- `PR_NUMBER`: PR number (if applicable)
- `TARGET_PATH`: Target directory path
- `STAGING_DIR`: Staging directory path
- `TIMESTAMP`: Current timestamp for archiving

#### `benchmark-regression.sh`
**Purpose**: Perform regression analysis against appropriate baseline

**Logic**:
- Fetches `gh-pages` branch for baselines
- Selects baseline based on context:
  - **PR**: `previews/pull-requests/pr-N/latest/benchmark.txt`
  - **Main**: `previews/master/latest/benchmark.txt`
- Runs `benchstat` comparison
- Parses output for regressions (>10% slowdown)
- Exits with code 1 if regression found, 0 otherwise

**Configuration**:
- `REGRESSION_THRESHOLD`: Percentage threshold (default: 10%)

#### `benchmark-archive.sh`
**Purpose**: Archive current latest and publish new benchmarks

**Logic**:
- Clones `gh-pages` branch
- Archives current `latest/` to `old/<timestamp>/`
- Publishes new benchmarks to `latest/`
- Cleans up old archives (keeps last 10)
- Commits changes to `gh-pages`

**Configuration**:
- `MAX_OLD_ARCHIVES`: Number of archives to keep (default: 10)

#### `generate-index.sh`
**Purpose**: Generate HTML navigation indexes

**Generates**:
- `/index.html` - Root landing page
- `/previews/index.html` - Previews overview
- `/previews/master/index.html` - Master overview
- `/previews/master/old/index.html` - Master archive listing
- `/previews/pull-requests/index.html` - PR overview
- `/previews/pull-requests/pr-N/index.html` - Individual PR overview
- `/previews/pull-requests/pr-N/old/index.html` - PR archive listing

**Features**:
- Dark theme styling
- Breadcrumb navigation
- Archive listings sorted by date
- Responsive design

### CI Workflow

**Trigger**: Push to any branch, pull request

**Jobs**:
1. **Run Benchmarks**: Execute `go test -bench=.` on configured packages
2. **Stage Results**: Call `benchmark-stage.sh`
3. **Regression Check**: Call `benchmark-regression.sh` (continue-on-error for PRs)
4. **Block on Regression**: Fail CI if main branch regression detected
5. **Archive & Publish**: Call `benchmark-archive.sh`
6. **Generate Indexes**: Call `generate-index.sh`
7. **Deploy**: Use `peaceiris/actions-gh-pages` to deploy
8. **Comment**: Post PR comment with results (PRs only)

**Permissions Required**:
- `contents: write` - For gh-pages commits
- `pull-requests: write` - For PR comments

---

## Configuration

### Environment Variables

Set in `.github/workflows/ci.yml`:

```yaml
env:
  REGRESSION_THRESHOLD: '10'    # Percentage slowdown threshold
  MAX_OLD_ARCHIVES: '10'        # Number of archives to retain
```

### Benchmark Packages

Configure in workflow:

```yaml
go test -bench=. -benchmem -count=3 \
  ./internal/crypto \
  ./internal/vault \
  ./tests
```

---

## Usage

### For Developers

**During PR Development**:
1. Push commits to your PR branch
2. CI runs benchmarks automatically
3. Check PR comment for:
   - Link to benchmark results
   - Regression status (informational)
   - Links to compare with master
4. View results at: `https://<user>.github.io/<repo>/previews/pull-requests/pr-N/latest/`

**Before Merging**:
- Review benchmark trends in PR archive
- Compare with master baseline
- Address any significant regressions

**After Merging**:
- Main branch CI runs regression check
- If regression detected: CI fails, merge is blocked
- Fix regression and push again

### For Maintainers

**Monitoring**:
- Check master baseline regularly: `/previews/master/latest/`
- Review archive for performance trends: `/previews/master/old/`
- Monitor PR benchmarks for concerning patterns

**Adjusting Threshold**:
```yaml
# In .github/workflows/ci.yml
env:
  REGRESSION_THRESHOLD: '15'  # Increase to 15%
```

**Cleanup**:
```bash
# Remove old PR benchmarks manually
git checkout gh-pages
rm -rf previews/pull-requests/pr-OLD
git commit -m "cleanup: remove old PR benchmarks"
git push origin gh-pages
```

---

## Edge Cases

### First Run (No Baseline)

**Scenario**: No previous benchmarks exist

**Behavior**:
- Regression check skipped
- Results published to `latest/`
- No archive created
- CI passes

### No Benchmarks Found

**Scenario**: `go test -bench=.` produces no results

**Behavior**:
- `benchmark-stage.sh` fails with error
- CI fails
- No deployment
- Baselines unchanged

### Concurrent PRs

**Scenario**: Multiple PRs running simultaneously

**Behavior**:
- Each PR has isolated directory (`pr-1/`, `pr-2/`)
- No interference between PRs
- Each compares against own history
- Atomic deployments prevent conflicts

### Regression on First Main Push

**Scenario**: First push to main with slow benchmarks

**Behavior**:
- No baseline to compare against
- Regression check skipped
- Slow benchmarks become new baseline
- Future pushes will compare against this

**Mitigation**: Establish good baseline before enabling blocking

---

## Troubleshooting

### Regression Check Always Passes

**Cause**: Baseline doesn't exist or threshold too high

**Solution**:
1. Verify baseline exists: `curl https://<user>.github.io/<repo>/previews/master/latest/benchmark.txt`
2. Check threshold in workflow
3. Review `benchstat` output in CI logs

### Archives Not Created

**Cause**: First run or permissions issue

**Solution**:
1. Verify this isn't the first run
2. Check `GITHUB_TOKEN` permissions
3. Review `benchmark-archive.sh` logs

### Deployment Fails

**Cause**: GitHub Pages not enabled or permissions

**Solution**:
1. Enable Pages: Settings → Pages → Source: `gh-pages` branch
2. Verify `contents: write` permission
3. Check `peaceiris/actions-gh-pages` logs

### Benchmarks Not Running

**Cause**: Package paths incorrect or no benchmark tests

**Solution**:
1. Verify packages exist: `ls -la ./internal/crypto`
2. Check for benchmark tests: `grep -r "^func Benchmark" .`
3. Run locally: `go test -bench=. ./internal/crypto`

---

## URLs

### Production

- **Root**: `https://<username>.github.io/<repo>/`
- **Master Latest**: `https://<username>.github.io/<repo>/previews/master/latest/`
- **Master Archive**: `https://<username>.github.io/<repo>/previews/master/old/`
- **PR Latest**: `https://<username>.github.io/<repo>/previews/pull-requests/pr-N/latest/`
- **PR Archive**: `https://<username>.github.io/<repo>/previews/pull-requests/pr-N/old/`

### Local Testing

```bash
# Clone gh-pages branch
git clone --branch gh-pages https://github.com/<user>/<repo>.git gh-pages-local
cd gh-pages-local

# Serve locally
python3 -m http.server 8000

# Open in browser
open http://localhost:8000
```

---

## Future Enhancements

### Potential Improvements

1. **Visualization**: Add charts showing performance trends over time
2. **Comparison Tool**: Side-by-side benchmark comparison UI
3. **Notifications**: Slack/Discord alerts for regressions
4. **Historical Analysis**: Long-term performance trend analysis
5. **Benchmark Badges**: README badges showing latest performance
6. **Custom Baselines**: Allow manual baseline selection
7. **Multi-Platform**: Run benchmarks on different OS/architectures

### Contributing

To improve the benchmark system:

1. Test changes thoroughly (see [TESTING_BENCHMARKS.md](./TESTING_BENCHMARKS.md))
2. Update documentation
3. Submit PR with clear description
4. Ensure all test scenarios pass

---

## References

- **Testing Guide**: [TESTING_BENCHMARKS.md](./TESTING_BENCHMARKS.md)
- **Go Benchmarking**: https://pkg.go.dev/testing#hdr-Benchmarks
- **benchstat**: https://pkg.go.dev/golang.org/x/perf/cmd/benchstat
- **GitHub Pages**: https://docs.github.com/en/pages
- **peaceiris/actions-gh-pages**: https://github.com/peaceiris/actions-gh-pages
