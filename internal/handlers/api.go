package handlers

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/unitedideas/nothumansearch/internal/crawler"
	"github.com/unitedideas/nothumansearch/internal/models"
)

type APIHandler struct {
	DB                        *sql.DB
	BaseURL                   string
	Auth                      *AuthService
	ProviderExchangeEnabled   bool
	searchRateLimiter         *mcpDiscoveryRateLimiter
	probeRateLimiter          *mcpDiscoveryRateLimiter
	prioritySearchRateLimiter *mcpDiscoveryRateLimiter
	priorityProbeRateLimiter  *mcpDiscoveryRateLimiter
}

// submitCrawlSem caps concurrent inline crawl goroutines spawned by /api/v1/submit.
// Without this, a bulk submitter can OOM a small Postgres instance by spawning
// hundreds of simultaneous crawl+upsert goroutines. Requests above the cap still
// queue in submissions table and get picked up by the scheduled recrawl.
var submitCrawlSem = make(chan struct{}, 4)

// submitRateLimiter is a per-IP rolling hourly limiter for /api/v1/submit.
// 20/hour/IP lets legitimate bulk submitters (e.g. directory importers) run
// through a few hundred URLs over an hour without blocking, but prevents
// single-IP floods of the submissions table. Legitimate agent use is far
// below this: most agents submit 1-3 sites total.
var (
	submitRLMu      sync.Mutex
	submitRLCounts  = map[string]int{}
	submitRLResetAt = time.Now().Add(time.Hour)
)

const submitRateLimit = 20

func submitRLAllow(ipHash string) (allowed bool, remaining int, resetUnix int64) {
	submitRLMu.Lock()
	defer submitRLMu.Unlock()
	now := time.Now()
	if now.After(submitRLResetAt) {
		submitRLCounts = map[string]int{}
		submitRLResetAt = now.Add(time.Hour)
	}
	if submitRLCounts[ipHash] >= submitRateLimit {
		return false, 0, submitRLResetAt.Unix()
	}
	submitRLCounts[ipHash]++
	remaining = submitRateLimit - submitRLCounts[ipHash]
	return true, remaining, submitRLResetAt.Unix()
}

func submitHashIP(r *http.Request) string {
	ip := strings.TrimSpace(r.RemoteAddr)
	// Fly-Client-IP is set by Fly's edge and is the authoritative client
	// address in production. X-Forwarded-For remains a fallback for local
	// reverse proxies and tests; RemoteAddr is the final direct-connection
	// fallback. Preferring the edge-owned header prevents callers from choosing
	// arbitrary abuse buckets by prepending a spoofed X-Forwarded-For value.
	if flyIP := strings.TrimSpace(r.Header.Get("Fly-Client-IP")); flyIP != "" {
		ip = flyIP
	} else if fwd := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); fwd != "" {
		ip = strings.TrimSpace(strings.Split(fwd, ",")[0])
	}
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}
	ip = strings.Trim(ip, "[]")
	sum := sha256.Sum256([]byte(ip))
	return hex.EncodeToString(sum[:8])
}

func demandRequestIsSynthetic(r *http.Request) bool {
	if r == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(r.Header.Get("NHS-Synthetic-Test")), "deploy-smoke")
}

// recordDemandSearchReceipt returns a public ID only after the full receipt and
// returned-result transaction commits. Callers must not advertise attribution
// when the demand schema is unavailable or a database write fails.
func recordDemandSearchReceipt(db *sql.DB, receipt models.DemandSearchReceipt, sites []models.Site) (string, error) {
	if db == nil {
		return "", models.ErrDemandStoreUnavailable
	}
	if receipt.PublicID == "" {
		publicID, err := models.GenerateDemandSearchID()
		if err != nil {
			return "", err
		}
		receipt.PublicID = publicID
	}
	if err := models.RecordDemandSearch(db, receipt, sites); err != nil {
		return "", err
	}
	return receipt.PublicID, nil
}

func NewAPIHandler(db *sql.DB) *APIHandler {
	return &APIHandler{
		DB:                        db,
		BaseURL:                   "https://nothumansearch.ai",
		ProviderExchangeEnabled:   true,
		searchRateLimiter:         newMCPDiscoveryRateLimiter(freeSearchHourlyLimit, time.Hour),
		probeRateLimiter:          newMCPDiscoveryRateLimiter(freeActiveProbeHourlyLimit, time.Hour),
		prioritySearchRateLimiter: newMCPDiscoveryRateLimiter(prioritySearchHourlyLimit, time.Hour),
		priorityProbeRateLimiter:  newMCPDiscoveryRateLimiter(priorityActiveProbeLimit, time.Hour),
	}
}

func (h *APIHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// GET /api/v1 — API index. Returned so that crawlers (including our own
// agent-first filter) can discover the structured API from the apex. The
// crawler's isAPIResponse check requires a JSON body at /api/v1; without
// this, NHS's own site loses the structured_api signal (15 points).
func (h *APIHandler) Index(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/v1" && r.URL.Path != "/api/v1/" {
		http.NotFound(w, r)
		return
	}
	h.writeJSON(w, 200, map[string]any{
		"$schema":            "https://schema.org/WebAPI",
		"name":               "Not Human Search API v1",
		"description":        "Free, neutral search for agent-ready sites. Separately disclosed provider-funded actions may be attached only after an organic search receipt and never change rank or score.",
		"version":            "1.1.0",
		"version_policy":     "Descriptive API release version, not a semantic-compatibility promise for controlled-pilot provider endpoints; provider action contracts carry their own explicit version.",
		"base_url":           "https://nothumansearch.ai/api/v1",
		"openapi_spec":       "https://nothumansearch.ai/openapi.yaml",
		"ai_plugin_manifest": "https://nothumansearch.ai/.well-known/ai-plugin.json",
		"mcp_endpoint":       "https://nothumansearch.ai/mcp",
		"endpoints": map[string]string{
			"search":                          "GET /api/v1/search?q=&category=&tag=&min_score=&has_api=&has_mcp=&has_openapi=&has_llms_txt=&page=",
			"site":                            "GET /api/v1/site/{domain}",
			"submit":                          "POST /api/v1/submit",
			"stats":                           "GET /api/v1/stats",
			"top":                             "GET /api/v1/top?category=&has_mcp=&has_openapi=&has_llms_txt=&limit=",
			"categories":                      "GET /api/v1/categories",
			"check":                           "POST /api/v1/check",
			"verify_mcp":                      "GET /api/v1/verify-mcp?url=",
			"commerce_catalog":                "GET /api/v1/catalog",
			"commerce_quote":                  "POST /api/v1/quote",
			"commerce_checkout":               "POST /api/v1/checkout",
			"api_key_plans":                   "GET /api/v1/api-keys/subscribe",
			"api_key_subscribe":               "POST /api/v1/api-keys/subscribe",
			"api_key_activate":                "GET /api/v1/api-keys/activate?session_id=",
			"monitor_register":                "POST /api/v1/monitor/register",
			"provider_claims":                 "GET|POST /api/v1/provider/claims (human session; DNS verification)",
			"provider_offers":                 "GET|POST /api/v1/provider/offers (human session; drafts require NHS commercial activation)",
			"provider_commercial_acceptances": "POST /api/v1/provider/commercial-acceptances (X-NHS-Provider-Key + Idempotency-Key; provider-authenticated acceptance only)",
			"provider_pilot_status":           "GET /api/v1/provider/pilot-status?limit= (X-NHS-Provider-Key; claim-scoped setup, terms, offer, handoff, and outcome continuity)",
			"provider_demand":                 "GET /api/v1/provider/demand?days= (X-NHS-Provider-Key; authenticated claim domain only; privacy-thresholded aggregate receipts)",
			"provider_controlled_intent":      "POST /api/v1/provider/action-tickets/resolve (X-NHS-Provider-Key; optional separately consented controlled intent after observed handoff; no query, identity, contact, charge, or proof)",
			"action_interests":                "POST /api/v1/action-interests (public; exact organic result + caller-attested principal interest; no provider contact)",
			"action_tickets":                  "POST /api/v1/action-tickets (public; exact consent v1; returns a bearer token and handoff endpoint, not the provider URL)",
			"action_ticket_handoff":           "POST /api/v1/action-tickets/handoff (public bearer JSON + nhs-provider-handoff-consent-v1; durable privacy-safe handoff receipt; no charge)",
			"provider_outcomes":               "POST /api/v1/provider/outcomes (X-NHS-Provider-Key + Idempotency-Key)",
			"receipt_verify":                  "POST /api/v1/action-receipts/verify (public signature, freshness, and current-state verification)",
		},
		"auth":       "none for discovery, action-interest receipts, action-ticket preparation, bearer handoff, or receipt verification; provider setup uses a human session; commercial acceptances, claim-scoped status and privacy-thresholded demand reports, optional post-handoff controlled-intent resolution, and outcome callbacks use a claim-scoped provider key; an optional API key only raises discovery throughput ceilings",
		"rate_limit": "free: search 240/hour/client and live verification 20/hour/client; priority API key: search 5000/hour/key and live verification 100/hour/key while monthly priority allocation remains",
		"provider_exchange": map[string]any{
			"setup_url":                                    "https://nothumansearch.ai/providers",
			"privacy_url":                                  "https://nothumansearch.ai/privacy",
			"consent_contract_url":                         "https://nothumansearch.ai/privacy#consent-v1",
			"organic_rank_sold":                            false,
			"raw_queries_sold":                             false,
			"agent_identities_sold":                        false,
			"direct_provider_access_free":                  true,
			"provider_mor_contract_required":               true,
			"provider_status_endpoint":                     "GET /api/v1/provider/pilot-status",
			"provider_status_scope":                        "authenticated_claim_only",
			"provider_demand_endpoint":                     "GET /api/v1/provider/demand?days=",
			"provider_demand_scope":                        "authenticated_claim_domain_only",
			"provider_demand_privacy_threshold_receipts":   models.ProviderDemandPrivacyThreshold,
			"provider_demand_returns_individual_receipts":  false,
			"provider_read_rate_limit":                     "240/hour/provider key per read surface",
			"ticket_preparation_contract":                  models.ProviderActionTicketPreparationV2,
			"handoff_contract_version":                     models.ProviderActionHandoffContractV1,
			"handoff_consent_version":                      models.ProviderActionHandoffConsentV1,
			"handoff_consent_url":                          "https://nothumansearch.ai/privacy#handoff-consent-v1",
			"controlled_intent_disclosure_optional":        true,
			"controlled_intent_disclosure_default":         false,
			"controlled_intent_disclosure_consent_version": models.ProviderControlledIntentDisclosureConsentV1,
			"controlled_intent_disclosure_consent_url":     "https://nothumansearch.ai/privacy#controlled-intent-disclosure-consent-v1",
			"controlled_intent_resolver_contract":          models.ProviderControlledIntentResolverV1,
			"controlled_intent_resolver_endpoint":          "POST /api/v1/provider/action-tickets/resolve",
			"controlled_intent_resolver_charge":            false,
			"controlled_intent_resolver_creates_proof":     false,
			"ticket_or_handoff_charge":                     false,
		},
	})
}

// GET /api/v1/search?q=...&category=...&min_score=...&page=...
func (h *APIHandler) Search(w http.ResponseWriter, r *http.Request) {
	protectReceiptBearingResponse(w)
	// Discovery is a public utility. Protect it with a temporary abuse throttle,
	// never a payment wall. Provider-funded products are monetized downstream of
	// search, so agents must be able to form demand and evaluate the index freely.
	access := resolveRequestRateAccess(h.DB, r)
	limiter := h.searchRateLimiter
	if access.tier == priorityRateLimitTier {
		limiter = h.prioritySearchRateLimiter
	}
	if limiter == nil {
		limit := freeSearchHourlyLimit
		if access.tier == priorityRateLimitTier {
			limit = prioritySearchHourlyLimit
		}
		limiter = newMCPDiscoveryRateLimiter(limit, time.Hour)
		if access.tier == priorityRateLimitTier {
			h.prioritySearchRateLimiter = limiter
		} else {
			h.searchRateLimiter = limiter
		}
	}
	remaining, retryAfter, ok := limiter.allow(access.bucket+":rest-search", time.Now())
	if ok && access.key != nil {
		var reserved bool
		access, reserved = reservePriorityUnit(h.DB, access, "rest", r.Method, "/api/v1/search", "")
		if !reserved {
			access = freeRequestRateAccess(r)
			if h.searchRateLimiter == nil {
				h.searchRateLimiter = newMCPDiscoveryRateLimiter(freeSearchHourlyLimit, time.Hour)
			}
			limiter = h.searchRateLimiter
			remaining, retryAfter, ok = limiter.allow(access.bucket+":rest-search", time.Now())
		}
	}
	access.setHeaders(w)
	w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limiter.limit))
	w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
	w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(retryAfter).Unix(), 10))
	if !ok {
		w.Header().Set("Retry-After", strconv.Itoa(max(1, int(retryAfter.Seconds()))))
		h.writeJSON(w, http.StatusTooManyRequests, map[string]interface{}{
			"error":       "rate_limit_exceeded",
			"message":     "Search safety limit exceeded; retry after the indicated interval. Search access is not paywalled.",
			"retry_after": max(1, int(retryAfter.Seconds())),
		})
		return
	}

	q := r.URL.Query()
	page := parsePositivePage(q.Get("page"))
	minScore := 0
	if ms := q.Get("min_score"); ms != "" {
		if s, err := strconv.Atoi(ms); err == nil {
			minScore = s
		}
	}
	perPage := 20
	if pp := q.Get("per_page"); pp != "" {
		if n, err := strconv.Atoi(pp); err == nil && n > 0 {
			perPage = n
		}
	}
	if perPage > 50 {
		perPage = 50
	}

	params := models.SearchParams{
		Query:      q.Get("q"),
		Category:   q.Get("category"),
		Tag:        q.Get("tag"),
		MinScore:   minScore,
		HasAPI:     q.Get("has_api") == "true",
		HasMCP:     q.Get("has_mcp") == "true",
		HasOpenAPI: q.Get("has_openapi") == "true",
		HasLLMsTxt: q.Get("has_llms_txt") == "true",
		Limit:      perPage,
		Page:       page,
	}

	sites, total, err := models.SearchSites(h.DB, params)
	if err != nil {
		access = releasePriorityUnit(h.DB, access)
		access.setHeaders(w)
		h.writeJSON(w, 500, map[string]string{"error": "search failed"})
		return
	}
	// Never return JSON null for results — consumers iterate without nil-check.
	if sites == nil {
		sites = []models.Site{}
	}

	synthetic := demandRequestIsSynthetic(r)
	searchID, receiptErr := recordDemandSearchReceipt(h.DB, models.DemandSearchReceipt{
		Surface:     "rest",
		Query:       params.Query,
		Category:    params.Category,
		HasAPI:      params.HasAPI,
		HasMCP:      params.HasMCP,
		HasOpenAPI:  params.HasOpenAPI,
		HasLLMsTxt:  params.HasLLMsTxt,
		ResultCount: total,
		Page:        page,
		PageSize:    perPage,
		Synthetic:   synthetic,
	}, sites)
	if receiptErr != nil {
		log.Printf("demand receipt REST search: %v", receiptErr)
	}

	response := map[string]interface{}{
		"access":                "free",
		"receipt_recorded":      searchID != "",
		"results":               sites,
		"total":                 total,
		"page":                  page,
		"per_page":              perPage,
		"has_next":              page*perPage < total,
		"paid_offers":           []publicProviderOffer{},
		"paid_offers_available": false,
		"action_interest": publicActionInterestOpportunity(
			h.BaseURL,
			searchID,
			sites,
			searchID != "" && len(sites) > 0 && !synthetic,
		),
	}
	response["action_interest"].(map[string]any)["endpoint"] = h.BaseURL + "/api/v1/action-interests"
	// The committed receipt belongs to free discovery, not the commercial
	// exchange. Return it in pilot, disabled recovery, and synthetic smoke mode
	// whenever the write succeeded; only provider-funded offer lookup is gated.
	if searchID != "" {
		response["search_id"] = searchID
	}
	if h.ProviderExchangeEnabled && !synthetic && searchID != "" {
		paidOffers, paidErr := models.ListPublicProviderOffersForOrganicResults(h.DB, searchID, sites)
		if paidErr != nil {
			log.Printf("provider offers REST search: %v", paidErr)
		} else if returnedErr := models.RecordProviderOffersReturned(h.DB, searchID, paidOffers); returnedErr != nil {
			log.Printf("provider offers REST return evidence: %v", returnedErr)
		} else {
			response["paid_offers"] = publicProviderOfferModelViews(paidOffers, h.BaseURL)
			response["paid_offers_available"] = len(paidOffers) > 0
		}
	}
	h.writeJSON(w, 200, response)
}

func parsePositivePage(raw string) int {
	if page, err := strconv.Atoi(strings.TrimSpace(raw)); err == nil && page > 0 {
		return page
	}
	return 1
}

// GET /api/v1/site/:domain
func (h *APIHandler) GetSite(w http.ResponseWriter, r *http.Request) {
	searchID := strings.TrimSpace(r.URL.Query().Get("search_id"))
	if searchID != "" {
		protectReceiptBearingResponse(w)
	}
	domain := strings.TrimPrefix(r.URL.Path, "/api/v1/site/")
	domain = strings.TrimPrefix(domain, "sites/")
	domain = strings.TrimPrefix(domain, "/api/v1/sites/")
	if domain == "" {
		h.writeJSON(w, 400, map[string]string{"error": "domain required"})
		return
	}

	site, err := models.GetSiteByDomain(h.DB, domain)
	if err != nil {
		h.writeJSON(w, 404, map[string]string{"error": "site not found"})
		return
	}
	if searchID != "" {
		recorded, selectionErr := models.RecordDemandSelection(h.DB, searchID, site.Domain, "rest")
		if selectionErr != nil {
			log.Printf("demand selection REST detail: %v", selectionErr)
		} else {
			w.Header().Set("NHS-Selection-Recorded", strconv.FormatBool(recorded))
		}
	}

	h.writeJSON(w, 200, site)
}

// POST /api/v1/submit  {"url": "https://example.com"}
func (h *APIHandler) SubmitSite(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		h.writeJSON(w, 405, map[string]string{"error": "POST required"})
		return
	}

	ipHash := submitHashIP(r)
	allowed, remaining, resetUnix := submitRLAllow(ipHash)
	w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", submitRateLimit))
	w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
	w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetUnix))
	if !allowed {
		w.Header().Set("Retry-After", fmt.Sprintf("%d", int(time.Until(time.Unix(resetUnix, 0)).Seconds())+1))
		h.writeJSON(w, 429, map[string]any{
			"error":     "rate limit exceeded: 20 submissions per hour per IP",
			"retry_sec": int(time.Until(time.Unix(resetUnix, 0)).Seconds()) + 1,
		})
		return
	}

	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.URL) == "" {
		h.writeJSON(w, 400, map[string]string{"error": "url required"})
		return
	}
	req.URL = strings.TrimSpace(req.URL)
	lowerURL := strings.ToLower(req.URL)
	if !strings.HasPrefix(lowerURL, "http://") && !strings.HasPrefix(lowerURL, "https://") {
		req.URL = "https://" + req.URL
	}
	if err := crawler.ValidatePublicURL(req.URL); err != nil {
		h.writeJSON(w, 400, map[string]string{"error": "url must resolve to a public HTTP(S) address"})
		return
	}

	_, err := h.DB.Exec(`
		INSERT INTO submissions (url, status) VALUES ($1, 'pending')
		ON CONFLICT DO NOTHING`, req.URL)
	if err != nil {
		h.writeJSON(w, 500, map[string]string{"error": "submission failed"})
		return
	}

	// Try to crawl immediately, but only if we're below concurrency cap.
	// Otherwise the submission stays in 'pending' and the scheduled recrawl
	// picks it up — avoiding OOM storms during bulk submissions.
	select {
	case submitCrawlSem <- struct{}{}:
		go func() {
			defer func() { <-submitCrawlSem }()
			site, err := crawler.CrawlSite(req.URL)
			if err != nil {
				log.Printf("submit crawl failed for %s: %v", req.URL, err)
				h.DB.Exec("UPDATE submissions SET status='failed' WHERE url=$1", req.URL)
				return
			}
			if err := models.UpsertSite(h.DB, site); err != nil {
				log.Printf("submit upsert failed for %s: %v", req.URL, err)
			}
			h.DB.Exec("UPDATE submissions SET status='crawled' WHERE url=$1", req.URL)
			log.Printf("submit crawl success: %s score=%d", site.Domain, site.AgenticScore)
		}()
	default:
		// semaphore full — leave as 'pending', recrawl will handle it
	}

	h.writeJSON(w, 201, map[string]string{"message": "submitted for crawling"})
}

// GET /api/v1/stats
func (h *APIHandler) Stats(w http.ResponseWriter, r *http.Request) {
	totalSites, avgScore, topCategory := models.GetStats(h.DB)
	w.Header().Set("Cache-Control", "public, max-age=300")
	h.writeJSON(w, 200, map[string]interface{}{
		"total_sites":  totalSites,
		"avg_score":    avgScore,
		"top_category": topCategory,
	})
}

// Top returns the highest-scored sites in the index, sorted by agentic_score
// DESC. Filterable by category and signal (has_mcp, has_llms_txt, etc).
// Public, free, cached 5 min — designed as a stable JSON other sites can
// mirror/embed. GET /api/v1/top?category=&has_mcp=&limit=
func (h *APIHandler) Top(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		h.writeJSON(w, 405, map[string]string{"error": "GET only"})
		return
	}
	q := r.URL.Query()
	limit := 50
	if l := q.Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}
	if limit > 100 {
		limit = 100
	}
	hasAPI := q.Get("has_api") == "true"
	hasMCP := q.Get("has_mcp") == "true"
	hasOpenAPI := q.Get("has_openapi") == "true"
	hasLLMsTxt := q.Get("has_llms_txt") == "true"

	sites, total, err := models.SearchSites(h.DB, models.SearchParams{
		Category:   q.Get("category"),
		Tag:        q.Get("tag"),
		HasAPI:     hasAPI,
		HasMCP:     hasMCP,
		HasOpenAPI: hasOpenAPI,
		HasLLMsTxt: hasLLMsTxt,
		Limit:      limit,
		Page:       1,
	})
	if err != nil {
		h.writeJSON(w, 500, map[string]string{"error": "query failed"})
		return
	}
	if sites == nil {
		sites = []models.Site{}
	}

	w.Header().Set("Cache-Control", "public, max-age=300")
	h.writeJSON(w, 200, map[string]interface{}{
		"results":     sites,
		"total":       total,
		"limit":       limit,
		"source":      "https://nothumansearch.ai",
		"description": "Highest-scored agent-ready sites, sorted by agentic readiness. Free, public, no auth.",
	})
}

// VerifyMCP is the REST wrapper around crawler.ProbeMCPJSONRPC — same
// behavior as the MCP verify_mcp tool, reachable by agents that don't
// speak MCP themselves.
// GET /api/v1/verify-mcp?url=https://example.com/mcp
func (h *APIHandler) VerifyMCP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		h.writeJSON(w, 405, map[string]string{"error": "GET only"})
		return
	}
	access := resolveRequestRateAccess(h.DB, r)
	limiter := h.probeRateLimiter
	if access.tier == priorityRateLimitTier {
		limiter = h.priorityProbeRateLimiter
	}
	if limiter == nil {
		limit := freeActiveProbeHourlyLimit
		if access.tier == priorityRateLimitTier {
			limit = priorityActiveProbeLimit
		}
		limiter = newMCPDiscoveryRateLimiter(limit, time.Hour)
		if access.tier == priorityRateLimitTier {
			h.priorityProbeRateLimiter = limiter
		} else {
			h.probeRateLimiter = limiter
		}
	}
	remaining, retryAfter, ok := limiter.allow(access.bucket+":rest-verify-mcp", time.Now())
	if ok && access.key != nil {
		var reserved bool
		access, reserved = reservePriorityUnit(h.DB, access, "rest", r.Method, "/api/v1/verify-mcp", "")
		if !reserved {
			access = freeRequestRateAccess(r)
			if h.probeRateLimiter == nil {
				h.probeRateLimiter = newMCPDiscoveryRateLimiter(freeActiveProbeHourlyLimit, time.Hour)
			}
			limiter = h.probeRateLimiter
			remaining, retryAfter, ok = limiter.allow(access.bucket+":rest-verify-mcp", time.Now())
		}
	}
	access.setHeaders(w)
	w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limiter.limit))
	w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
	w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(retryAfter).Unix(), 10))
	if !ok {
		retrySeconds := max(1, int(retryAfter.Seconds()))
		w.Header().Set("Retry-After", strconv.Itoa(retrySeconds))
		h.writeJSON(w, http.StatusTooManyRequests, map[string]interface{}{
			"error":       "rate_limit_exceeded",
			"message":     "Live-verification safety limit exceeded; retry after the indicated interval. Verification access is not paywalled.",
			"retry_after": retrySeconds,
		})
		return
	}
	raw := strings.TrimSpace(r.URL.Query().Get("url"))
	if raw == "" {
		access = releasePriorityUnit(h.DB, access)
		access.setHeaders(w)
		h.writeJSON(w, 400, map[string]string{"error": "url query param required"})
		return
	}
	// Case-insensitive prefix check — without ToLower, an input like
	// "HTTPS://example.com/mcp" would fail both HasPrefix checks and get
	// re-prefixed to "https://HTTPS://example.com/mcp" (broken URL).
	lower := strings.ToLower(raw)
	if !strings.HasPrefix(lower, "http://") && !strings.HasPrefix(lower, "https://") {
		raw = "https://" + raw
	}
	if err := crawler.ValidatePublicURL(raw); err != nil {
		access = releasePriorityUnit(h.DB, access)
		access.setHeaders(w)
		h.writeJSON(w, 400, map[string]string{"error": "url must resolve to a public HTTP(S) address"})
		return
	}

	// Don't cache — the caller is asking "is it live RIGHT NOW".
	w.Header().Set("Cache-Control", "no-store")

	verified := crawler.ProbeMCPJSONRPC(raw)
	note := "Endpoint responded with valid JSON-RPC 2.0 — server is live and MCP-compliant."
	if !verified {
		note = "Endpoint did not respond with valid JSON-RPC 2.0. Could be down, not an MCP server, or requires an initialize() handshake this probe does not send."
	}
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
	h.writeJSON(w, 200, map[string]interface{}{
		"verified": verified,
		"endpoint": raw,
		"note":     note,
	})
}

// GET /api/v1/categories
func (h *APIHandler) Categories(w http.ResponseWriter, r *http.Request) {
	cats, err := models.GetCategories(h.DB)
	if err != nil {
		h.writeJSON(w, 500, map[string]string{"error": "failed to get categories"})
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=300")
	h.writeJSON(w, 200, map[string]interface{}{
		"categories": cats,
	})
}

func (h *APIHandler) MCPAnalytics(w http.ResponseWriter, r *http.Request) {
	adminKey := os.Getenv("ADMIN_API_KEY")
	if adminKey == "" {
		h.writeJSON(w, 503, map[string]string{"error": "admin endpoint not configured"})
		return
	}
	auth := r.Header.Get("Authorization")
	if auth != "Bearer "+adminKey {
		h.writeJSON(w, 401, map[string]string{"error": "invalid admin key"})
		return
	}
	days := 14
	if d := r.URL.Query().Get("days"); d != "" {
		if n, err := strconv.Atoi(d); err == nil && n > 0 && n <= 90 {
			days = n
		}
	}
	data, err := models.GetMCPAnalytics(h.DB, days)
	if err != nil {
		h.writeJSON(w, 500, map[string]string{"error": "query failed"})
		return
	}
	data["days"] = days
	h.writeJSON(w, 200, data)
}

func (h *APIHandler) TrafficAnalytics(w http.ResponseWriter, r *http.Request) {
	adminKey := os.Getenv("ADMIN_API_KEY")
	if adminKey == "" {
		h.writeJSON(w, 503, map[string]string{"error": "admin endpoint not configured"})
		return
	}
	auth := r.Header.Get("Authorization")
	if auth != "Bearer "+adminKey {
		h.writeJSON(w, 401, map[string]string{"error": "invalid admin key"})
		return
	}
	days := 14
	if d := r.URL.Query().Get("days"); d != "" {
		if n, err := strconv.Atoi(d); err == nil && n > 0 && n <= 90 {
			days = n
		}
	}
	data, err := models.GetTrafficAnalytics(h.DB, days)
	if err != nil {
		h.writeJSON(w, 500, map[string]string{"error": "query failed"})
		return
	}
	data["days"] = days
	h.writeJSON(w, 200, data)
}

func (h *APIHandler) SignalAnalytics(w http.ResponseWriter, r *http.Request) {
	adminKey := os.Getenv("ADMIN_API_KEY")
	if adminKey == "" {
		h.writeJSON(w, 503, map[string]string{"error": "admin endpoint not configured"})
		return
	}
	auth := r.Header.Get("Authorization")
	if auth != "Bearer "+adminKey {
		h.writeJSON(w, 401, map[string]string{"error": "invalid admin key"})
		return
	}
	days := 14
	if d := r.URL.Query().Get("days"); d != "" {
		if n, err := strconv.Atoi(d); err == nil && n > 0 && n <= 90 {
			days = n
		}
	}
	data, err := models.GetIntentAnalytics(h.DB, days)
	if err != nil {
		h.writeJSON(w, 500, map[string]string{"error": "query failed"})
		return
	}
	data["days"] = days
	h.writeJSON(w, 200, data)
}

// ProviderDemandAnalytics is owner-controlled until a verified domain-claim
// flow exists. It exposes thresholded aggregates only: no raw queries, user
// agents, IP hashes, or individual search receipts.
func (h *APIHandler) ProviderDemandAnalytics(w http.ResponseWriter, r *http.Request) {
	adminKey := os.Getenv("ADMIN_API_KEY")
	if adminKey == "" {
		h.writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "admin endpoint not configured"})
		return
	}
	if r.Header.Get("Authorization") != "Bearer "+adminKey {
		h.writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid admin key"})
		return
	}
	domain := models.NormalizeProviderDomain(r.URL.Query().Get("domain"))
	if domain == "" {
		h.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "domain required"})
		return
	}
	days := 30
	if raw := r.URL.Query().Get("days"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 30 {
			days = parsed
		}
	}
	data, err := models.GetProviderDemandAnalytics(h.DB, domain, days)
	if err != nil {
		log.Printf("provider demand analytics: %v", err)
		h.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "query failed"})
		return
	}
	h.writeJSON(w, http.StatusOK, data)
}
