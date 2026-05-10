#!/bin/bash
set -euo pipefail

export PATH="${PATH:-/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin}"
export HOME="/Users/owlassist"

NHS_FLY_BIN="${NHS_FLY_BIN:-/opt/homebrew/bin/fly}"
NHS_FLY_CONFIG_DIR="${NHS_FLY_CONFIG_DIR:-/Users/owlassist/foundry-businesses/nothumansearch/tools/.fly-config}"
NHS_FLY_TOKEN_SERVICE="${NHS_FLY_TOKEN_SERVICE:-fly-api-token}"

nhs_fly_token_available() {
  /usr/bin/security find-generic-password -a foundry -s "$NHS_FLY_TOKEN_SERVICE" -w >/dev/null 2>&1
}

nhs_fly_cli_available() {
  [[ -x "$NHS_FLY_BIN" ]]
}

nhs_fly_unavailable_reason() {
  if ! nhs_fly_cli_available; then
    echo "fly_cli_missing"
    return
  fi
  if ! nhs_fly_token_available; then
    echo "fly_token_missing"
    return
  fi
  echo ""
}

nhs_fly_ssh() {
  local remote_command="$1"
  local reason
  reason="$(nhs_fly_unavailable_reason)"
  if [[ -n "$reason" ]]; then
    return 97
  fi
  mkdir -p "$NHS_FLY_CONFIG_DIR"
  env -i \
    HOME="/Users/owlassist" \
    PATH="$PATH" \
    FLY_CONFIG_DIR="$NHS_FLY_CONFIG_DIR" \
    FLY_ACCESS_TOKEN="$(/usr/bin/security find-generic-password -a foundry -s "$NHS_FLY_TOKEN_SERVICE" -w)" \
    "$NHS_FLY_BIN" ssh console -a nothumansearch -C "$remote_command"
}
