#!/bin/bash
set -euo pipefail

# benchmark-promote.sh
# Purpose: Promote validated benchmarks to baselines (only after regression passes)
# Outputs: Updated /tmp/pages/ with promoted baselines and archives

echo "=== Benchmark Promotion Script ==="

# Validate inputs
STAGING_DIR="${STAGING_DIR:-/tmp/pages}"
SAFE_NAME="${SAFE_NAME:-}"
IS_MAIN_PUSH="${IS_MAIN_PUSH:-false}"
MAX_OLD_ARCHIVES="${MAX_OLD_ARCHIVES:-10}"

if [ -z "$SAFE_NAME" ]; then
  echo "::error::SAFE_NAME not set"
  exit 1
fi

PREVIEW_DIR="${STAGING_DIR}/previews/${SAFE_NAME}"

if [ ! -d "$PREVIEW_DIR" ]; then
  echo "::error::Preview directory not found: $PREVIEW_DIR"
  exit 1
fi

echo "Promoting benchmarks from: previews/${SAFE_NAME}/"
echo "Is main push: $IS_MAIN_PUSH"

# Fetch current gh-pages content to check for existing latest/
BASELINE_DIR="/tmp/gh-pages-baseline"
rm -rf "$BASELINE_DIR" || true

if git ls-remote --exit-code --heads origin gh-pages &> /dev/null; then
  echo "Fetching existing gh-pages content..."
  git fetch origin gh-pages:gh-pages || true
  git worktree add "$BASELINE_DIR" gh-pages 2>/dev/null || {
    if [ -d "$BASELINE_DIR" ]; then
      echo "Using existing baseline directory"
    else
      mkdir -p "$BASELINE_DIR"
    fi
  }
else
  echo "No existing gh-pages branch (first run)"
  mkdir -p "$BASELINE_DIR"
fi

# Archive current latest/ to old/<timestamp>/ if it exists
if [ -d "${BASELINE_DIR}/latest" ] && [ -f "${BASELINE_DIR}/latest/benchmark.txt" ]; then
  TIMESTAMP=$(date -u +"%Y-%m-%dT%H-%M-%SZ")
  ARCHIVE_DIR="${STAGING_DIR}/old/${TIMESTAMP}"
  
  echo "Archiving current latest/ to old/${TIMESTAMP}/"
  mkdir -p "$ARCHIVE_DIR"
  cp -r "${BASELINE_DIR}/latest/." "$ARCHIVE_DIR/"
  
  echo "✅ Archived previous baseline to old/${TIMESTAMP}/"
else
  echo "No existing latest/ baseline to archive"
fi

# Copy existing old/ archives to staging (to preserve them)
if [ -d "${BASELINE_DIR}/old" ]; then
  echo "Preserving existing old/ archives..."
  mkdir -p "${STAGING_DIR}/old"
  cp -r "${BASELINE_DIR}/old/." "${STAGING_DIR}/old/" || true
fi

# Promote based on context
if [ "$IS_MAIN_PUSH" = "true" ]; then
  echo "Main branch push detected - promoting to both previews/main/ and latest/"
  
  # Promote to previews/main/
  MAIN_DIR="${STAGING_DIR}/previews/main"
  mkdir -p "$MAIN_DIR"
  cp -r "$PREVIEW_DIR/." "$MAIN_DIR/"
  echo "✅ Promoted to previews/main/"
  
  # Promote to latest/
  LATEST_DIR="${STAGING_DIR}/latest"
  mkdir -p "$LATEST_DIR"
  cp -r "$PREVIEW_DIR/." "$LATEST_DIR/"
  echo "✅ Promoted to latest/"
else
  echo "Non-main branch - promoting to latest/ only"
  
  # Promote to latest/
  LATEST_DIR="${STAGING_DIR}/latest"
  mkdir -p "$LATEST_DIR"
  cp -r "$PREVIEW_DIR/." "$LATEST_DIR/"
  echo "✅ Promoted to latest/"
fi

# Cleanup old archives (keep only last N)
if [ -d "${STAGING_DIR}/old" ]; then
  echo "Cleaning up old archives (keeping last ${MAX_OLD_ARCHIVES})..."
  
  # List archives sorted by name (timestamp), newest first
  ARCHIVES=($(ls -1 "${STAGING_DIR}/old" | sort -r))
  ARCHIVE_COUNT=${#ARCHIVES[@]}
  
  echo "Found ${ARCHIVE_COUNT} archives"
  
  if [ "$ARCHIVE_COUNT" -gt "$MAX_OLD_ARCHIVES" ]; then
    echo "Removing old archives beyond retention limit..."
    
    # Remove archives beyond the limit
    for ((i=MAX_OLD_ARCHIVES; i<ARCHIVE_COUNT; i++)); do
      ARCHIVE_TO_DELETE="${STAGING_DIR}/old/${ARCHIVES[$i]}"
      echo "Deleting: ${ARCHIVES[$i]}"
      rm -rf "$ARCHIVE_TO_DELETE"
    done
    
    echo "✅ Cleaned up $((ARCHIVE_COUNT - MAX_OLD_ARCHIVES)) old archives"
  else
    echo "Archive count within limit, no cleanup needed"
  fi
fi

# Cleanup baseline worktree
rm -rf "$BASELINE_DIR" || true

echo "✅ Benchmark promotion complete"
echo ""
echo "Summary:"
echo "  - Preview: previews/${SAFE_NAME}/"
if [ "$IS_MAIN_PUSH" = "true" ]; then
  echo "  - Main baseline: previews/main/"
fi
echo "  - Latest baseline: latest/"
if [ -d "${STAGING_DIR}/old" ]; then
  REMAINING_ARCHIVES=$(ls -1 "${STAGING_DIR}/old" 2>/dev/null | wc -l)
  echo "  - Archived baselines: ${REMAINING_ARCHIVES}"
fi
