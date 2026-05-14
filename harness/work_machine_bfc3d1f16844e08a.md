# API-key commerce subscribe handoff closeout

WorkItem: `work_machine_bfc3d1f16844e08a`
Date: 2026-05-14

## Completed locally

- Added browser-readable HTML for `GET /api/v1/api-keys/subscribe` when the requester sends `Accept: text/html`.
- Preserved the agent-readable JSON handoff for the same GET route.
- Reused the shared starter, pro, and scale API-plan summaries across subscribe, catalog, commerce, and agent surfaces.
- Added live-safe tests for:
  - HTML subscribe handoff with starter, pro, and scale plan copy.
  - POST subscribe error contract when Stripe is not configured, without a checkout URL.
  - Anonymous quota-exceeded handoff metadata, without a checkout URL.
  - Catalog and quote coverage for starter, pro, and scale.
  - `/.well-known/commerce.json` API-plan exposure, without raw checkout URLs.
  - `/.well-known/agent.json` API-plan exposure.
- Updated `harness/generated-work-items.json` to keep the remaining deploy/live-smoke follow-up gated.

## Verification

- `GOCACHE=/tmp/nhs-go-cache go test ./internal/handlers` passed.
- `GOCACHE=/tmp/nhs-go-cache go test ./...` was attempted and failed in the pre-existing `cmd/monitor-check` test build before touched packages:
  - `firstCheckFailedQuarantineReason` undefined.
  - `firstCheckZeroScoreQuarantineReason` undefined.
  - `firstCheckQuarantineReason` undefined.

## Commit blocker

The required commit could not be created from this runner because `.git` metadata writes are blocked:

```text
fatal: Unable to create '/Users/owlassist/foundry-businesses/nothumansearch/.git/index.lock': Operation not permitted
```

`git status` also hides the touched tracked paths because they are marked with index flags, and clearing those flags requires the same blocked `.git/index.lock` write. The worktree edits and this evidence artifact are present, but local commit creation is blocked by the execution environment.
