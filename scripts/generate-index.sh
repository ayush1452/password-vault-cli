#!/bin/bash
set -euo pipefail

# generate-index.sh
# Purpose: Generate HTML index pages for navigation
# Outputs: index.html files in /, /previews/, and /old/

echo "=== Generating Index Pages ==="

STAGING_DIR="${STAGING_DIR:-/tmp/pages}"

if [ ! -d "$STAGING_DIR" ]; then
  echo "::error::Staging directory not found: $STAGING_DIR"
  exit 1
fi

TIMESTAMP=$(date -u +"%Y-%m-%d %H:%M:%S UTC")
REPO_NAME="${GITHUB_REPOSITORY##*/}"
REPO_OWNER="${GITHUB_REPOSITORY_OWNER:-}"

# Generate root index.html
echo "Generating root index..."
cat > "${STAGING_DIR}/index.html" << 'EOF'
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Benchmark Results - PASSWORD_VAULT_CLI</title>
  <style>
    * { margin: 0; padding: 0; box-sizing: border-box; }
    body {
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
      line-height: 1.6;
      color: #333;
      background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
      min-height: 100vh;
      padding: 20px;
    }
    .container {
      max-width: 900px;
      margin: 0 auto;
      background: white;
      border-radius: 12px;
      box-shadow: 0 20px 60px rgba(0,0,0,0.3);
      overflow: hidden;
    }
    .header {
      background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
      color: white;
      padding: 40px 30px;
      text-align: center;
    }
    .header h1 {
      font-size: 2.5em;
      margin-bottom: 10px;
      font-weight: 700;
    }
    .header p {
      font-size: 1.1em;
      opacity: 0.9;
    }
    .content {
      padding: 40px 30px;
    }
    .card {
      background: #f8f9fa;
      border-radius: 8px;
      padding: 25px;
      margin-bottom: 20px;
      border-left: 4px solid #667eea;
      transition: transform 0.2s, box-shadow 0.2s;
    }
    .card:hover {
      transform: translateY(-2px);
      box-shadow: 0 4px 12px rgba(0,0,0,0.1);
    }
    .card h2 {
      color: #667eea;
      margin-bottom: 10px;
      font-size: 1.5em;
    }
    .card p {
      color: #666;
      margin-bottom: 15px;
    }
    .card a {
      display: inline-block;
      background: #667eea;
      color: white;
      padding: 10px 20px;
      border-radius: 6px;
      text-decoration: none;
      font-weight: 600;
      transition: background 0.2s;
    }
    .card a:hover {
      background: #5568d3;
    }
    .footer {
      text-align: center;
      padding: 20px;
      color: #999;
      font-size: 0.9em;
      border-top: 1px solid #eee;
    }
    .badge {
      display: inline-block;
      background: #28a745;
      color: white;
      padding: 4px 12px;
      border-radius: 12px;
      font-size: 0.85em;
      font-weight: 600;
      margin-left: 10px;
    }
  </style>
</head>
<body>
  <div class="container">
    <div class="header">
      <h1>🔐 PASSWORD_VAULT_CLI</h1>
      <p>Performance Benchmark Results</p>
    </div>
    <div class="content">
      <div class="card">
        <h2>📊 Latest Benchmarks <span class="badge">Current</span></h2>
        <p>View the most recent validated benchmark results from the main branch.</p>
        <a href="latest/">View Latest Results →</a>
      </div>
      <div class="card">
        <h2>🔍 Preview Benchmarks</h2>
        <p>Browse benchmark results from all pull requests and feature branches.</p>
        <a href="previews/">Browse All Previews →</a>
      </div>
      <div class="card">
        <h2>📦 Archived Baselines</h2>
        <p>Access historical benchmark baselines for trend analysis.</p>
        <a href="old/">View Archives →</a>
      </div>
    </div>
    <div class="footer">
      Last updated: TIMESTAMP_PLACEHOLDER
    </div>
  </div>
</body>
</html>
EOF

# Replace placeholders
sed "s/PASSWORD_VAULT_CLI/${REPO_NAME}/g" "${STAGING_DIR}/index.html" > "${STAGING_DIR}/index.html.tmp" && mv "${STAGING_DIR}/index.html.tmp" "${STAGING_DIR}/index.html"
sed "s/TIMESTAMP_PLACEHOLDER/${TIMESTAMP}/g" "${STAGING_DIR}/index.html" > "${STAGING_DIR}/index.html.tmp" && mv "${STAGING_DIR}/index.html.tmp" "${STAGING_DIR}/index.html"

# Generate previews/index.html
if [ -d "${STAGING_DIR}/previews" ]; then
  echo "Generating previews index..."
  
  # Build list of preview folders
  PREVIEW_LIST=""
  if [ -d "${STAGING_DIR}/previews" ]; then
    for preview in $(ls -1 "${STAGING_DIR}/previews" | sort -r); do
      if [ -d "${STAGING_DIR}/previews/${preview}" ]; then
        # Determine badge color and label
        BADGE_CLASS="badge-default"
        BADGE_LABEL="Branch"
        
        if [[ "$preview" == pr-* ]]; then
          BADGE_CLASS="badge-pr"
          BADGE_LABEL="Pull Request"
        elif [[ "$preview" == "main" ]] || [[ "$preview" == "master" ]]; then
          BADGE_CLASS="badge-main"
          BADGE_LABEL="Main Branch"
        fi
        
        PREVIEW_LIST="${PREVIEW_LIST}<div class=\"preview-item\"><a href=\"${preview}/\"><span class=\"preview-name\">${preview}</span><span class=\"${BADGE_CLASS}\">${BADGE_LABEL}</span></a></div>"
      fi
    done
  fi
  
  cat > "${STAGING_DIR}/previews/index.html" << 'EOF'
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Preview Benchmarks - PASSWORD_VAULT_CLI</title>
  <style>
    * { margin: 0; padding: 0; box-sizing: border-box; }
    body {
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
      line-height: 1.6;
      color: #333;
      background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
      min-height: 100vh;
      padding: 20px;
    }
    .container {
      max-width: 900px;
      margin: 0 auto;
      background: white;
      border-radius: 12px;
      box-shadow: 0 20px 60px rgba(0,0,0,0.3);
      overflow: hidden;
    }
    .header {
      background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
      color: white;
      padding: 40px 30px;
    }
    .header h1 {
      font-size: 2em;
      margin-bottom: 10px;
    }
    .header a {
      color: white;
      text-decoration: none;
      opacity: 0.9;
      font-size: 0.9em;
    }
    .header a:hover { opacity: 1; }
    .content {
      padding: 30px;
    }
    .preview-item {
      background: #f8f9fa;
      border-radius: 8px;
      margin-bottom: 12px;
      border-left: 4px solid #667eea;
      transition: transform 0.2s, box-shadow 0.2s;
    }
    .preview-item:hover {
      transform: translateX(4px);
      box-shadow: 0 2px 8px rgba(0,0,0,0.1);
    }
    .preview-item a {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 18px 20px;
      text-decoration: none;
      color: #333;
    }
    .preview-name {
      font-weight: 600;
      font-size: 1.1em;
      font-family: 'Monaco', 'Courier New', monospace;
    }
    .badge-default, .badge-pr, .badge-main {
      padding: 4px 12px;
      border-radius: 12px;
      font-size: 0.8em;
      font-weight: 600;
      color: white;
    }
    .badge-default { background: #6c757d; }
    .badge-pr { background: #17a2b8; }
    .badge-main { background: #28a745; }
    .empty {
      text-align: center;
      padding: 60px 20px;
      color: #999;
    }
    .footer {
      text-align: center;
      padding: 20px;
      color: #999;
      font-size: 0.9em;
      border-top: 1px solid #eee;
    }
  </style>
</head>
<body>
  <div class="container">
    <div class="header">
      <h1>🔍 Preview Benchmarks</h1>
      <a href="../">← Back to Home</a>
    </div>
    <div class="content">
      PREVIEW_LIST_PLACEHOLDER
    </div>
    <div class="footer">
      Last updated: TIMESTAMP_PLACEHOLDER
    </div>
  </div>
</body>
</html>
EOF
  
  if [ -z "$PREVIEW_LIST" ]; then
    PREVIEW_LIST="<div class=\"empty\">No preview benchmarks available yet.</div>"
  fi
  
  sed "s/PASSWORD_VAULT_CLI/${REPO_NAME}/g" "${STAGING_DIR}/previews/index.html" > "${STAGING_DIR}/previews/index.html.tmp" && mv "${STAGING_DIR}/previews/index.html.tmp" "${STAGING_DIR}/previews/index.html"
  sed "s|PREVIEW_LIST_PLACEHOLDER|${PREVIEW_LIST}|g" "${STAGING_DIR}/previews/index.html" > "${STAGING_DIR}/previews/index.html.tmp" && mv "${STAGING_DIR}/previews/index.html.tmp" "${STAGING_DIR}/previews/index.html"
  sed "s/TIMESTAMP_PLACEHOLDER/${TIMESTAMP}/g" "${STAGING_DIR}/previews/index.html" > "${STAGING_DIR}/previews/index.html.tmp" && mv "${STAGING_DIR}/previews/index.html.tmp" "${STAGING_DIR}/previews/index.html"
fi

# Generate old/index.html
if [ -d "${STAGING_DIR}/old" ]; then
  echo "Generating old archives index..."
  
  # Build list of archived folders
  ARCHIVE_LIST=""
  if [ -d "${STAGING_DIR}/old" ]; then
    for archive in $(ls -1 "${STAGING_DIR}/old" | sort -r); do
      if [ -d "${STAGING_DIR}/old/${archive}" ]; then
        # Parse timestamp for display
        DISPLAY_DATE=$(echo "$archive" | sed 's/T/ /g' | sed 's/-/:/g' | sed 's/Z//g')
        ARCHIVE_LIST="${ARCHIVE_LIST}<div class=\"archive-item\"><a href=\"${archive}/\"><span class=\"archive-name\">${DISPLAY_DATE}</span><span class=\"badge\">Archived</span></a></div>"
      fi
    done
  fi
  
  mkdir -p "${STAGING_DIR}/old"
  
  cat > "${STAGING_DIR}/old/index.html" << 'EOF'
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Archived Benchmarks - PASSWORD_VAULT_CLI</title>
  <style>
    * { margin: 0; padding: 0; box-sizing: border-box; }
    body {
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
      line-height: 1.6;
      color: #333;
      background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
      min-height: 100vh;
      padding: 20px;
    }
    .container {
      max-width: 900px;
      margin: 0 auto;
      background: white;
      border-radius: 12px;
      box-shadow: 0 20px 60px rgba(0,0,0,0.3);
      overflow: hidden;
    }
    .header {
      background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
      color: white;
      padding: 40px 30px;
    }
    .header h1 {
      font-size: 2em;
      margin-bottom: 10px;
    }
    .header a {
      color: white;
      text-decoration: none;
      opacity: 0.9;
      font-size: 0.9em;
    }
    .header a:hover { opacity: 1; }
    .content {
      padding: 30px;
    }
    .archive-item {
      background: #f8f9fa;
      border-radius: 8px;
      margin-bottom: 12px;
      border-left: 4px solid #6c757d;
      transition: transform 0.2s, box-shadow 0.2s;
    }
    .archive-item:hover {
      transform: translateX(4px);
      box-shadow: 0 2px 8px rgba(0,0,0,0.1);
    }
    .archive-item a {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 18px 20px;
      text-decoration: none;
      color: #333;
    }
    .archive-name {
      font-weight: 600;
      font-size: 1.1em;
      font-family: 'Monaco', 'Courier New', monospace;
    }
    .badge {
      background: #6c757d;
      color: white;
      padding: 4px 12px;
      border-radius: 12px;
      font-size: 0.8em;
      font-weight: 600;
    }
    .empty {
      text-align: center;
      padding: 60px 20px;
      color: #999;
    }
    .footer {
      text-align: center;
      padding: 20px;
      color: #999;
      font-size: 0.9em;
      border-top: 1px solid #eee;
    }
  </style>
</head>
<body>
  <div class="container">
    <div class="header">
      <h1>📦 Archived Benchmarks</h1>
      <a href="../">← Back to Home</a>
    </div>
    <div class="content">
      ARCHIVE_LIST_PLACEHOLDER
    </div>
    <div class="footer">
      Last updated: TIMESTAMP_PLACEHOLDER
    </div>
  </div>
</body>
</html>
EOF
  
  if [ -z "$ARCHIVE_LIST" ]; then
    ARCHIVE_LIST="<div class=\"empty\">No archived benchmarks available yet.</div>"
  fi
  
  sed "s/PASSWORD_VAULT_CLI/${REPO_NAME}/g" "${STAGING_DIR}/old/index.html" > "${STAGING_DIR}/old/index.html.tmp" && mv "${STAGING_DIR}/old/index.html.tmp" "${STAGING_DIR}/old/index.html"
  sed "s|ARCHIVE_LIST_PLACEHOLDER|${ARCHIVE_LIST}|g" "${STAGING_DIR}/old/index.html" > "${STAGING_DIR}/old/index.html.tmp" && mv "${STAGING_DIR}/old/index.html.tmp" "${STAGING_DIR}/old/index.html"
  sed "s/TIMESTAMP_PLACEHOLDER/${TIMESTAMP}/g" "${STAGING_DIR}/old/index.html" > "${STAGING_DIR}/old/index.html.tmp" && mv "${STAGING_DIR}/old/index.html.tmp" "${STAGING_DIR}/old/index.html"
fi

echo "✅ Index generation complete"
echo "Generated:"
echo "  - ${STAGING_DIR}/index.html"
if [ -f "${STAGING_DIR}/previews/index.html" ]; then
  echo "  - ${STAGING_DIR}/previews/index.html"
fi
if [ -f "${STAGING_DIR}/old/index.html" ]; then
  echo "  - ${STAGING_DIR}/old/index.html"
fi
