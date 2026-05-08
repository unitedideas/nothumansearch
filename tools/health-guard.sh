#!/bin/bash
# NHS health-guard - runs every 10 minutes.
# Checks public API + Fly Postgres state. If DB is in error state, it restarts
# the DB machine with a cooldown. Recrawl locks are logged so crawler pressure
# and API brownouts can be correlated without inspecting process environments.

set -euo pipefail

export PATH="/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin"
export HOME="/Users/owlassist"

FLY_BIN="/opt/homebrew/bin/fly"
APP_DIR="$(cd "$(dirname "$0")/.." && pwd)"
LOG_FILE="${APP_DIR}/tools/health-guard.log"
STATE_FILE="${APP_DIR}/tools/health-guard.state"
RECRAWL_LOCK_DIRS=(
  "${APP_DIR}/tools/recrawl.lock"
  "${APP_DIR}/tools/full-recrawl.lock"
  "${APP_DIR}/tools/seed-refresh.lock"
)
RESTART_COOLDOWN=1800

log() {
  echo "$(date '+%Y-%m-%d %H:%M:%S') $*" >> "$LOG_FILE"
}

notify() {
  local msg="$1"
  local token channel

  log "NOTIFY: $msg"
  token=$(security find-generic-password -a "foundry" -s "discord-bot-token" -w 2>/dev/null || true)
  if [[ -z "$token" ]]; then
    token=$(op item get z46ulot7jj7ztmeippu3n7xdjy --vault Foundry --fields "label=Owl Bot Discord Token" --reveal 2>/dev/null || true)
  fi
  channel=$(security find-generic-password -a "foundry" -s "discord-channel-id" -w 2>/dev/null || true)
  if [[ -z "$token" || -z "$channel" ]]; then
    log "  (no discord creds - skipping)"
    return
  fi
  curl -s -o /dev/null -X POST \
    -H "Authorization: Bot $token" \
    -H "Content-Type: application/json" \
    -H "User-Agent: FoundryBot/1.0" \
    -d "{\"content\":\"$msg\"}" \
    "https://discord.com/api/v10/channels/$channel/messages" || true
}

can_restart() {
  if [[ ! -f "$STATE_FILE" ]]; then
    return 0
  fi

  local last_restart now elapsed
  last_restart=$(cat "$STATE_FILE" 2>/dev/null || echo 0)
  now=$(date +%s)
  elapsed=$((now - last_restart))
  if (( elapsed < RESTART_COOLDOWN )); then
    log "  restart cooldown: ${elapsed}s < ${RESTART_COOLDOWN}s - skipping"
    return 1
  fi
  return 0
}

mark_restarted() {
  date +%s > "$STATE_FILE"
}

fly_cmd() {
  if ! /usr/bin/security find-generic-password -a foundry -s fly-api-token -w >/dev/null 2>&1; then
    log "fly_token_missing"
    return 1
  fi
  env -i HOME="/Users/owlassist" PATH="$PATH" FLY_ACCESS_TOKEN="$(/usr/bin/security find-generic-password -a foundry -s fly-api-token -w)" "$FLY_BIN" "$@"
}

recrawl_active() {
  local dir pid
  for dir in "${RECRAWL_LOCK_DIRS[@]}"; do
    pid=$(cat "${dir}/pid" 2>/dev/null || true)
    if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
      return 0
    fi
  done
  return 1
}

api_status=$(curl -s --max-time 20 -o /tmp/nhs-health.json -w "%{http_code}" "https://nothumansearch.ai/api/v1/stats" || true)
api_status="${api_status:-000}"
api_ok=0
if [[ "$api_status" == "200" ]]; then
  if python3 -c "import json,sys; json.loads(open('/tmp/nhs-health.json').read())['total_sites']" >/dev/null 2>&1; then
    api_ok=1
  fi
fi

db_status=$(fly_cmd status -a nothumansearch-db 2>/dev/null || true)
if echo "$db_status" | awk '/sjc/ && /error/ {found=1} END {exit !found}'; then
  db_state="error"
elif echo "$db_status" | awk '/sjc/ && /started/ {found=1} END {exit !found}'; then
  db_state="started"
elif echo "$db_status" | awk '/sjc/ && /standby|primary/ {found=1} END {exit !found}'; then
  db_state="started"
else
  db_state="unknown"
fi

recrawl_active_value=0
if recrawl_active; then
  recrawl_active_value=1
fi

log "api_status=$api_status api_ok=$api_ok db_state=$db_state recrawl_active=$recrawl_active_value"
if (( api_ok == 1 )) && [[ "$db_state" == "started" ]]; then
  exit 0
fi

if [[ "$db_state" == "error" ]]; then
  if can_restart; then
    notify "NHS Postgres in ERROR state - restarting DB machine (api_status=$api_status)"
    log "restarting DB..."
    machine_id=$(echo "$db_status" | awk '/sjc/ {print $1}' | head -1)
    if [[ -n "$machine_id" ]]; then
      if fly_cmd machine restart "$machine_id" -a nothumansearch-db >> "$LOG_FILE" 2>&1; then
        mark_restarted
        sleep 20
        new_status=$(curl -s --max-time 20 -o /dev/null -w "%{http_code}" "https://nothumansearch.ai/api/v1/stats" || true)
        new_status="${new_status:-000}"
        notify "NHS Postgres restarted. api_status=$new_status"
      else
        notify "NHS Postgres restart FAILED - needs manual attention"
      fi
    else
      notify "Could not find NHS Postgres machine id - manual action needed"
    fi
  fi
  exit 0
fi

if (( api_ok == 0 )); then
  if (( recrawl_active_value == 1 )); then
    log "API unhealthy while recrawl lock is active - recrawl wrapper should skip or throttle next phase"
  fi
  log "API unhealthy but db_state=$db_state - letting Fly autostart recover"
fi
