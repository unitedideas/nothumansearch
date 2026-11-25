# NHS discovery-quality aggregate closeout - 2026-06-13

Scope: aggregate-only refresh of sanitized discovery-quality and discovery-quarantine artifacts for WorkItem `work_machine_c5350c42eb1b3eca`.

No full recrawl, broad crawl, browser automation, raw domains, URLs, row IDs, descriptions, emails, tokens, private notes, or row-level candidate output were used or committed.

Refresh command:

```bash
NHS_DISCOVERY_INPUT=tools/seed-refresh.log ./tools/refresh-discovery-quality.sh
```

Refresh output:

```text
discovery_quality_refresh hard_signal_rows=9548 low_signal_rows=3670 category_other_low_signal=252 quarantine_active=true planner_priority=quarantine_first history=harness/discovery-quarantine-history.jsonl
```

The helper regenerated:

- `harness/discovery-quality-latest.json`
- `harness/discovery-quarantine-latest.json`
- `harness/discovery-quarantine-history.jsonl`

The regenerated JSON matched the existing worktree content, so no aggregate JSON diff was produced.

## Aggregate State

- `sample_rows`: 13218
- `hard_signal_rows`: 9548
- `low_signal_rows`: 3670
- `hard_signal_rate`: 0.7223
- `category_other`: 335
- `category_other_hard_agent_signal`: 83
- `category_other_low_signal`: 252
- `llms_only`: 649
- `schema_only`: 673
- `zero_score`: 1439

`hard_signal_other_review` remains aggregate-only:

- `rows`: 83
- `top_signal_sets`: `API=56`, `API,schema.org=27`
- `score_buckets`: `0_24=83`, `25_39=0`, `40_59=0`, `60_plus=0`

## Decision

The post-recrawl `category=other` state is a **no-op fixed point**, not a taxonomy-rule change and not a threshold adjustment.

Decision fields in `harness/discovery-quarantine-latest.json`:

- `post_recrawl_category_other_state=no_op_fixed_point`
- `category_other_low_signal=no_op_fixed_point`
- `taxonomy_rule=not_from_low_signal_aggregate_cohort`
- `threshold_adjustment=none`
- `decision_matrix.taxonomy_rule_change=false`
- `decision_matrix.threshold_adjustment=false`
- `decision_matrix.no_op_fixed_point=true`

Reason: aggregate `category=other` low-signal rows lack proven hard agent signals, and aggregate counts alone do not identify a narrow taxonomy rule. The hard-signal `category=other` cohort is still score-bucketed at `0_24`, so it remains a bounded private review surface rather than a public taxonomy or threshold change.

## Guard State

These cohorts remain audit-only unless a hard agent signal is proven:

- `llms_only`: `public_search=false`, `score_fix_targeting=false`
- `schema_only`: `public_search=false`, `score_fix_targeting=false`
- `zero_score`: `public_search=false`, `score_fix_targeting=false`
- `category_other_low_signal`: `public_search=false`, `score_fix_targeting=false`

`harness/generated-work-items.json` was intentionally left unchanged. This discovery-quality lane is at a fixed point from aggregate evidence; the existing generated queue already contains unrelated concrete follow-ups for commerce/admin traffic and credential-required monitor review.

## Verification

- `python3 -m json.tool harness/discovery-quality-latest.json`
- `python3 -m json.tool harness/discovery-quarantine-latest.json`
- `python3 -m json.tool harness/generated-work-items.json`
- `python3 tools/test-discovery-quality-report.py`
- `python3 tools/test-discovery-quarantine-report.py`
- `python3 tools/test-refresh-discovery-quality.py`
