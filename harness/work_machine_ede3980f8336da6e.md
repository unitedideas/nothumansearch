# Discovery quality fixed-point refresh

Work item: `work_machine_ede3980f8336da6e`
Observed at: `2026-05-26T07:14:03Z`

## Inputs

- Completed full-recrawl boundary: `tools/recrawl-health.log` records `2026-05-23 10:44:26` `full_recrawl` completion with `api_status=200`, `api_ok=1`, `workers=10`, `dry_run=0`.
- Latest aggregate seed-refresh input: `tools/seed-refresh.log`.
- Refresh command: `NHS_DISCOVERY_INPUT=tools/seed-refresh.log ./tools/refresh-discovery-quality.sh`.

## Refreshed aggregate state

- `sample_rows=6127`
- `hard_signal_rows=4440`
- `low_signal_rows=1687`
- `hard_signal_rate=0.7247`
- `category_other_low_signal=117`
- `category_other_hard_agent_signal=38`
- `llms_only=303`
- `schema_only=308`
- `zero_score=672`
- `passive_only_share=0.2753`
- `hard_signal_other_review.top_signal_sets`: `API=26`, `API,schema.org=12`

## Decision

`category=other` remains a no-op fixed point.

- Taxonomy-rule change: none.
- Threshold adjustment: none.
- Reason: the low-signal `category=other` cohort still lacks proven hard agent signals, and aggregate counts alone are not enough evidence for a narrow reusable taxonomy rule.

The hard-signal `category=other` review remains aggregate-only. Executor samples must not enter planner artifacts.

## Guards

The refreshed artifacts keep passive cohorts audit-only:

- `llms_only`: `public_search=false`, `score_fix_targeting=false`
- `schema_only`: `public_search=false`, `score_fix_targeting=false`
- `zero_score`: `public_search=false`, `score_fix_targeting=false`
- `category_other_low_signal`: `public_search=false`, `score_fix_targeting=false`

Public search remains protected by `models.AgentFirstFilter`; score-fix targeting still requires a hard agent signal.

## Follow-up queue

No new discovery-quality WorkItem was added. This lane is at a true fixed point unless later aggregate evidence proves a hard-signal pattern or breaks the guard contract.

Existing unrelated queue rows in `harness/generated-work-items.json` were left untouched.
