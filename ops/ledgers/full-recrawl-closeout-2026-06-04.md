# NHS Full Recrawl Closeout - 2026-06-04

Work item: `work_machine_0a14c78e4b4fd592`

## Boundary

Closeout only. No full recrawl, deploy, browser/desktop automation, secret reads, production data deletion, or process-environment inspection was performed.

## Wrapper Evidence

`tools/recrawl-health.log` records:

- `2026-06-04 05:39:27` seed refresh completion with `api_status=200`, `api_ok=1`, `workers=10`, `dry_run=0`.
- `2026-06-04 06:00:07` full recrawl start.
- `2026-06-04 10:33:06` full recrawl completion with `api_status=200`, `api_ok=1`, `workers=10`, `dry_run=0`.

`tools/seed-refresh.log` records the 2026-06-04 seed refresh aggregate total:

- Success: 476
- Failed: 7
- Total: 483

`tools/recrawl.log` records the 2026-06-04 full recrawl aggregate total:

- Success: 10043
- Failed: 386
- Total: 10429

## Public Aggregate Snapshot

The current runner could not resolve `nothumansearch.ai`, so `/api/v1/stats` and `/api/v1/categories` were not live-refreshed here. The planner-captured public aggregate snapshot for this same closeout item was:

- total_sites: 4269
- avg_score: 36
- top_category: developer
- developer: 1310, avg_score 38
- ai-tools: 921, avg_score 41
- other: 793, avg_score 28
- data: 393, avg_score 32
- finance: 186, avg_score 41
- productivity: 171, avg_score 39
- ecommerce: 150, avg_score 42
- communication: 121, avg_score 39

## Decision

This is closed as an aggregate-only recrawl closeout. The wrapper shows a completed full recrawl with healthy post-checks and stable aggregate totals. No stale-lock cleanup, replacement recrawl, deploy, or quality crawl is warranted from this evidence.

The only remaining follow-ups are separate lanes already retained in `harness/generated-work-items.json`: API-key commerce browser handoff/traffic buckets and credential-gated monitor quarantine review.
