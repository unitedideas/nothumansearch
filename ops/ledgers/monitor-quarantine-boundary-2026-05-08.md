# NHS Monitor Quarantine Boundary

Date: 2026-05-08
Automation: business-agent-not-human-search

## Action

Completed the spoof-looking monitor registration quarantine boundary.

- Shared-host apex registrations now return `monitoring_active=false`, `status=quarantined`, and a quarantine reason in the registration response.
- The `/monitor` UI no longer tells users those quarantined registrations are actively watched.
- `monitor-check` no longer logs raw email addresses on alert or per-row error paths; it logs email domain plus a stable redacted hash.

No public action was taken. No raw monitor rows, emails, payment IDs, or notes were written to this artifact.

## Proof

- `GOCACHE=/private/tmp/nhs-go-cache go test ./...`
- `fly deploy --remote-only`
- `curl -fsS https://nothumansearch.ai/health`
- `curl -fsS https://nothumansearch.ai/monitor | grep -E 'monitoring_active|Queued .* for review'`
- Signed `POST https://nothumansearch.ai/webhook/stripe` smoke returned HTTP 200 after deploy.

## Commits

- `e09155c` added the monitor-check redaction test.
- `ff7c61b` redacted monitor-check logs and exposed quarantine status to callers.

## Private Admin/Sales Handoff

Date: 2026-05-11
WorkItem: work_machine_8479a267466536b2
Source: admin monitor aggregate, not raw monitor rows

This is the private handoff for monitor registrations quarantined because their first monitor check returned zero agentic score or because the submitted host matches the shared-host apex risk class.

Committed artifacts must stay aggregate-safe:

- OK: counts by status, quarantine reason, age bucket, and decision outcome.
- OK: redacted email domain counts and stable internal hashes already produced by repo code.
- Not OK: raw email addresses, unsubscribe tokens, submitted domains tied to private monitor rows, private notes, payment identifiers, or support-thread text.

The raw monitor rows belong only in the later admin workflow behind the existing bearer-auth admin surface or a future private admin CLI.

The originating aggregate reported one active monitor and one quarantined monitor on 2026-05-11. The quarantined reason was `first_monitor_check_returned_zero_agentic_score`. `tools/monitor-check.log` shows the 2026-05-11 monitor-check run completed.

No raw monitor row, domain, email, token, or note was copied into this artifact.

Decision tree for each quarantined monitor:

1. Confirm the quarantine class using the private row plus the latest NHS crawl result.
2. If the submitted host is a shared-host apex or otherwise cannot represent a single owned property, keep it quarantined and ask the owner to submit the actual site hostname or rerun `/score` for the real property.
3. If the first check returned zero because the site was temporarily unreachable, blocked the crawler, or had no cached crawl yet, ask the owner to run `/score` again and only activate monitoring after a nonzero or explainable crawl result exists.
4. If the owner controls the site and the zero score is real, offer score-fix remediation focused on agent-readiness files and crawlability: `llms.txt`, OpenAPI, `/.well-known/ai-plugin.json`, structured API, MCP discovery, robots policy, and schema.
5. If the monitor is legitimate and the latest evidence shows a real agentic-readiness baseline, approve monitoring by moving it from quarantined to active and recording the review note privately.

Do not imply that score-fix remediation buys ranking placement. The offer is implementation help for public machine-readable readiness signals, not paid search ranking, paid inclusion, or preferred placement.

The private admin workflow should expose one row at a time with redacted email display, submitted domain, latest score, latest signal flags, quarantine reason, row age, operator actions, private notes, audit timestamp, and operator/source marker. Private notes must never enter committed artifacts or public ledgers.

Until that workflow exists, the safe manual path is admin-read-only inspection followed by a repo-local aggregate note. Do not email or publicly post from this work item.

## Aggregate Action Evidence Attempt

Date: 2026-05-12
WorkItem: work_machine_d4e39a4107911a0c
Source: aggregate monitor-admin action endpoint only

Added `tools/monitor-actions-redacted-read.sh` as the repo-local aggregate reader for:

- `GET /api/v1/admin/monitors/actions?days=N`
- day/action/count output only
- Keychain inline admin auth only
- no raw monitor rows, emails, tokens, submitted domains, payment identifiers, or private review notes

The worker could not record first-run action counts because both expected NHS admin Keychain services were unavailable in this execution context:

- `nhs-admin-api-key`
- `nothumansearch-admin-key`

No raw admin list endpoint was called. No row-level monitor data was read or copied into committed artifacts.

When the admin credential is restored, rerun:

```sh
tools/monitor-actions-redacted-read.sh
```

Then append only aggregate rows in this shape:

```text
YYYY-MM-DD approve_monitoring N
YYYY-MM-DD keep_quarantined N
YYYY-MM-DD request_score_rerun N
YYYY-MM-DD remediation_offered N
```

## Aggregate Action Evidence

Date: 2026-05-12T09:46:47Z
Automation: business-agent-not-human-search

Reran the aggregate-only monitor-admin action reader:

```sh
tools/monitor-actions-redacted-read.sh
```

Result:

```json
{"counts":[],"days":30}
```

Interpretation: the private admin action audit endpoint is reachable with inline Keychain auth, and no monitor admin actions have been recorded in the last 30 days. No raw monitor rows, emails, tokens, submitted domains, payment identifiers, or private review notes were read or committed.
