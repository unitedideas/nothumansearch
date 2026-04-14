# DNS Setup Required (Shane)

Both domains registered on GoDaddy. Fly.io certs created and waiting.

## nothumansearch.com
| Type | Name | Value |
|------|------|-------|
| A | @ | 66.241.124.16 |
| AAAA | @ | 2a09:8280:1::101:4be:0 |
| A | www | 66.241.124.16 |
| AAAA | www | 2a09:8280:1::101:4be:0 |

## nothumansearch.ai
| Type | Name | Value |
|------|------|-------|
| A | @ | 66.241.124.16 |
| AAAA | @ | 2a09:8280:1::101:4be:0 |
| A | www | 66.241.124.16 |
| AAAA | www | 2a09:8280:1::101:4be:0 |

## Notes
- Fly.io handles TLS automatically once DNS propagates
- nothumansearch.com will be canonical; .ai redirects to .com
- Server handles www→apex redirect
