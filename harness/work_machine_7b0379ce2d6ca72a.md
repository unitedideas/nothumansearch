# Work machine 7b0379ce2d6ca72a

Date: 2026-05-25

Scope: sanitized discovery-quality and discovery-quarantine closeout from the completed 2026-05-23 full-recrawl boundary plus the latest seed-refresh aggregate.

## Boundary Evidence

- `tools/recrawl-health.log` records 2026-05-23 `full_recrawl` preflight and completion with `api_status=200`, `api_ok=1`, `workers=10`, `dry_run=0`.
- `tools/recrawl.log` records 2026-05-23 `Done. Success: 9867, Failed: 379, Total: 10246` and `NHS full_recrawl complete`.
- `tools/recrawl-health.log` records 2026-05-24 `seed_refresh` completion with `api_status=200`, `api_ok=1`, `workers=10`, `dry_run=0`.
- `tools/seed-refresh.log` records 2026-05-24 `Done. Success: 477, Failed: 6, Total: 483` and `NHS seed_refresh complete`.

## Aggregate Refresh

Command:

```sh
NHS_DISCOVERY_INPUT=tools/seed-refresh.log ./tools/refresh-discovery-quality.sh
```

Output:

```text
discovery_quality_refresh hard_signal_rows=4096 low_signal_rows=1557 category_other_low_signal=108 quarantine_active=true planner_priority=quarantine_first history=harness/discovery-quarantine-history.jsonl
```

Refreshed aggregate artifacts:

- `harness/discovery-quality-latest.json`
- `harness/discovery-quarantine-latest.json`
- `harness/discovery-quarantine-history.jsonl`

The artifacts stayed aggregate-only. No raw domains, URLs, row IDs, descriptions, emails, tokens, private notes, or crawler row output are included in this committed proof.

## Current Aggregate State

- `sample_rows=5653`
- `hard_signal_rows=4096`
- `low_signal_rows=1557`
- `hard_signal_rate=0.7246`
- `llms_only=279`
- `schema_only=286`
- `zero_score=620`
- `category_other=143`
- `category_other_low_signal=108`
- `category_other_hard_agent_signal=35`

`hard_signal_other_review` remains aggregate-only:

- `rows=35`
- `score_buckets`: `0_24=35`, `25_39=0`, `40_59=0`, `60_plus=0`
- `top_signal_sets`: `API=24`, `API,schema.org=11`

## Decision

`category=other` remains a no-op fixed point for the low-signal cohort.

- Taxonomy-rule change: `none` from this aggregate cohort. The low-signal rows do not prove a narrow category rule.
- Threshold adjustment: `none`. Current threshold keeps passive and zero-score rows audit-only without weakening hard-signal requirements.
- Fixed point: `no_op_fixed_point` for `category_other_low_signal=108`.

The hard-signal `category=other` aggregate stays reviewable, but the current score distribution is concentrated in the `0_24` bucket and does not justify a taxonomy change from aggregate evidence alone.

## Guard State

These cohorts remain audit-only unless a hard agent signal is proven:

- `llms_only`: `public_search=false`, `score_fix_targeting=false`
- `schema_only`: `public_search=false`, `score_fix_targeting=false`
- `zero_score`: `public_search=false`, `score_fix_targeting=false`
- `category_other_low_signal`: `public_search=false`, `score_fix_targeting=false`

Public search remains protected by `AgentFirstFilter`. Score-fix targeting remains gated on `HasHardAgentSignal`.

## Follow-Up Queue

No new discovery-quality follow-up was added to `harness/generated-work-items.json` because this lane is at an explicit aggregate fixed point. Existing unrelated generated items remain untouched.

## Verification

- `python3 -m json.tool harness/discovery-quality-latest.json`
- `python3 -m json.tool harness/discovery-quarantine-latest.json`
- `NHS_DISCOVERY_INPUT=tools/seed-refresh.log ./tools/refresh-discovery-quality.sh`
- `python3 -m unittest tools/test-discovery-quality-report.py tools/test-discovery-quarantine-report.py tools/quality-gate-discovery-test.py`
- `GOCACHE=/private/tmp/nothumansearch-gocache go test ./internal/... ./cmd/server ./cmd/crawler`

Repo-wide `GOCACHE=/private/tmp/nothumansearch-gocache go test ./...` remains blocked by existing `cmd/monitor-check` compile drift: tests reference `firstCheckFailedQuarantineReason`, `firstCheckZeroScoreQuarantineReason`, and `firstCheckQuarantineReason`.
