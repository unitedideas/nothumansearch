# work_machine_5a929e533cd4f588

Result: complete; commit blocked by `.git/index.lock` creation permissions in this sandbox.

## Aggregate Output

- Closed the 2026-05-19 full-recrawl boundary using wrapper logs and aggregate public snapshots only.
- Did not start another full recrawl, deploy, clear a lock, inspect process environments, read secrets, use browser/desktop automation, or take public action.
- Wrote aggregate-only closeout detail to `ops/ledgers/work_machine_5a929e533cd4f588.md`.
- Refreshed the existing discovery-quality follow-up row in `harness/generated-work-items.json`.

## Aggregate Counts

- `full_recrawl_success`: 9830
- `full_recrawl_failed`: 396
- `full_recrawl_total`: 10226
- `workers`: 10
- `throttle_state`: full_pressure
- `post_recrawl_api_status`: 200
- `post_recrawl_api_ok`: 1
- `dry_run`: 0
- `total_sites`: 4175
- `avg_score`: 35
- `top_category`: developer
- `category_developer`: 1231
- `category_ai_tools`: 904
- `category_other`: 768
- `spam`: 1

## Evidence Boundary

Allowed sources used:

- `tools/recrawl-health.log`
- `tools/recrawl.log`
- repo-local lock-file check for `tools/full-recrawl.lock`
- latest planner-captured public aggregate `/api/v1/stats` and `/api/v1/categories` snapshots

Direct public aggregate refresh attempts for `/api/v1/stats` and `/api/v1/categories` failed in this runtime with `curl: (6) Could not resolve host: nothumansearch.ai`, so no newer live values were inferred.

## Verification

- `GOCACHE=/private/tmp/nhs-go-cache go test ./...` was attempted. It failed on existing `cmd/monitor-check` test drift: undefined `firstCheckFailedQuarantineReason`, `firstCheckZeroScoreQuarantineReason`, and `firstCheckQuarantineReason`. Other packages that ran passed or had no test files.
- Staging/commit was attempted, but `git update-index`/`git add` failed with `fatal: Unable to create '/Users/owlassist/foundry-businesses/nothumansearch/.git/index.lock': Operation not permitted`.

## Follow-Up

The useful next WorkItem remains the existing generated row: `Refresh NHS discovery-quality aggregate after 2026-05-19 full recrawl`. No replacement full-recrawl follow-up is warranted from this closeout.
