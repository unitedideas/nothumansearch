# NHS Full Recrawl Closeout - 2026-06-09

Scope: aggregate-only full-recrawl closeout for planner handoff. No product code, deploy, broad manual crawl, public action, private row inspection, or environment inspection was used.

Wrapper evidence:

- `tools/recrawl-health.log`: `2026-06-09 06:00:08 wrapper=full-recrawl event=start phase=full_recrawl pid=32652`
- `tools/recrawl-health.log`: `2026-06-09 10:49:35 wrapper=full-recrawl event=completion phase=full_recrawl api_status=200 api_ok=1 workers=10 dry_run=0`
- `tools/full-recrawl.lock`: absent at closeout.

Crawler aggregate:

- `success=10082`
- `failed=421`
- `total=10503`
- Completion line: `2026-06-09 10:49:35 NHS full_recrawl complete`

Seed refresh aggregate:

- `success=474`
- `failed=9`
- `total=483`
- Completion line: `2026-06-09 05:40:17 NHS seed_refresh complete`

Live public aggregate after closeout:

- `total_sites=4281`
- `avg_score=37`
- `top_category=developer`
- `developer=1310 avg_score=37`
- `ai-tools=912 avg_score=41`
- `other=803 avg_score=28`
- `spam=1 avg_score=0`

Decision:

- Active full-recrawl boundary is closed.
- Deploy-gated commerce/metadata repair work can proceed under its own deploy/public-action guardrails.
- Do not reopen recrawl work from this closeout; use targeted URL/file crawls, recategorize-only smokes, or public aggregate probes for later verification.
