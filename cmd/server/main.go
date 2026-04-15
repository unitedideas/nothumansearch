package main

import (
	"compress/gzip"
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
	mux.HandleFunc("/guide", webHandler.GuidePage)
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

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip static assets and Fly health checks from logging
		if strings.HasPrefix(r.URL.Path, "/static/") {
			next.ServeHTTP(w, r)
			return
		}
		if r.UserAgent() == "Fly-HealthCheck" {
			next.ServeHTTP(w, r)
			return
		}
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s %s", r.Method, r.URL.RequestURI(), r.UserAgent(), time.Since(start).Round(time.Millisecond))
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
func securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
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
