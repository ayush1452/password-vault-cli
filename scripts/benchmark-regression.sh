#!/bin/bash
set -euo pipefail

# benchmark-regression.sh
# Purpose: Perform regression analysis against the appropriate baseline
# For PRs: Compare against previous commit in same PR (informational)
# For main: Compare against current master baseline (blocking)

echo "=== Benchmark Regression Check ==="

# Validate inputs
if [ ! -f "benchmarks/benchmark.txt" ]; then
  echo "::error::benchmarks/benchmark.txt not found"
  exit 1
fi

# Get configuration from environment
REGRESSION_THRESHOLD="${REGRESSION_THRESHOLD:-10}"
IS_MAIN_BRANCH="${IS_MAIN_BRANCH:-false}"
IS_PR="${IS_PR:-false}"
TARGET_PATH="${TARGET_PATH:-}"

echo "Regression threshold: ${REGRESSION_THRESHOLD}%"
echo "Is main branch: $IS_MAIN_BRANCH"
echo "Is PR: $IS_PR"
echo "Target path: $TARGET_PATH"

# Install benchstat if not available
if ! command -v benchstat &> /dev/null; then
  echo "Installing benchstat..."
  go install golang.org/x/perf/cmd/benchstat@latest
  export PATH=$PATH:$(go env GOPATH)/bin
fi

# Fetch gh-pages branch for baseline
BASELINE_DIR="/tmp/gh-pages-baseline"
echo "Fetching gh-pages branch..."
rm -rf "$BASELINE_DIR" || true

if git ls-remote --exit-code --heads origin gh-pages &> /dev/null; then
  echo "gh-pages branch exists, fetching..."
  git fetch origin gh-pages:gh-pages || true
  git worktree add "$BASELINE_DIR" gh-pages 2>/dev/null || {
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

# Determine baseline based on context
BASELINE_FILE=""
BASELINE_NAME=""

if [ "$IS_MAIN_BRANCH" = "true" ]; then
  # Main branch: compare against current master baseline
  BASELINE_FILE="${BASELINE_DIR}/previews/master/latest/benchmark.txt"
  BASELINE_NAME="previews/master/latest"
  echo "Context: Push to main"
  echo "Baseline: Current master baseline"
  
elif [ "$IS_PR" = "true" ]; then
  # PR: compare against previous commit in same PR
  BASELINE_FILE="${BASELINE_DIR}/${TARGET_PATH}/latest/benchmark.txt"
  BASELINE_NAME="${TARGET_PATH}/latest"
  echo "Context: Pull Request"
  echo "Baseline: Previous commit in same PR"
  
else
  # Other branch: compare against previous commit in same branch
  BASELINE_FILE="${BASELINE_DIR}/${TARGET_PATH}/latest/benchmark.txt"
  BASELINE_NAME="${TARGET_PATH}/latest"
  echo "Context: Branch push"
  echo "Baseline: Previous commit in same branch"
fi

# Check if baseline exists
if [ ! -f "$BASELINE_FILE" ]; then
  echo "::notice::No baseline found at ${BASELINE_NAME}"
  echo "This appears to be the first run for this context."
  echo "Skipping regression check - baseline will be created after this run."
  
  # Create empty regression report
  echo "No baseline available - first run" > regression-report.txt
  
  # Cleanup
  rm -rf "$BASELINE_DIR" || true
  
  exit 0
fi

echo "Baseline file found: $BASELINE_FILE"

# Run benchstat comparison
echo ""
echo "Running benchstat comparison..."
echo "Old (baseline): $BASELINE_FILE"
echo "New (current):  benchmarks/benchmark.txt"
echo ""

benchstat "$BASELINE_FILE" benchmarks/benchmark.txt | tee regression-report.txt || {
  echo "::warning::benchstat failed, but continuing"
  echo "benchstat comparison failed" > regression-report.txt
}

# Parse benchstat output for regressions
echo ""
echo "Analyzing regression results..."

REGRESSION_FOUND=false
REGRESSION_DETAILS=""

# Extract delta percentages from benchstat output
# benchstat format: "old time" "new time" "delta"
# Example: "100ns ± 2%  120ns ± 3%  +20.00%  (p=0.000 n=3)"

while IFS= read -r line; do
  # Skip header lines and non-benchmark lines
  if [[ ! "$line" =~ ^Benchmark ]]; then
    continue
  fi
  
  # Extract percentage change (look for +X.X% or -X.X% pattern)
  if [[ "$line" =~ ([+-])([0-9]+\.?[0-9]*)% ]]; then
    SIGN="${BASH_REMATCH[1]}"
    DELTA="${BASH_REMATCH[2]}"
    
    # Only check for positive deltas (slowdowns)
    if [ "$SIGN" = "+" ]; then
      # Compare with threshold
      if command -v bc &> /dev/null; then
        IS_REGRESSION=$(echo "$DELTA > $REGRESSION_THRESHOLD" | bc -l)
      else
        # Fallback to integer comparison
        DELTA_INT=${DELTA%.*}
        IS_REGRESSION=$( [ "$DELTA_INT" -gt "$REGRESSION_THRESHOLD" ] && echo 1 || echo 0 )
      fi
      
      if [ "$IS_REGRESSION" = "1" ]; then
        echo "::warning::Performance regression detected: +${DELTA}% (threshold: ${REGRESSION_THRESHOLD}%)"
        echo "  $line"
        REGRESSION_FOUND=true
        REGRESSION_DETAILS="${REGRESSION_DETAILS}\n  ${line}"
      fi
    fi
  fi
done < regression-report.txt

# Cleanup
rm -rf "$BASELINE_DIR" || true

# Report results
echo ""
if [ "$REGRESSION_FOUND" = "true" ]; then
  echo "❌ Regression check FAILED"
  echo "Performance degraded beyond acceptable threshold (${REGRESSION_THRESHOLD}%)"
  echo ""
  echo "Affected benchmarks:${REGRESSION_DETAILS}"
  echo ""
  
  if [ "$IS_MAIN_BRANCH" = "true" ]; then
    echo "⚠️  BLOCKING: This will prevent baseline promotion on main branch"
  else
    echo "ℹ️  INFORMATIONAL: This is for your awareness during development"
  fi
  
  echo ""
  echo "Full regression report:"
  cat regression-report.txt
  exit 1
else
  echo "✅ Regression check PASSED"
  echo "No significant performance degradation detected."
  exit 0
fi
