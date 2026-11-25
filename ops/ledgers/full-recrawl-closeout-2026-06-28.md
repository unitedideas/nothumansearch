# NHS Full Recrawl Closeout - 2026-06-28

Scope: aggregate-only closeout for the full recrawl started at `2026-06-28T13:00:05Z`. No deploy, broad crawl, public action, private row inspection, process-environment inspection, or ad hoc lock deletion was used.

Closeout checks:

- `2026-06-28T17:10:31Z`: QLimit work item created while the full recrawl was still active.
- `2026-06-28T17:13:37Z`: executor closeout after wrapper completion was present.
- `2026-06-28T17:17:26Z`: WorkItem aggregate snapshot recorded public stats/categories after wrapper completion but before this worker could resolve the public host.
- `2026-06-28T17:32Z`: executor re-checked wrapper completion, lock absence, crawler totals, and public probe availability; that runner could not resolve `nothumansearch.ai`.
- `2026-06-28T18:10Z`: planner worker re-probed public aggregate endpoints successfully.

Wrapper evidence:

- `tools/recrawl-health.log`: `2026-06-28 06:00:05 wrapper=full-recrawl lock=full-recrawl.lock event=start phase=full_recrawl pid=77700`
- `tools/recrawl-health.log`: `2026-06-28 10:11:04 wrapper=full-recrawl lock=full-recrawl.lock event=health_check phase=post_full_recrawl api_status=200 api_ok=1`
- `tools/recrawl-health.log`: `2026-06-28 10:11:04 wrapper=full-recrawl lock=full-recrawl.lock event=completion phase=full_recrawl api_status=200 api_ok=1 workers=10 dry_run=0`
- `tools/full-recrawl.lock`: absent at closeout.

Crawler aggregate:

- `success=10157`
- `failed=374`
- `total=10531`
- Completion line: `2026/06/28 17:11:03 Done. Success: 10157, Failed: 374, Total: 10531`

Public aggregate:

- Wrapper post-recrawl health probe against `https://nothumansearch.ai/api/v1/stats` returned `api_status=200 api_ok=1`.
- Executor public probes for `https://nothumansearch.ai/api/v1/stats` and `https://nothumansearch.ai/api/v1/categories` failed from that runner with DNS resolution errors, so it used WorkItem-carried public aggregate evidence.
- Planner public stats probe at `2026-06-28T18:10Z`: `total_sites=4303`, `avg_score=37`, `top_category=developer`.
- Planner public categories probe at `2026-06-28T18:10Z`: `developer=1316`, `ai-tools=916`, `other=806`, `data=391`, `finance=197`, `productivity=171`, `ecommerce=154`, `communication=124`, `security=110`, `health=60`, `jobs=25`, `education=21`, `news=11`, `spam=1`.

Category drift note:

- Fresh post-completion category bodies were fetched by the planner worker after the executor DNS failure.
- Against the latest completed public closeout on `2026-06-26`, the post-completion planner public aggregate showed `other=806` versus `799` and `spam=1` versus `1`; `other` rose by 7 and `spam` stayed flat.

Decision:

- The 2026-06-28 full-recrawl boundary is closed by wrapper completion evidence, wrapper public-health success, crawl totals, lock absence, and a successful later planner aggregate probe.
- Work blocked only on this active recrawl boundary may proceed under its own deploy, public-action, and verification guardrails.
