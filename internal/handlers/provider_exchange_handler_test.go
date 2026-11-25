package handlers

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/unitedideas/nothumansearch/internal/models"
	"github.com/unitedideas/nothumansearch/internal/providerexchange"
)

func TestExtractProviderKeyUsesExplicitHeaderBeforeAuthorization(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/provider/outcomes", nil)
	req.Header.Set("X-NHS-Provider-Key", "nhs_provider_explicit")
	req.Header.Set("Authorization", "Bearer nhs_provider_bearer")
	if got := extractProviderKey(req); got != "nhs_provider_explicit" {
		t.Fatalf("extractProviderKey = %q", got)
	}
}

func TestProviderOutcomePayloadHashDoesNotPersistRawToken(t *testing.T) {
	t.Parallel()
	raw := "sensitive-attribution-token"
	got := providerOutcomePayloadHash("ticket", "accepted", raw)
	if len(got) != 64 || got == raw {
		t.Fatalf("payload hash = %q", got)
	}
	if got == providerOutcomePayloadHash("ticket", "activated", raw) {
		t.Fatal("different outcome reused payload hash")
	}
	if got != providerOutcomePayloadHash("ticket", " Accepted ", raw) {
		t.Fatal("semantic outcome retry produced a different payload hash")
	}
}

func TestProviderOutcomeDerivesTicketFromSignedBearer(t *testing.T) {
	t.Parallel()
	signed := "4b69ca8e-d61d-47e2-91dd-fecd9f711234"
	for _, asserted := range []string{"", "  " + strings.ToUpper(signed) + "  "} {
		if got, err := providerOutcomeTicketID(asserted, signed); err != nil || got != signed {
			t.Fatalf("providerOutcomeTicketID(%q) = %q, %v", asserted, got, err)
		}
	}
	if _, err := providerOutcomeTicketID(
		"5c79ca8e-d61d-47e2-91dd-fecd9f711234", signed,
	); err == nil {
		t.Fatal("mismatched compatibility ticket assertion was accepted")
	}
}

func TestProviderOwnershipFreshnessResponseUsesPersistentDNSContract(t *testing.T) {
	t.Parallel()
	lastSucceededAt := time.Unix(1_700_000_000, 0).UTC()
	nextCheckAt := lastSucceededAt.Add(models.ProviderClaimDNSRecheckInterval)
	claim := &models.ProviderClaim{
		VerificationLastSucceededAt:     &lastSucceededAt,
		VerificationNextCheckAt:         &nextCheckAt,
		VerificationConsecutiveFailures: 2,
	}

	got := providerOwnershipFreshness(claim)
	if got.ProofMethod != "dns_txt" || !got.RecordMustRemainPublished ||
		got.StoredChallengeMaterial != "sha256_hash_only" || got.RawDNSAnswersRetained ||
		!got.AutomaticReverification {
		t.Fatalf("ownership freshness persistence contract = %+v", got)
	}
	if got.RecheckIntervalSeconds != int64(models.ProviderClaimDNSRecheckInterval/time.Second) ||
		got.PaidActionsStopAfterConsecutiveFailures != models.ProviderClaimDNSFailureLimit ||
		got.PaidActionsStopWhenLastSuccessAgeReachesSeconds != int64(models.ProviderClaimVerificationFreshness/time.Second) {
		t.Fatalf("ownership freshness thresholds = %+v", got)
	}
	if got.LastSucceededAt == nil || !got.LastSucceededAt.Equal(lastSucceededAt) ||
		got.NextCheckAt == nil || !got.NextCheckAt.Equal(nextCheckAt) || got.ConsecutiveFailures != 2 {
		t.Fatalf("ownership freshness safe claim status = %+v", got)
	}

	response := providerClaimChallengeResponse(&models.ProviderClaim{ID: "claim-id"}, "one-time-token")
	if _, ok := response["ownership_freshness"].(providerOwnershipFreshnessResponse); !ok {
		t.Fatalf("challenge response ownership_freshness = %#v", response["ownership_freshness"])
	}
}

func TestProviderExchangeSignerRequiresDedicatedConfiguration(t *testing.T) {
	t.Setenv("ADMIN_API_KEY", "must-not-be-used-for-provider-signing")
	t.Setenv("NHS_PROVIDER_EXCHANGE_SIGNING_KEY_ID", "")
	t.Setenv("NHS_PROVIDER_EXCHANGE_SIGNING_KEY", "")
	t.Setenv("NHS_PROVIDER_EXCHANGE_PREVIOUS_SIGNING_KEYS_JSON", "")
	if _, err := providerExchangeSignerFromEnv(); err == nil {
		t.Fatal("provider signer accepted ADMIN_API_KEY fallback")
	}
	if err := ValidateProviderExchangeSigningConfiguration(); err == nil {
		t.Fatal("provider signing preflight accepted missing dedicated configuration")
	}

	t.Setenv("NHS_PROVIDER_EXCHANGE_SIGNING_KEY_ID", "nhs-provider-pilot-v1")
	t.Setenv("NHS_PROVIDER_EXCHANGE_SIGNING_KEY", "dedicated-provider-signing-fixture")
	signer, err := providerExchangeSignerFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if signer.ActiveKeyID() != "nhs-provider-pilot-v1" {
		t.Fatalf("active key id = %q", signer.ActiveKeyID())
	}
	if err := ValidateProviderExchangeSigningConfiguration(); err != nil {
		t.Fatalf("valid provider signing preflight failed: %v", err)
	}
}

func TestValidateProviderSigningKeyRetentionRequiresEveryPersistedKey(t *testing.T) {
	t.Parallel()
	activeSecret := strings.Repeat("a", 32)
	previousSecret := strings.Repeat("b", 32)
	signer, err := providerexchange.NewSignerKeyring(
		"nhs-provider-current", activeSecret,
		map[string]string{"nhs-provider-previous": previousSecret},
	)
	if err != nil {
		t.Fatalf("new signer keyring: %v", err)
	}
	issuedAt := time.Unix(1_700_000_000, 0).UTC()
	expiresAt := issuedAt.Add(24 * time.Hour)
	ticketID := "4b69ca8e-d61d-47e2-91dd-fecd9f711234"
	offerID := "a5a84ae1-62aa-4be0-91aa-1ed8a48ed321"
	proofFor := func(keyID, nonce string) models.ProviderSigningKeyProof {
		t.Helper()
		token, err := signer.SignAttribution(providerexchange.AttributionClaims{
			Version: providerexchange.AttributionTokenVersion,
			KeyID:   keyID, TicketID: ticketID, OfferID: offerID,
			IssuedAt: issuedAt.Unix(), ExpiresAt: expiresAt.Unix(), Nonce: nonce,
		})
		if err != nil {
			t.Fatal(err)
		}
		return models.ProviderSigningKeyProof{
			KeyID: keyID, Kind: models.ProviderSigningProofAttribution,
			TicketID: ticketID, OfferID: offerID, IssuedAt: issuedAt, ExpiresAt: expiresAt,
			TokenNonce: nonce, TokenHash: models.HashProviderSecret(token),
		}
	}
	proofs := []models.ProviderSigningKeyProof{
		proofFor("nhs-provider-current", strings.Repeat("c", 32)),
		proofFor("nhs-provider-previous", strings.Repeat("d", 32)),
	}
	if err := validateProviderSigningKeyRetention(signer, proofs); err != nil {
		t.Fatalf("loaded retained signing keys rejected: %v", err)
	}
	missing := proofs[0]
	missing.KeyID = "nhs-provider-retired"
	if err := validateProviderSigningKeyRetention(signer, []models.ProviderSigningKeyProof{missing}); err == nil {
		t.Fatal("missing persisted signing key was accepted")
	} else if strings.Contains(err.Error(), activeSecret) || strings.Contains(err.Error(), previousSecret) {
		t.Fatalf("signing-key retention error exposed secret material: %v", err)
	}
	replacement, err := providerexchange.NewSignerKeyring(
		"nhs-provider-current", strings.Repeat("z", 32), nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateProviderSigningKeyRetention(replacement, proofs[:1]); err == nil {
		t.Fatal("replacement secret under a reused key id was accepted")
	}

	receiptJSON, signature, err := signer.SignOutcomeReceipt(providerexchange.OutcomeReceipt{
		Version: providerexchange.OutcomeReceiptVersion, KeyID: "nhs-provider-current",
		ReceiptID: ticketID, TicketID: ticketID, OfferID: offerID,
		NHSEventID:         "552aee31-02ef-4fe2-94bd-086495341234",
		Outcome:            providerexchange.OutcomeRejected,
		ProviderReportedAt: issuedAt.Unix(), RecordedAt: issuedAt.Unix(),
		ExpiresAt: expiresAt.Unix(), ChargedMinor: 0, Currency: "usd",
		ChargeStatus: providerexchange.ChargeStatusNone,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := validateProviderSigningKeyRetention(signer, []models.ProviderSigningKeyProof{{
		KeyID: "nhs-provider-current", Kind: models.ProviderSigningProofOutcome,
		SignedReceipt: receiptJSON, Signature: signature,
	}}); err != nil {
		t.Fatalf("valid persisted outcome proof rejected: %v", err)
	}
	if err := validateProviderSigningKeyRetention(nil, nil); err == nil {
		t.Fatal("nil signer was accepted")
	}
}

func TestProviderExchangeConstructorChecksPersistedSigningKeyRetention(t *testing.T) {
	t.Parallel()
	source, err := os.ReadFile("provider_exchange_handler.go")
	if err != nil {
		t.Fatalf("read provider exchange handler: %v", err)
	}
	text := string(source)
	start := strings.Index(text, "func NewProviderExchangeHandler")
	end := strings.Index(text, "func validateProviderSigningKeyRetention")
	if start < 0 || end <= start {
		t.Fatal("could not isolate provider exchange constructor")
	}
	constructor := text[start:end]
	for _, required := range []string{
		"models.ProviderSigningKeyProofsInUse(db)",
		"validateProviderSigningKeyRetention(signer, retainedProofs)",
	} {
		if !strings.Contains(constructor, required) {
			t.Fatalf("provider exchange constructor missing %q", required)
		}
	}
}

func TestProtectedMigrationSigningPreflightShares022Transaction(t *testing.T) {
	t.Parallel()
	source, err := os.ReadFile("provider_exchange_handler.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	start := strings.Index(text, "func ProviderExchangeProtectedMigrationPreflight")
	if start < 0 {
		t.Fatal("could not find protected migration signing preflight")
	}
	end := strings.Index(text[start:], "func providerWriteJSON")
	if end < 0 {
		t.Fatal("could not isolate protected migration signing preflight")
	}
	body := text[start : start+end]
	lock := strings.Index(body, "LOCK TABLE public.action_tickets IN ACCESS EXCLUSIVE MODE")
	proofs := strings.Index(body, "models.ProviderSigningKeyProofsInUseTx(ctx, tx)")
	empty := strings.Index(body, "if len(retainedProofs) == 0")
	signer := strings.Index(body, "providerExchangeSignerFromEnv()")
	if lock < 0 || proofs <= lock || empty <= proofs || signer <= empty {
		t.Fatalf("022 signing preflight order lock=%d proofs=%d empty=%d signer=%d", lock, proofs, empty, signer)
	}
}
