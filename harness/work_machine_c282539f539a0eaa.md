# WorkItem closeout: work_machine_c282539f539a0eaa

Closed at: 2026-05-20T07:11:08Z

Policy: aggregate-only. This note does not include raw crawler domains, URLs, row IDs, names, descriptions, emails, tokens, process environments, private query logs, or private notes.

## Scope

QLimit worker closeout for the NHS full-recrawl boundary. No replacement full recrawl was started, no deploy was performed, no process environments were inspected, no secrets were read or printed, no browser or desktop automation was used, no public action was taken, and no production data was deleted.

## Wrapper Evidence

- `tools/recrawl-health.log` records the 2026-05-19 full-recrawl start at `2026-05-19 06:00:11`, preflight `api_status=200`, `api_ok=1`, `health_outcome action=full_pressure`, `workers=10`, and remote start.
- `tools/recrawl-health.log` records completion at `2026-05-19 10:26:32`, with post-run `api_status=200`, `api_ok=1`, `workers=10`, and `dry_run=0`.
- `tools/recrawl.log` records `Done. Success: 9830, Failed: 396, Total: 10226` and `2026-05-19 10:26:32 NHS full_recrawl complete`.
- Repo-local lock check found no remaining `tools/full-recrawl.lock` or `tools/recrawl.lock`, so no stale-lock clearing was performed.

## Closeout Values

- `full_recrawl_success=9830`
- `full_recrawl_failed=396`
- `full_recrawl_total=10226`
- `workers=10`
- `throttle_state=full_pressure`
- `post_recrawl_api_status=200`
- `post_recrawl_api_ok=1`
- `dry_run=0`

## Public Aggregate Snapshot

Direct shell refresh of `https://nothumansearch.ai/api/v1/stats` and `https://nothumansearch.ai/api/v1/categories` failed in this runtime with DNS resolution errors, so no newer public aggregate values were inferred from the failed refresh.

The latest planner-attached live public aggregate snapshot for this boundary, recorded at `2026-05-20T07:08:44Z`, is:

- `/api/v1/stats`: `total_sites=4175`, `avg_score=35`, `top_category=developer`.
- `/api/v1/categories`: `developer=1231`, `ai-tools=904`, `other=768 avg_score=27`, `spam=1`.

## Decision

Fixed point for the full-recrawl closeout lane. The active full-recrawl boundary is closed by same-day wrapper completion evidence, no full-recrawl lock remains, and the next useful local work is the existing aggregate discovery-quality follow-up in `harness/generated-work-items.json`, not another full recrawl.
