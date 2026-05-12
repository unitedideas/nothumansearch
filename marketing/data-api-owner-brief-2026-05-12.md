# Data/API Agent-Readiness Brief

Status: prepared, not published.
Automation: `business-agent-not-human-search`
Prepared: 2026-05-12T17:00Z

## Boundary

No public post, email, submission, browser action, or form submission was performed. This is a channel-ready brief for a later locked social/community operator. Any publication still needs active account verification, duplicate fingerprinting against the shared social ledger, and a sync-state public-action lock.

## Live Evidence

Sources checked during preparation:

- `https://nothumansearch.ai/api/v1/stats`: `total_sites=4238`, `avg_score=35`, `top_category=developer`.
- `https://nothumansearch.ai/api/v1/categories`: `data=403`, `developer=1300`, `ai-tools=892`, `finance=201`, `ecommerce=149`, `communication=118`, `security=116`.
- `https://nothumansearch.ai/api/v1/top?category=data&limit=8`: public top data/API examples only.

Top public data/API examples:

- `dchub.cloud` - score 100, all seven agent-readiness signals.
- `api.contrastcyber.com` - score 100, all seven signals; security intelligence API/MCP.
- `api.boostedchat.com` - score 95, missing Schema.org only.
- `api.theartofservice.com` - score 90, missing AI-friendly robots and Schema.org.
- `api.headlessoracle.com` - score 90, missing AI-friendly robots and Schema.org.
- `api.agentry.com` - score 90, missing AI-friendly robots and Schema.org.
- `api.socialintel.dev` - score 90, missing MCP among major implementation signals.
- `blocklens.co` - score 90, missing MCP among major implementation signals.

## Brief Copy

Subject/heading:

`403 data/API sites ranked for agent readiness`

Short post:

NHS currently indexes 403 data/API sites. The top of the category shows what "agent-ready" means in practice: data-center intelligence, security intelligence, travel booking, compliance frameworks, market-state verification, agent commerce, influencer search, and crypto analytics.

The pattern is concrete:

1. The score-100 examples expose the full machine-readable set: `llms.txt`, `ai-plugin.json`, OpenAPI, structured API, MCP, AI-friendly robots, and Schema.org.
2. The score-90/95 examples are not generic SaaS pages. They are callable APIs with one or two missing public signals, usually MCP, AI-friendly robots, or Schema.org.
3. For data providers, the gap is usually not "build AI." It is making the existing product discoverable and verifiable for agents before the agent tries browser scraping.

Public category page:
`https://nothumansearch.ai/api/v1/top?category=data&limit=8`

Score your own site:
`https://nothumansearch.ai/score`

## Longer Variant

NHS currently indexes 403 data/API sites and ranks them by agentic readiness, not generic popularity.

The highest-scoring examples are useful because they are specific: data-center intelligence, security intelligence, travel booking, compliance frameworks, market-state verification, agent commerce, influencer search, and crypto analytics. These are services an agent may actually need to call.

The readiness gaps are also specific. Several score-90/95 APIs already expose `llms.txt`, OpenAPI, structured API, and an AI plugin manifest, but miss MCP, AI-friendly robots, or Schema.org. That is a fixable distribution problem for API owners: publish the public surfaces that let agents discover, verify, and call the service without guessing.

Current public data/API top list:
`https://nothumansearch.ai/api/v1/top?category=data&limit=8`

Live checker:
`https://nothumansearch.ai/score`

## Publication Guard

Before any external use:

- Refresh `/api/v1/stats`, `/api/v1/categories`, and `/api/v1/top?category=data&limit=8`.
- Check the shared social ledger for duplicate fingerprints.
- Verify the active account identity for the selected channel.
- Take a sync-state public-action lock.
- Do not claim private demand, customer behavior, paid placement, or score bypass.
