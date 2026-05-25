# Work machine 708ec26e0d4828b5

Date: 2026-05-25

Scope: close out the active 2026-05-25 full-recrawl boundary without starting a new recrawl, deploying, inspecting process environments, or publishing row-level data.

## Boundary Evidence

- `tools/recrawl-health.log` records 2026-05-25 `seed_refresh` completion at 05:39:36 with `api_status=200`, `api_ok=1`, `workers=10`, `dry_run=0`.
- `tools/recrawl-health.log` records 2026-05-25 `full_recrawl` start at 06:00:10 with `api_status=200`, `api_ok=1`, `workers=10`.
- `tools/recrawl-health.log` records 2026-05-25 `full_recrawl` completion at 10:39:19 with `api_status=200`, `api_ok=1`, `workers=10`, `dry_run=0`.
- `tools/recrawl.log` records 2026-05-25 `Done. Success: 9889, Failed: 383, Total: 10272` and `NHS full_recrawl complete`.
- No stale-lock path was needed. `tools/full-recrawl.lock` was already released by the wrapper after completion.

## Aggregate Refresh

Command:

```sh
NHS_DISCOVERY_INPUT=tools/seed-refresh.log ./tools/refresh-discovery-quality.sh
```

Output:

```text
discovery_quality_refresh hard_signal_rows=4440 low_signal_rows=1687 category_other_low_signal=117 quarantine_active=true planner_priority=quarantine_first history=harness/discovery-quarantine-history.jsonl
```

Refreshed aggregate artifacts:

- `harness/discovery-quality-latest.json`
- `harness/discovery-quarantine-latest.json`
- `harness/discovery-quarantine-history.jsonl`

The committed artifacts are aggregate-only. No raw domains, URLs, row IDs, names, descriptions, emails, tokens, private notes, or crawler row output are included.

## Current Aggregate State

- `sample_rows=6127`
- `hard_signal_rows=4440`
- `low_signal_rows=1687`
- `hard_signal_rate=0.7247`
- `llms_only=303`
- `schema_only=308`
- `zero_score=672`
- `passive_or_soft_signal=404`
- `category_other=155`
- `category_other_low_signal=117`
- `category_other_hard_agent_signal=38`

`hard_signal_other_review` remains aggregate-only:

- `rows=38`
- `score_buckets`: `0_24=38`, `25_39=0`, `40_59=0`, `60_plus=0`
- `top_signal_sets`: `API=27`, `API,schema.org=11`

## Decision

`category=other` remains a no-op fixed point for the low-signal cohort.

- Taxonomy-rule change: `none` from this aggregate cohort. Low-signal rows lack hard agent signals and do not prove a narrow taxonomy rule.
- Threshold adjustment: `none`. Current guards keep passive and zero-score rows audit-only without weakening hard-signal requirements.
- Fixed point: `no_op_fixed_point` for `category_other_low_signal=117`.

The hard-signal `category=other` aggregate remains reviewable, but the score distribution is entirely in the `0_24` bucket and does not justify a taxonomy change from aggregate evidence alone.

## Guard State

These cohorts remain audit-only:

- `llms_only`: `public_search=false`, `score_fix_targeting=false`
- `schema_only`: `public_search=false`, `score_fix_targeting=false`
- `zero_score`: `public_search=false`, `score_fix_targeting=false`
- `category_other_low_signal`: `public_search=false`, `score_fix_targeting=false`

Public search remains protected by `AgentFirstFilter`. Score-fix targeting remains gated on `HasHardAgentSignal`.

## Live API Boundary

The wrapper recorded aggregate API health at completion as `api_status=200 api_ok=1`. Direct public API refresh from this runner failed because DNS resolution for `nothumansearch.ai` returned `curl: (6) Could not resolve host: nothumansearch.ai`; no live stats/categories were fabricated from stale evidence.

## Follow-Up Queue

The active full-recrawl closeout item was removed from `harness/generated-work-items.json`. No new discovery-quality follow-up was added because this lane is at an explicit aggregate fixed point.

## Verification

- `NHS_DISCOVERY_INPUT=tools/seed-refresh.log ./tools/refresh-discovery-quality.sh`
- `python3 -m json.tool harness/discovery-quality-latest.json`
- `python3 -m json.tool harness/discovery-quarantine-latest.json`
