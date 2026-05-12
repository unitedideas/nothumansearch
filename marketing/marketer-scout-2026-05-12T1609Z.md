# NHS marketing scout segment - 2026-05-12T16:09Z

Automation: `business-marketer-not-human-search`

## Boundary

No outreach was sent. No posts, browser/Computer Use, account creation, deploys, full recrawls, product-code edits, public submissions, or QLimit/global-queue writes were performed.

## Live surface checks

- `https://nothumansearch.ai/api/v1/stats`: `total_sites=4238`, `avg_score=35`, `top_category=developer`.
- `https://nothumansearch.ai/api/v1/categories`: 14 categories. Largest buckets: `developer=1300`, `ai-tools=892`, `other=775`, `data=403`, `finance=201`.
- `/.well-known/mcp.json`: advertises 11 tools and now describes public categories plus audit-only `other` and `spam`.
- `/llms.txt`: current site count is live and category copy now matches the public/audit-only category split.
- `/api/v1/catalog`: previously verified in the 11:08Z scout as listing score-fix plus Starter, Pro, and Scale API subscriptions.

## Public vertical evidence

These are public `/api/v1/top` results, not private admin/customer rows.

Data/API owner-channel shortlist from `GET /api/v1/top?category=data&limit=8`:

- `dchub.cloud` - score 100, data-center intelligence platform with all 7 signals.
- `api.contrastcyber.com` - score 100, security intelligence API/MCP with all 7 signals.
- `api.boostedchat.com` - score 95, travel booking API/MCP missing schema only.
- `api.theartofservice.com` - score 90, compliance framework API/MCP missing AI-friendly robots and schema.
- `api.headlessoracle.com` - score 90, signed market-state verification API/MCP missing AI-friendly robots and schema.
- `api.agentry.com` - score 90, agent-commerce infrastructure API/MCP missing AI-friendly robots and schema.
- `api.socialintel.dev` - score 90, influencer-search API missing MCP only among major implementation signals.
- `blocklens.co` - score 90, crypto analytics API missing MCP only among major implementation signals.

Developer-tools owner-channel shortlist from `GET /api/v1/top?category=developer&limit=8`:

- `agentprobe.fly.dev` - score 100, agentic commerce readiness testing.
- `xquik.com` - score 100, X automation platform with REST, webhooks, and MCP.
- `mcp.depscope.dev` - score 100, package intelligence for AI agents.
- `deadends.dev` - score 100, error-resolution knowledge base for agents.
- `agentdomainsearch.com` - score 100, agent-first domain registration and DNS management.
- `blackveilsecurity.com` - score 100, DNS/email security scanner.
- `agentndx.ai` - score 100, MCP server directory.
- `entia.systems` - score 100, AI-verified business identity infrastructure.

Communication owner-channel shortlist from `GET /api/v1/top?category=communication&limit=8`:

- `mail.misar.io` - score 100, outreach/email copilot with all 7 signals.
- `resend.com` - score 75, developer email platform missing ai-plugin and AI-friendly robots.
- `secondsim.co.uk` - score 70, business eSIM/WhatsApp number service missing OpenAPI and MCP.
- `postalform.com` - score 65, mail-a-letter API surface missing ai-plugin, AI-friendly robots, and MCP.
- `slack.com` - score 60, AI work platform missing OpenAPI, AI-friendly robots, MCP, and schema.
- `api.slack.com` - score 60, developer docs surface missing OpenAPI, AI-friendly robots, MCP, and schema.
- `pantrypersona.com` - score 60, pantry app for ChatGPT/Claude missing ai-plugin and OpenAPI.
- `kweenkl.com` - score 55, push notifications for AI agents missing ai-plugin, OpenAPI, and AI-friendly robots.

## Draft brief angles

Data/API:

`The data bucket is where agent-readiness becomes operational, not cosmetic. NHS has 403 data sites, including score-100 data-center and security-intelligence examples, plus score-90 market-state, compliance, commerce, and analytics APIs. The owner-channel angle is simple: agents need data sources that can be discovered, called, and verified without browser scraping.`

Developer tools:

`Developer tooling is NHS's densest public category: 1,300 indexed sites. The top results are not generic SaaS pages; they are agent-commerce probes, MCP directories, package-intelligence servers, X automation, DNS/domain tools, and security scanners with complete machine-readable surfaces. This is the strongest channel for showing what an agent-ready site looks like.`

Communication:

`Communication tools have a useful spread because the gaps are concrete. NHS has 118 communication sites; the top result is fully agent-ready, while major platforms and developer docs are often missing OpenAPI, MCP, AI-friendly robots, or schema. That makes the brief a practical owner checklist rather than a generic tools roundup.`

## Duplicate and channel checks

- `ops/sweeper/marketer-inbox.jsonl` had no existing data/API, developer-tools, or communication vertical brief rows before this segment.
- `outreach/distribution_log.csv` is saturated with broad MCP/API/GEO directory PRs, gists, email pitches, and existing NHS score-check action distribution.
- Shared social ledger contains older broad NHS posts and targeted company posts; no current vertical owner-channel brief row was found for these three categories.
- No public action was taken, so no public-action lock was claimed. Any later publication still needs active account verification, duplicate fingerprinting, and a sync-state public-action lock.

## Appended intake rows

Rows appended to `ops/sweeper/marketer-inbox.jsonl`:

- `Prepare data/API owner-channel brief from public top-category evidence.`
- `Prepare developer-tools owner-channel brief from public top-category evidence.`
- `Prepare communication-tools owner-channel brief from public top-category evidence.`
