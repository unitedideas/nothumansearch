# Show HN Drafts — Not Human Search

Launch-ready. Pick the variant closest to the audience you're seeing on HN that day.

---

## Variant A — "Search engine for AI agents" (direct)

**Title (≤80 chars):**
`Show HN: Not Human Search – a search engine for AI agents`

**Body:**

```
I've been building "Not Human Search" (nothumansearch.ai) — a search engine where AI agents discover agent-ready services, APIs, and tools. Instead of ranking for humans, it ranks for non-humans.

Every indexed site is scored 0-100 on 7 agentic signals:
- llms.txt (25 pts)
- ai-plugin.json (20)
- OpenAPI spec (20)
- structured API (15)
- MCP server (10)
- robots.txt AI crawler rules (5)
- Schema.org (5)

960+ sites indexed so far. The crawler auto-discovers new domains from the official MCP registry + awesome-mcp lists + llms.txt directories.

NHS is itself an MCP server:
  claude mcp add --transport http nothumansearch https://nothumansearch.ai/mcp

Tools: search_agents, get_site_details, get_stats, submit_site, register_monitor.

There's also a REST API (openapi.yaml, ai-plugin.json, llms.txt all served) and a /score endpoint that rates any URL on demand.

Scoring methodology + guides: https://nothumansearch.ai/guide

Happy to answer questions about the crawler, categorization, or why "agent-first" signals matter more than PageRank now.
```

---

## Variant B — "the web is forking" (provocative angle)

**Title:**
`Show HN: The web is splitting in two. I built the search engine for the other half.`

**Body:**

```
Google indexes ~50B pages for humans. But AI agents don't read pages — they read APIs, OpenAPI specs, llms.txt manifests, and MCP tool definitions. There's a growing corpus of "agent-first" sites whose value is invisible to Google because Google doesn't score the right signals.

So I built nothumansearch.ai. 960+ sites indexed so far, each scored 0-100 on 7 agentic signals: llms.txt, ai-plugin.json, OpenAPI, structured API, MCP, robots.txt AI rules, Schema.org.

The index grows via:
- Auto-discovery from the official MCP registry
- Weekly pulls from awesome-mcp-servers and llms.txt directories
- A /score endpoint anyone can call to add their site in real time

NHS is itself accessible as an MCP server at nothumansearch.ai/mcp — so agents wire it in at build time, not scraped at runtime:
  claude mcp add --transport http nothumansearch https://nothumansearch.ai/mcp

Interesting observations so far:
- 90% of sites publishing an llms.txt don't also expose an OpenAPI spec; most are content sites, not APIs.
- The 10% of sites scoring 70+ are disproportionately AI-tools and developer-APIs categories.
- Most sites claiming "AI-friendly" in their copy have zero programmatic signals.

Happy to go deep on crawler design, scoring methodology, or why I think a separate index for agents is a real long-term category.
```

---

## Variant C — technical deep-dive (for the r/programming-adjacent HN crowd)

**Title:**
`Show HN: I built a crawler that scores sites on how well they serve AI agents`

**Body:**

```
Wrote a Go crawler that evaluates any site on 7 "agentic readiness" signals — whether it publishes llms.txt, ai-plugin.json, OpenAPI spec, structured API, MCP server, robots.txt AI crawler rules, and Schema.org markup. Every detection is content-verified, not just a HEAD request (avoids false positives on /docs or /api routes that return HTML).

Detection gotchas I hit:
- /api and /docs often return rendered HTML pages, not JSON. Score requires 3+ API-ish indicators in the body.
- ai-plugin.json is a manifest, not a capability claim — still need to fetch and parse to verify.
- MCP detection at /mcp requires a JSON-RPC `initialize` call; just probing for 200 gives too many false positives.
- robots.txt AI rules: scoring explicit allows/denies for GPTBot, ClaudeBot, PerplexityBot, CCBot etc. not generic `User-agent: *`.

Aggregate results at nothumansearch.ai. 960+ sites indexed. Full scoring details at /guide, REST API at /api/v1, OpenAPI at /openapi.yaml, and NHS is itself an MCP server at /mcp.

Agent-first filter in the DB layer means a site only shows up in results if it has at least one programmatic signal agents can discover at build time. Schema.org + robots.txt alone don't qualify.

Happy to go deeper on any part of the stack (Go HTTP, Postgres FTS, ranking, MCP server impl) in comments.
```

---

## Posting guidance

- Post time: Tuesday-Thursday, 06:30-08:00 PT (noon UTC peak HN traffic)
- No karma penalty: use your main HN account; Show HN tag is fine
- Reply to every comment in the first 2 hours; HN ranks by reply velocity
- If someone asks "why not just use Google?", the answer is in Variant B's framing
- If someone asks "what's the revenue model?", answer: currently free to index/query; monetization will be premium API tier + premium placement in search results once there's organic demand signal (there isn't yet, so don't optimize for it)

## What not to post

- Do NOT include "AI-powered" or "revolutionary" in the title — classic HN flag
- Do NOT mention you're a founder/solo — just show the product
- Do NOT post screenshots in the body (HN strips them); link to /score for a live demo instead
