# WorkItem closeout: work_machine_ff7150ba53115627

Closed at: 2026-05-20T12:05:00Z

Policy: aggregate-only. This note does not include raw crawler domains, URLs, row IDs, names, descriptions, emails, tokens, process environments, private query logs, or private notes.

## Scope

QLimit worker closeout for the NHS full-recrawl boundary described by the 2026-05-17 stale-lock replacement. No replacement full recrawl was started, no deploy was performed, no process environments were inspected, no secrets were read or printed, no browser or desktop automation was used, no public action was taken, and no production data was deleted.

## Wrapper Evidence

- `tools/recrawl-health.log` records the 2026-05-16 full-recrawl start at `2026-05-16 06:00:08` with preflight `api_status=200`, `api_ok=1`, throttle reason `recent_health_guard_unhealthy`, `workers=2`, and `normal_workers=10`.
- `tools/recrawl-health.log` records stale-lock replacement at `2026-05-17 06:00:08` for prior `pid=36792`, `lock_age_seconds=86400`, `action=replace`.
- `tools/recrawl-health.log` records the replacement full-recrawl start at `2026-05-17 06:00:08`, preflight `api_status=200`, `api_ok=1`, `health_outcome action=full_pressure`, `workers=10`, and remote start.
- `tools/recrawl-health.log` records same-day completion at `2026-05-17 10:23:48`, with post-run `api_status=200`, `api_ok=1`, `workers=10`, and `dry_run=0`.
- `tools/recrawl.log` records `Done. Success: 9839, Failed: 379, Total: 10218` and `2026-05-17 10:23:48 NHS full_recrawl complete`.
- A later wrapper-safe full-recrawl completion exists at `2026-05-19 10:26:32` with `Success: 9830`, `Failed: 396`, `Total: 10226`, `workers=10`, `dry_run=0`, and post-run `api_status=200`, `api_ok=1`.
- Repo-local lock check found no remaining `tools/full-recrawl.lock` or `tools/recrawl.lock`, so no stale-lock clearing was performed.

## Closeout Values

- `full_recrawl_success=9839`
- `full_recrawl_failed=379`
- `full_recrawl_total=10218`
- `workers=10`
- `throttle_state=full_pressure`
- `post_recrawl_api_status=200`
- `post_recrawl_api_ok=1`
- `dry_run=0`

## Public Aggregate Snapshot

Direct shell refresh of `https://nothumansearch.ai/api/v1/stats` and `https://nothumansearch.ai/api/v1/categories` failed in this runtime with DNS resolution errors, so no shell-fetched values were inferred from those failed requests.

The public web fetch for `/api/v1/stats` succeeded at closeout time:

- `total_sites=4154`
- `avg_score=34`
- `top_category=other`

The public categories endpoint could not be refreshed from this runner. The latest planner-attached public aggregate category snapshot for the same boundary remains:

- `developer=1235 avg_score=34`
- `ai-tools=900 avg_score=40`
- `other=767 avg_score=27`
- `data=399 avg_score=32`
- `finance=199 avg_score=41`

## Decision

Fixed point for the full-recrawl closeout lane. The stale-lock boundary was resolved through the repo-local wrapper guard path, the same-day replacement recrawl completed with aggregate success/failure totals, no full-recrawl lock remains, and the next useful local work is the existing aggregate discovery-quality follow-up in `harness/generated-work-items.json`, not another full recrawl.
