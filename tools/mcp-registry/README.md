# Official MCP Registry entry

`server.json` is the next prepared manifest for `ai.nothumansearch/search`.
The registry API is authoritative for publication state. Avoid fixed catalog or tool counts, which drift independently.

Validate without authentication or publication:

```sh
go run ./tools/mcp-registry
```

Before publishing, reserve the version through the existing shared-operations workflow. The remote lock must be
`public-action-locks/agentic-discovery/nhs-official-registry-VERSION.json`, with `status=claimed`, the current
`claimed_by`, and `dedupe_key=nothumansearch:official-registry:VERSION`.

Publish through the canonical named reference:

```sh
/Users/shane/.local/bin/codex-secret check NHS_MCP_REGISTRY_PRIVATE_KEY
/Users/shane/.local/bin/codex-secret run \
  --env MCP_PRIVATE_KEY=NHS_MCP_REGISTRY_PRIVATE_KEY \
  -- go run ./tools/mcp-registry --publish \
     --expected-current-version 1.7.1 \
     --state-repo /path/to/existing/shared-state-worktree \
     --claim-owner ACTUAL_RESERVATION_OWNER
```

The publisher verifies the live baseline, shared reservation, domain proof, and active latest-version readback.
Authentication stays in memory. No key-valued arguments or shared login cache are used.
Missing access stops publication; restore the canonical reference without replacing the signing key.

After an uncertain write, rerun only after checking the exact version. Matching published metadata returns
`already-published`; conflicting metadata or a changed baseline stops the operation.
