#!/bin/bash
set -euo pipefail

# NHS automated recrawl — runs daily via launchd
# Processes pending submissions, then re-crawls all indexed sites

export PATH="/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin"

APP_DIR="$(cd "$(dirname "$0")/.." && pwd)"
LOG_FILE="${APP_DIR}/tools/recrawl.log"

echo "$(date '+%Y-%m-%d %H:%M:%S') NHS recrawl starting" >> "$LOG_FILE"

cd "$APP_DIR"

# Seed new sites first (idempotent — existing sites just get upserted)
fly ssh console -a nothumansearch -C "/app/crawler -seed -workers 10" >> "$LOG_FILE" 2>&1

# Then recrawl all (updates scores, categories, tags for existing sites)
fly ssh console -a nothumansearch -C "/app/crawler -recrawl -workers 10" >> "$LOG_FILE" 2>&1

# Submit key pages to IndexNow for Bing/Yandex
curl -s -X POST "https://api.indexnow.org/indexnow" \
  -H "Content-Type: application/json" \
  -d '{
    "host": "nothumansearch.fly.dev",
    "key": "bb1637af360f471ab2a1555d45d683ea",
    "keyLocation": "https://nothumansearch.fly.dev/bb1637af360f471ab2a1555d45d683ea.txt",
    "urlList": [
      "https://nothumansearch.fly.dev/",
      "https://nothumansearch.fly.dev/about",
      "https://nothumansearch.fly.dev/sitemap.xml"
    ]
  }' >> /dev/null 2>&1 || true

echo "$(date '+%Y-%m-%d %H:%M:%S') NHS recrawl complete" >> "$LOG_FILE"
echo "---" >> "$LOG_FILE"
