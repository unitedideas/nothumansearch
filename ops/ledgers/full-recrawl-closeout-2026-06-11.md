# NHS Full Recrawl Closeout - 2026-06-11

Scope: aggregate-only closeout for the full recrawl started at `2026-06-11T13:00:08Z`. No recrawl, deploy, public action, private row inspection, process-environment inspection, or ad hoc lock deletion was used.

Wrapper evidence:

- `tools/recrawl-health.log`: `2026-06-11 06:00:08 wrapper=full-recrawl lock=full-recrawl.lock event=start phase=full_recrawl pid=78208`
- `tools/recrawl-health.log`: `2026-06-11 10:58:31 wrapper=full-recrawl lock=full-recrawl.lock event=completion phase=full_recrawl api_status=200 api_ok=1 workers=10 dry_run=0`
- `tools/full-recrawl.lock`: absent at closeout.

Crawler aggregate:

- `success=10065`
- `failed=440`
- `total=10505`
- Completion line: `2026-06-11 10:58:31 NHS full_recrawl complete`

Live public aggregate after closeout:

- `total_sites=4174`
- `avg_score=35`
- `top_category=developer`
- Source: `https://nothumansearch.ai/api/v1/stats` via web fetch.

Category aggregate probe:

- Local shell DNS could not resolve `nothumansearch.ai`, and the web fetch path returned no readable payload for `/api/v1/categories` in this worker runtime.
- Last planner public category aggregate during the run: `developer=1310 avg_score=37`, `ai-tools=913 avg_score=41`, `other=806 avg_score=28`, `data=396 avg_score=32`, `finance=190 avg_score=41`, `productivity=173 avg_score=39`, `ecommerce=154 avg_score=42`, `communication=122 avg_score=39`.

API health:

- Wrapper postflight recorded `api_status=200 api_ok=1`.
- Direct local shell health probe could not resolve `nothumansearch.ai` in this worker runtime.

Worker verification:

- QLimit worker `work_machine_a9840d93f1a5932d` rechecked the aggregate-only boundary on 2026-06-11 after the wrapper completion line existed.
- Verification command: `GOCACHE=/Users/owlassist/foundry-businesses/nothumansearch/.gocache go test ./...` passed.

Decision:

- The 2026-06-11 full-recrawl boundary is closed by wrapper completion evidence.
- Deploy-gated work that was blocked only on this active recrawl boundary can proceed under its own deploy, public-action, and verification guardrails.
- Do not reopen recrawl work from this closeout; use targeted URL/file crawls, recategorize-only smokes, or public aggregate probes for later verification.
