#!/bin/sh
set -eu

cd "$(dirname "$0")/.."

PYTHONDONTWRITEBYTECODE=1 python3 -m unittest \
  tools/test-discovery-quality-report.py \
  tools/test-discovery-quarantine-report.py \
  tools/test-refresh-discovery-quality.py \
  tools/test-taxonomy-other-redacted-sample.py \
  tools/test-full-recrawl-closeout.py \
  tools/quality-gate-discovery-test.py

PYTHONDONTWRITEBYTECODE=1 python3 tools/quality-gate-discovery.py \
  --quarantine tools/fixtures/discovery-quarantine-ci.json \
  --repo-root .
