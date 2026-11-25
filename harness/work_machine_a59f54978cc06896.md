# NHS discovery-quality quarantine closeout - 2026-06-11

Aggregate-only closeout for QLimit WorkItem `work_machine_a59f54978cc06896`.

No full recrawl, broad crawl, deploy, browser automation, desktop automation, public action, production data deletion, process-environment inspection, or secret read was performed.

## Inputs

- Source boundary cited by the WorkItem: `ops/ledgers/full-recrawl-closeout-2026-05-18.md`.
- Existing sanitized artifacts:
  - `harness/discovery-quality-latest.json`
  - `harness/discovery-quarantine-latest.json`
  - `harness/discovery-quarantine-history.jsonl`
- Refresh command: `NHS_DISCOVERY_INPUT=tools/seed-refresh.log ./tools/refresh-discovery-quality.sh`.
- Refresh output: `discovery_quality_refresh hard_signal_rows=9208 low_signal_rows=3537 category_other_low_signal=243 quarantine_active=true planner_priority=quarantine_first history=harness/discovery-quarantine-history.jsonl`.

The refresh used the bounded seed-refresh aggregate log only. It did not print or commit raw domains, URLs, row IDs, descriptions, emails, tokens, private notes, or crawler row output.

## Refreshed Discovery Aggregates

- `sample_rows=12745`
- `hard_signal_rows=9208`
- `low_signal_rows=3537`
- `hard_signal_rate=0.7225`
- `category_other=323`
- `category_other_hard_agent_signal=80`
- `category_other_low_signal=243`
- `llms_only=625`
- `schema_only=647`
- `zero_score=1390`
- `passive_or_soft_signal=875`

`hard_signal_other_review` remains aggregate-only:

- `rows=80`
- `score_bucket_0_24=80`
- `top_signal_sets`: `API=54`, `API,schema.org=26`

## Decision

Post-recrawl `category=other` state is a no-op fixed point for this lane.

- `taxonomy_rule_change=false`
- `threshold_adjustment=false`
- `no_op_fixed_point=true`
- `category_other_low_signal=no_op_fixed_point`
- `taxonomy_rule=not_from_low_signal_aggregate_cohort`
- `threshold_adjustment=none`

Reason: low-signal `category=other` rows lack hard agent signals by definition. Aggregate counts alone are not evidence for a narrow taxonomy rule. Hard-signal `category=other` rows remain aggregate-review-only until a bounded sampler proves a specific taxonomy rule without committing row-level evidence.

## Guard State

These cohorts remain audit-only unless a hard agent signal is proven:

- `llms_only`: `public_search=false`, `score_fix_targeting=false`
- `schema_only`: `public_search=false`, `score_fix_targeting=false`
- `zero_score`: `public_search=false`, `score_fix_targeting=false`
- `category_other_low_signal`: `public_search=false`, `score_fix_targeting=false`

Public search remains guarded by `models.AgentFirstFilter`. Score-fix targeting remains guarded by `HasHardAgentSignal`.

## Follow-up State

No new WorkItem was added from this lane. The business is at a true fixed point for low-signal discovery quarantine: keep the cohorts audit-only, refresh aggregate counts after bounded discovery, and do not promote row-level candidate work without hard agent-signal proof.
