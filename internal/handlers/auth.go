package handlers

import (
	"database/sql"
	"fmt"
	"html"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/unitedideas/nothumansearch/internal/email"
	"github.com/unitedideas/nothumansearch/internal/models"
)

const sessionCookieName = "nhs_session"

// AuthService handles human (magic-link) login + sessions and the entitlement
// check shared by the website search and the REST search API. Bots authenticate
// with API keys; humans authenticate with a session cookie. Both an active
// session and an active API key flow from the same $9.99/mo subscription.
type AuthService struct {
	DB      *sql.DB
	BaseURL string
}

func NewAuthService(db *sql.DB, baseURL string) *AuthService {
	return &AuthService{DB: db, BaseURL: baseURL}
}

// CurrentAccount resolves the logged-in human from the session cookie, or nil.
func (a *AuthService) CurrentAccount(r *http.Request) *models.Account {
	if a == nil || a.DB == nil {
		return nil
	}
	c, err := r.Cookie(sessionCookieName)
	if err != nil || c.Value == "" {
		return nil
	}
	acct, err := models.ResolveSession(a.DB, c.Value)
	if err != nil {
		return nil
	}
	return acct
}

// SearchEntitled is retained only for legacy account/API-key surfaces. Core
// website, REST, and MCP discovery no longer calls it or varies results by it.
func (a *AuthService) SearchEntitled(r *http.Request) (bool, string, *models.APIKey) {
	if a == nil {
		return false, "", nil
	}
	if acct := a.CurrentAccount(r); acct.Active() {
		return true, "account", nil
	}
	if raw := extractAPIKey(r); raw != "" && a.DB != nil {
		if key, err := models.ResolveAPIKey(a.DB, raw); err == nil && key != nil {
			return true, "api_key", key
		}
	}
	return false, "", nil
}

// MeterKey is retained for legacy gated routes. Core discovery uses atomic
// priority reservations with free fallback instead of this historical meter.
func (a *AuthService) MeterKey(r *http.Request, key *models.APIKey, surface, path string) (overCap bool) {
	if a == nil || a.DB == nil || key == nil {
		return false
	}
	anon := models.HashAnonymousID(clientIP(r))
	used, _ := models.CurrentMonthUsage(a.DB, key, anon)
	if used+1 > key.MonthlyLimit {
		_ = models.RecordUsageEvent(a.DB, key, anon, surface, r.Method, path, "", 0, http.StatusPaymentRequired, r.UserAgent())
		return true
	}
	_ = models.RecordUsageEvent(a.DB, key, anon, surface, r.Method, path, "", 1, http.StatusOK, r.UserAgent())
	return false
}

func (a *AuthService) setSessionCookie(w http.ResponseWriter, raw string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    raw,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(30 * 24 * time.Hour),
		MaxAge:   30 * 24 * 60 * 60,
	})
}

func (a *AuthService) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: "", Path: "/", HttpOnly: true, MaxAge: -1})
}

func requestIsHTTPS(r *http.Request) bool {
	return r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

// StartSession logs a (just-paid) human in immediately by minting a session.
func (a *AuthService) StartSession(w http.ResponseWriter, r *http.Request, email string) error {
	acct, err := models.EnsureAccount(a.DB, email)
	if err != nil {
		return err
	}
	raw, err := models.CreateSession(a.DB, acct.ID)
	if err != nil {
		return err
	}
	a.setSessionCookie(w, raw, requestIsHTTPS(r))
	return nil
}

// Login serves the email form (GET) and issues a magic link (POST), both at /login.
func (a *AuthService) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		a.requestLogin(w, r)
		return
	}
	a.renderLogin(w, "", "")
}

func (a *AuthService) requestLogin(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	em := strings.ToLower(strings.TrimSpace(r.Form.Get("email")))
	if em == "" || !strings.Contains(em, "@") {
		a.renderLogin(w, "Enter a valid email address.", "")
		return
	}
	raw, err := models.CreateMagicLink(a.DB, em)
	if err != nil {
		log.Printf("auth: create magic link failed: %v", err)
		a.renderLogin(w, "Could not create a login link. Try again.", "")
		return
	}
	a.sendMagicLink(em, a.BaseURL+"/auth/verify?token="+raw)
	a.renderLogin(w, "", em)
}

func (a *AuthService) sendMagicLink(to, link string) {
	client, err := email.NewClientFromEnv()
	if err != nil {
		// No email transport configured (e.g. local dev) — log so the link is usable.
		log.Printf("auth: email unavailable; magic link for %s: %s", to, link)
		return
	}
	subject := "Your Not Human Search sign-in link"
	body := fmt.Sprintf(`<p>Click to sign in to Not Human Search:</p>`+
		`<p><a href="%s">Sign in to Not Human Search</a></p>`+
		`<p style="color:#888;font-size:13px;">This link expires in 20 minutes. If you didn't request it, ignore this email.</p>`,
		html.EscapeString(link))
	text := "Sign in to Not Human Search:\n" + link + "\n\nThis link expires in 20 minutes."
	if _, err := client.Send(to, subject, body, text); err != nil {
		log.Printf("auth: send magic link to %s failed: %v", to, err)
	}
}

// Verify consumes a magic link and starts a session (GET /auth/verify?token=).
func (a *AuthService) Verify(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	em, err := models.ConsumeMagicLink(a.DB, token)
	if err != nil {
		a.renderLogin(w, "That sign-in link is invalid or expired. Request a new one.", "")
		return
	}
	if err := a.StartSession(w, r, em); err != nil {
		log.Printf("auth: start session failed: %v", err)
		a.renderLogin(w, "Sign-in failed. Try again.", "")
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Logout revokes the session and clears the cookie (GET/POST /logout).
func (a *AuthService) Logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookieName); err == nil && c.Value != "" {
		_ = models.DeleteSession(a.DB, c.Value)
	}
	a.clearSessionCookie(w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *AuthService) renderLogin(w http.ResponseWriter, errMsg, sentTo string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var inner string
	if sentTo != "" {
		inner = fmt.Sprintf(`<h1>Check your email</h1><p class="muted">We sent a sign-in link to <strong>%s</strong>. It expires in 20 minutes.</p>`, html.EscapeString(sentTo))
	} else {
		e := ""
		if errMsg != "" {
			e = fmt.Sprintf(`<p class="err">%s</p>`, html.EscapeString(errMsg))
		}
		inner = `<h1>Sign in</h1><p class="muted">Enter your email and we'll send you a sign-in link. Need higher API throughput? ` +
			`<a href="/subscribe">Get a priority key for $9.99/mo</a>.</p>` + e +
			`<form method="POST" action="/login"><input type="email" name="email" placeholder="you@example.com" required autofocus>` +
			`<button type="submit">Email me a link</button></form>`
	}
	fmt.Fprint(w, authPage("Sign in — Not Human Search", inner))
}

// SubscribePage is the human-facing checkout entry: collect email, create a Stripe
// subscription Checkout session via the API, and redirect to Stripe.
func (a *AuthService) SubscribePage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	prefill := html.EscapeString(strings.TrimSpace(r.URL.Query().Get("email")))
	inner := `<h1>Priority API throughput — $9.99/mo</h1>` +
		`<p class="muted">Search and organic results are already free. This optional key provides 50,000 priority-throughput REST/MCP calls per month and higher hourly safety ceilings; requests fall back to the free tier after the allocation.</p>` +
		`<form id="sub"><input type="email" id="email" name="email" placeholder="you@example.com" value="` + prefill + `" required autofocus>` +
		`<button type="submit">Subscribe — $9.99/mo</button></form>` +
		`<p class="muted" id="msg"></p>` +
		`<p class="muted">Already a member? <a href="/login">Sign in</a>.</p>` +
		`<script>document.getElementById('sub').addEventListener('submit',function(e){e.preventDefault();` +
		`var b=this.querySelector('button');b.disabled=true;b.textContent='Redirecting…';` +
		`fetch('/api/v1/api-keys/subscribe',{method:'POST',headers:{'Content-Type':'application/json'},` +
		`body:JSON.stringify({email:document.getElementById('email').value,plan:'unlimited'})})` +
		`.then(function(r){return r.json()}).then(function(d){if(d.checkout_url){window.location=d.checkout_url}` +
		`else{document.getElementById('msg').textContent=d.error||'Could not start checkout.';b.disabled=false;b.textContent='Subscribe — $9.99/mo';}})` +
		`.catch(function(){document.getElementById('msg').textContent='Network error.';b.disabled=false;b.textContent='Subscribe — $9.99/mo';});});</script>`
	fmt.Fprint(w, authPage("Subscribe — Not Human Search", inner))
}

func authPage(title, inner string) string {
	return `<!DOCTYPE html><html lang="en"><head><meta charset="utf-8">` +
		`<meta name="viewport" content="width=device-width,initial-scale=1">` +
		`<title>` + html.EscapeString(title) + `</title><style>` +
		`:root{--bg:#0d0d0e;--surface:#141416;--border:#1f1f23;--text:#e0e0e0;--muted:#888;--accent:#d97757}` +
		`*{box-sizing:border-box}body{background:var(--bg);color:var(--text);font-family:Inter,system-ui,sans-serif;margin:0;` +
		`min-height:100vh;display:flex;align-items:center;justify-content:center;padding:24px}` +
		`.card{background:var(--surface);border:1px solid var(--border);border-radius:12px;padding:32px;max-width:420px;width:100%}` +
		`h1{font-size:22px;margin:0 0 8px}.muted{color:var(--muted);font-size:14px;line-height:1.5}` +
		`.err{color:#f87171;font-size:14px}a{color:var(--accent);text-decoration:none}` +
		`form{margin-top:16px;display:flex;flex-direction:column;gap:10px}` +
		`input{padding:12px 14px;background:var(--bg);border:1px solid var(--border);border-radius:8px;color:var(--text);font-size:15px}` +
		`button{padding:12px 14px;background:var(--accent);color:var(--bg);border:none;border-radius:8px;font-weight:600;font-size:15px;cursor:pointer}` +
		`button:disabled{opacity:.6}</style></head><body><div class="card">` +
		`<div style="font-weight:700;margin-bottom:18px"><a href="/" style="color:var(--text)">Not Human <span style="color:var(--accent)">Search</span></a></div>` +
		inner + `</div></body></html>`
}
