# DNS Setup — DONE (historical record)

**Status**: ✅ Live. Both domains have A/AAAA records pointing at Fly.io; TLS auto-issued; redirects working.

Kept here as a record of the IP values + which domain is canonical. Don't re-paste the old "DNS Setup Required" framing — it was completed before the site went live.

## Live state (confirmed 2026-04-18)

| Domain | Status |
|---|---|
| `nothumansearch.ai` | **canonical**, 200 OK |
| `nothumansearch.com` | 301 redirects to `nothumansearch.ai` |

Note: earlier drafts of this doc said `.com` would be canonical. **Reverse of reality** — we ended up making `.ai` the canonical host because it's the agent-facing brand. Server enforces this in Go middleware (domain-redirect in `cmd/server/main.go`).

## DNS records (both domains)

| Type | Name | Value |
|------|------|-------|
| A | @ | 66.241.124.16 |
| AAAA | @ | 2a09:8280:1::101:4be:0 |
| A | www | 66.241.124.16 |
| AAAA | www | 2a09:8280:1::101:4be:0 |

(Fly-assigned shared IPs. If Fly rotates them, check `fly ips list -a nothumansearch` for current values before updating DNS.)

## Ancillary records

- TLS: Fly.io auto-issues; no manual cert management.
- SPF TXT (for Resend-verified outbound email from `nothumansearch.ai`): **still missing on root domain** per main CLAUDE.md WIP. Adding `v=spf1 include:amazonses.com ~all` to the `.ai` root is Shane-gated (GoDaddy access).

## Verification

```bash
curl -sIL https://nothumansearch.com/ | grep -iE 'HTTP|location'   # expect 301 → .ai
curl -sIL https://nothumansearch.ai/  | head -1                    # expect 200
dig +short nothumansearch.ai @8.8.8.8                              # expect the A value above
```
