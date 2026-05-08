#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

INPUT="${NHS_DISCOVERY_INPUT:-$ROOT/tools/discover.err}"
QUALITY_OUTPUT="${NHS_DISCOVERY_QUALITY_OUTPUT:-$ROOT/harness/discovery-quality-latest.json}"
QUARANTINE_OUTPUT="${NHS_DISCOVERY_QUARANTINE_OUTPUT:-$ROOT/harness/discovery-quarantine-latest.json}"
HISTORY_OUTPUT="${NHS_DISCOVERY_HISTORY_OUTPUT:-$ROOT/harness/discovery-quarantine-history.jsonl}"

python3 "$ROOT/tools/discovery-quality-report.py" \
  --input "$INPUT" \
  --output "$QUALITY_OUTPUT" >/dev/null

python3 "$ROOT/tools/discovery-quarantine-report.py" \
  --input "$QUALITY_OUTPUT" \
  --output "$QUARANTINE_OUTPUT" \
  --history-output "$HISTORY_OUTPUT" >/dev/null

python3 - "$QUARANTINE_OUTPUT" "$HISTORY_OUTPUT" <<'PY'
import json
import pathlib
import sys

report = json.loads(pathlib.Path(sys.argv[1]).read_text(encoding="utf-8"))
history_path = pathlib.Path(sys.argv[2])
quarantine = report["quarantine"]
guards = report["public_guards"]

print(
    "discovery_quality_refresh "
    f"hard_signal_rows={quarantine['hard_signal_rows']} "
    f"low_signal_rows={quarantine['low_signal_rows']} "
    f"category_other_low_signal={quarantine['category_other_low_signal']} "
    f"quarantine_active={str(quarantine['active']).lower()} "
    f"planner_priority={guards['planner_priority']} "
    f"history={history_path.relative_to(pathlib.Path.cwd())}"
)
PY
