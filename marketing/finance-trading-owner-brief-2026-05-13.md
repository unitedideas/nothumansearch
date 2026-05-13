# Finance/Trading Agent-Readiness Brief

Status: prepared, not published.
Automation: `business-marketer-not-human-search`
Prepared: 2026-05-13T04:08Z

## Boundary

No public post, email, submission, browser action, account creation, or form submission was performed. This is a channel-ready brief for a later locked social/community operator. Publication still requires active account verification, duplicate fingerprinting against the shared social ledger, and a sync-state public-action lock.

## Live Evidence

Sources checked during preparation:

- `https://nothumansearch.ai/api/v1/stats`: `total_sites=4238`, `avg_score=35`, `top_category=developer`.
- `https://nothumansearch.ai/api/v1/categories`: `finance=201`, `developer=1300`, `ai-tools=892`, `data=403`.
- `https://nothumansearch.ai/api/v1/top?category=finance&limit=5`: public top finance/trading examples only.
- Admin MCP analytics, last 7 days, aggregate only: `tools/list=98366`, `initialize=12368`, `tools/call=431`, unknown tool names: 0. Query themes included finance/trading research, stock-market strategy, and market-analysis-agent patterns. No raw user identifiers were written.

Top public finance/trading examples:

- `terminalfeed.io` - score 100, all seven agent-readiness signals.
- `chartlibrary.io` - score 100, all seven signals.
- `prereason.com` - score 100, all seven signals.
- `devdrops.run` - score 95, missing Schema.org only.
- `ticksurfers.com` - score 90, missing MCP among major implementation signals.

## Brief Copy

Subject/heading:

`201 finance sites ranked for agent readiness`

Short post:

NHS currently indexes 201 finance sites. The top of the category shows the useful distinction for trading and market-data tools: agent readiness is not a finance-content claim, it is whether an agent can inspect and call the service without guessing.

The public top list includes market dashboards, stock-pattern cohort APIs, market-context APIs, x402 data APIs, and rules-based trading indicators.

The pattern is concrete:

1. The score-100 examples expose the full machine-readable set: `llms.txt`, `ai-plugin.json`, OpenAPI, structured API, MCP, AI-friendly robots, and Schema.org.
2. The score-90/95 examples are already callable or documented, but one public readiness signal is missing.
3. For finance API owners, the gap is usually not "add AI." It is making the existing service discoverable and verifiable before an agent tries scraping a dashboard.

Public finance top list:
`https://nothumansearch.ai/api/v1/top?category=finance&limit=5`

Score your own site:
`https://nothumansearch.ai/score`

## Longer Variant

NHS currently indexes 201 finance sites and ranks them by agentic readiness rather than popularity.

The highest-scoring examples are specific: market dashboards, stock-pattern cohort intelligence, market-context APIs, x402 data APIs, and trading-indicator platforms. These are services an agent may actually need to inspect before running a research or trading workflow.

The readiness gaps are also specific. A score-90 finance site may already publish `llms.txt`, an AI plugin manifest, OpenAPI, a structured API, AI-friendly robots, and Schema.org, but still miss MCP. A score-95 site may only be missing Schema.org. That is a fixable distribution problem for API owners: publish the public surfaces that let agents discover, verify, and call the service without guessing.

Current public finance top list:
`https://nothumansearch.ai/api/v1/top?category=finance&limit=5`

Live checker:
`https://nothumansearch.ai/score`

## Publication Guard

Before any external use:

- Refresh `/api/v1/stats`, `/api/v1/categories`, and `/api/v1/top?category=finance&limit=5`.
- Check the shared social ledger for duplicate fingerprints.
- Verify the active account identity for the selected channel.
- Take a sync-state public-action lock.
- Do not claim private demand, customer behavior, investment advice, paid placement, or score bypass.
