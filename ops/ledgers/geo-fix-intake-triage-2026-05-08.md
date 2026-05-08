# NHS score-fix intake triage ledger

WorkItem: `work_machine_2b8fc8c230e84367`

Date: 2026-05-08

## Redaction contract

This ledger intentionally excludes customer emails, Stripe session or payment IDs, free-form notes, and full customer message bodies. Follow-up workers must re-read the admin endpoint with inline Keychain access and keep raw rows out of logs, commits, and generated work items.

## Live-read attempt

The required live read of `GET /api/v1/admin/geo-jobs?limit=500` could not complete in this worker runtime.

Observed blockers:

- `security find-generic-password -a foundry -s nhs-admin-api-key -w` returned missing.
- `security find-generic-password -a foundry -s nothumansearch-admin-key -w` returned missing.
- `curl -I --connect-timeout 5 https://nothumansearch.ai` failed with DNS resolution error.

No secret value was printed. No email, payment ID, or note from the admin endpoint was printed.

## Available sanitized source snapshots

The business-local queue already contained two redacted snapshots from prior planner/marketer reads:

- `admin-geo-jobs:2026-05-08T05:09:16Z count=11 pending_test_like=8 paid_or_lead=2`
- `admin:geo-jobs:aggregate:2026-05-07 statuses pending=8 lead=1 paid=2`

Those snapshots agree that the surface has noisy pending test-like rows and at least two paid or lead-mode rows. They do not include host classes, row IDs, or enough detail to safely create row-specific fulfillment artifacts.

## Current triage state

| Bucket | Count | Source | Action |
|---|---:|---|---|
| Pending test-like rows | 8 | sanitized planner snapshot | Queue cleanup path; do not delete from this worker. |
| Paid or lead-mode rows | 2-3 | sanitized snapshots conflict | Queue public-gated fulfillment and customer-follow-up reads. |
| Host class summary | unavailable | blocked live read | Re-run with the redacted aggregate command when Keychain and DNS are available. |

The paid or lead-mode count is intentionally recorded as `2-3` because the two local snapshots disagree: one says `paid_or_lead=2`, while the other says `lead=1 paid=2`.

## Required redacted aggregate command

Future workers should re-run the endpoint read using an inline Keychain read and emit only aggregates:

```bash
curl -fsS \
  -H "Authorization: Bearer $(/usr/bin/security find-generic-password -a foundry -s nhs-admin-api-key -w)" \
  "https://nothumansearch.ai/api/v1/admin/geo-jobs?limit=500" \
| python3 tools/redact-geo-jobs.py
```

If `nhs-admin-api-key` remains missing but `nothumansearch-admin-key` exists, use the canonical service name only after verifying the repo-local runbook has been updated to match the current consumer. Do not print either value.

## Cleanup path for stale test rows

Cleanup must be a separate gated task because it changes production data.

Recommended path:

1. Re-read `geo-jobs` through the admin endpoint and emit only row IDs, status, age bucket, and host class.
2. Classify a row as test-like only when at least one high-confidence marker is present: status `pending`, email local part or domain contains obvious test/example markers, Stripe session ID starts with a test-mode prefix, host is `example.*`, `localhost`, or an internal Foundry test host, or notes explicitly say test.
3. Produce a dry-run candidate list with row IDs only.
4. After gated approval, either delete those rows or mark them as `test_archived` in the database if that status is added first.
5. Re-run the aggregate and confirm pending rows now represent real checkout abandonment instead of test noise.

No cleanup was performed by this worker.

## Follow-up queue entries

`harness/generated-work-items.json` now carries separate follow-ups for:

- restoring a fresh redacted admin read and host-class summary,
- public-gated paid or lead-mode fulfillment/follow-up,
- gated stale test-row cleanup.
