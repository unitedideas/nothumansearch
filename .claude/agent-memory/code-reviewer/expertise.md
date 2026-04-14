# Code Reviewer Expertise — Not Human Search

## False Positives
- `http.ServeMux` in Go 1.22+ routes specific paths before catch-alls by longest prefix match; `/robots.txt` registered before `/` is safe and not a conflict.
- `xml.NewEncoder(w).Encode(sm)` is idiomatic and does not double-encode; the manual `w.Write([]byte(xml.Header))` is intentional to prepend the XML declaration.

## Anti-patterns Seen
- Swallowing both the DB query error and the `rows.Scan` error in sitemap/llms.txt handlers — silent failures produce empty or partial responses with HTTP 200.
- `json.NewEncoder(w).Encode(...)` called after `w.Header().Set(...)` but before `w.WriteHeader()` — encoder triggers implicit 200 before any chance to set a non-200 status; acceptable only when 200 is always correct (which it is for these static manifests).
- `h.DB.QueryRow(...).Scan(...)` with neither error checked — in LLMsTxt, a DB failure silently yields `totalSites = 0` and the response carries stale/wrong data with no indication of error.
- Domain values from DB injected directly into XML `<loc>` fields via string concatenation — requires XML-escaping if domains can contain `&`, `<`, or `>`.
- `fmt.Fprintf` to `http.ResponseWriter` — write errors silently dropped; acceptable for these small static responses but not for large streamed content.

## Project-Specific Patterns
- DB handle is `database.DB` (package-level var from `internal/database`), passed into handlers at construction — always check it's non-nil before use.
- `BASE_URL` env var with fallback to `https://nothumansearch.fly.dev` — this hardcoded fallback leaks the fly subdomain publicly in the OpenAPI spec if env var is unset in production.
- Route registration order: SEO routes registered before `/` catch-all — correct; Go mux picks longest match.
- All handlers use `database/sql` directly, no ORM. Parameterized queries with `$1` placeholders everywhere in API layer.
- Static files served with `immutable` cache header and `http.FileServer`.
- `WriteTimeout: 30s` on server — sitemap DB query must complete well within this.
- The sitemap handler silently ignores DB errors and emits a partial (static-pages-only) sitemap with HTTP 200, indistinguishable from a full sitemap to crawlers.

## Severity Calibration
- Missing `ctx` propagation in DB calls: Warning (not Critical) — no long-running queries observed yet.
- Swallowed scan errors in sitemap loop: Warning — crawler sees incomplete sitemap silently.
- Hardcoded fallback URL in binary: Warning — leaks internal infra detail.
- No `rows.Err()` check after iteration loop: Warning — missed driver-level errors. `SearchSites` still missing this after tags update.
- `json.NewEncoder` on `ai-plugin.json` produces trailing newline — acceptable; all major consumers handle this.
- `generateTags` keyword `" ai "` with space padding: will miss "ai" at end-of-string or start of `combined`. The combined string starts with description, so "ai" at the very start of a description (zero prior space) won't match. Low-impact false negative.
- `domainTags` loop has a `break` after first domain match — correct, intentional dedup.
- Count query in `SearchSites` (`db.QueryRow(...).Scan(...)`) silently swallows errors — yields `total=0` on DB error, search still returns rows. Count query now returns error properly (fixed); rows.Err() check also present.
- `fmt.Fprintf` arg count in `LLMsTxt`: 10 verbs, 10 args — verified correct after tags update.
- `categorize()` uses `strings.Contains(d, domainKey)` over a Go map — map iteration order is random; two overlapping domain keys (e.g., `"lemon.squeezy"` and `"lemonsqueezy"`) can match in arbitrary order. Not a correctness bug today (both map to same category) but fragile.
- `loggingMiddleware` skips logging for `GET /` where `Fly-Forwarded-Port` header is absent — this silently suppresses all organic homepage visits from browsers; only Fly health checks (which include the header) get skipped. Logic is inverted.
- `domainTags` `break` after first domain match stops at the first matching key; map iteration is random, so if two domain keys match the same domain (e.g., `"facebook.com"` and `"developer.x"` on a `developer.x.com` host), which wins is non-deterministic. Very low-impact given disjoint keys today.
- `SearchSites` multi-word relevance: `array_to_string(tags, ' ')` called once per `CASE WHEN` per word — Postgres may inline this but it's repeated in both the WHERE conditions and ORDER BY expressions. Not a bug, low impact at current scale.
