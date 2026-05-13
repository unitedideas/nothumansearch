# work_machine_13867d0a4964e3c6

Result: complete and committed by the next `business-agent-not-human-search` run.

## Aggregate Output

- Extended `tools/discovery-quarantine-report.py` with `business_local_handoff`.
- Extended `tools/quality-gate-discovery.py` so review thresholds must include a bounded, non-public, non-domain-level handoff.
- Refreshed `harness/discovery-quarantine-latest.json` and `harness/discovery-quarantine-history.jsonl` from `tools/seed-refresh.log`.
- Added `ops/ledgers/seed-refresh-aggregate-handoff-2026-05-12.md` as the aggregate-only handoff ledger.
- Replaced the completed seed-refresh quality-gate generated WorkItem with bounded aggregate review follow-up in `harness/generated-work-items.json`.

## Current Aggregate Counts

- `sample_rows`: 1881
- `hard_signal_rows`: 1363
- `low_signal_rows`: 518
- `passive_only_share`: 0.2754
- `llms_only`: 92
- `schema_only`: 96
- `zero_score`: 207
- `category_other_low_signal`: 36
- `category_other_hard_agent_signal`: 12
- `quality_gate.status`: review
- `business_local_handoff.kind`: bounded_aggregate_review

## Guard State

Passive-only cohorts remain audit-only and excluded from public search and score-fix targeting. `models.AgentFirstFilter` remains hard-signal-only.

No full manual recrawl was run.

## Verification

- `python3 -m unittest tools/test-discovery-quality-report.py tools/test-discovery-quarantine-report.py tools/quality-gate-discovery-test.py` -> OK, 17 tests.
- `NHS_DISCOVERY_INPUT=tools/seed-refresh.log ./tools/refresh-discovery-quality.sh` -> `hard_signal_rows=1363 low_signal_rows=518 category_other_low_signal=36 quarantine_active=true planner_priority=quarantine_first`.
- `GOCACHE="$PWD/.gocache" go test ./...` -> OK.

## Commit Resolution

The follow-up run cleared assume-unchanged on the touched files, reran verification, and committed the aggregate handoff without printing row-level discovery data.
