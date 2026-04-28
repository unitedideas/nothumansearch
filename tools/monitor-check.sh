#!/bin/bash
set -euo pipefail

# NHS monitor-check — weekly job that re-crawls every monitored domain,
# diffs the agentic-readiness score/signals against the last check, and
# emails the watcher when something regressed. Runs inside the Fly machine
# so the crawl goes out from the same IP/user-agent as the daily recrawl.

export PATH="/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin"
FLY_BIN="/opt/homebrew/bin/fly"

APP_DIR="$(cd "$(dirname "$0")/.." && pwd)"
LOG_FILE="${APP_DIR}/tools/monitor-check.log"

echo "$(date '+%Y-%m-%d %H:%M:%S') NHS monitor-check starting" >> "$LOG_FILE"

cd "$APP_DIR"

env FLY_ACCESS_TOKEN="$(/usr/bin/security find-generic-password -a foundry -s fly-api-token -w)" \
  "$FLY_BIN" ssh console -a nothumansearch -C "/app/monitor-check -cutoff-hours 144 -limit 500" >> "$LOG_FILE" 2>&1

echo "$(date '+%Y-%m-%d %H:%M:%S') NHS monitor-check done" >> "$LOG_FILE"
