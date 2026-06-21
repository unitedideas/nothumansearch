// Fix-my-score paid intake + Stripe checkout.
//
// Flow:
//
//	GET  /fix/{host}            intake form
//	POST /fix/{host}            validate + create geo_fix_jobs row + Stripe session, redirect
//	GET  /fix/success           thank-you page
//	POST /webhook/stripe        flip status=paid + Discord ping
//	GET  /api/v1/admin/geo-jobs (Bearer auth) list paid + pending orders
//
// Pricing: $199 one-time, 72hr turnaround (manual fulfillment for now).
// Lead-mode fallback — if STRIPE_SECRET_KEY is unset, intake is recorded
// with status="lead", Discord fires, and we follow up via Stripe invoice.
// No fake "paid" status. Prevents fraud/confusion when Stripe isn't wired.
package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/unitedideas/nothumansearch/internal/models"
	"github.com/unitedideas/nothumansearch/internal/notify"

	gostripe "github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/webhook"
)

const fixPriceCents = 19900

// reportPriceCents is the self-serve "full report" tier (cheaper, no manual
// fulfillment — delivers the already-auto-generated per-site report on success).
// PRE-STAGED, NOT LIVE: the number is the owner's 4pm 2026-06-02 decision.
// $29 is the OWNER-BRIEF recommendation; owner sets the final value before deploy.
const reportPriceCents = 2900

const fixTargetScore = 95

type FixHandler struct {
	DB            *sql.DB
	BaseURL       string
	WebhookSecret string
	// PreviewEnabled gates the per-site "here's exactly what we'd ship for YOUR
	// domain" block on the intake form (env NHS_FIX_PREVIEW=1). Reversible kill:
	// unset the env var + redeploy → reverts to the generic deliverables list.
	PreviewEnabled bool
}

func NewFixHandler(db *sql.DB, baseURL string) *FixHandler {
	gostripe.Key = os.Getenv("STRIPE_SECRET_KEY")
	if gostripe.Key == "" {
		log.Println("fix: STRIPE_SECRET_KEY not set, /fix/* runs in test mode (lead-capture only)")
	}
	webhookSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")
	if webhookSecret == "" && gostripe.Key != "" {
		log.Println("WARNING: fix: STRIPE_SECRET_KEY is set but STRIPE_WEBHOOK_SECRET is missing. Webhook signature verification will fail.")
	}
	return &FixHandler{
		DB:             db,
		BaseURL:        baseURL,
		WebhookSecret:  webhookSecret,
		PreviewEnabled: os.Getenv("NHS_FIX_PREVIEW") == "1",
	}
}

// fixPreviewBlock renders a per-site breakdown: for each agent signal the site is
// MISSING, a row showing the points it would gain + a copy-paste snippet templated
// with the real domain; for each signal already PRESENT, a green "done" row (proof
// we actually inspected the site). Returns HTML for insertion above the $199 CTA.
// Returns "" when the preview flag is off so the caller falls back to the static list.
func fixPreviewBlock(site *models.Site) string {
	type sig struct {
		have    bool
		pts     int
		label   string
		snippet string // copy-paste fix, templated with the domain; "" = no snippet
	}
	d := site.Domain
	sigs := []sig{
		{site.HasLLMsTxt, models.ScoreLLMsTxt, "llms.txt — agent-facing site summary",
			"# /llms.txt for " + d + "\n> One-line description of " + d + " for AI agents.\n\n## Key pages\n- /: homepage\n- /api: API docs (if any)"},
		{site.HasAIPlugin, models.ScoreAIPlugin, ".well-known/ai-plugin.json — ChatGPT/Claude manifest",
			"{\n  \"schema_version\": \"v1\",\n  \"name_for_model\": \"" + d + "\",\n  \"description_for_model\": \"What " + d + " does, for an AI agent.\",\n  \"api\": { \"type\": \"openapi\", \"url\": \"https://" + d + "/openapi.yaml\" }\n}"},
		{site.HasOpenAPI, models.ScoreOpenAPI, "openapi.yaml — public endpoints in OpenAPI 3", ""},
		{site.HasStructuredAPI, models.ScoreStructuredAPI, "/api/v1 — literal JSON index for agents", ""},
		{site.HasMCPServer, models.ScoreMCPServer, "MCP server — first-class agent tool access", ""},
		{site.HasRobotsAI, models.ScoreRobotsAI, "robots.txt — explicit AI-crawler allow rules",
			"# robots.txt for " + d + " — allow AI agents\nUser-agent: GPTBot\nAllow: /\nUser-agent: ClaudeBot\nAllow: /\nUser-agent: PerplexityBot\nAllow: /\nSitemap: https://" + d + "/sitemap.xml"},
		{site.HasSchemaOrg, models.ScoreSchemaOrg, "Schema.org JSON-LD — Organization/FAQ markup", ""},
	}
	var b strings.Builder
	gained := 0
	b.WriteString(`<div style="margin:1rem 0;"><p style="color:#ccc;line-height:1.6;margin:0 0 .75rem;">Here's exactly what we'd ship for <strong style="color:#fff;">` + html.EscapeString(d) + `</strong>:</p>`)
	for _, s := range sigs {
		if s.have {
			b.WriteString(`<div style="display:flex;gap:.6rem;align-items:baseline;padding:.4rem .6rem;border-left:2px solid #2e7d4f;background:rgba(46,125,79,.08);border-radius:0 4px 4px 0;margin-bottom:.4rem;"><span style="color:#4caf72;font-weight:700;">✓</span><span style="color:#bbb;font-size:.9rem;">` + html.EscapeString(s.label) + ` — <span style="color:#4caf72;">already present</span></span></div>`)
			continue
		}
		gained += s.pts
		b.WriteString(`<div style="padding:.5rem .6rem;border-left:2px solid var(--accent);background:rgba(217,119,87,.08);border-radius:0 4px 4px 0;margin-bottom:.5rem;"><div style="display:flex;gap:.6rem;align-items:baseline;"><span style="color:var(--accent);font-weight:700;font-family:'IBM Plex Mono',monospace;">+` + strconv.Itoa(s.pts) + `</span><span style="color:#fff;font-size:.9rem;font-weight:600;">` + html.EscapeString(s.label) + `</span></div>`)
		if s.snippet != "" {
			b.WriteString(`<pre style="margin:.5rem 0 0;padding:.6rem;background:#0d0d0e;border:1px solid #2a2a2b;border-radius:4px;overflow-x:auto;font-size:.72rem;line-height:1.4;color:#cbd5b5;white-space:pre;">` + html.EscapeString(s.snippet) + `</pre>`)
		}
		b.WriteString(`</div>`)
	}
	projected := site.AgenticScore + gained
	if projected > 100 {
		projected = 100
	}
	b.WriteString(`<div style="margin:.8rem 0 0;padding:.7rem .8rem;background:#0d0d0e;border-radius:6px;font-family:'IBM Plex Mono',monospace;font-size:.9rem;color:#ccc;">Projected: <span style="color:#888;">` + strconv.Itoa(site.AgenticScore) + `</span> &rarr; <span style="color:var(--accent);font-weight:700;font-size:1.1rem;">` + strconv.Itoa(projected) + `</span> <span style="color:#888;font-size:.78rem;">after the ` + strconv.Itoa(len(sigs)) + `-signal uplift</span></div></div>`)
	return b.String()
}

func writeFixJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func normalizeFixPaymentMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", "stripe", "stripe_checkout", "stripe_link", "link":
		return "stripe_checkout"
	case "spt", "stripe_spt", "shared_payment_token", "agentic_checkout", "agentic_commerce":
		return "stripe_acp_spt"
	case "stripe_acp", "acp":
		return "unsupported"
	case "mpp", "x402", "mpp_x402", "machine_payments", "machine_payments_x402", "stripe_machine_payments":
		return "machine_payments_x402"
	default:
		return "unsupported"
	}
}

func (h *FixHandler) CommerceManifest(w http.ResponseWriter, r *http.Request) {
	writeFixJSON(w, http.StatusOK, map[string]interface{}{
		"seller": map[string]interface{}{
			"id":            "nothumansearch",
			"name":          "Not Human Search",
			"url":           h.BaseURL,
			"contact_email": "hello@nothumansearch.ai",
		},
		"version":  "2026-05-01",
		"currency": "USD",
		"agentic_payments": map[string]interface{}{
			"ready":           true,
			"supported_modes": []string{"stripe_checkout", "stripe_link", "link", "stripe_spt"},
			"unsupported_modes": map[string]string{
				"stripe_acp": "Stripe Agentic Commerce Protocol is private-preview gated for this seller surface.",
				"x402":       "No Stripe machine payments / x402 endpoint is deployed for Not Human Search.",
				"mpp":        "No machine-payment endpoint is deployed for Not Human Search.",
			},
			"stripe_spt":               "Supported for the one-time GEO uplift product. Submit a Link-issued shared_payment_granted_token to /api/v1/checkout.",
			"link":                     "Available inside Stripe Checkout when enabled on the Stripe account.",
			"private_preview_required": []string{"stripe_acp", "x402"},
			"endpoints": map[string]string{
				"catalog":        h.BaseURL + "/api/v1/catalog",
				"quote":          h.BaseURL + "/api/v1/quote",
				"checkout":       h.BaseURL + "/api/v1/checkout",
				"api_subscribe":  h.BaseURL + "/api/v1/api-keys/subscribe",
				"api_activation": h.BaseURL + "/api/v1/api-keys/activate",
			},
		},
		"products": h.commerceProducts(),
	})
}

func (h *FixHandler) AgentJSON(w http.ResponseWriter, r *http.Request) {
	writeFixJSON(w, http.StatusOK, map[string]interface{}{
		"name":        "Not Human Search",
		"description": "Search engine for agent-ready sites ranked by agentic readiness.",
		"url":         h.BaseURL,
		"capabilities": []string{
			"agentic-readiness-search",
			"mcp-server-discovery",
			"geo-uplift-service",
			"paid-api-keys",
		},
		"api": map[string]string{
			"base_url": h.BaseURL + "/api/v1",
			"openapi":  h.BaseURL + "/openapi.yaml",
		},
		"mcp": map[string]string{
			"endpoint": h.BaseURL + "/mcp",
			"manifest": h.BaseURL + "/.well-known/mcp.json",
		},
		"commerce": map[string]interface{}{
			"manifest":                  h.BaseURL + "/.well-known/commerce.json",
			"catalog":                   h.BaseURL + "/api/v1/catalog",
			"quote":                     h.BaseURL + "/api/v1/quote",
			"checkout":                  h.BaseURL + "/api/v1/checkout",
			"api_subscribe":             h.BaseURL + "/api/v1/api-keys/subscribe",
			"payment_modes":             []string{"stripe_checkout", "stripe_link", "link", "stripe_spt"},
			"agentic_payments_ready":    true,
			"unsupported_payment_modes": []string{"stripe_acp", "x402", "mpp"},
			"private_preview_required":  []string{"stripe_acp", "x402"},
		},
		"contact": "hello@nothumansearch.ai",
	})
}

func (h *FixHandler) fixProduct() map[string]interface{} {
	return map[string]interface{}{
		"id":          "nhs_geo_fix_my_score",
		"name":        "Fix my agent-readiness score",
		"description": "Done-for-you GEO uplift pull request for one website.",
		"type":        "one_time_service",
		"price": map[string]interface{}{
			"amount":   fixPriceCents,
			"currency": "USD",
			"display":  "$199 one-time",
		},
		"fulfillment": map[string]interface{}{
			"turnaround_hours": 72,
			"target_score":     "90+ after merge or refund",
		},
		"required_metadata": []string{"host", "email"},
		"checkout": map[string]interface{}{
			"mode":            "stripe_checkout",
			"endpoint":        h.BaseURL + "/api/v1/checkout",
			"supported_modes": []string{"stripe_checkout", "stripe_link", "link", "stripe_spt"},
		},
	}
}

func (h *FixHandler) apiProduct(plan models.APIPlan) map[string]interface{} {
	return map[string]interface{}{
		"id":          apiProductID(plan),
		"name":        "Not Human Search API " + strings.Title(plan.Name),
		"description": fmt.Sprintf("%d REST/MCP calls per month after the anonymous quota.", plan.MonthlyLimit),
		"type":        "recurring_subscription",
		"plan":        plan.Name,
		"quota": map[string]interface{}{
			"monthly_limit": plan.MonthlyLimit,
			"unit":          "billable REST or MCP calls",
		},
		"price": map[string]interface{}{
			"amount":   plan.PriceCents,
			"currency": "USD",
			"display":  fmt.Sprintf("$%.2f/mo", float64(plan.PriceCents)/100),
			"interval": "month",
		},
		"required_metadata": []string{"email", "plan"},
		"checkout": map[string]interface{}{
			"mode":            "stripe_checkout",
			"endpoint":        h.BaseURL + "/api/v1/api-keys/subscribe",
			"method":          "POST",
			"content_type":    "application/json",
			"supported_modes": []string{"stripe_checkout", "stripe_link", "link"},
			"example_body": map[string]string{
				"email": "buyer@example.com",
				"plan":  plan.Name,
			},
		},
		"activation": map[string]interface{}{
			"endpoint": h.BaseURL + "/api/v1/api-keys/activate?session_id={CHECKOUT_SESSION_ID}",
			"method":   "GET",
			"note":     "Raw API keys are shown once after paid Stripe checkout activation.",
		},
	}
}

func (h *FixHandler) commerceProducts() []map[string]interface{} {
	products := []map[string]interface{}{h.fixProduct()}
	for _, plan := range models.APIPlans() {
		products = append(products, h.apiProduct(plan))
	}
	return products
}

func apiProductID(plan models.APIPlan) string {
	return "nhs_api_" + plan.Name
}

func apiPlanFromProductID(productID string) (models.APIPlan, bool) {
	id := strings.ToLower(strings.TrimSpace(productID))
	id = strings.TrimPrefix(id, "api_")
	id = strings.TrimPrefix(id, "nhs_api_")
	switch id {
	case "unlimited", "starter", "pro", "scale":
		// Legacy tier names still resolve (all collapse to the single plan).
		return models.APIPlanFor(id), true
	default:
		return models.APIPlan{}, false
	}
}

func scoreFixEligible(site *models.Site) bool {
	return site != nil && site.HasHardAgentSignal() && site.AgenticScore < fixTargetScore
}

func (h *FixHandler) CommerceCatalog(w http.ResponseWriter, r *http.Request) {
	writeFixJSON(w, http.StatusOK, map[string]interface{}{
		"seller":   "nothumansearch",
		"currency": "USD",
		"products": h.commerceProducts(),
	})
}

func (h *FixHandler) CommerceQuote(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	productID := strings.TrimSpace(r.URL.Query().Get("product_id"))
	if r.Method == http.MethodPost && r.Body != nil {
		var req struct {
			ProductID string `json:"product_id"`
			ProductId string `json:"productId"`
			Plan      string `json:"plan"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&req); err != nil && err != io.EOF {
			writeFixJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		if productID == "" {
			productID = req.ProductID
		}
		if productID == "" {
			productID = req.ProductId
		}
		if productID == "" && req.Plan != "" {
			productID = "nhs_api_" + req.Plan
		}
	}
	if productID == "" || productID == "nhs_geo_fix_my_score" {
		writeFixJSON(w, http.StatusOK, map[string]interface{}{
			"seller":            "nothumansearch",
			"product_id":        "nhs_geo_fix_my_score",
			"currency":          "USD",
			"amount":            fixPriceCents,
			"total":             fixPriceCents,
			"payment_mode":      "stripe_checkout",
			"required_metadata": []string{"host", "email"},
			"checkout_endpoint": h.BaseURL + "/api/v1/checkout",
		})
		return
	}
	plan, ok := apiPlanFromProductID(productID)
	if !ok {
		writeFixJSON(w, http.StatusNotFound, map[string]string{"error": "unknown product_id"})
		return
	}
	writeFixJSON(w, http.StatusOK, map[string]interface{}{
		"seller":            "nothumansearch",
		"product_id":        apiProductID(plan),
		"plan":              plan.Name,
		"currency":          "USD",
		"amount":            plan.PriceCents,
		"total":             plan.PriceCents,
		"billing_interval":  "month",
		"monthly_limit":     plan.MonthlyLimit,
		"payment_mode":      "stripe_checkout",
		"required_metadata": []string{"email", "plan"},
		"checkout_endpoint": h.BaseURL + "/api/v1/api-keys/subscribe",
		"checkout_method":   "POST",
		"checkout_payload": map[string]string{
			"email": "buyer@example.com",
			"plan":  plan.Name,
		},
		"activation_endpoint": h.BaseURL + "/api/v1/api-keys/activate?session_id={CHECKOUT_SESSION_ID}",
	})
}

// ServeHTTP routes /fix/{host} for GET (form) and POST (checkout).
// /fix/success is handled by the separate handler below.
func (h *FixHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	host := strings.TrimPrefix(r.URL.Path, "/fix/")
	host = strings.TrimSuffix(host, "/")
	if host == "" || host == "success" || host == "cancel" {
		// success + cancel are dedicated handlers registered separately
		http.NotFound(w, r)
		return
	}
	host = strings.ToLower(host)

	switch r.Method {
	case http.MethodGet:
		h.intakeForm(w, r, host)
	case http.MethodPost:
		h.createCheckout(w, r, host)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *FixHandler) intakeForm(w http.ResponseWriter, r *http.Request, host string) {
	site, err := models.GetSiteByDomain(h.DB, host)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if !scoreFixEligible(site) {
		if site.HasHardAgentSignal() && site.AgenticScore >= fixTargetScore {
			h.scoreFixCompletePage(w, site)
			return
		}
		http.NotFound(w, r)
		return
	}
	go models.LogIntentFromRequest(h.DB, r, "fix_page_view", "site", site.Domain, map[string]any{
		"score":    site.AgenticScore,
		"category": site.Category,
	})

	// Deliverables block: per-site preview when enabled, else the original static list.
	deliverables := `<p style="color:#ccc;line-height:1.6;">We ship the 6-file GEO uplift as a pull request against your repo:</p>
    <ul>
      <li><strong>llms.txt</strong> — agent-facing site summary with key routes</li>
      <li><strong>openapi.yaml</strong> — public endpoints described in OpenAPI 3</li>
      <li><strong>.well-known/ai-plugin.json</strong> — ChatGPT/Claude plugin manifest</li>
      <li><strong>/api/v1</strong> literal JSON index file for static sites</li>
      <li><strong>robots.txt + sitemap.xml</strong> — agent-friendly rules</li>
      <li><strong>FAQ + schema.org JobPosting/Organization JSON-LD</strong></li>
    </ul>`
	if h.PreviewEnabled {
		deliverables = fixPreviewBlock(site)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html><head>
<title>Fix the agent-readiness score for %s</title>
<meta name="robots" content="noindex">
<meta name="viewport" content="width=device-width, initial-scale=1">
<style>
:root { --bg:#0d0d0e; --surface:#1a1a1b; --border:#2a2a2b; --text:#e0e0e0; --text-muted:#888; --accent:#d97757; }
body { font-family: 'Inter', -apple-system, sans-serif; background:var(--bg); color:var(--text); margin:0; padding:2rem 1rem; }
.wrap { max-width: 640px; margin: 0 auto; }
.card { background:var(--surface); border:1px solid var(--border); border-radius:12px; padding:2rem; }
h1 { color:var(--accent); font-size:1.5rem; margin:0 0 0.5rem; }
.host { font-family:'IBM Plex Mono',monospace; font-size:1rem; color:#aaa; margin-bottom:1rem; }
.score { font-family:'IBM Plex Mono',monospace; font-size:2rem; font-weight:700; color:var(--accent); }
ul { padding-left:1.25rem; line-height:1.8; color:#ccc; }
ul li strong { color:#fff; }
label { display:block; font-weight:600; margin: 1rem 0 0.25rem; }
input, textarea { width:100%%; box-sizing:border-box; background:var(--bg); color:var(--text); border:1px solid var(--border); border-radius:6px; padding:10px 12px; font-family:inherit; font-size:0.95rem; }
input:focus, textarea:focus { outline:none; border-color:var(--accent); }
textarea { min-height: 80px; resize: vertical; }
.hint { color:var(--text-muted); font-size:0.8rem; margin-top:0.25rem; }
.price-row { display:flex; align-items:baseline; gap:0.75rem; margin:1.5rem 0; padding:1rem; background:var(--bg); border-radius:8px; }
.price { font-size:2rem; font-weight:700; color:var(--accent); font-family:'IBM Plex Mono',monospace; }
.price-label { color:#888; font-size:0.85rem; }
.btn { display:inline-block; width:100%%; box-sizing:border-box; background:var(--accent); color:var(--bg); padding:14px 32px; border:0; border-radius:8px; font-weight:700; font-family:'IBM Plex Mono',monospace; letter-spacing:0.02em; font-size:1rem; cursor:pointer; text-align:center; }
.btn:hover { background:#e8835d; }
.back { display:inline-block; margin-top:1rem; color:#888; text-decoration:none; font-size:0.85rem; }
.back:hover { color:var(--accent); }
</style>
</head><body>
<div class="wrap">
  <div class="card">
    <h1>Fix the score for %s</h1>
    <div class="host">Currently: <span class="score">%d</span> <span style="color:#888;font-size:0.85rem;">&middot; target: 95+</span></div>
    %s
    <p style="color:#ccc;line-height:1.6;margin-top:1rem;"><strong style="color:#fff;">Two ways to fix it:</strong> get the <strong style="color:var(--accent);">$29 report instantly on-screen</strong> — the exact files &amp; fixes to apply yourself, no email or repo needed — or have us do it for $199, delivered as a pull request within 72 hours. <strong style="color:var(--accent);">$199 tier: full refund if your score doesn't hit 90+.</strong></p>
    <form method="POST" action="/fix/%s">
      <label for="email">Your email <span style="color:#888;font-weight:400;">— optional for the instant report</span></label>
      <input id="email" name="email" type="email" placeholder="you@%s">
      <div class="hint">Instant $29 report: optional (you get it on-screen right after checkout; Stripe emails your receipt). $199 done-for-you: required — it's where we send your PR link.</div>

      <label for="repo_url">Repo URL (optional)</label>
      <input id="repo_url" name="repo_url" type="url" placeholder="https://github.com/yourco/site">
      <div class="hint">Leave blank if the site is static or you'd rather email the files.</div>

      <label for="notes">Anything we should know?</label>
      <textarea id="notes" name="notes" placeholder="CMS in use, odd build pipeline, deadlines, etc."></textarea>

      <div class="price-row">
        <span class="price">$%d</span>
        <span class="price-label">self-serve report · instant download · the exact files &amp; fixes to apply yourself</span>
      </div>
      <button type="submit" name="tier" value="report" class="btn">Get the $%d report instantly &rarr;</button>
      <div class="price-row" style="margin-top:1.25rem;">
        <span class="price">$199</span>
        <span class="price-label">prefer it done for you? · one-time · 72hr turnaround · delivered as a PR</span>
      </div>
      <button type="submit" name="tier" value="managed" class="btn" style="background:transparent;border:1px solid var(--accent);color:var(--accent);">Have us do it — $199 &rarr;</button>
    </form>
    <a href="/site/%s" class="back">&larr; Back to score report</a>
  </div>
</div>
</body></html>`,
		site.Domain, site.Domain, site.AgenticScore, deliverables,
		site.Domain, site.Domain, reportPriceCents/100, reportPriceCents/100, site.Domain)
}

func (h *FixHandler) scoreFixCompletePage(w http.ResponseWriter, site *models.Site) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html><head>
<title>%s already meets the NHS score target</title>
<meta name="robots" content="noindex">
<meta name="viewport" content="width=device-width, initial-scale=1">
<style>
:root { --bg:#0d0d0e; --surface:#1a1a1b; --border:#2a2a2b; --text:#e0e0e0; --text-muted:#888; --accent:#d97757; }
body { font-family:'Inter', -apple-system, sans-serif; background:var(--bg); color:var(--text); margin:0; padding:2rem 1rem; }
.wrap { max-width:640px; margin:0 auto; }
.card { background:var(--surface); border:1px solid var(--border); border-radius:12px; padding:2rem; }
h1 { color:var(--accent); font-size:1.5rem; margin:0 0 0.5rem; }
.host { font-family:'IBM Plex Mono',monospace; font-size:1rem; color:#aaa; margin-bottom:1rem; }
.score { font-family:'IBM Plex Mono',monospace; font-size:2rem; font-weight:700; color:var(--accent); }
p { color:#ccc; line-height:1.6; }
.actions { display:flex; gap:12px; flex-wrap:wrap; margin-top:1.5rem; }
.btn { display:inline-block; background:var(--accent); color:var(--bg); padding:12px 18px; border-radius:8px; font-weight:700; font-family:'IBM Plex Mono',monospace; font-size:0.9rem; text-decoration:none; }
.btn.secondary { background:transparent; color:var(--accent); border:1px solid var(--accent); }
</style>
</head><body>
<div class="wrap">
  <div class="card">
    <h1>%s already meets the target</h1>
    <div class="host">Currently: <span class="score">%d</span> <span style="color:#888;font-size:0.85rem;">&middot; target: %d+</span></div>
    <p>This site does not need the paid score-fix implementation path. The useful next step is to monitor it so missing public agent-readiness signals are caught after future deploys.</p>
    <div class="actions">
      <a class="btn" href="/monitor?domain=%s">Monitor this score</a>
      <a class="btn secondary" href="/site/%s">Back to score report</a>
    </div>
  </div>
</div>
</body></html>`,
		site.Domain, site.Domain, site.AgenticScore, fixTargetScore,
		url.QueryEscape(site.Domain), url.PathEscape(site.Domain))
}

func (h *FixHandler) createCheckout(w http.ResponseWriter, r *http.Request, host string) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	email := strings.TrimSpace(r.FormValue("email"))
	repoURL := strings.TrimSpace(r.FormValue("repo_url"))
	notes := strings.TrimSpace(r.FormValue("notes"))
	// tier selects the product: "managed" = $199 done-for-you (default, the
	// anchor), "report" = self-serve auto-generated report (cheaper tier).
	tier := strings.TrimSpace(r.FormValue("tier"))
	if tier != "report" {
		tier = "managed"
	}
	priceCents := fixPriceCents
	if tier == "report" {
		priceCents = reportPriceCents
	}
	// Email is mandatory only for the managed (done-for-you) tier, where we
	// email the PR link. The self-serve report is delivered on-screen
	// (reportSuccessPage) and Stripe sends the receipt, so the cheap tier is
	// not gated on email — that upfront field was pure friction (D-1220).
	if tier == "managed" && (email == "" || !strings.Contains(email, "@")) {
		http.Error(w, "valid email required", http.StatusBadRequest)
		return
	}
	if email != "" && !strings.Contains(email, "@") {
		http.Error(w, "invalid email", http.StatusBadRequest)
		return
	}
	site, err := models.GetSiteByDomain(h.DB, host)
	if err != nil {
		http.Error(w, "unknown host", http.StatusNotFound)
		return
	}
	if !scoreFixEligible(site) {
		if site.HasHardAgentSignal() && site.AgenticScore >= fixTargetScore {
			http.Error(w, "score already meets target; monitor this score instead", http.StatusConflict)
			return
		}
		http.Error(w, "unknown host", http.StatusNotFound)
		return
	}

	j := &models.GeoFixJob{
		Host:       host,
		Email:      email,
		PriceCents: priceCents,
		Currency:   "usd",
		Status:     "pending",
	}
	if repoURL != "" {
		j.RepoURL = &repoURL
	}
	if notes != "" {
		j.Notes = &notes
	}
	if err := models.CreateGeoFixJob(h.DB, j); err != nil {
		log.Printf("fix: CreateGeoFixJob: %v", err)
		http.Error(w, "could not record intake", http.StatusInternalServerError)
		return
	}
	go models.LogIntentFromRequest(h.DB, r, "fix_checkout_start", "geo_fix_job", strconv.FormatInt(j.ID, 10), map[string]any{
		"host":         host,
		"email_domain": emailDomain(email),
		"repo_url":     repoURL != "",
		"source":       "fix_form",
	})

	// No Stripe key configured — fall back to lead-capture instead of faking
	// a payment. Record as status="lead" and bounce to the thank-you page.
	// Shane gets an alert and follows up manually to close via invoice.
	if gostripe.Key == "" {
		if _, err := h.DB.Exec(`UPDATE geo_fix_jobs SET status='lead', updated_at=NOW() WHERE id=$1`, j.ID); err != nil {
			log.Printf("fix: mark lead: %v", err)
		}
		notify.DiscordAsync(fmt.Sprintf("📥 **NHS fix-my-score lead** — %s · %s · $%d · %s (Stripe not configured — follow up manually)",
			host, email, priceCents/100, tier))
		http.Redirect(w, r, "/fix/success?id="+strconv.FormatInt(j.ID, 10)+"&lead=1", http.StatusSeeOther)
		return
	}

	// Tier-specific product framing. "managed" = the $199 done-for-you anchor;
	// "report" = the cheaper self-serve auto-generated report (no manual step).
	productName := "NHS Agent-Readiness Uplift"
	productDesc := fmt.Sprintf("Done-for-you GEO uplift PR for %s — target score %d+", host, fixTargetScore)
	successURL := h.BaseURL + "/fix/success?id=" + strconv.FormatInt(j.ID, 10) + "&session_id={CHECKOUT_SESSION_ID}"
	if tier == "report" {
		productName = "NHS GEO Fix Report (self-serve)"
		productDesc = fmt.Sprintf("Instant agent-readiness report for %s — the exact files + fixes to reach score %d+", host, fixTargetScore)
		successURL = h.BaseURL + "/fix/success?id=" + strconv.FormatInt(j.ID, 10) + "&tier=report&session_id={CHECKOUT_SESSION_ID}"
	}

	params := &gostripe.CheckoutSessionParams{
		LineItems: []*gostripe.CheckoutSessionLineItemParams{{
			PriceData: &gostripe.CheckoutSessionLineItemPriceDataParams{
				Currency: gostripe.String("usd"),
				ProductData: &gostripe.CheckoutSessionLineItemPriceDataProductDataParams{
					Name:        gostripe.String(productName),
					Description: gostripe.String(productDesc),
				},
				UnitAmount: gostripe.Int64(int64(priceCents)),
			},
			Quantity: gostripe.Int64(1),
		}},
		Mode:       gostripe.String(string(gostripe.CheckoutSessionModePayment)),
		SuccessURL: gostripe.String(successURL),
		CancelURL:  gostripe.String(h.BaseURL + "/fix/" + host),
		Metadata: map[string]string{
			"product":  "nhs_fix_my_score",
			"tier":     tier,
			"host":     host,
			"email":    email,
			"job_id":   strconv.FormatInt(j.ID, 10),
			"repo_url": repoURL,
		},
	}
	// Only prefill the customer email when we actually have one — passing an
	// empty string would 400 the Stripe API. Stripe collects it otherwise.
	if email != "" {
		params.CustomerEmail = gostripe.String(email)
	}
	s, err := session.New(params)
	if err != nil {
		log.Printf("fix: session.New: %v", err)
		http.Error(w, "checkout unavailable", http.StatusBadGateway)
		return
	}
	if err := models.SetGeoFixJobSession(h.DB, j.ID, s.ID); err != nil {
		log.Printf("fix: SetGeoFixJobSession: %v", err)
	}
	http.Redirect(w, r, s.URL, http.StatusSeeOther)
}

// POST /api/v1/checkout — agent-readable Stripe Checkout creation for the
// same fix-my-score product exposed at /fix/{host}.
func (h *FixHandler) AgenticCheckout(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProductID   string                 `json:"product_id"`
		ProductId   string                 `json:"productId"`
		PaymentMode string                 `json:"payment_mode"`
		SPT         string                 `json:"shared_payment_granted_token"`
		BuyerEmail  string                 `json:"buyer_email"`
		Host        string                 `json:"host"`
		Email       string                 `json:"email"`
		RepoURL     string                 `json:"repo_url"`
		Notes       string                 `json:"notes"`
		Metadata    map[string]interface{} `json:"metadata"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 32<<10)).Decode(&req); err != nil {
		writeFixJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	productID := strings.TrimSpace(req.ProductID)
	if productID == "" {
		productID = strings.TrimSpace(req.ProductId)
	}
	if productID == "" {
		productID = "nhs_geo_fix_my_score"
	}
	if productID != "nhs_geo_fix_my_score" {
		writeFixJSON(w, http.StatusNotFound, map[string]string{"error": "unknown product_id"})
		return
	}
	mode := normalizeFixPaymentMode(req.PaymentMode)
	if mode == "machine_payments_x402" {
		writeFixJSON(w, http.StatusNotImplemented, map[string]interface{}{
			"error":         "machine_payments_not_enabled",
			"payment_mode":  req.PaymentMode,
			"fallback_mode": "stripe_checkout",
			"fallback_url":  h.BaseURL + "/fix/" + strings.TrimSpace(req.Host),
			"note":          "No Stripe machine payments / x402 endpoint is deployed for Not Human Search.",
		})
		return
	}
	if mode != "stripe_checkout" && mode != "stripe_acp_spt" {
		writeFixJSON(w, http.StatusNotImplemented, map[string]interface{}{
			"error":           "unsupported_payment_mode",
			"supported_modes": []string{"stripe_checkout", "stripe_link", "link", "stripe_spt"},
			"fallback_url":    h.BaseURL + "/fix/" + strings.TrimSpace(req.Host),
		})
		return
	}
	if req.Host == "" && req.Metadata != nil {
		if value, ok := req.Metadata["host"].(string); ok {
			req.Host = value
		}
	}
	if req.Email == "" && req.Metadata != nil {
		if value, ok := req.Metadata["email"].(string); ok {
			req.Email = value
		}
	}
	if req.SPT == "" && req.Metadata != nil {
		if value, ok := req.Metadata["shared_payment_granted_token"].(string); ok {
			req.SPT = value
		}
	}
	if req.BuyerEmail == "" {
		req.BuyerEmail = req.Email
	}
	if req.BuyerEmail == "" && req.Metadata != nil {
		if value, ok := req.Metadata["buyer_email"].(string); ok {
			req.BuyerEmail = value
		}
	}
	host := strings.ToLower(strings.TrimSpace(req.Host))
	host = strings.TrimPrefix(host, "https://")
	host = strings.TrimPrefix(host, "http://")
	host = strings.Trim(host, "/")
	email := strings.TrimSpace(req.Email)
	repoURL := strings.TrimSpace(req.RepoURL)
	notes := strings.TrimSpace(req.Notes)
	if host == "" {
		writeFixJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "missing host", "required_metadata": []string{"host", "email"}})
		return
	}
	if email == "" || !strings.Contains(email, "@") {
		writeFixJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "valid email required", "required_metadata": []string{"host", "email"}})
		return
	}
	site, err := models.GetSiteByDomain(h.DB, host)
	if err != nil {
		writeFixJSON(w, http.StatusNotFound, map[string]string{"error": "unknown host"})
		return
	}
	if !scoreFixEligible(site) {
		if site.HasHardAgentSignal() && site.AgenticScore >= fixTargetScore {
			writeFixJSON(w, http.StatusConflict, map[string]interface{}{
				"error":       "score_already_meets_target",
				"score":       site.AgenticScore,
				"target":      fixTargetScore,
				"monitor_url": h.BaseURL + "/monitor?domain=" + url.QueryEscape(site.Domain),
				"site_url":    h.BaseURL + "/site/" + url.PathEscape(site.Domain),
				"note":        "Paid score-fix is for missing public agent-readiness signals, not ranking placement or score bypass.",
			})
			return
		}
		writeFixJSON(w, http.StatusNotFound, map[string]string{"error": "unknown host"})
		return
	}

	j := &models.GeoFixJob{
		Host:       host,
		Email:      email,
		PriceCents: fixPriceCents,
		Currency:   "usd",
		Status:     "pending",
	}
	if repoURL != "" {
		j.RepoURL = &repoURL
	}
	if notes != "" {
		j.Notes = &notes
	}
	if err := models.CreateGeoFixJob(h.DB, j); err != nil {
		log.Printf("fix: agentic CreateGeoFixJob: %v", err)
		writeFixJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not record intake"})
		return
	}
	go models.LogIntentFromRequest(h.DB, r, "fix_checkout_start", "geo_fix_job", strconv.FormatInt(j.ID, 10), map[string]any{
		"host":         host,
		"email_domain": emailDomain(email),
		"repo_url":     repoURL != "",
		"source":       "agentic_checkout",
		"mode":         mode,
	})
	if mode == "stripe_acp_spt" {
		if err := h.settleFixWithSPT(w, j, strings.TrimSpace(req.SPT), strings.TrimSpace(req.BuyerEmail)); err != nil {
			log.Printf("fix: spt settlement: %v", err)
		}
		return
	}

	if gostripe.Key == "" {
		if _, err := h.DB.Exec(`UPDATE geo_fix_jobs SET status='lead', updated_at=NOW() WHERE id=$1`, j.ID); err != nil {
			log.Printf("fix: mark agentic lead: %v", err)
		}
		notify.DiscordAsync(fmt.Sprintf("📥 **NHS fix-my-score lead** — %s · %s · $%d (Stripe not configured — follow up manually)",
			host, email, fixPriceCents/100))
		writeFixJSON(w, http.StatusAccepted, map[string]interface{}{
			"seller":       "nothumansearch",
			"product_id":   "nhs_geo_fix_my_score",
			"status":       "lead_recorded",
			"payment_mode": "stripe_checkout",
			"job_id":       j.ID,
			"message":      "Stripe is not configured; intake was recorded for manual invoice follow-up.",
		})
		return
	}

	params := &gostripe.CheckoutSessionParams{
		LineItems: []*gostripe.CheckoutSessionLineItemParams{{
			PriceData: &gostripe.CheckoutSessionLineItemPriceDataParams{
				Currency: gostripe.String("usd"),
				ProductData: &gostripe.CheckoutSessionLineItemPriceDataProductDataParams{
					Name:        gostripe.String("NHS Agent-Readiness Uplift"),
					Description: gostripe.String(fmt.Sprintf("Done-for-you GEO uplift PR for %s — target score 95+", host)),
				},
				UnitAmount: gostripe.Int64(int64(fixPriceCents)),
			},
			Quantity: gostripe.Int64(1),
		}},
		Mode:          gostripe.String(string(gostripe.CheckoutSessionModePayment)),
		SuccessURL:    gostripe.String(h.BaseURL + "/fix/success?id=" + strconv.FormatInt(j.ID, 10) + "&session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:     gostripe.String(h.BaseURL + "/fix/" + host),
		CustomerEmail: gostripe.String(email),
		Metadata: map[string]string{
			"product":  "nhs_fix_my_score",
			"host":     host,
			"email":    email,
			"job_id":   strconv.FormatInt(j.ID, 10),
			"repo_url": repoURL,
		},
	}
	s, err := session.New(params)
	if err != nil {
		log.Printf("fix: agentic session.New: %v", err)
		writeFixJSON(w, http.StatusBadGateway, map[string]string{"error": "checkout unavailable"})
		return
	}
	if err := models.SetGeoFixJobSession(h.DB, j.ID, s.ID); err != nil {
		log.Printf("fix: agentic SetGeoFixJobSession: %v", err)
	}
	writeFixJSON(w, http.StatusCreated, map[string]interface{}{
		"seller":            "nothumansearch",
		"product_id":        "nhs_geo_fix_my_score",
		"status":            "requires_customer_action",
		"payment_mode":      "stripe_checkout",
		"checkout_url":      s.URL,
		"stripe_session_id": s.ID,
		"job_id":            j.ID,
	})
}

func (h *FixHandler) settleFixWithSPT(w http.ResponseWriter, j *models.GeoFixJob, spt, buyerEmail string) error {
	if spt == "" || !strings.HasPrefix(spt, "spt_") {
		writeFixJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "missing shared_payment_granted_token", "payment_mode": "stripe_spt"})
		return nil
	}
	if buyerEmail == "" || !strings.Contains(buyerEmail, "@") {
		writeFixJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "valid buyer_email required", "payment_mode": "stripe_spt"})
		return nil
	}
	pi, err := confirmSharedPaymentToken(fixPriceCents, "usd", spt, "NHS Agent-Readiness Uplift", map[string]string{
		"seller":      "nothumansearch",
		"product":     "nhs_fix_my_score",
		"host":        j.Host,
		"job_id":      strconv.FormatInt(j.ID, 10),
		"buyer_email": buyerEmail,
	})
	if err != nil {
		writeFixJSON(w, http.StatusPaymentRequired, map[string]interface{}{
			"error":         "spt_settlement_failed",
			"payment_mode":  "stripe_spt",
			"message":       err.Error(),
			"fallback_mode": "stripe_checkout",
			"fallback_url":  h.BaseURL + "/fix/" + url.PathEscape(j.Host),
			"job_id":        j.ID,
		})
		return err
	}
	if err := models.SetGeoFixJobSession(h.DB, j.ID, pi.ID); err != nil {
		writeFixJSON(w, http.StatusInternalServerError, map[string]string{"error": "payment recorded at Stripe but local session update failed"})
		return err
	}
	paid, err := models.MarkGeoFixJobPaid(h.DB, pi.ID)
	if err != nil {
		writeFixJSON(w, http.StatusInternalServerError, map[string]string{"error": "payment recorded at Stripe but local paid update failed"})
		return err
	}
	notify.DiscordAsync(fmt.Sprintf("💳 **NHS fix-my-score SPT paid** — %s · %s · $%d · PI %s", paid.Host, paid.Email, fixPriceCents/100, pi.ID))
	models.LogIntentEvent(h.DB, models.IntentEvent{
		EventName:  "fix_paid",
		EntityType: "geo_fix_job",
		EntityID:   strconv.FormatInt(paid.ID, 10),
		Metadata: map[string]any{
			"host":                     paid.Host,
			"payment_mode":             "stripe_spt",
			"stripe_payment_intent_id": pi.ID,
		},
	})
	writeFixJSON(w, http.StatusCreated, map[string]interface{}{
		"seller":                   "nothumansearch",
		"product_id":               "nhs_geo_fix_my_score",
		"status":                   "paid",
		"payment_mode":             "stripe_spt",
		"stripe_payment_intent_id": pi.ID,
		"job_id":                   paid.ID,
		"message":                  "GEO uplift order paid and recorded",
	})
	return nil
}

type sptPaymentIntent struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

func confirmSharedPaymentToken(amount int, currency, spt, description string, metadata map[string]string) (*sptPaymentIntent, error) {
	key := os.Getenv("STRIPE_SPT_SECRET_KEY")
	if key == "" {
		key = os.Getenv("STRIPE_SECRET_KEY")
	}
	if key == "" {
		return nil, fmt.Errorf("STRIPE_SPT_SECRET_KEY is not configured")
	}
	values := url.Values{}
	values.Set("amount", fmt.Sprintf("%d", amount))
	values.Set("currency", strings.ToLower(currency))
	values.Set("confirm", "true")
	values.Set("payment_method_data[shared_payment_granted_token]", spt)
	if description != "" {
		values.Set("description", description)
	}
	for k, v := range metadata {
		if strings.TrimSpace(v) != "" {
			values.Set("metadata["+k+"]", v)
		}
	}
	req, err := http.NewRequest(http.MethodPost, "https://api.stripe.com/v1/payment_intents", bytes.NewBufferString(values.Encode()))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(key, "")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Stripe-Version", "2026-02-25.clover")
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 400 {
		var payload struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal(body, &payload) == nil && payload.Error.Message != "" {
			return nil, fmt.Errorf("stripe error: %s", payload.Error.Message)
		}
		return nil, fmt.Errorf("stripe error: status %d", resp.StatusCode)
	}
	var pi sptPaymentIntent
	if err := json.Unmarshal(body, &pi); err != nil {
		return nil, err
	}
	if pi.Status != "succeeded" {
		return nil, fmt.Errorf("payment_intent status %s", pi.Status)
	}
	return &pi, nil
}

// GET /fix/success — friendly thank-you page. If ?lead=1 we haven't charged
// anything yet (Stripe not wired) — word the page accordingly.
func (h *FixHandler) SuccessPage(w http.ResponseWriter, r *http.Request) {
	lead := r.URL.Query().Get("lead") == "1"
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Self-serve "report" tier: deliver the already-auto-generated per-site
	// report inline on success (no 72hr manual step). Falls through to the
	// standard message if the job/site can't be resolved.
	if !lead && r.URL.Query().Get("tier") == "report" {
		if id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64); err == nil {
			if job, err := models.GetGeoFixJob(h.DB, id); err == nil {
				if site, err := models.GetSiteByDomain(h.DB, job.Host); err == nil {
					h.reportSuccessPage(w, site)
					return
				}
			}
		}
	}

	title := "Payment received"
	body := "We'll email you the pull request link within 72 hours. Usually within 24."
	if lead {
		title = "Request received"
		body = "Thanks — we'll reach out within a business day to confirm scope and send a Stripe invoice."
	}
	fmt.Fprintf(w, `<!DOCTYPE html>
<html><head><title>%s — NHS</title>
<meta name="robots" content="noindex">
<style>
body { font-family:'Inter',-apple-system,sans-serif; background:#0d0d0e; color:#e0e0e0; display:flex; justify-content:center; align-items:center; min-height:100vh; margin:0; padding: 1rem; }
.card { background:#1a1a1b; border:1px solid #2a2a2b; border-radius:12px; padding:3rem; text-align:center; max-width:520px; }
h1 { color:#d97757; font-size:1.6rem; }
a { color:#d97757; text-decoration:none; }
.btn { display:inline-block; background:#d97757; color:#0d0d0e; padding:12px 24px; border-radius:8px; font-weight:700; margin-top:1rem; font-family:'IBM Plex Mono',monospace; }
</style></head>
<body><div class="card">
<h1>%s</h1>
<p>%s</p>
<a href="/" class="btn">Back to NHS</a>
</div></body></html>`, title, title, body)
}

// reportSuccessPage renders the paid self-serve report inline using the same
// auto-generated per-site block the preview uses (fixPreviewBlock). No manual
// fulfillment — the buyer gets the exact files + fixes immediately.
func (h *FixHandler) reportSuccessPage(w http.ResponseWriter, site *models.Site) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html><head><title>Your GEO fix report for %s — NHS</title>
<meta name="robots" content="noindex">
<meta name="viewport" content="width=device-width, initial-scale=1">
<style>
:root { --bg:#0d0d0e; --surface:#1a1a1b; --border:#2a2a2b; --text:#e0e0e0; --text-muted:#888; --accent:#d97757; }
body { font-family:'Inter',-apple-system,sans-serif; background:var(--bg); color:var(--text); margin:0; padding:2rem 1rem; }
.wrap { max-width:760px; margin:0 auto; }
.card { background:var(--surface); border:1px solid var(--border); border-radius:12px; padding:2rem; }
h1 { color:var(--accent); font-size:1.5rem; }
a { color:var(--accent); text-decoration:none; }
.btn { display:inline-block; background:var(--accent); color:#0d0d0e; padding:12px 24px; border-radius:8px; font-weight:700; margin-top:1.5rem; font-family:'IBM Plex Mono',monospace; }
pre { background:#0d0d0e; border:1px solid var(--border); border-radius:8px; padding:1rem; overflow:auto; }
</style></head>
<body><div class="wrap"><div class="card">
<h1>Payment received — your GEO fix report for %s</h1>
<p style="color:var(--text-muted);">Apply the fixes below to raise %s toward score %d+. Want it done for you instead? The $199 done-for-you tier ships these as a PR.</p>
%s
<a href="/" class="btn">Back to NHS</a>
</div></div></body></html>`,
		site.Domain, site.Domain, site.Domain, fixTargetScore, fixPreviewBlock(site))
}

// POST /webhook/stripe — Stripe events. Handles checkout.session.completed and
// nothing else for now (NHS has exactly one paid product).
// Requires STRIPE_WEBHOOK_SECRET to be set for signature verification.
func (h *FixHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	if h.WebhookSecret == "" {
		log.Printf("fix webhook: webhook secret not configured (STRIPE_WEBHOOK_SECRET missing)")
		// Return 2xx so Stripe stops retrying a webhook endpoint we can't
		// authenticate yet (NHS can run in lead-only mode without Stripe wired).
		writeJSON(w, 200, map[string]string{"ok": "true", "ignored": "webhook_not_configured"})
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 65536))
	if err != nil {
		http.Error(w, "read error", http.StatusBadRequest)
		return
	}
	event, err := webhook.ConstructEventWithOptions(body, r.Header.Get("Stripe-Signature"), h.WebhookSecret, webhook.ConstructEventOptions{
		IgnoreAPIVersionMismatch: true,
	})
	if err != nil {
		log.Printf("fix webhook: signature verification failed: %v", err)
		http.Error(w, "invalid signature", http.StatusBadRequest)
		return
	}

	if event.Type == "checkout.session.completed" {
		var cs gostripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &cs); err != nil {
			log.Printf("fix webhook: unmarshal: %v", err)
			w.WriteHeader(http.StatusOK)
			return
		}
		if cs.Metadata["product"] == "nhs_api_subscription" {
			// Robust activation: mark the human account active even if the
			// success-redirect (/api/v1/api-keys/activate, which also mints the
			// API key) was never hit. The API key is created on that redirect.
			email := cs.Metadata["email"]
			if email == "" {
				email = cs.CustomerEmail
			}
			customerID := ""
			if cs.Customer != nil {
				customerID = cs.Customer.ID
			}
			subID := ""
			if cs.Subscription != nil {
				subID = cs.Subscription.ID
			}
			if email != "" {
				if _, err := models.SetAccountSubscription(h.DB, email, customerID, subID, cs.Metadata["plan"], "active"); err != nil {
					log.Printf("fix webhook: account activate: %v", err)
				}
			}
			w.WriteHeader(http.StatusOK)
			return
		}
		// Only handle sessions tagged as nhs_fix_my_score.
		if cs.Metadata["product"] != "nhs_fix_my_score" {
			w.WriteHeader(http.StatusOK)
			return
		}
		j, err := models.MarkGeoFixJobPaid(h.DB, cs.ID)
		if err != nil {
			log.Printf("fix webhook: MarkGeoFixJobPaid: %v", err)
			w.WriteHeader(http.StatusOK)
			return
		}
		email := cs.Metadata["email"]
		host := cs.Metadata["host"]
		repoURL := cs.Metadata["repo_url"]
		amount := float64(cs.AmountTotal) / 100.0
		msg := fmt.Sprintf("💰 **NHS fix-my-score paid** — %s · %s · $%.2f · job #%d",
			host, email, amount, j.ID)
		if repoURL != "" {
			msg += " · repo " + repoURL
		}
		notify.DiscordAsync(msg)
		models.LogIntentEvent(h.DB, models.IntentEvent{
			EventName:  "fix_paid",
			EntityType: "geo_fix_job",
			EntityID:   strconv.FormatInt(j.ID, 10),
			Metadata: map[string]any{
				"host":              host,
				"email_domain":      emailDomain(email),
				"stripe_session_id": cs.ID,
				"amount_cents":      cs.AmountTotal,
				"currency":          string(cs.Currency),
			},
		})
	} else if event.Type == "customer.subscription.updated" || event.Type == "customer.subscription.deleted" {
		var sub gostripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			log.Printf("fix webhook: subscription unmarshal: %v", err)
			w.WriteHeader(http.StatusOK)
			return
		}
		if sub.Metadata["product"] != "nhs_api_subscription" {
			w.WriteHeader(http.StatusOK)
			return
		}
		customerID := ""
		if sub.Customer != nil {
			customerID = sub.Customer.ID
		}
		if err := models.UpsertSubscriptionStatus(h.DB, customerID, sub.ID, string(sub.Status), sub.Metadata["plan"]); err != nil {
			log.Printf("fix webhook: subscription status: %v", err)
		}
		// Keep the human account in lockstep with the API key: a cancellation or
		// lapse here flips the website session's entitlement off too.
		if email := sub.Metadata["email"]; email != "" {
			if _, err := models.SetAccountSubscription(h.DB, email, customerID, sub.ID, sub.Metadata["plan"], string(sub.Status)); err != nil {
				log.Printf("fix webhook: account status: %v", err)
			}
		} else if event.Type == "customer.subscription.deleted" {
			_ = models.DeactivateAccountBySubscription(h.DB, sub.ID)
		}
	}

	w.WriteHeader(http.StatusOK)
}

// GET /api/v1/admin/geo-jobs — bearer auth, same pattern as TrafficAnalytics.
func (h *FixHandler) AdminList(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	limit := 100
	if q := r.URL.Query().Get("limit"); q != "" {
		if n, err := strconv.Atoi(q); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}
	jobs, err := models.ListGeoFixJobs(h.DB, limit)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "query failed"})
		return
	}
	writeJSON(w, 200, map[string]interface{}{"jobs": jobs, "count": len(jobs)})
}

func (h *FixHandler) requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	adminKey := os.Getenv("ADMIN_API_KEY")
	if adminKey == "" {
		writeJSON(w, 503, map[string]string{"error": "admin endpoint not configured"})
		return false
	}
	if r.Header.Get("Authorization") != "Bearer "+adminKey {
		writeJSON(w, 401, map[string]string{"error": "invalid admin key"})
		return false
	}
	return true
}

// POST /api/v1/admin/geo-jobs/action
func (h *FixHandler) AdminAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, 405, map[string]string{"error": "POST required"})
		return
	}
	if !h.requireAdmin(w, r) {
		return
	}
	var req struct {
		ID       int64  `json:"id"`
		Action   string `json:"action"`
		Operator string `json:"operator"`
		Source   string `json:"source"`
		Notes    string `json:"notes"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid json"})
		return
	}
	if req.ID <= 0 {
		writeJSON(w, 400, map[string]string{"error": "geo-fix job id required"})
		return
	}
	if !models.ValidGeoFixJobAdminAction(req.Action) {
		writeJSON(w, 400, map[string]string{"error": "invalid action"})
		return
	}
	if strings.TrimSpace(req.Operator) == "" {
		writeJSON(w, 400, map[string]string{"error": "operator required"})
		return
	}
	action := strings.TrimSpace(req.Action)
	operator := strings.TrimSpace(req.Operator)
	source := strings.TrimSpace(req.Source)
	if source == "" {
		source = "admin_api"
	}
	if err := models.ApplyGeoFixJobAdminAction(h.DB, req.ID, action, operator, source, req.Notes); err != nil {
		if err == sql.ErrNoRows {
			writeJSON(w, 404, map[string]string{"error": "geo-fix job not found or not eligible for this action"})
			return
		}
		log.Printf("admin geo-fix action: %v", err)
		writeJSON(w, 500, map[string]string{"error": "action failed"})
		return
	}
	writeJSON(w, 200, map[string]any{
		"ok":         true,
		"id":         req.ID,
		"action":     action,
		"operator":   operator,
		"source":     source,
		"audited_at": time.Now().UTC().Format(time.RFC3339),
	})
}
