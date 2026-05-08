#!/bin/bash
set -euo pipefail

NHS_RECRAWL_WRAPPER_NAME="${NHS_RECRAWL_WRAPPER_NAME:-full-recrawl}"
NHS_RECRAWL_LOG_FILE="${NHS_RECRAWL_LOG_FILE:-$(cd "$(dirname "$0")/.." && pwd)/tools/recrawl.log}"
source "$(cd "$(dirname "$0")" && pwd)/recrawl-common.sh"

start_wrapper "full_recrawl"
run_remote "recrawl" "/app/crawler -recrawl -workers $workers"
finish_wrapper "full_recrawl"
run_indexnow
