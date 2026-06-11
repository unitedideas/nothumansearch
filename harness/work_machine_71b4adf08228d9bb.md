# Discovery Quality Refresh: 2026-05-19 Recrawl

Work item: `work_machine_71b4adf08228d9bb`
Observed boundary: 2026-05-19 full recrawl completed at 10:26:32 local time.

## Aggregate Inputs

- `tools/recrawl-health.log`: preflight `api_status=200 api_ok=1`, `full_pressure`, `workers=10`, remote recrawl start, post-recrawl `api_status=200 api_ok=1`, completion `dry_run=0`.
- `tools/recrawl.log`: `Success=9830`, `Failed=396`, `Total=10226`, followed by `NHS full_recrawl complete`.
- No full recrawl, broad crawl, browser automation, public action, credentialed private read, or row-level fetch was started for this refresh.

## Refreshed Aggregate State

- `sample_rows`: 9696
- `hard_signal_rows`: 4069
- `low_signal_rows`: 5627
- `hard_signal_rate`: 0.4197
- `category_other_low_signal`: 2423
- `category_other_hard_agent_signal`: 741
- `llms_only`: 979
- `schema_only`: 532
- `zero_score`: 2160
- `planner_priority`: `quarantine_first`

## Decision

Post-recrawl `category=other` state is a no-op fixed point.

- `taxonomy_rule_change`: false
- `threshold_adjustment`: false
- `no_op_fixed_point`: true
- `taxonomy_rule`: `not_from_low_signal_aggregate_cohort`
- `threshold_adjustment`: `none`

The aggregate shows a large low-signal `category=other` cohort, but those rows do not prove a narrow taxonomy rule by themselves. The hard-signal `category=other` cohort remains executor-sample territory only; row-level examples must not enter planner artifacts.

## Guard State

The following cohorts remain audit-only unless a hard agent signal is proven:

- `llms_only`: `public_search=false`, `score_fix_targeting=false`
- `schema_only`: `public_search=false`, `score_fix_targeting=false`
- `zero_score`: `public_search=false`, `score_fix_targeting=false`
- `category_other_low_signal`: `public_search=false`, `score_fix_targeting=false`

Public search remains protected by `models.AgentFirstFilter`. Score-fix targeting remains gated on `HasHardAgentSignal`.

## Artifacts Refreshed

- `harness/discovery-quality-latest.json`
- `harness/discovery-quarantine-latest.json`
- `harness/discovery-quarantine-history.jsonl`

No generated follow-up was added because this lane is explicitly closed as a no-op fixed point. Future work should reopen it only if a targeted aggregate helper or executor-only redacted sampler changes the hard-signal evidence.
