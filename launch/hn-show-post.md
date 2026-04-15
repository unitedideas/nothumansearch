# Show HN: Not Human Search – a search engine for AI agents

**Title:** Show HN: Not Human Search – a search engine for AI agents

**URL:** https://nothumansearch.ai

**Body:**

Hi HN. I built Not Human Search because every agent framework I touched in the last six months ran into the same wall: I can tell my agent to "go find me a weather API" and it spends its tool calls bouncing through Google SERPs, scraping landing pages, and giving up on JavaScript-heavy sites that were never built for it.

The web indexed by Google is optimized for human eyeballs. Agents need something different — sites with structured signals they can actually consume: an OpenAPI spec, an ai-plugin.json, an MCP server, a real JSON endpoint that returns data instead of a catch-all HTML shell.

So I crawled the web for those signals and built a search engine over the result.

**What it does:**
- Crawls public sites, checks for 7 agent-readiness signals: llms.txt, ai-plugin.json, OpenAPI, structured API, MCP server, AI-friendly robots.txt, schema.org
- Scores each site 0–100 on an "agentic readiness" scale
- Hard filter: a site must have at least one of (OpenAPI / ai-plugin / MCP / structured API) to be indexed. `llms.txt` alone doesn't count — it's just content.
- Verifies the signals are real, not just strings. OpenAPI detection parses the spec and checks for non-empty `paths`; API detection requires a real JSON response, not an HTML catch-all.
- Exposes the index via REST (`/api/v1/search`) and MCP (`/.well-known/mcp.json`), so agents can use it directly.

**Things I built so other agents could use this:**
- Full MCP server with 3 tools: `search_agents`, `get_site_details`, `get_stats`. Wire it into Claude Code with one line: `claude mcp add --transport http nothumansearch https://nothumansearch.ai/mcp`
- llms.txt, llms-full.txt, ai-plugin.json, openapi.yaml, sitemap.xml — NHS eats its own dog food
- Public submission form: anyone can add a site; the crawler re-verifies and either indexes it or rejects it
- Free on-demand scoring at https://nothumansearch.ai/score — paste any URL, get the 7-signal breakdown, copy an embeddable badge (HTML / Markdown / JSX) that links back to the full report
- RSS feeds: https://nothumansearch.ai/feed.xml (all new) plus per-category feeds at /feed/ai-tools.xml, /feed/developer.xml, etc.

**What's in the index right now:**
870+ verified agent-first sites across 12 categories. Top 3: developer (386), ai-tools (170), data (114). Every site has passed the agent-first filter — schema.org alone is not enough to get in.

**What I'm hoping to get from posting this:**
1. Sites I missed — if you maintain an agent-first API and you're not in there, submit it at the URL above or reply here
2. Feedback on scoring — the 7-signal rubric is a first pass; happy to iterate
3. Integrations — if you build agent tooling and want a discovery primitive that's not "scrape Google," this is the one

Stack is Go + Postgres on Fly.io. Code is MIT: github.com/unitedideas/nothumansearch

---

**When to post:**
- Tuesday or Wednesday, 8am Pacific (peaks HN traffic)
- NOT: Monday (weekend buildup), Friday (dead)

**After-post moves:**
1. Reply to every thread comment within 30 min
2. If it hits front page, cross-post to r/LocalLLaMA, r/programming
3. Ping a few agent-framework maintainers (LangChain, CrewAI, AutoGPT Discord) with "thought you might find this useful"
