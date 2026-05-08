#!/bin/bash
set -euo pipefail

NHS_RECRAWL_WRAPPER_NAME="${NHS_RECRAWL_WRAPPER_NAME:-seed-refresh}"
source "$(cd "$(dirname "$0")" && pwd)/recrawl-common.sh"

start_wrapper "seed_refresh"
run_remote "seed" "/app/crawler -seed -workers $workers"
finish_wrapper "seed_refresh"
