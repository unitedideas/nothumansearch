# MCP Marketplace remote-listing candidate

Date: 2026-07-30
Automation: `business-marketer-not-human-search`
Status: no submission, account creation, outreach, public post, browser action, deploy, crawl, or purchase performed

## Candidate

MCP Marketplace is a viable gated directory candidate for the existing Not
Human Search remote MCP server.

- Target: `https://mcp-marketplace.io/`
- Creator documentation: `https://mcp-marketplace.io/docs`
- FAQ: `https://mcp-marketplace.io/faq`
- Community guidelines: `https://mcp-marketplace.io/community-guidelines`
- Terms: `https://mcp-marketplace.io/terms`

The marketplace publicly claims more than 16,000 security-scanned servers.
Its creator documentation explicitly accepts a hosted remote endpoint, supports
Streamable HTTP, allows free listings, and says remote submissions receive
endpoint probing and trust analysis. Public exact-match web searches for
`Not Human Search`, `nothumansearch.ai`, and `ai.nothumansearch/search`
returned no marketplace result. That is candidate evidence only; a later
executor must repeat the search inside the authenticated marketplace before
submitting.

## Verified NHS packet

- Name: Not Human Search
- Publisher: `unitedideas`
- Public repository: `https://github.com/unitedideas/nothumansearch`
- Website: `https://nothumansearch.ai`
- Remote endpoint: `https://nothumansearch.ai/mcp`
- Transport: Streamable HTTP
- Official registry id: `ai.nothumansearch/search`
- Official registry version: `1.7.1`
- Pricing: free MCP server
- Category: Search & Web
- Current public stats: 4,367 indexed sites, average score 38
- Current live tools: 11

Live `initialize` negotiated protocol version `2025-06-18`. Live `tools/list`
returned `search_agents`, `get_site_details`, `get_stats`, `submit_site`,
`check_url`, `verify_mcp`, `register_monitor`, `list_categories`,
`get_top_sites`, `recent_additions`, and `find_mcp_servers`.

NHS `/mcp`, `/.well-known/mcp.json`, `/.well-known/agent.json`,
`/.well-known/commerce.json`, `/.well-known/ai-plugin.json`, `/llms.txt`,
`/openapi.yaml`, `/api/v1`, `/api/v1/catalog`, `/score`, `/monitor`,
`/report`, `/newest`, and `/top` returned HTTP 200.
`/.well-known/agent-card.json` returned HTTP 404, so the listing must not claim
A2A or Agent Card support.

Suggested listing description:

> Search the agentic web by readiness. Find agent-ready APIs and services,
> inspect public discovery signals, verify MCP endpoints, and monitor score
> changes through 11 remote tools.

## Account and execution blocker

Submission is account-gated. No existing MCP Marketplace fingerprint was found
in the NHS marketer inbox, NHS distribution log, or portfolio social ledgers.
No credential under the local Keychain service names
`mcp-marketplace-account` or `mcp-marketplace-io` was present. This does not
rule out an existing OAuth session, but the recurring worker has no verified
Foundry/Owl-owned publisher identity for the target and must not create one.

A later execution worker may proceed only if it can verify an existing
Foundry/Owl-owned MCP Marketplace publisher identity. It must search the
authenticated directory for the registry id, endpoint, brand, domain, and
repository; check duplicate ledgers and sync-state public-action locks; claim
the public-action lock; submit one free remote-server listing; and record the
receipt or live URL in `outreach/distribution_log.csv`.

## Claim boundary

The community guidelines prohibit promotional reviews and listings that exist
only to drive external traffic. Keep the submission about the actual free MCP
tool surface. Do not turn the listing into score-fix advertising, buy
placement, connect Stripe payouts, create reviews, or claim install volume.

Do not claim MCP Marketplace endorsement, approval before receipt, security
certification, customer demand, real-user tool success, completed payments,
revenue, uptime proof, paid placement, preferred inclusion, A2A support while
the Agent Card route is 404, x402/ACP/SPT/MPP support for NHS, or a
score-methodology bypass. Do not publish raw user-agent logs, raw MCP queries,
private monitor rows, private score-fix rows, emails, payment identifiers,
checkout URLs, API keys, or customer identifiers.
