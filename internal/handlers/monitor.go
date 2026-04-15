package handlers

import (
	"database/sql"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"strings"

	"github.com/unitedideas/nothumansearch/internal/models"
)

type MonitorHandler struct {
	DB      *sql.DB
	BaseURL string
}

func NewMonitorHandler(db *sql.DB, baseURL string) *MonitorHandler {
	return &MonitorHandler{DB: db, BaseURL: baseURL}
}

// POST /api/v1/monitor/register  {"email": "...", "domain": "..."}
func (h *MonitorHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeJSON(w, 405, map[string]string{"error": "POST required"})
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
input,button{font:inherit;padding:.6rem .8rem;border-radius:6px;border:1px solid #333;background:#151518;color:inherit;width:100%;box-sizing:border-box;margin-bottom:.5rem}
button{background:#d97757;border-color:#d97757;color:#0d0d0e;font-weight:600;cursor:pointer}
button:hover{background:#e0845f}
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
  <label>Your email<br><input type="email" name="email" required placeholder="you@example.com"></label>
  <label>Site to watch<br><input type="text" name="domain" required placeholder="example.com"></label>
  <button type="submit">Start monitoring</button>
</form>
<div id="msg"></div>
<p style="margin-top:2rem"><a href="/">&larr; Back to Not Human Search</a></p>

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
