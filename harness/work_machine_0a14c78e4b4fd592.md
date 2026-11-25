# work_machine_0a14c78e4b4fd592

Aggregate-only closeout proof for the 2026-06-04 NHS full recrawl.

## Scope Guard

- No full recrawl started.
- No deploy performed.
- No browser or desktop automation used.
- No process environments inspected.
- No secrets read or printed.
- Proof is limited to wrapper totals and public aggregate category/stat counts.

## Evidence

`tools/recrawl-health.log`:

- Seed refresh completed at `2026-06-04 05:39:27` with `api_status=200`, `api_ok=1`, `workers=10`, `dry_run=0`.
- Full recrawl started at `2026-06-04 06:00:07`.
- Full recrawl completed at `2026-06-04 10:33:06` with `api_status=200`, `api_ok=1`, `workers=10`, `dry_run=0`.

`tools/seed-refresh.log`:

- Success: 476
- Failed: 7
- Total: 483

`tools/recrawl.log`:

- Success: 10043
- Failed: 386
- Total: 10429

Planner-captured public aggregate snapshot for this work item:

- total_sites: 4269
- avg_score: 36
- top_category: developer
- category counts cited only at aggregate level: developer 1310, ai-tools 921, other 793, data 393, finance 186, productivity 171, ecommerce 150, communication 121.

The local runner could not resolve `nothumansearch.ai`, so live `/api/v1/stats` and `/api/v1/categories` were not re-probed here.

## Closeout

Closed as a fixed-point closeout. No additional recrawl or bounded quality crawl is needed from this aggregate evidence. The completed closeout row was removed from `harness/generated-work-items.json`; the remaining rows are separate follow-up lanes.
