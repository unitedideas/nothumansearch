# Not Human Search Harness

This directory is the business-local harness for `nothumansearch`.

Owns:
- Business plan
- Marketing plan
- Local agent registry
- Business-specific agent memory
- Runbooks
- Local ledgers

Shared QLimit owns reusable runtime, gates, validators, scheduler policy, resource locks, and common agents.

## Discovery Quality Refresh

Use `tools/refresh-discovery-quality.sh` to refresh local discovery-quality planning artifacts from `tools/discover.err` without running a crawl or mutating production rows.

The wrapper writes only aggregate artifacts:

- `harness/discovery-quality-latest.json`
- `harness/discovery-quarantine-latest.json`
- `harness/discovery-quarantine-history.jsonl`

The history file records weekly aggregate counts only: hard-signal rows, low-signal rows, `category_other_low_signal`, quarantine active state, and local planner priority. It must not include candidate domains, URLs, raw discovery log rows, or row identifiers. Trend history is only for business-local planner priority; it must not change public search ranking or score-fix targeting.

## Local harness verification

Run `tools/verify-harness-local.sh` from a checkout before publishing harness changes.
It exercises local fixture tests and the aggregate discovery gate. It does not
run discovery, recrawl production sites, or activate a scheduler.

Provider exchange is disabled, and production ingestion is held. Historical
ledgers and planning files do not authorize a new run. Scheduled work remains
with the existing shared-operations owner.
