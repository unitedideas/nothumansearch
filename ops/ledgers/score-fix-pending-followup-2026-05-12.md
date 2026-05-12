# NHS score-fix pending follow-up

Date: 2026-05-12
Automation: business-agent-not-human-search

## Aggregate read

Command:

```sh
tools/geo-jobs-redacted-read.sh
```

Result:

- Total score-fix rows: 11
- Real-candidate pending rows: 3
- Real-candidate pending age bucket: 3 in `7_29d`
- Real-candidate pending host classes: 2 `dot_com`, 1 `foundry_owned`
- Test-like rows: 8
- Real paid or lead rows: 0

## Public action

Sync-state lock: `public-action-locks/email-outreach/41e6f9e142f1fd9d.json`

Action completed:

- Sent 2 score-fix checkout-abandonment follow-up emails for the real-candidate `dot_com` pending cohort.
- Excluded the `foundry_owned` pending row from customer follow-up.
- Message IDs:
  - `36a738d8-710b-4ed7-a77c-1032def34a3f`
  - `6df61122-82c7-42b6-be85-5665e54b67cb`

Committed artifacts intentionally omit raw emails, hostnames, row IDs, Stripe IDs, and notes.

## Private-shadow reconciliation

Date: 2026-05-12
WorkItem: `work_machine_9ef0523b69ec15a4`

Required pre-read completed before this ledger update:

- Re-read `tools/geo-jobs-redacted-read.sh`.
- Re-read this score-fix follow-up ledger.

Live admin-state mutation was not locally feasible in this worker runtime because both configured Keychain aliases were unavailable to the repo helper:

- `nhs-admin-api-key`: missing
- `nothumansearch-admin-key`: missing

No raw admin rows were fetched. No customer-visible email was sent. No public action lock was created or reused.

Private-shadow classification:

- The previously contacted external pending cohort is treated as `follow_up_sent` for follow-up planning. A second customer email is blocked unless a future worker re-reads private admin state, proves the duplicate ledger allows another touch, and obtains a fresh public-action lock.
- The remaining `foundry_owned` pending row is excluded from customer follow-up and classified as `internal_test_or_cleanup` for private-state planning.

Aggregate-only proof from the prior redacted read already recorded in this ledger:

- Total score-fix rows: 11
- Real-candidate pending rows: 3
- Real-candidate pending host classes: 2 `dot_com`, 1 `foundry_owned`
- Test-like rows: 8
- Real paid or lead rows: 0

Remaining pending action classes after private-shadow reconciliation:

- External pending rows requiring no immediate customer action: 2 `follow_up_sent`, host class `dot_com`
- Internal pending rows requiring cleanup/classification only: 1 `internal_test_or_cleanup`, host class `foundry_owned`
- Customer follow-up due now: 0

## Admin-backed aggregate reconciliation

Date: 2026-05-12T16:26:00Z
Automation: business-agent-not-human-search

Command:

```sh
tools/geo-jobs-redacted-read.sh
```

Result:

- Helper credential path is restored: `nhs-admin-api-key` and `nothumansearch-admin-key` both return non-secret `SET` checks.
- Total score-fix rows: 11
- Real-candidate pending rows: 3
- Real-candidate pending age bucket: 3 in `7_29d`
- Real-candidate pending host classes: 2 `dot_com`, 1 `foundry_owned`
- Test-like rows: 8
- Real paid or lead rows: 0

Planning classification remains unchanged:

- Previously contacted external pending cohort: 2 `follow_up_sent`, host class `dot_com`
- Internal pending row: 1 `internal_test_or_cleanup`, host class `foundry_owned`
- Customer follow-up due now: 0

No customer-visible email was sent. No public action lock was created or reused.

## Private-shadow reconciliation

Date: 2026-05-12T15:19:42Z
WorkItem: `work_machine_e1702f99556c749f`

Required pre-read completed before this ledger update:

- Re-read `tools/geo-jobs-redacted-read.sh`.
- Re-read this score-fix follow-up ledger.

Live admin-state mutation remains blocked in this worker runtime because both configured Keychain aliases were unavailable to the repo helper:

- `nhs-admin-api-key`: missing
- `nothumansearch-admin-key`: missing

No raw admin rows were fetched. No customer-visible email was sent. No public action lock was created or reused.

Private-shadow classification:

- The previously contacted external pending cohort remains treated as `follow_up_sent` for follow-up planning. A second customer email remains blocked unless a future worker re-reads private admin state, proves the duplicate ledger allows another touch, and obtains a fresh public-action lock.
- The remaining `foundry_owned` pending row remains excluded from customer follow-up and classified as `internal_test_or_cleanup` for private-state planning.

Aggregate-only proof from the prior redacted read already recorded in this ledger:

- Total score-fix rows: 11
- Real-candidate pending rows: 3
- Real-candidate pending age bucket: 3 in `7_29d`
- Real-candidate pending host classes: 2 `dot_com`, 1 `foundry_owned`
- Test-like rows: 8
- Real paid or lead rows: 0

Remaining pending action classes after private-shadow reconciliation:

- External pending rows requiring no immediate customer action: 2 `follow_up_sent`, host class `dot_com`
- Internal pending rows requiring cleanup/classification only: 1 `internal_test_or_cleanup`, host class `foundry_owned`
- Customer follow-up due now: 0
