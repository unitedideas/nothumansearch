# Feed Syndication Owner Conversion - 2026-05-22

Automation: `business-marketer-not-human-search`

Scope: no public action, no outreach, no browser, no deploy. This packet is a gated owner-conversion artifact for a later execution worker.

## Fresh Evidence

- Public stats: `4,171` indexed sites, average score `35`, top category `developer`.
- Public categories: `developer=1,228`, `ai-tools=897`, `other=777`, `data=402`, `finance=194`, `productivity=174`, `ecommerce=148`, `communication=118`, `security=114`, `health=58`, `jobs=27`, `education=21`, `news=12`, `spam=1`.
- Live discovery surfaces:
  - `/feed.xml`: HTTP 200, RSS title `Not Human Search - New Agent-Ready Sites`, lastBuildDate `Fri, 22 May 2026 18:08:45 +0000`.
  - `/newest`, `/digest`, `/top`, `/mcp-servers`, `/openapi-apis`, `/llms-txt-sites`: HTTP 200.
  - `/llms.txt`, `/.well-known/mcp.json`, `/.well-known/agent.json`, `/.well-known/commerce.json`, `/.well-known/ai-plugin.json`, `/api/v1`, `/api/v1/catalog`, `/api/v1/quote`, `/openapi.yaml`, `/score`, `/monitor`, `/report`, `/robots.txt`, and `/mcp`: HTTP 200.
  - `/.well-known/agent-card.json`: HTTP 404.
- Aggregate traffic, last 168 hours:
  - root `/` = `3,369`.
  - `/badge/xquik.com.svg` = `2,385`; `/site/xquik.com` = `801`.
  - `/.well-known/commerce.json` = `1,503`.
  - `/.well-known/ai-plugin.json` = `693`; `/llms.txt` = `448`; `/openapi.yaml` = `421`.
  - `/api/v1/catalog` = `325`; `/robots.txt` = `303`; `/api/v1/checkout` = `295`; `/api/v1/quote` = `295`.
  - `/api/v1/search` = `179`; `/api/v1/submit` = `148`; `/about` = `99`; `/top` = `95`; `/score` = `78`; `/guide` = `74`; `/digest` = `65`; `/newest` = `65`.
- Aggregate referrers, last 168 hours:
  - `google.com=542`, `www.google.com=47`.
  - canonical and alias referrers remain material across `nothumansearch.ai`, `nothumansearch.com`, `www`, `http`, and the Fly app host.
  - `/score` referrer = `100`; `/top` referrer = `42`; `/site/chainray.online` referrer = `38`; `aurelianflo.com` referrer = `34`.
- Aggregate MCP analytics, last 7 days:
  - `tools/list=169,147`, `initialize=27,290`, `tools/call=181`.
  - tool calls include `search_agents=84`, `get_site_details=30`, `check_url=20`, `get_stats=19`, `get_top_sites=6`, `find_mcp_servers=6`, `recent_additions=6`, `list_categories=5`, `submit_site=3`, `verify_mcp=2`.
  - visible aggregate client families include Cherry Studio, Claude Code, `MCP-Catalog-Bot`, `MCPScoringEngine`, and `mcp-verify`.
- Feed examples are mixed score bands:
  - `amalgix.io`: public profile `100/100`; `/fix/amalgix.io` routes to the high-score monitor/report handoff.
  - `manifest.ly`: public profile `65/100`; `/fix/manifest.ly` routes to score-fix remediation intake.

## Segment

This run should treat RSS/feed readers and directory-page readers as a distinct owner-conversion segment. The current feed is not just content distribution; it is a recurring public list of newly indexed agent-ready sites with score bands. That creates a clean routing test:

1. feed subscribers and `/newest` readers should be routed to public profiles first;
2. high-score entries should route to free monitor registration, report sharing, and badge proof;
3. partial-score entries should route to `/score` and a missing-surface checklist before remediation;
4. protocol-directory readers should be routed to the matching discovery page first, then to `/score` or API-key/catalog surfaces by intent;
5. `/.well-known/agent-card.json` remains a blocker for A2A/Agent Card directory claims.

## Draft Positioning

For feed and directory readers:

> The NHS feed is a live stream of newly indexed agent-readable sites. Each entry should make the next action clear: inspect the public profile, monitor a strong score, or fix missing machine-readable surfaces after a score check.

For site owners:

> If your site appears in the newest feed with a strong score, monitor it and use the public report or badge as proof. If it appears with a partial score, start with the public score details before paying for remediation.

## Candidate Execution Tests

1. Prepare a feed-reader conversion test: RSS item or `/newest` entry -> public profile -> high-score monitor/report/badge proof or partial-score `/score` checklist.
2. Prepare a no-submit directory syndication packet for RSS/feed or agent-directory maintainers that already consume protocol directories, keeping A2A claims blocked until Agent Card exists.
3. Queue a product handoff only if a later worker confirms `/feed.xml`, `/newest`, or protocol-directory pages lack clear routes to `/score`, `/monitor`, `/report`, badges, and API catalog surfaces.

## Guardrails

- Do not imply feed items, directory-page traffic, searched domains, profile pages, badge routes, API/catalog routes, Google traffic, alias referrers, or external referrers prove customers, endorsements, partners, paid leads, monitor registrations, badge-install consent, private demand, completed payments, revenue, crawler compliance, SEO lift, or uptime proof.
- Do not claim A2A support while `/.well-known/agent-card.json` is 404.
- Do not claim x402, ACP, MPP, or completed-payment support from catalog/quote/checkout traffic.
- Do not sell ranking placement, preferred inclusion, or score-methodology bypass.
- Label Foundry-owned dogfood examples before using them in any proof packet.
- Before external use, refresh public stats, discovery surfaces, `/feed.xml`, `/newest`, `/digest`, protocol-directory pages, aggregate MCP analytics, aggregate traffic, representative high-score and partial-score `/site/{host}` pages, and high-score plus partial-score `/fix/{host}` behavior.
