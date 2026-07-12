# NHS Full Recrawl Closeout - 2026-07-12

Scope: aggregate-only closeout for the full recrawl started at `2026-07-12T13:00:05Z`. No deploy, registry publish, broad crawl, public action, private row inspection, process-environment inspection, or ad hoc lock deletion was used.

Closeout checks:

- `2026-07-12T17:08:56Z`: QLimit work item created while the full-recrawl boundary lacked July 12 wrapper completion evidence in the planner snapshot.
- `2026-07-12T16:46:33Z`: repo-local wrapper logs now show full-recrawl completion.
- `2026-07-12T17:08:00Z`: existing generated follow-up queue had already been refreshed with metadata, commerce, and credential-gated monitor follow-ups tied to July 12 aggregate evidence.
- `2026-07-12T17:12:51Z`: executor re-checked wrapper completion, lock absence, crawler totals, bounded discovery-quality aggregate output, and public probe availability.

Wrapper evidence:

- `tools/recrawl-health.log`: `2026-07-12 06:00:05 wrapper=full-recrawl lock=full-recrawl.lock event=start phase=full_recrawl pid=33940`
- `tools/recrawl-health.log`: `2026-07-12 09:46:33 wrapper=full-recrawl lock=full-recrawl.lock event=health_check phase=post_full_recrawl api_status=200 api_ok=1`
- `tools/recrawl-health.log`: `2026-07-12 09:46:33 wrapper=full-recrawl lock=full-recrawl.lock event=completion phase=full_recrawl api_status=200 api_ok=1 workers=10 dry_run=0`
- `tools/full-recrawl.lock`: absent at closeout.

Crawler aggregate:

- `success=10220`
- `failed=338`
- `total=10558`
- Completion line: `2026/07/12 16:46:32 Done. Success: 10220, Failed: 338, Total: 10558`
- Wrapper closeout marker: `2026-07-12 09:46:33 NHS full_recrawl complete`

Seed refresh aggregate:

- Wrapper completion: `2026-07-12 05:38:47 wrapper=seed-refresh lock=seed-refresh.lock event=completion phase=seed_refresh api_status=200 api_ok=1 workers=10 dry_run=0`
- WorkItem-carried seed aggregate: `success=477`, `failed=6`, `total=483`

Public aggregate:

- Wrapper post-recrawl health probe against the public stats endpoint returned `api_status=200 api_ok=1`.
- This runner could not resolve the public host for direct stats/categories probes, so no fresh public response body was committed from this runner.
- WorkItem-carried public stats snapshot during the active boundary: `total_sites=4308`, `avg_score=37`, `top_category=developer`.
- WorkItem-carried public categories snapshot: `developer=1313`, `ai-tools=922`, `other=805`, `data=393`, `finance=200`, `productivity=164`, `ecommerce=159`, `communication=121`, `security=113`, `health=62`, `jobs=26`, `education=21`, `news=8`, `spam=1`.
- Existing generated follow-up row last verified at `2026-07-12T17:08:00Z` records a post-completion public aggregate snapshot with `total_sites=4307`, `avg_score=37`, `top_category=developer`, and `spam=1`.

Bounded helper aggregate:

- `tools/refresh-discovery-quality.sh`: `hard_signal_rows=18449`, `low_signal_rows=7061`, `category_other_low_signal=472`, `quarantine_active=true`, `planner_priority=quarantine_first`.
- `harness/discovery-quarantine-latest.json`: `category_other_hard_agent_signal=162`, `category_other_low_signal=472`, `llms_only=1268`, `schema_only=1304`, `zero_score=2669`.
- Decision remains `no_op_fixed_point`: low-signal and passive cohorts stay audit-only; no taxonomy rule or threshold adjustment is justified from aggregate evidence alone.

Follow-up rows:

- `harness/generated-work-items.json` contains three July 12 follow-ups: registry/count metadata refresh gated by public-action lock, API-key commerce/admin aggregate repair gated by deploy, and monitor quarantine review gated by credentials.
- Those rows are aggregate-only and exclude raw domains, URLs, row IDs, emails, payment IDs, tokens, process environments, and discovery candidate domains.

Decision:

- The 2026-07-12 full-recrawl boundary is closed by wrapper completion evidence, wrapper public-health success, crawl totals, lock absence, and refreshed bounded aggregate helper output.
- Work blocked only on this active full-recrawl boundary may proceed under its own deploy, public-action, credential, and verification guardrails.
- Registry metadata publishing, deploy work, and any crawl work remain outside this closeout item.
