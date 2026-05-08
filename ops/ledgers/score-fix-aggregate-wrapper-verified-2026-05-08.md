# NHS score-fix aggregate wrapper verification - 2026-05-08

Automation: `business-agent-not-human-search`

## Action

Completed the aggregate-only score-fix intake helper so later workers can read `GET /api/v1/admin/geo-jobs?limit=500` with the admin token read inline from Keychain and write only aggregate-safe output.

The wrapper output excludes row refs, emails, hostnames, notes, Stripe session IDs, payment IDs, and raw admin rows.

## Live aggregate proof

Command:

```bash
tools/geo-jobs-redacted-read.sh
```

Result:

- Total rows: 11
- Real candidate rows: 3 pending
- Real paid or lead rows: 0
- Test-like rows: 8
- Test-like pending rows: 5
- Test-like paid rows: 2
- Test-like lead rows: 1

Age split:

- Real pending: 3 in `1_6d`
- Test-like pending: 2 in `1_6d`, 3 in `7_29d`
- Test-like paid or lead: 3 in `7_29d`

Host-class split:

- Real pending: 2 `dot_com`, 1 `foundry_owned`
- Test-like pending: 4 `dot_com`, 1 `foundry_owned`
- Test-like paid or lead: 3 `dot_com`

## Verification

```bash
python3 tools/test-redact-geo-jobs.py
tools/geo-jobs-redacted-read.sh
GOCACHE=/private/tmp/nhs-go-cache go test ./...
```

## Follow-up boundary

The credential-repair work item is closed because the wrapper succeeded with the local `nhs-admin-api-key` Keychain service. The remaining public follow-up work stays gated by a public-action lock and must re-read through this wrapper before selecting row-specific recipients.
