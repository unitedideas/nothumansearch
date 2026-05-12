# Category Discovery Surface Repair

Observed: 2026-05-12T10:00Z

Action: repaired public category vocabulary drift across agent-facing discovery
surfaces. Public categories now include `news`; audit-only buckets `other` and
`spam` are documented as transparent filter/audit buckets, not promoted
agent-ready inventory.

Changed surfaces:

- `/llms.txt`
- `/.well-known/mcp.json`
- `/openapi.yaml`
- MCP tool input descriptions

Proof:

- `go test ./internal/handlers ./internal/models`
- `curl -fsS https://nothumansearch.ai/llms.txt | grep -E 'Base URL: https://nothumansearch.ai/api/v1|Public categories:|Audit-only buckets|Live scorer: https://nothumansearch.ai/score'`
- `curl -fsS https://nothumansearch.ai/.well-known/mcp.json | python3 -m json.tool | grep -E 'Audit-only|news|spam|other'`
- `curl -fsS https://nothumansearch.ai/openapi.yaml | grep -E 'enum: \[ai-tools|audit-only|promoted discovery'`
- signed `POST https://nothumansearch.ai/webhook/stripe` smoke returned `200`

No public post, email, directory submission, customer-visible message, crawl, or
recategorize was run.
