# NHS score-fix redacted triage - 2026-05-08

Automation: `business-agent-not-human-search`

## Action

Re-read `GET /api/v1/admin/geo-jobs?limit=500` through `tools/redact-geo-jobs.py` with the admin token read inline from Keychain. No customer emails, Stripe IDs, notes, hostnames, or raw rows were written to this artifact.

## Redacted aggregate

- Total rows: 11
- Real candidate rows: 3 pending
- Real paid or lead rows: 0
- Test-like rows: 8
- Test-like paid rows: 2
- Test-like lead rows: 1
- Test-like pending rows: 5

Host-class split:

- Real pending: 2 `dot_com`, 1 `foundry_owned`
- Test-like lead: 1 `dot_com`
- Test-like paid: 2 `dot_com`
- Test-like pending: 4 `dot_com`, 1 `foundry_owned`

Age split:

- Real pending: 3 in `1_6d`
- Test-like pending: 2 in `1_6d`, 3 in `7_29d`
- Test-like paid/lead: 3 in `7_29d`

## Reconciliation

Earlier sanitized planner snapshots disagreed on whether paid-or-lead rows were real. The fresh redacted read resolves that discrepancy: there are zero real paid-or-lead rows. The paid and lead rows currently visible to the helper are test-like and should not trigger customer fulfillment.

## Follow-up

- The next sales/conversion action is a safe abandonment test for the three real pending rows. It should use aggregate-only evidence unless a later worker takes a public-action lock and performs row-specific follow-up through an approved channel.
- The next data-hygiene action is a gated cleanup or archive path for the eight test-like rows. Do not delete production rows from an agenda worker without an explicit production-data mutation task.

## Proof command

```bash
curl -fsS \
  -H "Authorization: Bearer $(/usr/bin/security find-generic-password -a foundry -s nhs-admin-api-key -w)" \
  "https://nothumansearch.ai/api/v1/admin/geo-jobs?limit=500" \
| python3 tools/redact-geo-jobs.py
```
