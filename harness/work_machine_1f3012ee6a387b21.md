# 2026-05-21 full-recrawl boundary closeout

WorkItem: `work_machine_1f3012ee6a387b21`
Observed: `2026-06-09`

Scope: aggregate-only closeout for the 2026-05-21 full-recrawl boundary. No replacement full recrawl, deploy, lock clearing, public action, browser automation, desktop automation, production-data deletion, credential read, private row fetch, raw-domain output, raw-URL output, row-id output, token output, or process-environment inspection was performed.

## Boundary Decision

Decision: completed same day.

The 2026-05-21 boundary is no longer active and did not require stale-lock handling. The allowed evidence shows a matching wrapper completion and crawler completion after the original active-lock handoff.

## Wrapper Evidence

- `tools/recrawl-health.log`: `2026-05-21 05:42:37 wrapper=seed-refresh event=completion phase=seed_refresh api_status=200 api_ok=1 workers=10 dry_run=0`
- `tools/recrawl-health.log`: `2026-05-21 06:00:12 wrapper=full-recrawl event=start phase=full_recrawl pid=43375`
- `tools/recrawl-health.log`: `2026-05-21 06:00:12 wrapper=full-recrawl event=health_check phase=preflight api_status=200 api_ok=1`
- `tools/recrawl-health.log`: `2026-05-21 06:00:12 wrapper=full-recrawl event=health_outcome phase=preflight action=full_pressure workers=10`
- `tools/recrawl-health.log`: `2026-05-21 06:00:12 wrapper=full-recrawl event=remote_start phase=recrawl command=/app/crawler_-recrawl_-workers_10`
- `tools/recrawl-health.log`: `2026-05-21 10:25:47 wrapper=full-recrawl event=health_check phase=post_full_recrawl api_status=200 api_ok=1`
- `tools/recrawl-health.log`: `2026-05-21 10:25:48 wrapper=full-recrawl event=completion phase=full_recrawl api_status=200 api_ok=1 workers=10 dry_run=0`
- Repo-local lock check found no current `tools/full-recrawl.lock`, `tools/recrawl.lock`, `tools/seed-refresh.lock`, or `tools/*crawler*.lock` file.

## Crawler Aggregate

- `tools/recrawl.log`: `2026/05/21 17:25:47 Done. Success: 9847, Failed: 389, Total: 10236`
- `tools/recrawl.log`: `2026-05-21 10:25:48 NHS full_recrawl complete`

Aggregate totals:

- `full_recrawl_success=9847`
- `full_recrawl_failed=389`
- `full_recrawl_total=10236`

## Seed Refresh Aggregate

- `tools/seed-refresh.log`: `2026/05/21 12:42:36 Done. Success: 469, Failed: 14, Total: 483`
- `tools/seed-refresh.log`: `2026-05-21 05:42:37 NHS seed_refresh complete`

Aggregate totals:

- `seed_refresh_success=469`
- `seed_refresh_failed=14`
- `seed_refresh_total=483`

## Public Aggregate Counts

The planner handoff's same-day public aggregate snapshot at `2026-05-21T15:07Z` recorded:

- `total_sites=4174`
- `avg_score=35`
- `top_category=developer`
- `developer=1228 avg_score=34`
- `ai-tools=901 avg_score=41`
- `other=771 avg_score=27`
- `data=403 avg_score=32`
- `finance=194 avg_score=40`
- `productivity=174 avg_score=39`
- `ecommerce=149 avg_score=41`
- `communication=120 avg_score=38`
- `security=114 avg_score=38`
- `health=59 avg_score=42`
- `jobs=27 avg_score=41`
- `education=21 avg_score=49`
- `news=12 avg_score=50`
- `spam=1 avg_score=0`

During this closeout, the external fetch path returned live `/api/v1/stats` as `total_sites=4174`, `avg_score=35`, and `top_category=developer`. Local shell DNS resolution for `nothumansearch.ai` failed, so no new local `curl` category snapshot was trusted or committed.

## Discovery-Quality Follow-Up

The sanitized discovery-quality lane has already been refreshed from the completed 2026-05-21 boundary and is at a fixed point:

- `harness/work_machine_161c82dd11ac61f2.md`
- `harness/work_machine_1878125c43df2ad7.md`
- `harness/discovery-quality-latest.json`
- `harness/discovery-quarantine-latest.json`
- `harness/discovery-quarantine-history.jsonl`

Current discovery-quality decision remains `no_op_fixed_point`:

- `taxonomy_rule_change=false`
- `threshold_adjustment=false`
- `no_op_fixed_point=true`
- `llms_only`, `schema_only`, `zero_score`, and `category_other_low_signal` remain audit-only with `public_search=false` and `score_fix_targeting=false`.

No new `harness/generated-work-items.json` row is warranted for this lane because the completed boundary has already been converted into aggregate fixed-point proof. Existing generated rows are separate deploy-gated, credential-gated, or private-cleanup lanes and were left unchanged.

## Verification

- `sed`/`grep` confirmed the 2026-05-21 wrapper, seed-refresh, and full-recrawl completion lines.
- `find` confirmed no current repo-local recrawl, full-recrawl, seed-refresh, or crawler lock file.
- External public `/api/v1/stats` fetch returned live aggregate stats; local shell `curl` could not resolve `nothumansearch.ai`.
- `python3 -m json.tool harness/generated-work-items.json`
- `GOCACHE=/private/tmp/nothumansearch-go-cache go test ./...`
