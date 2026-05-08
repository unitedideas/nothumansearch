#!/bin/bash
set -euo pipefail

# NHS monitor-check — weekly job that re-crawls every monitored domain,
# diffs the agentic-readiness score/signals against the last check, and
# emails the watcher when something regressed. Runs inside the Fly machine
# so the crawl goes out from the same IP/user-agent as the daily recrawl.

export PATH="/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin"
export HOME="/Users/owlassist"
FLY_BIN="/opt/homebrew/bin/fly"

APP_DIR="$(cd "$(dirname "$0")/.." && pwd)"
LOG_FILE="${APP_DIR}/tools/monitor-check.log"

echo "$(date '+%Y-%m-%d %H:%M:%S') NHS monitor-check starting" >> "$LOG_FILE"

cd "$APP_DIR"

REMOTE_COMMAND="${NHS_MONITOR_CHECK_REMOTE_COMMAND:-/app/monitor-check -cutoff-hours 144 -limit 500}"

if [[ "${NHS_MONITOR_CHECK_DRY_RUN:-0}" == "1" ]]; then
  echo "$(date '+%Y-%m-%d %H:%M:%S') NHS monitor-check dry-run remote_command=${REMOTE_COMMAND// /_}" >> "$LOG_FILE"
  exit 0
fi

if ! /usr/bin/security find-generic-password -a foundry -s fly-api-token -w >/dev/null 2>&1; then
  echo "$(date '+%Y-%m-%d %H:%M:%S') NHS monitor-check skipped: fly token missing" >> "$LOG_FILE"
  exit 1
fi

env -i HOME="/Users/owlassist" PATH="$PATH" FLY_ACCESS_TOKEN="$(/usr/bin/security find-generic-password -a foundry -s fly-api-token -w)" \
  "$FLY_BIN" ssh console -a nothumansearch -C "$REMOTE_COMMAND" >> "$LOG_FILE" 2>&1

echo "$(date '+%Y-%m-%d %H:%M:%S') NHS monitor-check done" >> "$LOG_FILE"
