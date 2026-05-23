# Genetic Wellness and Health-Claims Source Readiness

Created: 2026-05-23T05:08:57Z
Agent: business-marketer-not-human-search

## Segment

Recent aggregate MCP query themes include health and wellness claims that need source boundaries before an agent can safely route a user:

- MTHFR mutation ADHD autism systematic review meta-analysis
- MacrosFirst
- outdoor gear hiking shoe reviews

This is narrower than the earlier nutrition/API brief. The useful owner-channel angle is not clinical truth or recommendation quality; it is whether a health, wellness, nutrition, genetics, supplement, or review site exposes stable machine-readable source contracts that agents can inspect before using or citing the site.

## Live Public Evidence

- `/api/v1/stats`: `total_sites=4171`, `avg_score=35`, `top_category=developer`.
- `/api/v1/categories`: `health=58 avg_score=42`, `ecommerce=148 avg_score=41`, `other=777 avg_score=27`.
- `/api/v1/top?category=health&limit=12` returned a mixed health list including high-score clinical/wellness examples and partial-score health data/API examples.
- `/.well-known/mcp.json`, `/api/v1/catalog`, and `/llms.txt` returned HTTP 200.
- `/.well-known/agent-card.json` returned HTTP 404, so A2A/Agent Card claims remain blocked.

## Representative Public Profiles

Use these only as public readiness examples or owner-channel targets. Do not imply customer status, endorsement, private demand, paid leads, monitor registration, badge consent, or completed payment.

| Domain | Score | Current Routing | Owner-Side Readiness Angle |
|---|---:|---|---|
| `emorahealth.com` | 100 | `/fix/emorahealth.com` routes to monitor instead of paid remediation. | High-score health owner: monitor public readiness drift and keep report/badge proof shareable. |
| `monarchinitiative.org` | 65 | `/fix/monarchinitiative.org` routes to paid remediation intake. | Partial-score genetics/data owner: publish missing machine-readable surfaces before agents depend on the source. |
| `fhirfly.io` | 65 | Public profile exists. | Health API owner: make plan/auth/API contract metadata inspectable and monitorable. |
| `hipaaagent.ai` | 80 | Public profile exists. | Compliance-adjacent owner: readiness copy must avoid certification claims and focus on machine-readable contracts. |

## Draft Owner-Channel Angle

Agents looking up health claims, genetic variants, nutrition logs, or product reviews need stable sources they can inspect before use. A good owner path is:

1. Public score report first.
2. High-score sites get monitor/report/badge proof.
3. Partial-score sites get a missing-surface checklist before paid remediation.
4. API-heavy callers get catalog/API-key surfaces only when docs remain useful.

## Guardrails

- Do not claim medical accuracy, clinical endorsement, genetic interpretation quality, supplement efficacy, nutrition accuracy, product-review truth, HIPAA/privacy compliance, regulatory compliance, live data freshness, or comprehensive health coverage.
- Do not claim private demand, completed payments, revenue, customer endorsement, paid ranking placement, preferred inclusion, A2A support while `/.well-known/agent-card.json` is 404, x402/ACP/MPP support for NHS, or score-methodology bypass.
- Refresh live stats, categories, representative profiles, `/fix/{host}` routes, MCP/API/commerce discovery surfaces, and aggregate admin analytics before any external use.

## Handoff

Prepare one gated owner-channel touch, post, or product-handoff test for genetic wellness, health claims, nutrition tracking, supplement evidence, medical-data APIs, or review-source owners. This recurring scout did not post or send.
