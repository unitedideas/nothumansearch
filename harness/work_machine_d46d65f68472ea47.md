# Work Machine d46d65f68472ea47

Status: closeout complete, commit blocked by git metadata permissions.

## Aggregate Closeout

Updated `ops/ledgers/full-recrawl-closeout-2026-06-04.md` for this WorkItem.

Evidence used:

- `tools/recrawl-health.log` records 2026-06-04 seed refresh completion at 05:39:27 with `api_status=200`, `api_ok=1`, `workers=10`, `dry_run=0`.
- `tools/seed-refresh.log` records seed refresh totals: Success 476, Failed 7, Total 483.
- `tools/recrawl-health.log` records 2026-06-04 full recrawl start at 06:00:07 and completion at 10:33:06 with `api_status=200`, `api_ok=1`, `workers=10`, `dry_run=0`.
- `tools/recrawl.log` records full recrawl totals: Success 10043, Failed 386, Total 10429.
- Live `/api/v1/stats` and `/api/v1/categories` refresh was attempted from this executor, but DNS failed with `curl: (6) Could not resolve host: nothumansearch.ai`; the ledger preserves the planner-captured public aggregate snapshot instead.

Decision: no replacement full recrawl, deploy, stale-lock action, process-environment inspection, or quality crawl is warranted. The lane is a fixed point from aggregate evidence. If future public aggregate category/stat probes show drift, the bounded follow-up remains recategorize-only or small targeted file crawl with aggregate-only proof.

`harness/generated-work-items.json` was left unchanged because the recrawl closeout lane is at a true fixed point and existing follow-ups are unrelated lanes.

## Verification

`GOCACHE=/private/tmp/nhs-go-cache go test ./...` passed.

## Commit Blocker

`git update-index --no-assume-unchanged ops/ledgers/full-recrawl-closeout-2026-06-04.md` failed:

`fatal: Unable to create '/Users/owlassist/foundry-businesses/nothumansearch/.git/index.lock': Operation not permitted`

Next action for a git-writable worker: clear assume-unchanged on `ops/ledgers/full-recrawl-closeout-2026-06-04.md`, add this note if desired, commit the closeout ledger refresh, and do not rerun the recrawl.
