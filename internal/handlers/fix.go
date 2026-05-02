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

type FixHandler struct {
	DB            *sql.DB
	BaseURL       string
	WebhookSecret string
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
		DB:            db,
		BaseURL:       baseURL,
		WebhookSecret: webhookSecret,
	}
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
				"catalog":  h.BaseURL + "/api/v1/catalog",
				"quote":    h.BaseURL + "/api/v1/quote",
				"checkout": h.BaseURL + "/api/v1/checkout",
			},
		},
		"products": []map[string]interface{}{h.fixProduct()},
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

func (h *FixHandler) CommerceCatalog(w http.ResponseWriter, r *http.Request) {
	writeFixJSON(w, http.StatusOK, map[string]interface{}{
		"seller":   "nothumansearch",
		"currency": "USD",
		"products": []map[string]interface{}{h.fixProduct()},
	})
}

func (h *FixHandler) CommerceQuote(w http.ResponseWriter, r *http.Request) {
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
    <p style="color:#ccc;line-height:1.6;">We ship the 6-file GEO uplift as a pull request against your repo:</p>
    <ul>
      <li><strong>llms.txt</strong> — agent-facing site summary with key routes</li>
      <li><strong>openapi.yaml</strong> — public endpoints described in OpenAPI 3</li>
      <li><strong>.well-known/ai-plugin.json</strong> — ChatGPT/Claude plugin manifest</li>
      <li><strong>/api/v1</strong> literal JSON index file for static sites</li>
      <li><strong>robots.txt + sitemap.xml</strong> — agent-friendly rules</li>
      <li><strong>FAQ + schema.org JobPosting/Organization JSON-LD</strong></li>
    </ul>
    <p style="color:#ccc;line-height:1.6;">Turnaround: under 72 hours. If your score doesn't hit 90+ after merge, full refund.</p>
    <form method="POST" action="/fix/%s">
      <label for="email">Your email</label>
      <input id="email" name="email" type="email" required placeholder="you@%s">
      <div class="hint">Where we send the PR link and follow-up.</div>

      <label for="repo_url">Repo URL (optional)</label>
      <input id="repo_url" name="repo_url" type="url" placeholder="https://github.com/yourco/site">
      <div class="hint">Leave blank if the site is static or you'd rather email the files.</div>

      <label for="notes">Anything we should know?</label>
      <textarea id="notes" name="notes" placeholder="CMS in use, odd build pipeline, deadlines, etc."></textarea>

      <div class="price-row">
        <span class="price">$199</span>
        <span class="price-label">flat · one-time · 72hr turnaround</span>
      </div>
      <button type="submit" class="btn">Pay $199 &rarr;</button>
    </form>
    <a href="/site/%s" class="back">&larr; Back to score report</a>
  </div>
</div>
</body></html>`,
		site.Domain, site.Domain, site.AgenticScore,
		site.Domain, site.Domain, site.Domain)
}

func (h *FixHandler) createCheckout(w http.ResponseWriter, r *http.Request, host string) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	email := strings.TrimSpace(r.FormValue("email"))
	repoURL := strings.TrimSpace(r.FormValue("repo_url"))
	notes := strings.TrimSpace(r.FormValue("notes"))
	if email == "" || !strings.Contains(email, "@") {
		http.Error(w, "valid email required", http.StatusBadRequest)
		return
	}
	if _, err := models.GetSiteByDomain(h.DB, host); err != nil {
		http.Error(w, "unknown host", http.StatusNotFound)
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
		log.Printf("fix: CreateGeoFixJob: %v", err)
		http.Error(w, "could not record intake", http.StatusInternalServerError)
		return
	}

	// No Stripe key configured — fall back to lead-capture instead of faking
	// a payment. Record as status="lead" and bounce to the thank-you page.
	// Shane gets an alert and follows up manually to close via invoice.
	if gostripe.Key == "" {
		if _, err := h.DB.Exec(`UPDATE geo_fix_jobs SET status='lead', updated_at=NOW() WHERE id=$1`, j.ID); err != nil {
			log.Printf("fix: mark lead: %v", err)
		}
		notify.DiscordAsync(fmt.Sprintf("📥 **NHS fix-my-score lead** — %s · %s · $%d (Stripe not configured — follow up manually)",
			host, email, fixPriceCents/100))
		http.Redirect(w, r, "/fix/success?id="+strconv.FormatInt(j.ID, 10)+"&lead=1", http.StatusSeeOther)
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
	if _, err := models.GetSiteByDomain(h.DB, host); err != nil {
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

// POST /webhook/stripe — Stripe events. Handles checkout.session.completed and
// nothing else for now (NHS has exactly one paid product).
// Requires STRIPE_WEBHOOK_SECRET to be set for signature verification.
func (h *FixHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	if h.WebhookSecret == "" {
		log.Printf("fix webhook: webhook secret not configured (STRIPE_WEBHOOK_SECRET missing)")
		http.Error(w, "webhook not configured", http.StatusInternalServerError)
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
	}

	w.WriteHeader(http.StatusOK)
}

// GET /api/v1/admin/geo-jobs — bearer auth, same pattern as TrafficAnalytics.
func (h *FixHandler) AdminList(w http.ResponseWriter, r *http.Request) {
	adminKey := os.Getenv("ADMIN_API_KEY")
	if adminKey == "" {
		writeJSON(w, 503, map[string]string{"error": "admin endpoint not configured"})
		return
	}
	if r.Header.Get("Authorization") != "Bearer "+adminKey {
		writeJSON(w, 401, map[string]string{"error": "invalid admin key"})
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
