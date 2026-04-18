# Security Policy — Not Human Search

## Supported Versions

This is a continuously-deployed Go service — there's only one version at any given time (what's on `main` / what Fly.io is currently serving at `nothumansearch.ai`). Security fixes land on `main` and deploy immediately. Older commits are not supported.

## Reporting a Vulnerability

- **Email** (preferred for sensitive reports): security@nothumansearch.ai
- **GitHub issues**: https://github.com/unitedideas/nothumansearch/issues (for lower-severity issues where public discussion is fine)
- **Mirrored contact list**: https://nothumansearch.ai/.well-known/security.txt

Please include:

1. A reproducible test case or curl sequence.
2. The impact — what can an attacker do?
3. Your timeline. If you need a response within N days, tell me upfront.

## Response Commitment

- **Acknowledgement**: within **3 business days** of receiving a report.
- **Triage**: within **7 business days**, with a severity call + rough patch ETA.
- **Fix**: target **30 days** for high/critical. Lower severity can take longer; I'll keep you informed.
- **Disclosure**: coordinated. Default **30-day embargo** from the fix landing on `main` before public write-up. Longer if needed to protect users; shorter if the bug is already being actively exploited.

## Scope

In scope:

- The production site at `nothumansearch.ai`.
- The REST API at `/api/v1/*`.
- The MCP server at `/mcp`.
- Agent-discovery surfaces (`/.well-known/*`, `llms.txt`, `openapi.yaml`, `sitemap.xml`).
- The crawler (`/app/crawler` binary behavior when operating on third-party URLs — e.g. SSRF, SSRF-via-redirect).
- The on-demand check endpoint at `/api/v1/check` (live crawler invocation via HTTP).
- The MCP registry signing at `/.well-known/mcp-registry-auth`.
- Monitor subscription flow (`/api/v1/monitor/register` + worker email sends).

Out of scope:

- `*.fly.dev` — infrastructure provider, not the product.
- Third-party sites we index in the agentic-readiness score.
- The sites themselves that register at `/monitor` (we notify; we don't attest their security).
- Issues that require physical access to your own device.
- Clickjacking on pages without sensitive state changes.
- Self-XSS (injecting script into your own browser).
- Rate-limit bypass via IP rotation alone (bring impact, not theory).

## Safe Harbor

Good-faith security research is welcomed. I will not pursue legal action for research that:

- Operates within the Scope above.
- Doesn't intentionally degrade service for other users (no DoS).
- Doesn't access, modify, or destroy data belonging to other users beyond what's necessary to demonstrate the issue.
- Reports via the channels above before public disclosure.

I cannot grant safe harbor for actions that violate third-party terms — if your test touches another service, you need that service's policy.

## SSRF / Crawler-Abuse Note

Both `/api/v1/check` and `submit_site` invoke the crawler against a caller-provided URL. The crawler has SSRF protections (blocks 127.x / 10.x / 192.168.x / 172.16–31.x / 169.254.x / localhost / IPv6 loopback). Bypass vectors are **high priority**. Please report these first-class, with the exact target you reached and what you retrieved.

## What I Cannot Offer

- A formal bug bounty (no paid tier for reports yet).
- SLA in hours (solo operator — best-effort against the day-job clock).
- NDA / exclusivity agreements.

That said, clear reports get public credit in the changelog unless you ask to stay anonymous.
