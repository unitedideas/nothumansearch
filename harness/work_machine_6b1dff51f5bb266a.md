# WorkItem work_machine_6b1dff51f5bb266a

Aggregate-only closeout for the 2026-05-23 full-recrawl boundary plus latest seed-refresh aggregate.

No full recrawl, broad crawl, deploy, browser automation, desktop automation, public action, production data deletion, process-environment inspection, or secret read was performed.

## Inputs

- Full-recrawl wrapper boundary: `2026-05-23 06:00:08` start, `2026-05-23 10:44:26` completion, `api_status=200`, `api_ok=1`, `workers=10`, `dry_run=0`.
- Full-recrawl aggregate result: `Success=9867`, `Failed=379`, `Total=10246`.
- Latest seed-refresh wrapper boundary: `2026-05-24 05:39:06` completion, `api_status=200`, `api_ok=1`, `workers=10`, `dry_run=0`.
- Latest seed-refresh aggregate result: `Success=477`, `Failed=6`, `Total=483`.
- Refresh command: `NHS_DISCOVERY_INPUT=tools/seed-refresh.log ./tools/refresh-discovery-quality.sh`.
- Refresh output: `discovery_quality_refresh hard_signal_rows=4096 low_signal_rows=1557 category_other_low_signal=108 quarantine_active=true planner_priority=quarantine_first history=harness/discovery-quarantine-history.jsonl`.

## Refreshed Discovery Aggregates

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

`hard_signal_other_review` remains aggregate-only:

- `rows=35`
- `score_bucket_0_24=35`
- `top_signal_sets`: `API=24`, `API,schema.org=11`

## Decision

Current `category=other` state does not justify a taxonomy-rule change from the low-signal aggregate cohort.

- `category_other_low_signal`: `no_op_fixed_point`
- `taxonomy_rule`: `not_from_low_signal_aggregate_cohort`
- `threshold_adjustment`: `none`

Reason: the low-signal `category=other` cohort lacks hard agent signals by definition. It remains audit-only, and aggregate counts alone are not evidence for a narrow taxonomy rule.

## Guard State

The following cohorts remain audit-only unless a hard agent signal is proven:

- `llms_only`: `public_search=false`, `score_fix_targeting=false`
- `schema_only`: `public_search=false`, `score_fix_targeting=false`
- `zero_score`: `public_search=false`, `score_fix_targeting=false`
- `category_other_low_signal`: `public_search=false`, `score_fix_targeting=false`

Public search remains guarded by `models.AgentFirstFilter`. Score-fix targeting remains guarded by `HasHardAgentSignal`.

## Follow-up State

`harness/generated-work-items.json` is intentionally unchanged. This discovery-quality lane is at a true fixed point: no replacement full-recrawl, broad crawl, taxonomy-rule-change, threshold-adjustment, public-search, or score-fix-targeting follow-up is warranted from this aggregate evidence.

## Commit State

Commit is blocked in this runner: `git add -f ops/ledgers/discovery-quality-fixed-point-2026-05-25.md` failed with `.git/index.lock: Operation not permitted`.
