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

	"github.com/unitedideas/nothumansearch/internal/models"
)

var client = &http.Client{
	Timeout: 10 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 3 {
			return fmt.Errorf("too many redirects")
		}
		return nil
	},
}

const userAgent = "NotHumanSearch/1.0 (+https://nothumansearch.com/about)"

func fetch(rawURL string) (string, int, error) {
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/plain, application/json, text/html, */*")

	resp, err := client.Do(req)
	if err != nil {
		return "", 0, err
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
	base := fmt.Sprintf("%s://%s", u.Scheme, u.Host)

	site := &models.Site{
		Domain:      u.Host,
		URL:         base,
		CrawlStatus: "success",
	}

	now := time.Now()
	site.LastCrawledAt = &now

	// Fetch homepage for title/description
	if body, status, err := fetch(base); err == nil && status == 200 {
		site.Name = extractTitle(body)
		site.Description = extractMetaDescription(body)
		site.HasSchemaOrg = strings.Contains(body, "schema.org")
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

	// Check OpenAPI spec
	for _, path := range []string{"/openapi.yaml", "/openapi.json", "/api/openapi.yaml", "/api/openapi.json", "/swagger.json", "/api-docs"} {
		if body, status, err := fetch(base + path); err == nil && status == 200 && len(body) > 50 {
			if strings.Contains(body, "openapi") || strings.Contains(body, "swagger") || strings.Contains(body, "paths") {
				site.HasOpenAPI = true
				// Extract summary
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

	// Check for structured API (look for /api/ or API docs)
	for _, path := range []string{"/api", "/api/v1", "/docs", "/api-docs", "/developer"} {
		if _, status, err := fetch(base + path); err == nil && (status == 200 || status == 301 || status == 302) {
			site.HasStructuredAPI = true
			break
		}
	}

	// Calculate score
	site.AgenticScore = models.CalculateScore(site)

	// Auto-categorize
	site.Category = categorize(site)

	log.Printf("Crawled %s: score=%d llms=%v plugin=%v openapi=%v robots=%v api=%v schema=%v",
		site.Domain, site.AgenticScore,
		site.HasLLMsTxt, site.HasAIPlugin, site.HasOpenAPI,
		site.HasRobotsAI, site.HasStructuredAPI, site.HasSchemaOrg)

	return site, nil
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
		start = start + end
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
	}
	for domainKey, cat := range domainRules {
		if strings.Contains(d, domainKey) {
			return cat
		}
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
	}

	for _, rule := range rules {
		for _, kw := range rule.keywords {
			if strings.Contains(combined, kw) {
				return rule.name
			}
		}
	}
	return "other"
}
