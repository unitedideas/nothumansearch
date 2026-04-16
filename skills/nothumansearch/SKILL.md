---
name: nothumansearch
description: Search and score AI-ready websites via MCP. Find MCP servers, check site AI-readiness scores, and verify MCP endpoint compliance using the Not Human Search API at nothumansearch.ai.
---

# Not Human Search MCP Skill

Search, score, and verify AI-ready websites via the Not Human Search MCP server. Use this when you need to find MCP servers, check if a website is optimized for AI agents, or verify MCP endpoint compliance.

## MCP Server Setup

Add to your MCP configuration (Claude Code, Cursor, etc.):

```json
{
  "mcpServers": {
    "nothumansearch": {
      "url": "https://nothumansearch.ai/mcp"
    }
  }
}
```

The server uses JSON-RPC over HTTP (streamable). No authentication required.

## Available Tools

### search

Search the index of 1,750+ AI-ready websites.

```
Use tool: search
Input: { "query": "code review MCP server" }
```

Returns ranked results with name, URL, description, category, tags, and agentic score.

**When to use:** Finding MCP servers, AI tools, or websites optimized for agent consumption.

### check

Check any URL's AI-readiness score (0-100).

```
Use tool: check
Input: { "url": "https://example.com" }
```

Returns a score based on: llms.txt presence, robots.txt AI bot allowance, structured data, API availability, MCP endpoint, OpenAPI spec, and more.

**When to use:** Evaluating whether a website is ready for AI agent interaction.

### verify_mcp

Live JSON-RPC probe of any MCP endpoint URL.

```
Use tool: verify_mcp
Input: { "url": "https://example.com/mcp" }
```

Sends a real `initialize` JSON-RPC request and reports whether the endpoint responds correctly, plus discovered tools and capabilities.

**When to use:** Verifying that an MCP server endpoint is live and compliant.

## REST API (Alternative)

If not using MCP, the same data is available via REST:

```bash
# Search
curl "https://nothumansearch.ai/api/v1/search?q=code+review"

# Check a URL
curl "https://nothumansearch.ai/api/v1/check?url=https://example.com"

# Stats
curl "https://nothumansearch.ai/api/v1/stats"
```

## Example Workflows

### Find MCP servers for a specific task

```
1. search({ query: "database management" })
2. Review results, pick candidates
3. verify_mcp({ url: candidate.mcp_url }) for each
4. Connect to verified servers
```

### Audit your own site's AI readiness

```
1. check({ url: "https://yoursite.com" })
2. Review score breakdown
3. Address missing items (llms.txt, robots.txt, structured data)
4. Re-check to confirm improvement
```

## References

- [Not Human Search](https://nothumansearch.ai) - Live service
- [llms.txt](https://nothumansearch.ai/llms.txt) - Machine-readable docs
- [OpenAPI Spec](https://nothumansearch.ai/openapi.yaml) - Full API specification
- [MCP Registry](https://github.com/modelcontextprotocol/servers) - Listed as ai.nothumansearch/search
