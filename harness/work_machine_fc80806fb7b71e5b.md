# API-key commerce handoff local evidence

WorkItem: `work_machine_fc80806fb7b71e5b`
Date: 2026-05-14

## Completed locally

- Added a shared API-plan summary used by the GET `/api/v1/api-keys/subscribe` handoff.
- Added starter, pro, and scale plan summaries to `/.well-known/agent.json`.
- Extended quota-exceeded payloads with anonymous/API-key subject, catalog URL, quote URL, and available plan IDs.
- Added live-safe tests for:
  - GET subscribe handoff plans.
  - POST subscribe contract when Stripe is not configured, without any checkout URL.
  - Anonymous quota-exceeded handoff payload without any checkout URL.
  - Catalog and quote coverage for starter, pro, and scale.
  - Agent manifest API-key plan exposure.
- Updated `harness/generated-work-items.json` to replace this completed local item with a deploy-gated production smoke follow-up.

## Verification

- `GOCACHE=$PWD/.gocache go test ./internal/handlers ./internal/models ./internal/database` passed.
- `GOCACHE=$PWD/.gocache go test ./...` was attempted and failed before touched packages because `cmd/monitor-check/main_test.go` references missing symbols:
  - `firstCheckFailedQuarantineReason`
  - `firstCheckZeroScoreQuarantineReason`
  - `firstCheckQuarantineReason`

## Commit blocker

`git add internal/handlers/api_keys.go internal/handlers/api_quota.go internal/handlers/fix.go internal/handlers/api_keys_test.go internal/handlers/fix_test.go harness/generated-work-items.json && git commit -m "Expose API key commerce handoff"` failed with:

```text
fatal: Unable to create '/Users/owlassist/foundry-businesses/nothumansearch/.git/index.lock': Operation not permitted
```

The worktree file edits are present, but `.git` metadata writes are blocked in this execution environment, so the required commit could not be created locally.
