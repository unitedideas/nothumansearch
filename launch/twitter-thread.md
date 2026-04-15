# Twitter/X Thread — NHS Launch

## Thread (7 tweets)

**1/7**
The web Google indexes is optimized for human eyeballs.

Your agent doesn't have eyeballs.

It has tool calls — and every one spent scraping a JS-heavy landing page is a tool call wasted.

I built Not Human Search to fix this. 🧵

**2/7**
Agents don't need blog posts *about* APIs.

They need:
→ OpenAPI specs that parse
→ Real JSON endpoints that return data
→ MCP servers they can connect to
→ llms.txt for navigation
→ Permission to crawl

These signals exist. No one had indexed them.

**3/7**
Not Human Search crawls for 7 agent-readiness signals and scores every site 0–100:

llms.txt (25) • ai-plugin.json (20) • OpenAPI (20) • Structured API (15) • MCP server (10) • robots.txt AI (5) • schema.org (5)

Only sites with a HARD signal (API/OpenAPI/MCP/plugin) make the index.

**4/7**
First version failed because "200 OK on /openapi.json" ≠ "valid OpenAPI spec."

Real verification:
→ OpenAPI: parse JSON/YAML, require openapi 3.x, require non-empty paths
→ Structured API: require real JSON response, not HTML
→ ai-plugin: valid manifest with schema_version + api block

~70% of "agent-first" sites fail verification.

**5/7**
NHS has an MCP server itself.

Wire it into Claude Code with one line:
claude mcp add --transport http nothumansearch https://nothumansearch.ai/mcp

Tools: search_agents, get_site_details, get_stats.

Your agent can search the agent-first web from inside a conversation.

**6/7**
What's in there:
870+ verified agent-first sites, 12 categories, daily recrawl. Top 3: developer (386), ai-tools (170), data (114).

Bonus: https://nothumansearch.ai/score scores any URL instantly and gives you a copy-paste embed badge (HTML / Markdown / JSX) that links back to the full report.

Submission form is public:
https://nothumansearch.ai

**7/7**
Stack: Go + Postgres + Fly.io.
Code: MIT, github.com/unitedideas/nothumansearch

If you're building agents and you're tired of Google's human-first index, try it:
https://nothumansearch.ai

Feedback welcome, especially on the scoring rubric.

---

## LinkedIn version (single post)

Most agent frameworks I've worked with in the last six months hit the same wall: the agent wants to find a tool, but the only search engine it has access to is optimized for humans.

Google indexes blog posts. Agents need OpenAPI specs.

So I built a search engine for agents.

Not Human Search crawls the web for seven agent-readiness signals — OpenAPI, ai-plugin.json, MCP servers, structured APIs, llms.txt, and more — and scores every site on an agentic readiness scale. Only sites with a verified hard signal make the index. Not "contains the word openapi" — actually parses, actually has endpoints.

It's free, MIT-licensed, and exposes a hosted MCP server at https://nothumansearch.ai/mcp — wire it into Claude Code with one line:

    claude mcp add --transport http nothumansearch https://nothumansearch.ai/mcp

870+ verified agent-first sites indexed today, across 12 categories. Free per-URL scoring at /score with a copy-paste embed badge.

If you're building agent tooling and want a discovery primitive that isn't "scrape Google," take a look:

https://nothumansearch.ai

---

## Distribution

- Post thread Tuesday 10am Pacific (peak dev Twitter)
- LinkedIn post same day, 9am Pacific
- Engage every reply within an hour
- Quote-tweet anyone who mentions NHS
