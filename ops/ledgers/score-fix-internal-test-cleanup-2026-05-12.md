# Score-Fix Internal Test Cleanup - 2026-05-12

Automation: `business-agent-not-human-search`

Action: used the private admin action `POST /api/v1/admin/geo-jobs/action` with `action=mark_internal_test` to reclassify one Foundry-owned pending score-fix row as internal test data.

Boundary: aggregate-only. No row ids, hosts, emails, notes, Stripe ids, or raw admin rows are committed here.

Before:

```json
{
  "summary": [
    {"class": "real_candidate", "count": 3, "status": "pending"},
    {"class": "test_like", "count": 1, "status": "lead"},
    {"class": "test_like", "count": 2, "status": "paid"},
    {"class": "test_like", "count": 5, "status": "pending"}
  ],
  "by_status_host_class": [
    {"class": "real_candidate", "count": 2, "host_class": "dot_com", "status": "pending"},
    {"class": "real_candidate", "count": 1, "host_class": "foundry_owned", "status": "pending"},
    {"class": "test_like", "count": 1, "host_class": "dot_com", "status": "lead"},
    {"class": "test_like", "count": 2, "host_class": "dot_com", "status": "paid"},
    {"class": "test_like", "count": 4, "host_class": "dot_com", "status": "pending"},
    {"class": "test_like", "count": 1, "host_class": "foundry_owned", "status": "pending"}
  ]
}
```

After:

```json
{
  "summary": [
    {"class": "real_candidate", "count": 2, "status": "pending"},
    {"class": "test_like", "count": 1, "status": "internal_test"},
    {"class": "test_like", "count": 1, "status": "lead"},
    {"class": "test_like", "count": 2, "status": "paid"},
    {"class": "test_like", "count": 5, "status": "pending"}
  ],
  "by_status_host_class": [
    {"class": "real_candidate", "count": 2, "host_class": "dot_com", "status": "pending"},
    {"class": "test_like", "count": 1, "host_class": "foundry_owned", "status": "internal_test"},
    {"class": "test_like", "count": 1, "host_class": "dot_com", "status": "lead"},
    {"class": "test_like", "count": 2, "host_class": "dot_com", "status": "paid"},
    {"class": "test_like", "count": 4, "host_class": "dot_com", "status": "pending"},
    {"class": "test_like", "count": 1, "host_class": "foundry_owned", "status": "pending"}
  ]
}
```

Proof commands:

```sh
NHS_GEO_JOBS_LIMIT=500 ./tools/geo-jobs-redacted-read.sh
python3 tools/geo-jobs-mark-internal-test.py
```

Remaining actionable score-fix queue: two real-candidate external `dot_com` pending rows. Customer follow-up, if sent later, must use the email-outreach public-action lock path and commit only aggregate counts plus message ids.

## Aggregate closeout retry - 2026-05-13

Automation: `business-agent-not-human-search`

Required pre-read completed before this update:

- `tools/geo-jobs-redacted-read.sh`
- `ops/ledgers/score-fix-pending-followup-2026-05-12.md`
- `ops/ledgers/score-fix-internal-test-cleanup-2026-05-12.md`

Credential path in this worker runtime:

- `nhs-admin-api-key`: `SET`
- `nothumansearch-admin-key`: `SET`

Command:

```sh
./tools/geo-jobs-redacted-read.sh
```

Fresh aggregate-only result:

- Total score-fix rows: 11
- Real-candidate pending rows: 2, host class `dot_com`
- Test-like internal-test rows: 2, host class `foundry_owned`
- Test-like pending rows: 4, host class `dot_com`
- Test-like lead rows: 1, host class `dot_com`
- Test-like paid rows: 2, host class `dot_com`

Decision:

- The Foundry-owned pending cleanup lane is closed; no Foundry-owned pending row remains.
- The two external pending rows already have follow-up proof in `ops/ledgers/score-fix-pending-followup-2026-05-12.md`.
- Customer-visible score-fix follow-up due now: 0.
- No customer-visible email was sent, no public-action lock was created or reused, and no external customer row was mutated.
