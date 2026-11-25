# NHS Full Recrawl Closeout - 2026-06-18

Scope: aggregate-only closeout for the full recrawl started at `2026-06-18T13:00:04Z`. No deploy, broad crawl, public action, private row inspection, process-environment inspection, or ad hoc lock deletion was used.

Wrapper evidence:

- `tools/recrawl-health.log`: `2026-06-18 06:00:04 wrapper=full-recrawl lock=full-recrawl.lock event=start phase=full_recrawl pid=79872`
- `tools/recrawl-health.log`: `2026-06-18 10:26:59 wrapper=full-recrawl lock=full-recrawl.lock event=completion phase=full_recrawl api_status=200 api_ok=1 workers=10 dry_run=0`
- `tools/full-recrawl.lock`: absent at closeout.

Crawler aggregate:

- `success=10141`
- `failed=378`
- `total=10519`
- Completion line: `2026-06-18 10:26:59 NHS full_recrawl complete`

Live public aggregate after closeout, re-verified at `2026-06-18T19:08:19Z`:

- `total_sites=4290`
- `avg_score=37`
- `top_category=developer`
- Source: `https://nothumansearch.ai/api/v1/stats`.

Category aggregate probe:

- `developer=1303 avg_score=38`
- `ai-tools=926 avg_score=42`
- `other=801 avg_score=28`
- `data=396 avg_score=32`
- `finance=191 avg_score=42`
- `productivity=171 avg_score=39`
- `ecommerce=153 avg_score=42`
- `communication=124 avg_score=39`

API health:

- Wrapper postflight recorded `api_status=200 api_ok=1`.
- Direct local public probes for `/api/v1/stats` and `/api/v1/categories` succeeded in the planner runtime.

Decision:

- The 2026-06-18 full-recrawl boundary is closed by wrapper completion evidence and lock absence.
- Deploy-gated commerce/admin work that was blocked only on this active recrawl boundary can proceed under its own deploy, public-action, and verification guardrails.
- Do not reopen recrawl work from this closeout; use targeted URL/file crawls, recategorize-only smokes, or public aggregate probes for later verification.
