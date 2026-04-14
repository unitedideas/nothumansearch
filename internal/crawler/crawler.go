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

const userAgent = "NotHumanSearch/1.0 (+https://nothumansearch.ai/about)"

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
	// Also check if llms.txt or homepage mentions MCP
	if !site.HasMCPServer {
		contentToCheck := strings.ToLower(site.LLMsTxtContent + " " + site.Description)
		if strings.Contains(contentToCheck, "mcp server") || strings.Contains(contentToCheck, "mcp endpoint") ||
			strings.Contains(contentToCheck, "model context protocol") || strings.Contains(contentToCheck, "mcp-server") {
			site.HasMCPServer = true
		}
	}

	// Check for structured API — high-confidence paths first, then content-verified paths
	for _, path := range []string{"/api/v1", "/api/v2", "/api-docs", "/graphql"} {
		if _, status, err := fetch(base + path); err == nil && (status == 200 || status == 301 || status == 302) {
			site.HasStructuredAPI = true
			break
		}
	}
	if !site.HasStructuredAPI {
		// These paths need content verification (many sites have generic /docs or /developer pages)
		apiIndicators := []string{"api", "endpoint", "rest", "graphql", "sdk", "authentication",
			"rate limit", "api key", "access token", "bearer", "curl", "request", "response",
			"json", "webhook", "oauth", "api reference", "openapi", "swagger"}
		for _, path := range []string{"/api", "/docs/api", "/developer", "/developers"} {
			if body, status, err := fetch(base + path); err == nil && (status == 200 || status == 301 || status == 302) {
				bodyLower := strings.ToLower(body)
				matches := 0
				for _, indicator := range apiIndicators {
					if strings.Contains(bodyLower, indicator) {
						matches++
					}
				}
				// Need at least 3 API indicators to confirm this is actual API documentation
				if matches >= 3 {
					site.HasStructuredAPI = true
					break
				}
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
		{"communication", []string{"messaging platform", "chat platform", "sms api", "email delivery", "push notification", "real-time communication", "notification service"}},
		{"productivity", []string{"project management", "task management", "collaboration platform", "workflow automation", "no-code", "low-code", "scheduling platform", "form builder", "crm platform"}},
		{"ai-tools", []string{"llm observability", "llmops", "agent framework", "agent platform", "ai orchestration", "prompt engineering", "model serving", "ai gateway"}},
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
