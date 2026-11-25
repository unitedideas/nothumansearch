# Full-recrawl closeout — 2026-07-31

Automation: `business-planner-not-human-search`
Evidence boundary: aggregate wrapper and public product proof only; no raw
per-domain crawl rows, private records, secrets, deploys, or new crawls

## Wrapper result

- `tools/recrawl-health.log` records the full-recrawl start at 06:00:07 PT,
  successful preflight at HTTP 200, and remote start at 06:00:08 PT.
- The same wrapper records `post_full_recrawl` health at HTTP 200 and completion
  with `api_ok=1` at 09:59:13 PT.
- No full-recrawl, seed-refresh, or discovery lock remained at the 10:09 PT
  planner check.
- The prior July 30 run recorded completion with `api_status=000` and
  `api_ok=0`. Current `finish_wrapper` still performs one postflight probe,
  records completion regardless of health, and does not propagate the failed
  postflight as a nonzero result. The July 31 green closeout does not close that
  durability defect.

## Bounded public proof

- `https://nothumansearch.ai/health`: status `ok`, database `ok`.
- `https://nothumansearch.ai/api/v1/stats`: 4,343 total sites, average score 38,
  top category `developer`.
- `https://nothumansearch.ai/api/v1/categories`: developer 1,320 at average 38;
  other 850 at average 29; spam 1 at average 0.
- `https://nothumansearch.ai/mcp`: 11 tools; `find_mcp_servers` and
  `register_monitor` remain advertised.
- `/monitor`, `/score`, and `/report` returned HTTP 200. `/privacy`, `/terms`,
  and `/.well-known/agent-card.json` returned 404.

The active-boundary WorkItem is closed by this evidence. The remaining recrawl
work is bounded wrapper hardening: retry transient postflight health failures,
gate IndexNow on success, propagate exhausted failure to launchd/sync-state,
and always release the local run lock. Do not start another full recrawl to
verify that repair.
