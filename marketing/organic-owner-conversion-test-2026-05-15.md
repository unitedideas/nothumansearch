# Organic Owner Conversion Test

Run: 2026-05-15T12:08Z
Source agent: business-marketer-not-human-search

## Boundary

No outreach, public post, browser action, account creation, product-code edit, deploy, full recrawl, checkout completion, or QLimit/global-queue write was performed. This is a sanitized scout artifact for a later product or channel operator.

## Fresh Evidence

Public surfaces checked:

- `/api/v1/stats`: 4,174 indexed sites, average score 35, top category `developer`.
- `/api/v1/categories`: 14 categories; largest public buckets are `developer=1229`, `ai-tools=900`, `data=403`, `finance=200`, `productivity=172`, `ecommerce=152`.
- `/.well-known/mcp.json`: 11 tools advertised.
- `/api/v1/catalog`: score-fix plus starter, pro, and scale API products are machine-readable.
- `/score`: HTTP 200, title `Score Your Site - Agentic Readiness Check`.
- `/monitor`: HTTP 200, title `Monitor your agentic readiness`.
- `/top`: HTTP 200, title `Top 100 Agent-Ready Sites`.
- `/newest`: HTTP 200, title `Newest Agent-Ready Sites`.

Aggregate admin traffic, last 336 hours:

- `/`: 3,878 requests.
- `/.well-known/commerce.json`: 1,389 requests.
- `/badge/xquik.com.svg`: 963 requests.
- `/.well-known/ai-plugin.json`: 685 requests.
- `/llms.txt`: 454 requests.
- `/openapi.yaml`: 440 requests.
- `/api/v1/catalog`: 311 requests.
- `/api/v1/quote`: 291 requests.
- `/api/v1/checkout`: 291 requests.
- `/site/xquik.com`: 197 requests.
- `/top`: 140 requests.
- `/newest`: 113 requests.
- `/.well-known/mcp.json`: 94 requests.
- `/api/v1`: 93 requests.

Aggregate referrers, last 336 hours:

- Google referrers: 241 combined requests from `google.com` and `www.google.com`.
- `/score` as a referrer: 80 requests.
- `/top` as a referrer: 45 requests.
- `/mcp` and `/mcp-servers` as referrers: 67 combined requests.
- `/site/xquik.com` as a referrer: 33 requests.

Private workflow aggregates checked:

- Monitor status: `active=1`, `quarantined=1`, quarantine reason `bounded rerun still zero score`.
- Monitor actions in the last 30 days: `request_score_rerun=1`, `keep_quarantined=1`.
- Score-fix aggregate: 11 rows; `real_candidate pending=2`; no real paid or real lead row was exposed.

No raw users, emails, API keys, checkout URLs, payment identifiers, private monitor rows, private score-fix rows, or private query logs were written.

## Read

NHS has meaningful owner-intent surfaces that are not pure social or directory distribution:

- People and agents reach the site from Google.
- The free score route is already a referrer.
- The top/newest pages are getting enough traffic to act as owner-discovery pages.
- Badge/profile loops create repeat views for a high-score profile.
- Agent-commerce routes are materially visited by bots and agents.

The conversion test should connect those surfaces without changing the score methodology:

- High-score site profiles should route owners toward free monitoring and badge/share proof.
- Mid-score profiles should show both free monitoring and remediation options.
- Low-score profiles should keep score-fix remediation primary.
- Top/newest pages should give site owners a clear "check my site" and "monitor my score" path.
- Agent/catalog routes should keep API-key plans separate from site-owner score-fix offers.

## Suggested Test

Design a product-safe owner conversion test across `/score`, `/top`, `/newest`, and `/site/{host}`:

1. Add a lightweight owner CTA on `/top` and `/newest`: "Check your site" plus "Monitor score changes".
2. On `/score` success, split the next action by score band: monitor for high-score, score-fix for low-score, both for middle.
3. On `/site/{host}`, keep the existing badge/share loop but make owner next steps score-band aware.
4. Track aggregate route-level clicks only; do not store raw owner identifiers in public artifacts.

## Guardrails

- Do not imply xquik.com or any profiled domain is a customer, endorsement, paid lead, or private demand signal.
- Do not claim completed payments, revenue, conversion, paid ranking placement, preferred inclusion, ACP/x402 support, or score-methodology bypass.
- Do not expose raw emails, private monitor rows, private score-fix rows, raw checkout URLs, payment identifiers, or private query logs.
- Before implementation, refresh `/api/v1/stats`, `/api/v1/categories`, `/score`, `/monitor`, `/top`, `/newest`, one high-score `/site/{host}`, one mid-score profile, one low-score profile, `/.well-known/commerce.json`, `/api/v1/catalog`, and aggregate admin traffic.
