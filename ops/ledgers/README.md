# nothumansearch Ledgers

Local append-only ledgers for this business belong here. Cross-business public-action locks and shared automation leases remain in Foundry sync-state and QLimit shared state.

## Full recrawl closeout - 2026-06-28

Aggregate-only closeout for the full recrawl started at `2026-06-28T13:00:05Z`. No deploy, broad crawl, public action, private row inspection, process-environment inspection, or ad hoc lock deletion was used.

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
- Executor public probes for `https://nothumansearch.ai/api/v1/stats` and `https://nothumansearch.ai/api/v1/categories` failed from this runner with DNS resolution errors, so no new public aggregate body was recorded.
- Last pre-completion planner public stats from `2026-06-28T16:08:09Z`: `total_sites=4303`, `avg_score=37`, `top_category=developer`.
- Last pre-completion planner public categories from `2026-06-28T16:08:09Z`: `developer=1316`, `ai-tools=914`, `other=805`, `data=393`, `finance=197`, `productivity=171`, `ecommerce=154`, `communication=125`, `security=110`, `health=60`, `jobs=25`, `education=21`, `news=11`, `spam=1`.

Category drift note: fresh post-completion category bodies could not be fetched from this runner because DNS resolution failed. Against the latest completed public closeout on `2026-06-26`, the planner's pre-completion aggregate showed `other=805` versus `799` and `spam=1` versus `1`; no material drift was visible before completion.

Decision: the 2026-06-28 full-recrawl boundary is closed by wrapper completion evidence, wrapper public-health success, crawl totals, and lock absence. Work blocked only on this active recrawl boundary may proceed under its own deploy, public-action, and verification guardrails.

## Full recrawl closeout - 2026-07-12

Aggregate-only closeout for the full recrawl started at `2026-07-12T13:00:05Z`. No deploy, registry publish, broad crawl, private row inspection, process-environment inspection, public action, or ad hoc lock deletion was used.

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

Public aggregate:

- Wrapper post-recrawl health probe against `https://nothumansearch.ai/api/v1/stats` returned `api_status=200 api_ok=1`.
- Executor public probes for `https://nothumansearch.ai/api/v1/stats` and `https://nothumansearch.ai/api/v1/categories` failed from this runner with DNS resolution errors, so no new public aggregate body was recorded.
- Last planner public stats from `2026-07-12T15:08Z`: `total_sites=4307`, `avg_score=37`.
- Last planner public categories from `2026-07-12T15:08Z`: `developer=1314 avg_score=38`, `ai-tools=922 avg_score=43`, `other=803 avg_score=28`, `spam=1 avg_score=0`.

Bounded aggregate helper output:

- `tools/seed-refresh.log`: `2026/07/12 12:38:46 Done. Success: 477, Failed: 6, Total: 483`
- `tools/discover.log`: weekly discovery aggregate remained `Total unique candidates=3707`, `New domains to crawl=0`, `already_indexed=3707`.
- `tools/refresh-discovery-quality.sh`: `hard_signal_rows=18449`, `low_signal_rows=7061`, `category_other_low_signal=472`, `quarantine_active=true`, `planner_priority=quarantine_first`.
- `harness/discovery-quarantine-latest.json`: `no_op_fixed_point=true`, `taxonomy_rule_change=false`, `threshold_adjustment=false`; low-signal cohorts remain audit-only and out of score-fix targeting.

Decision: the 2026-07-12 full-recrawl boundary is closed by wrapper completion evidence, wrapper public-health success, crawl totals, and lock absence. Work previously blocked on this active recrawl may proceed under its own deploy, public-action, and verification guardrails.
