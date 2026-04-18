# MCP Registry manifest

Tracked source-of-truth for the entry published as `ai.nothumansearch/search`
at https://registry.modelcontextprotocol.io.

Past sessions published this from ad-hoc `/tmp/*.json` which meant the
canonical version history lived only in the registry API. This file
fixes that so every future version bump is a commit.

## Publish flow

1. Edit `server.json` — bump `version`, update `description` (100-char max).
2. Login as the registered domain:

   ```bash
   mcp-publisher login http \
     --domain nothumansearch.ai \
     --private-key "$(/usr/bin/security find-generic-password -a foundry -s nhs-mcp-registry-privkey -w)"
   ```

3. Publish:

   ```bash
   cp tools/mcp-registry/server.json /tmp/server.json
   cd /tmp && mcp-publisher publish
   ```

4. Verify:

   ```bash
   curl -s "https://registry.modelcontextprotocol.io/v0/servers?search=nothumansearch&version=latest" \
     | python3 -m json.tool | grep -E 'version|description' | head
   ```

5. Commit the updated `server.json` with a message like:
   `registry: bump NHS manifest to vX.Y.Z (published to registry)`

## Version history

See `registry.modelcontextprotocol.io/v0/servers?search=nothumansearch`
for the authoritative timeline. Recent bumps:

- `v1.7.0` — check_url tool (11 total).
- `v1.6.0` — find_mcp_servers + recent_additions tools (10 total).
- `v1.5.0` — list_categories + get_top_sites.
- `v1.4.0` — verify_mcp.
