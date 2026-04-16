# Not Human Search — Reddit Drafts

Updated 2026-04-16. Post when Shane creates accounts.

---

## r/MachineLearning [P]

**Title:** `[P] Not Human Search — a search engine that indexes 8,600+ sites by how well they serve AI agents`

**Body:**
```
Built nothumansearch.ai — it crawls the web and scores sites 0-100 on how well they serve AI agents, not humans.

7 signals scored: llms.txt (25pts), ai-plugin.json (20), OpenAPI spec (20), structured API (15), live MCP server (10), robots.txt AI rules (5), Schema.org (5).

Some findings from the index:
- 51% of sites publish llms.txt, but only 4% expose an OpenAPI spec
- Only 0.7% score 70+ out of 8,632 indexed sites
- 498 sites serve a verified MCP endpoint (validated via JSON-RPC probe)
- Finance and security categories have the highest average agentic scores

NHS is itself an MCP server:
  claude mcp add --transport http nothumansearch https://nothumansearch.ai/mcp

REST API, OpenAPI spec, and embeddable score badges also available.

Source: https://nothumansearch.ai
Score any site: https://nothumansearch.ai/score
Guide: https://nothumansearch.ai/guide
```

---

## r/artificial

**Title:** `I indexed 8,600+ sites by how well they serve AI agents. Here's what I found.`

**Body:**
```
Built a crawler that scores websites on "agentic readiness" — how well they can be discovered and used by AI agents rather than humans.

Key findings:
- The average score is 17.6/100. The agentic web is extremely nascent.
- 51% of sites have an llms.txt file, but only 4% have an OpenAPI spec. Most sites are optimizing for LLM context, not programmatic access.
- Only 498 out of 8,632 sites serve a live MCP (Model Context Protocol) endpoint.
- Finance and security sites invest the most in agentic interfaces (~24/100 avg score).
- The llms.txt + MCP combo is the most common signal pairing (334 sites).

The index is at nothumansearch.ai. You can score any URL at /score and get an instant breakdown of what signals it exposes.

The whole thing is also available as an MCP server, so agents can search the index programmatically:
  claude mcp add --transport http nothumansearch https://nothumansearch.ai/mcp
```

---

## r/SideProject

**Title:** `I built a search engine for AI agents — 8,600+ sites indexed and scored on "agentic readiness"`

**Body:**
```
Side project: nothumansearch.ai — Google, but for AI agents.

The idea: as AI agents become more autonomous, they need to discover services programmatically. Google ranks for humans (page titles, backlinks). NHS ranks for agents (APIs, OpenAPI specs, MCP servers, llms.txt files).

Stack: Go + Postgres on Fly.io. Single binary — crawler + server. Auto-discovers new sites weekly from the official MCP registry, PulseMCP, awesome-mcp lists, and llms.txt directories.

Scoring: 0-100 based on 7 signals. Content-verified — the crawler actually probes MCP endpoints with JSON-RPC, parses OpenAPI specs, validates ai-plugin.json manifests. Not just checking if a URL returns 200.

Currently at 8,632 sites. Average score: 17.6/100. Only 59 sites score 70+.

Revenue model: free to search/submit. Planning premium API tier + promoted listings once there's organic demand.

https://nothumansearch.ai
```

---

## r/ChatGPT / r/ClaudeAI

**Title:** `I built a search engine that AI agents can use to discover tools — 8,600+ sites scored`

**Body:**
```
nothumansearch.ai — a search engine designed for AI agents, not humans.

Instead of ranking by page titles and backlinks, it scores sites on signals agents actually care about: does it have an API? OpenAPI spec? MCP server? llms.txt?

You can add it to Claude as an MCP server:
  claude mcp add --transport http nothumansearch https://nothumansearch.ai/mcp

Then Claude can search for agent-compatible tools directly. 6 tools available: search_agents, get_site_details, get_stats, submit_site, register_monitor, verify_mcp.

8,632 sites indexed. Score any URL at nothumansearch.ai/score.
```

---

## Posting guidance

- r/MachineLearning: [P] tag. Lead with data, not marketing. Show the findings.
- r/artificial: "here's what I found" angle. The data is the hook.
- r/SideProject: stack + revenue model. They want the indie story.
- r/ChatGPT/r/ClaudeAI: MCP install command is the hook. Make it actionable.
