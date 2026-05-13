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
