# WorkItem closeout: work_machine_6ff84c9bb5c43bc3

Repo-local evidence file for the 2026-05-20 NHS full-recrawl boundary closeout.

## Outcome

- No replacement full recrawl was started.
- No deploy was performed.
- No process environments were inspected.
- No secrets were read or printed.
- No public action, browser automation, desktop automation, or production data mutation was performed.
- `tools/full-recrawl.lock` was not present, so no stale-lock clearing was performed.

## Aggregate Evidence

- 2026-05-16 full-recrawl started at `06:00:08` with preflight `api_status=200`, `api_ok=1`, throttle `reason=recent_health_guard_unhealthy`, `workers=2`, and `normal_workers=10`.
- No 2026-05-16 same-day `full_recrawl complete` line was present in inspected wrapper evidence.
- Later wrapper progress superseded the stale 2026-05-16 boundary: 2026-05-19 full-recrawl started at `06:00:11`, ran at full pressure with `workers=10`, and completed at `10:26:32`.
- `tools/recrawl-health.log` records post-run `api_status=200`, `api_ok=1`, `workers=10`, and `dry_run=0`.
- `tools/recrawl.log` records `Done. Success: 9830, Failed: 396, Total: 10226` and `2026-05-19 10:26:32 NHS full_recrawl complete`.
- `tools/recrawl.log` and `tools/recrawl-health.log` do not contain a 2026-05-20 same-day `full_recrawl complete` line.

## Public Aggregate Snapshot

Direct shell refresh of `https://nothumansearch.ai/api/v1/stats` and `/api/v1/categories` returned DNS resolution failure in this runtime, so no newer aggregate values were inferred.

Latest planner-captured public aggregate values for this boundary:

- `total_sites=4175`
- `avg_score=35`
- `top_category=developer`
- `developer=1231 avg_score=34`
- `ai-tools=904 avg_score=41`
- `other=768 avg_score=27`
- `data=400 avg_score=32`
- `finance=196 avg_score=40`

## Follow-up State

`harness/generated-work-items.json` already tracks the useful next action as aggregate discovery-quality refresh after the completed 2026-05-19 recrawl. This closeout did not create a new recrawl/deploy task because the full-recrawl boundary itself is at a fixed point.

## Commit Blocker

Commit creation was attempted but this runtime could not create `.git/index.lock`:

`fatal: Unable to create '/Users/owlassist/foundry-businesses/nothumansearch/.git/index.lock': Operation not permitted`

The repo-local state artifact is left here for the next credentialed or writable-git worker to commit.
