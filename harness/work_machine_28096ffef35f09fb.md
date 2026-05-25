# QLimit closeout - work_machine_28096ffef35f09fb

Aggregate-only closeout for the 2026-05-23 full-recrawl boundary and 2026-05-24 seed-refresh aggregate.

No full recrawl, broad crawl, deploy, browser automation, desktop automation, public action, production data deletion, process-environment inspection, or secret read was performed.

## Inputs

- `tools/recrawl-health.log` records the 2026-05-23 full-recrawl completion at `2026-05-23 10:44:26` with `api_status=200`, `api_ok=1`, `workers=10`, and `dry_run=0`.
- `tools/recrawl.log` records the completed full-recrawl aggregate: `Success=9867`, `Failed=379`, `Total=10246`.
- `tools/recrawl-health.log` records the 2026-05-24 seed-refresh completion at `2026-05-24 05:39:06` with `api_status=200`, `api_ok=1`, `workers=10`, and `dry_run=0`.
- `tools/seed-refresh.log` records the completed seed-refresh aggregate: `Success=477`, `Failed=6`, `Total=483`.

## Refreshed Artifacts

`NHS_DISCOVERY_INPUT=tools/seed-refresh.log ./tools/refresh-discovery-quality.sh` refreshed and gate-checked:

- `harness/discovery-quality-latest.json`
- `harness/discovery-quarantine-latest.json`
- `harness/discovery-quarantine-history.jsonl`

The refresh was idempotent against the checked-in aggregate artifacts.

## Aggregate State

- `sample_rows=5653`
- `hard_signal_rows=4096`
- `low_signal_rows=1557`
- `hard_signal_rate=0.7246`
- `category_other=143`
- `category_other_hard_agent_signal=35`
- `category_other_low_signal=108`
- `llms_only=279`
- `schema_only=286`
- `zero_score=620`
- `passive_or_soft_signal=372`

The hard-signal `category=other` review remains aggregate-only: `rows=35`, all in score bucket `0_24`, with top signal sets `API=24` and `API,schema.org=11`.

## Decision

Current `category=other` state remains a no-op fixed point for this lane.

- Taxonomy-rule change: `none`
- Threshold adjustment: `none`
- `category_other_low_signal`: `no_op_fixed_point`
- Taxonomy rule decision: `not_from_low_signal_aggregate_cohort`

Reason: low-signal `category=other` rows do not prove a hard agent signal and remain audit-only. Aggregate counts alone are not enough evidence for a narrow taxonomy rule. Any future taxonomy-rule work must come from hard-signal `category=other` evidence and must stay summarized before entering committed planner artifacts.

## Guard State

These cohorts remain audit-only unless a hard agent signal is proven:

- `llms_only`: `public_search=false`, `score_fix_targeting=false`
- `schema_only`: `public_search=false`, `score_fix_targeting=false`
- `zero_score`: `public_search=false`, `score_fix_targeting=false`
- `category_other_low_signal`: `public_search=false`, `score_fix_targeting=false`

Public search remains guarded by `models.AgentFirstFilter`. Score-fix targeting remains guarded by `HasHardAgentSignal`.

## Verification

- `NHS_DISCOVERY_INPUT=tools/seed-refresh.log ./tools/refresh-discovery-quality.sh` -> `hard_signal_rows=4096 low_signal_rows=1557 category_other_low_signal=108 quarantine_active=true planner_priority=quarantine_first`.
- `python3 -m unittest tools/test-discovery-quality-report.py tools/test-discovery-quarantine-report.py tools/quality-gate-discovery-test.py` -> OK, 17 tests.

No new follow-up WorkItem is warranted from this discovery-quality lane. Existing generated work items remain for unrelated API-key commerce/admin traffic and private score-fix cleanup work.
