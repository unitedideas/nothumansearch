#!/bin/sh
# Read the geo-fix admin endpoint and emit aggregate-only redacted JSON.
# The admin token is read inline from Keychain and is never printed.
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
BASE_URL="${NHS_BASE_URL:-https://nothumansearch.ai}"
LIMIT="${NHS_GEO_JOBS_LIMIT:-500}"
SERVICE="${NHS_ADMIN_KEYCHAIN_SERVICE:-nhs-admin-api-key}"

if ! /usr/bin/security find-generic-password -a foundry -s "$SERVICE" -w >/dev/null 2>&1; then
  printf 'missing Keychain service: %s\n' "$SERVICE" >&2
  exit 2
fi

/usr/bin/curl -fsS \
  -H "Authorization: Bearer $(/usr/bin/security find-generic-password -a foundry -s "$SERVICE" -w)" \
  "$BASE_URL/api/v1/admin/geo-jobs?limit=$LIMIT" \
| python3 "$SCRIPT_DIR/redact-geo-jobs.py" --aggregate-only
