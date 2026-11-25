package handlers

import (
	"context"
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
	DB                         *sql.DB
	BaseURL                    string
	Auth                       *AuthService
	Signer                     *providerexchange.Signer
	TXTResolver                providerTXTResolver
	PageTemplate               *template.Template
	claimLimit                 *mcpDiscoveryRateLimiter
	offerLimit                 *mcpDiscoveryRateLimiter
	ticketLimit                *mcpDiscoveryRateLimiter
	handoffLimit               *mcpDiscoveryRateLimiter
	resolverAuthLimit          *mcpDiscoveryRateLimiter
	resolverLimit              *mcpDiscoveryRateLimiter
	outcomeLimit               *mcpDiscoveryRateLimiter
	commercialLimit            *mcpDiscoveryRateLimiter
	providerReadLimit          *mcpDiscoveryRateLimiter
	resolveProviderKey         func(*sql.DB, string) (*models.ProviderAPIKey, error)
	recordCommercialAcceptance func(*sql.DB, *models.ProviderAPIKey, models.ProviderCommercialAcceptanceInput) (*models.ProviderCommercialAcceptanceEvent, bool, error)
	verifyPilotCompany         func(*sql.DB, string, string, string, string) (*models.ProviderPilotCompany, bool, error)
	recordVerifiedFunding      func(*sql.DB, models.VerifiedProviderFundingInput) (*models.ProviderCommercialCommitmentEvent, bool, error)
	recordVerifiedTerms        func(*sql.DB, models.VerifiedProviderTermsInput) (*models.ProviderCommercialCommitmentEvent, bool, error)
	reverseVerifiedFunding     func(*sql.DB, models.ProviderFundingReversalInput) (*models.ProviderCommercialCommitmentEvent, bool, error)
	recordActionHandoff        func(*sql.DB, models.ProviderActionHandoffInput) (*models.ActionTicket, *models.ProviderActionHandoffReceipt, error)
	resolveControlledIntent    func(*sql.DB, *models.ProviderAPIKey, string, string, string) (*models.ProviderControlledIntentResolution, error)
	receiptState               func(*sql.DB, string, string) (*models.PublicOutcomeReceiptState, error)
	leaseDNSChecks             func(*sql.DB, time.Time, int) ([]models.ProviderClaimDNSLease, error)
	completeDNSCheck           func(*sql.DB, string, string, []string, time.Time) (*models.ProviderClaimDNSCheckResult, error)
	dnsNow                     func() time.Time
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
	handler, err := NewProviderExchangePageHandler(db, baseURL, auth, templatesDir)
	if err != nil {
		return nil, err
	}
	handler.Signer = signer
	handler.TXTResolver = netProviderTXTResolver{resolver: nil}
	handler.claimLimit = newMCPDiscoveryRateLimiter(20, time.Hour)
	handler.offerLimit = newMCPDiscoveryRateLimiter(60, time.Hour)
	handler.ticketLimit = newMCPDiscoveryRateLimiter(120, time.Hour)
	handler.handoffLimit = newMCPDiscoveryRateLimiter(240, time.Hour)
	handler.resolverAuthLimit = newMCPDiscoveryRateLimiter(120, time.Hour)
	handler.resolverLimit = newMCPDiscoveryRateLimiter(1000, time.Hour)
	handler.outcomeLimit = newMCPDiscoveryRateLimiter(1000, time.Hour)
	handler.commercialLimit = newMCPDiscoveryRateLimiter(200, time.Hour)
	handler.providerReadLimit = newMCPDiscoveryRateLimiter(240, time.Hour)
	handler.resolveProviderKey = models.ResolveProviderAPIKey
	handler.recordCommercialAcceptance = models.RecordProviderCommercialAcceptance
	handler.verifyPilotCompany = models.VerifyProviderPilotCompany
	handler.recordVerifiedFunding = models.RecordVerifiedProviderFunding
	handler.recordVerifiedTerms = models.RecordVerifiedProviderTerms
	handler.reverseVerifiedFunding = models.ReverseVerifiedProviderFunding
	handler.recordActionHandoff = models.RecordActionTicketHandoff
	handler.resolveControlledIntent = models.ResolveProviderControlledIntent
	handler.receiptState = models.GetPublicOutcomeReceiptState
	handler.leaseDNSChecks = models.LeaseDueProviderClaimDNSChecks
	handler.completeDNSCheck = models.CompleteProviderClaimDNSCheck
	handler.dnsNow = time.Now
	return handler, nil
}

// NewProviderExchangePageHandler builds only the read-only provider and privacy
// surfaces. It deliberately does not load signing material or install any
// mutation dependency, so the new-schema-compatible binary can keep free
// discovery and Stage 1 observation online while commercial exchange writes
// are disabled during a forward recovery.
func NewProviderExchangePageHandler(db *sql.DB, baseURL string, auth *AuthService, templatesDir string) (*ProviderExchangeHandler, error) {
	pageTemplate, err := template.ParseFiles(
		filepath.Join(templatesDir, "providers.html"),
		filepath.Join(templatesDir, "privacy.html"),
	)
	if err != nil {
		return nil, fmt.Errorf("provider page template: %w", err)
	}
	return &ProviderExchangeHandler{
		DB:           db,
		BaseURL:      strings.TrimSuffix(baseURL, "/"),
		Auth:         auth,
		PageTemplate: pageTemplate,
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
		case models.ProviderSigningProofManifest:
			manifest, err := signer.VerifyCommercialProofManifestSignature(proof.SignedManifest, proof.Signature)
			if err != nil || manifest.KeyID != proof.KeyID {
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

type ProviderSigningRetentionItem struct {
	KeyID string `json:"key_id"`
	Kind  string `json:"kind"`
}

type ProviderSigningRetentionReport struct {
	SignerRequired         bool                           `json:"signer_required"`
	ConfigurationValidated bool                           `json:"configuration_validated"`
	PersistedProofCount    int                            `json:"persisted_proof_count"`
	Proofs                 []ProviderSigningRetentionItem `json:"proofs"`
}

// ValidateProviderExchangeSigningRetentionReadOnly verifies the injected
// keyring against a bounded sample of every persisted signing domain without
// returning key material, token hashes, signatures, or signed payloads. A pilot
// preflight always requires a configured signer; disabled recovery requires it
// only when persisted proof material already exists.
func ValidateProviderExchangeSigningRetentionReadOnly(
	db *sql.DB, requireSigner bool,
) (*ProviderSigningRetentionReport, error) {
	return ValidateProviderExchangeSigningRetentionReadOnlyContext(context.Background(), db, requireSigner)
}

// ValidateProviderExchangeSigningRetentionReadOnlyContext applies the caller's
// deadline to both the schema probe and bounded persisted-proof lookup.
func ValidateProviderExchangeSigningRetentionReadOnlyContext(
	ctx context.Context, db *sql.DB, requireSigner bool,
) (*ProviderSigningRetentionReport, error) {
	if ctx == nil || db == nil {
		return nil, errors.New("provider exchange database is unavailable")
	}
	report := &ProviderSigningRetentionReport{
		SignerRequired: requireSigner,
		Proofs:         []ProviderSigningRetentionItem{},
	}
	var proofTablesExist bool
	if err := db.QueryRowContext(ctx, `
		SELECT to_regclass('public.action_tickets') IS NOT NULL
		   AND to_regclass('public.outcome_receipts') IS NOT NULL`).Scan(&proofTablesExist); err != nil {
		return nil, fmt.Errorf("provider exchange signing-store inspection: %w", err)
	}
	proofs := []models.ProviderSigningKeyProof{}
	if proofTablesExist {
		var err error
		proofs, err = models.ProviderSigningKeyProofsInUseContext(ctx, db)
		if err != nil {
			return nil, fmt.Errorf("provider exchange signing-key retention lookup: %w", err)
		}
	}
	report.PersistedProofCount = len(proofs)
	for _, proof := range proofs {
		report.Proofs = append(report.Proofs, ProviderSigningRetentionItem{
			KeyID: proof.KeyID,
			Kind:  proof.Kind,
		})
	}
	if !requireSigner && len(proofs) == 0 {
		return report, nil
	}
	report.SignerRequired = true
	signer, err := providerExchangeSignerFromEnv()
	if err != nil {
		return nil, fmt.Errorf("provider exchange signing configuration: %w", err)
	}
	if err := validateProviderSigningKeyRetention(signer, proofs); err != nil {
		return nil, fmt.Errorf("provider exchange signing-key retention: %w", err)
	}
	report.ConfigurationValidated = true
	return report, nil
}

// ProviderExchangeProtectedMigrationPreflight closes the signing-key TOCTOU
// window at the one-way 022 cutover. It shares the migration transaction and
// the same ACCESS EXCLUSIVE ticket-writer lock. Empty stores need no signer;
// persisted proof requires compatible retained material in both pilot and
// disabled modes. Receipted migrations never invoke this hook.
func ProviderExchangeProtectedMigrationPreflight(ctx context.Context, tx *sql.Tx, migrationName string) error {
	if migrationName != "022_provider_commercial_proof.sql" {
		return nil
	}
	if ctx == nil || tx == nil {
		return errors.New("provider exchange migration preflight transaction is unavailable")
	}
	if _, err := tx.ExecContext(ctx, `LOCK TABLE public.action_tickets IN ACCESS EXCLUSIVE MODE`); err != nil {
		return fmt.Errorf("lock action-ticket writers for signing-key preflight: %w", err)
	}
	retainedProofs, err := models.ProviderSigningKeyProofsInUseTx(ctx, tx)
	if err != nil {
		return fmt.Errorf("provider exchange signing-key retention lookup: %w", err)
	}
	if len(retainedProofs) == 0 {
		return nil
	}
	signer, err := providerExchangeSignerFromEnv()
	if err != nil {
		return fmt.Errorf("provider exchange signing configuration: %w", err)
	}
	if err := validateProviderSigningKeyRetention(signer, retainedProofs); err != nil {
		return fmt.Errorf("provider exchange signing-key retention: %w", err)
	}
	return nil
}

func providerWriteJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Referrer-Policy", "no-referrer")
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
	if raw == "" {
		providerWriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "valid provider callback key required"})
		return nil
	}
	resolve := h.resolveProviderKey
	if resolve == nil {
		if h.DB == nil {
			providerWriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "provider key authentication unavailable"})
			return nil
		}
		resolve = models.ResolveProviderAPIKey
	}
	key, err := resolve(h.DB, raw)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			providerWriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "provider key authentication unavailable"})
			return nil
		}
		providerWriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "valid provider callback key required"})
		return nil
	}
	if key == nil {
		providerWriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "valid provider callback key required"})
		return nil
	}
	return key
}

func (h *ProviderExchangeHandler) requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	return requireAdminAPIKey(w, r)
}

func requireAdminAPIKey(w http.ResponseWriter, r *http.Request) bool {
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
	case errors.Is(err, models.ErrProviderPilotStage1NotReady):
		return http.StatusConflict, "Stage 1 demand proof is not ready for a provider pilot"
	case errors.Is(err, models.ErrProviderPilotTopicNotCandidate):
		return http.StatusUnprocessableEntity, "demand topic is not a Stage 1 pilot candidate"
	case errors.Is(err, models.ErrProviderPilotNotDraft),
		errors.Is(err, models.ErrProviderPilotNotActive),
		errors.Is(err, models.ErrProviderPilotCohortFull),
		errors.Is(err, models.ErrProviderPilotCohortNotReady),
		errors.Is(err, models.ErrProviderPilotEnrollmentConflict),
		errors.Is(err, models.ErrProviderPilotEnrollmentNotEligible),
		errors.Is(err, models.ErrProviderPilotTicketCap):
		return http.StatusConflict, err.Error()
	case errors.Is(err, models.ErrProviderPilotReviewSnapshotChanged):
		return http.StatusConflict, "provider pilot review snapshot changed; fetch and review the current candidate"
	case errors.Is(err, models.ErrProviderPilotReviewRequired):
		return http.StatusConflict, "current provider pilot review required before this transition"
	case errors.Is(err, models.ErrProviderProofManifestSnapshotChanged):
		return http.StatusConflict, "provider proof manifest snapshot changed; fetch and review the current candidate"
	case errors.Is(err, models.ErrProviderProofManifestNotIssuable):
		return http.StatusConflict, "provider proof manifest is unavailable until the closed-pilot outcome and chronological review gates pass"
	case errors.Is(err, models.ErrProviderProofManifestRequestConflict):
		return http.StatusConflict, "provider proof manifest already exists for this pilot with different issuance evidence"
	case errors.Is(err, models.ErrProviderProofManifestIntegrity):
		return http.StatusInternalServerError, "stored provider proof manifest failed integrity verification"
	case errors.Is(err, models.ErrProviderCommercialEvidenceRequired):
		return http.StatusConflict, "provider-authenticated and owner-verified commercial evidence is required"
	case errors.Is(err, models.ErrProviderLegacyBudgetMutation):
		return http.StatusConflict, "use the verified commercial funding or reversal workflow for this pilot company"
	case errors.Is(err, models.ErrProviderCommercialLedgerContaminated):
		return http.StatusConflict, "offer contains unverified legacy budget rows; create a clean replacement offer before attaching commercial evidence"
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
