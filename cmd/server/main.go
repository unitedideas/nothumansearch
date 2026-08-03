package main

import (
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/unitedideas/nothumansearch/internal/database"
	"github.com/unitedideas/nothumansearch/internal/handlers"
	"github.com/unitedideas/nothumansearch/internal/models"
)

var releaseRevision = "development"

const (
	providerExchangeModeEnv      = "NHS_PROVIDER_EXCHANGE_MODE"
	providerExchangeModePilot    = "pilot"
	providerExchangeModeDisabled = "disabled"
)

func configuredProviderExchangeMode() (string, error) {
	mode := strings.ToLower(strings.TrimSpace(os.Getenv(providerExchangeModeEnv)))
	if mode == "" {
		return "", fmt.Errorf("%s must be explicitly set to %q or %q", providerExchangeModeEnv, providerExchangeModePilot, providerExchangeModeDisabled)
	}
	switch mode {
	case providerExchangeModePilot, providerExchangeModeDisabled:
		return mode, nil
	default:
		return "", fmt.Errorf("%s must be %q or %q", providerExchangeModeEnv, providerExchangeModePilot, providerExchangeModeDisabled)
	}
}

func configuredListenAddress(port string) (string, error) {
	host := strings.TrimSpace(os.Getenv("NHS_LISTEN_HOST"))
	if host == "" {
		return ":" + port, nil
	}
	if net.ParseIP(host) == nil {
		return "", errors.New("NHS_LISTEN_HOST must be an IP address")
	}
	return net.JoinHostPort(host, port), nil
}

func providerExchangeDisabledHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Retry-After", "3600")
	w.WriteHeader(http.StatusServiceUnavailable)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": "provider exchange writes are disabled; free discovery and action-interest observation remain available",
	})
}

type databasePinger interface {
	Ping() error
}

type healthPayload struct {
	Status           string `json:"status"`
	Database         string `json:"db"`
	ReleaseRevision  string `json:"release_revision"`
	ProviderExchange string `json:"provider_exchange"`
}

func healthHandler(db databasePinger, configuredMode ...string) http.Handler {
	mode := providerExchangeModeDisabled
	if len(configuredMode) > 0 && configuredMode[0] != "" {
		mode = configuredMode[0]
	}
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		response := healthPayload{
			Status: "ok", Database: "ok", ReleaseRevision: releaseRevision,
			ProviderExchange: mode,
		}
		if db == nil || db.Ping() != nil {
			response.Status = "degraded"
			response.Database = "unreachable"
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		_ = json.NewEncoder(w).Encode(response)
	})
}

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
	providerExchangeMode, err := configuredProviderExchangeMode()
	if err != nil {
		log.Fatalf("provider exchange mode: %v", err)
	}
	// Fail before touching the database when the dedicated exchange signer is
	// absent or malformed. NewProviderExchangeHandler later verifies retained
	// persisted proof material after migrations are known current.
	if providerExchangeMode == providerExchangeModePilot {
		if err := handlers.ValidateProviderExchangeSigningConfiguration(); err != nil {
			log.Fatalf("provider exchange signing preflight: %v", err)
		}
	}

	connectContext, cancelConnect := context.WithTimeout(context.Background(), 30*time.Second)
	if err := database.ConnectWithReleaseRevisionContext(connectContext, releaseRevision); err != nil {
		cancelConnect()
		log.Fatalf("database: %v", err)
	}
	cancelConnect()
	log.Println("connected to database")
	if err := database.RunMigrationsWithPreflight(
		filepath.Join(projectRoot, "migrations"), releaseRevision,
		handlers.ProviderExchangeProtectedMigrationPreflight,
	); err != nil {
		log.Fatalf("migration: %v", err)
	}

	// Retention prune for page_views + mcp_requests. Runs hourly, deletes rows
	// older than 30 days. Prevents unbounded growth on 256MB Postgres (page_views
	// audit on 2026-04-17 showed 77k rows in 2 days — ~1M/month if unchecked).
	go func() {
		prune := func() {
			if database.DB == nil {
				return
			}
			if res, err := database.DB.Exec(`DELETE FROM page_views WHERE created_at < now() - interval '30 days'`); err != nil {
				log.Printf("page_views prune: %v", err)
			} else if n, _ := res.RowsAffected(); n > 0 {
				log.Printf("page_views prune: deleted %d rows", n)
			}
			if res, err := database.DB.Exec(`DELETE FROM mcp_requests WHERE created_at < now() - interval '30 days'`); err != nil {
				log.Printf("mcp_requests prune: %v", err)
			} else if n, _ := res.RowsAffected(); n > 0 {
				log.Printf("mcp_requests prune: deleted %d rows", n)
			}
			if res, err := database.DB.Exec(`DELETE FROM intent_events WHERE created_at < now() - interval '30 days'`); err != nil {
				log.Printf("intent_events prune: %v", err)
			} else if n, _ := res.RowsAffected(); n > 0 {
				log.Printf("intent_events prune: deleted %d rows", n)
			}
			// Legacy search_queries contains raw query text plus IP/UA fields. New
			// discovery telemetry never writes this table, but historical rows must
			// still obey the same bounded retention promise.
			if res, err := database.DB.Exec(`DELETE FROM search_queries WHERE created_at < now() - interval '30 days'`); err != nil {
				log.Printf("search_queries prune: %v", err)
			} else if n, _ := res.RowsAffected(); n > 0 {
				log.Printf("search_queries prune: deleted %d rows", n)
			}
			// Usage events support calendar-month quota accounting. Keep 35 days so
			// day-one usage remains available through a 31-day month, then delete the
			// event (including its legacy anonymous hash and user agent).
			if res, err := database.DB.Exec(`DELETE FROM usage_events WHERE created_at < now() - interval '35 days'`); err != nil {
				log.Printf("usage_events prune: %v", err)
			} else if n, _ := res.RowsAffected(); n > 0 {
				log.Printf("usage_events prune: deleted %d rows", n)
			}
			// Redact controlled action-ticket intent before deleting search receipts.
			// Commercial IDs, consent attestations, terms snapshots, and signed
			// accounting proof remain; the controlled discovery constraints do not.
			if n, err := models.RedactExpiredActionTicketIntent(database.DB); err != nil {
				log.Printf("action ticket intent redaction: %v", err)
			} else if n > 0 {
				log.Printf("action ticket intent redaction: redacted %d rows", n)
			}
			// Provider-independent action interests become ineligible for replay and
			// reporting at their explicit expiry. This hourly prune performs the
			// subsequent physical deletion before the source-receipt cascade runs.
			if res, err := database.DB.Exec(`DELETE FROM action_interest_receipts WHERE expires_at <= now()`); err != nil {
				log.Printf("action_interest_receipts prune: %v", err)
			} else if n, _ := res.RowsAffected(); n > 0 {
				log.Printf("action_interest_receipts prune: deleted %d rows", n)
			}
			// Search receipts cascade-delete returned-result and selection rows.
			// Provider products use thresholded controlled-topic aggregates rather
			// than exporting individual searches.
			if res, err := database.DB.Exec(`DELETE FROM search_receipts WHERE created_at < now() - interval '30 days'`); err != nil {
				log.Printf("search_receipts prune: %v", err)
			} else if n, _ := res.RowsAffected(); n > 0 {
				log.Printf("search_receipts prune: deleted %d rows", n)
			}
			if res, err := database.DB.Exec(`DELETE FROM submissions WHERE status IN ('failed','rejected','duplicate') AND created_at < now() - interval '30 days'`); err != nil {
				log.Printf("submissions prune: %v", err)
			} else if n, _ := res.RowsAffected(); n > 0 {
				log.Printf("submissions prune: deleted %d rows", n)
			}
			// Magic links are single-use credentials, not an analytics record. Remove
			// consumed or expired rows promptly so provider onboarding cannot create
			// an unbounded token table even under bounded email abuse.
			if res, err := database.DB.Exec(`DELETE FROM magic_links WHERE used_at IS NOT NULL OR expires_at <= NOW()`); err != nil {
				log.Printf("magic_links prune: %v", err)
			} else if n, _ := res.RowsAffected(); n > 0 {
				log.Printf("magic_links prune: deleted %d rows", n)
			}
			// Sessions are bearer credentials. Remove expired rows on the same hourly
			// retention pass so stale credentials do not accumulate indefinitely.
			if res, err := database.DB.Exec(`DELETE FROM sessions WHERE expires_at <= NOW()`); err != nil {
				log.Printf("sessions prune: %v", err)
			} else if n, _ := res.RowsAffected(); n > 0 {
				log.Printf("sessions prune: deleted %d rows", n)
			}
		}
		// Redact/delete already-expired privacy data as soon as the current schema
		// is available, then retry hourly. Read paths enforce their own time bounds;
		// this task performs the separately disclosed physical cleanup.
		prune()
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			prune()
		}
	}()
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
	apiHandler.BaseURL = baseURL
	seoHandler := handlers.NewSEOHandler(database.DB, baseURL)
	monitorHandler := handlers.NewMonitorHandler(database.DB, baseURL)
	mcpHandler := handlers.NewMCPHandler(database.DB, baseURL)
	actionInterestHandler := handlers.NewActionInterestHandler(database.DB, baseURL)
	mcpHandler.ActionInterests = actionInterestHandler
	checkHandler := handlers.NewCheckHandler(database.DB)
	badgeHandler := handlers.NewBadgeHandler(database.DB)
	digestHandler, err := handlers.NewDigestHandler(database.DB, baseURL, templatesDir)
	if err != nil {
		log.Fatalf("digest template: %v", err)
	}
	fixHandler := handlers.NewFixHandler(database.DB, baseURL)
	apiKeyHandler := handlers.NewAPIKeyHandler(database.DB, baseURL)

	// Legacy account/API-key support remains for existing customers and future
	// high-throughput plans. Core website, REST, and MCP discovery is free.
	authSvc := handlers.NewAuthService(database.DB, baseURL)
	apiHandler.Auth = authSvc
	apiKeyHandler.Auth = authSvc
	webHandler.Auth = authSvc
	var providerExchangeHandler *handlers.ProviderExchangeHandler
	var providerSettlementHandler *handlers.ProviderSettlementHandler
	if providerExchangeMode == providerExchangeModePilot {
		providerExchangeHandler, err = handlers.NewProviderExchangeHandler(database.DB, baseURL, authSvc, templatesDir)
		providerSettlementHandler = handlers.NewProviderSettlementHandler(database.DB, baseURL, fixHandler.WebhookSecret)
		fixHandler.ProviderSettlement = providerSettlementHandler
	} else {
		providerExchangeHandler, err = handlers.NewProviderExchangePageHandler(database.DB, baseURL, authSvc, templatesDir)
	}
	if err != nil {
		log.Fatalf("provider exchange init: %v", err)
	}
	apiHandler.ProviderExchangeEnabled = providerExchangeMode == providerExchangeModePilot
	mcpHandler.ProviderExchangeEnabled = providerExchangeMode == providerExchangeModePilot
	if providerExchangeMode == providerExchangeModePilot {
		mcpHandler.ProviderExchange = providerExchangeHandler
	}
	// Provider DNS ownership is a renewable proof, not a one-time badge. Each
	// instance may run this worker because PostgreSQL leases due rows with
	// SKIP LOCKED and per-acquisition lease IDs.
	if providerExchangeMode == providerExchangeModePilot {
		go func() {
			run := func() {
				ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
				defer cancel()
				stats, err := providerExchangeHandler.ReverifyDueProviderClaims(ctx, 10)
				if err != nil {
					log.Printf("provider DNS reverification: %v (leased=%d matched=%d failed=%d revoked=%d completion_errors=%d)",
						err, stats.Leased, stats.Matched, stats.Failed, stats.Revoked, stats.CompletionErrors)
					return
				}
				if stats.Leased > 0 {
					log.Printf("provider DNS reverification: leased=%d matched=%d failed=%d revoked=%d",
						stats.Leased, stats.Matched, stats.Failed, stats.Revoked)
				}
			}
			run()
			ticker := time.NewTicker(15 * time.Minute)
			defer ticker.Stop()
			for range ticker.C {
				run()
			}
		}()
	}

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

	// Google Site Verification (HTML file method for Search Console)
	// Filename from env: GOOGLE_SITE_VERIFICATION (e.g., "google1234abc")
	// File served at: /google1234abc.html
	if gsvKey := os.Getenv("GOOGLE_SITE_VERIFICATION"); gsvKey != "" {
		filePath := "/" + gsvKey + ".html"
		mux.HandleFunc(filePath, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "public, max-age=604800")
			fmt.Fprintf(w, "google-site-verification: %s\n", gsvKey)
		})
	}

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
	mux.Handle("/health", healthHandler(database.DB, providerExchangeMode))
	mux.HandleFunc("/robots.txt", seoHandler.Robots)
	mux.HandleFunc("/llms.txt", seoHandler.LLMsTxt)
	mux.HandleFunc("/llm.txt", seoHandler.LLMsTxt)
	mux.HandleFunc("/.well-known/llms.txt", seoHandler.LLMsTxt)
	mux.HandleFunc("/.well-known/llm.txt", seoHandler.LLMsTxt)
	mux.HandleFunc("/.well-known/ai-plugin.json", seoHandler.AIPluginManifest)
	mux.HandleFunc("/.well-known/mcp.json", seoHandler.MCPManifest)
	mux.HandleFunc("/.well-known/agent.json", fixHandler.AgentJSON)
	mux.HandleFunc("/.well-known/commerce.json", fixHandler.CommerceManifest)
	mux.HandleFunc("/.well-known/glama.json", seoHandler.GlamaManifest)
	mux.HandleFunc("/.well-known/security.txt", seoHandler.SecurityTxt)
	mux.HandleFunc("/security.txt", seoHandler.SecurityTxt)
	mux.HandleFunc("/llms-full.txt", seoHandler.LLMsFullTxt)
	mux.HandleFunc("/openapi.yaml", seoHandler.OpenAPISpec)
	mux.HandleFunc("/sitemap.xml", seoHandler.Sitemap)
	mux.HandleFunc("/feed.xml", seoHandler.Feed)
	mux.HandleFunc("/rss.xml", seoHandler.Feed)
	mux.HandleFunc("/feed/", seoHandler.Feed) // /feed/{category}.xml

	// Weekly MCP ecosystem digest — HTML (/digest), JSON (/digest.json), RSS (/digest.rss).
	mux.HandleFunc("/digest", digestHandler.HTMLHandler)
	mux.HandleFunc("/digest.json", digestHandler.JSONHandler)
	mux.HandleFunc("/digest.rss", digestHandler.RSSHandler)

	// Web
	mux.HandleFunc("/", webHandler.HomePage)
	mux.HandleFunc("/about", webHandler.AboutPage)
	mux.HandleFunc("/stats", webHandler.StatsPage)
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
	mux.HandleFunc("/install", func(w http.ResponseWriter, r *http.Request) {
		// Curl-pipe-bash friendly installer. Serves a shell script that wires
		// NHS MCP into the user's Claude Code install; also handles Cursor,
		// Cline, and Continue by printing the right snippet for each.
		// Usage: curl -fsSL https://nothumansearch.ai/install | bash
		w.Header().Set("Content-Type", "text/x-shellscript; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=300")
		w.Write([]byte(installScript))
	})
	mux.HandleFunc("/install.sh", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/x-shellscript; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=300")
		w.Write([]byte(installScript))
	})
	mux.HandleFunc("/mcp-servers", webHandler.MCPServersPage)
	mux.HandleFunc("/ai-tools", webHandler.AIToolsPage)
	mux.HandleFunc("/developer-apis", webHandler.DeveloperPage)
	mux.HandleFunc("/openapi-apis", webHandler.OpenAPIPage)
	mux.HandleFunc("/llms-txt-sites", webHandler.LLMsTxtPage)
	mux.HandleFunc("/top", webHandler.TopPage)
	mux.HandleFunc("/top-100", webHandler.TopPage)
	mux.HandleFunc("/leaderboard", webHandler.TopPage)
	mux.HandleFunc("/newest", webHandler.NewestPage)
	mux.HandleFunc("/latest", webHandler.NewestPage)
	mux.HandleFunc("/new", webHandler.NewestPage)
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
	mux.HandleFunc("/api/v1/catalog", fixHandler.CommerceCatalog)
	mux.HandleFunc("/api/v1/products", fixHandler.CommerceCatalog)
	mux.HandleFunc("/api/v1/quote", fixHandler.CommerceQuote)
	mux.HandleFunc("/api/v1/checkout", fixHandler.AgenticCheckout)
	mux.HandleFunc("/api/v1/api-keys/subscribe", apiKeyHandler.Subscribe)
	mux.HandleFunc("/api/v1/api-keys/activate", apiKeyHandler.Activate)
	if providerExchangeMode == providerExchangeModePilot {
		mux.HandleFunc("/api/v1/provider/claims", providerExchangeHandler.Claims)
		mux.HandleFunc("/api/v1/provider/claims/", providerExchangeHandler.ClaimAction)
		mux.HandleFunc("/api/v1/provider/offers", providerExchangeHandler.Offers)
		mux.HandleFunc("/api/v1/provider/offers/", providerExchangeHandler.OfferAction)
		mux.HandleFunc("/api/v1/provider/commercial-acceptances", providerExchangeHandler.ProviderCommercialAcceptances)
		mux.HandleFunc("/api/v1/provider/pilot-status", providerExchangeHandler.ProviderPilotStatus)
		mux.HandleFunc("/api/v1/provider/demand", providerExchangeHandler.ProviderDemand)
		mux.HandleFunc("/api/v1/provider/action-tickets/resolve", providerExchangeHandler.ResolveProviderControlledIntent)
		mux.HandleFunc("/api/v1/provider/outcomes", providerExchangeHandler.ProviderOutcomes)
		mux.HandleFunc("/api/v1/provider/receipts/", providerExchangeHandler.ProviderReceipt)
		mux.HandleFunc("/api/v1/action-tickets", providerExchangeHandler.ActionTickets)
		mux.HandleFunc("/api/v1/action-tickets/handoff", providerExchangeHandler.ActionTicketHandoff)
		mux.HandleFunc("/api/v1/action-receipts/verify", providerExchangeHandler.VerifyOutcomeReceipt)
	} else {
		for _, path := range []string{
			"/api/v1/provider/claims", "/api/v1/provider/claims/",
			"/api/v1/provider/offers", "/api/v1/provider/offers/",
			"/api/v1/provider/commercial-acceptances", "/api/v1/provider/pilot-status",
			"/api/v1/provider/demand", "/api/v1/provider/action-tickets/resolve",
			"/api/v1/provider/outcomes",
			"/api/v1/provider/receipts/", "/api/v1/action-tickets",
			"/api/v1/action-tickets/handoff", "/api/v1/action-receipts/verify",
		} {
			mux.HandleFunc(path, providerExchangeDisabledHandler)
		}
	}
	mux.HandleFunc("/api/v1/action-interests", actionInterestHandler.Record)
	mux.HandleFunc("/providers", providerExchangeHandler.ProvidersPage)
	mux.HandleFunc("/privacy", providerExchangeHandler.PrivacyPage)

	// Human auth (magic-link login + session) and the subscribe entry point.
	mux.HandleFunc("/login", authSvc.FailClosedLogin)
	mux.HandleFunc("/auth/verify", authSvc.VerifyNoStore)
	mux.HandleFunc("/logout", authSvc.Logout)
	mux.HandleFunc("/subscribe", authSvc.SubscribePage)

	// Cached discovery reads stay public. Search has an in-handler abuse throttle;
	// indexed site details are free so agents can evaluate results without a key.
	mux.HandleFunc("/api/v1/search", apiHandler.Search)
	mux.HandleFunc("/api/v1/site/", apiHandler.GetSite)
	mux.HandleFunc("/api/v1/sites/", apiHandler.GetSite)
	mux.HandleFunc("/api/v1/submit", apiHandler.SubmitSite)
	mux.HandleFunc("/api/v1/stats", apiHandler.Stats)
	mux.HandleFunc("/api/v1/top", apiHandler.Top)
	mux.HandleFunc("/api/v1/categories", apiHandler.Categories)
	mux.HandleFunc("/api/v1/verify-mcp", apiHandler.VerifyMCP)
	mux.HandleFunc("/api/v1/admin/traffic", apiHandler.TrafficAnalytics)
	mux.HandleFunc("/api/v1/admin/mcp", apiHandler.MCPAnalytics)
	mux.HandleFunc("/api/v1/admin/signals", apiHandler.SignalAnalytics)
	mux.HandleFunc("/api/v1/admin/demand", apiHandler.ProviderDemandAnalytics)
	mux.HandleFunc("/api/v1/admin/demand-stage1", actionInterestHandler.Stage1DemandProof)
	if providerExchangeMode == providerExchangeModePilot {
		mux.HandleFunc("/api/v1/admin/provider-pilot/action", providerExchangeHandler.AdminProviderPilotEpochAction)
		mux.HandleFunc("/api/v1/admin/provider-pilot/epoch", providerExchangeHandler.AdminProviderPilotEpochStatus)
		mux.HandleFunc("/api/v1/admin/provider-offers/action", providerExchangeHandler.AdminOfferAction)
		mux.HandleFunc("/api/v1/admin/provider-commercial/action", providerExchangeHandler.AdminCommercialAction)
		mux.HandleFunc("/api/v1/admin/provider-pilot-queue", providerExchangeHandler.AdminProviderPilotQueue)
		mux.HandleFunc("/api/v1/admin/provider-pilot-review", providerExchangeHandler.AdminProviderPilotReview)
		mux.HandleFunc("/api/v1/admin/provider-proof-manifest", providerExchangeHandler.AdminProviderProofManifest)
		mux.HandleFunc("/api/v1/admin/provider-settlements/checkout", providerSettlementHandler.AdminCreateCheckout)
		mux.HandleFunc("/api/v1/admin/provider-settlements/status", providerSettlementHandler.AdminStatus)
	} else {
		mux.HandleFunc("/api/v1/admin/provider-pilot/action", providerExchangeDisabledHandler)
		mux.HandleFunc("/api/v1/admin/provider-pilot/epoch", providerExchangeDisabledHandler)
		mux.HandleFunc("/api/v1/admin/provider-offers/action", providerExchangeDisabledHandler)
		mux.HandleFunc("/api/v1/admin/provider-commercial/action", providerExchangeDisabledHandler)
		mux.HandleFunc("/api/v1/admin/provider-pilot-queue", providerExchangeDisabledHandler)
		mux.HandleFunc("/api/v1/admin/provider-pilot-review", providerExchangeDisabledHandler)
		mux.HandleFunc("/api/v1/admin/provider-proof-manifest", providerExchangeDisabledHandler)
		mux.HandleFunc("/api/v1/admin/provider-settlements/checkout", providerExchangeDisabledHandler)
		mux.HandleFunc("/api/v1/admin/provider-settlements/status", providerExchangeDisabledHandler)
	}
	mux.HandleFunc("/api/v1/admin/provider-proof", providerExchangeHandler.AdminProof)
	mux.HandleFunc("/api/v1/admin/geo-jobs", fixHandler.AdminList)
	mux.HandleFunc("/api/v1/admin/geo-jobs/action", fixHandler.AdminAction)

	// Paid fix-my-score intake + Stripe. /fix/success is registered before the
	// catch-all /fix/ so it wins the match.
	mux.HandleFunc("/fix/success", fixHandler.SuccessPage)
	mux.HandleFunc("/fix/", fixHandler.ServeHTTP)
	mux.HandleFunc("/webhook/stripe", fixHandler.HandleWebhook)
	mux.Handle("/api/v1/check", checkHandler)

	// Embeddable score badges: /badge/{domain}.svg
	mux.Handle("/badge/", badgeHandler)
	mux.HandleFunc("/api/v1/monitor/register", monitorHandler.Register)
	mux.HandleFunc("/api/v1/admin/monitors", monitorHandler.AdminList)
	mux.HandleFunc("/api/v1/admin/monitors/action", monitorHandler.AdminAction)
	mux.HandleFunc("/api/v1/admin/monitors/actions", monitorHandler.AdminActionCounts)
	mux.HandleFunc("/api/v1/admin/monitors/status", monitorHandler.AdminStatusCounts)

	// Monitor (free feature — email alerts when a site's agentic readiness drops)
	mux.HandleFunc("/monitor", monitorHandler.LandingPage)
	mux.HandleFunc("/monitor/unsubscribe/", monitorHandler.Unsubscribe)

	// MCP server — agents connect here to search NHS as a tool.
	// GET returns a friendly info blurb; POST is JSON-RPC 2.0.
	mux.Handle("/mcp", mcpHandler)
	mux.Handle("/mcp/", mcpHandler)

	// Middleware chain: logging → security → domain redirect → CORS → gzip → handler
	// Security runs before canonical-host handling so even redirects and rejected
	// noncanonical capability URLs receive the response-hardening contract.
	handler := loggingMiddleware(securityHeadersMiddleware(domainRedirectMiddleware(corsMiddleware(gzipMiddleware(mux)))))

	listenAddress, err := configuredListenAddress(*port)
	if err != nil {
		log.Fatalf("listen address: %v", err)
	}
	log.Printf("Not Human Search starting on %s", listenAddress)
	srv := &http.Server{
		Addr:         listenAddress,
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
	"python-requests", "python-urllib", "go-http-client", "httpx", "scrapy", "fetch",
	"lighthouse", "pagespeed", "headlesschrome", "phantomjs",
	"semrush", "ahrefs", "mj12bot", "dotbot", "petalbot", "bytespider",
	"gptbot", "chatgpt", "claudebot", "anthropic", "meta-externalagent",
	"oai-searchbot", "amazonbot", "diffbot", "youbot", "duckassistbot",
	"ccbot", "firecrawl", "chiark", "agentdiscoveryindex", "montexi",
	"dataforseo", "wp-admin/setup-config", "wlwmanifest",
	"java/", "libwww", "lwp-trivial", "nutch", "httpie",
	// MCP clients + internal infra (added 2026-04-17 after audit showed
	// claude-code/* hammering /mcp with 20k+ is_bot=false rows bloating page_views)
	"claude-code/", "claude-desktop", "cursor-mcp", "mcp-client",
	"nhs-discovery", "consul health check", "fly-check", "fly-proxy",
	"kube-probe", "google-cloud-checks",
}

// noLogPathPrefixes are request paths we skip from page_views entirely.
// /mcp gets its own table (mcp_requests). /health is internal infra noise.
// These exclusions keep page_views focused on human/organic traffic.
var noLogPathPrefixes = []string{
	"/mcp",                     // JSON-RPC endpoint — logged to mcp_requests table, except privacy-bypassed tools
	"/api/v1/action-interests", // receipt table only; never bind interest invocation to IP/UA telemetry
	"/api/v1/action-tickets",   // ticket/handoff tables only; do not timestamp-correlate principal or agent network telemetry
	"/api/v1/provider/action-tickets/resolve", // free read-only resolver; its contract forbids analytics and identity/network collection
	"/health",         // internal Consul/Fly health checks
	"/metrics",        // Prometheus scrapers
	"/webhook/stripe", // Stripe retries/probes are monitored separately
}

// shouldLogPageView returns false for paths we explicitly exclude.
func shouldLogPageView(p string) bool {
	for _, pre := range noLogPathPrefixes {
		if p == pre || strings.HasPrefix(p, pre+"/") {
			return false
		}
	}
	return true
}

// pageViewReferer intentionally drops the caller-controlled Referer value.
// It can contain query text, credentials, or private capability URLs, while
// ordinary discovery analytics need only the bounded route and status fields.
func pageViewReferer(_ *http.Request) string {
	return ""
}

func suppressRequestLineIdentityTelemetry(p string) bool {
	for _, prefix := range []string{
		"/api/v1/action-interests",
		"/api/v1/action-tickets",
		"/api/v1/provider/action-tickets/resolve",
		"/mcp",
	} {
		if p == prefix || strings.HasPrefix(p, prefix+"/") {
			return true
		}
	}
	return false
}

func isBotUA(ua string) bool {
	lower := strings.ToLower(ua)
	if lower == "node" || lower == "" {
		return true
	}
	for _, p := range botPatterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

// scannerPathPrefixes are URL prefixes that indicate a CMS/vuln scanner.
// NHS is Go — no WordPress, no PHP, no .NET. Any request here is a bot.
var scannerPathPrefixes = []string{
	"/wp-admin", "/wp-login", "/wp-content", "/wp-includes", "/wp-json",
	"/wordpress", "/xmlrpc.php", "/phpmyadmin", "/pma", "/myadmin",
	"/admin.php", "/setup-config.php",
	"/.env", "/.git", "/.DS_Store", "/.aws", "/.ssh",
	"/old", "/old2", "/backup", "/backups", "/bak",
	"/test", "/tests", "/staging", "/stage",
	"/cms", "/blog/wp-admin", "/site/wp-admin",
}

// scannerPathSuffixes are file extensions that don't exist on NHS's Go stack.
var scannerPathSuffixes = []string{
	".php", ".php7", ".php5", ".phtml",
	".asp", ".aspx", ".ashx", ".asmx",
	".jsp", ".jspx", ".do", ".action",
	".cgi", ".pl",
}

// scannerSubstrings catches nested probes like /blog/old/wp-admin/setup-config.php.
var scannerSubstrings = []string{
	"wp-admin", "wp-login", "wp-content", "wp-includes", "xmlrpc",
	"phpmyadmin", "wlwmanifest", "/.env", "/.git/",
	"setup-config", "admin-ajax",
}

// isScannerPath returns true if the request path looks like a CMS/vuln probe.
func isScannerPath(p string) bool {
	lp := strings.ToLower(p)
	for _, pre := range scannerPathPrefixes {
		if lp == pre || strings.HasPrefix(lp, pre+"/") || strings.HasPrefix(lp, pre+".") {
			return true
		}
	}
	for _, suf := range scannerPathSuffixes {
		if strings.HasSuffix(lp, suf) {
			return true
		}
	}
	for _, sub := range scannerSubstrings {
		if strings.Contains(lp, sub) {
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

		// Short-circuit scanner probes with 410 Gone. Log as is_bot=true so
		// they never pollute the human-traffic report.
		if isScannerPath(r.URL.Path) {
			start := time.Now()
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.Header().Set("Cache-Control", "public, max-age=86400")
			w.WriteHeader(http.StatusGone)
			w.Write([]byte("410 Gone — this path does not exist on nothumansearch.ai\n"))
			dur := time.Since(start)
			log.Printf("%s %s 410 (scanner) %s %s", r.Method, r.URL.Path, r.UserAgent(), dur.Round(time.Millisecond))
			if database.DB != nil {
				ip := r.RemoteAddr
				if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
					ip = strings.TrimSpace(strings.Split(fwd, ",")[0])
				}
				ip = strings.Split(ip, ":")[0]
				h := sha256.Sum256([]byte(ip))
				ipHash := hex.EncodeToString(h[:16])
				ua := r.UserAgent()
				ref := pageViewReferer(r)
				go database.DB.Exec(`INSERT INTO page_views (path, method, status, ip_hash, user_agent, referer, duration_ms, is_bot) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
					r.URL.Path, r.Method, http.StatusGone, ipHash, ua, ref, dur.Milliseconds(), true)
			}
			return
		}

		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(sw, r)
		dur := time.Since(start)
		if !suppressRequestLineIdentityTelemetry(r.URL.Path) {
			log.Printf("%s %s %d %s %s", r.Method, r.URL.Path, sw.status, r.UserAgent(), dur.Round(time.Millisecond))
		}

		if database.DB != nil && shouldLogPageView(r.URL.Path) {
			ip := r.RemoteAddr
			if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
				ip = strings.TrimSpace(strings.Split(fwd, ",")[0])
			}
			ip = strings.Split(ip, ":")[0]
			h := sha256.Sum256([]byte(ip))
			ipHash := hex.EncodeToString(h[:16])
			ua := r.UserAgent()
			ref := pageViewReferer(r)
			bot := isBotUA(ua) || isScannerPath(r.URL.Path)
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
		// Redirect nothumansearch.com → nothumansearch.ai (canonical) and www
		// variants → apex. Capability-bearing URLs fail closed instead of copying
		// their token into a Location header or permanent redirect cache.
		switch host {
		case "nothumansearch.com", "www.nothumansearch.com":
		case "www.nothumansearch.ai":
		default:
			next.ServeHTTP(w, r)
			return
		}
		if redirectRequestCarriesCapability(r) {
			setPrivateRedirectHeaders(w)
			http.Error(w, "use the canonical nothumansearch.ai host for this private link", http.StatusMisdirectedRequest)
			return
		}
		target := "https://nothumansearch.ai" + r.URL.RequestURI()
		status := http.StatusPermanentRedirect
		if r.URL.RawQuery != "" || (r.Method != http.MethodGet && r.Method != http.MethodHead) {
			setPrivateRedirectHeaders(w)
			status = http.StatusTemporaryRedirect
		}
		http.Redirect(w, r, target, status)
	})
}

func redirectRequestCarriesCapability(r *http.Request) bool {
	if r == nil {
		return true
	}
	path := r.URL.Path
	if path == "/auth/verify" || strings.HasPrefix(path, "/monitor/unsubscribe/") ||
		path == "/fix/success" {
		return true
	}
	if r.URL.RawQuery != "" && (strings.HasPrefix(path, "/site/") ||
		strings.HasPrefix(path, "/api/v1/site/")) {
		return true
	}
	query := r.URL.Query()
	for _, key := range []string{"token", "search_id", "session_id", "nhs_attribution"} {
		if query.Has(key) {
			return true
		}
	}
	return false
}

func setPrivateRedirectHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Referrer-Policy", "no-referrer")
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
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Idempotency-Key, X-NHS-Provider-Key, X-API-Key")
		if r.Method == "OPTIONS" {
			w.WriteHeader(204)
			return
		}
		next.ServeHTTP(w, r)
	})
}

const installScript = `#!/bin/sh
# Not Human Search — one-line MCP installer
# Usage:  curl -fsSL https://nothumansearch.ai/install | sh
#
# Wires the NHS MCP server into Claude Code if installed. Prints
# copy-paste snippets for Cursor, Cline, and Continue.
#
# NHS endpoint: https://nothumansearch.ai/mcp (streamable-http, no auth)
# 14 tools: search_agents, get_site_details, get_stats, list_categories,
# get_top_sites, submit_site, register_monitor, verify_mcp,
# check_url, find_mcp_servers, recent_additions, record_action_interest,
# prepare_provider_action, handoff_provider_action

set -eu
ENDPOINT="https://nothumansearch.ai/mcp"
NAME="nothumansearch"

banner() {
  printf '\n\033[1;38;5;173mNot Human Search — MCP installer\033[0m\n'
  printf 'Endpoint: %s\n\n' "$ENDPOINT"
}

banner

if command -v claude >/dev/null 2>&1; then
  printf 'Claude Code detected. Installing MCP server...\n'
  if claude mcp add --transport http "$NAME" "$ENDPOINT" 2>&1; then
    printf '\n\033[1;32mInstalled.\033[0m Restart Claude Code, then try:\n'
    printf '  "Find MCP servers that search Jira"\n'
    printf '  "Search the agentic web for payment APIs"\n\n'
  else
    printf '\n\033[1;33mInstall failed.\033[0m Run manually:\n'
    printf '  claude mcp add --transport http %s %s\n\n' "$NAME" "$ENDPOINT"
  fi
else
  printf '\033[1;33mClaude Code not found on PATH.\033[0m\n'
  printf 'Install it from https://claude.com/claude-code, then rerun this script.\n\n'
fi

cat <<SNIPPETS
For other MCP clients:

Cursor:     Add to ~/.cursor/mcp.json or project .cursor/mcp.json
            { "mcpServers": { "$NAME": { "url": "$ENDPOINT" } } }

Cline:      Settings -> MCP Servers -> Add:
            Name: $NAME   Transport: HTTP   URL: $ENDPOINT

Continue:   Add to ~/.continue/config.json
            "mcpServers": [ { "name": "$NAME", "url": "$ENDPOINT" } ]

Docs:       https://nothumansearch.ai/mcp-servers
SNIPPETS
`
