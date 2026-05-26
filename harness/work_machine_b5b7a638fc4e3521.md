# Work machine b5b7a638fc4e3521

Date: 2026-05-25

Scope: close out the 2026-05-23 full-recrawl boundary from wrapper evidence only. No deploy was run, no full recrawl was started, no lock was cleared, no secrets were read, no process environments were inspected, and no browser or desktop automation was used.

## Boundary Evidence

- `tools/recrawl-health.log` records 2026-05-23 `full_recrawl` start at 06:00:08 with `api_status=200`, `api_ok=1`, `health_outcome action=full_pressure`, and `workers=10`.
- `tools/recrawl-health.log` records remote start for `/app/crawler -recrawl -workers 10` at 06:00:08.
- `tools/recrawl-health.log` records post-run health at 10:44:26 with `api_status=200` and `api_ok=1`.
- `tools/recrawl-health.log` records 2026-05-23 `full_recrawl` completion at 10:44:26 with `workers=10` and `dry_run=0`.
- `tools/recrawl.log` records 2026-05-23 `Done. Success: 9867, Failed: 379, Total: 10246` and `NHS full_recrawl complete`.
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

Direct live refresh of public aggregate endpoints from this runner failed with DNS resolution errors:

```text
curl -fsS https://nothumansearch.ai/api/v1/stats
curl: (6) Could not resolve host: nothumansearch.ai

curl -fsS https://nothumansearch.ai/api/v1/categories
curl: (6) Could not resolve host: nothumansearch.ai
```

No aggregate values were fabricated from the failed refresh. The delta below compares the WorkItem's planner-captured public aggregate snapshot at 2026-05-23T15:08:00Z with the latest planner-captured public aggregate snapshot already persisted in `harness/generated-work-items.json` at 2026-05-26T00:08:06Z.

- `total_sites`: 4180 -> 4172 (`-8`)
- `avg_score`: 35 -> 35 (`0`)
- `top_category`: developer -> developer
- `developer`: 1230 avg 34 -> 1230 avg 34 (`0`)
- `ai-tools`: 902 avg 41 -> 904 avg 41 (`+2`)
- `other`: 778 avg 27 -> 774 avg 27 (`-4`)
- `data`: 402 avg 32 -> 402 avg 32 (`0`)
- `finance`: 195 avg 41 -> 192 avg 40 (`-3`, avg `-1`)
- `productivity`: 172 avg 39 -> 171 avg 39 (`-1`)
- `ecommerce`: 148 avg 41 -> 149 avg 41 (`+1`)
- `communication`: 119 avg 38 -> 118 avg 38 (`-1`)
- `security`: 115 avg 39 -> 113 avg 39 (`-2`)
- `health`: 58 avg 42 -> 59 avg 42 (`+1`)
- `jobs`: 27 avg 41 -> 26 avg 41 (`-1`)
- `education`: 21 avg 49 -> 21 avg 49 (`0`)
- `news`: 12 avg 50 -> 12 avg 50 (`0`)
- `spam`: 1 avg 0 -> 1 avg 0 (`0`)

## Decision

The 2026-05-23 full-recrawl boundary is closed. A same-day wrapper completion line exists, the success/failure/total counts are present in `tools/recrawl.log`, the post-run aggregate health check passed, and no repo-local full-recrawl lock remains.

No new WorkItem was added to `harness/generated-work-items.json` for this lane because the closeout is at a fixed point. The queue already contains the remaining unrelated follow-up lanes for API-key commerce deploy verification and private score-fix cleanup.

Future stale locks or missing completion signals must follow `tools/recrawl-common.sh` and the repo-local recrawl guard/runbook path after recording aggregate stale-lock evidence only. Do not deploy, start a replacement full recrawl, clear a lock, or inspect process environments as closeout work.

## Verification

- `grep -n "2026-05-23.*wrapper=full-recrawl\|2026/05/23.*Done\. Success\|2026-05-23.*NHS full_recrawl" tools/recrawl-health.log tools/recrawl.log`
- `test -f tools/full-recrawl.lock && stat ... || echo 'full-recrawl.lock absent'`
- `python3 -m json.tool harness/generated-work-items.json`
- `GOCACHE=/private/tmp/nothumansearch-gocache go test ./internal/... ./cmd/server ./cmd/crawler`

## Commit Status

Commit is blocked in this runner: `git add harness/work_machine_b5b7a638fc4e3521.md && git commit -m "Close May 23 recrawl boundary"` failed with `.git/index.lock: Operation not permitted`.
