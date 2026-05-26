# Work machine c0740e2f9b251fc2

Date: 2026-05-26

Scope: close out the 2026-05-23 full-recrawl boundary from wrapper-safe aggregate evidence only. No deploy was run, no full recrawl was started, no lock was cleared, no secrets were read, no process environments were inspected, and no browser or desktop automation was used.

## Boundary Evidence

- `tools/recrawl-health.log` records the 2026-05-23 `full_recrawl` start at 06:00:08 with `api_status=200`, `api_ok=1`, `health_outcome action=full_pressure`, and `workers=10`.
- `tools/recrawl-health.log` records the 2026-05-23 remote start for `/app/crawler -recrawl -workers 10`.
- `tools/recrawl-health.log` records the 2026-05-23 post-run health check at 10:44:26 with `api_status=200` and `api_ok=1`.
- `tools/recrawl-health.log` records the 2026-05-23 completion at 10:44:26 with `workers=10` and `dry_run=0`.
- `tools/recrawl.log` records the matching 2026-05-23 completion totals: `Success: 9867`, `Failed: 379`, `Total: 10246`.
- Repo-local lock check found no remaining `tools/full-recrawl.lock`, so no stale-lock clearing was performed.

## Closeout Values

- `full_recrawl_success=9867`
- `full_recrawl_failed=379`
- `full_recrawl_total=10246`
- `workers=10`
- `throttle_state=full_pressure`
- `post_recrawl_api_status=200`
- `post_recrawl_api_ok=1`
- `dry_run=0`

## Aggregate Snapshot Delta

Direct live refresh of public aggregate endpoints from this runner failed with local DNS resolution errors. No aggregate values were fabricated from that failed refresh.

The delta below compares the WorkItem's planner-captured public aggregate snapshot at 2026-05-23T17:08:00Z with the latest planner-captured public aggregate snapshot already persisted in `harness/generated-work-items.json` at 2026-05-26T03:08:34Z.

- `total_sites`: 4173 -> 4172 (`-1`)
- `avg_score`: 35 -> 35 (`0`)
- `top_category`: developer -> developer
- `developer`: 1229 avg 34 -> 1230 avg 34 (`+1`)
- `ai-tools`: 902 avg 41 -> 904 avg 41 (`+2`)
- `other`: 779 avg 27 -> 774 avg 27 (`-5`)
- `data`: 399 avg 32 -> 402 avg 32 (`+3`)
- `finance`: 195 avg 41 -> 192 avg 40 (`-3`, avg `-1`)
- `productivity`: 172 avg 39 -> 171 avg 39 (`-1`)
- `ecommerce`: 146 avg 41 -> 149 avg 41 (`+3`)
- `communication`: 119 avg 38 -> 118 avg 38 (`-1`)
- `security`: 113 avg 39 -> 113 avg 39 (`0`)
- `health`: 59 avg 42 -> 59 avg 42 (`0`)
- `jobs`: 26 avg 41 -> 26 avg 41 (`0`)
- `education`: 21 avg 49 -> 21 avg 49 (`0`)
- `news`: 12 avg 50 -> 12 avg 50 (`0`)
- `spam`: 1 avg 0 -> 1 avg 0 (`0`)

## Decision

The 2026-05-23 full-recrawl boundary is closed. A same-day wrapper completion line exists, the success/failure/total counts are present in `tools/recrawl.log`, the post-run aggregate health check passed, and no repo-local full-recrawl lock remains.

No new recrawl WorkItem was added to `harness/generated-work-items.json` because this lane is at a fixed point. The queue already contains the remaining unrelated follow-up lanes for deploy-gated API-key commerce verification and private score-fix cleanup.

Future stale locks or missing completion signals must follow `tools/recrawl-common.sh` and the repo-local recrawl guard/runbook path after recording aggregate stale-lock evidence only. Do not deploy, start a replacement full recrawl, clear a lock, or inspect process environments as closeout work.

## Verification

- `grep -n "2026-05-23" tools/recrawl-health.log`
- `grep -n "2026/05/23" tools/recrawl.log`
- `stat` check for `tools/full-recrawl.lock`
- `python3 -m json.tool harness/generated-work-items.json`
- `GOCACHE=/private/tmp/nothumansearch-gocache go test ./...`

`harness/generated-work-items.json` validated as JSON. The full Go test command still fails only on the known adjacent `cmd/monitor-check` compile drift: undefined `firstCheckFailedQuarantineReason`, `firstCheckZeroScoreQuarantineReason`, and `firstCheckQuarantineReason`. That drift is unrelated to the recrawl closeout and does not reopen this lane.

## Commit Status

Commit is blocked in this runner: `git add harness/work_machine_c0740e2f9b251fc2.md && git commit -m "Close May 23 recrawl boundary"` failed because `.git/index.lock` could not be created: `Operation not permitted`.
