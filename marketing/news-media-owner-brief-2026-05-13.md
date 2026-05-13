# News/Media Agent-Readiness Brief

Status: prepared, not published.
Automation: `business-marketer-not-human-search`
Prepared: 2026-05-13T06:08Z

## Boundary

No public post, email, submission, browser action, account creation, form submission, deploy, product-code edit, or full recrawl was performed. This is a channel-ready brief for a later locked social/community or owner-channel operator. Publication still requires active account verification, duplicate fingerprinting against the shared social ledger, and a sync-state public-action lock.

## Live Evidence

Sources checked during preparation:

- `https://nothumansearch.ai/api/v1/stats`: `total_sites=4238`, `avg_score=35`, `top_category=developer`.
- `https://nothumansearch.ai/api/v1/categories`: `news=11`, `avg_score=50`.
- `https://nothumansearch.ai/api/v1/top?category=news&limit=8`: public top news/media examples only.
- Admin MCP analytics, last 7 days, aggregate only: `tools/list=99206`, `initialize=12563`, `tools/call=431`, unknown tool names: 0. Query themes included news lookup and current-event/source-specific checks. No raw user identifiers were written.

Top public news/media examples:

- `informedclearly.com` - score 70; publishes `llms.txt`, `ai-plugin.json`, OpenAPI, and Schema.org.
- `hallucinationherald.com` - score 65; publishes `llms.txt`, `ai-plugin.json`, MCP, AI-friendly robots, and Schema.org.
- `biztoc.com` - score 65; publishes `ai-plugin.json`, OpenAPI, structured API, AI-friendly robots, and Schema.org.
- `zadar.tv` - score 55; publishes `llms.txt`, OpenAPI, AI-friendly robots, and Schema.org.
- `aibtc.news` - score 50; publishes `llms.txt`, structured API, AI-friendly robots, and Schema.org.

## Brief Copy

Subject/heading:

`11 news and media sites ranked for agent readiness`

Short post:

NHS currently indexes 11 news and media sites. The category is small, but it shows a useful pattern for publishers: agent visibility is not the same thing as writing about AI.

The strongest examples expose some mix of `llms.txt`, `ai-plugin.json`, OpenAPI, structured API, MCP, AI-friendly robots, and Schema.org. That gives agents a cleaner path to inspect source metadata, citation surfaces, feeds, or subscription context without scraping brittle HTML.

The gap is concrete:

1. A score-70 publisher can already expose `llms.txt`, an AI plugin manifest, OpenAPI, and Schema.org, but still miss MCP, structured API, and AI-friendly robots.
2. A score-65 publisher can expose MCP and `llms.txt`, but still miss OpenAPI or a structured API.
3. For media owners, the fix is usually not "add AI content." It is publishing the public machine-readable surfaces that make the publication verifiable to agents.

Public news/media top list:
`https://nothumansearch.ai/api/v1/top?category=news&limit=8`

Score your own site:
`https://nothumansearch.ai/score`

## Longer Variant

NHS currently indexes 11 news and media sites and ranks them by agent-readiness signals rather than brand popularity.

The current top examples include a global news summary site, an autonomous AI newspaper, a business-news hub, regional publishers, and niche media surfaces. The common thread is not editorial category. It is whether an agent can identify the site, inspect published metadata, and find a machine-readable path before falling back to page scraping.

For publishers, the readiness gap is unusually fixable. `llms.txt`, OpenAPI, an AI plugin manifest, Schema.org, AI-friendly robots, and MCP are public distribution surfaces. They help agents understand what the site is, what endpoints or feeds exist, what the allowed access pattern is, and how to cite or inspect the publication without guessing.

Current public news/media top list:
`https://nothumansearch.ai/api/v1/top?category=news&limit=8`

Live checker:
`https://nothumansearch.ai/score`

## Publication Guard

Before any external use:

- Refresh `/api/v1/stats`, `/api/v1/categories`, and `/api/v1/top?category=news&limit=8`.
- Check the shared social ledger for duplicate fingerprints.
- Verify the active account identity for the selected channel.
- Take a sync-state public-action lock.
- Do not claim private demand, customer behavior, editorial endorsement, paid ranking, score bypass, or investment/media-quality advice.
