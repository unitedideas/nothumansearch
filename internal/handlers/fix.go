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
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

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
		PaymentMethodTypes: gostripe.StringSlice([]string{"card"}),
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
