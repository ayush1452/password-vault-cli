#!/bin/bash
set -euo pipefail

# benchmark-stage.sh
# Purpose: Stage benchmark results into the proper folder structure
# New structure: previews/master/ or previews/pull-requests/pr-N/
# Outputs: /tmp/pages/ directory with staged content and environment variables

echo "=== Benchmark Staging Script ==="

# Validate inputs
if [ ! -f "benchmarks/benchmark.txt" ]; then
  echo "::error::benchmarks/benchmark.txt not found. Cannot stage benchmarks."
  exit 1
fi

# Check if benchmark.txt has actual results
if ! grep -q "^Benchmark" benchmarks/benchmark.txt; then
  echo "::error::No benchmarks found in benchmark.txt. Cannot stage empty results."
  exit 1
fi

# Determine context
EVENT="${GITHUB_EVENT_NAME:-unknown}"
REF="${GITHUB_REF:-}"
BASE_REF="${GITHUB_BASE_REF:-}"

echo "Event: $EVENT"
echo "Ref: $REF"
echo "Base Ref: $BASE_REF"

# Initialize variables
IS_MAIN_BRANCH="false"
IS_PR="false"
PR_NUMBER=""
TARGET_PATH=""

# Determine target path based on context
if [ "$EVENT" = "pull_request" ]; then
  IS_PR="true"
  
  # Extract PR number from event
  if [ -f "$GITHUB_EVENT_PATH" ]; then
    PR_NUMBER=$(jq -r '.number // ""' < "$GITHUB_EVENT_PATH")
    if [ -n "$PR_NUMBER" ]; then
      TARGET_PATH="previews/pull-requests/pr-${PR_NUMBER}"
      echo "Pull Request #${PR_NUMBER}"
    else
      echo "::error::Could not extract PR number from event"
      exit 1
    fi
  else
    echo "::error::GITHUB_EVENT_PATH not found"
    exit 1
  fi
  
elif [ "$EVENT" = "push" ]; then
  if [[ "$REF" == "refs/heads/main" ]] || [[ "$REF" == "refs/heads/master" ]]; then
    IS_MAIN_BRANCH="true"
    TARGET_PATH="previews/master"
    echo "Push to main branch"
  else
    # Non-main branch push (treat like a PR without number)
    BRANCH="${REF#refs/heads/}"
    SAFE_BRANCH=$(echo "$BRANCH" | sed 's#[^A-Za-z0-9._-]#-#g' | tr '[:upper:]' '[:lower:]')
    TARGET_PATH="previews/pull-requests/${SAFE_BRANCH}"
    echo "Push to branch: $BRANCH"
  fi
else
  echo "::error::Unsupported event type: $EVENT"
  exit 1
fi

echo "Target path: $TARGET_PATH"

# Create staging directory structure
STAGING_DIR="/tmp/staging"
STAGING_PATH="${STAGING_DIR}/${TARGET_PATH}/latest"

echo "Creating staging directories..."
rm -rf "$STAGING_DIR" || true
mkdir -p "$STAGING_PATH"

# Copy benchmark results
echo "Copying benchmark results..."
cp benchmarks/benchmark.txt "$STAGING_PATH/"

# Create a simple index.html for the benchmark results
cat > "$STAGING_PATH/index.html" <<EOF
<!DOCTYPE html>
<html>
<head>
    <title>Benchmark Results</title>
    <style>
        body { font-family: monospace; margin: 20px; background: #1e1e1e; color: #d4d4d4; }
        pre { background: #2d2d2d; padding: 15px; border-radius: 5px; overflow-x: auto; }
        h1 { color: #4ec9b0; }
        a { color: #569cd6; }
    </style>
</head>
<body>
    <h1>Benchmark Results</h1>
    <p><a href="../">← Back</a></p>
    <pre>$(cat benchmarks/benchmark.txt)</pre>
</body>
</html>
EOF

# Create timestamp for archiving
TIMESTAMP=$(date -u +"%Y-%m-%d-%H%M%S")

# Export environment variables for downstream steps
echo "Exporting environment variables..."
{
  echo "IS_MAIN_BRANCH=${IS_MAIN_BRANCH}"
  echo "IS_PR=${IS_PR}"
  echo "PR_NUMBER=${PR_NUMBER}"
  echo "TARGET_PATH=${TARGET_PATH}"
  echo "STAGING_DIR=${STAGING_DIR}"
  echo "STAGING_PATH=${STAGING_PATH}"
  echo "TIMESTAMP=${TIMESTAMP}"
} >> "$GITHUB_ENV"

echo "✅ Benchmark staging complete"
echo "Staged to: ${TARGET_PATH}/latest"
