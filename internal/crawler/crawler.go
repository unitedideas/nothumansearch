package crawler

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/unitedideas/nothumansearch/internal/models"
)

var client = &http.Client{
	Timeout: 15 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 3 {
			return fmt.Errorf("too many redirects")
		}
		return nil
	},
}

// Categorize returns the category for an already-populated Site (no HTTP).
// Exposed so the crawler CLI can re-apply categorize() rules over existing
// DB rows without re-hitting the network — useful when new domainRules are
// added to categorize() and recrawl is slow or stuck on HTTP.
func Categorize(site *models.Site) string { return categorize(site) }

// GenerateTags is the exported wrapper around generateTags for recategorize mode.
func GenerateTags(site *models.Site) pq.StringArray { return generateTags(site) }

const userAgent = "NotHumanSearch/1.0 (+https://nothumansearch.ai/about)"

// ProbeMCPJSONRPC POSTs a tools/list request to an MCP endpoint and verifies
// the response is valid JSON-RPC 2.0 with a result.tools array. This is the
// only way to confirm a claimed MCP server actually exists and responds —
// manifest files can lie and text mentions can't be trusted. Short timeout
// (6s) so this doesn't slow the crawler. Exported so the MCP server can
// expose it as a verify_mcp tool for agents.
func ProbeMCPJSONRPC(endpoint string) bool {
	payload := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	req, err := http.NewRequest("POST", endpoint, payload)
	if err != nil {
		return false
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("User-Agent", userAgent)

	probeClient := &http.Client{Timeout: 6 * time.Second}
	resp, err := probeClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return false
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return false
	}
	// Streamable-http transport may wrap JSON-RPC in SSE "data: " prefixes.
	raw := strings.TrimSpace(string(body))
	if strings.HasPrefix(raw, "data:") {
		for _, line := range strings.Split(raw, "\n") {
			if strings.HasPrefix(line, "data:") {
				raw = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
				break
			}
		}
	}
	var rpc struct {
		JSONRPC string `json:"jsonrpc"`
		Result  struct {
			Tools []any `json:"tools"`
		} `json:"result"`
		Error *struct {
			Code int `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(raw), &rpc); err != nil {
		return false
	}
	if rpc.JSONRPC != "2.0" {
		return false
	}
	// Accept either a successful tools array or a method-specific error
	// (some servers require initialize() first — still proves MCP-compliance).
	if rpc.Error != nil {
		return rpc.Error.Code == -32601 || rpc.Error.Code == -32600
	}
	return rpc.Result.Tools != nil
}

func fetch(rawURL string) (string, int, error) {
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/plain, application/json, text/html, */*")

	resp, err := client.Do(req)
	if err != nil {
		// One retry on timeout
		resp, err = client.Do(req)
		if err != nil {
			return "", 0, err
		}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024)) // 512KB max
	if err != nil {
		return "", resp.StatusCode, err
	}
	return string(body), resp.StatusCode, nil
}

// CrawlSite checks a domain for all agentic readiness signals.
func CrawlSite(siteURL string) (*models.Site, error) {
	u, err := url.Parse(siteURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme == "" {
		u.Scheme = "https"
	}
	// Normalize: strip "www." and lowercase so "www.prisma.io" and "prisma.io"
	// collapse into a single canonical record.
	canonicalHost := strings.ToLower(strings.TrimPrefix(u.Host, "www."))
	base := fmt.Sprintf("%s://%s", u.Scheme, canonicalHost)

	site := &models.Site{
		Domain:      canonicalHost,
		URL:         base,
		CrawlStatus: "success",
	}

	now := time.Now()
	site.LastCrawledAt = &now

	// Fetch homepage for title/description
	var homepageBody string
	if body, status, err := fetch(base); err == nil && status == 200 {
		site.Name = extractTitle(body)
		site.Description = extractMetaDescription(body)
		site.HasSchemaOrg = strings.Contains(body, "schema.org")
		homepageBody = body
	}

	// Detect favicon so the frontend can render a clean letter-avatar instead of
	// third-party placeholder globes when a site has no real favicon.
	if fu, ok := detectFavicon(base, homepageBody); ok {
		site.HasFavicon = true
		site.FaviconURL = fu
	}

	// Check llms.txt
	for _, path := range []string{"/llms.txt", "/.well-known/llms.txt"} {
		if body, status, err := fetch(base + path); err == nil && status == 200 && len(body) > 10 {
			site.HasLLMsTxt = true
			if len(body) > 2000 {
				site.LLMsTxtContent = body[:2000]
			} else {
				site.LLMsTxtContent = body
			}
			break
		}
	}

	// Check ai-plugin.json
	for _, path := range []string{"/.well-known/ai-plugin.json", "/ai-plugin.json"} {
		if body, status, err := fetch(base + path); err == nil && status == 200 {
			var plugin map[string]interface{}
			if json.Unmarshal([]byte(body), &plugin) == nil {
				if _, ok := plugin["name_for_human"]; ok {
					site.HasAIPlugin = true
					if name, ok := plugin["name_for_human"].(string); ok && site.Name == "" {
						site.Name = name
					}
					if desc, ok := plugin["description_for_human"].(string); ok && site.Description == "" {
						site.Description = desc
					}
					break
				}
			}
		}
	}

	// Check OpenAPI spec — must be a real OpenAPI 3.x or Swagger 2.x document,
	// not just any HTML page containing the word "openapi".
	for _, path := range []string{"/openapi.yaml", "/openapi.json", "/api/openapi.yaml", "/api/openapi.json", "/swagger.json", "/api-docs.json", "/api/v1/openapi.json", "/api/v2/openapi.json", "/spec.json"} {
		if body, status, err := fetch(base + path); err == nil && status == 200 && len(body) > 100 {
			if isValidOpenAPI(body) {
				site.HasOpenAPI = true
				summary := body
				if len(summary) > 500 {
					summary = summary[:500]
				}
				site.OpenAPISummary = summary
				break
			}
		}
	}

	// Check robots.txt for AI bot permissions
	if body, status, err := fetch(base + "/robots.txt"); err == nil && status == 200 {
		bodyLower := strings.ToLower(body)
		aiSignals := []string{"gptbot", "chatgpt", "claudebot", "anthropic", "perplexity", "cohere", "applebot"}
		for _, signal := range aiSignals {
			if strings.Contains(bodyLower, signal) {
				site.HasRobotsAI = true
				break
			}
		}
	}

	// Check for MCP server support
	for _, path := range []string{"/.well-known/mcp.json", "/mcp.json", "/.well-known/mcp-server.json"} {
		if body, status, err := fetch(base + path); err == nil && status == 200 {
			var mcpManifest map[string]interface{}
			if json.Unmarshal([]byte(body), &mcpManifest) == nil {
				// Valid JSON with MCP-like structure
				if _, hasName := mcpManifest["name"]; hasName {
					site.HasMCPServer = true
					if endpoint, ok := mcpManifest["endpoint"].(string); ok {
						site.MCPEndpoint = endpoint
					} else if url, ok := mcpManifest["url"].(string); ok {
						site.MCPEndpoint = url
					}
					break
				}
				if _, hasTools := mcpManifest["tools"]; hasTools {
					site.HasMCPServer = true
					break
				}
			}
		}
	}
	// If no manifest was found, try a live JSON-RPC probe against common endpoints.
	// Only a valid tools/list response counts — text mentions alone proved too noisy
	// (481/951 = 51% of the index claimed has_mcp; most were unverifiable).
	if !site.HasMCPServer {
		probeTargets := []string{base + "/mcp", base + "/api/mcp"}
		for _, target := range probeTargets {
			if ProbeMCPJSONRPC(target) {
				site.HasMCPServer = true
				site.MCPEndpoint = target
				break
			}
		}
	} else if site.MCPEndpoint != "" {
		// Manifest declared an endpoint — verify it actually responds. Leaves
		// manifest-only claims in place (some hosts declare MCP preview/planned
		// without live endpoint), but lets us log divergence for quality work.
		if !ProbeMCPJSONRPC(site.MCPEndpoint) {
			log.Printf("mcp manifest-only (probe failed): %s endpoint=%s", site.Domain, site.MCPEndpoint)
		}
	}

	// Check for structured API — require an actual JSON-ish / API-typical response.
	// Redirects DON'T count: many sites 301 unknown paths to homepage.
	// Status 200 alone is insufficient: sites with catch-all HTML routes hit everything.
	apiPaths := []string{"/api/v1", "/api/v2", "/api/v3", "/api", "/v1", "/v2", "/v3", "/graphql", "/rest/v1", "/openai/v1"}
	if strings.HasPrefix(strings.ToLower(site.Domain), "api.") {
		apiPaths = append([]string{"/"}, apiPaths...)
	}
	for _, path := range apiPaths {
		if body, status, err := fetch(base + path); err == nil && status < 500 && isAPIResponse(body) {
			site.HasStructuredAPI = true
			break
		}
	}
	// Probe api.{domain} and developer.{domain} for JSON/auth responses.
	if !site.HasStructuredAPI && !strings.HasPrefix(site.Domain, "api.") {
		for _, sub := range []string{"api", "developer", "developers"} {
			apiBase := fmt.Sprintf("https://%s.%s", sub, site.Domain)
			for _, path := range []string{"/", "/api", "/api/v1", "/v1", "/v2"} {
				if body, status, err := fetch(apiBase + path); err == nil && status < 500 && isAPIResponse(body) {
					site.HasStructuredAPI = true
					break
				}
			}
			if site.HasStructuredAPI {
				break
			}
		}
	}
	// Doc-page fallback: check main domain + docs.{domain} subdomain for API documentation.
	if !site.HasStructuredAPI {
		apiIndicators := []string{"endpoint", "rest api", "graphql", "bearer token", "api key",
			"rate limit", "access token", "curl -x", "curl --", "webhook", "oauth 2", "api reference",
			"openapi", "swagger", "application/json", "sdk", "client library", "pip install",
			"npm install", "x-api-key", "base url", "http method", "get /", "post /", "put /", "delete /"}
		docPaths := []string{"/docs", "/docs/api", "/documentation", "/developer", "/developers",
			"/api-docs", "/api", "/reference", "/api-reference", "/docs/reference",
			"/docs/guides", "/docs/quickstart", "/docs/rest-api"}
		docBases := []string{base}
		if !strings.HasPrefix(site.Domain, "docs.") {
			docBases = append(docBases, fmt.Sprintf("https://docs.%s", site.Domain))
		}
		for _, docBase := range docBases {
			for _, path := range docPaths {
				if body, status, err := fetch(docBase + path); err == nil && status == 200 {
					bodyLower := strings.ToLower(body)
					matches := 0
					for _, indicator := range apiIndicators {
						if strings.Contains(bodyLower, indicator) {
							matches++
						}
					}
					if matches >= 3 {
						site.HasStructuredAPI = true
						break
					}
				}
			}
			if site.HasStructuredAPI {
				break
			}
		}
	}

	// Calculate score
	site.AgenticScore = models.CalculateScore(site)

	// Auto-categorize
	site.Category = categorize(site)

	// Generate tags for search discoverability
	site.Tags = generateTags(site)

	log.Printf("Crawled %s: score=%d llms=%v plugin=%v openapi=%v robots=%v api=%v schema=%v",
		site.Domain, site.AgenticScore,
		site.HasLLMsTxt, site.HasAIPlugin, site.HasOpenAPI,
		site.HasRobotsAI, site.HasStructuredAPI, site.HasSchemaOrg)

	return site, nil
}

// detectFavicon resolves a site's favicon if it exists. Returns absolute URL + ok.
// Looks for <link rel="icon"> in homepage HTML first (most accurate), then falls back
// to /favicon.ico. Does NOT accept HTML error pages as favicons.
func detectFavicon(base, html string) (string, bool) {
	// Try <link rel="icon"> / rel="shortcut icon" in homepage HTML
	lower := strings.ToLower(html)
	for _, relToken := range []string{`rel="icon"`, `rel='icon'`, `rel="shortcut icon"`, `rel='shortcut icon'`, `rel="apple-touch-icon"`, `rel='apple-touch-icon'`} {
		idx := strings.Index(lower, relToken)
		if idx == -1 {
			continue
		}
		tagStart := strings.LastIndex(lower[:idx], "<link")
		if tagStart == -1 {
			continue
		}
		tagEnd := strings.Index(lower[tagStart:], ">")
		if tagEnd == -1 {
			continue
		}
		tag := html[tagStart : tagStart+tagEnd+1]
		hrefIdx := strings.Index(strings.ToLower(tag), `href="`)
		var quote byte = '"'
		if hrefIdx == -1 {
			hrefIdx = strings.Index(strings.ToLower(tag), `href='`)
			quote = '\''
		}
		if hrefIdx == -1 {
			continue
		}
		start := hrefIdx + 6
		end := strings.Index(tag[start:], string(quote))
		if end == -1 {
			continue
		}
		href := strings.TrimSpace(tag[start : start+end])
		if href == "" {
			continue
		}
		// Resolve relative URLs
		resolved := resolveURL(base, href)
		if resolved == "" {
			continue
		}
		if verifyFavicon(resolved) {
			return resolved, true
		}
	}
	// Fallback: /favicon.ico
	candidate := strings.TrimSuffix(base, "/") + "/favicon.ico"
	if verifyFavicon(candidate) {
		return candidate, true
	}
	return "", false
}

func resolveURL(base, href string) string {
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href
	}
	if strings.HasPrefix(href, "//") {
		return "https:" + href
	}
	if strings.HasPrefix(href, "/") {
		return strings.TrimSuffix(base, "/") + href
	}
	return strings.TrimSuffix(base, "/") + "/" + href
}

func verifyFavicon(u string) bool {
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return false
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return false
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	if err != nil || len(body) < 16 {
		return false
	}
	// Reject HTML (site returning 200 HTML for unknown paths)
	ct := strings.ToLower(resp.Header.Get("Content-Type"))
	if strings.Contains(ct, "text/html") {
		return false
	}
	head := strings.ToLower(strings.TrimSpace(string(body)))
	if strings.HasPrefix(head, "<!doctype") || strings.HasPrefix(head, "<html") {
		return false
	}
	// Accept if content-type is image/* or if magic bytes look like ICO/PNG/SVG/GIF
	if strings.HasPrefix(ct, "image/") {
		return true
	}
	b := body
	// ICO: 00 00 01 00; PNG: 89 50 4E 47; GIF: 47 49 46 38; JPEG: FF D8 FF; SVG: <svg or <?xml
	if len(b) >= 4 {
		if b[0] == 0x00 && b[1] == 0x00 && (b[2] == 0x01 || b[2] == 0x02) && b[3] == 0x00 {
			return true
		}
		if b[0] == 0x89 && b[1] == 0x50 && b[2] == 0x4E && b[3] == 0x47 {
			return true
		}
		if b[0] == 0x47 && b[1] == 0x49 && b[2] == 0x46 && b[3] == 0x38 {
			return true
		}
		if b[0] == 0xFF && b[1] == 0xD8 && b[2] == 0xFF {
			return true
		}
	}
	if strings.Contains(head[:min(200, len(head))], "<svg") {
		return true
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// isValidOpenAPI returns true only if the body is a real OpenAPI 3.x or Swagger 2.x
// spec. Rejects HTML landing pages that happen to contain the word "openapi".
func isValidOpenAPI(body string) bool {
	trimmed := strings.TrimSpace(body)
	if len(trimmed) == 0 {
		return false
	}
	// Reject HTML
	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(lower, "<!doctype") || strings.HasPrefix(lower, "<html") || strings.HasPrefix(lower, "<?xml") {
		return false
	}
	// Try JSON: must have openapi/swagger field AND a paths object
	if strings.HasPrefix(trimmed, "{") {
		var doc map[string]interface{}
		if json.Unmarshal([]byte(trimmed), &doc) == nil {
			_, hasOpenAPI := doc["openapi"]
			_, hasSwagger := doc["swagger"]
			paths, hasPaths := doc["paths"].(map[string]interface{})
			if (hasOpenAPI || hasSwagger) && hasPaths && len(paths) > 0 {
				return true
			}
		}
		return false
	}
	// YAML: heuristic — must have both a version declaration at top level AND a paths: block
	// with at least one endpoint beneath it.
	hasVersion := false
	for _, line := range strings.Split(trimmed, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "openapi:") || strings.HasPrefix(t, "swagger:") {
			hasVersion = true
			break
		}
	}
	if !hasVersion {
		return false
	}
	// Must have paths: block followed by at least one indented endpoint (starts with / or has :)
	pathsIdx := strings.Index(trimmed, "\npaths:")
	if pathsIdx == -1 && !strings.HasPrefix(trimmed, "paths:") {
		return false
	}
	// Require at least one route under paths
	afterPaths := trimmed
	if pathsIdx >= 0 {
		afterPaths = trimmed[pathsIdx:]
	}
	return strings.Contains(afterPaths, "\n  /") || strings.Contains(afterPaths, "\n '/") || strings.Contains(afterPaths, "\n \"/")
}

// isAPIResponse returns true if the response body looks like an API (JSON/GraphQL),
// not an HTML page. Prevents catch-all routes on HTML sites from falsely matching.
func isAPIResponse(body string) bool {
	trimmed := strings.TrimSpace(body)
	if len(trimmed) == 0 {
		return false
	}
	// HTML disqualifies
	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(lower, "<!doctype") || strings.HasPrefix(lower, "<html") || strings.HasPrefix(lower, "<head") {
		return false
	}
	// JSON object or array
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		var v interface{}
		if json.Unmarshal([]byte(trimmed), &v) == nil {
			return true
		}
	}
	// GraphQL usually returns JSON too, but an unauthenticated GET often returns
	// {"errors":[...]} or a "must POST" message — still JSON, caught above.
	// Some APIs return plain text error like "Unauthorized" — accept if it mentions
	// auth/token/api and is short.
	if len(trimmed) < 200 {
		for _, hint := range []string{"\"message\"", "\"error\"", "\"data\"", "unauthorized", "authentication required", "api key", "api-key", "bearer", "forbidden", "not found", "invalid token"} {
			if strings.Contains(strings.ToLower(trimmed), hint) {
				return true
			}
		}
	}
	return false
}

func extractTitle(html string) string {
	start := strings.Index(html, "<title>")
	if start == -1 {
		start = strings.Index(html, "<title ")
		if start == -1 {
			return ""
		}
		end := strings.Index(html[start:], ">")
		if end == -1 {
			return ""
		}
		// Advance past the ">" of the opening <title ...> tag. Previously
		// this was `start + end` which points AT the ">", causing extracted
		// titles like "<title dir='ltr'>Foo</title>" to start with ">Foo"
		// instead of "Foo". Caught by TestExtractTitle.
		start = start + end + 1
	} else {
		start += 7
	}
	end := strings.Index(html[start:], "</title>")
	if end == -1 {
		return ""
	}
	title := strings.TrimSpace(html[start : start+end])
	if len(title) > 200 {
		title = title[:200]
	}
	return title
}

func extractMetaDescription(html string) string {
	lower := strings.ToLower(html)
	idx := strings.Index(lower, `name="description"`)
	if idx == -1 {
		idx = strings.Index(lower, `name='description'`)
	}
	if idx == -1 {
		return ""
	}

	// Find the enclosing <meta tag by scanning backwards for '<'
	tagStart := strings.LastIndex(lower[:idx], "<")
	if tagStart == -1 {
		return ""
	}
	tagEnd := strings.Index(lower[tagStart:], ">")
	if tagEnd == -1 {
		return ""
	}
	tag := html[tagStart : tagStart+tagEnd+1]
	tagLower := strings.ToLower(tag)

	// Extract content="..." from within this specific meta tag
	contentIdx := strings.Index(tagLower, `content="`)
	if contentIdx == -1 {
		contentIdx = strings.Index(tagLower, `content='`)
	}
	if contentIdx == -1 {
		return ""
	}
	quote := tag[contentIdx+8]
	start := contentIdx + 9
	end := strings.Index(tag[start:], string(quote))
	if end == -1 {
		return ""
	}
	desc := tag[start : start+end]
	if len(desc) > 500 {
		desc = desc[:500]
	}
	return desc
}

func categorize(site *models.Site) string {
	d := strings.ToLower(site.Domain)
	desc := strings.ToLower(site.Description)
	name := strings.ToLower(site.Name)

	// Pass 1: exact domain matches (highest confidence, avoids false positives
	// from generic keywords like "learn" or "security" in descriptions)
	domainRules := map[string]string{
		"aidevboard":      "jobs",
		"greenhouse.io":   "jobs",
		"lever.co":        "jobs",
		"ashbyhq.com":     "jobs",
		"workable.com":    "jobs",
		"stripe.com":      "finance",
		"plaid.com":       "finance",
		"mercury.com":     "finance",
		"brex.com":        "finance",
		"alpaca.markets":  "finance",
		"polygon.io":      "finance",
		"coinbase.com":    "finance",
		"wise.com":        "finance",
		"shopify":         "ecommerce",
		"bigcommerce":     "ecommerce",
		"woocommerce":     "ecommerce",
		"snipcart":        "ecommerce",
		"square.com":      "ecommerce",
		"openai":          "ai-tools",
		"anthropic":       "ai-tools",
		"cohere":          "ai-tools",
		"mistral":         "ai-tools",
		"groq.com":        "ai-tools",
		"together.ai":     "ai-tools",
		"fireworks.ai":    "ai-tools",
		"replicate.com":   "ai-tools",
		"huggingface":     "ai-tools",
		"deepgram":        "ai-tools",
		"elevenlabs":      "ai-tools",
		"stability.ai":    "ai-tools",
		"perplexity.ai":   "ai-tools",
		"assemblyai":      "ai-tools",
		"pinecone":        "ai-tools",
		"qdrant":          "ai-tools",
		"supabase":        "data",
		"neon.tech":       "data",
		"planetscale":     "data",
		"turso":           "data",
		"upstash":         "data",
		"weaviate":        "data",
		"chroma":          "data",
		"snowflake":       "data",
		"databricks":      "data",
		"fivetran":        "data",
		"segment.com":     "data",
		"mixpanel":        "data",
		"amplitude":       "data",
		"posthog":         "data",
		"fly.io":          "developer",
		"vercel.com":      "developer",
		"render.com":      "developer",
		"railway.app":     "developer",
		"deno.com":        "developer",
		"bun.sh":          "developer",
		"modal.com":       "developer",
		"cloudflare":      "developer",
		"github.com":      "developer",
		"sentry.io":       "developer",
		"grafana":         "developer",
		"datadog":         "developer",
		"langchain":       "developer",
		"llamaindex":      "developer",
		"crewai":          "developer",
		"autogen":         "developer",
		"composio":        "developer",
		"browserbase":     "developer",
		"e2b.dev":         "developer",
		"auth0.com":       "security",
		"clerk.com":       "security",
		"1password":       "security",
		"snyk.io":         "security",
		"workos.com":      "security",
		"health.gov":      "health",
		"pubmed":          "health",
		"clinicaltrials":  "health",

		// Communication
		"slack.com":        "communication",
		"discord.com":      "communication",
		"twilio.com":       "communication",
		"sendgrid.com":     "communication",
		"postmark.com":     "communication",
		"resend.com":       "communication",
		"pusher.com":       "communication",
		"onesignal":        "communication",
		"pushover.net":     "communication",
		"ntfy.sh":          "communication",

		// Productivity / Collaboration
		"zapier.com":       "productivity",
		"make.com":         "productivity",
		"notion.so":        "productivity",
		"airtable.com":     "productivity",
		"linear.app":       "productivity",
		"asana.com":        "productivity",
		"trello.com":       "productivity",
		"monday.com":       "productivity",
		"calendly":         "productivity",
		"cal.com":          "productivity",
		"cronofy":          "productivity",
		"typeform":         "productivity",
		"tally.so":         "productivity",
		"buffer.com":       "productivity",
		"hootsuite":        "productivity",

		// AI-tools (additional)
		"exa.ai":           "ai-tools",
		"tavily.com":       "ai-tools",
		"serper.dev":       "ai-tools",
		"unstructured.io":  "ai-tools",
		"leonardo.ai":      "ai-tools",
		"ideogram.ai":      "ai-tools",
		"runway.com":       "ai-tools",
		"luma.ai":          "ai-tools",
		"suno.com":         "ai-tools",
		"cursor.com":       "ai-tools",

		// Developer (additional)
		"betterstack":      "developer",
		"convex.dev":       "developer",
		"dnsimple":         "developer",
		"name.com":         "developer",
		"browserstack":     "developer",
		"lambdatest":       "developer",
		"mux.com":          "developer",
		"uploadthing":      "developer",
		"cloudinary":       "developer",

		// Data (additional)
		"api.census.gov":   "data",
		"data.gov":         "data",
		"developer.nrel":   "data",
		"contentful":       "data",
		"sanity.io":        "data",
		"storyblok":        "data",
		"dbt.com":          "data",

		// Ecommerce (additional)
		"aftership":        "ecommerce",
		"easypost":         "ecommerce",
		"goshippo":         "ecommerce",
		"shipstation":      "ecommerce",

		// Social media
		"developer.x":      "communication",
		"developers.facebook": "communication",
		"developers.reddit":   "communication",

		// Search / Knowledge
		"wikipedia.org":    "data",

		// Agent infrastructure
		"modelcontextprotocol": "developer",
		"smithery.ai":     "developer",
		"glama.ai":        "developer",
		"phidata":         "developer",

		// Finance (additional)
		"close.com":       "finance",
		"hubspot":         "productivity",
		"pipedrive":       "productivity",

		// Translation
		"deepl.com":       "ai-tools",
		"libretranslate":  "ai-tools",

		// Document / PDF
		"docusign.com":    "productivity",
		"docparser.com":   "data",
		"pdf.co":          "data",
		"smallpdf.com":    "productivity",

		// Foundry businesses
		"agentcanary":     "security",
		"8bitconcepts":    "ai-tools",
		"nothumansearch":  "ai-tools",

		// Cloud providers
		"cloud.google":    "developer",
		"travelport":      "data",
		"amadeus":         "data",

		// Music / Audio
		"developer.spotify": "data",
		"soundcloud.com":    "data",

		// Maps
		"mapbox.com":      "developer",
		"here.com":        "developer",

		// Remaining catch-alls
		"newsapi":          "data",
		"ai.google":        "ai-tools",
		"openweathermap":   "data",
		"exchangeratesapi": "finance",

		// Education
		"coursera":         "education",
		"udemy":            "education",
		"edx.org":          "education",
		"khanacademy":      "education",
		"duolingo":         "education",

		// Healthcare (additional)
		"healthkit":        "health",
		"developer.apple":  "developer",
		"fhir.org":         "health",
		"openfda":          "health",
		"medlineplus":      "health",
		"rxnav":            "health",

		// Security (additional)
		"letsencrypt":      "security",
		"vault.hashicorp":  "security",
		"virustotal":       "security",
		"haveibeenpwned":   "security",
		"cve.mitre":        "security",

		// Jobs (additional)
		"smartrecruiters":  "jobs",
		"breezy.hr":        "jobs",
		"recruitee":        "jobs",
		"bamboohr":         "jobs",
		"gusto.com":        "jobs",

		// Ecommerce (additional)
		"paddle.com":       "ecommerce",
		"lemonsqueezy":     "ecommerce",
		"lemon.squeezy":    "ecommerce",
		"gumroad":          "ecommerce",
		"printful":         "ecommerce",

		// Design / Creative
		"figma.com":        "developer",
		"canva.com":        "productivity",
		"dribbble":         "developer",

		// Legal
		"docuseal":         "productivity",
		"termly":           "security",

		// Real Estate
		"zillow":           "data",
		"realtor.com":      "data",

		// Food
		"doordash":         "ecommerce",
		"yelp.com":         "data",

		// IoT
		"particle.io":      "developer",
		"arduino":          "developer",

		// Crypto / Web3
		"etherscan":        "finance",
		"alchemy.com":      "developer",
		"moralis":          "developer",

		// Productivity (additional)
		"clickup":          "productivity",
		"todoist":          "productivity",
		"jira.atlassian":   "productivity",

		// MCP Infrastructure
		"mcpservers":       "developer",
		"mcpmarket":        "developer",
		"stagehand.dev":    "developer",

		// AI Agent Observability / LLMOps
		"agentops":         "ai-tools",
		"langfuse":         "ai-tools",
		"helicone":         "ai-tools",
		"braintrust":       "ai-tools",
		"arize.com":        "ai-tools",
		"openpipe":         "ai-tools",
		"galileo.ai":       "ai-tools",

		// AI Agent Orchestration
		"langflow":         "ai-tools",
		"flowiseai":        "ai-tools",
		"botpress":         "ai-tools",
		"voiceflow":        "ai-tools",
		"activepieces":     "developer",
		"dify.ai":          "ai-tools",
		"agno.ai":          "ai-tools",
		"mastra.ai":        "developer",
		"vellum.ai":        "ai-tools",

		// Voice / Speech
		"cartesia.ai":      "ai-tools",
		"lmnt.com":         "ai-tools",

		// Image / Video / 3D
		"fal.ai":           "ai-tools",
		"pika.art":         "ai-tools",
		"tripo3d":          "ai-tools",

		// Music
		"loudly.com":       "ai-tools",
		"lalal.ai":         "ai-tools",
		"mubert.com":       "ai-tools",

		// Vector DBs
		"milvus.io":        "data",
		"zilliz.com":       "data",

		// Real Estate
		"attomdata":        "data",
		"rentcast":         "data",
		"housecanary":      "data",
		"estated.com":      "data",

		// Legal Tech
		"sec-api.io":       "data",
		"courtlistener":    "data",

		// Sports Data
		"sportradar":       "data",
		"sportsdata.io":    "data",
		"api-sports":       "data",
		"sportmonks":       "data",
		"thesportsdb":      "data",

		// Wearables / Fitness
		"tryterra":         "health",
		"sahha.ai":         "health",

		// Food / Nutrition
		"spoonacular":      "data",
		"edamam.com":       "data",
		"nutritionix":      "data",

		// Climate / Environmental
		"climatiq":         "data",
		"open-meteo":       "data",
		"getambee":         "data",

		// Supply Chain
		"flexport":         "ecommerce",
		"shipengine":       "ecommerce",
		"fleetbase":        "ecommerce",

		// Bioinformatics
		"uniprot":          "health",
		"rcsb.org":         "health",

		// Fintech Banking
		"moderntreasury":   "finance",
		"moov.io":          "finance",
		"lithic.com":       "finance",
		"column.com":       "finance",
		"increase.com":     "finance",
		"mangopay":         "finance",
		"getlago":          "finance",
		"tryfinch":         "finance",

		// KYC / Identity
		"withpersona":      "security",
		"onfido.com":       "security",
		"socure.com":       "security",
		"alloy.com":        "security",

		// B2B Data
		"clay.com":         "data",
		"apollo.io":        "data",
		"peopledatalabs":   "data",
		"proxycurl":        "data",

		// Healthcare
		"metriport":        "health",

		// Cybersecurity
		"shodan.io":        "security",
		"greynoise":        "security",

		// Geospatial
		"protomaps":        "developer",
		"overturemaps":     "developer",
		"felt.com":         "developer",

		// Travel
		"duffel.com":       "data",
		"kiwi.com":         "data",

		// HR / Payroll
		"merge.dev":        "developer",

		// E-commerce data
		"serpapi":           "data",

		// Email (newer)
		"loops.so":         "communication",

		// Agentic infra
		"browsercat":       "developer",

		// llms.txt early adopters
		"mintlify":         "developer",
		"tinybird":         "data",
		"flatfile":         "data",
		"plain.com":        "communication",
		"inkeep.com":       "ai-tools",
		"axiom.co":         "developer",
		"openphone":        "communication",
		"smartcar.com":     "developer",
		"stedi.com":        "developer",
		"infisical":        "security",
		"screenshotone":    "developer",
		"buildwithfern":    "developer",
		"tryvital":         "health",
		"projectdiscovery": "security",
		"conductor.is":     "developer",
		"ionq.com":         "developer",

		// Batch 3: fix "other" category sites
		// Communication / Email / Messaging
		"mailchimp":        "communication",
		"sinch.com":        "communication",
		"messagebird":      "communication",
		"zendesk":          "communication",
		"intercom.com":     "communication",
		"whereby.com":      "communication",
		"zoom.us":          "communication",
		"mailgun":          "communication",
		"drip.com":         "communication",
		"customer.io":      "communication",

		// Developer / CI/CD / Infra
		"crates.io":        "developer",
		"trigger.dev":      "developer",
		"circleci":         "developer",
		"semaphoreci":      "developer",
		"terraform.io":     "developer",
		"codesandbox":      "developer",
		"crawlee.dev":      "developer",
		"apify.com":        "developer",
		"buildkite":        "developer",
		"hub.docker":       "developer",
		"replit.com":       "developer",
		"firecrawl":        "developer",
		"pypi.org":         "developer",
		"bitbucket.org":    "developer",
		"gitlab.com":       "developer",
		"stackblitz":       "developer",
		"sourcegraph":      "developer",

		// Data / Weather / News / Analytics
		"opencagedata":     "data",
		"tomorrow.io":      "data",
		"usaspending":      "data",
		"developer.nytimes": "data",
		"brightdata":       "data",
		"timescale.com":    "data",
		"influxdata":       "data",
		"visualcrossing":   "data",
		"guardian.co":      "data",
		"gnews.io":         "data",
		"elastic.co":       "data",
		"api.worldbank":    "data",
		"rss2json":         "data",
		"brave.com":        "data",

		// Finance / Billing / Accounting
		"chargebee":        "finance",
		"paypal.com":       "finance",
		"quickbooks.intuit": "finance",
		"freshbooks":       "finance",
		"recurly.com":      "finance",

		// AI Tools / OCR / Image / Search
		"codeium.com":      "ai-tools",
		"reducto.ai":       "ai-tools",
		"mindee.com":       "ai-tools",
		"you.com":          "ai-tools",
		"relevanceai":      "ai-tools",
		"dust.tt":          "ai-tools",
		"mathpix.com":      "ai-tools",
		"ocr.space":        "ai-tools",
		"remove.bg":        "ai-tools",
		"imgix.com":        "ai-tools",

		// Productivity / Automation / Integration
		"ifttt.com":        "productivity",
		"tray.io":          "productivity",
		"workato.com":      "productivity",
		"legalzoom":        "productivity",

		// Security
		"ipinfo.io":        "security",

		// Final "other" cleanup
		"razorpay.com":     "finance",
		"moz.com":          "data",
		"temporal.io":      "developer",
		"rossum.ai":        "ai-tools",
		"clio.com":         "productivity",
		"mollie.com":       "finance",
		"tinypng.com":      "developer",

		// Batch 4: new seeds from research (2026-04-13)
		"speakeasy.com":    "developer",
		"scalar.com":       "developer",
		"readme.com":       "developer",
		"dub.co":           "data",
		"writer.com":       "ai-tools",
		"frigade.com":      "developer",
		"basehub.com":      "data",
		"openpipe.ai":      "ai-tools",
		"dotenvx.com":      "developer",
		"datafold.com":     "data",
		"dynamic.xyz":      "developer",
		"velt.dev":         "developer",
		"salesbricks.com":  "ecommerce",
		"hyperline.co":     "finance",
		"aporia.com":       "ai-tools",
		"pinata.cloud":     "developer",
		"wordlift.io":      "ai-tools",
		"micro1.ai":        "jobs",
		"campsite.com":     "communication",
		"portkey.ai":       "ai-tools",
		"context7.com":     "developer",
		"stainlessapi.com": "developer",
		"pulsemcp.com":     "developer",
		"mcp.so":           "developer",
		"opentools.com":    "ai-tools",
		"llmstxthub.com":   "data",

		// Batch 5: llms.txt directory imports (2026-04-14)
		"play.ai":          "ai-tools",
		"svelte.dev":       "developer",
		"answer.ai":        "ai-tools",
		"fastht.ml":        "developer",
		"ongoody.com":      "ecommerce",
		"embedchain.ai":    "ai-tools",
		"argil.ai":         "ai-tools",
		"axle.insure":      "data",
		"unifygtm.com":     "productivity",
		"fabric.inc":       "ecommerce",
		"meshconnect.com":  "finance",
		"skip.build":       "developer",
		"flowx.ai":         "productivity",
		"solidfi.com":      "finance",
		"cobo.com":         "finance",
		"dopp.finance":     "finance",
		"sardine.ai":       "security",
		"oxla.com":         "data",
		"aptible.com":      "developer",
		"rubric.com":       "ai-tools",
		"sitespeak.ai":     "ai-tools",
		"adyen.com":        "finance",
		"uploadcare.com":   "developer",
		"configcat.com":    "developer",
		"mariadb.com":      "data",
		"hydrolix.io":      "data",
		"printify.com":     "ecommerce",
		"readwise.io":      "data",
		"nuxt.com":         "developer",
		"nextjs.org":       "developer",
		"postman.com":      "developer",
		"nvidia.com":       "ai-tools",
		"retool.com":       "developer",
		"dreamhost.com":    "developer",
		"vite.dev":         "developer",
		"nextiva.com":      "communication",
		"claspo.io":        "developer",
		"brandefense.io":   "security",

		// Batch 6: post-bulk-crawl "other" cleanup (2026-04-14)
		// Developer / DevTools
		"trunk.io":              "developer",
		"semgrep.com":           "security",
		"searchcode.com":        "developer",
		"parseable.com":         "developer",
		"donobu.com":            "developer",
		"transloadit.com":       "developer",
		"dalfox.hahwul.com":     "security",
		"prisma.io":             "developer",
		"tolgee.io":             "developer",
		"scalekit.com":          "security",
		"justrunmy.app":         "developer",
		"better-auth.com":       "security",
		"arcjet.com":            "security",
		"skyvern.com":           "developer",
		"gofastmcp.com":         "developer",
		"control-plane.io":      "developer",
		"fault-project.com":     "developer",
		"webflow.com":           "developer",
		"launchdarkly.com":      "developer",
		"mcpserver.space":       "developer",
		"dxt.services":          "developer",
		"pipeboard.co":          "developer",
		"docspring.com":         "developer",
		"iplocate.io":           "data",
		"webnode.com":           "developer",
		"wordpress.com":         "developer",
		"twittershots.com":      "developer",
		"rhino-inquisitor.com":  "developer",
		"rolalabs.in":           "developer",
		"anytrack.io":           "developer",

		// AI-Tools
		"quicktranscript.app":   "ai-tools",
		"necto.co":              "ai-tools",
		"findmine.com":          "ai-tools",
		"restorephoto.online":   "ai-tools",
		"hypestudio.org":        "ai-tools",
		"aipageready.com":       "ai-tools",
		"promptpilot.online":    "ai-tools",
		"blueshift.com":         "ai-tools",
		"miro.com":              "ai-tools",
		"rankability.com":       "ai-tools",

		// Data
		"lotsofcsvs.com":        "data",
		"dataforseo.com":        "data",
		"datastax.com":          "data",
		"theirstack.com":        "data",
		"builtwith.com":         "data",

		// Ecommerce
		"newnorm.shop":          "ecommerce",
		"sweetandbrew.com":      "ecommerce",
		"bwstays.com":           "ecommerce",
		"greetwell.com":         "ecommerce",
		"maplebridge.io":        "ecommerce",
		"alleastbayproperties.com": "ecommerce",
		"printify":              "ecommerce",

		// Productivity
		"waitlister.me":         "productivity",
		"youropinion.is":        "productivity",
		"thecrawltool.com":      "productivity",
		"semalt.com":            "productivity",
		"gravitywp.com":         "productivity",

		// News
		"informedclearly.com":   "news",
		"zadar.tv":              "news",

		// Jobs
		"upstaff.com":           "jobs",
		"unautomated.xyz":       "jobs",

		// Finance
		"govtribe.com":          "finance",
		"tip.md":                "finance",

		// Batch 7: post-MCP-registry-bulk-crawl "other" cleanup (2026-04-13)
		// AI-Tools / MCP hubs / agent infrastructure
		"0nmcp.com":                "ai-tools",
		"octo-dock.com":            "ai-tools",
		"fidensa.com":              "ai-tools",
		"prover.axiomatic-ai.com":  "ai-tools",
		"axiomatic-ai.com":         "ai-tools",
		"occam.fit":                "ai-tools",
		"pilotgentic.com":          "ai-tools",
		"moderatecontent.com":      "ai-tools",
		"disco.leap-labs.com":      "ai-tools",
		"leap-labs.com":            "ai-tools",
		"gleanmark.com":            "ai-tools",
		"prowl.chat":               "ai-tools",
		"hjarni.com":               "ai-tools",
		"getunblocked.com":         "ai-tools",
		"audioscrape.com":          "ai-tools",
		"promptot.com":             "ai-tools",

		// Finance / Fintech / Crypto
		"prereason.com":            "finance",
		"revettr.com":              "finance",
		"lona.agency":              "finance",
		"getcurrentoffer.com":      "finance",
		"youneedabudget.com":       "finance",
		"bigredcloud.com":          "finance",
		"audioalpha.io":            "finance",
		"emc2ai.io":                "finance",
		"blockscout.com":           "finance",
		"dexpaprika.com":           "finance",
		"bamboosnow.co":            "finance",

		// Data / Research / Geospatial
		"dchub.cloud":              "data",
		"monarchinitiative.org":    "health",
		"wolframalpha.com":         "data",
		"deepmapai.com":            "data",
		"daedalmap.com":            "data",
		"olyport.com":              "data",
		"meteosource.com":          "data",
		"exchangerate-api.com":     "data",
		"fedlex-connector.ch":      "data",
		"opencaselaw.ch":           "data",
		"clickmeter.com":           "data",
		"canada-holidays.ca":       "data",
		"goveda.com":               "data",
		"sabermetrics.blazesportsintel.com": "data",
		"blazesportsintel.com":     "data",
		"guruwalk.com":             "data",

		// Developer / DevTools / PDF / APIs
		"rapidapi.com":             "developer",
		"agentdomainsearch.com":    "developer",
		"catchdoms.com":            "developer",
		"domainkits.com":           "developer",
		"instapods.com":            "developer",
		"prodlint.com":             "developer",
		"codesearch.debian.net":    "developer",
		"kubernetes.io":            "developer",
		"dev.to":                   "developer",
		"pdfbroker.io":             "developer",
		"formapi.io":               "developer",
		"shotstack.io":             "developer",
		"stoplight.io":             "developer",
		"image-charts.com":         "developer",
		"thenounproject.com":       "developer",
		"fungenerators.com":        "developer",
		"qrcodly.de":               "developer",
		"cloudrf.com":              "developer",
		"mymlh-mcp.git.ci":         "developer",
		"jooq-mcp.martinelli.ch":   "developer",
		"martinelli.ch":            "developer",
		"waveguard-api":            "developer",
		"wix.com":                  "developer",

		// Productivity / Workflow
		"app.basicops.com":         "productivity",
		"basicops.com":             "productivity",
		"automatedclientacquisition.com": "productivity",
		"brandomica.com":           "productivity",
		"brandcode.studio":         "productivity",
		"epublys.com":              "productivity",
		"flat.io":                  "productivity",
		"readinglist.live":         "productivity",
		"memesio.com":              "productivity",
		"allgoodinsp.com":          "productivity",
		"handwrytten.com":          "productivity",
		"paichart.app":             "productivity",
		"notion.com":               "productivity",
		"heylead-api":              "productivity",

		// Education
		"admit-coach.com":          "education",
		"nuberea.com":              "education",
		"bible.simplecohortllc.com": "education",
		"simplecohortllc.com":      "education",
		"poemist.com":              "education",

		// News / Media
		"biztoc.com":               "news",
		"groundhog-day.com":        "news",
		"whisky-circle.info":       "news",

		// Jobs
		"qubitsok.com":             "jobs",

		// Ecommerce / Real estate / Property
		"immoswipe.ch":             "ecommerce",
		"la-palma24.net":           "ecommerce",
		"reilize.com":              "ecommerce",
		"laei.ro":                  "ecommerce",
		"shapedjt.com":             "ecommerce",
		"orbiteos.cloud":           "ecommerce",
		"freetv-app.com":           "ecommerce",

		// Communication
		"mandrillapp.com":          "communication",

		// Sports (mapped to data — no sports category)
		"flaim.app":                "data",

		// Screenshot / media services
		"urlbox.io":                "developer",

		// Batch 8: post-second-bulk-crawl "other" cleanup (2026-04-15)
		// Developer / MCP infra / APIs
		"reqres.in":                      "developer",
		"mcp-router.net":                 "developer",
		"mcpjam.com":                     "developer",
		"sushimcp.com":                   "developer",
		"ragrabbit.com":                  "developer",
		"hyprmcp.com":                    "developer",
		"metrifyr.cloud":                 "developer",
		"contextplus.vercel.app":         "developer",
		"fastapi-mcp.tadata.com":         "developer",
		"sec-edgar-mcp.amorelli.tech":    "developer",
		"jupyter-mcp-server.datalayer.tech": "developer",
		"shields.io":                     "developer",
		"cloudconvert.com":               "developer",
		"urlscan.io":                     "security",
		"postcodes.io":                   "data",
		"geodb-cities-api.wirefreethought.com": "data",
		"improvmx.com":                   "communication",
		"jamdesk.com":                    "developer",
		"pypi.python.org":                "developer",
		"strava.github.io":               "developer",
		"sonarsource.com":                "developer",
		"n8n.partnerlinks.io":            "productivity",
		"md.genedai.me":                  "developer",
		"color.serialif.com":             "developer",
		"dev.twitch.tv":                  "communication",

		// AI-tools / agentic / MCP products
		"tandem.ac":                      "ai-tools",
		"misaligned.top":                 "ai-tools",
		"usesideways.com":                "ai-tools",
		"sideways.aribadernatal.com":     "ai-tools",
		"translatesheet.com":             "ai-tools",
		"bottalk.io":                     "ai-tools",
		"xactions.app":                   "ai-tools",

		// Finance / Crypto / Trading
		"coinmarketcap.com":              "finance",
		"stockmarketscan.com":            "finance",
		"lendtrain.com":                  "finance",
		"tosheroon.com":                  "finance",

		// Data / Research / Open Data
		"supplymaven.com":                "data",
		"signals.global":                 "data",
		"opendata.umea.se":               "data",
		"data.nantesmetropole.fr":        "data",
		"data.ratp.fr":                   "data",
		"gettreatmenthelp.com":           "health",
		"wger.de":                        "health",

		// Ecommerce / Real Estate
		"smarthomeexplorer.com":          "ecommerce",
		"xfish.hu":                       "ecommerce",

		// Batch 9: post-recategorize-only pass cleanup (2026-04-15)
		// Developer / APIs / DevTools
		"zipcodeapi.com":                 "data",
		"linkpreview.net":                "developer",
		"filestack.com":                  "developer",
		"gorules.io":                     "developer",
		"urlbae.com":                     "developer",
		"zenquotes.io":                   "data",
		"blackhistoryapi.io":             "data",
		"scorebat.com":                   "data",
		"data.oddsmagnet.com":            "data",
		"oddsmagnet.com":                 "data",
		"pipedream-policy-brief.vercel.app": "data",
		"corrently.io":                   "data",

		// AI-tools
		"imagine.art":                    "ai-tools",
		"voice.couillardelectricllc.com": "ai-tools",
		"ikangai.com":                    "ai-tools",
		"randomlovecraft.com":            "ai-tools",

		// Finance
		"billplz.com":                    "finance",
		"aruannik.ee":                    "finance",

		// Education
		"kaggle.com":                     "education",
		"slidemaster.tw":                 "education",

		// Productivity / personal / agency
		"taishoku-anshin-daiko.com":      "productivity",
		"berger.team":                    "productivity",
		"modeser.com":                    "productivity",
		"campos.works":                   "productivity",
		"angshumangupta.com":             "productivity",
		"cesaryague.es":                  "productivity",

		// Developer (personal tech blogs)
		"sethhobson.com":                 "developer",
		"glucn.com":                      "developer",

		// Ecommerce
		"photo-fotograf.com":             "ecommerce",

		// 2026-04-15 bulk-import cleanup: push top "other" domains into real buckets.
		// Each mapped from the NHS API's /api/v1/search?category=other listing after
		// mcp.so discovery run surfaced many un-ruled agent-first sites.
		"attio.com":             "productivity",
		"ticktick.com":           "productivity",
		"coda.io":                "productivity",
		"approval.studio":        "productivity",
		"allourthings.io":        "productivity",
		"dynamoi.com":            "productivity",
		"zenda.com.ar":           "productivity",
		"copus.network":          "productivity",
		"notion.site":            "productivity",
		"urlbox.com":             "developer",
		"gitingest.com":          "developer",
		"cronalert.com":          "developer",
		"appscripthub.com":       "developer",
		"shadcnstudio.com":       "developer",
		"stackfox.co":            "security",
		"vatsense.com":           "finance",
		"frihet.io":              "finance",
		"moonpay.com":            "finance",
		"ib.phos.nz":             "finance",
		"cryptorugmunch.app":     "finance",
		"codex.io":               "data",
		"swisscarinfo.ch":        "data",
		"spaceprompts.com":       "ai-tools",
		"superlines.io":          "ai-tools",
		"aistatusdashboard.com":  "ai-tools",
		"echomindr.com":          "ai-tools",
		"rube.app":               "ai-tools",
		"creativeclaw.co":        "ai-tools",
		"sourcelibrary.org":      "education",

		// Batch 10: 2026-04-15 post-bulk-crawl final "other" trim
		"jseek.co":               "jobs",
		"propfirmdealfinder.com": "finance",
		"rftools.io":             "developer",
		"crunchtools.com":        "developer",
		"onetool.beycom.online":  "ai-tools",
		"hukuk.damdafa.com":      "productivity",
		"foto.damdafa.com":       "ai-tools",
		"ai.fin-discovery.com":   "finance",
		"fin-discovery.com":      "finance",

		// Batch 11: 2026-04-15 session end — Foundry siblings + well-known "other" sites
		"passdown.arflow.io":    "productivity",
		"poddrop.arflow.io":     "ecommerce",
		"typefully.com":         "communication",
		"wasabi.com":             "data",
		"databuddy.cc":           "developer",
		"dealsurface.com":        "data",
		"barevalue.com":          "productivity",
		"open.bigmodel.cn":       "ai-tools",
		"agentbazaar.tech":       "ai-tools",

		// Batch 12: 2026-04-15 session end — remaining clear "other" wins
		"ifr.spocont.com":              "finance",
		"booboooking.com":               "productivity",
		"doktor.mx":                     "health",
		"cloudlatitude.io":              "developer",
		"custom-icon-badges.demolab.com": "developer",
		"agent.cryptopolitan.com":       "finance",
		"theysaidso.com":                "data",
	}
	for domainKey, cat := range domainRules {
		if strings.Contains(d, domainKey) {
			return cat
		}
	}

	// Pass 1.4: MCP server domains (anywhere in the domain) → developer infrastructure.
	// Catches many bulk-imported one-off hosted MCP servers like
	// *-mcp.*, mcp-*.com, *.mcp.foo, mcp anywhere in leftmost label, etc.
	if strings.Contains(d, "-mcp.") || strings.Contains(d, "-mcp-") ||
		strings.HasSuffix(d, "-mcp") || strings.Contains(d, ".mcp.") ||
		strings.HasSuffix(d, "mcp.com") || strings.HasSuffix(d, "mcp.io") ||
		strings.HasSuffix(d, "mcp.net") || strings.HasSuffix(d, "mcp.app") {
		return "developer"
	}
	// subdomain containing "mcp" (e.g., verylongmcp.example.com)
	if first := strings.SplitN(d, ".", 2); len(first) > 0 && strings.Contains(first[0], "mcp") {
		return "developer"
	}

	// Cloud Run / Vercel API-style ephemeral hostnames (e.g., fabric-api-*.run.app,
	// the-lounge-*.run.app) are almost always developer tooling or API backends.
	if strings.HasSuffix(d, ".run.app") || strings.HasSuffix(d, ".vercel.app") ||
		strings.HasSuffix(d, ".fly.dev") || strings.HasSuffix(d, ".modal.run") {
		return "developer"
	}
	if strings.Contains(d, "-api.") || strings.Contains(d, "-api-") ||
		strings.HasPrefix(d, "api-") {
		return "developer"
	}

	// Pass 1.5: subdomain + TLD hints (next-highest confidence after exact rules)
	// api.* subdomains are almost always data/API services
	if strings.HasPrefix(d, "api.") || strings.HasPrefix(d, "docs.api.") {
		return "data"
	}
	// developer.* / docs.* are dev platform docs
	if strings.HasPrefix(d, "developer.") || strings.HasPrefix(d, "developers.") || strings.HasPrefix(d, "docs.") {
		return "developer"
	}
	// mcp.* subdomains are hosted MCP servers — developer infrastructure
	if strings.HasPrefix(d, "mcp.") {
		return "developer"
	}
	// Pass 2: keyword matches on description/name (lower confidence)
	combined := desc + " " + name
	type catRule struct {
		name     string
		keywords []string
	}
	rules := []catRule{
		{"jobs", []string{"job board", "career board", "hiring platform", "recruiting platform"}},
		{"health", []string{"healthcare", "medical", "clinical trial", "biotech", "pharmaceutical"}},
		{"education", []string{"education platform", "online course", "learning platform", "tutorial platform"}},
		{"ecommerce", []string{"ecommerce", "online store", "shopping", "retail platform"}},
		{"finance", []string{"fintech", "payment processing", "banking", "trading platform", "investment"}},
		{"security", []string{"cybersecurity", "identity verification", "vulnerability", "penetration testing"}},
		{"ai-tools", []string{"language model", "machine learning", "inference", "embeddings", "text-to-speech", "speech-to-text", "generative ai"}},
		{"data", []string{"database", "data warehouse", "analytics platform", "etl", "data integration", "data pipeline"}},
		{"developer", []string{"developer platform", "devtool", "developer tool", "hosting platform", "deployment", "runtime", "infrastructure"}},
		{"communication", []string{"messaging platform", "chat platform", "sms api", "email delivery", "push notification", "real-time communication", "notification service"}},
		{"productivity", []string{"project management", "task management", "collaboration platform", "workflow automation", "no-code", "low-code", "scheduling platform", "form builder", "crm platform"}},
		{"ai-tools", []string{"llm observability", "llmops", "agent framework", "agent platform", "ai orchestration", "prompt engineering", "model serving", "ai gateway"}},
		{"developer", []string{"mcp server", "mcp client", "model context protocol", "mcp-server", "hosted mcp", "remote mcp"}},
	}

	for _, rule := range rules {
		for _, kw := range rule.keywords {
			if strings.Contains(combined, kw) {
				return rule.name
			}
		}
	}

	// Pass 3: broader single-word keyword fallback. Lower confidence but catches
	// many bulk imports that had narrow phrasing evade pass 2.
	broadKeywords := []struct {
		cat      string
		keywords []string
	}{
		{"ai-tools", []string{"ai assistant", "ai agent", "llm ", "chatbot", "image generation", "text generation", "voice ai", "ai video"}},
		{"developer", []string{"sdk", "open source", "devtools", "code editor", "cli tool", "api client", "webhook", "git-based"}},
		{"data", []string{"dataset", "time series", "weather data", "financial data", "news api", "search api", "scraping"}},
		{"security", []string{"encryption", "mfa", "sso", "audit log", "compliance"}},
		{"finance", []string{"invoice", "billing", "subscription", "stripe", "crypto exchange"}},
		{"ecommerce", []string{"product catalog", "checkout", "storefront", "order management", "inventory"}},
		{"communication", []string{"email api", "sms ", "voip", "chat widget", "inbox", "customer support"}},
		{"productivity", []string{"task tracker", "kanban", "gantt", "meeting notes", "scheduling"}},
		{"health", []string{"patient", "diagnosis", "ehr ", "wellness", "fitness tracker"}},
		{"education", []string{"classroom", "tutoring", "lesson plan", "quiz ", "learner"}},
		{"jobs", []string{"applicant tracking", "recruiter", "job listing", "talent pool"}},
		{"ecommerce", []string{"real estate", "property listing", "ferienhaus", "vacation rental", "short term rental"}},
		{"finance", []string{"bitcoin", "cryptocurrency", "crypto ", "stock market", "mortgage", "cashback", "credit card"}},
		{"ai-tools", []string{"context engine", "agentic", "ai-powered", "prompt management", "vector memory"}},
		{"data", []string{"open data", "public data", "statistical data", "census data", "government data"}},
		{"news", []string{"breaking news", "daily news", "news aggregator"}},
		// 2026-04-15 additions: capture more mcp.so-discovered domains via keywords.
		{"developer", []string{"screenshot service", "uptime monitoring", "status monitoring", "cron job", "api monitor", "monitoring dashboard"}},
		{"productivity", []string{"crm", "proofing software", "online proofing", "calendar app", "to-do list", "todo list", "notion", "collaborative workspace"}},
		{"ai-tools", []string{"ai prompt", "prompt manager", "ai search", "ai brand", "ai status", "founder search", "ai-native"}},
		{"finance", []string{"vat number", "vat validation", "tax api", "accounting", "bookkeeping", "billing platform", "invoicing"}},
		{"data", []string{"blockchain data", "token data", "onchain data", "vehicle data", "registry api"}},
		{"security", []string{"bot traffic", "bot detection", "bot control", "rate limit"}},
		{"education", []string{"ancient text", "digital library", "scholarly archive", "research library"}},
	}
	for _, rule := range broadKeywords {
		for _, kw := range rule.keywords {
			if strings.Contains(combined, kw) {
				return rule.cat
			}
		}
	}

	// TLD-based last-resort hints
	if strings.HasSuffix(d, ".ai") || strings.HasSuffix(d, ".ml") {
		return "ai-tools"
	}
	if strings.HasSuffix(d, ".dev") || strings.HasSuffix(d, ".sh") {
		return "developer"
	}

	return "other"
}

// generateTags creates search-friendly tags from the site's signals, domain, and description.
func generateTags(site *models.Site) pq.StringArray {
	tagSet := make(map[string]bool)

	// Signal-based tags
	if site.HasLLMsTxt {
		tagSet["llms-txt"] = true
	}
	if site.HasAIPlugin {
		tagSet["ai-plugin"] = true
	}
	if site.HasOpenAPI {
		tagSet["openapi"] = true
		tagSet["api"] = true
	}
	if site.HasStructuredAPI {
		tagSet["api"] = true
	}
	if site.HasMCPServer {
		tagSet["mcp"] = true
	}
	if site.HasRobotsAI {
		tagSet["ai-friendly"] = true
	}

	// Extract keywords from description and name (pad with spaces for boundary matching)
	combined := " " + strings.ToLower(site.Description+" "+site.Name) + " "
	keywordMap := map[string][]string{
		"payment":        {"payment", "payments", "pay ", "checkout", "billing"},
		"api":            {"api ", "apis", "rest api", "graphql", "endpoint"},
		"database":       {"database", "postgres", "mysql", "sql", "nosql", "db "},
		"authentication": {"auth", "login", "oauth", "sso", "identity"},
		"email":          {"email", "smtp", "inbox", "mail"},
		"messaging":      {"messaging", "chat", "sms", "notification"},
		"hosting":        {"hosting", "deploy", "server", "cloud"},
		"ai":             {"artificial intelligence", " ai ", "machine learning", "llm", "gpt", "neural"},
		"ml":             {"machine learning", "deep learning", "training", "inference", "model"},
		"search":         {"search engine", "search api", "search ", "discovery"},
		"analytics":      {"analytics", "tracking", "metrics", "insights"},
		"storage":        {"storage", "file", "blob", "upload", "cdn"},
		"monitoring":     {"monitoring", "observability", "logging", "alerting", "apm"},
		"ecommerce":      {"ecommerce", "e-commerce", "commerce", "storefront", "shopping", "cart"},
		"fintech":        {"fintech", "financial", "banking", "trading", "investment"},
		"security":       {"security", "encryption", "vulnerability", "penetration", "firewall"},
		"devtools":       {"developer tool", "dev tool", "sdk", "cli", "framework"},
		"automation":     {"automation", "workflow", "integration", "orchestration"},
		"vector-db":      {"vector", "embeddings", "similarity", "semantic search"},
		"healthcare":     {"health", "medical", "clinical", "biotech", "pharma"},
		"jobs":           {"job board", "jobs", "hiring", "recruiting", "career"},
		"cms":            {"content management", "headless cms", "cms"},
		"data-pipeline":  {"etl", "data pipeline", "data integration", "ingestion"},
		"translation":    {"translat", "localization", "multilingual", "language translation"},
		"shipping":       {"shipping", "shipment", "logistics", "delivery", "parcel", "freight"},
		"calendar":       {"calendar", "scheduling", "appointment", "booking"},
		"travel":         {"travel", "flight", "hotel", "booking", "airline", "reservation"},
		"music":          {"music", "audio", "streaming", "playlist", "podcast"},
		"crm":            {"crm", "customer relationship", "sales pipeline", "lead management"},
		"notifications":  {"notification", "push notification", "alert", "webhook"},
		"forms":          {"form builder", "survey", "questionnaire"},
		"testing":          {"testing", "test automation", "qa", "browser testing"},
		"social-media":     {"social media", "social network", "social platform"},
		"image-generation": {"image generation", "text-to-image", "image synthesis", "generate images"},
		"video-generation": {"video generation", "text-to-video", "video synthesis"},
		"pdf":              {"pdf", "document conversion", "document processing"},
	}

	for tag, keywords := range keywordMap {
		for _, kw := range keywords {
			if strings.Contains(combined, kw) {
				tagSet[tag] = true
				break
			}
		}
	}

	// Domain-based tags for well-known services
	domainTags := map[string][]string{
		"stripe":     {"payment", "api", "fintech", "billing", "subscriptions"},
		"plaid":      {"payment", "api", "fintech", "banking"},
		"square":     {"payment", "ecommerce", "pos"},
		"shopify":    {"ecommerce", "api", "storefront"},
		"twilio":     {"messaging", "sms", "api", "voice"},
		"sendgrid":   {"email", "api"},
		"resend":     {"email", "api"},
		"postmark":   {"email", "api"},
		"github":     {"api", "devtools", "git", "code"},
		"openai":     {"ai", "ml", "api", "llm"},
		"anthropic":  {"ai", "ml", "api", "llm"},
		"supabase":   {"database", "api", "auth", "realtime"},
		"cloudflare": {"cdn", "security", "dns", "hosting"},
		"vercel":     {"hosting", "devtools", "frontend"},
		"sentry":     {"monitoring", "error-tracking", "devtools"},
		"posthog":    {"analytics", "devtools"},
		"datadog":    {"monitoring", "observability"},
		"auth0":      {"authentication", "security", "api"},
		"clerk":      {"authentication", "api"},
		"pinecone":   {"vector-db", "ai", "api"},
		"weaviate":   {"vector-db", "ai", "api"},
		"zapier":     {"automation", "api", "integration"},
		"deepl":      {"translation", "api", "nlp"},
		"easypost":   {"shipping", "api", "logistics"},
		"goshippo":   {"shipping", "api", "logistics"},
		"shipstation": {"shipping", "api", "ecommerce"},
		"cal.com":    {"calendar", "scheduling", "api"},
		"calendly":   {"calendar", "scheduling", "api"},
		"amadeus":    {"travel", "flights", "api"},
		"hubspot":    {"crm", "marketing", "api"},
		"spotify":    {"music", "audio", "api"},
		"onesignal":  {"notifications", "push", "api"},
		"typeform":   {"forms", "surveys", "api"},
		"browserstack": {"testing", "qa", "api"},
		"developer.x":  {"social-media", "api", "twitter"},
		"facebook.com":  {"social-media", "api"},
		"reddit.com":    {"social-media", "api", "community"},
		"buffer.com":    {"social-media", "scheduling", "api"},
		"hootsuite":     {"social-media", "scheduling", "analytics"},
		"leonardo.ai":   {"image-generation", "ai", "api"},
		"runway.com":    {"video-generation", "ai", "api"},
		"luma.ai":       {"video-generation", "3d", "ai"},
		"pdf.co":        {"pdf", "document", "api"},
		"smallpdf":      {"pdf", "document"},
		"ntfy.sh":       {"notifications", "push", "api"},
	}
	d := strings.ToLower(site.Domain)
	for domainKey, tags := range domainTags {
		if strings.Contains(d, domainKey) {
			for _, t := range tags {
				tagSet[t] = true
			}
			break
		}
	}

	var tags []string
	for t := range tagSet {
		tags = append(tags, t)
	}
	return pq.StringArray(tags)
}
