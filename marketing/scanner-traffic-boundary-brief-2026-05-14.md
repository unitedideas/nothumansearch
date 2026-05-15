# Scanner-Traffic Boundary Brief

Run: 2026-05-15T01:08Z  
Agent: `business-marketer-not-human-search`

## Why This Exists

NHS is now getting enough machine traffic that scanner paths show up in the 14-day route list. That is useful operational evidence, but it should not be treated as buyer demand, owner interest, or agent-commerce conversion.

The marketing boundary is simple: scanner probes prove that bots inspect the surface; they do not prove that site owners want remediation or that agents are buying.

## Live Aggregate Evidence

Public state refreshed during this run:

- `/api/v1/stats`: 4,172 sites, average score 35, top category `developer`.
- `/api/v1/categories`: 14 categories, including audit-only `other` and `spam`.
- `/.well-known/mcp.json`: 11 MCP tools advertised.
- `/api/v1/catalog`: score-fix plus starter/pro/scale API plans advertised.

Admin aggregate traffic over 336 hours:

- `/`: 3,866
- `/.well-known/commerce.json`: 1,329
- `/badge/xquik.com.svg`: 904
- `/api/v1/catalog`: 299
- `/api/v1/quote`: 279
- `/api/v1/checkout`: 279
- `/credentials.json`: 65
- `/secrets.json`: 64

Safety checks on scanner-like paths:

- `/credentials.json`: HTTP 404
- `/secrets.json`: HTTP 404
- `/.env`: HTTP 410
- `/wp-login.php`: HTTP 410

## Marketing Use

Use this as an internal guardrail for future owner-channel copy and scout reports:

1. Commerce and catalog routes can support agent-commerce conversion tests because they are explicit product surfaces.
2. Badge/profile loops can support owner-conversion tests when paired with public profile pages and aggregate counts.
3. Credential, secret, WordPress, and environment-file probes are scanner noise unless a later security/analytics worker classifies them differently.

## Draft Copy Boundary

Safe framing:

> NHS sees both explicit agent-discovery traffic and generic scanner traffic. The useful signal is not that bots probe everything; it is whether your public agent surfaces tell legitimate agents what exists, how to call it, and what not to trust.

Unsafe framing:

> Agents are looking for secrets on NHS.

Do not use the unsafe framing. It implies intent and could make ordinary scanner noise sound like customer demand.

## Follow-Up Row

Queue a product/analytics handoff to label scanner-like routes separately in future marketer scout reports. The goal is not a product-code change from this runtime; it is a later implementation that keeps demand claims clean.
