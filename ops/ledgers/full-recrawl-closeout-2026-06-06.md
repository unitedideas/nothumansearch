# NHS full-recrawl aggregate closeout - 2026-06-06

Recorded by: `business-planner-not-human-search`
Recorded at: 2026-06-07T01:07:53Z

## Wrapper evidence

- `tools/recrawl-health.log` records `seed_refresh` completion at 2026-06-06 05:39:30 with `api_status=200`.
- `tools/recrawl-health.log` records `full_recrawl` completion at 2026-06-06 10:21:26 with `api_status=200`.
- `tools/full-recrawl.lock` is absent.
- `tools/recrawl.log` records `Done. Success: 10093, Failed: 393, Total: 10486`.
- `tools/recrawl.log` records `NHS full_recrawl complete`.

## Public aggregate evidence

- `total_sites`: 4282
- `avg_score`: 37
- `top_category`: developer

Category aggregates:

- developer: 1320, avg_score 37
- ai-tools: 917, avg_score 41
- other: 796, avg_score 28
- data: 394, avg_score 32
- finance: 187, avg_score 41
- productivity: 172, avg_score 39
- ecommerce: 151, avg_score 42
- communication: 121, avg_score 39
- security: 108, avg_score 39
- health: 57, avg_score 41
- jobs: 26, avg_score 42
- education: 21, avg_score 49
- news: 11, avg_score 50
- spam: 1, avg_score 0

## Decision

The 2026-06-06 full-recrawl lane is closed from aggregate evidence. Do not start a replacement full recrawl for this proof. Deploy-gated product work can proceed after its own normal deploy checks.

Any quality follow-up should be bounded: recategorize-only, single-site crawl, or a small file crawl with explicit target evidence.
