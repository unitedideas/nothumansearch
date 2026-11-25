package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/unitedideas/nothumansearch/internal/models"
	"github.com/unitedideas/nothumansearch/internal/providerexchange"
)

const (
	controlledIntentTestTicketID = "3a59ca8e-d61d-47e2-91dd-fecd9f711234"
	controlledIntentTestOfferID  = "4b69ca8e-d61d-47e2-91dd-fecd9f711234"
	controlledIntentTestClaimID  = "5c79ca8e-d61d-47e2-91dd-fecd9f711234"
)

func controlledIntentTestSignerAndToken(t *testing.T, expiresAt time.Time) (*providerexchange.Signer, string) {
	t.Helper()
	signer, err := providerexchange.NewSigner("test-controlled-intent-key-0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	issuedAt := expiresAt.Add(-time.Hour)
	claims, err := providerexchange.NewAttributionClaims(
		controlledIntentTestTicketID, controlledIntentTestOfferID, issuedAt, expiresAt,
	)
	if err != nil {
		t.Fatal(err)
	}
	token, err := signer.SignAttribution(claims)
	if err != nil {
		t.Fatal(err)
	}
	return signer, token
}

func controlledIntentTestHandler(
	t *testing.T,
	resolve func(*sql.DB, *models.ProviderAPIKey, string, string, string) (*models.ProviderControlledIntentResolution, error),
) (*ProviderExchangeHandler, string) {
	t.Helper()
	signer, token := controlledIntentTestSignerAndToken(t, time.Now().UTC().Add(time.Hour))
	return &ProviderExchangeHandler{
		Signer:            signer,
		resolverAuthLimit: newMCPDiscoveryRateLimiter(100, time.Hour),
		resolverLimit:     newMCPDiscoveryRateLimiter(100, time.Hour),
		resolveProviderKey: func(_ *sql.DB, raw string) (*models.ProviderAPIKey, error) {
			if raw != "provider-test-key" {
				t.Fatalf("provider key = %q", raw)
			}
			return &models.ProviderAPIKey{ID: 17, ProviderClaimID: controlledIntentTestClaimID, Status: "active"}, nil
		},
		resolveControlledIntent: resolve,
	}, token
}

func TestControlledIntentResolverReturnsOnlyNarrowConsentedBundle(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC().Truncate(time.Second)
	var expectedToken string
	h, token := controlledIntentTestHandler(t, func(
		_ *sql.DB, key *models.ProviderAPIKey, ticketID, offerID, rawToken string,
	) (*models.ProviderControlledIntentResolution, error) {
		if key.ID != 17 || key.ProviderClaimID != controlledIntentTestClaimID ||
			ticketID != controlledIntentTestTicketID || offerID != controlledIntentTestOfferID ||
			rawToken != expectedToken {
			t.Fatalf("resolver binding key=%#v ticket=%q offer=%q token_match=%t", key, ticketID, offerID, rawToken == expectedToken)
		}
		return &models.ProviderControlledIntentResolution{
			ResolverContractVersion: models.ProviderControlledIntentResolverV1,
			TicketID:                ticketID,
			OfferID:                 offerID,
			OfferVersion:            3,
			ActionType:              "trial",
			ControlledIntent: models.ProviderControlledIntent{
				DemandTopic: "developer-tools", RegionCode: "US", BudgetBand: "500_1999",
				Urgency: "7_days", RequirementFlags: []string{"api_access", "sandbox"},
			},
			ObservedAt:           now,
			IntentAvailableUntil: now.Add(24 * time.Hour),
			ConsentVersion:       models.ProviderControlledIntentDisclosureConsentV1,
		}, nil
	})
	expectedToken = token
	body, err := json.Marshal(map[string]string{"attribution_token": " " + token + " "})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/provider/action-tickets/resolve", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-NHS-Provider-Key", "provider-test-key")
	rr := httptest.NewRecorder()
	h.ResolveProviderControlledIntent(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("resolver status=%d body=%s", rr.Code, rr.Body.String())
	}
	for name, want := range map[string]string{
		"Cache-Control": "private, no-store", "Pragma": "no-cache", "Referrer-Policy": "no-referrer",
	} {
		if got := rr.Header().Get(name); got != want {
			t.Fatalf("resolver %s=%q, want %q", name, got, want)
		}
	}
	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	wantTop := []string{"action_type", "consent_version", "controlled_intent", "intent_available_until", "observed_at", "offer_id", "offer_version", "resolver_contract_version", "ticket_id"}
	gotTop := make([]string, 0, len(payload))
	for key := range payload {
		gotTop = append(gotTop, key)
	}
	sort.Strings(gotTop)
	if strings.Join(gotTop, ",") != strings.Join(wantTop, ",") {
		t.Fatalf("resolver top-level fields=%v, want %v; body=%s", gotTop, wantTop, rr.Body.String())
	}
	intent, ok := payload["controlled_intent"].(map[string]any)
	if !ok {
		t.Fatalf("controlled_intent=%#v", payload["controlled_intent"])
	}
	wantIntent := []string{"budget_band", "demand_topic", "region_code", "requirement_flags", "urgency"}
	gotIntent := make([]string, 0, len(intent))
	for key := range intent {
		gotIntent = append(gotIntent, key)
	}
	sort.Strings(gotIntent)
	if strings.Join(gotIntent, ",") != strings.Join(wantIntent, ",") {
		t.Fatalf("controlled-intent fields=%v, want %v", gotIntent, wantIntent)
	}
	for _, forbidden := range []string{
		"query", "search_receipt", "provider_claim", "contact", "email", "agent", "principal_id",
		"action_url", "bounty", "currency", "charge", "commercial_proof", "token",
	} {
		if strings.Contains(strings.ToLower(rr.Body.String()), forbidden) {
			t.Fatalf("resolver response contains forbidden marker %q: %s", forbidden, rr.Body.String())
		}
	}
}

func TestControlledIntentResolverRejectsUnknownSensitiveFieldsBeforeModel(t *testing.T) {
	t.Parallel()
	called := false
	h, token := controlledIntentTestHandler(t, func(
		*sql.DB, *models.ProviderAPIKey, string, string, string,
	) (*models.ProviderControlledIntentResolution, error) {
		called = true
		return nil, errors.New("must not run")
	})
	for _, field := range []string{"query", "contact", "notes", "agent_id", "principal_id", "ticket_id"} {
		called = false
		body := `{"attribution_token":` + quoteJSON(token) + `,"` + field + `":"private"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/provider/action-tickets/resolve", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-NHS-Provider-Key", "provider-test-key")
		rr := httptest.NewRecorder()
		h.ResolveProviderControlledIntent(rr, req)
		if rr.Code != http.StatusBadRequest || called {
			t.Fatalf("field=%s status=%d called=%t body=%s", field, rr.Code, called, rr.Body.String())
		}
	}
}

func TestControlledIntentResolverStatusAndAuthenticationBoundaries(t *testing.T) {
	t.Parallel()

	t.Run("invalid token", func(t *testing.T) {
		h, _ := controlledIntentTestHandler(t, nil)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/provider/action-tickets/resolve", strings.NewReader(`{"attribution_token":"invalid"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-NHS-Provider-Key", "provider-test-key")
		rr := httptest.NewRecorder()
		h.ResolveProviderControlledIntent(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
	})

	t.Run("expired token", func(t *testing.T) {
		signer, token := controlledIntentTestSignerAndToken(t, time.Now().UTC().Add(-time.Minute))
		h := &ProviderExchangeHandler{
			Signer: signer, resolverAuthLimit: newMCPDiscoveryRateLimiter(10, time.Hour), resolverLimit: newMCPDiscoveryRateLimiter(10, time.Hour),
			resolveProviderKey: func(*sql.DB, string) (*models.ProviderAPIKey, error) {
				return &models.ProviderAPIKey{ID: 17, ProviderClaimID: controlledIntentTestClaimID}, nil
			},
		}
		req := httptest.NewRequest(http.MethodPost, "/api/v1/provider/action-tickets/resolve", strings.NewReader(`{"attribution_token":`+quoteJSON(token)+`}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-NHS-Provider-Key", "provider-test-key")
		rr := httptest.NewRecorder()
		h.ResolveProviderControlledIntent(rr, req)
		if rr.Code != http.StatusGone {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
	})

	for name, testCase := range map[string]struct {
		err  error
		want int
	}{
		"unavailable": {sql.ErrNoRows, http.StatusNotFound},
		"invalid":     {models.ErrInvalidProviderExchange, http.StatusBadRequest},
		"dependency":  {errors.New("database unavailable"), http.StatusServiceUnavailable},
	} {
		name, modelErr, want := name, testCase.err, testCase.want
		t.Run(name, func(t *testing.T) {
			h, token := controlledIntentTestHandler(t, func(*sql.DB, *models.ProviderAPIKey, string, string, string) (*models.ProviderControlledIntentResolution, error) {
				return nil, modelErr
			})
			req := httptest.NewRequest(http.MethodPost, "/api/v1/provider/action-tickets/resolve", strings.NewReader(`{"attribution_token":`+quoteJSON(token)+`}`))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-NHS-Provider-Key", "provider-test-key")
			rr := httptest.NewRecorder()
			h.ResolveProviderControlledIntent(rr, req)
			if rr.Code != want {
				t.Fatalf("status=%d, want=%d body=%s", rr.Code, want, rr.Body.String())
			}
		})
	}

	t.Run("provider key required", func(t *testing.T) {
		h, token := controlledIntentTestHandler(t, nil)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/provider/action-tickets/resolve", strings.NewReader(`{"attribution_token":`+quoteJSON(token)+`}`))
		rr := httptest.NewRecorder()
		h.ResolveProviderControlledIntent(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
	})

	t.Run("provider key dependency unavailable", func(t *testing.T) {
		h, token := controlledIntentTestHandler(t, nil)
		h.resolveProviderKey = func(*sql.DB, string) (*models.ProviderAPIKey, error) {
			return nil, errors.New("database unavailable")
		}
		req := httptest.NewRequest(http.MethodPost, "/api/v1/provider/action-tickets/resolve", strings.NewReader(`{"attribution_token":`+quoteJSON(token)+`}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-NHS-Provider-Key", "provider-test-key")
		rr := httptest.NewRecorder()
		h.ResolveProviderControlledIntent(rr, req)
		if rr.Code != http.StatusServiceUnavailable || !strings.Contains(rr.Body.String(), "authentication unavailable") {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
	})

	t.Run("signer required", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/provider/action-tickets/resolve", strings.NewReader(`{"attribution_token":"opaque"}`))
		rr := httptest.NewRecorder()
		(&ProviderExchangeHandler{}).ResolveProviderControlledIntent(rr, req)
		if rr.Code != http.StatusServiceUnavailable {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
	})

	t.Run("post only", func(t *testing.T) {
		h, _ := controlledIntentTestHandler(t, nil)
		rr := httptest.NewRecorder()
		h.ResolveProviderControlledIntent(rr, httptest.NewRequest(http.MethodGet, "/api/v1/provider/action-tickets/resolve", nil))
		if rr.Code != http.StatusMethodNotAllowed || rr.Header().Get("Allow") != http.MethodPost {
			t.Fatalf("status=%d allow=%q body=%s", rr.Code, rr.Header().Get("Allow"), rr.Body.String())
		}
	})
}

func TestControlledIntentResolverLimitsInvalidKeysBeforeDatabaseAmplification(t *testing.T) {
	t.Parallel()
	signer, token := controlledIntentTestSignerAndToken(t, time.Now().UTC().Add(time.Hour))
	lookups := 0
	h := &ProviderExchangeHandler{
		Signer:            signer,
		resolverAuthLimit: newMCPDiscoveryRateLimiter(1, time.Hour),
		resolverLimit:     newMCPDiscoveryRateLimiter(100, time.Hour),
		resolveProviderKey: func(*sql.DB, string) (*models.ProviderAPIKey, error) {
			lookups++
			return nil, sql.ErrNoRows
		},
	}
	invoke := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/provider/action-tickets/resolve", strings.NewReader(`{"attribution_token":`+quoteJSON(token)+`}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-NHS-Provider-Key", "random-provider-key")
		req.RemoteAddr = "203.0.113.20:443"
		rr := httptest.NewRecorder()
		h.ResolveProviderControlledIntent(rr, req)
		return rr
	}
	if rr := invoke(); rr.Code != http.StatusUnauthorized {
		t.Fatalf("first status=%d body=%s", rr.Code, rr.Body.String())
	}
	if rr := invoke(); rr.Code != http.StatusTooManyRequests || rr.Header().Get("Retry-After") == "" {
		t.Fatalf("second status=%d retry=%q body=%s", rr.Code, rr.Header().Get("Retry-After"), rr.Body.String())
	}
	if lookups != 1 {
		t.Fatalf("provider key database lookups=%d, want 1", lookups)
	}
}
