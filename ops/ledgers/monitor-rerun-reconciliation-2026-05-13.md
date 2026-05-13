# NHS Monitor Rerun Reconciliation

Date: 2026-05-13T15:00:00Z
Automation: business-agent-not-human-search
Source: private monitor-admin aggregate workflow

## Action

Reconciled the recorded monitor `request_score_rerun` action using aggregate-only monitor admin readers.

No public action was taken. No raw monitor row, email, submitted domain, token, private note, payment identifier, or row id was committed.

## Proof

Commands:

```sh
tools/monitor-status-redacted-read.sh
tools/monitor-actions-redacted-read.sh
GOCACHE=/private/tmp/nhs-go-cache go test ./...
```

Aggregate results:

```json
{"counts":[{"status":"active","count":1},{"status":"quarantined","quarantine_reason":"first monitor check returned zero agentic score","count":1}]}
{"counts":[{"day":"2026-05-13T00:00:00Z","action":"request_score_rerun","count":1}],"days":30}
```

`GOCACHE=/private/tmp/nhs-go-cache go test ./...` passed.

## Outcome

The `request_score_rerun` action is recorded, but aggregate status still shows one quarantined monitor for `first monitor check returned zero agentic score`. Local monitor-check evidence has no completed run after the 2026-05-13 action; latest completed live run remains 2026-05-11.

The quarantined monitor remains excluded from active checks until a later bounded rerun/review records `approve_monitoring`, `keep_quarantined`, or `remediation_offered`.

## Bounded rerun outcome

Date: 2026-05-13T19:05:00Z
Automation: business-agent-not-human-search

Command:

```sh
python3 tools/monitor-quarantine-rerun.py
```

Aggregate result:

```json
{
  "action": "keep_quarantined",
  "rerun_score_bucket": "zero",
  "before_status_counts": [
    {"status": "active", "quarantine_reason": "", "count": 1},
    {"status": "quarantined", "quarantine_reason": "first monitor check returned zero agentic score", "count": 1}
  ],
  "after_status_counts": [
    {"status": "active", "quarantine_reason": "", "count": 1},
    {"status": "quarantined", "quarantine_reason": "bounded rerun still zero score", "count": 1}
  ],
  "action_counts": [
    {"day": "2026-05-13T00:00:00Z", "action": "keep_quarantined", "count": 1},
    {"day": "2026-05-13T00:00:00Z", "action": "request_score_rerun", "count": 1}
  ]
}
```

Outcome: the requested rerun has a final private admin outcome. The row remains quarantined because the bounded rerun still returned zero score, and active monitor checks still have one active row plus one quarantined row.

No raw monitor row, email, submitted domain, token, private note, payment identifier, or row id was committed.
