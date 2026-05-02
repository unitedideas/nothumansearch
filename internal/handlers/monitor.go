package handlers

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/unitedideas/nothumansearch/internal/models"
)

type MonitorHandler struct {
	DB      *sql.DB
	BaseURL string

	// Per-IP rate limiter for Register. Prevents a single attacker from
	// flooding DB writes with unique (email, domain) pairs. Flagged in the
	// code-reviewer expertise as unbounded-write risk.
	rlMu      sync.Mutex
	rlCounts  map[string]int
	rlResetAt time.Time
}

const (
	monitorRegisterLimit  = 5 // per IP per hour
	monitorRegisterWindow = time.Hour
)

func NewMonitorHandler(db *sql.DB, baseURL string) *MonitorHandler {
	return &MonitorHandler{
		DB:        db,
		BaseURL:   baseURL,
		rlCounts:  map[string]int{},
		rlResetAt: time.Now().Add(monitorRegisterWindow),
	}
}

func (h *MonitorHandler) rlAllow(ipHash string) (allowed bool, remaining int, resetUnix int64) {
	h.rlMu.Lock()
	defer h.rlMu.Unlock()
	now := time.Now()
	if now.After(h.rlResetAt) {
		h.rlCounts = map[string]int{}
		h.rlResetAt = now.Add(monitorRegisterWindow)
	}
	if h.rlCounts[ipHash] >= monitorRegisterLimit {
		return false, 0, h.rlResetAt.Unix()
	}
	h.rlCounts[ipHash]++
	remaining = monitorRegisterLimit - h.rlCounts[ipHash]
	return true, remaining, h.rlResetAt.Unix()
}

func monitorHashIP(r *http.Request) string {
	ip := r.RemoteAddr
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		ip = strings.Split(fwd, ",")[0]
	}
	ip = strings.TrimSpace(ip)
	sum := sha256.Sum256([]byte(ip))
	return hex.EncodeToString(sum[:8])
}

// POST /api/v1/monitor/register  {"email": "...", "domain": "..."}
func (h *MonitorHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeJSON(w, 405, map[string]string{"error": "POST required"})
		return
	}

	ipHash := monitorHashIP(r)
	allowed, remaining, resetUnix := h.rlAllow(ipHash)
	w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", monitorRegisterLimit))
	w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
	w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetUnix))
	if !allowed {
		w.Header().Set("Retry-After", fmt.Sprintf("%d", int(time.Until(time.Unix(resetUnix, 0)).Seconds())+1))
		writeJSON(w, 429, map[string]any{
			"error":     "rate limit exceeded: 5 monitor registrations per hour per IP",
			"retry_sec": int(time.Until(time.Unix(resetUnix, 0)).Seconds()) + 1,
		})
		return
	}

	var req struct {
		Email  string `json:"email"`
		Domain string `json:"domain"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid json"})
		return
	}

	m, err := models.RegisterMonitor(h.DB, req.Email, req.Domain)
	if err != nil {
		switch err {
		case models.ErrInvalidEmail:
			writeJSON(w, 400, map[string]string{"error": "invalid email"})
		case models.ErrInvalidDomain:
			writeJSON(w, 400, map[string]string{"error": "invalid or unsupported domain"})
		case models.ErrTooManyMonitors:
			writeJSON(w, 429, map[string]string{"error": "too many monitors for this email"})
		default:
			log.Printf("monitor register: %v", err)
			writeJSON(w, 500, map[string]string{"error": "registration failed"})
		}
		return
	}

	writeJSON(w, 201, map[string]interface{}{
		"ok":              true,
		"domain":          m.Domain,
		"unsubscribe_url": h.BaseURL + "/monitor/unsubscribe/" + m.Token,
	})
}

// GET /monitor/unsubscribe/{token}
func (h *MonitorHandler) Unsubscribe(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimPrefix(r.URL.Path, "/monitor/unsubscribe/")
	if token == "" || strings.Contains(token, "/") {
		http.Error(w, "bad token", http.StatusBadRequest)
		return
	}
	// Fetch first so we can tell the user what we removed (404 vs 200).
	m, err := models.GetMonitorByToken(h.DB, token)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err := models.DeleteMonitorByToken(h.DB, token); err != nil {
		http.Error(w, "delete failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := unsubTmpl.Execute(w, map[string]string{"Domain": m.Domain, "Email": m.Email}); err != nil {
		log.Printf("unsub template: %v", err)
	}
}

// GET /monitor — landing page with signup form.
func (h *MonitorHandler) LandingPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := landingTmpl.Execute(w, nil); err != nil {
		log.Printf("landing template: %v", err)
	}
}

// writeJSON is a package-local helper matching api.go's pattern without
// coupling the two handlers.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

var unsubTmpl = template.Must(template.New("unsub").Parse(`<!doctype html>
<html><head><meta charset="utf-8"><title>Unsubscribed</title>
<style>body{font-family:ui-sans-serif,system-ui,sans-serif;max-width:40rem;margin:4rem auto;padding:0 1rem;background:#0d0d0e;color:#e8e6e3}
a{color:#d97757}</style></head>
<body><h1>Unsubscribed</h1>
<p>We&rsquo;ll stop watching <strong>{{.Domain}}</strong> for <strong>{{.Email}}</strong>.</p>
<p><a href="/">Back to Not Human Search</a></p></body></html>`))

var landingTmpl = template.Must(template.New("landing").Parse(`<!doctype html>
<html><head><meta charset="utf-8"><title>Monitor your agentic readiness — Not Human Search</title>
<meta name="description" content="Get an email when your site breaks for AI agents. Free monitoring of llms.txt, OpenAPI, ai-plugin, and MCP signals.">
<style>body{font-family:ui-sans-serif,system-ui,sans-serif;max-width:40rem;margin:4rem auto;padding:0 1rem;background:#0d0d0e;color:#e8e6e3}
h1{font-size:2rem;margin-bottom:.25rem}
.sub{color:#999;margin-bottom:2rem}
label{display:block;font-weight:600;margin:1rem 0 .35rem}
input,button{font:inherit;padding:.6rem .8rem;border-radius:6px;border:1px solid #333;background:#151518;color:inherit;width:100%;box-sizing:border-box;margin-bottom:.35rem}
button{background:#d97757;border-color:#d97757;color:#0d0d0e;font-weight:600;cursor:pointer}
button:hover{background:#e0845f}
.help{display:block;color:#999;font-size:.85rem;line-height:1.45;margin:0 0 .25rem}
.ok{color:#7fd97f;margin-top:1rem}
.err{color:#ff7777;margin-top:1rem}
ul{color:#aaa;line-height:1.7}
a{color:#d97757}</style></head>
<body>
<h1>Monitor your agentic readiness</h1>
<p class="sub">Get an email when your site breaks for AI agents &mdash; free.</p>

<p>We&rsquo;ll check your domain weekly for:</p>
<ul>
  <li><code>llms.txt</code> still present and well-formed</li>
  <li><code>ai-plugin.json</code> still valid</li>
  <li><code>OpenAPI</code> spec still parses with non-empty paths</li>
  <li>MCP server still reachable at <code>/.well-known/mcp.json</code></li>
  <li>Overall agentic-readiness score hasn&rsquo;t dropped</li>
</ul>

<form id="f">
  <label for="monitor-email">Email for alerts</label>
  <input id="monitor-email" type="email" name="email" required placeholder="you@example.com" aria-describedby="monitor-email-help">
  <span class="help" id="monitor-email-help">Used only for readiness-regression alerts and unsubscribe links.</span>
  <label for="monitor-domain">Domain to watch</label>
  <input id="monitor-domain" type="text" name="domain" required placeholder="example.com" aria-describedby="monitor-domain-help">
  <span class="help" id="monitor-domain-help">Enter the root domain without a path. NHS checks the standard agent-readiness files weekly.</span>
  <button type="submit">Start monitoring</button>
</form>
<div id="msg"></div>
<p style="margin-top:2rem"><a href="/">&larr; Back to Not Human Search</a></p>

<div style="margin-top:2.5rem;padding-top:1rem;border-top:1px solid #333;font-size:13px;color:#aaa">
  <strong style="color:#d97757">AI engineering research</strong> (live data, refreshed weekly):
  <a href="https://8bitconcepts.com/research/q2-2026-ai-hiring-snapshot.html?utm_source=nothumansearch&amp;utm_medium=footer" target="_blank" rel="noopener">Hiring snapshot</a> &middot;
  <a href="https://8bitconcepts.com/research/q2-2026-ai-compensation-by-skill.html?utm_source=nothumansearch&amp;utm_medium=footer" target="_blank" rel="noopener">Compensation by skill</a> &middot;
  <a href="https://8bitconcepts.com/research/q2-2026-mcp-ecosystem-health.html?utm_source=nothumansearch&amp;utm_medium=footer" target="_blank" rel="noopener">MCP ecosystem</a> &middot;
  <a href="https://8bitconcepts.com/research/q2-2026-remote-vs-onsite-ai-hiring.html?utm_source=nothumansearch&amp;utm_medium=footer" target="_blank" rel="noopener">Remote vs onsite</a> &middot;
  <a href="https://8bitconcepts.com/research/q2-2026-entry-level-ai-gap.html?utm_source=nothumansearch&amp;utm_medium=footer" target="_blank" rel="noopener">Entry-level gap</a>
</div>

<script>
document.getElementById('f').addEventListener('submit', async (e) => {
  e.preventDefault();
  const fd = new FormData(e.target);
  const r = await fetch('/api/v1/monitor/register', {
    method: 'POST', headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({email: fd.get('email'), domain: fd.get('domain')})
  });
  const j = await r.json();
  const msg = document.getElementById('msg');
  if (r.ok) {
    msg.className = 'ok';
    msg.textContent = "Watching " + j.domain + ". You'll get an email if anything breaks.";
    e.target.reset();
  } else {
    msg.className = 'err';
    msg.textContent = j.error || 'Registration failed.';
  }
});
</script>
</body></html>`))
