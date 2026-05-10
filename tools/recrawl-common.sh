#!/bin/bash
set -euo pipefail

export PATH="/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin"
export HOME="/Users/owlassist"

APP_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${APP_DIR}/tools/fly-auth.sh"
WRAPPER_NAME="${NHS_RECRAWL_WRAPPER_NAME:-$(basename "$0" .sh)}"
LOG_FILE="${NHS_RECRAWL_LOG_FILE:-${APP_DIR}/tools/${WRAPPER_NAME}.log}"
HEALTH_LOG="${NHS_RECRAWL_HEALTH_LOG:-${APP_DIR}/tools/recrawl-health.log}"
GUARD_LOG="${APP_DIR}/tools/health-guard.log"
LOCK_DIR="${NHS_RECRAWL_LOCK_DIR:-${APP_DIR}/tools/${WRAPPER_NAME}.lock}"
HEALTH_URL="${NHS_RECRAWL_HEALTH_URL:-https://nothumansearch.ai/api/v1/stats}"
HEALTH_FIXTURE="${NHS_RECRAWL_HEALTH_FIXTURE:-}"
FULL_WORKERS="${NHS_RECRAWL_WORKERS:-10}"
THROTTLED_WORKERS="${NHS_RECRAWL_THROTTLED_WORKERS:-2}"
STALE_LOCK_SECONDS="${NHS_RECRAWL_STALE_LOCK_SECONDS:-43200}"
DRY_RUN="${NHS_RECRAWL_DRY_RUN:-0}"

api_status="000"
api_ok="0"
workers="$FULL_WORKERS"

fly_ssh() {
  local reason
  reason="$(nhs_fly_unavailable_reason)"
  if [[ -n "$reason" ]]; then
    log_health "event=remote_skip reason=$reason"
    echo "$(date '+%Y-%m-%d %H:%M:%S') NHS $WRAPPER_NAME skipped: $reason" >> "$LOG_FILE"
    return 1
  fi
  nhs_fly_ssh "$1"
}

log_health() {
  echo "$(date '+%Y-%m-%d %H:%M:%S') wrapper=$WRAPPER_NAME lock=$(basename "$LOCK_DIR") $*" >> "$HEALTH_LOG"
}

check_api_health() {
  local tmp_file
  if [[ -n "$HEALTH_FIXTURE" ]]; then
    api_status="200"
    api_ok="0"
    if python3 -c "import json,sys; json.loads(open(sys.argv[1]).read())['total_sites']" "$HEALTH_FIXTURE" >/dev/null 2>&1; then
      api_ok="1"
    fi
    return
  fi

  tmp_file="$(mktemp /tmp/nhs-recrawl-health.XXXXXX)"
  api_status=$(curl -s --max-time 20 -o "$tmp_file" -w "%{http_code}" "$HEALTH_URL" || true)
  api_status="${api_status:-000}"
  api_ok="0"
  if [[ "$api_status" == "200" ]]; then
    if python3 -c "import json,sys; json.loads(open(sys.argv[1]).read())['total_sites']" "$tmp_file" >/dev/null 2>&1; then
      api_ok="1"
    fi
  fi
  rm -f "$tmp_file"
}

recent_guard_unhealthy() {
  [[ -f "$GUARD_LOG" ]] && tail -6 "$GUARD_LOG" | grep -q "api_ok=0"
}

acquire_lock() {
  if mkdir "$LOCK_DIR" 2>/dev/null; then
    echo "$$" > "${LOCK_DIR}/pid"
    date +%s > "${LOCK_DIR}/started_at"
    return 0
  fi

  local pid started now age
  pid="$(cat "${LOCK_DIR}/pid" 2>/dev/null || true)"
  started="$(cat "${LOCK_DIR}/started_at" 2>/dev/null || echo 0)"
  now="$(date +%s)"
  age=$((now - started))
  if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null && (( age < STALE_LOCK_SECONDS )); then
    log_health "event=skip phase=lock reason=already_running pid=$pid lock_age_seconds=$age"
    exit 0
  fi
  log_health "event=lock_stale pid=${pid:-unknown} lock_age_seconds=$age action=replace"
  rm -rf "$LOCK_DIR"
  mkdir "$LOCK_DIR"
  echo "$$" > "${LOCK_DIR}/pid"
  date +%s > "${LOCK_DIR}/started_at"
}

release_lock() {
  rm -rf "$LOCK_DIR"
}

select_workers() {
  workers="$FULL_WORKERS"
  if recent_guard_unhealthy; then
    workers="$THROTTLED_WORKERS"
    log_health "event=throttle phase=preflight reason=recent_health_guard_unhealthy workers=$workers normal_workers=$FULL_WORKERS"
  else
    log_health "event=health_outcome phase=preflight action=full_pressure workers=$workers"
  fi
}

preflight_or_skip() {
  local phase="$1"
  check_api_health
  log_health "event=health_check phase=preflight api_status=$api_status api_ok=$api_ok"
  if [[ "$api_ok" != "1" ]]; then
    log_health "event=skip phase=preflight reason=api_unhealthy api_status=$api_status api_ok=$api_ok"
    echo "$(date '+%Y-%m-%d %H:%M:%S') NHS $phase skipped: api unhealthy (status=$api_status ok=$api_ok)" >> "$LOG_FILE"
    exit 0
  fi
}

run_remote() {
  local phase="$1"
  local command="$2"
  log_health "event=remote_start phase=$phase command=${command// /_}"
  if [[ "$DRY_RUN" == "1" ]]; then
    log_health "event=remote_skip phase=$phase reason=dry_run command=${command// /_}"
    return 0
  fi
  fly_ssh "$command" >> "$LOG_FILE" 2>&1
}

run_indexnow() {
  if [[ "$DRY_RUN" == "1" ]]; then
    log_health "event=indexnow_skip reason=dry_run"
    return
  fi

  curl -s -X POST "https://api.indexnow.org/indexnow" -H "Content-Type: application/json" -d '{"host":"nothumansearch.ai","key":"bb1637af360f471ab2a1555d45d683ea","keyLocation":"https://nothumansearch.ai/bb1637af360f471ab2a1555d45d683ea.txt","urlList":["https://nothumansearch.ai/","https://nothumansearch.ai/about","https://nothumansearch.ai/sitemap.xml","https://nothumansearch.ai/llms.txt","https://nothumansearch.ai/llms-full.txt","https://nothumansearch.ai/openapi.yaml","https://nothumansearch.ai/api/v1","https://nothumansearch.ai/.well-known/ai-plugin.json","https://nothumansearch.ai/.well-known/mcp.json"]}' >> /dev/null 2>&1 || true
}

start_wrapper() {
  local phase="$1"
  acquire_lock
  trap release_lock EXIT
  echo "$(date '+%Y-%m-%d %H:%M:%S') NHS $phase starting" >> "$LOG_FILE"
  log_health "event=start phase=$phase pid=$$"
  cd "$APP_DIR"
  preflight_or_skip "$phase"
  select_workers
}

finish_wrapper() {
  local phase="$1"
  check_api_health
  log_health "event=health_check phase=post_${phase} api_status=$api_status api_ok=$api_ok"
  echo "$(date '+%Y-%m-%d %H:%M:%S') NHS $phase complete" >> "$LOG_FILE"
  echo "---" >> "$LOG_FILE"
  log_health "event=completion phase=$phase api_status=$api_status api_ok=$api_ok workers=$workers dry_run=$DRY_RUN"
}
