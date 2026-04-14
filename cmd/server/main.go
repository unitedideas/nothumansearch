package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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

	// SEO / GEO
	mux.HandleFunc("/robots.txt", seoHandler.Robots)
	mux.HandleFunc("/llms.txt", seoHandler.LLMsTxt)
	mux.HandleFunc("/.well-known/llms.txt", seoHandler.LLMsTxt)
	mux.HandleFunc("/.well-known/ai-plugin.json", seoHandler.AIPluginManifest)
	mux.HandleFunc("/openapi.yaml", seoHandler.OpenAPISpec)
	mux.HandleFunc("/sitemap.xml", seoHandler.Sitemap)

	// Web
	mux.HandleFunc("/", webHandler.HomePage)
	mux.HandleFunc("/about", webHandler.AboutPage)
	mux.HandleFunc("/submit", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/#submit", http.StatusMovedPermanently)
	})
	mux.HandleFunc("/site/", webHandler.SitePage)

	// API
	mux.HandleFunc("/api/v1/search", apiHandler.Search)
	mux.HandleFunc("/api/v1/site/", apiHandler.GetSite)
	mux.HandleFunc("/api/v1/submit", apiHandler.SubmitSite)
	mux.HandleFunc("/api/v1/stats", apiHandler.Stats)
	mux.HandleFunc("/api/v1/categories", apiHandler.Categories)

	// Middleware chain: logging → domain redirect → CORS → handler
	handler := loggingMiddleware(domainRedirectMiddleware(corsMiddleware(mux)))

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
