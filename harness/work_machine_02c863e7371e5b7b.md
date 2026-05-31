# Monitor Quarantine Review Blocked

Date: 2026-05-31T00:00:00-07:00
WorkItem: work_machine_02c863e7371e5b7b
Source: private monitor-admin workflow

Updated `tools/monitor-quarantine-rerun.py` so the private rerun helper can process more than one eligible first-check zero-score quarantine in a single run while emitting aggregate-safe output only.

The helper failed closed before any monitor row review because this worker runtime cannot read either expected NHS admin Keychain service:

```text
missing Keychain service: nhs-admin-api-key or nothumansearch-admin-key
```

Aggregate refresh commands:

```sh
tools/monitor-status-redacted-read.sh
tools/monitor-actions-redacted-read.sh
```

Both aggregate readers failed closed with the same credential boundary before fetching admin data.

Latest safe aggregate snapshot from planner input:

```text
status=active count=3
status=quarantined reason=bounded rerun still zero score count=1
status=quarantined reason=first monitor check returned zero agentic score count=2
day=2026-05-13 action=keep_quarantined count=1
day=2026-05-13 action=request_score_rerun count=1
```

Verification:

```sh
GOCACHE=/private/tmp/nhs-go-cache go test ./...
```

Result: passed.

Next action: restore `nhs-admin-api-key` or `nothumansearch-admin-key` in the executor runtime, then rerun `python3 tools/monitor-quarantine-rerun.py`, `tools/monitor-status-redacted-read.sh`, and `tools/monitor-actions-redacted-read.sh`.

No raw monitor domains, URLs, row ids, emails, tokens, payment identifiers, private notes, or review notes were committed or written here.
