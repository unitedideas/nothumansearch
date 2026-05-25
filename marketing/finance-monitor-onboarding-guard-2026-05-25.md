# Finance Monitor Onboarding Guard - 2026-05-25

Purpose: prepare one gated owner-channel or product-handoff test for finance, market-data, ETF, trading, and investment-research sites where the free monitor path needs score-band framing before any score-fix or public copy.

## Evidence

- Live public stats: 4,172 indexed sites, average score 35, top category developer.
- Live finance category: 192 sites, average score 40.
- Live public finance top examples: terminalfeed.io 100, chartlibrary.io 100, prereason.com 100, devdrops.run 95, razorpay.com 90, ticksurfers.com 90, lendtrain.com 85, debridge.com 80.
- Live public data top examples relevant to finance/infrastructure: api.headlessoracle.com 100, api.contrastcyber.com 100, dchub.cloud 95, daedalmap.com 90, api.theartofservice.com 90, api.agentry.com 90, api.socialintel.dev 90, blocklens.co 90.
- Live surfaces checked: `/score`, `/monitor`, `/report`, `/newest`, `/top`, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/commerce.json`, `/.well-known/ai-plugin.json`, `/api/v1`, `/api/v1/catalog`, `/llms.txt`, `/openapi.yaml`, and `/feed.xml` returned 200.
- A2A blocker: `/.well-known/agent-card.json` returned 404, so do not claim Agent Card or A2A readiness.
- MCP aggregate, 7 days: tools/list 175,230, initialize 25,162, tools/call 379; top tools were search_agents 147, check_url 87, get_site_details 58, get_stats 26, submit_site 20, verify_mcp 13, register_monitor 4.
- Aggregate query themes, 7 days: model/API 7, finance/market-data 4, local/civic 3, publisher/news 1, education/research 1.
- Traffic aggregate, 168 hours: `/` 3,370, `/badge/xquik.com.svg` 2,667, `/.well-known/commerce.json` 1,337, `/site/xquik.com` 1,108, `/.well-known/ai-plugin.json` 609, `/llms.txt` 442, `/openapi.yaml` 377, `/api/v1/catalog` 319, `/api/v1/checkout` 255, `/api/v1/quote` 255.
- Monitor worker proof, 2026-05-25: completed normally with 5 due monitors; aggregate outcome was two first-check zero-score quarantines, two first-check partial/low-score finance or market-data style monitors, and one stable high-score monitor.

## Segment

Finance and market-data owners often expose APIs, feeds, pricing, fund pages, or research surfaces that agents try to call directly. The marketing angle is not that NHS verifies financial truth. It is that NHS can show whether public source contracts are machine-readable enough for agents before owners rely on AI/browser clients discovering them correctly.

Use the free monitor as the first conversion step:

- High-score finance/data owners: free monitor, report page, and badge proof.
- Partial-score owners: `/score` first, then missing-surface checklist before any paid remediation.
- Zero-score or quarantined monitor cases: private review/remediation handoff only; do not use them as public proof.
- API-heavy callers: API-key/catalog surfaces only when the current docs and plan metadata remain useful.

## Candidate Channel Shapes

- Owner-channel post for finance API and market-data product builders.
- Directory candidate for finance/data API communities that accept machine-readable source tooling.
- Product-handoff test that links `/score`, `/monitor`, and `/report` from a finance-specific missing-surface checklist.

## Boundaries

Do not claim trading performance, investment advice, market-data accuracy, price freshness, fund data correctness, compliance certification, security certification, uptime, API reliability, monitor registration volume, customer demand, private demand, paid leads, completed payments, revenue, endorsement, paid placement, preferred inclusion, x402/ACP/SPT/MPP support for NHS, A2A support while `/.well-known/agent-card.json` is 404, or score-methodology bypass.

Do not publish raw MCP queries, raw user-agent strings, private monitor rows, customer identifiers, payment identifiers, raw checkout URLs, or row-level score-fix/admin data.
