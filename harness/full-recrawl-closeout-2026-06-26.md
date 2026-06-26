# NHS Full Recrawl Closeout - 2026-06-26

Scope: aggregate-only closeout for the full recrawl started at `2026-06-26T13:00:05Z`. No deploy, broad crawl, public action, private row inspection, process-environment inspection, or ad hoc lock deletion was used.

Closeout check: `2026-06-26T17:19:39Z`.

Wrapper evidence:

- `tools/recrawl-health.log`: `2026-06-26 06:00:05 wrapper=full-recrawl lock=full-recrawl.lock event=start phase=full_recrawl pid=19302`
- `tools/recrawl-health.log`: `2026-06-26 10:17:34 wrapper=full-recrawl lock=full-recrawl.lock event=health_check phase=post_full_recrawl api_status=200 api_ok=1`
- `tools/recrawl-health.log`: `2026-06-26 10:17:34 wrapper=full-recrawl lock=full-recrawl.lock event=completion phase=full_recrawl api_status=200 api_ok=1 workers=10 dry_run=0`
- `tools/full-recrawl.lock`: absent at closeout.

Crawler aggregate:

- `success=10135`
- `failed=394`
- `total=10529`
- Completion line: `2026-06-26 10:17:34 NHS full_recrawl complete`

Live public aggregate after closeout:

- `total_sites=4174`
- `avg_score=35`
- `top_category=developer`
- Source: `https://nothumansearch.ai/api/v1/stats` via public fetch.

Category aggregate probe:

- Shell probes for `https://nothumansearch.ai/api/v1/categories` failed in this runner with DNS resolution errors.
- Repo-local Fly helper was unavailable in this runner (`nhs_fly_ssh` exited 97), so no remote fallback was used.
- Latest planner public category aggregate during the same active recrawl boundary: `developer=1313`, `ai-tools=923`, `other=799`, `data=396`, `finance=196`, `productivity=173`, `ecommerce=153`, `communication=123`, `security=110`, `health=60`, `jobs=25`, `education=21`, `news=10`, `spam=1`.
- No post-recrawl category count is claimed beyond the wrapper's postflight `api_status=200 api_ok=1`.

Decision:

- The 2026-06-26 full-recrawl boundary is closed by wrapper completion evidence and lock absence.
- Commerce/admin deploy work blocked only on this active recrawl boundary may proceed under its own deploy, public-action, and verification guardrails.
- Do not reopen recrawl work from this closeout; use targeted URL/file crawls, recategorize-only smokes, or public aggregate probes for later verification.
