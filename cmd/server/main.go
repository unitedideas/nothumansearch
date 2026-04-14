package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
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
		baseURL = "https://nothumansearch.fly.dev"
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
	mux.HandleFunc("/site/", webHandler.SitePage)

	// API
	mux.HandleFunc("/api/v1/search", apiHandler.Search)
	mux.HandleFunc("/api/v1/site/", apiHandler.GetSite)
	mux.HandleFunc("/api/v1/submit", apiHandler.SubmitSite)
	mux.HandleFunc("/api/v1/stats", apiHandler.Stats)

	// CORS middleware
	handler := corsMiddleware(mux)

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
