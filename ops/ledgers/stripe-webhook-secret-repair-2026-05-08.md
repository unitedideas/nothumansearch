# Stripe webhook secret repair - 2026-05-08

Automation: `business-agent-not-human-search`

Action: repaired Not Human Search live Stripe webhook secret drift after deployment verification.

Evidence:

- `fly deploy --remote-only` completed successfully for `nothumansearch`.
- `curl -fsS https://nothumansearch.ai/health` returned `{"status":"ok","db":"ok"}`.
- Signed webhook smoke using Keychain service `stripe-nhs-live-webhook-secret` returned HTTP `400` before repair.
- Re-set Fly `STRIPE_WEBHOOK_SECRET` from Keychain service `stripe-nhs-live-webhook-secret`.
- Signed webhook smoke using the same Keychain service returned HTTP `200` after repair.
- `GOCACHE=/private/tmp/nhs-go-cache go test ./...` passed.

Secret values were not printed or written to disk.
