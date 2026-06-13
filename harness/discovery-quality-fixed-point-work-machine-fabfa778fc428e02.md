# Discovery Quality Fixed Point - work_machine_fabfa778fc428e02

Aggregate-only closeout for QLimit WorkItem `work_machine_fabfa778fc428e02`.

No full recrawl, broad crawl, deploy, browser automation, desktop automation, public action, production data deletion, secret read, raw candidate export, or row-level sampler output was performed.

## Inputs

- Completed full-recrawl boundary used for the sanitized aggregate refresh:
  - local start: `2026-05-19 06:00:11`
  - local completion: `2026-05-19 10:26:32`
  - crawler-summary window: `2026/05/19 13:00:11` through `2026/05/19 17:26:32`
  - aggregate crawler result from the run log: `Success=9830`, `Failed=396`, `Total=10226`
- Refreshed artifacts:
  - `harness/discovery-quality-latest.json`
  - `harness/discovery-quarantine-latest.json`
  - `harness/discovery-quarantine-history.jsonl`

## Refreshed Aggregates

- `sample_rows=9827`
- `hard_signal_rows=4114`
- `low_signal_rows=5713`
- `hard_signal_rate=0.4186`
- `category_other_hard_agent_signal=752`
- `category_other_low_signal=2466`
- `llms_only=1004`
- `schema_only=538`
- `zero_score=2187`

## Decision

Post-recrawl `category=other` state is a no-op fixed point for this lane.

- `taxonomy_rule_change=false`
- `threshold_adjustment=false`
- `no_op_fixed_point=true`
- `category_other_low_signal=no_op_fixed_point`
- `taxonomy_rule=not_from_low_signal_aggregate_cohort`
- `threshold_adjustment=none`

Reason: low-signal `category=other`, `llms_only`, `schema_only`, and `zero_score` cohorts do not prove API, OpenAPI, MCP, or ai-plugin support. They remain audit-only with `public_search=false` and `score_fix_targeting=false`. Aggregate counts alone are not evidence for a taxonomy rule change or threshold adjustment.

`hard_signal_other_review` remains aggregate-only. Its low-score API-only and API+schema.org cohorts do not justify a taxonomy rule without a generalized private pattern.

## Follow-up State

`harness/generated-work-items.json` remains unchanged because this lane is at a true fixed point. Existing follow-ups already cover stale aggregate-source guarding and sampler failure diagnostics. No replacement full recrawl, broad crawl, public-search change, score-fix-targeting change, taxonomy-rule change, or threshold adjustment is warranted from this WorkItem.
