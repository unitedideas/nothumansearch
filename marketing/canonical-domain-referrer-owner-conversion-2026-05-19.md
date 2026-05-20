# Canonical Domain Referrer Owner Conversion

Run: 2026-05-19T23:07Z `business-marketer-not-human-search`

## Fresh Evidence

- Public stats: 4,175 indexed sites, average score 35, top category `developer`.
- Public categories: developer 1,231; ai-tools 904; other 768; data 400; finance 196; productivity 173; ecommerce 148; communication 120; security 115; health 59; jobs 27; education 21; news 12; spam 1.
- Aggregate 168h traffic shows canonical and alias traffic continuing to loop through both `nothumansearch.ai` and `nothumansearch.com`:
  - `https://nothumansearch.ai/`: 2,052 referrers
  - `https://google.com`: 542 referrers
  - `https://nothumansearch.com/`: 156 referrers
  - `https://www.nothumansearch.ai/`: 141 referrers
  - `https://www.nothumansearch.com/`: 132 referrers
  - `http://www.nothumansearch.ai`: 131 referrers
  - `http://nothumansearch.ai`: 126 referrers
  - `http://www.nothumansearch.com`: 124 referrers
  - `http://nothumansearch.com`: 120 referrers
  - `https://nothumansearch.ai/score`: 119 referrers
- Route traffic still concentrates on machine-readable and owner-readiness surfaces:
  - `/badge/xquik.com.svg`: 1,904
  - `/.well-known/commerce.json`: 1,692
  - `/.well-known/ai-plugin.json`: 768
  - `/site/xquik.com`: 592
  - `/llms.txt`: 494
  - `/openapi.yaml`: 456
  - `/api/v1/catalog`: 362
  - `/api/v1/quote`: 333
  - `/api/v1/checkout`: 333
  - `/robots.txt`: 309
  - `/api/v1/submit`: 147
- Public surface smoke:
  - `/monitor`, `/score`, `/api/v1`, `/.well-known/mcp.json`, `/.well-known/agent.json`, and `/.well-known/commerce.json` return 200.
  - `/.well-known/agent-card.json` returns 404.
  - High-score `/fix/nothumansearch.ai` and `/fix/xquik.com` now show the "already meets the NHS score target" branch, not paid remediation intake.

## Scout Read

The alias/referrer pattern is not evidence of customers, paid leads, or private demand. It is a conversion-routing signal: users and crawlers arrive through `.com`, `www`, and `http` variants, then land on score, profile, badge, and machine-readable commerce surfaces.

This should become a product-safe owner-conversion test:

1. Keep all aliases canonical to `https://nothumansearch.ai/`.
2. Use canonical-domain traffic to test whether `/score`, `/monitor`, `/report`, `/badge/{host}.svg`, and `/site/{host}` give the same owner next step.
3. Route high-score owners to free monitor/report/badge proof.
4. Route partial-score owners to `/score` before remediation.
5. Route API-heavy callers from `/api/v1/catalog`, `/api/v1/quote`, and `/api/v1/checkout` to API plans only after docs remain useful.

## Draft Owner-Channel Angle

Agents do not always arrive at the canonical URL. They follow `www`, old domains, badges, manifests, OpenAPI files, commerce manifests, and profile pages.

NHS can use the canonical-domain test as an owner-facing proof point:

> If agents can reach your site through five aliases, a badge URL, an OpenAPI file, and a plugin manifest, all of those paths should converge on the same machine-readable source of truth. NHS checks that routing and monitors it for drift.

## Guardrails

- Do not claim alias/referrer traffic is private demand, monitor registration, customer proof, badge-install consent, revenue, completed checkout, or endorsement.
- Do not claim crawler compliance, legal permission, SEO lift, A2A support, x402/ACP support, paid placement, preferred inclusion, or score-methodology bypass.
- Keep `/.well-known/agent-card.json` as a compatibility gap until it returns a deliberate 200 or documented no-support response.
- Before public use, refresh public stats, categories, alias redirects, monitor/score/profile/fix routes, commerce/agent manifests, OpenAPI, MCP tools, aggregate traffic, and aggregate MCP analytics.
