# work_machine_9d39c59aaae98640

Result: complete.

## Aggregate Output

- Reused the existing aggregate-only 2026-05-13 full-recrawl closeout ledger.
- Added this WorkItem ID to the closeout ledger so QLimit can reconcile the duplicate planner intake.
- Re-verified wrapper completion, crawler completion summaries, and absence of repo-local recrawl lock files.
- Did not start a full recrawl, seed refresh, recategorize, deploy, public action, browser automation, or any process/environment inspection.

## Aggregate Counts

- `total_sites`: 4169
- `avg_score`: 35
- `category_other`: 767
- `spam`: 1
- `full_recrawl_success`: 9801
- `full_recrawl_failed`: 405
- `full_recrawl_total`: 10206
- `seed_refresh_success`: 475
- `seed_refresh_failed`: 8
- `seed_refresh_total`: 483

## Evidence Boundary

Allowed sources used:

- `tools/recrawl-health.log`
- `tools/recrawl.log`
- `tools/seed-refresh.log`
- repo-local lock-file check under `tools/`
- public aggregate API attempts for `/api/v1/stats` and `/api/v1/categories`

The public aggregate API attempts failed in this sandbox with DNS resolution errors for `nothumansearch.ai`, so no newer live aggregate values were inferred. The closeout uses the WorkItem aggregate snapshot plus local wrapper/log proof.

## Follow-Up

Keep `harness/generated-work-items.json` focused on the monitor quarantine reconciliation item. No replacement full-recrawl follow-up is warranted from this closeout.
