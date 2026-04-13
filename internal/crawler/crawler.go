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
	// Look for content= nearby
	region := html[max(0, idx-200) : min(len(html), idx+300)]
	contentIdx := strings.Index(strings.ToLower(region), `content="`)
	if contentIdx == -1 {
		return ""
	}
	start := contentIdx + 9
	end := strings.Index(region[start:], `"`)
	if end == -1 {
		return ""
	}
	desc := region[start : start+end]
	if len(desc) > 500 {
		desc = desc[:500]
	}
	return desc
}

func categorize(site *models.Site) string {
	d := strings.ToLower(site.Domain)
	desc := strings.ToLower(site.Description)
	combined := d + " " + desc

	categories := map[string][]string{
		"jobs":       {"job", "career", "hiring", "recruit"},
		"ai-tools":  {"ai tool", "machine learning", "llm", "gpt", "model"},
		"developer": {"developer", "api", "sdk", "framework", "devtool"},
		"data":      {"data", "analytics", "database", "dataset"},
		"finance":   {"finance", "fintech", "payment", "banking"},
		"health":    {"health", "medical", "clinical", "biotech"},
		"education": {"education", "learn", "course", "tutorial"},
		"ecommerce": {"shop", "store", "ecommerce", "commerce", "retail"},
		"security":  {"security", "auth", "identity", "cyber"},
	}

	for cat, keywords := range categories {
		for _, kw := range keywords {
			if strings.Contains(combined, kw) {
				return cat
			}
		}
	}
	return "other"
}
