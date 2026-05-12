# Monitor Quarantine Admin Actions

WorkItem: `work_machine_530e86ea8811bac7`

Scope: add an authenticated private admin action path for quarantined monitor rows without committing row-level private data.

Implemented:

- Added migration `016_monitor_admin_actions.sql`.
- Added private audit table `monitor_admin_actions` with action, operator, source, notes, and `created_at`.
- Added monitor columns for latest admin action metadata, score-rerun request timestamp, remediation-offered timestamp, and private review notes.
- Added model actions:
  - `approve_monitoring`
  - `keep_quarantined`
  - `request_score_rerun`
  - `remediation_offered`
- Added authenticated `POST /api/v1/admin/monitors/action`.
- Added authenticated aggregate-only `GET /api/v1/admin/monitors/actions?days=N`.
- Kept weekly due monitor selection active-only so quarantined rows remain excluded until approval.

Privacy boundary:

- Committed artifacts contain only schema, aggregate-safe action names, and implementation notes.
- Admin action reporting returns day/action/count only.
- Raw emails, submitted domains, unsubscribe tokens, and private review notes remain confined to the authenticated admin workflow and database rows.

Verification:

- `GOCACHE=$PWD/.gocache go test ./...`
- `GOCACHE=$PWD/.gocache go test ./internal/models ./internal/handlers`
- `python3 -m json.tool harness/generated-work-items.json`
- Initial plain `go test ./...` failed only because the sandbox could not write the default macOS Go build cache under `~/Library/Caches/go-build`.
