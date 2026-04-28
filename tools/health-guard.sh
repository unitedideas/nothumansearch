#!/bin/bash
# NHS health-guard — runs every 10 minutes.
# Checks public API + Fly Postgres state; if DB is in error/critical state
# (e.g. after bulk-submit OOM), restarts the DB machine to clear it and
# sends a Discord alert. Idempotent — safe to run repeatedly.

set -euo pipefail

export PATH="/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin"
FLY_BIN="/opt/homebrew/bin/fly"

APP_DIR="$(cd "$(dirname "$0")/.." && pwd)"
LOG_FILE="${APP_DIR}/tools/health-guard.log"
STATE_FILE="${APP_DIR}/tools/health-guard.state"

# Only restart DB at most once per 30 minutes, to avoid restart loops.
RESTART_COOLDOWN=1800

log() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') $*" >> "$LOG_FILE"
}

notify() {
    local msg="$1"
    log "NOTIFY: $msg"
    local token channel
    token=$(security find-generic-password -a "foundry" -s "discord-bot-token" -w 2>/dev/null || true)
    if [[ -z "$token" ]]; then
        token=$(op item get z46ulot7jj7ztmeippu3n7xdjy --vault Foundry --fields "label=Owl Bot Discord Token" --reveal 2>/dev/null || true)
    fi
    channel=$(security find-generic-password -a "foundry" -s "discord-channel-id" -w 2>/dev/null || true)
    if [[ -z "$token" || -z "$channel" ]]; then
        log "  (no discord creds — skipping)"
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
        log "  restart cooldown: ${elapsed}s < ${RESTART_COOLDOWN}s — skipping"
        return 1
    fi
    return 0
}

mark_restarted() {
    date +%s > "$STATE_FILE"
}

fly_cmd() {
    env FLY_ACCESS_TOKEN="$(/usr/bin/security find-generic-password -a foundry -s fly-api-token -w)" \
        "$FLY_BIN" "$@"
}

# --- Check 1: public API responds with valid JSON ---
api_status=$(curl -s --max-time 20 -o /tmp/nhs-health.json -w "%{http_code}" "https://nothumansearch.ai/api/v1/stats" || echo "000")
api_ok=0
if [[ "$api_status" == "200" ]]; then
    if python3 -c "import json,sys; json.loads(open('/tmp/nhs-health.json').read())['total_sites']" >/dev/null 2>&1; then
        api_ok=1
    fi
fi

# --- Check 2: Fly DB role ---
# `fly status` for the DB app returns ROLE=primary|standby when healthy,
# ROLE=error when the cluster is stuck (e.g. post-OOM recovery loop).
db_role=$(fly_cmd status -a nothumansearch-db 2>/dev/null | awk '/started/ && /sjc/ {print $3}' | head -1)

log "api_status=$api_status api_ok=$api_ok db_role=$db_role"

# --- Decide action ---
if (( api_ok == 1 )) && [[ "$db_role" == "primary" || "$db_role" == "standby" ]]; then
    # Healthy. Nothing to do.
    exit 0
fi

if [[ "$db_role" == "error" ]]; then
    if can_restart; then
        notify "⚠️  NHS Postgres in ERROR state — restarting DB machine (api_status=$api_status)"
        log "restarting DB..."
        machine_id=$(fly_cmd status -a nothumansearch-db 2>/dev/null | awk '/started/ && /sjc/ {print $1}' | head -1)
        if [[ -n "$machine_id" ]]; then
            if fly_cmd machine restart "$machine_id" -a nothumansearch-db >> "$LOG_FILE" 2>&1; then
                mark_restarted
                sleep 20
                # Re-check
                new_status=$(curl -s --max-time 20 -o /dev/null -w "%{http_code}" "https://nothumansearch.ai/api/v1/stats" || echo "000")
                notify "✅ NHS Postgres restarted. api_status=$new_status"
            else
                notify "❌ NHS Postgres restart FAILED — needs manual attention"
            fi
        else
            notify "❌ Could not find NHS Postgres machine id — manual action needed"
        fi
    fi
    exit 0
fi

# API down but DB reports OK — app machines may be wedged.
if (( api_ok == 0 )); then
    log "API unhealthy but db_role=$db_role — letting Fly autostart recover"
fi
