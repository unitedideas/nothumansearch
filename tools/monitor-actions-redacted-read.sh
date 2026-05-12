#!/bin/sh
# Read aggregate monitor-admin action counts.
# The admin token is read inline from Keychain and is never printed.
set -eu

BASE_URL="${NHS_BASE_URL:-https://nothumansearch.ai}"
DAYS="${NHS_MONITOR_ACTION_DAYS:-30}"
SERVICE="${NHS_ADMIN_KEYCHAIN_SERVICE:-}"

if [ -z "$SERVICE" ]; then
  for candidate in nhs-admin-api-key nothumansearch-admin-key; do
    if /usr/bin/security find-generic-password -a foundry -s "$candidate" -w >/dev/null 2>&1; then
      SERVICE="$candidate"
      break
    fi
  done
fi

if [ -z "$SERVICE" ] || ! /usr/bin/security find-generic-password -a foundry -s "$SERVICE" -w >/dev/null 2>&1; then
  printf 'missing Keychain service: nhs-admin-api-key or nothumansearch-admin-key\n' >&2
  exit 2
fi

/usr/bin/curl -fsS \
  -H "Authorization: Bearer $(/usr/bin/security find-generic-password -a foundry -s "$SERVICE" -w)" \
  "$BASE_URL/api/v1/admin/monitors/actions?days=$DAYS"
