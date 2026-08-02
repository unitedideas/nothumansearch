package handlers

import (
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/unitedideas/nothumansearch/internal/models"
	"github.com/unitedideas/nothumansearch/internal/providerexchange"
)

type ProviderExchangeHandler struct {
	DB               *sql.DB
	BaseURL          string
	Auth             *AuthService
	Signer           *providerexchange.Signer
	TXTResolver      providerTXTResolver
	PageTemplate     *template.Template
	claimLimit       *mcpDiscoveryRateLimiter
	offerLimit       *mcpDiscoveryRateLimiter
	ticketLimit      *mcpDiscoveryRateLimiter
	outcomeLimit     *mcpDiscoveryRateLimiter
	receiptState     func(*sql.DB, string, string) (*models.PublicOutcomeReceiptState, error)
	leaseDNSChecks   func(*sql.DB, time.Time, int) ([]models.ProviderClaimDNSLease, error)
	completeDNSCheck func(*sql.DB, string, string, []string, time.Time) (*models.ProviderClaimDNSCheckResult, error)
	dnsNow           func() time.Time
}

func NewProviderExchangeHandler(db *sql.DB, baseURL string, auth *AuthService, templatesDir string) (*ProviderExchangeHandler, error) {
	signer, err := providerExchangeSignerFromEnv()
	if err != nil {
		return nil, fmt.Errorf("provider exchange signing configuration: %w", err)
	}
	retainedProofs, err := models.ProviderSigningKeyProofsInUse(db)
	if err != nil {
		return nil, fmt.Errorf("provider exchange signing-key retention lookup: %w", err)
	}
	if err := validateProviderSigningKeyRetention(signer, retainedProofs); err != nil {
		return nil, fmt.Errorf("provider exchange signing-key retention: %w", err)
	}
	pageTemplate, err := template.ParseFiles(
		filepath.Join(templatesDir, "providers.html"),
		filepath.Join(templatesDir, "privacy.html"),
	)
	if err != nil {
		return nil, fmt.Errorf("provider page template: %w", err)
	}
	return &ProviderExchangeHandler{
		DB:               db,
		BaseURL:          strings.TrimSuffix(baseURL, "/"),
		Auth:             auth,
		Signer:           signer,
		TXTResolver:      netProviderTXTResolver{resolver: nil},
		PageTemplate:     pageTemplate,
		claimLimit:       newMCPDiscoveryRateLimiter(20, time.Hour),
		offerLimit:       newMCPDiscoveryRateLimiter(60, time.Hour),
		ticketLimit:      newMCPDiscoveryRateLimiter(120, time.Hour),
		outcomeLimit:     newMCPDiscoveryRateLimiter(1000, time.Hour),
		receiptState:     models.GetPublicOutcomeReceiptState,
		leaseDNSChecks:   models.LeaseDueProviderClaimDNSChecks,
		completeDNSCheck: models.CompleteProviderClaimDNSCheck,
		dnsNow:           time.Now,
	}, nil
}

// validateProviderSigningKeyRetention fails startup when a persisted proof can
// no longer be reproduced or verified. This detects missing IDs and replacement
// of secret material under a reused ID without persisting a secret fingerprint.
func validateProviderSigningKeyRetention(signer *providerexchange.Signer, proofs []models.ProviderSigningKeyProof) error {
	if signer == nil {
		return errors.New("signer is unavailable")
	}
	for _, proof := range proofs {
		if !signer.SupportsKeyID(proof.KeyID) {
			return fmt.Errorf("verification material missing for persisted key id %q", proof.KeyID)
		}
		switch proof.Kind {
		case models.ProviderSigningProofAttribution:
			token, err := signer.SignAttribution(providerexchange.AttributionClaims{
				Version: providerexchange.AttributionTokenVersion,
				KeyID:   proof.KeyID, TicketID: proof.TicketID, OfferID: proof.OfferID,
				IssuedAt: proof.IssuedAt.UTC().Unix(), ExpiresAt: proof.ExpiresAt.UTC().Unix(),
				Nonce: proof.TokenNonce,
			})
			if err != nil {
				return fmt.Errorf("persisted attribution proof cannot be reconstructed for key id %q", proof.KeyID)
			}
			got := models.HashProviderSecret(token)
			if len(got) != len(proof.TokenHash) ||
				subtle.ConstantTimeCompare([]byte(got), []byte(proof.TokenHash)) != 1 {
				return fmt.Errorf("verification material changed for persisted key id %q", proof.KeyID)
			}
		case models.ProviderSigningProofOutcome:
			receipt, err := signer.VerifyOutcomeReceiptSignature(proof.SignedReceipt, proof.Signature)
			if err != nil || receipt.KeyID != proof.KeyID {
				return fmt.Errorf("verification material changed for persisted key id %q", proof.KeyID)
			}
		default:
			return errors.New("persisted signing proof kind is invalid")
		}
	}
	return nil
}

func providerExchangeSignerFromEnv() (*providerexchange.Signer, error) {
	const (
		activeKeyIDEnv  = "NHS_PROVIDER_EXCHANGE_SIGNING_KEY_ID"
		activeSecretEnv = "NHS_PROVIDER_EXCHANGE_SIGNING_KEY"
		previousKeysEnv = "NHS_PROVIDER_EXCHANGE_PREVIOUS_SIGNING_KEYS_JSON"
	)
	keyID := strings.TrimSpace(os.Getenv(activeKeyIDEnv))
	secret := os.Getenv(activeSecretEnv)
	previous := map[string]string{}
	if raw := strings.TrimSpace(os.Getenv(previousKeysEnv)); raw != "" {
		if len(raw) > 16<<10 {
			return nil, errors.New("previous signing keyring configuration is too large")
		}
		decoder := json.NewDecoder(strings.NewReader(raw))
		if err := decoder.Decode(&previous); err != nil {
			return nil, errors.New("previous signing keyring configuration is invalid")
		}
		var trailing any
		if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
			return nil, errors.New("previous signing keyring configuration has trailing data")
		}
	}
	return providerexchange.NewSignerKeyring(keyID, secret, previous)
}

// ValidateProviderExchangeSigningConfiguration performs the secret-free startup
// preflight that must run before migrations. Persisted proof retention is
// checked later, after the database schema is current.
func ValidateProviderExchangeSigningConfiguration() error {
	if _, err := providerExchangeSignerFromEnv(); err != nil {
		return fmt.Errorf("provider exchange signing configuration: %w", err)
	}
	return nil
}

func providerWriteJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func (h *ProviderExchangeHandler) requireAccount(w http.ResponseWriter, r *http.Request, mutation bool) *models.Account {
	if mutation && !requestOriginAllowed(r, h.BaseURL) {
		providerWriteJSON(w, http.StatusForbidden, map[string]string{"error": "cross-origin mutation rejected"})
		return nil
	}
	if h.Auth == nil {
		providerWriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "provider account authentication unavailable"})
		return nil
	}
	account := h.Auth.CurrentAccount(r)
	if account == nil {
		providerWriteJSON(w, http.StatusUnauthorized, map[string]string{
			"error":     "provider account session required",
			"login_url": h.BaseURL + "/login",
		})
		return nil
	}
	return account
}

func extractProviderKey(r *http.Request) string {
	if r == nil {
		return ""
	}
	if key := strings.TrimSpace(r.Header.Get("X-NHS-Provider-Key")); key != "" {
		return key
	}
	authorization := strings.TrimSpace(r.Header.Get("Authorization"))
	for _, prefix := range []string{"Bearer ", "Provider "} {
		if strings.HasPrefix(authorization, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(authorization, prefix))
		}
	}
	return ""
}

func (h *ProviderExchangeHandler) requireProviderKey(w http.ResponseWriter, r *http.Request) *models.ProviderAPIKey {
	raw := extractProviderKey(r)
	if raw == "" || h.DB == nil {
		providerWriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "valid provider callback key required"})
		return nil
	}
	key, err := models.ResolveProviderAPIKey(h.DB, raw)
	if err != nil || key == nil {
		providerWriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "valid provider callback key required"})
		return nil
	}
	return key
}

func (h *ProviderExchangeHandler) requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	configured := os.Getenv("ADMIN_API_KEY")
	if configured == "" {
		providerWriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "admin endpoint not configured"})
		return false
	}
	authorization := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(authorization, "Bearer ") {
		providerWriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid admin key"})
		return false
	}
	provided := strings.TrimSpace(strings.TrimPrefix(authorization, "Bearer "))
	if !constantTimeStringEqual(provided, configured) {
		providerWriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid admin key"})
		return false
	}
	return true
}

func providerOutcomePayloadHash(ticketID, outcome, attributionToken string) string {
	outcome = strings.ToLower(strings.TrimSpace(outcome))
	digest := sha256.Sum256([]byte(strings.TrimSpace(ticketID) + "\x00" + outcome + "\x00" + strings.TrimSpace(attributionToken)))
	return hex.EncodeToString(digest[:])
}

func providerExchangeStatus(err error) (int, string) {
	switch {
	case err == nil:
		return http.StatusOK, ""
	case errors.Is(err, sql.ErrNoRows):
		return http.StatusNotFound, "not found"
	case errors.Is(err, models.ErrProviderSiteClaimed), errors.Is(err, models.ErrProviderClaimExists):
		return http.StatusConflict, err.Error()
	case errors.Is(err, models.ErrProviderAPIKeyExists):
		return http.StatusConflict, err.Error()
	case errors.Is(err, models.ErrProviderChallengeExpired), errors.Is(err, models.ErrActionTicketExpired):
		return http.StatusGone, err.Error()
	case errors.Is(err, models.ErrProviderChallengeMismatch), errors.Is(err, models.ErrProviderClaimNotVerified),
		errors.Is(err, models.ErrProviderClaimVerificationStale):
		return http.StatusConflict, err.Error()
	case errors.Is(err, models.ErrProviderOfferNotPublic):
		return http.StatusNotFound, "provider offer unavailable"
	case errors.Is(err, models.ErrProviderOfferRevoked):
		return http.StatusConflict, "provider offer authorization was revoked"
	case errors.Is(err, models.ErrProviderOfferLimit):
		return http.StatusConflict, "provider offer limit reached"
	case errors.Is(err, models.ErrProviderBudgetLimit):
		return http.StatusUnprocessableEntity, "provider budget limit reached"
	case errors.Is(err, models.ErrInsufficientProviderFunds), errors.Is(err, models.ErrProviderTermsCreditLimit):
		return http.StatusPaymentRequired, "provider budget requires replenishment; the principal is not charged"
	case errors.Is(err, models.ErrProviderIdempotency), errors.Is(err, models.ErrProviderOutcomeExists), errors.Is(err, models.ErrProviderOutcomeTransition):
		return http.StatusConflict, err.Error()
	case errors.Is(err, models.ErrActionTicketExists):
		return http.StatusConflict, err.Error()
	case errors.Is(err, models.ErrInvalidProviderExchange):
		return http.StatusBadRequest, "invalid provider exchange request"
	default:
		return http.StatusInternalServerError, "provider exchange operation failed"
	}
}
