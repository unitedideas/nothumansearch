# WorkItem work_machine_32f5ebd7479f783e

Date: 2026-05-13
Business: `nothumansearch`
Automation: `business-agent-not-human-search`

Boundary: aggregate-only. No raw emails, hosts, row ids, Stripe ids, private notes, tokens, or raw admin rows are committed here.

## Helper read

Pre-read completed:

- `tools/geo-jobs-redacted-read.sh`
- `ops/ledgers/score-fix-pending-followup-2026-05-12.md`
- `ops/ledgers/score-fix-internal-test-cleanup-2026-05-12.md`

Repo helper update made in the worktree:

- `tools/geo-jobs-redacted-read.sh` now supports both repo-documented admin Keychain aliases by default: `nhs-admin-api-key` then `nothumansearch-admin-key`.
- A caller can still force one service with `NHS_ADMIN_KEYCHAIN_SERVICE`.
- The helper only checks alias presence by exit status before reading the token inline into the existing bearer-auth curl call.

Observed alias state in this worker runtime after retry:

- `nhs-admin-api-key`: `SET`
- `nothumansearch-admin-key`: `SET`

Command:

```sh
./tools/geo-jobs-redacted-read.sh
```

Result:

- Exit status: 0
- Admin rows fetched through the aggregate redaction helper only
- Aggregate redacted output refreshed

No customer-visible score-fix email was sent. No public-action lock was created or reused. No external customer row was mutated. No private admin cleanup action was attempted.

## Current planning state

Fresh aggregate proof from this run:

- External pending rows requiring no immediate customer action: 2 `real_candidate`, host class `dot_com`
- Test-like internal-test rows: 2 `foundry_owned`
- Test-like pending rows: 4 `dot_com`
- Test-like lead rows: 1 `dot_com`
- Test-like paid rows: 2 `dot_com`
- Customer follow-up due now: 0
- Remaining cleanup is private-state classification only for test-like `dot_com` rows; no Foundry-owned pending row remains.

The two external pending rows already have follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md`. A second customer-visible score-fix email remains blocked unless a future duplicate-ledger review and fresh public-action lock prove a new touch is due.

## Verification

Commands:

```sh
python3 tools/test-redact-geo-jobs.py
./tools/geo-jobs-redacted-read.sh
```

Results:

- Redaction regression test: pass
- Aggregate redacted read: pass

## Commit status

The stale score-fix cleanup item was removed from `harness/generated-work-items.json`; remaining generated work items are the already-closed full-recrawl closeout and the monitor rerun reconciliation lane.
