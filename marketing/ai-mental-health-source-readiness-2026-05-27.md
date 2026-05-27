# AI mental-health source readiness - 2026-05-27

Automation: `business-marketer-not-human-search`

Scope: no public action, outreach, browser/Computer Use, account creation, product-code edit, deploy, full recrawl, checkout completion, or QLimit/global-queue write. This is a sanitized scout artifact for a later gated operator.

## Evidence

- Public stats: `total_sites=4180`, `avg_score=35`, and `top_category=developer`.
- Public categories: `developer=1234`, `ai-tools=901`, `other=778`, `data=401`, `finance=193`, `productivity=172`, `ecommerce=149`, `communication=120`, `security=112`, `health=60`, `jobs=26`, and `education=21`.
- Live public surfaces returned 200: `/llms.txt`, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/commerce.json`, `/.well-known/ai-plugin.json`, `/api/v1`, `/api/v1/catalog`, `/api/v1/api-keys/subscribe`, `/openapi.yaml`, `/feed.xml`, `/score`, `/monitor`, `/report`, and `/mcp`.
- `/.well-known/agent-card.json` returned 404, so A2A and Agent Card claims remain blocked.
- Aggregate MCP analytics, 7 days: `tools/list=183202`, `initialize=28209`, and `tools/call=400`.
- Aggregate MCP tool calls, 7 days: `search_agents=145`, `check_url=89`, `get_site_details=68`, `get_stats=28`, `submit_site=21`, `verify_mcp=14`, `find_mcp_servers=9`, `recent_additions=7`, `list_categories=7`, `get_top_sites=7`, and `register_monitor=5`.
- Aggregate MCP query themes still include health-adjacent and evidence-sensitive lookups: `MTHFR mutation ADHD autism systematic review meta-analysis`, plus model/provider, local-event, ecommerce, and product-review themes.
- Public `ai-tools` top list includes health-adjacent high-score AI sites: `teenanxiety.ai=100` and `teenadhd.ai=100`, alongside non-health AI tools and Foundry-owned dogfood examples.
- Public `health` top list includes `emorahealth.com=100`, `zgts.in=100`, `opdstar.com=80`, `plith.ai=80`, `monarchinitiative.org=65`, `fhirfly.io=65`, and several lower-score health or clinical workflow examples.
- Score-band route checks: `/site/teenanxiety.ai` showed `100/100` and `/fix/teenanxiety.ai` returned the already-meets-target monitor handoff. `/site/monarchinitiative.org` showed `65/100` and `/fix/monarchinitiative.org` returned the paid remediation intake.
- Aggregate traffic, 168 hours: `/=3386`, `/badge/xquik.com.svg=2651`, `/.well-known/commerce.json=1288`, `/site/xquik.com=1108`, `/.well-known/ai-plugin.json=574`, `/llms.txt=424`, `/openapi.yaml=374`, `/api/v1/catalog=310`, `/badge/aidevboard.com.svg=289`, `/robots.txt=281`, `/badge/8bitconcepts.com.svg=265`, `/api/v1/checkout=245`, `/api/v1/quote=245`, `/api/v1/search=233`, and `/api/v1/submit=146`.
- Aggregate referrers, 168 hours: `google.com=640`, `/score=84`, and `aurelianflo.com=57`, with canonical-domain aliases still material.

## Read

This is not another generic healthcare brief. The narrower owner segment is AI mental-health, clinician-reviewed guidance, ADHD/anxiety resources, genetics/wellness evidence, and phenotype or clinical knowledge-base owners that need source contracts agents can inspect without turning NHS into a medical endorsement engine.

The useful conversion shape is score-band aware:

1. High-score AI mental-health or health-data owners get free monitor/report/badge proof.
2. Partial-score health or clinical knowledge-base owners get `/score` and a missing-surface checklist before any remediation offer.
3. API-heavy health-data owners can be routed to API-key/catalog surfaces only when the docs remain useful and price-copy is not overstated.
4. Zero-score or quarantined monitor outcomes stay private/admin-only.

## Candidate Test

Prepare one gated owner-channel touch, channel post, or product-handoff test for AI mental-health, ADHD/anxiety guidance, genetics/wellness evidence, phenotype knowledge-base, or clinical source-contract owners.

Suggested framing:

- Agents increasingly need to inspect the source contract behind health-adjacent answers.
- NHS can show whether the public site exposes `llms.txt`, OpenAPI, API/MCP, Schema.org, robots policy, monitorable score drift, and score-band routing.
- A high score is a proof surface; a partial score is a missing-surface checklist, not a claim about clinical quality.

## Boundaries

Do not imply AI mental-health, ADHD, anxiety, genetics, wellness, clinical, phenotype, healthcare, health-data, top-list, referrer, badge, or profiled domains are customers, partners, endorsements, paid leads, monitor registrations, badge-install consent, private demand, completed payments, revenue, medical accuracy, clinical endorsement, therapeutic efficacy, diagnosis quality, genetic interpretation quality, supplement efficacy, nutrition accuracy, HIPAA/privacy compliance, regulatory compliance, live data freshness, clinical reliability, provider-directory accuracy, crawler compliance, SEO lift, uptime proof, A2A support while `/.well-known/agent-card.json` is 404, x402/ACP/SPT/MPP support for NHS, paid ranking placement, preferred inclusion, or score-methodology bypass.

Do not publish raw user-agent strings, raw MCP queries, private monitor rows, raw checkout URLs, payment identifiers, buyer emails, private score-fix rows, or private customer identifiers.

## Next Gated Action

Use this packet for exactly one gated owner-channel touch, channel post, directory candidate, or product-handoff test after refreshing public stats, health and ai-tools top lists, `/score`, `/monitor`, `/report`, representative `/site/{host}` pages, high-score and partial-score `/fix/{host}` routes, MCP/discovery surfaces, aggregate MCP analytics, aggregate traffic, and latest monitor worker proof.
