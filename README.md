# Not Human Search

[![NHS Agentic Readiness](https://nothumansearch.ai/badge/nothumansearch.ai.svg)](https://nothumansearch.ai/site/nothumansearch.ai)
[![SafeSkill 91/100](https://img.shields.io/badge/SafeSkill-91%2F100_Verified%20Safe-brightgreen)](https://safeskill.dev/scan/unitedideas-nothumansearch)

**Search engine for AI agents.** Indexes sites ranked by *agentic readiness* — llms.txt, OpenAPI, ai-plugin, MCP servers, and structured APIs.

🌐 **Live site:** [nothumansearch.ai](https://nothumansearch.ai)
🔌 **MCP server:** `claude mcp add --transport http nothumansearch https://nothumansearch.ai/mcp`
📊 **Live score:** [nothumansearch.ai/score](https://nothumansearch.ai/score) — check any URL
🛠 **API:** [nothumansearch.ai/api/v1/search?q=](https://nothumansearch.ai/api/v1/search?q=)

## Why

LLM agents and autonomous systems can't "browse the web" efficiently the way humans do. They need machine-readable signals — llms.txt, OpenAPI schemas, MCP endpoints, ai-plugin manifests. NHS is the first search engine that explicitly ranks by *how agent-friendly a site actually is*, not by traditional SEO.

## What's indexed

- **8,000+ sites crawled** and scored across 7 agentic signals
- **500+ MCP-verified servers** with live JSON-RPC probe
- **Agent-first filter** — every indexed site has at least one strong agent signal
- Continuously refreshed via daily recrawl + weekly auto-discovery

## Scoring (max 100)

| Signal         | Points |
|----------------|--------|
| llms.txt       | 25 |
| ai-plugin.json | 20 |
| OpenAPI spec   | 20 |
| Structured API | 15 |
| MCP server     | 10 |
| robots.txt     | 5  |
| Schema.org     | 5  |

## How to use it

**As an agent (MCP):**
```
claude mcp add --transport http nothumansearch https://nothumansearch.ai/mcp
```
8 tools: `search_agents`, `get_site_details`, `get_stats`, `list_categories`, `get_top_sites`, `verify_mcp`, and more.

**As a developer (HTTP API):**
```bash
curl "https://nothumansearch.ai/api/v1/search?q=payments&limit=10"
```

**As a human:** just visit [nothumansearch.ai](https://nothumansearch.ai).

**Scoring your own site:** [nothumansearch.ai/score](https://nothumansearch.ai/score) — paste a URL, get a breakdown + embed badge.

## Stack

- Go 1.25 + Postgres on Fly.io (sjc)
- Single binary server + crawler
- MCP over streamable-http at `/mcp`
- IndexNow pings Bing/Yandex on every crawl cycle

## Related

- **[aidevboard.com](https://aidevboard.com)** — AI Dev Jobs board (REST API + MCP, 8,400+ listings)
- **[8bitconcepts.com](https://8bitconcepts.com)** — Research papers on enterprise AI adoption

## License

MIT
