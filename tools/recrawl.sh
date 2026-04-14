#!/bin/bash
set -euo pipefail

# NHS automated recrawl — runs daily via launchd
# Processes pending submissions, then re-crawls all indexed sites

export PATH="/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin"

APP_DIR="$(cd "$(dirname "$0")/.." && pwd)"
LOG_FILE="${APP_DIR}/tools/recrawl.log"

echo "$(date '+%Y-%m-%d %H:%M:%S') NHS recrawl starting" >> "$LOG_FILE"

cd "$APP_DIR"

# Run recrawl on Fly with 10 workers
fly ssh console -a nothumansearch -C "/app/crawler -recrawl -workers 10" >> "$LOG_FILE" 2>&1

echo "$(date '+%Y-%m-%d %H:%M:%S') NHS recrawl complete" >> "$LOG_FILE"
echo "---" >> "$LOG_FILE"
