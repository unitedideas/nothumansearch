# NHS full-recrawl boundary closeout - work_machine_cdc5874f0a82fd8c

Recorded: 2026-05-20T09:15:02Z

## Boundary decision

No new full recrawl was started. No deploy was run.

The repo-local wrapper evidence shows the older 2026-05-16 throttled full-recrawl boundary was superseded by a stale-lock replacement on 2026-05-17 and a later same-day full-recrawl completion on 2026-05-19. Current lock check found no `tools/full-recrawl.lock` and no `tools/recrawl.lock`, so no stale-lock cleanup was needed.

## Wrapper evidence

`tools/recrawl-health.log` records:

- 2026-05-16 06:00:08 full_recrawl preflight `api_status=200 api_ok=1`.
- 2026-05-16 06:00:08 throttled preflight `reason=recent_health_guard_unhealthy workers=2 normal_workers=10`.
- 2026-05-17 06:00:08 stale lock replacement for prior pid 36792 after `lock_age_seconds=86400`.
- 2026-05-17 10:23:48 full_recrawl completion with `api_status=200 api_ok=1 workers=10 dry_run=0`.
- 2026-05-19 06:00:11 full_recrawl preflight `api_status=200 api_ok=1`.
- 2026-05-19 06:00:11 full-pressure worker selection `workers=10`.
- 2026-05-19 10:26:32 full_recrawl completion with `api_status=200 api_ok=1 workers=10 dry_run=0`.

`tools/recrawl.log` records the latest full-recrawl aggregate completion:

- `Success: 9830`
- `Failed: 396`
- `Total: 10226`
- completion line: `2026-05-19 10:26:32 NHS full_recrawl complete`

## Public aggregate stats

Direct public aggregate refresh from this runner failed closed with DNS resolution errors:

- `/api/v1/stats`: `curl: (6) Could not resolve host: nothumansearch.ai`
- `/api/v1/categories`: `curl: (6) Could not resolve host: nothumansearch.ai`

No newer public aggregate values were inferred. The latest planner-captured public aggregate snapshot remains:

- `total_sites=4175`
- `avg_score=35`
- top category: `developer`
- selected category aggregates: `developer=1228`, `ai-tools=902`, `other=770 avg_score=27`, `spam=1`

## Follow-up state

`harness/generated-work-items.json` already carries the bounded next step: refresh sanitized discovery-quality and discovery-quarantine aggregates after the completed 2026-05-19 recrawl. That follow-up remains local/private and explicitly forbids raw domains, URLs, row IDs, descriptions, emails, tokens, private notes, full recrawls, or broad crawls.

## Verification

- `python3 -m json.tool harness/generated-work-items.json` passed.
- `go test ./...` did not pass because `cmd/monitor-check` tests still reference undefined helpers: `firstCheckFailedQuarantineReason`, `firstCheckZeroScoreQuarantineReason`, and `firstCheckQuarantineReason`.
- Commit creation was attempted, but git metadata writes are blocked in this runner: `fatal: Unable to create '/Users/owlassist/foundry-businesses/nothumansearch/.git/index.lock': Operation not permitted`.
