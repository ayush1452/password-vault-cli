#!/bin/bash
set -euo pipefail

# generate-index.sh
# Purpose: Generate HTML index pages for the new directory structure
# Structure: / → previews/ → master/ | pull-requests/ → pr-N/ → latest/ | old/

echo "=== Generating Index Pages ==="

PAGES_DIR="/tmp/gh-pages"

if [ ! -d "$PAGES_DIR" ]; then
  echo "::error::Pages directory not found: $PAGES_DIR"
  exit 1
fi

cd "$PAGES_DIR"

# CSS styles for all index pages
read -r -d '' STYLES <<'EOF' || true
body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
  max-width: 1200px;
  margin: 40px auto;
  padding: 0 20px;
  background: #0d1117;
  color: #c9d1d9;
}
h1 {
  color: #58a6ff;
  border-bottom: 2px solid #21262d;
  padding-bottom: 10px;
}
h2 {
  color: #8b949e;
  margin-top: 30px;
}
a {
  color: #58a6ff;
  text-decoration: none;
}
a:hover {
  text-decoration: underline;
}
.card {
  background: #161b22;
  border: 1px solid #30363d;
  border-radius: 6px;
  padding: 16px;
  margin: 12px 0;
}
.card:hover {
  border-color: #58a6ff;
}
.meta {
  color: #8b949e;
  font-size: 14px;
  margin-top: 8px;
}
.breadcrumb {
  color: #8b949e;
  margin-bottom: 20px;
}
.breadcrumb a {
  color: #58a6ff;
}
ul {
  list-style: none;
  padding: 0;
}
li {
  margin: 8px 0;
}
.icon {
  margin-right: 8px;
}
EOF

# Function to generate root index
generate_root_index() {
  echo "Generating root index..."
  cat > index.html <<EOF
<!DOCTYPE html>
<html>
<head>
    <title>Benchmark Results</title>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <style>${STYLES}</style>
</head>
<body>
    <h1>📊 Benchmark Results</h1>
    <p>Performance benchmarks for the password-vault-cli project.</p>
    
    <div class="card">
        <h2>📁 <a href="previews/">Browse Previews</a></h2>
        <p class="meta">View benchmark results organized by branch type</p>
    </div>
    
    <h2>Quick Links</h2>
    <ul>
        <li><span class="icon">🎯</span><a href="previews/master/latest/">Latest Master Benchmarks</a></li>
        <li><span class="icon">📜</span><a href="previews/master/old/">Master Archive</a></li>
        <li><span class="icon">🔀</span><a href="previews/pull-requests/">Pull Request Benchmarks</a></li>
    </ul>
    
    <p class="meta">Last updated: $(date -u +"%Y-%m-%d %H:%M:%S UTC")</p>
</body>
</html>
EOF
}

# Function to generate previews index
generate_previews_index() {
  echo "Generating previews index..."
  mkdir -p previews
  cat > previews/index.html <<EOF
<!DOCTYPE html>
<html>
<head>
    <title>Benchmark Previews</title>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <style>${STYLES}</style>
</head>
<body>
    <div class="breadcrumb">
        <a href="../">Home</a> / Previews
    </div>
    
    <h1>📁 Benchmark Previews</h1>
    
    <div class="card">
        <h2>🎯 <a href="master/">Master Branch</a></h2>
        <p class="meta">Benchmarks from the main branch</p>
        <ul>
            <li><a href="master/latest/">Latest</a> - Current master benchmarks</li>
            <li><a href="master/old/">Archive</a> - Historical master benchmarks</li>
        </ul>
    </div>
    
    <div class="card">
        <h2>🔀 <a href="pull-requests/">Pull Requests</a></h2>
        <p class="meta">Benchmarks from pull requests and feature branches</p>
    </div>
    
    <p class="meta">Last updated: $(date -u +"%Y-%m-%d %H:%M:%S UTC")</p>
</body>
</html>
EOF
}

# Function to generate master index
generate_master_index() {
  echo "Generating master index..."
  mkdir -p previews/master
  
  # Count archives
  ARCHIVE_COUNT=0
  if [ -d "previews/master/old" ]; then
    ARCHIVE_COUNT=$(find previews/master/old -mindepth 1 -maxdepth 1 -type d 2>/dev/null | wc -l)
  fi
  
  cat > previews/master/index.html <<EOF
<!DOCTYPE html>
<html>
<head>
    <title>Master Branch Benchmarks</title>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <style>${STYLES}</style>
</head>
<body>
    <div class="breadcrumb">
        <a href="../../">Home</a> / <a href="../">Previews</a> / Master
    </div>
    
    <h1>🎯 Master Branch Benchmarks</h1>
    
    <div class="card">
        <h2><a href="latest/">📈 Latest</a></h2>
        <p class="meta">Current benchmarks from the main branch</p>
    </div>
    
    <div class="card">
        <h2><a href="old/">📜 Archive</a></h2>
        <p class="meta">Historical benchmarks (${ARCHIVE_COUNT} archived versions)</p>
    </div>
    
    <p class="meta">Last updated: $(date -u +"%Y-%m-%d %H:%M:%S UTC")</p>
</body>
</html>
EOF
}

# Function to generate master/old index
generate_master_old_index() {
  echo "Generating master/old index..."
  mkdir -p previews/master/old
  
  cat > previews/master/old/index.html <<EOF
<!DOCTYPE html>
<html>
<head>
    <title>Master Branch Archive</title>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <style>${STYLES}</style>
</head>
<body>
    <div class="breadcrumb">
        <a href="../../../">Home</a> / <a href="../../">Previews</a> / <a href="../">Master</a> / Archive
    </div>
    
    <h1>📜 Master Branch Archive</h1>
    <p>Historical benchmark results from the main branch</p>
    
    <h2>Archived Versions</h2>
EOF

  # List archived versions (newest first)
  if [ -d "previews/master/old" ]; then
    find previews/master/old -mindepth 1 -maxdepth 1 -type d -printf '%T@ %p\n' 2>/dev/null | \
      sort -rn | \
      cut -d' ' -f2- | \
      while read -r dir; do
        name=$(basename "$dir")
        echo "    <div class=\"card\">" >> previews/master/old/index.html
        echo "        <a href=\"${name}/\">📦 ${name}</a>" >> previews/master/old/index.html
        echo "    </div>" >> previews/master/old/index.html
      done
  fi

  cat >> previews/master/old/index.html <<EOF
    
    <p class="meta">Last updated: $(date -u +"%Y-%m-%d %H:%M:%S UTC")</p>
</body>
</html>
EOF
}

# Function to generate pull-requests index
generate_pull_requests_index() {
  echo "Generating pull-requests index..."
  mkdir -p previews/pull-requests
  
  cat > previews/pull-requests/index.html <<EOF
<!DOCTYPE html>
<html>
<head>
    <title>Pull Request Benchmarks</title>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <style>${STYLES}</style>
</head>
<body>
    <div class="breadcrumb">
        <a href="../../">Home</a> / <a href="../">Previews</a> / Pull Requests
    </div>
    
    <h1>🔀 Pull Request Benchmarks</h1>
    <p>Benchmark results from pull requests and feature branches</p>
    
    <h2>Active PRs and Branches</h2>
EOF

  # List PR directories (newest first)
  if [ -d "previews/pull-requests" ]; then
    find previews/pull-requests -mindepth 1 -maxdepth 1 -type d -printf '%T@ %p\n' 2>/dev/null | \
      sort -rn | \
      cut -d' ' -f2- | \
      while read -r dir; do
        name=$(basename "$dir")
        # Count archives for this PR
        archive_count=0
        if [ -d "$dir/old" ]; then
          archive_count=$(find "$dir/old" -mindepth 1 -maxdepth 1 -type d 2>/dev/null | wc -l)
        fi
        
        echo "    <div class=\"card\">" >> previews/pull-requests/index.html
        echo "        <h3><a href=\"${name}/\">${name}</a></h3>" >> previews/pull-requests/index.html
        echo "        <p class=\"meta\">${archive_count} archived versions</p>" >> previews/pull-requests/index.html
        echo "        <ul>" >> previews/pull-requests/index.html
        echo "            <li><a href=\"${name}/latest/\">Latest</a></li>" >> previews/pull-requests/index.html
        echo "            <li><a href=\"${name}/old/\">Archive</a></li>" >> previews/pull-requests/index.html
        echo "        </ul>" >> previews/pull-requests/index.html
        echo "    </div>" >> previews/pull-requests/index.html
      done
  fi

  cat >> previews/pull-requests/index.html <<EOF
    
    <p class="meta">Last updated: $(date -u +"%Y-%m-%d %H:%M:%S UTC")</p>
</body>
</html>
EOF
}

# Function to generate individual PR index
generate_pr_index() {
  local pr_dir="$1"
  local pr_name=$(basename "$pr_dir")
  
  echo "Generating index for $pr_name..."
  
  # Count archives
  archive_count=0
  if [ -d "$pr_dir/old" ]; then
    archive_count=$(find "$pr_dir/old" -mindepth 1 -maxdepth 1 -type d 2>/dev/null | wc -l)
  fi
  
  cat > "$pr_dir/index.html" <<EOF
<!DOCTYPE html>
<html>
<head>
    <title>${pr_name} Benchmarks</title>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <style>${STYLES}</style>
</head>
<body>
    <div class="breadcrumb">
        <a href="../../../">Home</a> / <a href="../../">Previews</a> / <a href="../">Pull Requests</a> / ${pr_name}
    </div>
    
    <h1>📊 ${pr_name} Benchmarks</h1>
    
    <div class="card">
        <h2><a href="latest/">📈 Latest</a></h2>
        <p class="meta">Current benchmarks for this PR</p>
    </div>
    
    <div class="card">
        <h2><a href="old/">📜 Archive</a></h2>
        <p class="meta">Historical benchmarks (${archive_count} archived versions)</p>
    </div>
    
    <p class="meta">Last updated: $(date -u +"%Y-%m-%d %H:%M:%S UTC")</p>
</body>
</html>
EOF
}

# Function to generate PR old index
generate_pr_old_index() {
  local pr_old_dir="$1"
  local pr_name=$(basename "$(dirname "$pr_old_dir")")
  
  echo "Generating old index for $pr_name..."
  
  cat > "$pr_old_dir/index.html" <<EOF
<!DOCTYPE html>
<html>
<head>
    <title>${pr_name} Archive</title>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <style>${STYLES}</style>
</head>
<body>
    <div class="breadcrumb">
        <a href="../../../../">Home</a> / <a href="../../../">Previews</a> / <a href="../../">Pull Requests</a> / <a href="../">${pr_name}</a> / Archive
    </div>
    
    <h1>📜 ${pr_name} Archive</h1>
    <p>Historical benchmark results for this PR</p>
    
    <h2>Archived Versions</h2>
EOF

  # List archived versions (newest first)
  find "$pr_old_dir" -mindepth 1 -maxdepth 1 -type d -printf '%T@ %p\n' 2>/dev/null | \
    sort -rn | \
    cut -d' ' -f2- | \
    while read -r dir; do
      name=$(basename "$dir")
      echo "    <div class=\"card\">" >> "$pr_old_dir/index.html"
      echo "        <a href=\"${name}/\">📦 ${name}</a>" >> "$pr_old_dir/index.html"
      echo "    </div>" >> "$pr_old_dir/index.html"
    done

  cat >> "$pr_old_dir/index.html" <<EOF
    
    <p class="meta">Last updated: $(date -u +"%Y-%m-%d %H:%M:%S UTC")</p>
</body>
</html>
EOF
}

# Generate all index pages
generate_root_index
generate_previews_index

# Generate master indexes if directory exists
if [ -d "previews/master" ]; then
  generate_master_index
  if [ -d "previews/master/old" ]; then
    generate_master_old_index
  fi
fi

# Generate pull-requests indexes if directory exists
if [ -d "previews/pull-requests" ]; then
  generate_pull_requests_index
  
  # Generate index for each PR
  find previews/pull-requests -mindepth 1 -maxdepth 1 -type d 2>/dev/null | while read -r pr_dir; do
    generate_pr_index "$pr_dir"
    
    # Generate old index if it exists
    if [ -d "$pr_dir/old" ]; then
      generate_pr_old_index "$pr_dir/old"
    fi
  done
fi

echo "✅ Index generation complete"
echo ""
echo "Generated:"
if [ -f "index.html" ]; then echo "  - /index.html"; fi
if [ -f "previews/index.html" ]; then echo "  - /previews/index.html"; fi
if [ -f "previews/master/index.html" ]; then echo "  - /previews/master/index.html"; fi
if [ -f "previews/master/old/index.html" ]; then echo "  - /previews/master/old/index.html"; fi
if [ -f "previews/pull-requests/index.html" ]; then echo "  - /previews/pull-requests/index.html"; fi

# Count PR indexes
pr_count=$(find previews/pull-requests -mindepth 1 -maxdepth 1 -name "index.html" 2>/dev/null | wc -l)
if [ "$pr_count" -gt 0 ]; then
  echo "  - ${pr_count} PR index pages"
fi
