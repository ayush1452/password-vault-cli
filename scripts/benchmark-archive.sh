#!/bin/bash
set -euo pipefail

# benchmark-archive.sh
# Purpose: Archive current latest/ to old/ and copy new benchmarks to latest/
# Works for both master and PR contexts

echo "=== Benchmark Archive & Publish Script ==="

# Get configuration from environment
IS_MAIN_BRANCH="${IS_MAIN_BRANCH:-false}"
IS_PR="${IS_PR:-false}"
TARGET_PATH="${TARGET_PATH:-}"
STAGING_DIR="${STAGING_DIR:-/tmp/staging}"
TIMESTAMP="${TIMESTAMP:-$(date -u +"%Y-%m-%d-%H%M%S")}"
MAX_OLD_ARCHIVES="${MAX_OLD_ARCHIVES:-10}"

echo "Is main branch: $IS_MAIN_BRANCH"
echo "Is PR: $IS_PR"
echo "Target path: $TARGET_PATH"
echo "Timestamp: $TIMESTAMP"
echo "Max archives to keep: $MAX_OLD_ARCHIVES"

# Validate inputs
if [ -z "$TARGET_PATH" ]; then
  echo "::error::TARGET_PATH not set"
  exit 1
fi

if [ ! -d "$STAGING_DIR" ]; then
  echo "::error::Staging directory not found: $STAGING_DIR"
  exit 1
fi

# Fetch gh-pages branch
PAGES_DIR="/tmp/gh-pages"
echo "Fetching gh-pages branch..."
rm -rf "$PAGES_DIR" || true

if git ls-remote --exit-code --heads origin gh-pages &> /dev/null; then
  echo "gh-pages branch exists, cloning..."
  git clone --depth=1 --branch=gh-pages "https://x-access-token:${GITHUB_TOKEN}@github.com/${GITHUB_REPOSITORY}.git" "$PAGES_DIR" || {
    echo "::warning::Could not clone gh-pages, creating new"
    mkdir -p "$PAGES_DIR"
    cd "$PAGES_DIR"
    git init
    git checkout -b gh-pages
  }
else
  echo "gh-pages branch does not exist, creating new"
  mkdir -p "$PAGES_DIR"
  cd "$PAGES_DIR"
  git init
  git checkout -b gh-pages
fi

cd "$PAGES_DIR"

# Create target directory structure
TARGET_DIR="${PAGES_DIR}/${TARGET_PATH}"
LATEST_DIR="${TARGET_DIR}/latest"
OLD_DIR="${TARGET_DIR}/old"

mkdir -p "$TARGET_DIR"
mkdir -p "$OLD_DIR"

# Archive current latest/ to old/ if it exists
if [ -d "$LATEST_DIR" ] && [ "$(ls -A "$LATEST_DIR")" ]; then
  echo "Archiving current latest/ to old/${TIMESTAMP}/"
  ARCHIVE_DIR="${OLD_DIR}/${TIMESTAMP}"
  mkdir -p "$ARCHIVE_DIR"
  cp -r "$LATEST_DIR"/* "$ARCHIVE_DIR"/ || true
  echo "✅ Archived to: ${TARGET_PATH}/old/${TIMESTAMP}/"
else
  echo "No existing latest/ to archive (first run)"
fi

# Copy new benchmarks from staging to latest/
echo "Publishing new benchmarks to latest/..."
rm -rf "$LATEST_DIR" || true
mkdir -p "$LATEST_DIR"
cp -r "${STAGING_DIR}/${TARGET_PATH}/latest"/* "$LATEST_DIR"/ || {
  echo "::error::Failed to copy from staging"
  exit 1
}
echo "✅ Published to: ${TARGET_PATH}/latest/"

# Cleanup old archives (keep last N)
echo "Cleaning up old archives (keeping last ${MAX_OLD_ARCHIVES})..."
ARCHIVE_COUNT=$(find "$OLD_DIR" -mindepth 1 -maxdepth 1 -type d | wc -l)
echo "Current archive count: $ARCHIVE_COUNT"

if [ "$ARCHIVE_COUNT" -gt "$MAX_OLD_ARCHIVES" ]; then
  echo "Removing oldest archives..."
  # List directories by modification time, oldest first, skip the last N
  find "$OLD_DIR" -mindepth 1 -maxdepth 1 -type d -printf '%T+ %p\n' | \
    sort | \
    head -n -"$MAX_OLD_ARCHIVES" | \
    cut -d' ' -f2- | \
    while read -r dir; do
      echo "  Removing: $(basename "$dir")"
      rm -rf "$dir"
    done
  echo "✅ Cleanup complete"
else
  echo "Archive count within limit, no cleanup needed"
fi

# Copy all staged content to pages directory
echo "Copying all staged content..."
cp -r "$STAGING_DIR"/* "$PAGES_DIR"/ || true

# Create .nojekyll file
touch "$PAGES_DIR/.nojekyll"

echo "✅ Archive and publish complete"
echo ""
echo "Summary:"
echo "  - Published: ${TARGET_PATH}/latest/"
echo "  - Archived: ${TARGET_PATH}/old/${TIMESTAMP}/"
echo "  - Total archives: $(find "$OLD_DIR" -mindepth 1 -maxdepth 1 -type d | wc -l)"
