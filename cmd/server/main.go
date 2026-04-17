package main

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/unitedideas/nothumansearch/internal/database"
	"github.com/unitedideas/nothumansearch/internal/handlers"
)

func main() {
	port := flag.String("port", "8091", "server port")
	flag.Parse()

	if p := os.Getenv("PORT"); p != "" {
		*port = p
	}

	projectRoot := os.Getenv("APP_ROOT")
	if projectRoot == "" {
		exe, err := os.Executable()
		if err == nil {
			projectRoot = filepath.Dir(exe)
		} else {
			projectRoot = "."
		}
	}

	if err := database.Connect(); err != nil {
		log.Fatalf("database: %v", err)
	}
	log.Println("connected to database")

	if err := database.RunMigrations(filepath.Join(projectRoot, "migrations")); err != nil {
		log.Printf("WARNING: migration: %v", err)
	}
	// Belt-and-braces: ensure favicon columns exist (was added in 006 migration).
	if _, err := database.DB.Exec(`ALTER TABLE sites ADD COLUMN IF NOT EXISTS has_favicon BOOLEAN DEFAULT FALSE`); err != nil {
		log.Printf("ensure has_favicon: %v", err)
	}
	if _, err := database.DB.Exec(`ALTER TABLE sites ADD COLUMN IF NOT EXISTS favicon_url TEXT DEFAULT ''`); err != nil {
		log.Printf("ensure favicon_url: %v", err)
	}

	templatesDir := filepath.Join(projectRoot, "templates")
	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "https://nothumansearch.ai"
	}

	webHandler, err := handlers.NewWebHandler(database.DB, templatesDir)
	if err != nil {
		log.Fatalf("templates: %v", err)
	}
	apiHandler := handlers.NewAPIHandler(database.DB)
	seoHandler := handlers.NewSEOHandler(database.DB, baseURL)
	monitorHandler := handlers.NewMonitorHandler(database.DB, baseURL)
	mcpHandler := handlers.NewMCPHandler(database.DB, baseURL)
	checkHandler := handlers.NewCheckHandler(database.DB)
	badgeHandler := handlers.NewBadgeHandler(database.DB)

	mux := http.NewServeMux()

	// Static
	staticDir := filepath.Join(projectRoot, "static")
	mux.Handle("/static/", http.StripPrefix("/static/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		http.FileServer(http.Dir(staticDir)).ServeHTTP(w, r)
	})))

	// IndexNow key verification
	mux.HandleFunc("/bb1637af360f471ab2a1555d45d683ea.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("bb1637af360f471ab2a1555d45d683ea"))
	})

	// Official MCP registry HTTP-based domain authentication. This public-key
	// proof lets `mcp-publisher login http --domain nothumansearch.ai` sign
	// registry publishes with the matching private key. The private key itself
	// lives in macOS Keychain (account "foundry", service
	// "nhs-mcp-registry-privkey") and is never checked in.
	mux.HandleFunc("/.well-known/mcp-registry-auth", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("v=MCPv1; k=ed25519; p=1qXOvfXi+Dim0+NN9XiDyB0pO6seHUwAiNxjUyoraZM=\n"))
	})

	// SEO / GEO
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		// Simple DB-reachability check. Returns 200 + body "ok" if Postgres
		// responds to a trivial ping within 2s, else 503. Used by Fly
		// machine checks + external uptime monitors.
		w.Header().Set("Cache-Control", "no-store")
		if err := database.DB.Ping(); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(503)
			w.Write([]byte(`{"status":"degraded","db":"unreachable"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","db":"ok"}`))
	})
	mux.HandleFunc("/robots.txt", seoHandler.Robots)
	mux.HandleFunc("/llms.txt", seoHandler.LLMsTxt)
	mux.HandleFunc("/.well-known/llms.txt", seoHandler.LLMsTxt)
	mux.HandleFunc("/.well-known/ai-plugin.json", seoHandler.AIPluginManifest)
	mux.HandleFunc("/.well-known/mcp.json", seoHandler.MCPManifest)
	mux.HandleFunc("/llms-full.txt", seoHandler.LLMsFullTxt)
	mux.HandleFunc("/openapi.yaml", seoHandler.OpenAPISpec)
	mux.HandleFunc("/sitemap.xml", seoHandler.Sitemap)
	mux.HandleFunc("/feed.xml", seoHandler.Feed)
	mux.HandleFunc("/rss.xml", seoHandler.Feed)
	mux.HandleFunc("/feed/", seoHandler.Feed) // /feed/{category}.xml

	// Web
	mux.HandleFunc("/", webHandler.HomePage)
	mux.HandleFunc("/about", webHandler.AboutPage)
	mux.HandleFunc("/score", webHandler.ScorePage)
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		// Live status dashboard — polls /health on the main three Foundry
		// products via client-side JS so each check is an independent CORS
		// request and the page loads instantly.
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=60")
		w.Write([]byte(statusHTML))
	})
	mux.HandleFunc("/guide", webHandler.GuidePage)
	mux.HandleFunc("/report", webHandler.ReportPage)
	mux.HandleFunc("/submit", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/#submit", http.StatusMovedPermanently)
	})
	mux.HandleFunc("/site/", webHandler.SitePage)
	mux.HandleFunc("/tag/", webHandler.TagPage)
	mux.HandleFunc("/mcp-servers", webHandler.MCPServersPage)
	mux.HandleFunc("/ai-tools", webHandler.AIToolsPage)
	mux.HandleFunc("/developer-apis", webHandler.DeveloperPage)
	mux.HandleFunc("/openapi-apis", webHandler.OpenAPIPage)
	mux.HandleFunc("/llms-txt-sites", webHandler.LLMsTxtPage)
	webHandler.RegisterCategoryLandings(mux)

	// Classic URL synonyms — 301 to canonical pages so agents pattern-matching
	// URLs don't hit 404s. Improves discoverability + preserves link equity.
	for _, alias := range []struct{ from, to string }{
		{"/mcp-server", "/mcp-servers"},
		{"/ai", "/ai-tools"},
		{"/tools", "/ai-tools"},
		{"/apis", "/developer-apis"},
		{"/api-directory", "/developer-apis"},
		{"/agents-directory", "/ai-tools"},
		{"/agents", "/ai-tools"},
		{"/llm", "/tag/llm"},
		{"/llms", "/tag/llm"},
		{"/openapi", "/openapi-apis"},
		{"/llms-txt", "/llms-txt-sites"},
		{"/search", "/"},
		{"/category/ai-tools", "/ai-tools"},
		{"/category/developer", "/developer-apis"},
		{"/category/data", "/data-apis"},
		{"/category/finance", "/finance-apis"},
		{"/category/productivity", "/productivity-apis"},
		{"/category/ecommerce", "/ecommerce-apis"},
		{"/category/security", "/security-apis"},
		{"/category/communication", "/communication-apis"},
		{"/category/jobs", "/jobs-apis"},
	} {
		to := alias.to
		mux.HandleFunc(alias.from, func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, to, http.StatusMovedPermanently)
		})
	}

	// API
	mux.HandleFunc("/api/v1", apiHandler.Index)
	mux.HandleFunc("/api/v1/search", apiHandler.Search)
	mux.HandleFunc("/api/v1/site/", apiHandler.GetSite)
	mux.HandleFunc("/api/v1/submit", apiHandler.SubmitSite)
	mux.HandleFunc("/api/v1/stats", apiHandler.Stats)
	mux.HandleFunc("/api/v1/categories", apiHandler.Categories)
	mux.HandleFunc("/api/v1/admin/traffic", apiHandler.TrafficAnalytics)
	mux.HandleFunc("/api/v1/admin/mcp", apiHandler.MCPAnalytics)
	mux.Handle("/api/v1/check", checkHandler)

	// Embeddable score badges: /badge/{domain}.svg
	mux.Handle("/badge/", badgeHandler)
	mux.HandleFunc("/api/v1/monitor/register", monitorHandler.Register)

	// Monitor (free feature — email alerts when a site's agentic readiness drops)
	mux.HandleFunc("/monitor", monitorHandler.LandingPage)
	mux.HandleFunc("/monitor/unsubscribe/", monitorHandler.Unsubscribe)

	// MCP server — agents connect here to search NHS as a tool.
	// GET returns a friendly info blurb; POST is JSON-RPC 2.0.
	mux.Handle("/mcp", mcpHandler)
	mux.Handle("/mcp/", mcpHandler)

	// Middleware chain: logging → domain redirect → security → CORS → gzip → handler
	handler := loggingMiddleware(domainRedirectMiddleware(securityHeadersMiddleware(corsMiddleware(gzipMiddleware(mux)))))

	log.Printf("Not Human Search starting on :%s", *port)
	srv := &http.Server{
		Addr:         ":" + *port,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server: %v", err)
	}
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (sw *statusWriter) WriteHeader(code int) {
	sw.status = code
	sw.ResponseWriter.WriteHeader(code)
}

var botPatterns = []string{
	"bot", "crawl", "spider", "slurp", "archive", "curl/", "wget",
	"python-requests", "go-http-client", "httpx", "scrapy", "fetch",
	"lighthouse", "pagespeed", "headlesschrome", "phantomjs",
	"semrush", "ahrefs", "mj12bot", "dotbot", "petalbot", "bytespider",
	"gptbot", "chatgpt", "claudebot", "anthropic", "meta-externalagent",
	"oai-searchbot", "amazonbot", "diffbot", "youbot", "duckassistbot",
	"ccbot", "firecrawl",
}

func isBotUA(ua string) bool {
	lower := strings.ToLower(ua)
	for _, p := range botPatterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/static/") {
			next.ServeHTTP(w, r)
			return
		}
		if r.UserAgent() == "Fly-HealthCheck" {
			next.ServeHTTP(w, r)
			return
		}
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(sw, r)
		dur := time.Since(start)
		log.Printf("%s %s %d %s %s", r.Method, r.URL.Path, sw.status, r.UserAgent(), dur.Round(time.Millisecond))

		if database.DB != nil {
			ip := r.RemoteAddr
			if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
				ip = strings.TrimSpace(strings.Split(fwd, ",")[0])
			}
			ip = strings.Split(ip, ":")[0]
			h := sha256.Sum256([]byte(ip))
			ipHash := hex.EncodeToString(h[:16])
			ua := r.UserAgent()
			ref := r.Referer()
			bot := isBotUA(ua)
			go database.DB.Exec(`INSERT INTO page_views (path, method, status, ip_hash, user_agent, referer, duration_ms, is_bot) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
				r.URL.Path, r.Method, sw.status, ipHash, ua, ref, dur.Milliseconds(), bot)
		}
	})
}

// domainRedirectMiddleware redirects .com → .ai and www → apex
func domainRedirectMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		// Strip port if present
		if idx := strings.LastIndex(host, ":"); idx != -1 {
			host = host[:idx]
		}
		// Redirect nothumansearch.com → nothumansearch.ai (canonical)
		// Redirect www variants → apex
		switch host {
		case "nothumansearch.com", "www.nothumansearch.com":
			target := "https://nothumansearch.ai" + r.URL.RequestURI()
			http.Redirect(w, r, target, http.StatusMovedPermanently)
			return
		case "www.nothumansearch.ai":
			target := "https://nothumansearch.ai" + r.URL.RequestURI()
			http.Redirect(w, r, target, http.StatusMovedPermanently)
			return
		}
		next.ServeHTTP(w, r)
	})
}

const statusHTML = `<!DOCTYPE html>
<html lang="en"><head>
<meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Status — Foundry Main Products</title>
<meta name="description" content="Live status for Not Human Search, AI Dev Jobs, and 8bitconcepts. DB health + uptime, updated every minute.">
<link rel="icon" type="image/svg+xml" href="/static/img/logo.svg">
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{background:#0d0d0e;color:#e8e8e9;font-family:'Inter',system-ui,sans-serif;line-height:1.7;padding:40px 20px;min-height:100vh}
.wrap{max-width:640px;margin:0 auto}
h1{font-size:28px;color:#fff;margin-bottom:8px;letter-spacing:-0.01em}
.sub{color:#8b8d91;margin-bottom:32px;font-size:14px}
.svc{display:flex;align-items:center;padding:16px 20px;background:#111214;border:1px solid rgba(255,255,255,0.07);border-radius:8px;margin-bottom:10px;gap:12px}
.dot{width:10px;height:10px;border-radius:50%;background:#555}
.dot.ok{background:#4ade80;box-shadow:0 0 8px rgba(74,222,128,0.4)}
.dot.bad{background:#f87171;box-shadow:0 0 8px rgba(248,113,113,0.4)}
.name{font-weight:600;color:#fff;flex:1}
.url{font-family:'IBM Plex Mono',ui-monospace,monospace;color:#8b8d91;font-size:13px}
.state{font-family:'IBM Plex Mono',ui-monospace,monospace;font-size:13px;color:#8b8d91;min-width:80px;text-align:right}
.state.ok{color:#4ade80}
.state.bad{color:#f87171}
a{color:#d97757;text-decoration:none}
.foot{margin-top:32px;font-size:13px;color:#8b8d91;text-align:center}
</style>
</head><body><div class="wrap">
<h1>Foundry Status</h1>
<p class="sub">Live health for Not Human Search, AI Dev Jobs, and 8bitconcepts. Auto-refreshes every 60s.</p>

<div class="svc" data-url="https://nothumansearch.ai/health">
  <span class="dot"></span>
  <div class="name">Not Human Search<br><span class="url">nothumansearch.ai/health</span></div>
  <span class="state">checking…</span>
</div>
<div class="svc" data-url="https://aidevboard.com/health">
  <span class="dot"></span>
  <div class="name">AI Dev Jobs<br><span class="url">aidevboard.com/health</span></div>
  <span class="state">checking…</span>
</div>
<div class="svc" data-url="https://8bitconcepts.com/">
  <span class="dot"></span>
  <div class="name">8bitconcepts<br><span class="url">8bitconcepts.com/</span></div>
  <span class="state">checking…</span>
</div>

<p class="foot">← <a href="/">Not Human Search</a> · Last checked: <span id="ts">—</span></p>

<script>
async function check(el){
  const url = el.dataset.url;
  const dot = el.querySelector('.dot');
  const state = el.querySelector('.state');
  try {
    const t0 = performance.now();
    const r = await fetch(url, {cache:'no-store'});
    const ms = Math.round(performance.now() - t0);
    if (r.ok) {
      dot.classList.add('ok'); dot.classList.remove('bad');
      state.textContent = r.status + ' · ' + ms + 'ms';
      state.classList.add('ok'); state.classList.remove('bad');
    } else {
      dot.classList.add('bad'); dot.classList.remove('ok');
      state.textContent = 'HTTP ' + r.status;
      state.classList.add('bad'); state.classList.remove('ok');
    }
  } catch(e) {
    dot.classList.add('bad'); dot.classList.remove('ok');
    state.textContent = 'unreachable';
    state.classList.add('bad'); state.classList.remove('ok');
  }
}
function runAll(){
  document.querySelectorAll('.svc').forEach(check);
  document.getElementById('ts').textContent = new Date().toLocaleTimeString();
}
runAll();
setInterval(runAll, 60000);
</script>
</div></body></html>`

// gzipMiddleware compresses responses if the client sent Accept-Encoding: gzip.
// Uses a sync.Pool of gzip.Writers so we don't allocate per request. Falls
// through uncompressed for Upgrade/Sec-Websocket or already-encoded paths.
var gzipWriterPool = sync.Pool{New: func() any { return gzip.NewWriter(io.Discard) }}

type gzipResponseWriter struct {
	http.ResponseWriter
	writer      io.Writer
	wroteHeader bool
}

func (g *gzipResponseWriter) WriteHeader(code int) {
	if !g.wroteHeader {
		g.Header().Del("Content-Length") // length changes after compression
		g.wroteHeader = true
	}
	g.ResponseWriter.WriteHeader(code)
}

func (g *gzipResponseWriter) Write(b []byte) (int, error) {
	if !g.wroteHeader {
		g.WriteHeader(http.StatusOK)
	}
	return g.writer.Write(b)
}

func gzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		// Skip SSE / upgrade paths
		if r.Header.Get("Upgrade") != "" || strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
			next.ServeHTTP(w, r)
			return
		}
		gz := gzipWriterPool.Get().(*gzip.Writer)
		gz.Reset(w)
		defer func() { gz.Close(); gzipWriterPool.Put(gz) }()

		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Add("Vary", "Accept-Encoding")
		next.ServeHTTP(&gzipResponseWriter{ResponseWriter: w, writer: gz}, r)
	})
}

// securityHeadersMiddleware adds standard hardening headers to every response.
// HSTS is 1yr with preload-eligible flags; NHS has been HTTPS-only since launch.
// Also adds Link header advertising agent-discovery resources (llms.txt,
// openapi.yaml, mcp manifest) so agents can find them without parsing HTML.
func securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		h.Set("Link", `</llms.txt>; rel="describedby"; type="text/plain", </openapi.yaml>; rel="alternate"; type="application/yaml", </.well-known/mcp.json>; rel="alternate"; type="application/json"`)
		next.ServeHTTP(w, r)
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(204)
			return
		}
		next.ServeHTTP(w, r)
	})
}
