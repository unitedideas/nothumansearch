# NHS Report Agent-First Corpus Repair

Date: 2026-05-13
Automation: `business-agent-not-human-search`

## Change

`/report` now uses the same agent-first corpus boundary as `/api/v1/stats`:

- Summary counts filter with `models.AgentFirstFilter`.
- Category table filters with `models.AgentFirstFilter`.
- Top-site table filters with `models.AgentFirstFilter`.
- Public report copy now labels the corpus as agent-first and names the hard-signal requirement.

## Proof

- Local tests: `GOCACHE=/tmp/nhs-go-cache-business-agent go test ./...`
- Deploy: `fly deploy --remote-only`
- Live report check: `curl -fsS https://nothumansearch.ai/report`
- Live stats check: `curl -fsS https://nothumansearch.ai/api/v1/stats`
- Live health check: `curl -fsS https://nothumansearch.ai/health`
- Signed Stripe webhook smoke: returned HTTP `200` using Keychain service `stripe-nhs-live-webhook-secret`.

## Live Result

- `/api/v1/stats`: `total_sites=4240`, `avg_score=35`
- `/report`: contains `4240`, `Agent-First Sites`, hard-signal language, and no stale `10206` or `23.2` report corpus values.

## Boundary

No public post, email, directory submission, owner outreach, row-level admin read, full recrawl, or external public-action lock was needed. This was a product-code deploy plus private proof ledger.
