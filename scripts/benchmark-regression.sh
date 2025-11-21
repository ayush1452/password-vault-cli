#!/bin/bash
set -euo pipefail

# benchmark-regression.sh
# Purpose: Perform regression analysis against the correct baseline
# Outputs: regression-report.txt and exit code (0=pass, 1=fail)

echo "=== Benchmark Regression Check ==="

# Validate inputs
if [ ! -f "benchmark.txt" ]; then
  echo "::error::benchmark.txt not found"
  exit 1
fi

# Get configuration
REGRESSION_THRESHOLD="${REGRESSION_THRESHOLD:-10}"
IS_PR_TO_MAIN="${IS_PR_TO_MAIN:-false}"
IS_MAIN_PUSH="${IS_MAIN_PUSH:-false}"
BASELINE_DIR="/tmp/gh-pages-baseline"

echo "Regression threshold: ${REGRESSION_THRESHOLD}%"
echo "Is PR to main: $IS_PR_TO_MAIN"
echo "Is main push: $IS_MAIN_PUSH"

# Install benchstat if not available
if ! command -v benchstat &> /dev/null; then
  echo "Installing benchstat..."
  go install golang.org/x/perf/cmd/benchstat@latest
  export PATH=$PATH:$(go env GOPATH)/bin
fi

# Fetch gh-pages branch for baseline comparison
echo "Fetching gh-pages branch for baselines..."
rm -rf "$BASELINE_DIR" || true

if git ls-remote --exit-code --heads origin gh-pages &> /dev/null; then
  echo "gh-pages branch exists, fetching..."
  git fetch origin gh-pages:gh-pages || true
  git worktree add "$BASELINE_DIR" gh-pages 2>/dev/null || {
    # Worktree might already exist, try to use it
    if [ -d "$BASELINE_DIR" ]; then
      echo "Using existing baseline directory"
    else
      echo "::warning::Could not fetch gh-pages branch"
      mkdir -p "$BASELINE_DIR"
    fi
  }
else
  echo "::notice::gh-pages branch does not exist yet (first run)"
  mkdir -p "$BASELINE_DIR"
fi

# Determine which baseline to use
BASELINE_FILE=""
BASELINE_NAME=""

if [ "$IS_PR_TO_MAIN" = "true" ] || [ "$IS_MAIN_PUSH" = "true" ]; then
  # Compare against previews/main/
  BASELINE_FILE="${BASELINE_DIR}/previews/main/benchmark.txt"
  BASELINE_NAME="previews/main"
  echo "Using baseline: previews/main/ (PR to main or push to main)"
else
  # Compare against latest/
  BASELINE_FILE="${BASELINE_DIR}/latest/benchmark.txt"
  BASELINE_NAME="latest"
  echo "Using baseline: latest/ (other PR or branch)"
fi

# Check if baseline exists
if [ ! -f "$BASELINE_FILE" ]; then
  echo "::notice::No baseline found at ${BASELINE_NAME}/benchmark.txt"
  echo "This appears to be the first run. Skipping regression check."
  echo "Baseline will be created after this run."
  
  # Create empty regression report
  echo "No baseline available - first run" > regression-report.txt
  
  # Cleanup
  rm -rf "$BASELINE_DIR" || true
  
  exit 0
fi

echo "Baseline file found: $BASELINE_FILE"

# Run benchstat comparison
echo "Running benchstat comparison..."
echo "Old (baseline): $BASELINE_FILE"
echo "New (current): benchmark.txt"

benchstat "$BASELINE_FILE" benchmark.txt | tee regression-report.txt || {
  echo "::warning::benchstat failed, but continuing"
  echo "benchstat comparison failed" > regression-report.txt
}

# Parse benchstat output for regressions
echo ""
echo "Analyzing regression results..."

# Look for performance regressions exceeding threshold
# benchstat output format: "old time" "new time" "delta"
# Example: "100ns ± 2%  120ns ± 3%  +20.00%"
# We want to catch lines with positive delta > threshold

REGRESSION_FOUND=false

# Extract delta percentages from benchstat output
# Look for lines with format: +XX.XX% or ~(+XX.XX%)
while IFS= read -r line; do
  # Skip header lines and non-benchmark lines
  if [[ ! "$line" =~ ^Benchmark ]]; then
    continue
  fi
  
  # Extract percentage change (look for +X.X% pattern)
  if [[ "$line" =~ \+([0-9]+\.?[0-9]*)% ]]; then
    DELTA="${BASH_REMATCH[1]}"
    
    # Compare with threshold (using bc for float comparison)
    if command -v bc &> /dev/null; then
      IS_REGRESSION=$(echo "$DELTA > $REGRESSION_THRESHOLD" | bc -l)
    else
      # Fallback to integer comparison if bc not available
      DELTA_INT=${DELTA%.*}
      IS_REGRESSION=$( [ "$DELTA_INT" -gt "$REGRESSION_THRESHOLD" ] && echo 1 || echo 0 )
    fi
    
    if [ "$IS_REGRESSION" = "1" ]; then
      echo "::error::Performance regression detected: +${DELTA}% (threshold: ${REGRESSION_THRESHOLD}%)"
      echo "Affected benchmark: $line"
      REGRESSION_FOUND=true
    fi
  fi
done < regression-report.txt

# Cleanup
rm -rf "$BASELINE_DIR" || true

# Exit based on regression status
if [ "$REGRESSION_FOUND" = "true" ]; then
  echo ""
  echo "❌ Regression check FAILED"
  echo "Performance degraded beyond acceptable threshold (${REGRESSION_THRESHOLD}%)"
  echo "Preview will be published for inspection, but baselines will NOT be updated."
  echo ""
  echo "Full regression report:"
  cat regression-report.txt
  exit 1
else
  echo ""
  echo "✅ Regression check PASSED"
  echo "No significant performance degradation detected."
  exit 0
fi
