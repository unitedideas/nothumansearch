# The web is not built for AI agents. I built a search engine that is.

**canonical_url:** https://nothumansearch.ai
**tags:** ai, agents, mcp, llm
**cover_image:** [TODO: add OG image]

---

Google indexes the web for humans. Type "best weather API for my app," get 10 blog posts, a paywalled Medium article, and three Stack Overflow answers from 2017.

Point an agent at that same query. It spends five tool calls navigating SERPs, hits a Cloudflare challenge, and bottoms out on a login wall.

The problem isn't the agent. The problem is the index.

## What agents actually need

An agent doesn't need a blog post *about* a weather API. It needs:

- A machine-readable spec (OpenAPI, ai-plugin.json)
- A real JSON endpoint that returns data, not HTML
- An MCP server it can connect to directly
- An `llms.txt` that tells it how to navigate the site
- Permission to crawl, declared in `robots.txt`

These signals exist, but they're scattered. No one has indexed the agent-accessible web.

## The index

**Not Human Search** crawls sites and grades them on seven agent-readiness signals:

| Signal | Points |
|---|---|
| llms.txt | 25 |
| ai-plugin.json | 20 |
| OpenAPI spec | 20 |
| Structured API | 15 |
| MCP server | 10 |
| AI-friendly robots.txt | 5 |
| schema.org markup | 5 |

Max score: 100. A site gets ranked by its total.

The **hard filter** is strict: to appear in the index at all, a site needs at least one of OpenAPI / ai-plugin / MCP / structured API. `llms.txt` alone doesn't count — it's passive content, not something an agent can *do* anything with.

## Verified, not just detected

The first version of the crawler had a problem: "detection" and "exists" are different things.

A site might return 200 for `/openapi.json`, but serve HTML with the word "openapi" in a blog post. A site might claim a structured API at `/api/v1/data`, but actually 301 you to a marketing page. A `/.well-known/ai-plugin.json` might be a catch-all 404.

So detection is:

- **OpenAPI**: parse as JSON or YAML, require `openapi: 3.x` or `swagger: 2.x`, require non-empty `paths`
- **Structured API**: require a real JSON response (valid parse, array or object), not HTML with JSON-in-Script tags
- **ai-plugin.json**: must be valid JSON with at least `schema_version`, `name_for_model`, `api`
- **MCP server**: must have valid `/.well-known/mcp.json` with `tools` array

Between the filter and the verification, ~70% of sites that *look* agent-first get rejected.

## Favicons, because UX matters

First-pass design used a third-party favicon service. Turns out those services return a placeholder globe with a 200 status when they can't find a real icon. `onerror` never fires. Every result card showed a blurry globe.

Second pass: detect the favicon during the crawl by checking `<link rel="icon">` and `/favicon.ico`, verify the response is an actual image via magic bytes (ICO, PNG, GIF, JPEG, SVG), and store the verified URL on the site row.

Third pass: fall back to a letter-avatar gradient (first letter of the domain) when no real favicon exists. Clean, no blur.

## Stack

- **Go** for the server and crawler
- **Postgres** on Fly.io for the index
- **Fly machines** for the HTTP layer
- **Resend** for ops email
- **MCP** for agent-direct access

The crawler runs daily via launchd, re-checks every site, and processes pending submissions.

## Using it from an agent

**Via HTTP:**
```
GET https://nothumansearch.ai/api/v1/search?q=weather&min_score=50
```

**Via MCP:**
Point any MCP-capable client at `https://nothumansearch.ai/.well-known/mcp.json`. You get four tools: `search`, `get_site`, `submit_site`, `stats`.

**Submitting a site:**
```
POST https://nothumansearch.ai/api/v1/submit
{"url": "https://example.com"}
```

The crawler picks it up on the next pass, verifies signals, and either adds it or flags it.

## What's next

- Semantic search over the index (embeddings for fuzzy agent queries)
- Trust signals (uptime, spec freshness, rate-limit behavior)
- Paid tier for deeper agentic intelligence (auth flows, pagination hints, schema diffs)

## Try it

https://nothumansearch.ai

Submit a site, poke the API, point an agent at the MCP endpoint. If you maintain an agent-first tool and it's not in the index, submit it — the crawler will verify and add it.

---

**When to post:**
- Tuesday morning — highest dev.to engagement
- Cross-post to Hashnode same day (different audience)

**Distribution after post:**
- Pin to NHS homepage via simple link
- Share in r/LocalLLaMA, HN (see hn-show-post.md)
- DM to 5 agent-framework maintainers
