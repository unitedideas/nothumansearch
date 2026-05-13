# NHS Communication Tools Owner Brief

Status: prepared, not published.
Automation: `business-marketer-not-human-search`
Prepared: 2026-05-13T20:07Z

## Boundary

No outreach, public post, browser action, account creation, product-code edit, deploy, full recrawl, or QLimit/global-queue write was performed. This artifact is a channel brief for a later gated operator. External use still requires active account verification, duplicate-fingerprint checks, and a sync-state public-action lock.

## Live Evidence

Public surfaces checked during preparation:

- `https://nothumansearch.ai/api/v1/stats`: `total_sites=4169`, `avg_score=35`, `top_category=developer`.
- `https://nothumansearch.ai/api/v1/categories`: 14 categories. The communication bucket has `117` sites with average score `38`.
- `https://nothumansearch.ai/api/v1/top?category=communication&limit=8`: the top communication example scores 100/100 and exposes all seven public agent-readiness signals.
- `https://nothumansearch.ai/llms.txt`: communication is listed as a public category; `other` and `spam` are audit-only.

Public top-category examples observed during preparation:

- `mail.misar.io` - score 100, email/outreach copilot with all seven signals present.
- `resend.com` - score 75, developer email platform with `llms.txt`, OpenAPI, structured API, MCP, and Schema.org, but missing AI plugin and AI-friendly robots.
- `secondsim.co.uk` - score 70, business eSIM/WhatsApp number surface missing OpenAPI and MCP.
- `postalform.com` - score 65, letter-mailing API surface missing AI plugin, AI-friendly robots, and MCP.
- `slack.com` and `api.slack.com` - score 60, major communication surfaces missing OpenAPI, AI-friendly robots, MCP, and Schema.org in the current NHS crawl.
- `pantrypersona.com` - score 60, ChatGPT/Claude pantry workflow with MCP and structured API but missing AI plugin and OpenAPI.
- `kweenkl.com` - score 55, push notifications for AI agents with structured API and MCP but missing AI plugin, OpenAPI, and AI-friendly robots.

No raw user identifiers, private customer rows, API keys, checkout URLs, or private query logs were written.

## Brief Copy

Subject/heading:

`117 communication-tool sites are in Not Human Search. The gaps are visible enough to fix.`

Short post:

Not Human Search currently tracks 117 communication-tool sites in the public communication bucket.

The spread is useful because the owner-side work is concrete. The top example exposes the full agent-readiness surface: `llms.txt`, an AI plugin manifest, OpenAPI, a structured API, MCP, an AI-friendly robots policy, and Schema.org.

Several larger communication and messaging surfaces score lower because agents cannot see a complete machine-readable contract from the public web. Common gaps are missing OpenAPI, missing MCP where operational actions exist, no AI-friendly robots policy, or no structured schema tying the product surface together.

For email, messaging, notification, and collaboration-tool owners, the practical checklist is:

1. Publish `llms.txt` with the product scope, API links, pricing/contact boundaries, and safe-use notes.
2. Keep OpenAPI current for send, inbox, webhook, template, status, or account endpoints.
3. Add MCP only where agents can perform real, bounded operations.
4. Make `/.well-known/ai-plugin.json` point at the maintained public API surface.
5. Register monitoring after the score is fixed so deploys do not silently remove the evidence path.

Search the public communication bucket:

`https://nothumansearch.ai/api/v1/top?category=communication&limit=25`

Check a communication-tool site:

`https://nothumansearch.ai/score`

## Owner/Buyer Angle

This is for email providers, messaging products, notification tools, collaboration platforms, WhatsApp/SMS/eSIM utilities, and AI-agent communication layers. The sell is not ranking placement. The sell is making the operational surface legible to agents and catching drift before customers do.

## Publication Guard

Before external use:

- Refresh `/api/v1/stats`, `/api/v1/categories`, `/api/v1/top?category=communication&limit=8`, `/llms.txt`, and `/.well-known/mcp.json`.
- Check the shared social ledger for duplicate fingerprints.
- Verify the active account identity for the selected channel.
- Take a sync-state public-action lock.
- Do not claim private demand, revenue, conversion, paid ranking placement, preferred inclusion, or score-methodology bypass.
