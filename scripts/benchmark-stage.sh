#!/bin/bash
set -euo pipefail

# benchmark-stage.sh
# Purpose: Stage benchmark results into the proper folder structure
# Outputs: /tmp/pages/ directory with staged content and environment variables

echo "=== Benchmark Staging Script ==="

# Validate inputs
if [ ! -f "benchmark.txt" ]; then
  echo "::error::benchmark.txt not found. Cannot stage benchmarks."
  exit 1
fi

# Check if benchmark.txt has actual results
if grep -q "No benchmarks found" benchmark.txt; then
  echo "::error::No benchmarks found in benchmark.txt. Cannot stage empty results."
  exit 1
fi

# Determine safe folder name based on event type
EVENT="${GITHUB_EVENT_NAME:-unknown}"
REF="${GITHUB_REF:-}"
BASE_REF="${GITHUB_BASE_REF:-}"

echo "Event: $EVENT"
echo "Ref: $REF"
echo "Base Ref: $BASE_REF"

# Compute SAFE_NAME for preview folder
if [ "$EVENT" = "pull_request" ]; then
  # Extract PR number from event path
  if [ -f "$GITHUB_EVENT_PATH" ]; then
    PR_NUM=$(jq -r '.number // "unknown"' < "$GITHUB_EVENT_PATH")
    SAFE_NAME="pr-${PR_NUM}"
    echo "Pull Request #${PR_NUM}"
  else
    SAFE_NAME="pr-unknown"
  fi
elif [[ "$REF" == refs/heads/* ]]; then
  # Extract branch name
  BRANCH="${REF#refs/heads/}"
  SAFE_NAME="$BRANCH"
  echo "Branch: $BRANCH"
else
  # Fallback to run ID
  SAFE_NAME="run-${GITHUB_RUN_ID:-unknown}"
  echo "Using run ID for safe name"
fi

# Sanitize SAFE_NAME: replace special chars with dash, lowercase
SAFE_NAME=$(echo "$SAFE_NAME" | sed 's#[^A-Za-z0-9._-]#-#g' | tr '[:upper:]' '[:lower:]')
echo "Safe folder name: $SAFE_NAME"

# Determine target branch for PRs
TARGET_BRANCH=""
IS_PR_TO_MAIN="false"
IS_MAIN_PUSH="false"

if [ "$EVENT" = "pull_request" ]; then
  TARGET_BRANCH="${BASE_REF}"
  echo "PR targets branch: $TARGET_BRANCH"
  
  if [ "$TARGET_BRANCH" = "main" ] || [ "$TARGET_BRANCH" = "master" ]; then
    IS_PR_TO_MAIN="true"
    echo "This PR targets main branch"
  fi
elif [ "$EVENT" = "push" ]; then
  if [[ "$REF" == "refs/heads/main" ]] || [[ "$REF" == "refs/heads/master" ]]; then
    IS_MAIN_PUSH="true"
    echo "This is a push to main branch"
  fi
fi

# Create staging directory structure
STAGING_DIR="/tmp/pages"
PREVIEW_DIR="${STAGING_DIR}/previews/${SAFE_NAME}"

echo "Creating staging directories..."
rm -rf "$STAGING_DIR" || true
mkdir -p "$PREVIEW_DIR"

# Copy benchmark results to preview folder
echo "Copying benchmark.txt to preview folder..."
cp benchmark.txt "$PREVIEW_DIR/"

# Copy any HTML files if they exist (from benchmark-action or custom templates)
if [ -d "dev/bench" ]; then
  echo "Copying dev/bench contents to preview folder..."
  cp -r dev/bench/. "$PREVIEW_DIR/" || true
fi

# Create .nojekyll file to prevent Jekyll processing
touch "${STAGING_DIR}/.nojekyll"

# Verify staging
echo "Staging directory contents:"
ls -la "$STAGING_DIR" || true
echo "Preview directory contents:"
ls -la "$PREVIEW_DIR" || true

# Export environment variables for downstream steps
echo "Exporting environment variables..."
{
  echo "SAFE_NAME=${SAFE_NAME}"
  echo "TARGET_BRANCH=${TARGET_BRANCH}"
  echo "IS_PR_TO_MAIN=${IS_PR_TO_MAIN}"
  echo "IS_MAIN_PUSH=${IS_MAIN_PUSH}"
  echo "STAGING_DIR=${STAGING_DIR}"
  echo "PREVIEW_DIR=${PREVIEW_DIR}"
} >> "$GITHUB_ENV"

echo "✅ Benchmark staging complete"
echo "Preview will be published to: previews/${SAFE_NAME}/"
