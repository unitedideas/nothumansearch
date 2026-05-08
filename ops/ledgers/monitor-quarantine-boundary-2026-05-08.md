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
