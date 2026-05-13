# NHS score-fix high-score gate - 2026-05-13

Action: deployed a score-fix eligibility gate so domains already at the target
score do not enter the paid remediation intake.

Why:
- `/fix/{host}` was previously gated only by hard agent signal presence.
- A site already scoring at or above the 95+ target could still see a paid
  "$199" remediation intake.
- Paid score-fix must remain implementation help for missing public
  agent-readiness signals, not ranking placement or score bypass.

Implementation:
- `scoreFixEligible` now requires a hard agent signal and `AgenticScore < 95`.
- Human GET `/fix/{host}` for high-score domains returns a monitor/report
  handoff instead of a payment form.
- Human POST `/fix/{host}` for high-score domains returns HTTP 409 and does
  not create a `geo_fix_jobs` row.
- Agentic checkout returns `score_already_meets_target` with monitor/report
  URLs and does not create a checkout for high-score domains.

Live proof:
- `curl -sS -o /tmp/nhs-fix-high.html -w 'high_fix_http=%{http_code}\n' https://nothumansearch.ai/fix/nothumansearch.ai`
  - `high_fix_http=200`
  - page contains `already meets the target`
  - page contains `/monitor?domain=nothumansearch.ai`
- `curl -sS -o /tmp/nhs-fix-candidate.html -w '%{http_code}' https://nothumansearch.ai/fix/bernstein.run`
  - `http=200`
  - page contains the paid form
- `curl -sS -o /tmp/nhs-health.json -w 'health_http=%{http_code}\n' https://nothumansearch.ai/health`
  - `health_http=200`
- signed Stripe webhook smoke:
  - `stripe_webhook_smoke_http=200`

Local proof:
- `GOCACHE=/tmp/nhs-go-cache-business-agent go test ./internal/handlers ./internal/models ./internal/database`
- `GOCACHE=/tmp/nhs-go-cache-business-agent go test ./...`

Privacy:
- No raw customer rows, emails, payment identifiers, monitor rows, or private
  score-fix notes were read or committed.
