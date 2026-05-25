# Work machine 7393d01bb1274cad

Date: 2026-05-25

Scope: close out the 2026-05-25 full-recrawl run from wrapper evidence only. No new full recrawl was started, no deploy was run, and no process environment was inspected.

## Boundary Evidence

- `tools/recrawl-health.log` records 2026-05-25 `seed_refresh` completion at 05:39:36 with `api_status=200`, `api_ok=1`, `workers=10`, `dry_run=0`.
- `tools/recrawl-health.log` records 2026-05-25 `full_recrawl` start at 06:00:10 with `api_status=200`, `api_ok=1`, `workers=10`.
- `tools/recrawl-health.log` records 2026-05-25 `full_recrawl` completion at 10:39:19 with `api_status=200`, `api_ok=1`, `workers=10`, `dry_run=0`.
- `tools/recrawl.log` records 2026-05-25 `Done. Success: 9889, Failed: 383, Total: 10272` and `NHS full_recrawl complete`.
- The stale-lock path was not used. The wrapper released `tools/full-recrawl.lock` after completion.

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

The artifacts are aggregate-only. They do not include raw domains, URLs, row IDs, names, descriptions, emails, tokens, private notes, or crawler row output.

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
- `top_signal_sets`: `API=26`, `API,schema.org=12`

## Decision

`category=other` remains a no-op fixed point for the low-signal cohort.

- Taxonomy-rule change: `none` from this aggregate cohort. Low-signal rows lack hard agent signals and do not prove a narrow category rule.
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

The wrapper recorded aggregate API health at completion as `api_status=200 api_ok=1`.

Direct public API refresh from this runner failed for both live aggregate endpoints:

```text
curl -fsS https://nothumansearch.ai/api/v1/stats
curl: (6) Could not resolve host: nothumansearch.ai

curl -fsS https://nothumansearch.ai/api/v1/categories
curl: (6) Could not resolve host: nothumansearch.ai
```

No post-completion live stats or categories were fabricated from stale evidence. A follow-up is queued for a DNS-capable worker to capture only aggregate `/api/v1/stats` and `/api/v1/categories` output.

## Follow-Up Queue

`harness/generated-work-items.json` was updated with a DNS-capable live aggregate stats/categories capture follow-up. No new discovery-quality follow-up was added because that lane is at an explicit aggregate fixed point.

## Verification

- `NHS_DISCOVERY_INPUT=tools/seed-refresh.log ./tools/refresh-discovery-quality.sh`
- `python3 -m json.tool harness/discovery-quality-latest.json`
- `python3 -m json.tool harness/discovery-quarantine-latest.json`
- `python3 -m unittest tools/test-discovery-quality-report.py tools/test-discovery-quarantine-report.py tools/quality-gate-discovery-test.py`
- `GOCACHE=/private/tmp/nothumansearch-gocache go test ./internal/... ./cmd/server ./cmd/crawler`

## Commit Status

Commit is blocked in this runner: `git add harness/work_machine_7393d01bb1274cad.md harness/generated-work-items.json` failed with `.git/index.lock: Operation not permitted`.
