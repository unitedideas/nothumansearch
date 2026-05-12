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
