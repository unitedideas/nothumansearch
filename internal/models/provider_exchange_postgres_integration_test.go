package models_test

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lib/pq"
	"github.com/unitedideas/nothumansearch/internal/database"
	"github.com/unitedideas/nothumansearch/internal/handlers"
	"github.com/unitedideas/nothumansearch/internal/models"
	"github.com/unitedideas/nothumansearch/internal/providerexchange"
	"github.com/unitedideas/nothumansearch/internal/testpostgres"
)

var postgresProviderPilotEpochID string
var postgresCommercialProviderFixtures map[string]*postgresCommercialProvider

// TestProviderExchangePostgresReleaseRegressions is opt-in because it requires
// an isolated PostgreSQL database that the test may migrate and populate. It
// exercises transaction and PostgreSQL locking behavior that cannot be proved
// by the source-contract unit tests.
func TestProviderExchangePostgresReleaseRegressions(t *testing.T) {
	dsn := testpostgres.DSN(t, "NHS_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set NHS_TEST_POSTGRES_DSN to an isolated disposable PostgreSQL database or set NHS_EMBEDDED_POSTGRES=1")
	}
	t.Setenv("DATABASE_URL", dsn)
	if err := database.Connect(); err != nil {
		t.Fatalf("connect isolated PostgreSQL: %v", err)
	}
	t.Cleanup(func() { _ = database.DB.Close() })
	migrationDir := filepath.Join("..", "..", "migrations")
	const releaseRevision = "1111111111111111111111111111111111111111"
	if err := database.RunMigrations(migrationDir, releaseRevision); err != nil {
		t.Fatalf("apply all migrations through repository runner: %v", err)
	}
	if err := database.RunMigrations(migrationDir, releaseRevision); err != nil {
		t.Fatalf("immediately replay all migrations through repository runner: %v", err)
	}
	db := database.DB

	proofs, err := models.ProviderSigningKeyProofsInUse(db)
	if err != nil {
		t.Fatalf("signing-key proofs on empty provider store: %v", err)
	}
	if len(proofs) != 0 {
		t.Fatalf("signing-key proofs on empty provider store = %d, want 0", len(proofs))
	}

	accountID, site := createPostgresProviderFixture(t, db)
	exerciseStage1FactIntegrityPostgres(t, db, site)
	exerciseActionInterestPostgres(t, db, site)
	claim, rawChallenge, err := models.CreateProviderClaim(db, accountID, site.ID)
	if err != nil {
		t.Fatalf("create provider claim: %v", err)
	}
	claim, err = models.VerifyProviderClaim(db, accountID, claim.ID, rawChallenge)
	if err != nil {
		t.Fatalf("verify provider claim: %v", err)
	}
	_, providerKey, err := models.CreateProviderAPIKey(db, accountID, claim.ID)
	if err != nil {
		t.Fatalf("create provider API key: %v", err)
	}
	company := establishPostgresPilotCompany(t, db, providerKey, "release")
	cohortProviders := preparePostgresCommercialProviderFixtures(t, db)
	postgresProviderPilotEpochID = createPostgresActiveProviderPilot(
		t, db, claim.ID, company, cohortProviders,
	)
	t.Cleanup(func() {
		postgresProviderPilotEpochID = ""
		postgresCommercialProviderFixtures = nil
	})
	signer, err := providerexchange.NewSignerKeyring(
		"pg-release-v1",
		strings.Repeat("s", 32),
		nil,
	)
	if err != nil {
		t.Fatalf("create provider signer: %v", err)
	}

	const bountyCents int64 = 10_000
	zero := int64(0)
	offer, err := models.CreateProviderOffer(db, accountID, claim.ID, models.ProviderOfferInput{
		OfferName:           "PostgreSQL release offer",
		OfferSummary:        "Exercises capped prepaid accounting and late credits.",
		ActionType:          "lead",
		ActionURL:           "https://provider.example/start",
		ChargeEvent:         "accepted",
		BountyCents:         bountyCents,
		Currency:            "usd",
		PrincipalPriceMode:  "free",
		PrincipalPriceCents: &zero,
		PrincipalCurrency:   "usd",
		BillingMode:         "prepaid",
	})
	if err != nil {
		t.Fatalf("create provider offer: %v", err)
	}
	recordPostgresVerifiedFunding(
		t, db, offer.ID, models.ProviderMoneyMaximumCents,
		"release-initial-cap", "", time.Time{},
	)
	if _, err := models.ActivateProviderOffer(
		db, offer.ID, "operator:pg-release:before-review", "evidence:pg-release:before-review",
	); !errors.Is(err, models.ErrProviderPilotReviewRequired) {
		t.Fatalf("activate provider offer before review error = %v, want ErrProviderPilotReviewRequired", err)
	}
	if _, err := db.Exec(`
		UPDATE provider_offers
		SET status='active', provider_pilot_epoch_id=$2::uuid,
		    terms_evidence_reference='evidence:pg-release:before-review',
		    activated_at=clock_timestamp(), updated_at=clock_timestamp()
		WHERE id=$1::uuid`, offer.ID, postgresProviderPilotEpochID); err == nil {
		t.Fatal("database activated provider offer before current offer review")
	} else {
		assertPostgresConstraint(t, err, "provider_offer_activation_review")
	}
	var blockedOfferStatus, blockedOfferPilotID string
	if err := db.QueryRow(`
		SELECT status, COALESCE(provider_pilot_epoch_id::text,'')
		FROM provider_offers WHERE id=$1::uuid`, offer.ID).
		Scan(&blockedOfferStatus, &blockedOfferPilotID); err != nil {
		t.Fatalf("read provider offer after blocked activation: %v", err)
	}
	if blockedOfferStatus != "draft" || blockedOfferPilotID != "" {
		t.Fatalf("provider offer after blocked activation = status:%q pilot:%q, want draft/unbound", blockedOfferStatus, blockedOfferPilotID)
	}
	offer, err = activatePostgresProviderOffer(
		t, db, offer.ID, "operator:pg-release", "evidence:pg-release",
	)
	if err != nil {
		t.Fatalf("activate provider offer: %v", err)
	}
	exerciseProviderPilotTopicIsolationPostgres(t, db, site, offer.ID)
	exerciseProviderPilotSpamReclassificationPostgres(t, db, signer, site, offer.ID)
	exerciseProviderTicketIssuanceAfterRowLock(t, db, signer, providerKey, accountID, claim.ID, site)
	exerciseProviderCapacityReservations(t, db, signer, providerKey, accountID, claim.ID, site)
	expiredHandoffTicketID := exerciseProviderOutcomeAuthorizationAcrossExpiry(
		t, db, signer, providerKey, accountID, claim.ID, site,
	)
	exerciseProviderPilotStatusQueuePostgres(
		t, db, signer, providerKey, company, expiredHandoffTicketID,
	)

	// Organic site membership alone is insufficient. The exact paid
	// offer/version must have been committed as returned evidence before a
	// ticket can be minted.
	missingEvidenceSearchID := recordPostgresDemandReceipt(t, db, site, "missing-evidence")
	_, _, _, err = models.CreateActionTicket(db, models.ActionTicketInput{
		ProviderOfferID:       offer.ID,
		SearchReceiptPublicID: missingEvidenceSearchID,
		DemandTopic:           "developer-tools",
		PrincipalConsent:      true,
		ConsentVersion:        models.ProviderPrincipalConsentV1,
		TTL:                   time.Hour,
	}, signer)
	if !errors.Is(err, models.ErrProviderOfferNotPublic) {
		t.Fatalf("ticket without exact returned-offer evidence error = %v, want ErrProviderOfferNotPublic", err)
	}
	assertPostgresOfferTicketCount(t, db, offer.ID, 0)

	immutableOffer, err := models.CreateProviderOffer(db, accountID, claim.ID, models.ProviderOfferInput{
		OfferName:           "Version-bound disclosure offer",
		OfferSummary:        "Exercises exact paid-offer version evidence.",
		ActionType:          "trial",
		ActionURL:           "https://provider.example/versioned",
		ChargeEvent:         "accepted",
		BountyCents:         bountyCents,
		Currency:            "usd",
		PrincipalPriceMode:  "free",
		PrincipalPriceCents: &zero,
		PrincipalCurrency:   "usd",
		BillingMode:         "prepaid",
	})
	if err != nil {
		t.Fatalf("create version-bound offer: %v", err)
	}
	recordPostgresVerifiedFunding(
		t, db, immutableOffer.ID, bountyCents,
		"release-version-fund", "", time.Time{},
	)
	immutableOffer, err = activatePostgresProviderOffer(
		t, db, immutableOffer.ID, "operator:pg-release", "evidence:pg-release-version",
	)
	if err != nil {
		t.Fatalf("activate version-bound offer: %v", err)
	}
	immutableSearchID := recordPostgresDemandReceipt(t, db, site, "immutable-version")
	recordPostgresReturnedOffer(t, db, site, immutableSearchID, immutableOffer.ID)
	// Migration 022 makes the old stale-version fixture impossible: once either
	// party has committed commercial evidence, direct SQL cannot rewrite any
	// hash-bearing term beneath a returned offer or ticket. Prove that stronger
	// boundary and then prove the original disclosure remains usable.
	if _, err := db.Exec(`
		UPDATE provider_offers
		SET commercial_terms_sha256=$1
		WHERE id=$2::uuid`, strings.Repeat("0", 64), immutableOffer.ID); err == nil {
		t.Fatal("database accepted a commercial hash rewrite after verified funding")
	} else {
		assertPostgresConstraint(t, err, "provider_offer_commercial_immutability")
	}
	immutableTicket, immutableReturnedOffer, _, err := models.CreateActionTicket(db, models.ActionTicketInput{
		ProviderOfferID:       immutableOffer.ID,
		SearchReceiptPublicID: immutableSearchID,
		DemandTopic:           "developer-tools",
		PrincipalConsent:      true,
		ConsentVersion:        models.ProviderPrincipalConsentV1,
		TTL:                   time.Hour,
	}, signer)
	if err != nil {
		t.Fatalf("create ticket after rejected commercial hash rewrite: %v", err)
	}
	if immutableReturnedOffer.OrganicPosition != 1 {
		t.Fatalf("prepared offer organic position=%d, want exact returned position 1", immutableReturnedOffer.OrganicPosition)
	}
	recordPostgresOutcome(
		t, db, signer, providerKey, immutableTicket.ID, "rejected", "immutable-version-release",
	)
	assertPostgresOfferTicketCount(t, db, immutableOffer.ID, 1)

	ticketOne := createPostgresActionTicket(t, db, signer, site, offer.ID, "sequence-one")
	chargedOne := recordPostgresOutcome(t, db, signer, providerKey, ticketOne.ID, "accepted", "sequence-one-charge")
	if chargedOne.ChargeStatus != string(providerexchange.ChargeStatusCharged) || chargedOne.BilledCents != bountyCents {
		t.Fatalf("first charge = status %q cents %d", chargedOne.ChargeStatus, chargedOne.BilledCents)
	}
	assertPostgresProviderBalance(t, db, offer.ID, models.ProviderMoneyMaximumCents-bountyCents)

	if _, err := models.FundProviderOffer(db, offer.ID, bountyCents, "usd", "pg-release-blocked-topup"); !errors.Is(err, models.ErrProviderLegacyBudgetMutation) {
		t.Fatalf("legacy top-up after pilot verification error = %v, want ErrProviderLegacyBudgetMutation", err)
	}
	creditedOne := recordPostgresOutcome(t, db, signer, providerKey, ticketOne.ID, "invalid", "sequence-one-credit")
	if creditedOne.ChargeStatus != string(providerexchange.ChargeStatusCredited) || creditedOne.BilledCents != bountyCents {
		t.Fatalf("first late credit = status %q cents %d", creditedOne.ChargeStatus, creditedOne.BilledCents)
	}
	assertPostgresProviderBalance(t, db, offer.ID, models.ProviderMoneyMaximumCents)

	// Queue a forbidden legacy top-up and late credit at the same time. The
	// verified-company boundary must reject the new legacy mutation while the
	// resolution path remains usable and restores the hard balance cap.
	ticketTwo := createPostgresActionTicket(t, db, signer, site, offer.ID, "concurrent-two")
	recordPostgresOutcome(t, db, signer, providerKey, ticketTwo.ID, "accepted", "concurrent-two-charge")
	assertPostgresProviderBalance(t, db, offer.ID, models.ProviderMoneyMaximumCents-bountyCents)

	type concurrentResult struct {
		operation string
		receipt   *models.OutcomeReceipt
		err       error
	}
	start := make(chan struct{})
	results := make(chan concurrentResult, 2)
	go func() {
		<-start
		_, err := models.FundProviderOffer(db, offer.ID, bountyCents, "usd", "pg-release-concurrent-topup")
		results <- concurrentResult{operation: "topup", err: err}
	}()
	go func() {
		<-start
		receipt, _, err := models.RecordProviderOutcome(db, providerKey, models.ProviderOutcomeInput{
			ActionTicketID: ticketTwo.ID,
			IdempotencyKey: "pg-concurrent-credit-0001",
			PayloadHash:    postgresPayloadHash("concurrent-two-credit"),
			Outcome:        "invalid",
		}, signer)
		results <- concurrentResult{operation: "credit", receipt: receipt, err: err}
	}()
	close(start)
	var topupResult, creditResult concurrentResult
	for range 2 {
		result := <-results
		switch result.operation {
		case "topup":
			topupResult = result
		case "credit":
			creditResult = result
		default:
			t.Fatalf("unknown concurrent result %q", result.operation)
		}
	}
	if !errors.Is(topupResult.err, models.ErrProviderLegacyBudgetMutation) {
		t.Fatalf("concurrent legacy top-up error = %v, want ErrProviderLegacyBudgetMutation", topupResult.err)
	}
	if creditResult.err != nil {
		t.Fatalf("concurrent late credit: %v", creditResult.err)
	}
	if creditResult.receipt == nil ||
		creditResult.receipt.ChargeStatus != string(providerexchange.ChargeStatusCredited) ||
		creditResult.receipt.BilledCents != bountyCents {
		t.Fatalf("concurrent late credit receipt = %#v", creditResult.receipt)
	}
	assertPostgresProviderBalance(t, db, offer.ID, models.ProviderMoneyMaximumCents)

	// A subsequent ticket can still charge, proving the cap/credit sequence did
	// not brick the active offer.
	ticketThree := createPostgresActionTicket(t, db, signer, site, offer.ID, "post-race-three")
	recordPostgresOutcome(t, db, signer, providerKey, ticketThree.ID, "accepted", "post-race-three-charge")
	assertPostgresProviderBalance(t, db, offer.ID, models.ProviderMoneyMaximumCents-bountyCents)

	// Pruning the query-free search receipt must cascade-delete its exact
	// returned-offer evidence while preserving the commercial ticket via SET
	// NULL and redacting every controlled intent field in the same operation.
	retentionSearchID := recordPostgresDemandReceipt(t, db, site, "retention-four")
	recordPostgresReturnedOffer(t, db, site, retentionSearchID, offer.ID)
	retentionTicket, _, _, err := models.CreateActionTicket(db, models.ActionTicketInput{
		ProviderOfferID:       offer.ID,
		SearchReceiptPublicID: retentionSearchID,
		DemandTopic:           "developer-tools",
		RegionCode:            "US-WA",
		BudgetBand:            "500_1999",
		Urgency:               "7_days",
		RequirementFlags:      []string{"api_access", "human_support"},
		PrincipalConsent:      true,
		ConsentVersion:        models.ProviderPrincipalConsentV1,
		TTL:                   time.Hour,
	}, signer)
	if err != nil {
		t.Fatalf("create retention-bound action ticket: %v", err)
	}
	var returnedBeforeDelete int
	if err := db.QueryRow(`
		SELECT COUNT(*)::int
		FROM provider_offers_returned
		WHERE search_receipt_id=$1::uuid`, retentionTicket.SearchReceiptID,
	).Scan(&returnedBeforeDelete); err != nil {
		t.Fatalf("count exact returned-offer evidence before receipt delete: %v", err)
	}
	if returnedBeforeDelete != 1 {
		t.Fatalf("exact returned-offer evidence before receipt delete = %d, want 1", returnedBeforeDelete)
	}
	if _, err := db.Exec(`DELETE FROM search_receipts WHERE public_id=$1`, retentionSearchID); err != nil {
		t.Fatalf("delete retention-bound search receipt: %v", err)
	}
	var (
		searchReceiptCleared bool
		demandTopic          string
		regionCode           string
		budgetBand           string
		urgency              string
		requirementCount     int
		intentRedacted       bool
		principalConsent     bool
		returnedAfterDelete  int
	)
	if err := db.QueryRow(`
		SELECT search_receipt_id IS NULL, demand_topic, region_code,
		       budget_band, urgency, cardinality(requirement_flags),
		       intent_redacted_at IS NOT NULL, principal_consent
		FROM action_tickets WHERE id=$1::uuid`, retentionTicket.ID,
	).Scan(
		&searchReceiptCleared, &demandTopic, &regionCode, &budgetBand,
		&urgency, &requirementCount, &intentRedacted, &principalConsent,
	); err != nil {
		t.Fatalf("read ticket after search-receipt pruning: %v", err)
	}
	if !searchReceiptCleared || demandTopic != "redacted" || regionCode != "" ||
		budgetBand != "unspecified" || urgency != "unspecified" ||
		requirementCount != 0 || !intentRedacted || !principalConsent {
		t.Fatalf(
			"ticket after receipt delete = cleared:%t topic:%q region:%q budget:%q urgency:%q flags:%d redacted:%t consent:%t",
			searchReceiptCleared, demandTopic, regionCode, budgetBand, urgency,
			requirementCount, intentRedacted, principalConsent,
		)
	}
	if err := db.QueryRow(`
		SELECT COUNT(*)::int
		FROM provider_offers_returned
		WHERE search_receipt_id=$1::uuid`, retentionTicket.SearchReceiptID,
	).Scan(&returnedAfterDelete); err != nil {
		t.Fatalf("count exact returned-offer evidence after receipt delete: %v", err)
	}
	if returnedAfterDelete != 0 {
		t.Fatalf("exact returned-offer evidence after receipt delete = %d, want 0", returnedAfterDelete)
	}

	proofs, err = models.ProviderSigningKeyProofsInUse(db)
	if err != nil {
		t.Fatalf("signing-key proofs with persisted tickets and outcomes: %v", err)
	}
	if len(proofs) != 2 {
		t.Fatalf("signing-key proofs = %d, want one attribution and one outcome proof", len(proofs))
	}
	proofKinds := map[string]bool{}
	for _, proof := range proofs {
		if proof.KeyID != signer.ActiveKeyID() {
			t.Fatalf("proof key ID = %q, want %q", proof.KeyID, signer.ActiveKeyID())
		}
		proofKinds[proof.Kind] = true
		switch proof.Kind {
		case models.ProviderSigningProofAttribution:
			token, err := signer.SignAttribution(providerexchange.AttributionClaims{
				Version: providerexchange.AttributionTokenVersion,
				KeyID:   proof.KeyID, TicketID: proof.TicketID, OfferID: proof.OfferID,
				IssuedAt: proof.IssuedAt.Unix(), ExpiresAt: proof.ExpiresAt.Unix(),
				Nonce: proof.TokenNonce,
			})
			if err != nil {
				t.Fatalf("reconstruct attribution proof: %v", err)
			}
			if models.HashProviderSecret(token) != proof.TokenHash {
				t.Fatal("reconstructed attribution proof does not match persisted token hash")
			}
		case models.ProviderSigningProofOutcome:
			if _, err := signer.VerifyOutcomeReceiptSignature(proof.SignedReceipt, proof.Signature); err != nil {
				t.Fatalf("verify persisted outcome proof: %v", err)
			}
		default:
			t.Fatalf("unexpected signing proof kind %q", proof.Kind)
		}
	}
	if !proofKinds[models.ProviderSigningProofAttribution] || !proofKinds[models.ProviderSigningProofOutcome] {
		t.Fatalf("signing proof kinds = %#v", proofKinds)
	}

	// A legacy/directly persisted Unicode URL can be within the database's raw
	// byte cap yet expand beyond the final 2048-byte URL after percent encoding.
	// Ticket creation must preflight the exact final representation before its
	// transaction commits.
	unicodeURL := "https://provider.example/" + repeatPostgresRune('é', 400)
	unicodeInput := models.ProviderOfferInput{
		OfferName:           "Unicode expansion offer",
		OfferSummary:        "Exercises final attributed URL length preflight.",
		ActionType:          "demo",
		ActionURL:           "https://provider.example/unicode",
		ChargeEvent:         "accepted",
		BountyCents:         bountyCents,
		Currency:            "usd",
		PrincipalPriceMode:  "free",
		PrincipalPriceCents: &zero,
		PrincipalCurrency:   "usd",
		BillingMode:         "prepaid",
	}
	unicodeOffer, err := models.CreateProviderOffer(db, accountID, claim.ID, unicodeInput)
	if err != nil {
		t.Fatalf("persist raw-byte-bounded Unicode offer: %v", err)
	}
	// Simulate a legacy/imported draft that predates normalized-length
	// validation. Migration 022 requires its canonical commercial hash to move
	// with the draft before either party can commit evidence.
	unicodeInput.ActionURL = unicodeURL
	if _, err := db.Exec(`
		UPDATE provider_offers
		SET action_url=$1, commercial_terms_sha256=$2
		WHERE id=$3::uuid`, unicodeURL,
		postgresCommercialTermsHash(unicodeOffer.ID, unicodeOffer.Version, unicodeInput),
		unicodeOffer.ID); err != nil {
		t.Fatalf("persist legacy Unicode action URL and exact terms hash: %v", err)
	}
	recordPostgresVerifiedFunding(
		t, db, unicodeOffer.ID, bountyCents,
		"release-unicode-fund", "", time.Time{},
	)
	if _, err := activatePostgresProviderOffer(
		t, db, unicodeOffer.ID, "operator:pg-release", "evidence:pg-release-unicode",
	); err != nil {
		t.Fatalf("activate Unicode offer: %v", err)
	}
	searchID := recordPostgresDemandReceipt(t, db, site, "unicode-four")
	recordPostgresReturnedOffer(t, db, site, searchID, unicodeOffer.ID)
	_, _, _, err = models.CreateActionTicket(db, models.ActionTicketInput{
		ProviderOfferID:       unicodeOffer.ID,
		SearchReceiptPublicID: searchID,
		DemandTopic:           "developer-tools",
		PrincipalConsent:      true,
		ConsentVersion:        models.ProviderPrincipalConsentV1,
		TTL:                   time.Hour,
	}, signer)
	if !errors.Is(err, models.ErrInvalidProviderExchange) {
		t.Fatalf("Unicode-expanded action URL ticket error = %v, want ErrInvalidProviderExchange", err)
	}
	var unicodeTicketCount int
	if err := db.QueryRow(
		`SELECT COUNT(*)::int FROM action_tickets WHERE provider_offer_id=$1::uuid`,
		unicodeOffer.ID,
	).Scan(&unicodeTicketCount); err != nil {
		t.Fatalf("count Unicode offer tickets: %v", err)
	}
	if unicodeTicketCount != 0 {
		t.Fatalf("Unicode-expanded action URL left %d committed ticket(s), want 0", unicodeTicketCount)
	}

	exerciseProviderActionHandoffPostgres(t, db, signer)
	exerciseProviderCommercialProofPostgres(t, db, signer)
	exerciseProviderExchangeHTTPLoopback(t, db)
	exerciseProviderPilotReviewEvidencePostgres(t, db, signer)
	exerciseProviderPilotCloseBoundaryPostgres(
		t, db, signer, providerKey, site, offer.ID,
	)
	exerciseProviderProofManifestPostgres(t, db, signer)
	exerciseProviderPilotLifecycleEventCompletenessPostgres(t, db)
}

func exerciseProviderProofManifestPostgres(
	t *testing.T,
	db *sql.DB,
	signer *providerexchange.Signer,
) {
	t.Helper()
	const bounty int64 = 1_000
	providers := make([]*postgresCommercialProvider, 0, models.ProviderPilotMinimumCohort)
	for index := 0; index < models.ProviderPilotMinimumCohort; index++ {
		provider := createPostgresCommercialProviderDirect(
			t, db, "manifest-"+string(rune('a'+index)),
		)
		recordPostgresStage1EligibilityReceipt(
			t, db, provider.site, fmt.Sprintf("manifest-%02d", index+1),
		)
		providers = append(providers, provider)
	}
	epoch, err := models.CreateProviderPilotEpoch(db, models.ProviderPilotEpochInput{
		DemandTopic:       "developer-tools",
		CohortLimit:       models.ProviderPilotMinimumCohort,
		ProviderTicketCap: 3,
		TotalTicketCap:    models.ProviderPilotMinimumTotalTickets,
		OwnerReference:    "owner:manifest-pilot",
		EvidenceReference: "evidence:manifest-pilot",
	})
	if err != nil {
		t.Fatalf("create proof-manifest pilot: %v", err)
	}
	review := func(reviewType, subjectID string) *models.ProviderPilotReviewEvent {
		t.Helper()
		candidate, err := models.GetProviderPilotReviewCandidate(
			db, epoch.ID, reviewType, subjectID,
		)
		if err != nil {
			t.Fatalf("get proof-manifest %s review candidate: %v", reviewType, err)
		}
		event, created, err := models.RecordProviderPilotReview(
			db,
			models.ProviderPilotReviewInput{
				ProviderPilotEpochID:   epoch.ID,
				ReviewType:             reviewType,
				SubjectID:              subjectID,
				ExpectedSnapshotSHA256: candidate.SubjectSnapshotSHA256,
				OwnerReference:         "owner:manifest:" + reviewType,
				EvidenceReference:      "evidence:manifest:" + reviewType + ":" + subjectID[:8],
			},
		)
		if err != nil || !created {
			t.Fatalf("record proof-manifest %s review = event:%#v created:%t err:%v", reviewType, event, created, err)
		}
		return event
	}
	for _, provider := range providers {
		if _, err := models.EnrollProviderPilotCompany(db, models.ProviderPilotEnrollmentInput{
			ProviderPilotEpochID: epoch.ID,
			ProviderClaimID:      provider.claim.ID,
			OwnerReference:       "owner:manifest:enroll",
			EvidenceReference:    "evidence:manifest:enroll:" + provider.claim.ID[:8],
		}); err != nil {
			t.Fatalf("enroll proof-manifest provider: %v", err)
		}
		review("provider", provider.claim.ID)
	}
	epoch, err = models.ActivateProviderPilotEpoch(db, models.ProviderPilotEpochActionInput{
		ProviderPilotEpochID: epoch.ID,
		OwnerReference:       "owner:manifest:activate",
		EvidenceReference:    "evidence:manifest:activate",
	})
	if err != nil {
		t.Fatalf("activate proof-manifest pilot: %v", err)
	}

	offers := make([]*models.ProviderOffer, 0, len(providers))
	initialAcceptances := make([]*models.ProviderCommercialAcceptanceEvent, 0, len(providers))
	initialTerms := make([]*models.ProviderCommercialCommitmentEvent, 0, len(providers))
	for index, provider := range providers[:3] {
		suffix := "manifest-" + string(rune('a'+index))
		offer := createPostgresCommercialOffer(t, db, provider, suffix, "terms", bounty)
		acceptance, terms := recordPostgresVerifiedTerms(
			t, db, provider.key, offer.ID, suffix+"-terms", "", "",
		)
		review("offer", offer.ID)
		if _, err := models.ActivateProviderOffer(
			db, offer.ID, "owner:"+suffix, "evidence:"+suffix,
		); err != nil {
			t.Fatalf("activate proof-manifest offer %q: %v", suffix, err)
		}
		offers = append(offers, offer)
		initialAcceptances = append(initialAcceptances, acceptance)
		initialTerms = append(initialTerms, terms)
	}

	ticketsByProvider := []int{2, 2, 1}
	acceptedTickets := make([]*models.ActionTicket, 0, 5)
	callbackCount := 0
	for providerIndex, ticketCount := range ticketsByProvider {
		for ticketIndex := 0; ticketIndex < ticketCount; ticketIndex++ {
			suffix := "manifest-ticket-" + string(rune('a'+providerIndex)) + string(rune('1'+ticketIndex))
			ticket, rawToken := createPostgresUnhandedActionTicket(
				t, db, signer, providers[providerIndex].site, offers[providerIndex].ID, suffix,
			)
			review("ticket", ticket.ID)
			_, handoff, err := models.RecordActionTicketHandoff(db, models.ProviderActionHandoffInput{
				ActionTicketID:                             ticket.ID,
				AttributionToken:                           rawToken,
				PrincipalHandoffConsent:                    true,
				HandoffConsentVersion:                      models.ProviderActionHandoffConsentV1,
				PrincipalControlledIntentDisclosureConsent: true,
				ControlledIntentDisclosureConsentVersion:   models.ProviderControlledIntentDisclosureConsentV1,
			})
			if err != nil {
				t.Fatalf("record proof-manifest handoff %q: %v", suffix, err)
			}
			review("handoff", handoff.ID)
			accepted := recordPostgresOutcome(
				t, db, signer, providers[providerIndex].key, ticket.ID,
				"accepted", suffix+"-accepted",
			)
			review("callback", accepted.ID)
			callbackCount++
			acceptedTickets = append(acceptedTickets, ticket)
			if providerIndex == 0 {
				activated := recordPostgresOutcome(
					t, db, signer, providers[providerIndex].key, ticket.ID,
					"activated", suffix+"-activated",
				)
				review("callback", activated.ID)
				callbackCount++
			}
		}
	}
	if len(acceptedTickets) != 5 || callbackCount != 7 {
		t.Fatalf("proof-manifest fixture counts = tickets:%d callbacks:%d, want 5/7", len(acceptedTickets), callbackCount)
	}
	recordPostgresVerifiedTerms(
		t, db, providers[0].key, offers[0].ID, "manifest-a-renewal",
		initialAcceptances[0].ID, initialTerms[0].ID,
	)

	activeCandidate, err := models.GetProviderCommercialProofManifestCandidate(db, epoch.ID, signer)
	if err != nil {
		t.Fatalf("preview active proof manifest: %v", err)
	}
	if activeCandidate.Issuable || !reflect.DeepEqual(
		activeCandidate.IssuanceBlockers, []string{"pilot_not_closed"},
	) {
		t.Fatalf("active proof-manifest candidate=%#v", activeCandidate)
	}
	if _, _, err := models.IssueProviderCommercialProofManifest(
		db,
		models.ProviderCommercialProofManifestInput{
			ProviderPilotEpochID:   epoch.ID,
			ExpectedSnapshotSHA256: activeCandidate.ProofSnapshotSHA256,
			OwnerReference:         "owner:manifest:issue",
			EvidenceReference:      "evidence:manifest:issue",
		},
		signer,
	); !errors.Is(err, models.ErrProviderProofManifestNotIssuable) {
		t.Fatalf("active-pilot manifest issue error=%v, want not issuable", err)
	}

	closed, err := models.CloseProviderPilotEpoch(db, models.ProviderPilotEpochActionInput{
		ProviderPilotEpochID: epoch.ID,
		OwnerReference:       "owner:manifest:close",
		EvidenceReference:    "evidence:manifest:close",
	})
	if err != nil || closed.Status != "closed" {
		t.Fatalf("close proof-manifest pilot = epoch:%#v err:%v", closed, err)
	}
	candidate, err := models.GetProviderCommercialProofManifestCandidate(db, epoch.ID, signer)
	if err != nil {
		t.Fatalf("preview closed proof manifest: %v", err)
	}
	if !candidate.Issuable || len(candidate.IssuanceBlockers) != 0 ||
		!candidate.OutcomeReceiptIntegrityValid || !candidate.ReviewIntegrityValid ||
		candidate.SignatureVerificationScope != providerexchange.CommercialProofVerificationScopeV1 ||
		candidate.ReviewEvidenceContractVersion != providerexchange.CommercialProofReviewEvidenceV1 ||
		candidate.MarketPolicyContractVersion != providerexchange.CommercialProofMarketPolicyV1 ||
		len(candidate.ReviewEvidenceSHA256) != 64 ||
		!candidate.MonetaryAmountsWithheldForPrivacy ||
		len(candidate.VerifiedPrepaidSettled) != 0 ||
		len(candidate.VerifiedPrepaidNetDebited) != 0 ||
		len(candidate.VerifiedTermsNetReceivable) != 0 ||
		!candidate.PilotThresholdsMet || candidate.VerifiedProviderCompanies != 3 ||
		candidate.VerifiedProviderAcceptedHandoffs != 5 ||
		candidate.VerifiedProviderActivations != 2 ||
		candidate.VerifiedProviderRenewals != 1 ||
		candidate.ReviewCoverage.Providers.Required != 3 ||
		candidate.ReviewCoverage.Offers.Required != 3 ||
		candidate.ReviewCoverage.Tickets.Required != 5 ||
		candidate.ReviewCoverage.Handoffs.Required != 5 ||
		candidate.ReviewCoverage.Callbacks.Required != 7 ||
		candidate.ReviewCoverage.Providers.Valid != 3 ||
		candidate.ReviewCoverage.Offers.Valid != 3 ||
		candidate.ReviewCoverage.Tickets.Valid != 5 ||
		candidate.ReviewCoverage.Handoffs.Valid != 5 ||
		candidate.ReviewCoverage.Callbacks.Valid != 7 {
		t.Fatalf("closed proof-manifest candidate=%#v", candidate)
	}
	poisonManifest := providerexchange.CommercialProofManifest{
		Version:                           providerexchange.CommercialProofManifestVersion,
		KeyID:                             "pg-release-v1",
		SignatureVerificationScope:        candidate.SignatureVerificationScope,
		ManifestContractVersion:           candidate.ManifestContractVersion,
		ManifestID:                        "00000000-0000-4000-8000-000000000028",
		ProviderPilotEpochID:              candidate.ProviderPilotEpochID,
		ProviderPilotContractVersion:      candidate.ProviderPilotContractVersion,
		ReviewContractVersion:             candidate.ReviewContractVersion,
		ReviewEvidenceContractVersion:     candidate.ReviewEvidenceContractVersion,
		MarketPolicyContractVersion:       candidate.MarketPolicyContractVersion,
		ProofSnapshotSHA256:               candidate.ProofSnapshotSHA256,
		ReviewEvidenceSHA256:              candidate.ReviewEvidenceSHA256,
		PilotDemandTopic:                  candidate.PilotDemandTopic,
		PilotStatus:                       candidate.PilotStatus,
		IssuedAt:                          time.Now().UTC().Unix(),
		OutcomeReceiptIntegrityValid:      candidate.OutcomeReceiptIntegrityValid,
		ReviewIntegrityValid:              candidate.ReviewIntegrityValid,
		VerifiedOutcomeReceipts:           candidate.VerifiedOutcomeReceipts,
		RejectedOutcomeReceipts:           candidate.RejectedOutcomeReceipts,
		VerifiedOutcomeLedgerEntries:      candidate.VerifiedOutcomeLedgerEntries,
		RejectedOutcomeLedgerEntries:      candidate.RejectedOutcomeLedgerEntries,
		VerifiedProviderCompanies:         candidate.VerifiedProviderCompanies,
		VerifiedProviderAcceptedHandoffs:  candidate.VerifiedProviderAcceptedHandoffs,
		VerifiedProviderActivations:       candidate.VerifiedProviderActivations,
		VerifiedProviderRenewals:          candidate.VerifiedProviderRenewals,
		VerifiedProviderConversions:       candidate.VerifiedProviderConversions,
		ReviewCoverage:                    candidate.ReviewCoverage,
		MonetaryAmountsWithheldForPrivacy: candidate.MonetaryAmountsWithheldForPrivacy,
		VerifiedPrepaidSettled:            candidate.VerifiedPrepaidSettled,
		VerifiedPrepaidNetDebited:         candidate.VerifiedPrepaidNetDebited,
		VerifiedTermsNetReceivable:        candidate.VerifiedTermsNetReceivable,
		PilotThresholdsMet:                candidate.PilotThresholdsMet,
		OrganicRankSold:                   false,
		RawQueriesSold:                    false,
		AgentIdentitiesSold:               false,
		EvidenceScope:                     candidate.EvidenceScope,
	}
	poisonCanonical, poisonSignature, err := signer.SignCommercialProofManifest(poisonManifest)
	if err != nil {
		t.Fatalf("sign proof-manifest poison-control fixture: %v", err)
	}
	var poisonDocument map[string]any
	if err := json.Unmarshal([]byte(poisonCanonical), &poisonDocument); err != nil {
		t.Fatalf("decode proof-manifest poison-control fixture: %v", err)
	}
	poisonCases := []struct {
		name       string
		constraint string
		mutate     func(map[string]any)
	}{
		{
			name:       "extra private field",
			constraint: "provider_proof_manifest_json_shape",
			mutate: func(document map[string]any) {
				document["private_provider_id"] = providers[0].claim.ID
			},
		},
		{
			name:       "wrong top-level type",
			constraint: "provider_proof_manifest_json_types",
			mutate: func(document map[string]any) {
				document["v"] = "1"
			},
		},
		{
			name:       "wrong nested review type",
			constraint: "provider_proof_manifest_review_shape",
			mutate: func(document map[string]any) {
				coverage := document["review_coverage"].(map[string]any)
				providersCoverage := coverage["providers"].(map[string]any)
				providersCoverage["required"] = "3"
			},
		},
	}
	for _, testCase := range poisonCases {
		t.Run("proof manifest rejects direct SQL "+testCase.name, func(t *testing.T) {
			encoded, err := json.Marshal(poisonDocument)
			if err != nil {
				t.Fatalf("copy proof-manifest poison-control fixture: %v", err)
			}
			var document map[string]any
			if err := json.Unmarshal(encoded, &document); err != nil {
				t.Fatalf("decode proof-manifest poison case: %v", err)
			}
			testCase.mutate(document)
			malformed, err := json.Marshal(document)
			if err != nil {
				t.Fatalf("encode proof-manifest poison case: %v", err)
			}
			var manifestID string
			if err := db.QueryRow(`SELECT uuid_generate_v4()::text`).Scan(&manifestID); err != nil {
				t.Fatalf("create proof-manifest poison ID: %v", err)
			}
			issuedAt := time.Now().UTC().Truncate(time.Second)
			_, insertErr := db.Exec(`
				INSERT INTO provider_commercial_proof_manifests (
					id, provider_pilot_epoch_id, manifest_contract_version,
					proof_snapshot_sha256, review_evidence_sha256, key_id,
					signed_manifest, signature, payload_sha256,
					owner_reference, evidence_reference, issued_at, created_at
				) VALUES (
					$1::uuid, $2::uuid, $3, $4, $5, $6,
					$7, $8, $9, $10, $11, $12, $12
				)`,
				manifestID, epoch.ID, candidate.ManifestContractVersion,
				candidate.ProofSnapshotSHA256, candidate.ReviewEvidenceSHA256,
				"pg-release-v1", string(malformed), poisonSignature,
				strings.Repeat("0", 64), "owner:manifest:poison-control",
				"evidence:manifest:poison-control", issuedAt,
			)
			if insertErr == nil {
				t.Fatalf("direct SQL accepted malformed proof manifest: %s", malformed)
			}
			var pqErr *pq.Error
			if !errors.As(insertErr, &pqErr) || pqErr.Code != "23514" ||
				pqErr.Constraint != testCase.constraint {
				t.Fatalf("direct SQL malformed proof manifest error=%v, want 23514/%s", insertErr, testCase.constraint)
			}
		})
	}
	var poisonRows int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM provider_commercial_proof_manifests
		WHERE provider_pilot_epoch_id=$1::uuid`, epoch.ID).Scan(&poisonRows); err != nil {
		t.Fatalf("count rejected proof-manifest poison rows: %v", err)
	}
	if poisonRows != 0 {
		t.Fatalf("malformed direct SQL consumed %d immutable proof-manifest rows", poisonRows)
	}
	if _, _, err := models.IssueProviderCommercialProofManifest(
		db,
		models.ProviderCommercialProofManifestInput{
			ProviderPilotEpochID:   epoch.ID,
			ExpectedSnapshotSHA256: strings.Repeat("0", 64),
			OwnerReference:         "owner:manifest:issue",
			EvidenceReference:      "evidence:manifest:issue",
		},
		signer,
	); !errors.Is(err, models.ErrProviderProofManifestSnapshotChanged) {
		t.Fatalf("stale proof-manifest digest error=%v, want snapshot changed", err)
	}
	input := models.ProviderCommercialProofManifestInput{
		ProviderPilotEpochID:   epoch.ID,
		ExpectedSnapshotSHA256: candidate.ProofSnapshotSHA256,
		OwnerReference:         "owner:manifest:issue",
		EvidenceReference:      "evidence:manifest:issue",
	}
	const manifestAdminKey = "manifest-loopback-admin-not-production"
	t.Setenv("ADMIN_API_KEY", manifestAdminKey)
	manifestHandler := &handlers.ProviderExchangeHandler{DB: db, Signer: signer}
	callManifestAdmin := func(method, target string, body any) *httptest.ResponseRecorder {
		t.Helper()
		var encodedBody *strings.Reader
		if body == nil {
			encodedBody = strings.NewReader("")
		} else {
			encoded, encodeErr := json.Marshal(body)
			if encodeErr != nil {
				t.Fatalf("encode proof-manifest admin request: %v", encodeErr)
			}
			encodedBody = strings.NewReader(string(encoded))
		}
		request := httptest.NewRequest(method, target, encodedBody)
		request.Header.Set("Authorization", "Bearer "+manifestAdminKey)
		if body != nil {
			request.Header.Set("Content-Type", "application/json")
		}
		response := httptest.NewRecorder()
		manifestHandler.AdminProviderProofManifest(response, request)
		return response
	}
	previewHTTP := callManifestAdmin(
		http.MethodGet,
		"/api/v1/admin/provider-proof-manifest?pilot_id="+epoch.ID,
		nil,
	)
	if previewHTTP.Code != http.StatusOK {
		t.Fatalf("preview proof manifest over HTTP = %d %s", previewHTTP.Code, previewHTTP.Body.String())
	}
	var previewEnvelope struct {
		Candidate               models.ProviderCommercialProofManifestCandidate `json:"manifest_candidate"`
		Issued                  bool                                            `json:"issued"`
		CommercialProofCreated  bool                                            `json:"commercial_proof_created"`
		PubliclyReleased        bool                                            `json:"publicly_released"`
		IndependentlyVerifiable bool                                            `json:"independently_verifiable"`
		EvidenceScope           string                                          `json:"evidence_scope"`
	}
	if err := json.Unmarshal(previewHTTP.Body.Bytes(), &previewEnvelope); err != nil {
		t.Fatalf("decode proof-manifest HTTP preview: %v", err)
	}
	if previewEnvelope.Issued || previewEnvelope.CommercialProofCreated ||
		previewEnvelope.PubliclyReleased || previewEnvelope.IndependentlyVerifiable ||
		!previewEnvelope.Candidate.Issuable ||
		previewEnvelope.Candidate.ProofSnapshotSHA256 != candidate.ProofSnapshotSHA256 ||
		!strings.Contains(previewEnvelope.EvidenceScope, "not independent provider truth") {
		t.Fatalf("proof-manifest HTTP preview overstated or changed evidence: %#v", previewEnvelope)
	}
	gateTx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin proof-manifest concurrency gate: %v", err)
	}
	defer gateTx.Rollback()
	var lockedPilotID string
	if err := gateTx.QueryRow(`
		SELECT id::text FROM provider_pilot_epochs
		WHERE id=$1::uuid FOR UPDATE`, epoch.ID).Scan(&lockedPilotID); err != nil {
		t.Fatalf("lock proof-manifest pilot concurrency gate: %v", err)
	}
	type manifestIssueResult struct {
		record  *models.ProviderCommercialProofManifestRecord
		created bool
		err     error
	}
	issueResults := make(chan manifestIssueResult, 2)
	var issueWait sync.WaitGroup
	for range 2 {
		issueWait.Add(1)
		go func() {
			defer issueWait.Done()
			record, created, issueErr := models.IssueProviderCommercialProofManifest(
				db, input, signer,
			)
			issueResults <- manifestIssueResult{
				record: record, created: created, err: issueErr,
			}
		}()
	}
	waitForProviderManifestIssueWaiters(t, db, 2)
	if err := gateTx.Commit(); err != nil {
		t.Fatalf("release proof-manifest concurrency gate: %v", err)
	}
	issueWait.Wait()
	close(issueResults)
	var record, concurrentReplay *models.ProviderCommercialProofManifestRecord
	for result := range issueResults {
		if result.err != nil {
			t.Fatalf("concurrent proof-manifest issue = record:%#v created:%t err:%v",
				result.record, result.created, result.err)
		}
		if result.created {
			if record != nil {
				t.Fatal("concurrent proof-manifest issuance created two records")
			}
			record = result.record
		} else {
			if concurrentReplay != nil {
				t.Fatal("concurrent proof-manifest issuance returned two replays")
			}
			concurrentReplay = result.record
		}
	}
	if record == nil || record.Replayed || concurrentReplay == nil ||
		!concurrentReplay.Replayed || concurrentReplay.ID != record.ID ||
		concurrentReplay.SignedManifest != record.SignedManifest ||
		concurrentReplay.Signature != record.Signature ||
		concurrentReplay.PayloadSHA256 != record.PayloadSHA256 {
		t.Fatalf("concurrent proof-manifest results = created:%#v replay:%#v", record, concurrentReplay)
	}
	var manifestRows int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM provider_commercial_proof_manifests
		WHERE provider_pilot_epoch_id=$1::uuid`, epoch.ID).Scan(&manifestRows); err != nil {
		t.Fatalf("count concurrent proof-manifest rows: %v", err)
	}
	if manifestRows != 1 {
		t.Fatalf("concurrent proof-manifest issuance stored %d rows, want 1", manifestRows)
	}

	issueHTTP := callManifestAdmin(
		http.MethodPost,
		"/api/v1/admin/provider-proof-manifest",
		map[string]string{
			"provider_pilot_epoch_id":  input.ProviderPilotEpochID,
			"expected_snapshot_sha256": input.ExpectedSnapshotSHA256,
			"owner_reference":          input.OwnerReference,
			"evidence_reference":       input.EvidenceReference,
		},
	)
	if issueHTTP.Code != http.StatusOK {
		t.Fatalf("replay proof manifest over HTTP = %d %s", issueHTTP.Code, issueHTTP.Body.String())
	}
	var issueEnvelope struct {
		Manifest                models.ProviderCommercialProofManifestRecord `json:"manifest"`
		Created                 bool                                         `json:"created"`
		IdempotentReplay        bool                                         `json:"idempotent_replay"`
		CommercialProofCreated  bool                                         `json:"commercial_proof_created"`
		PubliclyReleased        bool                                         `json:"publicly_released"`
		IndependentlyVerifiable bool                                         `json:"independently_verifiable"`
	}
	if err := json.Unmarshal(issueHTTP.Body.Bytes(), &issueEnvelope); err != nil {
		t.Fatalf("decode proof-manifest HTTP replay: %v", err)
	}
	if issueEnvelope.Created || !issueEnvelope.IdempotentReplay ||
		!issueEnvelope.CommercialProofCreated || issueEnvelope.PubliclyReleased ||
		issueEnvelope.IndependentlyVerifiable ||
		issueEnvelope.Manifest.ID != record.ID ||
		issueEnvelope.Manifest.Signature != record.Signature {
		t.Fatalf("proof-manifest HTTP replay response=%#v", issueEnvelope)
	}
	manifest, err := signer.VerifyCommercialProofManifestSignature(
		record.SignedManifest, record.Signature,
	)
	if err != nil || manifest.ManifestID != record.ID ||
		manifest.ProofSnapshotSHA256 != candidate.ProofSnapshotSHA256 ||
		manifest.ReviewEvidenceSHA256 != candidate.ReviewEvidenceSHA256 ||
		manifest.SignatureVerificationScope != providerexchange.CommercialProofVerificationScopeV1 ||
		manifest.MarketPolicyContractVersion != providerexchange.CommercialProofMarketPolicyV1 ||
		!manifest.MonetaryAmountsWithheldForPrivacy ||
		len(manifest.VerifiedPrepaidSettled) != 0 ||
		len(manifest.VerifiedPrepaidNetDebited) != 0 ||
		len(manifest.VerifiedTermsNetReceivable) != 0 ||
		manifest.VerifiedProviderCompanies != 3 ||
		manifest.VerifiedProviderAcceptedHandoffs != 5 ||
		manifest.VerifiedProviderActivations != 2 ||
		manifest.VerifiedProviderRenewals != 1 || manifest.OrganicRankSold ||
		manifest.RawQueriesSold || manifest.AgentIdentitiesSold {
		t.Fatalf("verify issued proof manifest = manifest:%#v err:%v", manifest, err)
	}
	lowerManifest := strings.ToLower(record.SignedManifest)
	for _, forbidden := range []string{
		`"provider_claim_id"`, `"provider_offer_id"`, `"action_ticket_id"`,
		`"handoff_receipt_id"`, `"outcome_receipt_id"`, `"search_receipt"`,
		`"query"`, `"principal"`, `"agent_id"`, `"owner_reference"`,
		`"evidence_reference"`, `"signed_receipt"`, `"token"`, `"bearer"`,
	} {
		if strings.Contains(lowerManifest, forbidden) {
			t.Fatalf("issued proof manifest exposed forbidden field %q: %s", forbidden, record.SignedManifest)
		}
	}
	replayed, replayCreated, err := models.IssueProviderCommercialProofManifest(db, input, signer)
	if err != nil || replayCreated || !replayed.Replayed ||
		replayed.ID != record.ID || replayed.Signature != record.Signature {
		t.Fatalf("replay proof manifest = record:%#v created:%t err:%v", replayed, replayCreated, err)
	}
	existingHTTP := callManifestAdmin(
		http.MethodGet,
		"/api/v1/admin/provider-proof-manifest?pilot_id="+epoch.ID,
		nil,
	)
	if existingHTTP.Code != http.StatusOK {
		t.Fatalf("read issued proof manifest over HTTP = %d %s", existingHTTP.Code, existingHTTP.Body.String())
	}
	var existingEnvelope struct {
		Manifest                models.ProviderCommercialProofManifestRecord `json:"manifest"`
		Issued                  bool                                         `json:"issued"`
		CommercialProofCreated  bool                                         `json:"commercial_proof_created"`
		PubliclyReleased        bool                                         `json:"publicly_released"`
		IndependentlyVerifiable bool                                         `json:"independently_verifiable"`
	}
	if err := json.Unmarshal(existingHTTP.Body.Bytes(), &existingEnvelope); err != nil {
		t.Fatalf("decode issued proof-manifest HTTP readback: %v", err)
	}
	if !existingEnvelope.Issued || !existingEnvelope.CommercialProofCreated ||
		existingEnvelope.PubliclyReleased || existingEnvelope.IndependentlyVerifiable ||
		existingEnvelope.Manifest.ID != record.ID ||
		existingEnvelope.Manifest.Signature != record.Signature ||
		strings.Contains(existingHTTP.Body.String(), "owner:manifest:issue") ||
		strings.Contains(existingHTTP.Body.String(), "evidence:manifest:issue") {
		t.Fatalf("issued proof-manifest HTTP readback changed scope or exposed internal refs: %#v", existingEnvelope)
	}
	conflict := input
	conflict.EvidenceReference = "evidence:manifest:different"
	if _, _, err := models.IssueProviderCommercialProofManifest(db, conflict, signer); !errors.Is(err, models.ErrProviderProofManifestRequestConflict) {
		t.Fatalf("conflicting proof-manifest replay error=%v, want request conflict", err)
	}
	readBack, err := models.GetProviderCommercialProofManifest(db, epoch.ID, signer)
	if err != nil || readBack.ID != record.ID || readBack.Signature != record.Signature {
		t.Fatalf("read proof manifest = record:%#v err:%v", readBack, err)
	}
	if result, err := db.Exec(`
		UPDATE provider_commercial_proof_manifests
		SET owner_reference='owner:manifest:rewritten'
		WHERE id=$1::uuid`, record.ID); err != nil {
		t.Fatalf("append-only manifest update: %v", err)
	} else if affected, _ := result.RowsAffected(); affected != 0 {
		t.Fatalf("append-only manifest update affected %d rows", affected)
	}
	if result, err := db.Exec(`
		DELETE FROM provider_commercial_proof_manifests WHERE id=$1::uuid`, record.ID); err != nil {
		t.Fatalf("append-only manifest delete: %v", err)
	} else if affected, _ := result.RowsAffected(); affected != 0 {
		t.Fatalf("append-only manifest delete affected %d rows", affected)
	}
	retained, err := models.ProviderSigningKeyProofsInUse(db)
	if err != nil {
		t.Fatalf("read signing retention with proof manifest: %v", err)
	}
	foundManifestProof := false
	for _, proof := range retained {
		if proof.Kind == models.ProviderSigningProofManifest && proof.KeyID == manifest.KeyID {
			verified, verifyErr := signer.VerifyCommercialProofManifestSignature(
				proof.SignedManifest, proof.Signature,
			)
			if verifyErr != nil || verified.ManifestID != record.ID {
				t.Fatalf("retained manifest proof = manifest:%#v err:%v", verified, verifyErr)
			}
			foundManifestProof = true
		}
	}
	if !foundManifestProof {
		t.Fatal("signing-key retention omitted the persisted proof manifest")
	}
}

func waitForProviderManifestIssueWaiters(t *testing.T, db *sql.DB, want int) {
	t.Helper()
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		var waiting int
		if err := db.QueryRow(`
			SELECT COUNT(*)
			FROM pg_stat_activity
			WHERE datname=current_database()
			  AND pid <> pg_backend_pid()
			  AND state='active'
			  AND wait_event_type='Lock'
			  AND query LIKE '%SELECT id::text FROM provider_pilot_epochs%'
			  AND query LIKE '%FOR UPDATE%'`).Scan(&waiting); err != nil {
			t.Fatalf("inspect proof-manifest issuance waiters: %v", err)
		}
		if waiting >= want {
			return
		}
		select {
		case <-deadline.C:
			t.Fatalf("proof-manifest issuance waiters=%d, want at least %d", waiting, want)
		case <-ticker.C:
		}
	}
}

func exerciseStage1FactIntegrityPostgres(t *testing.T, db *sql.DB, site models.Site) {
	t.Helper()
	searchID, err := models.GenerateDemandSearchID()
	if err != nil {
		t.Fatalf("generate Stage 1 fact-integrity search id: %v", err)
	}
	var receiptID string
	var searchCreatedAt time.Time
	if err := db.QueryRow(`
		INSERT INTO search_receipts (
			public_id, surface, explicit_category, demand_topics,
			result_count, page_number, page_size, is_synthetic, created_at
		) VALUES (
			$1, 'rest', 'developer', ARRAY['developer-tools']::text[],
			1, 1, 10, false, TIMESTAMPTZ '2000-01-01 00:00:00Z'
		)
		RETURNING id::text, created_at`, searchID).Scan(&receiptID, &searchCreatedAt); err != nil {
		t.Fatalf("insert Stage 1 fact-integrity search: %v", err)
	}
	var returnedID int64
	var returnedAt time.Time
	if err := db.QueryRow(`
		INSERT INTO organic_results_returned (
			search_receipt_id, site_id, site_domain_snapshot,
			organic_position, score_snapshot, returned_at
		) VALUES (
			$1::uuid, $2::uuid, $3, 1, 90,
			TIMESTAMPTZ '2000-01-01 00:00:00Z'
		)
		RETURNING id, returned_at`, receiptID, site.ID, site.Domain).
		Scan(&returnedID, &returnedAt); err != nil {
		t.Fatalf("insert Stage 1 fact-integrity organic result: %v", err)
	}
	var selectionID int64
	var selectedAt time.Time
	if err := db.QueryRow(`
		INSERT INTO result_selections (
			search_receipt_id, site_id, site_domain_snapshot, surface, selected_at
		) VALUES (
			$1::uuid, $2::uuid, $3, 'rest',
			TIMESTAMPTZ '2000-01-01 00:00:00Z'
		)
		RETURNING id, selected_at`, receiptID, site.ID, site.Domain).
		Scan(&selectionID, &selectedAt); err != nil {
		t.Fatalf("insert Stage 1 fact-integrity result selection: %v", err)
	}
	var interestID string
	var interestCreatedAt, interestExpiresAt time.Time
	if err := db.QueryRow(`
		INSERT INTO action_interest_receipts (
			public_id, search_receipt_id, source_is_synthetic,
			site_domain_snapshot, action_type, surface,
			caller_attests_principal_interest, confirmation_version,
			created_at, expires_at
		) VALUES (
			'nhs_air_FFFFFFFFFFFFFFFF', $1::uuid, false, $2,
			'trial', 'rest', true, 'nhs-action-interest-v1',
			TIMESTAMPTZ '2000-01-01 00:00:00Z',
			clock_timestamp() + INTERVAL '1 day'
		)
		RETURNING id::text, created_at, expires_at`, receiptID, site.Domain).
		Scan(&interestID, &interestCreatedAt, &interestExpiresAt); err != nil {
		t.Fatalf("insert Stage 1 fact-integrity action interest: %v", err)
	}
	for label, observed := range map[string]time.Time{
		"search created_at":          searchCreatedAt,
		"organic returned_at":        returnedAt,
		"selection selected_at":      selectedAt,
		"action-interest created_at": interestCreatedAt,
	} {
		if observed.Year() == 2000 || time.Since(observed) < 0 || time.Since(observed) > time.Minute {
			t.Fatalf("%s was not database-owned: %s", label, observed)
		}
	}
	if want := searchCreatedAt.Add(30 * 24 * time.Hour); !interestExpiresAt.Equal(want) {
		t.Fatalf("action-interest expiry=%s, want database-derived source boundary %s", interestExpiresAt, want)
	}

	for _, mutation := range []struct {
		query      string
		constraint string
		args       []any
	}{
		{
			query:      `UPDATE search_receipts SET created_at=TIMESTAMPTZ '2000-01-01 00:00:00Z' WHERE id=$1::uuid`,
			constraint: "search_receipt_stage1_immutable", args: []any{receiptID},
		},
		{
			query:      `UPDATE organic_results_returned SET returned_at=TIMESTAMPTZ '2000-01-01 00:00:00Z' WHERE id=$1`,
			constraint: "organic_result_stage1_immutable", args: []any{returnedID},
		},
		{
			query:      `UPDATE result_selections SET selected_at=TIMESTAMPTZ '2000-01-01 00:00:00Z' WHERE id=$1`,
			constraint: "result_selection_stage1_immutable", args: []any{selectionID},
		},
	} {
		if _, err := db.Exec(mutation.query, mutation.args...); err == nil {
			t.Fatalf("database accepted Stage 1 fact mutation for %s", mutation.constraint)
		} else {
			assertPostgresConstraintCode(t, err, "23514", mutation.constraint)
		}
	}

	if _, err := db.Exec(`
		UPDATE action_interest_receipts
		SET expires_at=clock_timestamp()+INTERVAL '2 days'
		WHERE id=$1::uuid`, interestID); err != nil {
		t.Fatalf("exercise append-only action-interest rule: %v", err)
	}
	var persistedInterestExpiry time.Time
	if err := db.QueryRow(`SELECT expires_at FROM action_interest_receipts WHERE id=$1::uuid`, interestID).
		Scan(&persistedInterestExpiry); err != nil {
		t.Fatalf("read immutable action-interest expiry: %v", err)
	}
	if !persistedInterestExpiry.Equal(interestExpiresAt) {
		t.Fatalf("action-interest expiry changed from %s to %s", interestExpiresAt, persistedInterestExpiry)
	}

	var stage1Anchor time.Time
	if err := db.QueryRow(`
		SELECT applied_at FROM nhs_schema_migrations
		WHERE name='025_stage1_fact_integrity.sql'`).Scan(&stage1Anchor); err != nil {
		t.Fatalf("read Stage 1 migration anchor: %v", err)
	}
	if result, err := db.Exec(`
		UPDATE nhs_schema_migrations SET applied_at=TIMESTAMPTZ '2000-01-01 00:00:00Z'
		WHERE name='025_stage1_fact_integrity.sql'`); err != nil {
		t.Fatalf("exercise immutable migration ledger update rule: %v", err)
	} else if affected, _ := result.RowsAffected(); affected != 0 {
		t.Fatalf("immutable migration ledger update affected %d rows", affected)
	}
	if result, err := db.Exec(`
		DELETE FROM nhs_schema_migrations
		WHERE name='025_stage1_fact_integrity.sql'`); err != nil {
		t.Fatalf("exercise immutable migration ledger delete rule: %v", err)
	} else if affected, _ := result.RowsAffected(); affected != 0 {
		t.Fatalf("immutable migration ledger delete affected %d rows", affected)
	}
	var persistedStage1Anchor time.Time
	if err := db.QueryRow(`
		SELECT applied_at FROM nhs_schema_migrations
		WHERE name='025_stage1_fact_integrity.sql'`).Scan(&persistedStage1Anchor); err != nil {
		t.Fatalf("read immutable Stage 1 migration anchor: %v", err)
	}
	if !persistedStage1Anchor.Equal(stage1Anchor) {
		t.Fatalf("Stage 1 migration anchor changed from %s to %s", stage1Anchor, persistedStage1Anchor)
	}

	var anchorFunction string
	if err := db.QueryRow(`
		SELECT pg_get_functiondef(
			'public.lock_provider_pilot_stage1_epoch_anchor()'::regprocedure
		)`).Scan(&anchorFunction); err != nil {
		t.Fatalf("read Stage 1 anchor-lock function: %v", err)
	}
	if !strings.Contains(anchorFunction, "FOR UPDATE") ||
		!strings.Contains(anchorFunction, "025_stage1_fact_integrity.sql") {
		t.Fatalf("Stage 1 anchor-lock function lacks UPDATE-strength lock: %s", anchorFunction)
	}

	if _, err := db.Exec(`DELETE FROM search_receipts WHERE id=$1::uuid`, receiptID); err != nil {
		t.Fatalf("clean Stage 1 fact-integrity fixture: %v", err)
	}
}

func exerciseProviderPilotTopicIsolationPostgres(
	t *testing.T,
	db *sql.DB,
	site models.Site,
	offerID string,
) {
	t.Helper()
	searchID := "nhs_sr_pg_release_wrong_topic"
	if err := models.RecordDemandSearch(db, models.DemandSearchReceipt{
		PublicID: searchID, Surface: "rest", Query: "hiring jobs",
		Category: "jobs", ResultCount: 1, Page: 1, PageSize: 10,
	}, []models.Site{site}); err != nil {
		t.Fatalf("record cross-topic organic search: %v", err)
	}
	offers, err := models.ListPublicProviderOffersForOrganicResults(
		db, searchID, []models.Site{site},
	)
	if err != nil {
		t.Fatalf("list cross-topic provider sidecars: %v", err)
	}
	if len(offers) != 0 {
		t.Fatalf("cross-topic search exposed %d paid sidecar(s), want 0", len(offers))
	}
	var organicCount, paidEvidenceCount int
	if err := db.QueryRow(`
		SELECT
		  (SELECT COUNT(*)::int
		   FROM organic_results_returned organic
		   JOIN search_receipts receipt ON receipt.id=organic.search_receipt_id
		   WHERE receipt.public_id=$1 AND organic.site_id=$2::uuid
		     AND organic.organic_position=1),
		  (SELECT COUNT(*)::int
		   FROM provider_offers_returned returned
		   JOIN search_receipts receipt ON receipt.id=returned.search_receipt_id
		   WHERE receipt.public_id=$1 AND returned.provider_offer_id=$3::uuid)`,
		searchID, site.ID, offerID,
	).Scan(&organicCount, &paidEvidenceCount); err != nil {
		t.Fatalf("inspect cross-topic organic/paid evidence: %v", err)
	}
	if organicCount != 1 || paidEvidenceCount != 0 {
		t.Fatalf(
			"cross-topic boundary organic=%d paid_evidence=%d, want 1/0",
			organicCount, paidEvidenceCount,
		)
	}

	noOrganicSearchID := "nhs_sr_pg_release_no_organic"
	if err := models.RecordDemandSearch(db, models.DemandSearchReceipt{
		PublicID: noOrganicSearchID, Surface: "rest", Query: "developer api",
		Category: "developer", ResultCount: 0, Page: 1, PageSize: 10,
	}, nil); err != nil {
		t.Fatalf("record no-organic developer search: %v", err)
	}
	_, err = db.Exec(`
		INSERT INTO provider_offers_returned (
			search_receipt_id, provider_offer_id, provider_claim_id,
			provider_pilot_epoch_id_snapshot,
			offer_version_snapshot, offer_name_snapshot, action_type_snapshot,
			disclosure_snapshot, bounty_cents_snapshot, currency_snapshot,
			charge_event_snapshot, commercial_terms_contract_version_snapshot,
			commercial_terms_sha256_snapshot
		)
		SELECT receipt.id, offer.id, offer.provider_claim_id,
		       offer.provider_pilot_epoch_id,
		       offer.version, offer.offer_name, offer.action_type,
		       offer.disclosure_label, offer.bounty_cents, offer.currency,
		       offer.charge_event, offer.commercial_terms_contract_version,
		       offer.commercial_terms_sha256
		FROM search_receipts receipt
		JOIN provider_offers offer ON offer.id=$2::uuid
		WHERE receipt.public_id=$1`, noOrganicSearchID, offerID)
	if err == nil {
		t.Fatal("direct SQL returned a paid offer absent from the organic result set")
	}
	assertPostgresConstraint(t, err, "provider_returned_offer_organic_result")
}

func exerciseProviderPilotSpamReclassificationPostgres(
	t *testing.T,
	db *sql.DB,
	signer *providerexchange.Signer,
	site models.Site,
	offerID string,
) {
	t.Helper()
	searchID := recordPostgresDemandReceipt(t, db, site, "stage1-spam-reclassification")
	recordPostgresReturnedOffer(t, db, site, searchID, offerID)
	if _, err := db.Exec(`
		UPDATE sites SET category='spam' WHERE id=$1::uuid`, site.ID); err != nil {
		t.Fatalf("reclassify active provider site as spam: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec(`UPDATE sites SET category='developer' WHERE id=$1::uuid`, site.ID)
	})
	offers, err := models.ListPublicProviderOffersForOrganicResults(
		db, searchID, []models.Site{site},
	)
	if err != nil {
		t.Fatalf("list sidecars after provider spam reclassification: %v", err)
	}
	if len(offers) != 0 {
		t.Fatalf("spam-reclassified provider exposed %d paid sidecars", len(offers))
	}
	if _, _, _, err := models.CreateActionTicket(db, models.ActionTicketInput{
		ProviderOfferID:       offerID,
		SearchReceiptPublicID: searchID,
		DemandTopic:           "developer-tools",
		PrincipalConsent:      true,
		ConsentVersion:        models.ProviderPrincipalConsentV1,
		TTL:                   time.Hour,
	}, signer); !errors.Is(err, models.ErrProviderOfferNotPublic) {
		t.Fatalf("ticket for spam-reclassified provider error=%v, want ErrProviderOfferNotPublic", err)
	}
	var ticketCount int
	if err := db.QueryRow(`
		SELECT COUNT(*)::int
		FROM action_tickets ticket
		JOIN search_receipts receipt ON receipt.id=ticket.search_receipt_id
		WHERE receipt.public_id=$1 AND ticket.provider_offer_id=$2::uuid`,
		searchID, offerID).Scan(&ticketCount); err != nil {
		t.Fatalf("count spam-reclassified provider tickets: %v", err)
	}
	if ticketCount != 0 {
		t.Fatalf("spam-reclassified provider created %d tickets", ticketCount)
	}
	if _, err := db.Exec(`
		UPDATE sites SET category='developer' WHERE id=$1::uuid`, site.ID); err != nil {
		t.Fatalf("restore active provider site after spam test: %v", err)
	}
}

func exerciseProviderPilotCloseBoundaryPostgres(
	t *testing.T,
	db *sql.DB,
	signer *providerexchange.Signer,
	providerKey *models.ProviderAPIKey,
	site models.Site,
	offerID string,
) {
	t.Helper()
	resolvableTicket := createPostgresActionTicket(
		t, db, signer, site, offerID, "pilot-close-resolvable",
	)
	blockedTicket, blockedToken := createPostgresUnhandedActionTicket(
		t, db, signer, site, offerID, "pilot-close-blocked",
	)
	recordPostgresTicketReview(t, db, blockedTicket, "pilot-close-blocked")
	for label, query := range map[string]string{
		"bearer hash": `
			UPDATE action_tickets SET token_hash=$2
			WHERE id=$1::uuid`,
		"bounty": `
			UPDATE action_tickets SET bounty_cents_snapshot=bounty_cents_snapshot+1
			WHERE id=$1::uuid`,
		"action URL": `
			UPDATE action_tickets SET action_url_snapshot=action_url_snapshot || '/mutated'
			WHERE id=$1::uuid`,
	} {
		var mutationErr error
		if label == "bearer hash" {
			_, mutationErr = db.Exec(
				query, blockedTicket.ID, postgresPayloadHash("mutated-ticket-bearer"),
			)
		} else {
			_, mutationErr = db.Exec(query, blockedTicket.ID)
		}
		if mutationErr == nil {
			t.Fatalf("direct SQL rewrote pilot ticket %s", label)
		}
		assertPostgresConstraint(t, mutationErr, "action_ticket_pilot_snapshot_immutable")
	}
	proofBeforeCloseResolution, err := models.GetProviderExchangeProof(
		db, postgresProviderPilotEpochID, signer,
	)
	if err != nil {
		t.Fatalf("read proof before post-close resolution: %v", err)
	}
	closed, err := models.CloseProviderPilotEpoch(db, models.ProviderPilotEpochActionInput{
		ProviderPilotEpochID: postgresProviderPilotEpochID,
		OwnerReference:       "operator:close:pg-pilot",
		EvidenceReference:    "evidence:close:pg-pilot",
	})
	if err != nil {
		t.Fatalf("close bounded provider pilot: %v", err)
	}
	if closed.Status != "closed" || closed.ClosedAt == nil {
		t.Fatalf("closed provider pilot = %#v", closed)
	}

	if _, _, err := models.RecordActionTicketHandoff(db, models.ProviderActionHandoffInput{
		ActionTicketID:          blockedTicket.ID,
		AttributionToken:        blockedToken,
		PrincipalHandoffConsent: true,
		HandoffConsentVersion:   models.ProviderActionHandoffConsentV1,
	}); !errors.Is(err, models.ErrProviderPilotNotActive) {
		t.Fatalf("model handoff after pilot close error=%v, want ErrProviderPilotNotActive", err)
	}

	_, err = db.Exec(`
		INSERT INTO provider_action_handoff_receipts (
			action_ticket_id, provider_claim_id, provider_offer_id,
			offer_version_snapshot,
			commercial_terms_contract_version_snapshot,
			commercial_terms_sha256_snapshot, presented_token_hash,
			principal_handoff_consent, handoff_consent_version,
			principal_controlled_intent_disclosure_consent,
			controlled_intent_disclosure_consent_version,
			event_contract_version
		)
		SELECT id, provider_claim_id, provider_offer_id,
		       offer_version_snapshot,
		       commercial_terms_contract_version_snapshot,
		       commercial_terms_sha256_snapshot, $2,
		       true, 'nhs-provider-handoff-consent-v1', false, '',
		       'nhs-action-handoff-v1'
		FROM action_tickets WHERE id=$1::uuid`,
		blockedTicket.ID, models.HashProviderSecret(blockedToken),
	)
	if err == nil {
		t.Fatal("direct SQL created a new handoff after pilot close")
	}
	assertPostgresConstraint(t, err, "provider_pilot_handoff_active_epoch")

	var blockedHandoffCount int
	if err := db.QueryRow(`
		SELECT COUNT(*)::int FROM provider_action_handoff_receipts
		WHERE action_ticket_id=$1::uuid`, blockedTicket.ID).Scan(&blockedHandoffCount); err != nil {
		t.Fatalf("count blocked post-close handoffs: %v", err)
	}
	if blockedHandoffCount != 0 {
		t.Fatalf("post-close handoff count=%d, want 0", blockedHandoffCount)
	}

	receipt := recordPostgresOutcome(
		t, db, signer, providerKey, resolvableTicket.ID,
		"accepted", "pilot-close-late-resolution",
	)
	if receipt.ChargeStatus != string(providerexchange.ChargeStatusCharged) {
		t.Fatalf("post-close resolution charge status=%q, want charged", receipt.ChargeStatus)
	}
	proofAfterCloseResolution, err := models.GetProviderExchangeProof(
		db, postgresProviderPilotEpochID, signer,
	)
	if err != nil {
		t.Fatalf("read proof after post-close resolution: %v", err)
	}
	if proofAfterCloseResolution.VerifiedProviderAcceptedHandoffs !=
		proofBeforeCloseResolution.VerifiedProviderAcceptedHandoffs+1 {
		t.Fatalf(
			"post-close accepted proof=%d, before=%d; exact authorized resolution was lost",
			proofAfterCloseResolution.VerifiedProviderAcceptedHandoffs,
			proofBeforeCloseResolution.VerifiedProviderAcceptedHandoffs,
		)
	}
}

// exerciseProviderPilotLifecycleEventCompletenessPostgres proves that a direct
// writer cannot commit draft creation, enrollment, activation, or closure
// without the matching canonical append-only event. Production model paths
// write each state row and event in one transaction and must still succeed.
func exerciseProviderPilotLifecycleEventCompletenessPostgres(t *testing.T, db *sql.DB) {
	t.Helper()
	proof, err := models.GetStage1DemandProof(db, 30)
	if err != nil {
		t.Fatalf("read lifecycle-completeness Stage 1 proof: %v", err)
	}
	stage1SHA, err := models.ProviderPilotStage1SnapshotSHA256(proof)
	if err != nil {
		t.Fatalf("hash lifecycle-completeness Stage 1 proof: %v", err)
	}

	directCreate, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	var omittedEventPilotID string
	err = directCreate.QueryRow(`
		INSERT INTO provider_pilot_epochs (
			contract_version, demand_topic, stage1_started_at,
			stage1_evidence_as_of, stage1_evidence_sha256, cohort_limit,
			provider_ticket_cap, total_ticket_cap, owner_reference,
			evidence_reference
		) VALUES ($1,'developer-tools',$2,$3,$4,10,1,10,$5,$6)
		RETURNING id::text`,
		models.ProviderPilotEpochContractV1, proof.Stage1StartedAt, proof.AsOf,
		stage1SHA, "operator:lifecycle-create-026", "evidence:lifecycle-create-026",
	).Scan(&omittedEventPilotID)
	if err != nil {
		_ = directCreate.Rollback()
		t.Fatalf("insert direct epoch without event: %v", err)
	}
	if err := directCreate.Commit(); err == nil {
		t.Fatal("direct epoch creation committed without a canonical created event")
	} else {
		assertPostgresConstraint(t, err, "provider_pilot_created_event_required")
	}

	epoch, err := models.CreateProviderPilotEpoch(db, models.ProviderPilotEpochInput{
		DemandTopic:       "developer-tools",
		CohortLimit:       10,
		ProviderTicketCap: 1,
		TotalTicketCap:    10,
		OwnerReference:    "operator:lifecycle-create-026",
		EvidenceReference: "evidence:lifecycle-create-026",
	})
	if err != nil {
		t.Fatalf("create epoch with canonical event: %v", err)
	}

	type companyClaim struct {
		companyID string
		claimID   string
	}
	rows, err := db.Query(`
		SELECT company.id::text, company.provider_claim_id::text
		FROM provider_pilot_companies company
		JOIN provider_claims claim ON claim.id=company.provider_claim_id
		WHERE claim.status='verified'
		  AND claim.verification_last_succeeded_at >
		      clock_timestamp()-INTERVAL '7 days'
		  AND EXISTS (
		      SELECT 1
		      FROM provider_pilot_enrollments prior
		      WHERE prior.provider_claim_id=company.provider_claim_id
		        AND provider_pilot_enrollment_eligibility_is_current(
		            prior.provider_pilot_epoch_id, prior.provider_claim_id
		        )
		  )
		ORDER BY company.id
		LIMIT 10`)
	if err != nil {
		t.Fatal(err)
	}
	freshCompanies := make([]companyClaim, 0, 10)
	for rows.Next() {
		var company companyClaim
		if err := rows.Scan(&company.companyID, &company.claimID); err != nil {
			_ = rows.Close()
			t.Fatal(err)
		}
		freshCompanies = append(freshCompanies, company)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		t.Fatal(err)
	}
	_ = rows.Close()
	if len(freshCompanies) != 10 {
		t.Fatalf("fresh lifecycle cohort=%d, want 10", len(freshCompanies))
	}

	directEnrollment, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := directEnrollment.Exec(`
		INSERT INTO provider_pilot_enrollments (
			provider_pilot_epoch_id, provider_pilot_company_id,
			provider_claim_id, owner_reference, evidence_reference
		) VALUES ($1::uuid,$2::uuid,$3::uuid,$4,$5)`,
		epoch.ID, freshCompanies[0].companyID, freshCompanies[0].claimID,
		"operator:lifecycle-enroll-026", "evidence:lifecycle-enroll-026",
	); err != nil {
		_ = directEnrollment.Rollback()
		t.Fatalf("insert direct enrollment without event: %v", err)
	}
	if err := directEnrollment.Commit(); err == nil {
		t.Fatal("direct enrollment committed without a canonical enrollment event")
	} else {
		assertPostgresConstraint(t, err, "provider_pilot_enrollment_event_required")
	}

	for _, company := range freshCompanies {
		if _, err := models.EnrollProviderPilotCompany(db, models.ProviderPilotEnrollmentInput{
			ProviderPilotEpochID: epoch.ID,
			ProviderClaimID:      company.claimID,
			OwnerReference:       "operator:lifecycle-enroll-026",
			EvidenceReference:    "evidence:lifecycle-enroll-026",
		}); err != nil {
			t.Fatalf("enroll lifecycle cohort with canonical event: %v", err)
		}
		recordPostgresPilotReview(
			t, db, epoch.ID, "provider", company.claimID,
			"lifecycle-"+company.claimID[:8],
		)
	}

	directActivation, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := directActivation.Exec(`
		UPDATE provider_pilot_epochs SET status='active'
		WHERE id=$1::uuid`, epoch.ID); err != nil {
		_ = directActivation.Rollback()
		t.Fatalf("direct activation without event: %v", err)
	}
	if err := directActivation.Commit(); err == nil {
		t.Fatal("direct activation committed without a canonical activated event")
	} else {
		assertPostgresConstraint(t, err, "provider_pilot_activated_event_required")
	}

	if _, err := models.ActivateProviderPilotEpoch(db, models.ProviderPilotEpochActionInput{
		ProviderPilotEpochID: epoch.ID,
		OwnerReference:       "operator:lifecycle-activate-026",
		EvidenceReference:    "evidence:lifecycle-activate-026",
	}); err != nil {
		t.Fatalf("activate epoch with canonical event: %v", err)
	}

	directClose, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := directClose.Exec(`
		UPDATE provider_pilot_epochs SET status='closed'
		WHERE id=$1::uuid`, epoch.ID); err != nil {
		_ = directClose.Rollback()
		t.Fatalf("direct close without event: %v", err)
	}
	if err := directClose.Commit(); err == nil {
		t.Fatal("direct close committed without a canonical closed event")
	} else {
		assertPostgresConstraint(t, err, "provider_pilot_closed_event_required")
	}

	if _, err := models.CloseProviderPilotEpoch(db, models.ProviderPilotEpochActionInput{
		ProviderPilotEpochID: epoch.ID,
		OwnerReference:       "operator:lifecycle-close-026",
		EvidenceReference:    "evidence:lifecycle-close-026",
	}); err != nil {
		t.Fatalf("close epoch with canonical event: %v", err)
	}
}

func exerciseProviderTicketIssuanceAfterRowLock(
	t *testing.T,
	db *sql.DB,
	signer *providerexchange.Signer,
	providerKey *models.ProviderAPIKey,
	accountID int64,
	claimID string,
	site models.Site,
) {
	t.Helper()
	const bounty int64 = 20_000
	zero := int64(0)
	offer, err := models.CreateProviderOffer(db, accountID, claimID, models.ProviderOfferInput{
		OfferName:           "Post-row-lock ticket issuance",
		OfferSummary:        "Proves ticket time is read only after provider offer eligibility is locked.",
		ActionType:          "purchase",
		ActionURL:           "https://provider.example/capacity/post-row-lock",
		ChargeEvent:         "accepted",
		BountyCents:         bounty,
		Currency:            "usd",
		PrincipalPriceMode:  "free",
		PrincipalPriceCents: &zero,
		PrincipalCurrency:   "usd",
		BillingMode:         "prepaid",
	})
	if err != nil {
		t.Fatalf("create post-row-lock offer: %v", err)
	}
	recordPostgresVerifiedFunding(
		t, db, offer.ID, bounty, "post-row-lock-fund", "", time.Time{},
	)
	offer, err = activatePostgresProviderOffer(
		t, db, offer.ID, "operator:pg-post-row-lock", "evidence:pg-post-row-lock",
	)
	if err != nil {
		t.Fatalf("activate post-row-lock offer: %v", err)
	}
	searchID := recordPostgresDemandReceipt(t, db, site, "post-row-lock")
	recordPostgresReturnedOffer(t, db, site, searchID, offer.ID)

	blockingTx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin post-row-lock blocker: %v", err)
	}
	defer blockingTx.Rollback()
	if _, err := blockingTx.Exec(`
		UPDATE provider_offers SET updated_at=updated_at WHERE id=$1::uuid`, offer.ID); err != nil {
		t.Fatalf("hold provider offer row lock: %v", err)
	}

	type ticketResult struct {
		ticket *models.ActionTicket
		err    error
	}
	result := make(chan ticketResult, 1)
	go func() {
		ticket, _, _, createErr := models.CreateActionTicket(db, models.ActionTicketInput{
			ProviderOfferID:       offer.ID,
			SearchReceiptPublicID: searchID,
			DemandTopic:           "developer-tools",
			PrincipalConsent:      true,
			ConsentVersion:        models.ProviderPrincipalConsentV1,
			TTL:                   time.Hour,
		}, signer)
		result <- ticketResult{ticket: ticket, err: createErr}
	}()

	waitDeadline := time.Now().Add(2 * time.Second)
	for {
		select {
		case got := <-result:
			t.Fatalf("ticket creation returned before held offer-row release: ticket=%#v err=%v", got.ticket, got.err)
		default:
		}
		var waitingOnOfferRow bool
		if err := db.QueryRow(`
			SELECT EXISTS (
				SELECT 1 FROM pg_stat_activity
				WHERE datname=current_database()
				  AND pid<>pg_backend_pid()
				  AND state='active'
				  AND wait_event_type='Lock'
				  AND position('SELECT receipt.id::text, receipt.is_synthetic, organic.organic_position' IN query)>0
			)`).Scan(&waitingOnOfferRow); err != nil {
			t.Fatalf("observe ticket waiting on offer row: %v", err)
		}
		if waitingOnOfferRow {
			break
		}
		if time.Now().After(waitDeadline) {
			t.Fatal("ticket creation did not reach the held provider offer row")
		}
		time.Sleep(10 * time.Millisecond)
	}
	time.Sleep(1200 * time.Millisecond)
	var releaseAt time.Time
	if err := db.QueryRow(`SELECT clock_timestamp()`).Scan(&releaseAt); err != nil {
		t.Fatalf("read database clock before row-lock release: %v", err)
	}
	if err := blockingTx.Rollback(); err != nil {
		t.Fatalf("release provider offer row lock: %v", err)
	}

	select {
	case got := <-result:
		if got.err != nil || got.ticket == nil {
			t.Fatalf("create ticket after row-lock wait: ticket=%#v err=%v", got.ticket, got.err)
		}
		if got.ticket.CreatedAt.Before(releaseAt.UTC().Truncate(time.Second)) {
			t.Fatalf("ticket created_at=%v predates row-lock release boundary %v", got.ticket.CreatedAt, releaseAt)
		}
		if got.ticket.ExpiresAt.Sub(got.ticket.CreatedAt) != time.Hour {
			t.Fatalf("ticket lifetime=%v, want 1h", got.ticket.ExpiresAt.Sub(got.ticket.CreatedAt))
		}
		recordPostgresOutcome(t, db, signer, providerKey, got.ticket.ID, "rejected", "post-row-lock-release")
	case <-time.After(5 * time.Second):
		t.Fatal("ticket creation did not finish after provider offer row release")
	}
}

func exerciseProviderCapacityReservations(
	t *testing.T,
	db *sql.DB,
	signer *providerexchange.Signer,
	providerKey *models.ProviderAPIKey,
	accountID int64,
	claimID string,
	site models.Site,
) {
	t.Helper()
	const bounty int64 = 25_000
	zero := int64(0)
	createOffer := func(name, actionType, billingMode string) *models.ProviderOffer {
		input := models.ProviderOfferInput{
			OfferName:           name,
			OfferSummary:        "Exercises one atomic provider capacity promise per live ticket.",
			ActionType:          actionType,
			ActionURL:           "https://provider.example/capacity/" + actionType,
			ChargeEvent:         "accepted",
			BountyCents:         bounty,
			Currency:            "usd",
			PrincipalPriceMode:  "free",
			PrincipalPriceCents: &zero,
			PrincipalCurrency:   "usd",
			BillingMode:         billingMode,
		}
		if billingMode == "terms" {
			limit, days := bounty, 30
			input.TermsCreditLimitCents = &limit
			input.TermsPeriodDays = &days
		}
		offer, err := models.CreateProviderOffer(db, accountID, claimID, input)
		if err != nil {
			t.Fatalf("create %s reservation offer: %v", billingMode, err)
		}
		if billingMode == "prepaid" {
			recordPostgresVerifiedFunding(
				t, db, offer.ID, bounty, "capacity-prepaid-fund", "", time.Time{},
			)
		} else {
			recordPostgresVerifiedTerms(
				t, db, providerKey, offer.ID, "capacity-terms", "", "",
			)
		}
		offer, err = activatePostgresProviderOffer(
			t, db, offer.ID, "operator:pg-capacity", "evidence:pg-capacity-"+billingMode,
		)
		if err != nil {
			t.Fatalf("activate %s reservation offer: %v", billingMode, err)
		}
		return offer
	}

	type ticketResult struct {
		ticket *models.ActionTicket
		err    error
	}
	runConcurrent := func(offer *models.ProviderOffer, prefix string, wantBudgetError error) *models.ActionTicket {
		searchIDs := []string{
			recordPostgresDemandReceipt(t, db, site, prefix+"-one"),
			recordPostgresDemandReceipt(t, db, site, prefix+"-two"),
		}
		// Commit both exact disclosures while the one bounty is still fully
		// available. Ticket creation, not response timing, owns the capacity
		// serialization boundary.
		for _, searchID := range searchIDs {
			recordPostgresReturnedOffer(t, db, site, searchID, offer.ID)
		}
		start := make(chan struct{})
		results := make(chan ticketResult, len(searchIDs))
		var wait sync.WaitGroup
		for _, searchID := range searchIDs {
			wait.Add(1)
			go func(id string) {
				defer wait.Done()
				<-start
				ticket, _, _, err := models.CreateActionTicket(db, models.ActionTicketInput{
					ProviderOfferID:       offer.ID,
					SearchReceiptPublicID: id,
					DemandTopic:           "developer-tools",
					PrincipalConsent:      true,
					ConsentVersion:        models.ProviderPrincipalConsentV1,
					TTL:                   time.Hour,
				}, signer)
				results <- ticketResult{ticket: ticket, err: err}
			}(searchID)
		}
		close(start)
		wait.Wait()
		close(results)
		var winner *models.ActionTicket
		successes, budgetFailures := 0, 0
		for result := range results {
			switch {
			case result.err == nil:
				successes++
				winner = result.ticket
			case errors.Is(result.err, wantBudgetError):
				budgetFailures++
			default:
				t.Fatalf("%s concurrent ticket error = %v", prefix, result.err)
			}
		}
		if successes != 1 || budgetFailures != 1 || winner == nil {
			t.Fatalf("%s concurrent results successes=%d budget_failures=%d winner=%#v", prefix, successes, budgetFailures, winner)
		}
		return winner
	}

	prepaid := createOffer("Prepaid capacity reservation", "quote", "prepaid")
	prepaidWinner := runConcurrent(prepaid, "capacity-prepaid", models.ErrInsufficientProviderFunds)
	if _, err := models.AdjustProviderOfferBudget(
		db, prepaid.ID, -bounty, "usd", "pg-capacity-blocked-withdrawal",
	); !errors.Is(err, models.ErrProviderLegacyBudgetMutation) {
		t.Fatalf("legacy prepaid withdrawal after pilot verification error = %v, want ErrProviderLegacyBudgetMutation", err)
	}
	recordPostgresOutcome(t, db, signer, providerKey, prepaidWinner.ID, "rejected", "capacity-prepaid-release")
	prepaidAfterRelease := createPostgresActionTicket(t, db, signer, site, prepaid.ID, "capacity-prepaid-after-release")
	recordPostgresOutcome(t, db, signer, providerKey, prepaidAfterRelease.ID, "rejected", "capacity-prepaid-release-two")

	terms := createOffer("Terms capacity reservation", "booking", "terms")
	termsWinner := runConcurrent(terms, "capacity-terms", models.ErrProviderTermsCreditLimit)
	recordPostgresOutcome(t, db, signer, providerKey, termsWinner.ID, "rejected", "capacity-terms-release")

	var reservationCount, consumeCount, releaseCount int
	if err := db.QueryRow(`
		SELECT
			COUNT(*) FILTER (WHERE event_type='reserve')::int,
			COUNT(*) FILTER (WHERE event_type='consume')::int,
			COUNT(*) FILTER (WHERE event_type='release')::int
		FROM provider_capacity_events
		WHERE provider_offer_id IN ($1::uuid,$2::uuid)`, prepaid.ID, terms.ID).
		Scan(&reservationCount, &consumeCount, &releaseCount); err != nil {
		t.Fatalf("read provider capacity event proof: %v", err)
	}
	if reservationCount != 3 || consumeCount != 0 || releaseCount != 3 {
		t.Fatalf("capacity events reserve=%d consume=%d release=%d, want 3/0/3", reservationCount, consumeCount, releaseCount)
	}
}

func exerciseProviderOutcomeAuthorizationAcrossExpiry(
	t *testing.T,
	db *sql.DB,
	signer *providerexchange.Signer,
	providerKey *models.ProviderAPIKey,
	accountID int64,
	claimID string,
	site models.Site,
) string {
	t.Helper()
	const bounty int64 = 30_000
	zero := int64(0)
	offer, err := models.CreateProviderOffer(db, accountID, claimID, models.ProviderOfferInput{
		OfferName:           "Post-lock outcome authorization",
		OfferSummary:        "Proves callbacks are authorized after lock waits and ticket expiry.",
		ActionType:          "signup",
		ActionURL:           "https://provider.example/capacity/post-lock-expiry",
		ChargeEvent:         "accepted",
		BountyCents:         bounty,
		Currency:            "usd",
		PrincipalPriceMode:  "free",
		PrincipalPriceCents: &zero,
		PrincipalCurrency:   "usd",
		BillingMode:         "prepaid",
	})
	if err != nil {
		t.Fatalf("create post-lock expiry offer: %v", err)
	}
	recordPostgresVerifiedFunding(
		t, db, offer.ID, bounty, "post-lock-expiry-fund", "", time.Time{},
	)
	offer, err = activatePostgresProviderOffer(
		t, db, offer.ID, "operator:pg-post-lock-expiry", "evidence:pg-post-lock-expiry",
	)
	if err != nil {
		t.Fatalf("activate post-lock expiry offer: %v", err)
	}

	expiringTicket, expiringToken := createPostgresUnhandedActionTicketWithTTL(
		t, db, signer, site, offer.ID, "post-lock-expiry-first", 6*time.Second,
	)
	recordPostgresTicketReview(t, db, expiringTicket, "post-lock-expiry-first")
	expiringTicket, handoff, err := models.RecordActionTicketHandoff(db, models.ProviderActionHandoffInput{
		ActionTicketID:          expiringTicket.ID,
		AttributionToken:        expiringToken,
		PrincipalHandoffConsent: true,
		HandoffConsentVersion:   models.ProviderActionHandoffConsentV1,
	})
	if err != nil || handoff == nil || expiringTicket.Status != "redirected" {
		t.Fatalf("create signed short-lived ticket handoff: ticket=%#v receipt=%#v err=%v", expiringTicket, handoff, err)
	}
	expiresAt := expiringTicket.ExpiresAt

	blockingTx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin post-lock expiry blocker: %v", err)
	}
	defer blockingTx.Rollback()
	if _, err := blockingTx.Exec(`
		SELECT pg_advisory_xact_lock(hashtext($1), hashtext($2))`,
		"nhs-provider-offer", offer.ID); err != nil {
		t.Fatalf("hold post-lock expiry offer lock: %v", err)
	}

	started := make(chan struct{})
	result := make(chan error, 1)
	go func() {
		close(started)
		_, _, recordErr := models.RecordProviderOutcome(db, providerKey, models.ProviderOutcomeInput{
			ActionTicketID: expiringTicket.ID,
			IdempotencyKey: "pg-post-lock-expiry-callback-0001",
			PayloadHash:    postgresPayloadHash("post-lock-expiry-callback"),
			Outcome:        "accepted",
		}, signer)
		result <- recordErr
	}()
	<-started

	waitDeadline := time.Now().Add(2 * time.Second)
	for {
		var waitingOnAdvisoryLock bool
		if err := db.QueryRow(`
			SELECT EXISTS (
				SELECT 1
				FROM pg_stat_activity
				WHERE datname=current_database()
				  AND pid<>pg_backend_pid()
				  AND state='active'
				  AND wait_event_type='Lock'
				  AND wait_event='advisory'
				  AND position('pg_advisory_xact_lock' IN query)>0
			)`).Scan(&waitingOnAdvisoryLock); err != nil {
			t.Fatalf("observe callback waiting on offer lock: %v", err)
		}
		if waitingOnAdvisoryLock {
			break
		}
		if time.Now().After(waitDeadline) {
			t.Fatal("callback did not reach the held offer advisory lock before deadline")
		}
		time.Sleep(10 * time.Millisecond)
	}

	var observedAt time.Time
	if err := db.QueryRow(`SELECT clock_timestamp()`).Scan(&observedAt); err != nil {
		t.Fatalf("read database clock before ticket expiry: %v", err)
	}
	if !expiresAt.After(observedAt) {
		t.Fatalf("callback reached offer lock at %v after ticket expiry %v", observedAt, expiresAt)
	}
	for {
		var expired bool
		if err := db.QueryRow(`
			SELECT expires_at<=clock_timestamp()
			FROM action_tickets WHERE id=$1::uuid`, expiringTicket.ID).Scan(&expired); err != nil {
			t.Fatalf("observe ticket expiry while callback is blocked: %v", err)
		}
		if expired {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err := blockingTx.Commit(); err != nil {
		t.Fatalf("release post-lock expiry offer lock: %v", err)
	}
	select {
	case err := <-result:
		if !errors.Is(err, models.ErrActionTicketExpired) {
			t.Fatalf("post-lock expiry callback error = %v, want ErrActionTicketExpired", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("post-lock expiry callback did not finish after releasing offer lock")
	}

	var outcomeCount, chargeCount, terminalCapacityCount int
	if err := db.QueryRow(`
		SELECT
			(SELECT COUNT(*)::int FROM outcome_receipts WHERE action_ticket_id=$1::uuid),
			(SELECT COUNT(*)::int FROM provider_budget_ledger
			 WHERE action_ticket_id=$1::uuid AND entry_type='charge'),
			(SELECT COUNT(*)::int FROM provider_capacity_events
			 WHERE action_ticket_id=$1::uuid AND event_type IN ('consume','release'))`,
		expiringTicket.ID).Scan(&outcomeCount, &chargeCount, &terminalCapacityCount); err != nil {
		t.Fatalf("read rejected post-lock expiry effects: %v", err)
	}
	if outcomeCount != 0 || chargeCount != 0 || terminalCapacityCount != 0 {
		t.Fatalf(
			"expired callback effects outcomes=%d charges=%d terminal_capacity=%d, want 0/0/0",
			outcomeCount, chargeCount, terminalCapacityCount,
		)
	}
	// Reuse immediately after PostgreSQL crosses the exact expiry boundary.
	// A second-truncated process clock would still see this reservation as live
	// for nearly a second and fail this deterministic boundary regression.
	reusedTicket := createPostgresActionTicket(
		t, db, signer, site, offer.ID, "post-lock-expiry-reused",
	)
	if reusedTicket.ID == expiringTicket.ID {
		t.Fatalf("capacity reuse returned original expired ticket %q", reusedTicket.ID)
	}
	recordPostgresOutcome(
		t, db, signer, providerKey, reusedTicket.ID, "rejected", "post-lock-expiry-reused-release",
	)
	assertPostgresProviderBalance(t, db, offer.ID, bounty)

	var reserveCount, consumeCount, releaseCount int
	if err := db.QueryRow(`
		SELECT
			COUNT(*) FILTER (WHERE event_type='reserve')::int,
			COUNT(*) FILTER (WHERE event_type='consume')::int,
			COUNT(*) FILTER (WHERE event_type='release')::int
		FROM provider_capacity_events
		WHERE provider_offer_id=$1::uuid`, offer.ID).
		Scan(&reserveCount, &consumeCount, &releaseCount); err != nil {
		t.Fatalf("read post-lock expiry capacity proof: %v", err)
	}
	if reserveCount != 2 || consumeCount != 0 || releaseCount != 1 {
		t.Fatalf(
			"post-lock expiry capacity reserve=%d consume=%d release=%d, want 2/0/1",
			reserveCount, consumeCount, releaseCount,
		)
	}
	return expiringTicket.ID
}

func exerciseProviderPilotStatusQueuePostgres(
	t *testing.T,
	db *sql.DB,
	signer *providerexchange.Signer,
	providerKey *models.ProviderAPIKey,
	company *models.ProviderPilotCompany,
	expiredHandoffTicketID string,
) {
	t.Helper()

	// A later provider-authenticated company acceptance must not hide the one
	// canonical owner-verified company or create an impossible owner queue row.
	secondAcceptance := recordPostgresPilotAcceptance(t, db, providerKey, "release-second-company")
	status, err := models.GetProviderPilotStatus(db, providerKey, 25)
	if err != nil {
		t.Fatalf("read provider status after second company acceptance: %v", err)
	}
	if !status.CompanyOwnerVerified ||
		status.CompanyAcceptanceID != company.ProviderAcceptanceEventID ||
		status.CompanyAcceptanceID == secondAcceptance.ID {
		t.Fatalf("canonical company status drifted after duplicate acceptance: %#v", status)
	}
	pendingCompanies, err := models.GetProviderPilotQueue(db, "pending_company", 100)
	if err != nil {
		t.Fatalf("read pending-company queue after duplicate acceptance: %v", err)
	}
	for _, item := range pendingCompanies.Items {
		if item.ProviderClaimID == providerKey.ProviderClaimID {
			t.Fatalf("canonical company left in pending-company queue: %#v", item)
		}
	}

	// Offer review readiness uses the exact same verified-commercial predicate
	// as activation. A partially reversed prepaid fund remains actionable at its
	// positive residual, a fully reversed fund does not, and a current terms
	// renewal supersedes an expired acceptance in both candidate and digest.
	prepaidProvider := createPostgresCommercialProvider(t, db, "queue-review-prepaid")
	prepaidOffer := createPostgresCommercialOffer(
		t, db, prepaidProvider, "queue-review-prepaid", "prepaid", 1_000,
	)
	prepaidFund := recordPostgresVerifiedFunding(
		t, db, prepaidOffer.ID, 3_000, "queue-review-prepaid", "", time.Time{},
	)
	if _, created, err := models.ReverseVerifiedProviderFunding(
		db, models.ProviderFundingReversalInput{
			RelatedCommitmentEventID: prepaidFund.ID,
			AmountCents:              1_000,
			SourceSystem:             "pg-settlement",
			SourceEventID:            "queue-review-prepaid-partial-reversal",
			SourceEffectiveAt:        postgresDatabaseClock(t, db),
			OperatorReference:        "operator:queue-review-prepaid-partial-reversal",
			OwnerEvidenceReference:   "evidence:queue-review-prepaid-partial-reversal",
		},
	); err != nil || !created {
		t.Fatalf("partially reverse prepaid review fixture: created=%t err=%v", created, err)
	}

	renewedProvider := createPostgresCommercialProvider(t, db, "queue-review-renewed-terms")
	renewedOffer := createPostgresCommercialOffer(
		t, db, renewedProvider, "queue-review-renewed-terms", "terms", 1_000,
	)
	initialAcceptance, initialCommitment := recordPostgresVerifiedTerms(
		t, db, renewedProvider.key, renewedOffer.ID,
		"queue-review-renewed-terms-initial", "", "",
	)
	renewalAcceptance, renewalCommitment := recordPostgresVerifiedTerms(
		t, db, renewedProvider.key, renewedOffer.ID,
		"queue-review-renewed-terms-current", initialAcceptance.ID, initialCommitment.ID,
	)
	// The immutable writers cannot naturally age a 30-day acceptance during one
	// test. Emulate elapsed time in this disposable database so the renewal is
	// the oldest *currently qualifying* exact commitment.
	renewalClock := postgresDatabaseClock(t, db)
	expiryTx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin renewed-terms expiry fixture: %v", err)
	}
	defer expiryTx.Rollback()
	if _, err := expiryTx.Exec(`ALTER TABLE provider_commercial_acceptance_events DISABLE RULE provider_commercial_acceptance_no_update`); err != nil {
		t.Fatalf("disable acceptance immutability for renewal fixture: %v", err)
	}
	if _, err := expiryTx.Exec(`ALTER TABLE provider_commercial_commitment_events DISABLE RULE provider_commercial_commitment_no_update`); err != nil {
		t.Fatalf("disable commitment immutability for renewal fixture: %v", err)
	}
	if _, err := expiryTx.Exec(`
		UPDATE provider_commercial_acceptance_events
		SET provider_accepted_at=$2::timestamptz-INTERVAL '2 days',
		    valid_until=$2::timestamptz-INTERVAL '1 day',
		    created_at=$2::timestamptz-INTERVAL '2 days'
		WHERE id=$1::uuid`, initialAcceptance.ID, renewalClock); err != nil {
		t.Fatalf("expire preceding terms acceptance: %v", err)
	}
	if _, err := expiryTx.Exec(`
		UPDATE provider_commercial_commitment_events
		SET provider_accepted_at=$2::timestamptz-INTERVAL '2 days',
		    valid_until=$2::timestamptz-INTERVAL '1 day'
		WHERE id=$1::uuid`, initialCommitment.ID, renewalClock); err != nil {
		t.Fatalf("expire preceding verified terms commitment: %v", err)
	}
	if _, err := expiryTx.Exec(`ALTER TABLE provider_commercial_acceptance_events ENABLE RULE provider_commercial_acceptance_no_update`); err != nil {
		t.Fatalf("restore acceptance immutability after renewal fixture: %v", err)
	}
	if _, err := expiryTx.Exec(`ALTER TABLE provider_commercial_commitment_events ENABLE RULE provider_commercial_commitment_no_update`); err != nil {
		t.Fatalf("restore commitment immutability after renewal fixture: %v", err)
	}
	if err := expiryTx.Commit(); err != nil {
		t.Fatalf("commit renewed-terms expiry fixture: %v", err)
	}

	reversedProvider := createPostgresCommercialProvider(t, db, "queue-review-reversed-prepaid")
	reversedOffer := createPostgresCommercialOffer(
		t, db, reversedProvider, "queue-review-reversed-prepaid", "prepaid", 1_000,
	)
	reversedFund := recordPostgresVerifiedFunding(
		t, db, reversedOffer.ID, 1_000, "queue-review-reversed-prepaid", "", time.Time{},
	)
	if _, created, err := models.ReverseVerifiedProviderFunding(
		db, models.ProviderFundingReversalInput{
			RelatedCommitmentEventID: reversedFund.ID,
			AmountCents:              1_000,
			SourceSystem:             "pg-settlement",
			SourceEventID:            "queue-review-reversed-prepaid-full-reversal",
			SourceEffectiveAt:        postgresDatabaseClock(t, db),
			OperatorReference:        "operator:queue-review-reversed-prepaid-full-reversal",
			OwnerEvidenceReference:   "evidence:queue-review-reversed-prepaid-full-reversal",
		},
	); err != nil || !created {
		t.Fatalf("fully reverse prepaid review fixture: created=%t err=%v", created, err)
	}

	offerReviews, err := models.GetProviderPilotQueue(db, "offer_review_required", 100)
	if err != nil {
		t.Fatalf("read verified-commercial offer review queue: %v", err)
	}
	findOfferReview := func(offerID string) *models.ProviderPilotQueueItem {
		t.Helper()
		for index := range offerReviews.Items {
			if offerReviews.Items[index].OfferID == offerID {
				return &offerReviews.Items[index]
			}
		}
		return nil
	}
	prepaidReview := findOfferReview(prepaidOffer.ID)
	if prepaidReview == nil || prepaidReview.CommitmentEventID != prepaidFund.ID ||
		prepaidReview.CommitmentEventType != "prepaid_fund" ||
		!providerHashPatternForPostgresTest(prepaidReview.SubjectSnapshotSHA256) {
		t.Fatalf("positive residual prepaid offer review queue item=%#v", prepaidReview)
	}
	renewedReview := findOfferReview(renewedOffer.ID)
	if renewedReview == nil || renewedReview.CommitmentEventID != renewalCommitment.ID ||
		renewedReview.CommitmentEventType != "terms_renewal" ||
		!providerHashPatternForPostgresTest(renewedReview.SubjectSnapshotSHA256) {
		t.Fatalf("renewed terms offer review queue item=%#v", renewedReview)
	}
	if findOfferReview(reversedOffer.ID) != nil {
		t.Fatal("fully reversed prepaid fund entered offer review queue")
	}

	prepaidCandidate, err := models.GetProviderPilotReviewCandidate(
		db, postgresProviderPilotEpochID, "offer", prepaidOffer.ID,
	)
	if err != nil || prepaidCandidate.CommitmentEventID != prepaidFund.ID ||
		prepaidCandidate.CommitmentEventType != "prepaid_fund" {
		t.Fatalf("positive residual prepaid review candidate=%#v err=%v", prepaidCandidate, err)
	}
	renewedCandidate, err := models.GetProviderPilotReviewCandidate(
		db, postgresProviderPilotEpochID, "offer", renewedOffer.ID,
	)
	if err != nil || renewedCandidate.CommitmentEventID != renewalCommitment.ID ||
		renewedCandidate.CommitmentEventID == initialCommitment.ID ||
		renewedCandidate.CommitmentEventType != "terms_renewal" ||
		renewedCandidate.ProviderAcceptanceEventID != renewalAcceptance.ID ||
		renewedCandidate.CommitmentValidUntil == nil {
		t.Fatalf("renewed terms review candidate=%#v err=%v", renewedCandidate, err)
	}
	if _, err := models.GetProviderPilotReviewCandidate(
		db, postgresProviderPilotEpochID, "offer", reversedOffer.ID,
	); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("fully reversed prepaid review candidate error=%v, want sql.ErrNoRows", err)
	}

	recordPostgresPilotReview(
		t, db, postgresProviderPilotEpochID, "offer", prepaidOffer.ID, "queue-review-prepaid",
	)
	recordPostgresPilotReview(
		t, db, postgresProviderPilotEpochID, "offer", renewedOffer.ID, "queue-review-renewed-terms",
	)
	offerReviews, err = models.GetProviderPilotQueue(db, "offer_review_required", 100)
	if err != nil {
		t.Fatalf("read offer review queue after reviews: %v", err)
	}
	for _, item := range offerReviews.Items {
		if item.OfferID == prepaidOffer.ID || item.OfferID == renewedOffer.ID {
			t.Fatalf("reviewed commercial offer remained in review queue: %#v", item)
		}
	}

	// A provider may accept while verified and then be revoked. Its immutable
	// acceptance is historical evidence, not actionable owner work.
	revokedProvider := createPostgresProviderIdentity(t, db, "queue-revoked-company")
	revokedAcceptance := recordPostgresPilotAcceptance(
		t, db, revokedProvider.key, "queue-revoked-company",
	)
	if err := models.RevokeProviderClaim(
		db, revokedProvider.accountID, revokedProvider.claim.ID,
	); err != nil {
		t.Fatalf("revoke pending-company queue fixture: %v", err)
	}
	pendingCompanies, err = models.GetProviderPilotQueue(db, "pending_company", 100)
	if err != nil {
		t.Fatalf("read pending-company queue after claim revocation: %v", err)
	}
	for _, item := range pendingCompanies.Items {
		if item.AcceptanceEventID == revokedAcceptance.ID {
			t.Fatalf("revoked claim remained in pending-company queue: %#v", item)
		}
	}

	// Emulate a superseded legacy acceptance in this disposable database. The
	// current writer prevents this state, but queue reads still fail closed if a
	// historical import or older release left one behind.
	supersededProvider := createPostgresCommercialProvider(t, db, "queue-superseded-terms")
	supersededOffer := createPostgresCommercialOffer(
		t, db, supersededProvider, "queue-superseded-terms", "terms", 1_000,
	)
	supersededAcceptance := recordPostgresUnverifiedTermsAcceptance(
		t, db, supersededProvider.key, supersededOffer.ID, "queue-superseded-terms",
	)
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin superseded-terms fixture: %v", err)
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`ALTER TABLE provider_offers DISABLE TRIGGER provider_offer_commercial_immutability_enforced`); err != nil {
		t.Fatalf("disable commercial immutability for legacy fixture: %v", err)
	}
	if _, err := tx.Exec(`
		UPDATE provider_offers SET version=version+1, updated_at=clock_timestamp()
		WHERE id=$1::uuid`, supersededOffer.ID); err != nil {
		t.Fatalf("supersede legacy terms fixture: %v", err)
	}
	if _, err := tx.Exec(`ALTER TABLE provider_offers ENABLE TRIGGER provider_offer_commercial_immutability_enforced`); err != nil {
		t.Fatalf("restore commercial immutability trigger: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit superseded-terms fixture: %v", err)
	}
	expiredProvider := createPostgresCommercialProvider(t, db, "queue-expired-terms")
	expiredOffer := createPostgresCommercialOffer(
		t, db, expiredProvider, "queue-expired-terms", "terms", 1_000,
	)
	expiredAcceptance := recordPostgresUnverifiedTermsAcceptance(
		t, db, expiredProvider.key, expiredOffer.ID, "queue-expired-terms",
	)
	expiredTx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin expired-terms fixture: %v", err)
	}
	defer expiredTx.Rollback()
	if _, err := expiredTx.Exec(`ALTER TABLE provider_commercial_acceptance_events DISABLE RULE provider_commercial_acceptance_no_update`); err != nil {
		t.Fatalf("disable acceptance immutability for expiry fixture: %v", err)
	}
	if _, err := expiredTx.Exec(`
		UPDATE provider_commercial_acceptance_events accepted
		SET provider_accepted_at=fixture.accepted_at,
		    created_at=fixture.accepted_at,
		    valid_until=fixture.accepted_at+INTERVAL '1 day'
		FROM (SELECT clock_timestamp()-INTERVAL '2 days' AS accepted_at) fixture
		WHERE accepted.id=$1::uuid`, expiredAcceptance.ID); err != nil {
		t.Fatalf("expire legacy terms acceptance fixture: %v", err)
	}
	if _, err := expiredTx.Exec(`ALTER TABLE provider_commercial_acceptance_events ENABLE RULE provider_commercial_acceptance_no_update`); err != nil {
		t.Fatalf("restore acceptance immutability rule: %v", err)
	}
	if err := expiredTx.Commit(); err != nil {
		t.Fatalf("commit expired-terms fixture: %v", err)
	}

	validProvider := createPostgresCommercialProvider(t, db, "queue-valid-terms")
	validOffer := createPostgresCommercialOffer(
		t, db, validProvider, "queue-valid-terms", "terms", 1_000,
	)
	validAcceptance := recordPostgresUnverifiedTermsAcceptance(
		t, db, validProvider.key, validOffer.ID, "queue-valid-terms",
	)
	pendingTerms, err := models.GetProviderPilotQueue(db, "pending_terms", 1)
	if err != nil {
		t.Fatalf("read pending-terms queue with superseded predecessor: %v", err)
	}
	if len(pendingTerms.Items) != 1 ||
		pendingTerms.Items[0].AcceptanceEventID != validAcceptance.ID {
		t.Fatalf("expired or superseded terms starved valid owner work: %#v", pendingTerms.Items)
	}
	if pendingTerms.Items[0].AcceptanceEventID == supersededAcceptance.ID ||
		pendingTerms.Items[0].AcceptanceEventID == expiredAcceptance.ID {
		t.Fatal("expired or superseded terms acceptance remained actionable")
	}
	if _, created, err := models.RecordVerifiedProviderTerms(
		db,
		models.VerifiedProviderTermsInput{
			ProviderOfferID:           validOffer.ID,
			ProviderAcceptanceEventID: validAcceptance.ID,
			SourceSystem:              "pg-contract",
			SourceEventID:             "queue-valid-terms",
			SourceEffectiveAt:         validAcceptance.ProviderAcceptedAt,
			OperatorReference:         "operator:queue-valid-terms",
			OwnerEvidenceReference:    "evidence:queue-valid-terms",
		},
	); err != nil || !created {
		t.Fatalf("clean up valid pending terms: created=%t err=%v", created, err)
	}
	if _, err := activatePostgresProviderOffer(
		t, db, validOffer.ID, "operator:queue-valid-terms", "evidence:queue-valid-terms",
	); err != nil {
		t.Fatalf("activate fresh callback queue fixture: %v", err)
	}

	// The oldest unanswered handoff is already expired. Add a newer handoff for
	// a provider whose DNS verification then becomes stale; neither row may
	// starve a fresh provider callback from the bounded owner queue.
	staleProvider := createPostgresCommercialProvider(t, db, "queue-stale-callback")
	staleOffer := createPostgresCommercialOffer(
		t, db, staleProvider, "queue-stale-callback", "prepaid", 1_000,
	)
	recordPostgresVerifiedFunding(
		t, db, staleOffer.ID, 1_000, "queue-stale-callback", "", time.Time{},
	)
	if _, err := activatePostgresProviderOffer(
		t, db, staleOffer.ID, "operator:queue-stale-callback", "evidence:queue-stale-callback",
	); err != nil {
		t.Fatalf("activate stale callback queue fixture: %v", err)
	}
	staleTicket, staleToken := createPostgresUnhandedActionTicket(
		t, db, signer, staleProvider.site, staleOffer.ID, "queue-stale-callback",
	)
	recordPostgresTicketReview(t, db, staleTicket, "queue-stale-callback")
	if _, _, err := models.RecordActionTicketHandoff(db, models.ProviderActionHandoffInput{
		ActionTicketID:          staleTicket.ID,
		AttributionToken:        staleToken,
		PrincipalHandoffConsent: true,
		HandoffConsentVersion:   models.ProviderActionHandoffConsentV1,
	}); err != nil {
		t.Fatalf("record stale queue handoff: %v", err)
	}
	if _, err := db.Exec(`
		UPDATE provider_claims
		SET verification_last_succeeded_at=clock_timestamp()-INTERVAL '8 days',
		    updated_at=clock_timestamp()
		WHERE id=$1::uuid`, staleProvider.claim.ID); err != nil {
		t.Fatalf("stale callback claim fixture: %v", err)
	}
	freshTicket, freshToken := createPostgresUnhandedActionTicket(
		t, db, signer, validProvider.site, validOffer.ID, "queue-fresh-callback",
	)
	recordPostgresTicketReview(t, db, freshTicket, "queue-fresh-callback")
	if _, _, err := models.RecordActionTicketHandoff(db, models.ProviderActionHandoffInput{
		ActionTicketID:          freshTicket.ID,
		AttributionToken:        freshToken,
		PrincipalHandoffConsent: true,
		HandoffConsentVersion:   models.ProviderActionHandoffConsentV1,
	}); err != nil {
		t.Fatalf("record fresh queue handoff: %v", err)
	}
	pendingCallbacks, err := models.GetProviderPilotQueue(db, "handoff_awaiting_callback", 1)
	if err != nil {
		t.Fatalf("read pending callback queue after expiry: %v", err)
	}
	if len(pendingCallbacks.Items) != 1 ||
		pendingCallbacks.Items[0].TicketID != freshTicket.ID {
		t.Fatalf("expired handoff starved fresh callback: %#v", pendingCallbacks.Items)
	}
	if pendingCallbacks.Items[0].TicketID == expiredHandoffTicketID {
		t.Fatal("expired handoff remained in callback queue")
	}
	if pendingCallbacks.Items[0].TicketID == staleTicket.ID {
		t.Fatal("DNS-stale provider handoff remained in callback queue")
	}
	if _, err := db.Exec(`
		UPDATE provider_claims
		SET verification_last_succeeded_at=clock_timestamp(),
		    verification_last_attempted_at=clock_timestamp(),
		    verification_next_check_at=clock_timestamp()+INTERVAL '6 hours',
		    updated_at=clock_timestamp()
		WHERE id=$1::uuid`, staleProvider.claim.ID); err != nil {
		t.Fatalf("refresh stale callback claim fixture: %v", err)
	}
	recordPostgresOutcome(
		t, db, signer, staleProvider.key, staleTicket.ID, "rejected", "queue-stale-callback-release",
	)
	recordPostgresOutcome(
		t, db, signer, validProvider.key, freshTicket.ID, "rejected", "queue-fresh-callback-release",
	)
}

func exerciseActionInterestPostgres(t *testing.T, db *sql.DB, site models.Site) {
	t.Helper()
	columnRows, err := db.Query(`
		SELECT column_name
		FROM information_schema.columns
		WHERE table_schema='public' AND table_name='action_interest_receipts'
		ORDER BY ordinal_position`)
	if err != nil {
		t.Fatalf("read action-interest catalog columns: %v", err)
	}
	var columns []string
	for columnRows.Next() {
		var column string
		if err := columnRows.Scan(&column); err != nil {
			_ = columnRows.Close()
			t.Fatalf("scan action-interest catalog column: %v", err)
		}
		columns = append(columns, column)
	}
	if err := columnRows.Err(); err != nil {
		_ = columnRows.Close()
		t.Fatalf("iterate action-interest catalog columns: %v", err)
	}
	if err := columnRows.Close(); err != nil {
		t.Fatalf("close action-interest catalog columns: %v", err)
	}
	wantColumns := []string{
		"id", "public_id", "search_receipt_id", "source_is_synthetic",
		"site_domain_snapshot", "action_type", "surface",
		"caller_attests_principal_interest", "confirmation_version",
		"created_at", "expires_at", "stage1_integrity_generation",
	}
	if !reflect.DeepEqual(columns, wantColumns) {
		t.Fatalf("action-interest catalog columns = %v, want %v", columns, wantColumns)
	}

	recordSearch := func(synthetic bool, returned []models.Site) string {
		t.Helper()
		searchID, err := models.GenerateDemandSearchID()
		if err != nil {
			t.Fatalf("generate action-interest search ID: %v", err)
		}
		if err := models.RecordDemandSearch(db, models.DemandSearchReceipt{
			PublicID: searchID, Surface: "rest", Category: "developer",
			ResultCount: len(returned), Page: 1, PageSize: 10, Synthetic: synthetic,
		}, returned); err != nil {
			t.Fatalf("record action-interest search: %v", err)
		}
		return searchID
	}
	inputFor := func(searchID, action string) models.ActionInterestInput {
		return models.ActionInterestInput{
			SearchID: searchID, Domain: site.Domain, ActionType: action,
			Surface: "rest", CallerAttestsPrincipalInterest: true,
			ConfirmationVersion: models.ActionInterestConfirmationV1,
		}
	}

	searchID := recordSearch(false, []models.Site{site})
	created, err := models.RecordActionInterest(db, inputFor(searchID, "quote"))
	if err != nil {
		t.Fatalf("create action interest: %v", err)
	}
	if created.Replayed || !strings.HasPrefix(created.PublicID, "nhs_air_") || created.Domain != site.Domain {
		t.Fatalf("new action-interest receipt = %#v", created)
	}
	if ttl := created.ExpiresAt.Sub(created.CreatedAt); ttl <= 29*24*time.Hour || ttl > 30*24*time.Hour {
		t.Fatalf("action-interest TTL = %v, want source-bounded <=30 days", ttl)
	}
	replayInput := inputFor(searchID, "quote")
	replayInput.Surface = "mcp"
	replayed, err := models.RecordActionInterest(db, replayInput)
	if err != nil {
		t.Fatalf("replay action interest: %v", err)
	}
	if !replayed.Replayed || replayed.PublicID != created.PublicID || replayed.Surface != "rest" {
		t.Fatalf("action-interest replay = %#v, want stable original %#v", replayed, created)
	}
	if _, err := models.RecordActionInterest(db, inputFor(searchID, "demo")); !errors.Is(err, models.ErrActionInterestConflict) {
		t.Fatalf("changed action-interest error = %v, want conflict", err)
	}

	for name, input := range map[string]models.ActionInterestInput{
		"false confirmation": func() models.ActionInterestInput {
			v := inputFor(searchID, "quote")
			v.CallerAttestsPrincipalInterest = false
			return v
		}(),
		"wrong version": func() models.ActionInterestInput {
			v := inputFor(searchID, "quote")
			v.ConfirmationVersion = "nhs-action-interest-v2"
			return v
		}(),
		"unknown action": inputFor(searchID, "lead"),
		"url domain": func() models.ActionInterestInput {
			v := inputFor(searchID, "quote")
			v.Domain = "https://" + site.Domain + "/private?token=secret"
			return v
		}(),
	} {
		if _, err := models.RecordActionInterest(db, input); !errors.Is(err, models.ErrInvalidActionInterest) {
			t.Fatalf("%s error = %v, want invalid action interest", name, err)
		}
	}

	if _, err := models.RecordActionInterest(db, inputFor(recordSearch(false, nil), "quote")); !errors.Is(err, models.ErrActionInterestUnavailable) {
		t.Fatalf("non-returned domain error = %v, want unavailable", err)
	}
	syntheticID := recordSearch(true, []models.Site{site})
	if _, err := models.RecordActionInterest(db, inputFor(syntheticID, "quote")); !errors.Is(err, models.ErrActionInterestUnavailable) {
		t.Fatalf("synthetic source error = %v, want unavailable", err)
	}
	if _, err := db.Exec(`
		INSERT INTO action_interest_receipts (
			public_id, search_receipt_id, source_is_synthetic,
			site_domain_snapshot, action_type, surface,
			caller_attests_principal_interest, confirmation_version, expires_at
		)
		SELECT 'nhs_air_AAAAAAAAAAAAAAAA', receipt.id, false,
		       returned.site_domain_snapshot, 'quote', 'rest', true,
		       'nhs-action-interest-v1', NOW() + INTERVAL '1 day'
		FROM search_receipts receipt
		JOIN organic_results_returned returned ON returned.search_receipt_id=receipt.id
		WHERE receipt.public_id=$1`, syntheticID); err == nil {
		t.Fatal("database accepted an action interest bound to a synthetic search receipt")
	} else {
		var pqErr *pq.Error
		if !errors.As(err, &pqErr) || string(pqErr.Code) != "23503" || pqErr.Constraint != "action_interest_receipts_non_synthetic_fk" {
			t.Fatalf("synthetic direct insert error = %#v, want 23503 action_interest_receipts_non_synthetic_fk", err)
		}
	}

	ttlConstraintSearchID := recordSearch(false, []models.Site{site})
	assertTTLConstraint := func(publicID, ttl, wantConstraint string) {
		t.Helper()
		err := runPostgresStage1FixtureMutation(t, db,
			`ALTER TABLE action_interest_receipts DISABLE TRIGGER stage1_action_interest_insert_timestamp_owned`,
			`ALTER TABLE action_interest_receipts ENABLE TRIGGER stage1_action_interest_insert_timestamp_owned`, `
			WITH fixture_clock AS (
				SELECT clock_timestamp() AS at
			)
			INSERT INTO action_interest_receipts (
				public_id, search_receipt_id, source_is_synthetic,
				site_domain_snapshot, action_type, surface,
				caller_attests_principal_interest, confirmation_version,
				created_at, expires_at
			)
			SELECT $2, receipt.id, false,
			       returned.site_domain_snapshot, 'quote', 'rest', true,
			       'nhs-action-interest-v1', fixture_clock.at,
			       fixture_clock.at + $3::interval
			FROM search_receipts receipt
			JOIN organic_results_returned returned
			  ON returned.search_receipt_id=receipt.id
			CROSS JOIN fixture_clock
			WHERE receipt.public_id=$1`, ttlConstraintSearchID, publicID, ttl)
		if err == nil {
			t.Fatalf("database accepted action-interest TTL %q; want %s violation", ttl, wantConstraint)
		}
		var pqErr *pq.Error
		if !errors.As(err, &pqErr) || string(pqErr.Code) != "23514" || pqErr.Constraint != wantConstraint {
			t.Fatalf("action-interest TTL %q error = %#v, want 23514 %s", ttl, err, wantConstraint)
		}
	}
	assertTTLConstraint(
		"nhs_air_BBBBBBBBBBBBBBBB",
		"0 seconds",
		"action_interest_receipts_positive_ttl",
	)
	assertTTLConstraint(
		"nhs_air_CCCCCCCCCCCCCCCC",
		"30 days 1 second",
		"action_interest_receipts_max_ttl",
	)

	staleID := recordSearch(false, []models.Site{site})
	mutatePostgresStage1SearchFixture(t, db,
		`UPDATE search_receipts SET created_at=NOW()-INTERVAL '31 days' WHERE public_id=$1`,
		staleID,
	)
	if _, err := models.RecordActionInterest(db, inputFor(staleID, "trial")); !errors.Is(err, models.ErrActionInterestUnavailable) {
		t.Fatalf("stale source error = %v, want unavailable", err)
	}

	disposableSite := models.Site{
		Domain: "action-interest-delete.example",
		URL:    "https://action-interest-delete.example",
	}
	if err := db.QueryRow(`
		INSERT INTO sites (domain, url, name, description, category, crawl_status)
		VALUES ($1,$2,'Disposable interest site','FK referential-action fixture','developer','success')
		RETURNING id::text`, disposableSite.Domain, disposableSite.URL).Scan(&disposableSite.ID); err != nil {
		t.Fatalf("insert disposable action-interest site: %v", err)
	}
	deleteSearchID := recordSearch(false, []models.Site{disposableSite})
	deleteReceipt, err := models.RecordActionInterest(db, models.ActionInterestInput{
		SearchID: deleteSearchID, Domain: disposableSite.Domain, ActionType: "quote",
		Surface: "rest", CallerAttestsPrincipalInterest: true,
		ConfirmationVersion: models.ActionInterestConfirmationV1,
	})
	if err != nil {
		t.Fatalf("create disposable-site action interest: %v", err)
	}
	if _, err := db.Exec(`DELETE FROM sites WHERE id=$1::uuid`, disposableSite.ID); err != nil {
		t.Fatalf("delete site referenced by action-interest source: %v", err)
	}
	var remainingInterest int
	var returnedSiteID sql.NullString
	if err := db.QueryRow(`SELECT COUNT(*)::int FROM action_interest_receipts WHERE public_id=$1`, deleteReceipt.PublicID).Scan(&remainingInterest); err != nil {
		t.Fatalf("count action interest after site delete: %v", err)
	}
	if err := db.QueryRow(`
		SELECT returned.site_id::text
		FROM organic_results_returned returned
		JOIN search_receipts receipt ON receipt.id=returned.search_receipt_id
		WHERE receipt.public_id=$1 AND returned.site_domain_snapshot=$2`, deleteSearchID, disposableSite.Domain).Scan(&returnedSiteID); err != nil {
		t.Fatalf("read returned result after site delete: %v", err)
	}
	if remainingInterest != 1 || returnedSiteID.Valid {
		t.Fatalf("site delete result interest_count=%d returned_site_id=%v, want 1 and NULL", remainingInterest, returnedSiteID)
	}

	expirySearchID := recordSearch(false, []models.Site{site})
	mutatePostgresStage1SearchFixture(t, db, `
		UPDATE search_receipts
		SET created_at=clock_timestamp()-INTERVAL '30 days'+INTERVAL '2 seconds'
		WHERE public_id=$1`, expirySearchID)
	blockingTx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin expiry row lock: %v", err)
	}
	var returnedID int64
	if err := blockingTx.QueryRow(`
		SELECT returned.id
		FROM organic_results_returned returned
		JOIN search_receipts receipt ON receipt.id=returned.search_receipt_id
		WHERE receipt.public_id=$1 AND returned.site_domain_snapshot=$2
		FOR UPDATE OF returned`, expirySearchID, site.Domain).Scan(&returnedID); err != nil {
		_ = blockingTx.Rollback()
		t.Fatalf("lock returned result across expiry: %v", err)
	}
	expiryStarted := make(chan struct{})
	expiryResult := make(chan error, 1)
	go func() {
		close(expiryStarted)
		_, recordErr := models.RecordActionInterest(db, inputFor(expirySearchID, "trial"))
		expiryResult <- recordErr
	}()
	<-expiryStarted
	time.Sleep(2200 * time.Millisecond)
	if err := blockingTx.Commit(); err != nil {
		t.Fatalf("release expiry row lock: %v", err)
	}
	if err := <-expiryResult; !errors.Is(err, models.ErrActionInterestUnavailable) {
		t.Fatalf("lock-across-expiry error = %v, want unavailable", err)
	}
	var expiredInsertCount int
	if err := db.QueryRow(`
		SELECT COUNT(*)::int
		FROM action_interest_receipts interest
		JOIN search_receipts receipt ON receipt.id=interest.search_receipt_id
		WHERE receipt.public_id=$1`, expirySearchID).Scan(&expiredInsertCount); err != nil {
		t.Fatalf("count lock-across-expiry inserts: %v", err)
	}
	if expiredInsertCount != 0 {
		t.Fatalf("lock-across-expiry created %d receipt(s), want 0", expiredInsertCount)
	}

	concurrentID := recordSearch(false, []models.Site{site})
	start := make(chan struct{})
	type result struct {
		receipt *models.ActionInterestReceipt
		err     error
	}
	results := make(chan result, 2)
	var wait sync.WaitGroup
	for range 2 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			receipt, err := models.RecordActionInterest(db, inputFor(concurrentID, "application"))
			results <- result{receipt: receipt, err: err}
		}()
	}
	close(start)
	wait.Wait()
	close(results)
	var publicID string
	createdCount, replayCount := 0, 0
	for result := range results {
		if result.err != nil {
			t.Fatalf("concurrent action interest: %v", result.err)
		}
		if publicID == "" {
			publicID = result.receipt.PublicID
		} else if result.receipt.PublicID != publicID {
			t.Fatalf("concurrent action interests returned %q and %q", publicID, result.receipt.PublicID)
		}
		if result.receipt.Replayed {
			replayCount++
		} else {
			createdCount++
		}
	}
	if createdCount != 1 || replayCount != 1 {
		t.Fatalf("concurrent action-interest outcomes created=%d replayed=%d", createdCount, replayCount)
	}

	cascadeID := recordSearch(false, []models.Site{site})
	cascadeReceipt, err := models.RecordActionInterest(db, inputFor(cascadeID, "booking"))
	if err != nil {
		t.Fatalf("create cascade action interest: %v", err)
	}
	if _, err := db.Exec(`DELETE FROM search_receipts WHERE public_id=$1`, cascadeID); err != nil {
		t.Fatalf("delete cascade search receipt: %v", err)
	}
	var cascadeCount int
	if err := db.QueryRow(`SELECT COUNT(*)::int FROM action_interest_receipts WHERE public_id=$1`, cascadeReceipt.PublicID).Scan(&cascadeCount); err != nil {
		t.Fatalf("count cascaded action interest: %v", err)
	}
	if cascadeCount != 0 {
		t.Fatalf("search receipt deletion left %d action-interest rows", cascadeCount)
	}

	stage1, err := models.GetStage1DemandProof(db, 30)
	if err != nil {
		t.Fatalf("stage 1 demand proof: %v", err)
	}
	if stage1.CommercialProof || !stage1.SyntheticExcluded || !stage1.CountsAreReceiptsNotUniqueAgents ||
		stage1.SearchReceiptsWithActionInterest < 2 || stage1.ObservationWindowMet || stage1.Stage1Ready {
		t.Fatalf("stage 1 proof contract = %#v", stage1)
	}
	if len(stage1.ActionTypes) != 0 || len(stage1.DemandTopics) != 0 {
		t.Fatalf("below-threshold segmented buckets leaked: actions=%v topics=%v", stage1.ActionTypes, stage1.DemandTopics)
	}

	// Put the provider's quote cohort exactly at the privacy threshold so one
	// expired-but-not-yet-deleted row would be observable in both owner and
	// provider analytics if either query forgot the logical expiry predicate.
	var activeQuoteInterests int
	if err := db.QueryRow(`
		SELECT COUNT(*)::int
		FROM action_interest_receipts
		WHERE site_domain_snapshot=$1
		  AND action_type='quote'
		  AND expires_at > clock_timestamp()`, site.Domain).Scan(&activeQuoteInterests); err != nil {
		t.Fatalf("count active quote interests: %v", err)
	}
	for activeQuoteInterests < models.ProviderDemandPrivacyThreshold {
		thresholdSearchID := recordSearch(false, []models.Site{site})
		if _, err := models.RecordActionInterest(db, inputFor(thresholdSearchID, "quote")); err != nil {
			t.Fatalf("create threshold quote interest %d: %v", activeQuoteInterests+1, err)
		}
		activeQuoteInterests++
	}

	expiredFixtureSearchID := recordSearch(false, []models.Site{site})
	stage1BeforeExpiredFixture, err := models.GetStage1DemandProof(db, 30)
	if err != nil {
		t.Fatalf("stage 1 proof before expired fixture: %v", err)
	}
	providerBeforeExpiredFixture, err := models.GetProviderDemandAnalytics(db, site.Domain, 30)
	if err != nil {
		t.Fatalf("provider analytics before expired fixture: %v", err)
	}
	providerSummary, ok := providerBeforeExpiredFixture["summary"].(map[string]any)
	if !ok {
		t.Fatalf("provider analytics summary type = %T", providerBeforeExpiredFixture["summary"])
	}
	providerInterestCount, countOK := providerSummary["action_interest_receipts"].(int)
	if !countOK || providerInterestCount < models.ProviderDemandPrivacyThreshold || providerSummary["action_interest_suppressed"] != false {
		t.Fatalf("provider action-interest threshold fixture not observable: %#v", providerSummary)
	}
	providerActionTypes, ok := providerBeforeExpiredFixture["action_types"].([]map[string]any)
	if !ok {
		t.Fatalf("provider action_types type = %T", providerBeforeExpiredFixture["action_types"])
	}
	quoteBucketVisible := false
	for _, bucket := range providerActionTypes {
		if bucket["action_type"] == "quote" {
			count, ok := bucket["receipt_count"].(int)
			quoteBucketVisible = ok && count >= models.ProviderDemandPrivacyThreshold
			break
		}
	}
	if !quoteBucketVisible {
		t.Fatalf("provider quote threshold bucket missing: %#v", providerActionTypes)
	}

	insertPostgresHistoricalActionInterestFixture(t, db, `
		WITH fixture_clock AS (
			SELECT clock_timestamp() AS at
		)
		INSERT INTO action_interest_receipts (
			public_id, search_receipt_id, source_is_synthetic,
			site_domain_snapshot, action_type, surface,
			caller_attests_principal_interest, confirmation_version,
			created_at, expires_at
		)
		SELECT 'nhs_air_DDDDDDDDDDDDDDDD', receipt.id, false,
		       returned.site_domain_snapshot, 'quote', 'rest', true,
		       'nhs-action-interest-v1', fixture_clock.at-INTERVAL '2 days',
		       fixture_clock.at-INTERVAL '1 day'
		FROM search_receipts receipt
		JOIN organic_results_returned returned
		  ON returned.search_receipt_id=receipt.id
		CROSS JOIN fixture_clock
		WHERE receipt.public_id=$1`, expiredFixtureSearchID)
	var expiredFixtureRows int
	if err := db.QueryRow(`
		SELECT COUNT(*)::int
		FROM action_interest_receipts
		WHERE public_id='nhs_air_DDDDDDDDDDDDDDDD'
		  AND expires_at <= clock_timestamp()`).Scan(&expiredFixtureRows); err != nil {
		t.Fatalf("confirm physically present expired fixture: %v", err)
	}
	if expiredFixtureRows != 1 {
		t.Fatalf("physically present expired fixture rows = %d, want 1", expiredFixtureRows)
	}

	stage1AfterExpiredFixture, err := models.GetStage1DemandProof(db, 30)
	if err != nil {
		t.Fatalf("stage 1 proof after expired fixture: %v", err)
	}
	stage1BeforeExpiredFixture.AsOf = time.Time{}
	stage1AfterExpiredFixture.AsOf = time.Time{}
	if !reflect.DeepEqual(stage1BeforeExpiredFixture, stage1AfterExpiredFixture) {
		t.Fatalf("expired row changed stage 1 analytics: before=%#v after=%#v", stage1BeforeExpiredFixture, stage1AfterExpiredFixture)
	}
	providerAfterExpiredFixture, err := models.GetProviderDemandAnalytics(db, site.Domain, 30)
	if err != nil {
		t.Fatalf("provider analytics after expired fixture: %v", err)
	}
	if !reflect.DeepEqual(providerBeforeExpiredFixture, providerAfterExpiredFixture) {
		t.Fatalf("expired row changed provider analytics: before=%#v after=%#v", providerBeforeExpiredFixture, providerAfterExpiredFixture)
	}

	// Build a clean exact Stage 1 boundary matrix. The protected migration-025
	// receipt is deliberately backdated only inside this disposable database so
	// real elapsed-time boundaries can be exercised without a 14-day test run.
	if _, err := db.Exec(`DELETE FROM search_receipts`); err != nil {
		t.Fatalf("reset Stage 1 boundary fixture: %v", err)
	}
	var originalStage1StartedAt time.Time
	if err := db.QueryRow(`
		SELECT applied_at FROM nhs_schema_migrations
		WHERE name='025_stage1_fact_integrity.sql'`).Scan(&originalStage1StartedAt); err != nil {
		t.Fatalf("read protected Stage 1 epoch: %v", err)
	}
	mutatePostgresStage1LedgerFixture(t, db, `
		UPDATE nhs_schema_migrations
		SET applied_at=clock_timestamp()-INTERVAL '20 days'
		WHERE name='025_stage1_fact_integrity.sql'`)

	boundarySearch := func(label, category string, synthetic bool, returned []models.Site, resultCount int) string {
		t.Helper()
		searchID, err := models.GenerateDemandSearchID()
		if err != nil {
			t.Fatalf("generate Stage 1 boundary search %s: %v", label, err)
		}
		if err := models.RecordDemandSearch(db, models.DemandSearchReceipt{
			PublicID: searchID, Surface: "rest", Category: category,
			ResultCount: resultCount, Page: 1, PageSize: 10, Synthetic: synthetic,
		}, returned); err != nil {
			t.Fatalf("record Stage 1 boundary search %s: %v", label, err)
		}
		return searchID
	}
	// A pilot topic needs breadth, not merely repeated traffic to one domain.
	// Keep the tenth returned domain classified as spam until the exact
	// receipt-count boundary has been observed so the fixture proves both
	// halves of the candidate gate independently.
	stage1TopicSites := make([]models.Site, 0, models.Stage1CandidateTopicDomains-1)
	for index := 0; index < models.Stage1CandidateTopicDomains-1; index++ {
		candidateSite := models.Site{
			Domain: fmt.Sprintf("stage1-topic-%02d.example", index+1),
			URL:    fmt.Sprintf("https://stage1-topic-%02d.example", index+1),
		}
		category := "developer"
		if index == models.Stage1CandidateTopicDomains-2 {
			category = "spam"
		}
		if err := db.QueryRow(`
			INSERT INTO sites (
				domain, url, name, description, has_structured_api,
				agentic_score, category, crawl_status
			) VALUES ($1,$2,$3,'Stage 1 topic breadth fixture',true,90,$4,'success')
			RETURNING id::text`,
			candidateSite.Domain, candidateSite.URL,
			fmt.Sprintf("Stage 1 topic %02d", index+1), category,
		).Scan(&candidateSite.ID); err != nil {
			t.Fatalf("insert Stage 1 topic site %d: %v", index+1, err)
		}
		stage1TopicSites = append(stage1TopicSites, candidateSite)
	}

	historicalID := boundarySearch("pre-epoch", "developer", false, []models.Site{site}, 1)
	if matched, err := models.RecordDemandSelection(db, historicalID, site.Domain, "rest"); err != nil || !matched {
		t.Fatalf("record pre-epoch selection matched=%t err=%v", matched, err)
	}
	if _, err := models.RecordActionInterest(db, inputFor(historicalID, "quote")); err != nil {
		t.Fatalf("record pre-epoch action interest: %v", err)
	}
	mutatePostgresStage1SearchFixture(t, db, `
		UPDATE search_receipts
		SET created_at=(SELECT applied_at-INTERVAL '1 second'
		                FROM nhs_schema_migrations
		                WHERE name='025_stage1_fact_integrity.sql')
		WHERE public_id=$1`, historicalID)
	_ = boundarySearch("empty-page-total", "developer", false, nil, 100)
	_ = boundarySearch("synthetic-return", "developer", true, []models.Site{site}, 1)

	eligibleIDs := make([]string, 0, 100)
	for index := 0; index < 99; index++ {
		category := ""
		returnedSites := []models.Site{site}
		if index < 19 {
			category = "developer"
			topicSiteIndex := index % (len(stage1TopicSites) - 1)
			if index == 18 {
				// The tenth distinct domain must occur on exactly one row so
				// moving that row past the cutoff removes the breadth proof.
				topicSiteIndex = len(stage1TopicSites) - 1
			}
			returnedSites = append(
				returnedSites,
				stage1TopicSites[topicSiteIndex],
			)
		}
		eligibleIDs = append(eligibleIDs, boundarySearch(
			fmt.Sprintf("eligible-%03d", index+1), category, false,
			returnedSites, len(returnedSites),
		))
	}
	mutatePostgresStage1SearchFixture(t, db, `
		UPDATE search_receipts SET created_at=clock_timestamp()-INTERVAL '13 days'
		WHERE public_id=$1`, eligibleIDs[0])
	for index := 0; index < 19; index++ {
		if matched, err := models.RecordDemandSelection(db, eligibleIDs[index], site.Domain, "rest"); err != nil || !matched {
			t.Fatalf("record boundary selection %d matched=%t err=%v", index+1, matched, err)
		}
	}
	for index := 0; index < 9; index++ {
		if _, err := models.RecordActionInterest(db, inputFor(eligibleIDs[index], "quote")); err != nil {
			t.Fatalf("record boundary action interest %d: %v", index+1, err)
		}
	}

	// The migration-level exact returned-result FK must reject a selection for a
	// domain that was not in the receipt's organic response.
	if _, err := db.Exec(`
		INSERT INTO result_selections (
			search_receipt_id, site_domain_snapshot, surface
		)
		SELECT id, 'not-returned.example', 'rest'
		FROM search_receipts WHERE public_id=$1`, eligibleIDs[0]); err == nil {
		t.Fatal("database accepted a Stage 1 selection for a non-returned domain")
	} else {
		assertPostgresConstraintCode(t, err, "23503", "result_selections_returned_result_fk")
	}

	// A physically present but expired interest must not enter the as-of cohort.
	insertPostgresHistoricalActionInterestFixture(t, db, `
		INSERT INTO action_interest_receipts (
			public_id, search_receipt_id, source_is_synthetic,
			site_domain_snapshot, action_type, surface,
			caller_attests_principal_interest, confirmation_version,
			created_at, expires_at
		)
		SELECT 'nhs_air_EEEEEEEEEEEEEEEE', receipt.id, false,
		       returned.site_domain_snapshot, 'quote', 'rest', true,
		       'nhs-action-interest-v1', clock_timestamp()-INTERVAL '2 days',
		       clock_timestamp()-INTERVAL '1 day'
		FROM search_receipts receipt
		JOIN organic_results_returned returned ON returned.search_receipt_id=receipt.id
		WHERE receipt.public_id=$1`, eligibleIDs[98])

	belowAll, err := models.GetStage1DemandProof(db, 30)
	if err != nil {
		t.Fatalf("read below-all Stage 1 boundary: %v", err)
	}
	if belowAll.MeaningfulSearchReceipts != 100-1 || belowAll.SearchReceiptsWithSelection != 19 ||
		belowAll.SearchReceiptsWithActionInterest != 9 || belowAll.PilotCandidateTopicAvailable ||
		belowAll.ObservationWindowMet || belowAll.Stage1Ready {
		t.Fatalf("below-all Stage 1 boundary = %#v", belowAll)
	}

	eligibleIDs = append(eligibleIDs, boundarySearch(
		"eligible-100", "developer", false,
		[]models.Site{site, stage1TopicSites[0]}, 2,
	))
	atHundred, err := models.GetStage1DemandProof(db, 30)
	if err != nil {
		t.Fatalf("read 100-search Stage 1 boundary: %v", err)
	}
	if atHundred.MeaningfulSearchReceipts != 100 || !atHundred.TargetsMet["meaningful_search_receipts"] ||
		atHundred.TargetsMet["search_receipts_with_selection"] ||
		atHundred.TargetsMet["search_receipts_with_action_interest"] ||
		atHundred.TargetsMet["pilot_candidate_topic_receipts"] ||
		atHundred.TargetsMet["observation_window_days"] || atHundred.Stage1Ready {
		t.Fatalf("100-search-only Stage 1 boundary = %#v", atHundred)
	}
	if _, err := db.Exec(`
		UPDATE sites SET category='developer'
		WHERE id=$1::uuid`, stage1TopicSites[len(stage1TopicSites)-1].ID); err != nil {
		t.Fatalf("restore tenth non-spam Stage 1 topic domain: %v", err)
	}
	uniqueTopicReceiptID := eligibleIDs[18]
	uniqueTopicSite := stage1TopicSites[len(stage1TopicSites)-1]
	var originalTopicReturnedAt time.Time
	if err := db.QueryRow(`
		SELECT returned.returned_at
		FROM organic_results_returned returned
		JOIN search_receipts receipt ON receipt.id=returned.search_receipt_id
		WHERE receipt.public_id=$1
		  AND returned.site_id=$2::uuid
		  AND returned.site_domain_snapshot=$3`,
		uniqueTopicReceiptID, uniqueTopicSite.ID, uniqueTopicSite.Domain,
	).Scan(&originalTopicReturnedAt); err != nil {
		t.Fatalf("read unique tenth-domain returned-result clock: %v", err)
	}
	postCutoffReturnedAt := atHundred.AsOf.Add(24 * time.Hour)
	mutatePostgresStage1OrganicFixture(t, db, `
		UPDATE organic_results_returned returned
		SET returned_at=$4
		FROM search_receipts receipt
		WHERE receipt.id=returned.search_receipt_id
		  AND receipt.public_id=$1
		  AND returned.site_id=$2::uuid
		  AND returned.site_domain_snapshot=$3`,
		uniqueTopicReceiptID, uniqueTopicSite.ID, uniqueTopicSite.Domain,
		postCutoffReturnedAt)

	if matched, err := models.RecordDemandSelection(db, eligibleIDs[19], site.Domain, "rest"); err != nil || !matched {
		t.Fatalf("record twentieth Stage 1 selection matched=%t err=%v", matched, err)
	}
	if _, err := models.RecordActionInterest(db, inputFor(eligibleIDs[9], "quote")); err != nil {
		t.Fatalf("record tenth Stage 1 action interest: %v", err)
	}
	mutatePostgresStage1SearchFixture(t, db, `
		UPDATE search_receipts SET created_at=clock_timestamp()-INTERVAL '14 days 1 second'
		WHERE public_id=$1`, eligibleIDs[0])

	lateResultProof, err := models.GetStage1DemandProof(db, 30)
	if err != nil {
		t.Fatalf("read post-cutoff tenth-domain Stage 1 boundary: %v", err)
	}
	if lateResultProof.MeaningfulSearchReceipts != 100 ||
		lateResultProof.SearchReceiptsWithSelection != 20 ||
		lateResultProof.SearchReceiptsWithActionInterest != 10 ||
		!lateResultProof.ObservationWindowMet ||
		lateResultProof.PilotCandidateTopicAvailable ||
		lateResultProof.TargetsMet["pilot_candidate_topic_receipts"] ||
		lateResultProof.Stage1Ready {
		t.Fatalf("post-cutoff tenth-domain Stage 1 boundary = %#v", lateResultProof)
	}
	for _, target := range []string{
		"meaningful_search_receipts",
		"search_receipts_with_selection",
		"search_receipts_with_action_interest",
		"observation_window_days",
	} {
		if !lateResultProof.TargetsMet[target] {
			t.Fatalf("post-cutoff tenth-domain unexpectedly missed %q: %#v", target, lateResultProof)
		}
	}
	var epochCountBefore int
	if err := db.QueryRow(`SELECT COUNT(*)::int FROM provider_pilot_epochs`).Scan(&epochCountBefore); err != nil {
		t.Fatalf("count epochs before post-cutoff candidate probe: %v", err)
	}
	if _, err := models.CreateProviderPilotEpoch(db, models.ProviderPilotEpochInput{
		DemandTopic:       "developer-tools",
		CohortLimit:       models.ProviderPilotMinimumCohort,
		ProviderTicketCap: 1,
		TotalTicketCap:    models.ProviderPilotMinimumTotalTickets,
		OwnerReference:    "owner:post-cutoff-domain-probe",
		EvidenceReference: "evidence:post-cutoff-domain-probe",
	}); !errors.Is(err, models.ErrProviderPilotStage1NotReady) {
		t.Fatalf("post-cutoff tenth-domain pilot error = %v, want ErrProviderPilotStage1NotReady", err)
	}
	if _, err := db.Exec(`
		INSERT INTO provider_pilot_epochs (
			contract_version, demand_topic, stage1_started_at,
			stage1_evidence_as_of, stage1_evidence_sha256, cohort_limit,
			provider_ticket_cap, total_ticket_cap, status,
			owner_reference, evidence_reference
		) VALUES ($1,'developer-tools',$2,$3,$4,10,1,10,'draft',$5,$6)`,
		models.ProviderPilotEpochContractV1, lateResultProof.Stage1StartedAt,
		lateResultProof.AsOf, strings.Repeat("0", 64),
		"owner:post-cutoff-direct-probe", "evidence:post-cutoff-direct-probe",
	); err == nil {
		t.Fatal("database accepted a pilot whose tenth topic domain arrived after its evidence cutoff")
	} else {
		assertPostgresConstraint(t, err, "provider_pilot_stage1_thresholds")
	}
	var epochCountAfter int
	if err := db.QueryRow(`SELECT COUNT(*)::int FROM provider_pilot_epochs`).Scan(&epochCountAfter); err != nil {
		t.Fatalf("count epochs after post-cutoff candidate probe: %v", err)
	}
	if epochCountAfter != epochCountBefore {
		t.Fatalf("post-cutoff candidate probe changed epoch count from %d to %d", epochCountBefore, epochCountAfter)
	}

	mutatePostgresStage1OrganicFixture(t, db, `
		UPDATE organic_results_returned returned
		SET returned_at=$4
		FROM search_receipts receipt
		WHERE receipt.id=returned.search_receipt_id
		  AND receipt.public_id=$1
		  AND returned.site_id=$2::uuid
		  AND returned.site_domain_snapshot=$3`,
		uniqueTopicReceiptID, uniqueTopicSite.ID, uniqueTopicSite.Domain,
		originalTopicReturnedAt)

	ready, err := models.GetStage1DemandProof(db, 30)
	if err != nil {
		t.Fatalf("read exact-ready Stage 1 boundary: %v", err)
	}
	if ready.MeaningfulSearchReceipts != 100 || ready.SearchReceiptsWithSelection != 20 ||
		ready.SearchReceiptsWithActionInterest != 10 || !ready.PilotCandidateTopicAvailable ||
		len(ready.PilotCandidateTopics) != 1 || ready.PilotCandidateTopics[0].Value != "developer-tools" ||
		ready.PilotCandidateTopics[0].ReceiptCount != 20 ||
		ready.ObservationSpanSeconds < int64(14*24*time.Hour/time.Second) ||
		!ready.ObservationWindowMet || !ready.Stage1Ready {
		t.Fatalf("exact-ready Stage 1 boundary = %#v", ready)
	}
	for target, met := range ready.TargetsMet {
		if !met {
			t.Fatalf("exact-ready Stage 1 target %q not met: %#v", target, ready)
		}
	}
	readySHA, err := models.ProviderPilotStage1SnapshotSHA256(ready)
	if err != nil {
		t.Fatalf("hash exact-ready Stage 1 boundary: %v", err)
	}
	readyEpochProbe, err := db.Begin()
	if err != nil {
		t.Fatalf("begin exact-ready epoch gate probe: %v", err)
	}
	var readyEpochID string
	err = readyEpochProbe.QueryRow(`
		INSERT INTO provider_pilot_epochs (
			contract_version, demand_topic, stage1_started_at,
			stage1_evidence_as_of, stage1_evidence_sha256, cohort_limit,
			provider_ticket_cap, total_ticket_cap, status,
			owner_reference, evidence_reference
		) VALUES ($1,'developer-tools',$2,$3,$4,10,1,10,'draft',$5,$6)
		RETURNING id::text`,
		models.ProviderPilotEpochContractV1, ready.Stage1StartedAt, ready.AsOf,
		readySHA, "owner:within-window-direct-probe",
		"evidence:within-window-direct-probe",
	).Scan(&readyEpochID)
	rollbackErr := readyEpochProbe.Rollback()
	if err != nil {
		t.Fatalf("database rejected within-window tenth-domain pilot gate: %v", err)
	}
	if rollbackErr != nil {
		t.Fatalf("rollback exact-ready epoch gate probe: %v", rollbackErr)
	}
	if readyEpochID == "" {
		t.Fatal("within-window tenth-domain pilot gate returned no epoch ID")
	}
	mutatePostgresStage1LedgerFixture(t, db, `
		UPDATE nhs_schema_migrations SET applied_at=$1
		WHERE name='025_stage1_fact_integrity.sql'`, originalStage1StartedAt)

}

// execPostgresStage1FixtureMutation is a disposable-database-only clock
// control. It disables one named 025 guard and performs the historical fixture
// mutation in the same transaction, then restores the guard before commit.
// This helper exists only in a _test.go file; no production binary or operator
// surface can backdate Stage 1 facts through it.
func runPostgresStage1FixtureMutation(
	t *testing.T,
	db *sql.DB,
	disableGuard, enableGuard, query string,
	args ...any,
) error {
	t.Helper()
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin disposable Stage 1 fixture mutation: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.Exec(disableGuard); err != nil {
		return fmt.Errorf("disable disposable Stage 1 fixture guard: %w", err)
	}
	if _, err := tx.Exec(query, args...); err != nil {
		return err
	}
	if _, err := tx.Exec(enableGuard); err != nil {
		return fmt.Errorf("restore disposable Stage 1 fixture guard: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit disposable Stage 1 fixture mutation: %w", err)
	}
	return nil
}

func execPostgresStage1FixtureMutation(
	t *testing.T,
	db *sql.DB,
	disableGuard, enableGuard, query string,
	args ...any,
) {
	t.Helper()
	if err := runPostgresStage1FixtureMutation(
		t, db, disableGuard, enableGuard, query, args...,
	); err != nil {
		t.Fatalf("mutate disposable Stage 1 fixture: %v", err)
	}
}

func mutatePostgresStage1SearchFixture(
	t *testing.T,
	db *sql.DB,
	query string,
	args ...any,
) {
	t.Helper()
	execPostgresStage1FixtureMutation(
		t, db,
		`ALTER TABLE search_receipts DISABLE TRIGGER search_receipt_stage1_immutability_enforced`,
		`ALTER TABLE search_receipts ENABLE TRIGGER search_receipt_stage1_immutability_enforced`,
		query, args...,
	)
}

func mutatePostgresStage1OrganicFixture(
	t *testing.T,
	db *sql.DB,
	query string,
	args ...any,
) {
	t.Helper()
	execPostgresStage1FixtureMutation(
		t, db,
		`ALTER TABLE organic_results_returned DISABLE TRIGGER organic_result_stage1_immutability_enforced`,
		`ALTER TABLE organic_results_returned ENABLE TRIGGER organic_result_stage1_immutability_enforced`,
		query, args...,
	)
}

func insertPostgresHistoricalActionInterestFixture(
	t *testing.T,
	db *sql.DB,
	query string,
	args ...any,
) {
	t.Helper()
	execPostgresStage1FixtureMutation(
		t, db,
		`ALTER TABLE action_interest_receipts DISABLE TRIGGER stage1_action_interest_insert_timestamp_owned`,
		`ALTER TABLE action_interest_receipts ENABLE TRIGGER stage1_action_interest_insert_timestamp_owned`,
		query, args...,
	)
}

func mutatePostgresStage1LedgerFixture(
	t *testing.T,
	db *sql.DB,
	query string,
	args ...any,
) {
	t.Helper()
	execPostgresStage1FixtureMutation(
		t, db,
		`ALTER TABLE nhs_schema_migrations DISABLE RULE nhs_schema_migrations_no_update`,
		`ALTER TABLE nhs_schema_migrations ENABLE RULE nhs_schema_migrations_no_update`,
		query, args...,
	)
}

type postgresCommercialProvider struct {
	accountID   int64
	site        models.Site
	claim       *models.ProviderClaim
	key         *models.ProviderAPIKey
	rawKey      string
	company     *models.ProviderPilotCompany
	companyHash string
}

func exerciseProviderActionHandoffPostgres(
	t *testing.T,
	db *sql.DB,
	signer *providerexchange.Signer,
) {
	t.Helper()
	const bounty int64 = 1_000
	provider := createPostgresCommercialProvider(t, db, "handoff-boundary")
	offer := createPostgresCommercialOffer(
		t, db, provider, "handoff-boundary", "prepaid", bounty,
	)
	recordPostgresVerifiedFunding(
		t, db, offer.ID, 10*bounty, "handoff-boundary-fund", "", time.Time{},
	)
	if _, err := activatePostgresProviderOffer(
		t, db, offer.ID, "operator:handoff-boundary", "evidence:handoff-boundary",
	); err != nil {
		t.Fatalf("activate handoff-boundary offer: %v", err)
	}

	assertTicketState := func(ticketID, wantStatus string, wantReceipts int) {
		t.Helper()
		var status string
		var receiptCount int
		if err := db.QueryRow(`SELECT status FROM action_tickets WHERE id=$1::uuid`, ticketID).Scan(&status); err != nil {
			t.Fatalf("read handoff-boundary ticket %s: %v", ticketID, err)
		}
		if err := db.QueryRow(`
			SELECT COUNT(*)::int FROM provider_action_handoff_receipts
			WHERE action_ticket_id=$1::uuid`, ticketID).Scan(&receiptCount); err != nil {
			t.Fatalf("count handoff receipts for %s: %v", ticketID, err)
		}
		if status != wantStatus || receiptCount != wantReceipts {
			t.Fatalf("ticket %s state = status:%q receipts:%d, want %q/%d", ticketID, status, receiptCount, wantStatus, wantReceipts)
		}
	}
	handoffInput := func(ticketID, token string) models.ProviderActionHandoffInput {
		return models.ProviderActionHandoffInput{
			ActionTicketID:          ticketID,
			AttributionToken:        token,
			PrincipalHandoffConsent: true,
			HandoffConsentVersion:   models.ProviderActionHandoffConsentV1,
		}
	}

	// Application validation and the transaction both fail closed before a
	// durable observation or status transition when the separate handoff
	// consent or exact bearer is absent.
	ticket, rawToken := createPostgresUnhandedActionTicket(
		t, db, signer, provider.site, offer.ID, "handoff-exact",
	)
	missingConsent := handoffInput(ticket.ID, rawToken)
	missingConsent.PrincipalHandoffConsent = false
	if _, _, err := models.RecordActionTicketHandoff(db, missingConsent); !errors.Is(err, models.ErrInvalidProviderExchange) {
		t.Fatalf("missing handoff consent error = %v, want ErrInvalidProviderExchange", err)
	}
	wrongConsentVersion := handoffInput(ticket.ID, rawToken)
	wrongConsentVersion.HandoffConsentVersion = "nhs-provider-handoff-consent-v0"
	if _, _, err := models.RecordActionTicketHandoff(db, wrongConsentVersion); !errors.Is(err, models.ErrInvalidProviderExchange) {
		t.Fatalf("wrong handoff consent version error = %v, want ErrInvalidProviderExchange", err)
	}
	if _, _, err := models.RecordActionTicketHandoff(
		db, handoffInput(ticket.ID, "wrong-provider-action-bearer"),
	); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("wrong handoff bearer error = %v, want sql.ErrNoRows", err)
	}
	assertTicketState(ticket.ID, "created", 0)
	if _, _, err := models.RecordActionTicketHandoff(
		db, handoffInput(ticket.ID, rawToken),
	); !errors.Is(err, models.ErrProviderPilotReviewRequired) {
		t.Fatalf("handoff before ticket review error = %v, want ErrProviderPilotReviewRequired", err)
	}
	if _, err := db.Exec(`
		INSERT INTO provider_action_handoff_receipts (
			action_ticket_id, provider_claim_id, provider_offer_id,
			offer_version_snapshot,
			commercial_terms_contract_version_snapshot,
			commercial_terms_sha256_snapshot, presented_token_hash,
			principal_handoff_consent, handoff_consent_version,
			event_contract_version
		)
		SELECT id, provider_claim_id, provider_offer_id,
		       offer_version_snapshot,
		       commercial_terms_contract_version_snapshot,
		       commercial_terms_sha256_snapshot, token_hash,
		       true, $2, $3
		FROM action_tickets WHERE id=$1::uuid`,
		ticket.ID, models.ProviderActionHandoffConsentV1,
		models.ProviderActionHandoffContractV1,
	); err == nil {
		t.Fatal("database recorded provider handoff before current ticket review")
	} else {
		assertPostgresConstraint(t, err, "provider_handoff_ticket_review")
	}
	assertTicketState(ticket.ID, "created", 0)
	reviewQueue, err := models.GetProviderPilotQueue(db, "ticket_review_required", 100)
	if err != nil {
		t.Fatalf("read ticket review-required queue: %v", err)
	}
	var queuedReview *models.ProviderPilotQueueItem
	for index := range reviewQueue.Items {
		if reviewQueue.Items[index].SubjectID == ticket.ID {
			queuedReview = &reviewQueue.Items[index]
			break
		}
	}
	if queuedReview == nil || queuedReview.State != "ticket_review_required" ||
		queuedReview.ProviderPilotEpochID != ticket.ProviderPilotEpochID ||
		queuedReview.ReviewType != "ticket" || queuedReview.TicketID != ticket.ID ||
		len(queuedReview.SubjectSnapshotSHA256) != 64 {
		t.Fatalf("ticket review-required queue item = %#v", queuedReview)
	}
	recordPostgresTicketReview(t, db, ticket, "handoff-exact")
	reviewQueue, err = models.GetProviderPilotQueue(db, "ticket_review_required", 100)
	if err != nil {
		t.Fatalf("read ticket review-required queue after review: %v", err)
	}
	for _, item := range reviewQueue.Items {
		if item.SubjectID == ticket.ID {
			t.Fatalf("reviewed ticket remained in review-required queue: %#v", item)
		}
	}

	redirected, receipt, err := models.RecordActionTicketHandoff(
		db, handoffInput(ticket.ID, rawToken),
	)
	if err != nil {
		t.Fatalf("record exact provider handoff: %v", err)
	}
	if redirected.Status != "redirected" || receipt.Replayed ||
		receipt.ActionTicketID != ticket.ID ||
		receipt.ProviderOfferID != offer.ID ||
		receipt.PresentedTokenHash != models.HashProviderSecret(rawToken) ||
		!receipt.PrincipalHandoffConsent ||
		receipt.HandoffConsentVersion != models.ProviderActionHandoffConsentV1 ||
		receipt.PrincipalControlledIntentDisclosureConsent ||
		receipt.ControlledIntentDisclosureConsentVersion != "" ||
		receipt.EventContractVersion != models.ProviderActionHandoffContractV1 ||
		!receipt.ObservedAt.Equal(receipt.CreatedAt) {
		t.Fatalf("exact provider handoff = ticket:%#v receipt:%#v", redirected, receipt)
	}
	assertTicketState(ticket.ID, "redirected", 1)
	if _, err := models.ResolveProviderControlledIntent(
		db, provider.key, ticket.ID, offer.ID, rawToken,
	); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("declined controlled-intent disclosure error = %v, want sql.ErrNoRows", err)
	}
	upgradeDisclosure := handoffInput(ticket.ID, rawToken)
	upgradeDisclosure.PrincipalControlledIntentDisclosureConsent = true
	upgradeDisclosure.ControlledIntentDisclosureConsentVersion = models.ProviderControlledIntentDisclosureConsentV1
	if _, _, err := models.RecordActionTicketHandoff(db, upgradeDisclosure); !errors.Is(err, models.ErrInvalidProviderExchange) {
		t.Fatalf("handoff replay upgraded declined disclosure consent: %v", err)
	}
	replayedTicket, replayedReceipt, err := models.RecordActionTicketHandoff(
		db,
		handoffInput("  "+strings.ToUpper(ticket.ID)+"  ", "  "+rawToken+"  "),
	)
	if err != nil {
		t.Fatalf("replay canonicalized provider handoff: %v", err)
	}
	if replayedTicket.ID != ticket.ID || replayedTicket.Status != "redirected" ||
		!replayedReceipt.Replayed || replayedReceipt.ID != receipt.ID {
		t.Fatalf("canonicalized provider handoff replay = ticket:%#v receipt:%#v", replayedTicket, replayedReceipt)
	}
	assertTicketState(ticket.ID, "redirected", 1)
	if _, err := db.Exec(`
		UPDATE sites SET category='spam' WHERE id=$1::uuid`, provider.site.ID); err != nil {
		t.Fatalf("reclassify handed-off provider site as spam: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec(`UPDATE sites SET category='developer' WHERE id=$1::uuid`, provider.site.ID)
	})
	if _, _, err := models.RecordProviderOutcome(db, provider.key, models.ProviderOutcomeInput{
		ActionTicketID: ticket.ID,
		IdempotencyKey: "pg-handoff-spam-positive-0001",
		PayloadHash:    postgresPayloadHash("handoff-spam-positive"),
		Outcome:        "accepted",
	}, signer); !errors.Is(err, models.ErrProviderOutcomeTransition) {
		t.Fatalf("positive outcome for spam-reclassified provider error=%v, want ErrProviderOutcomeTransition", err)
	}
	// Economic cleanup stays possible: negative outcomes do not claim a
	// successful connection and must remain recordable after reclassification.
	recordPostgresOutcome(
		t, db, signer, provider.key, ticket.ID, "rejected", "handoff-terminal-rejected",
	)
	if _, err := db.Exec(`
		UPDATE sites SET category='developer' WHERE id=$1::uuid`, provider.site.ID); err != nil {
		t.Fatalf("restore handed-off provider site after spam test: %v", err)
	}
	if _, _, err := models.RecordActionTicketHandoff(
		db, handoffInput(ticket.ID, rawToken),
	); !errors.Is(err, models.ErrInvalidProviderExchange) {
		t.Fatalf("terminal ticket handoff replay error = %v, want ErrInvalidProviderExchange", err)
	}
	assertTicketState(ticket.ID, "rejected", 1)

	createControlledIntentTicketWithTTL := func(suffix string, ttl time.Duration) (*models.ActionTicket, string) {
		t.Helper()
		controlledTicket, controlledToken := createPostgresActionTicketFixture(
			t, db, signer, provider.site, offer.ID, suffix, models.ActionTicketInput{
				DemandTopic:      "developer-tools",
				RegionCode:       "US-WA",
				BudgetBand:       "500_1999",
				Urgency:          "7_days",
				RequirementFlags: []string{"api_access", "sandbox"},
				PrincipalConsent: true,
				ConsentVersion:   models.ProviderPrincipalConsentV1,
				TTL:              ttl,
			})
		return controlledTicket, controlledToken
	}
	createControlledIntentTicket := func(suffix string) (*models.ActionTicket, string) {
		t.Helper()
		return createControlledIntentTicketWithTTL(suffix, time.Hour)
	}
	consentedHandoff := func(controlledTicket *models.ActionTicket, controlledToken string) *models.ProviderActionHandoffReceipt {
		t.Helper()
		recordPostgresTicketReview(t, db, controlledTicket, "controlled-"+controlledTicket.ID[:8])
		input := handoffInput(controlledTicket.ID, controlledToken)
		input.PrincipalControlledIntentDisclosureConsent = true
		input.ControlledIntentDisclosureConsentVersion = models.ProviderControlledIntentDisclosureConsentV1
		_, controlledReceipt, handoffErr := models.RecordActionTicketHandoff(db, input)
		if handoffErr != nil {
			t.Fatalf("record consented controlled-intent handoff: %v", handoffErr)
		}
		if controlledReceipt == nil || controlledReceipt.Replayed ||
			!controlledReceipt.PrincipalControlledIntentDisclosureConsent ||
			controlledReceipt.ControlledIntentDisclosureConsentVersion != models.ProviderControlledIntentDisclosureConsentV1 {
			t.Fatalf("consented controlled-intent receipt=%#v", controlledReceipt)
		}
		return controlledReceipt
	}

	controlledTicket, controlledToken := createControlledIntentTicket("controlled-intent-main")
	if _, err := models.ResolveProviderControlledIntent(
		db, provider.key, controlledTicket.ID, offer.ID, controlledToken,
	); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("controlled intent resolved before observed handoff: %v", err)
	}
	controlledReceipt := consentedHandoff(controlledTicket, controlledToken)
	pairTicket, pairToken := createControlledIntentTicket("controlled-intent-pair-constraint")
	recordPostgresTicketReview(t, db, pairTicket, "controlled-pair-constraint")
	insertInvalidDisclosurePair := func(consent bool, version string) error {
		_, insertErr := db.Exec(`
			INSERT INTO provider_action_handoff_receipts (
				action_ticket_id, provider_claim_id, provider_offer_id,
				offer_version_snapshot,
				commercial_terms_contract_version_snapshot,
				commercial_terms_sha256_snapshot, presented_token_hash,
				principal_handoff_consent, handoff_consent_version,
				principal_controlled_intent_disclosure_consent,
				controlled_intent_disclosure_consent_version,
				event_contract_version
			)
			SELECT id, provider_claim_id, provider_offer_id,
			       offer_version_snapshot,
			       commercial_terms_contract_version_snapshot,
			       commercial_terms_sha256_snapshot, token_hash,
			       true, $2, $3, $4, $5
			FROM action_tickets WHERE id=$1::uuid`,
			pairTicket.ID, models.ProviderActionHandoffConsentV1,
			consent, version, models.ProviderActionHandoffContractV1,
		)
		return insertErr
	}
	for _, testCase := range []struct {
		consent bool
		version string
	}{
		{consent: false, version: models.ProviderControlledIntentDisclosureConsentV1},
		{consent: true, version: ""},
		{consent: true, version: "nhs-provider-controlled-intent-disclosure-consent-v0"},
	} {
		if err := insertInvalidDisclosurePair(testCase.consent, testCase.version); err == nil {
			t.Fatalf("database accepted invalid controlled-intent disclosure pair %#v", testCase)
		} else {
			assertPostgresConstraint(t, err, "provider_handoff_intent_disclosure_consent_pair")
		}
	}
	recordPostgresOutcome(
		t, db, signer, provider.key, pairTicket.ID,
		"rejected", "controlled-intent-pair-release",
	)
	_ = pairToken
	readResolverSideEffects := func(ticketID string) [4]int {
		t.Helper()
		var counts [4]int
		if err := db.QueryRow(`
			SELECT
			  (SELECT COUNT(*)::int FROM provider_action_handoff_receipts WHERE action_ticket_id=$1::uuid),
			  (SELECT COUNT(*)::int FROM outcome_receipts WHERE action_ticket_id=$1::uuid),
			  (SELECT COUNT(*)::int FROM provider_budget_ledger WHERE action_ticket_id=$1::uuid),
			  (SELECT COUNT(*)::int FROM provider_capacity_events WHERE action_ticket_id=$1::uuid)`,
			ticketID,
		).Scan(&counts[0], &counts[1], &counts[2], &counts[3]); err != nil {
			t.Fatalf("read controlled-intent resolver side effects: %v", err)
		}
		return counts
	}
	beforeResolve := readResolverSideEffects(controlledTicket.ID)
	resolution, err := models.ResolveProviderControlledIntent(
		db, provider.key, controlledTicket.ID, offer.ID, controlledToken,
	)
	if err != nil {
		t.Fatalf("resolve exact consented controlled intent: %v", err)
	}
	if resolution.ResolverContractVersion != models.ProviderControlledIntentResolverV1 ||
		resolution.TicketID != controlledTicket.ID || resolution.OfferID != offer.ID ||
		resolution.OfferVersion != controlledTicket.OfferVersionSnapshot ||
		resolution.ActionType != controlledTicket.ActionTypeSnapshot ||
		resolution.ControlledIntent.DemandTopic != "developer-tools" ||
		resolution.ControlledIntent.RegionCode != "US-WA" ||
		resolution.ControlledIntent.BudgetBand != "500_1999" ||
		resolution.ControlledIntent.Urgency != "7_days" ||
		!reflect.DeepEqual(resolution.ControlledIntent.RequirementFlags, []string{"api_access", "sandbox"}) ||
		resolution.ConsentVersion != models.ProviderControlledIntentDisclosureConsentV1 ||
		!resolution.ObservedAt.Equal(controlledReceipt.ObservedAt) ||
		resolution.IntentAvailableUntil.After(controlledTicket.ExpiresAt) {
		t.Fatalf("controlled-intent resolution=%#v", resolution)
	}
	if afterResolve := readResolverSideEffects(controlledTicket.ID); afterResolve != beforeResolve {
		t.Fatalf("read-only controlled-intent resolver changed rows: before=%v after=%v", beforeResolve, afterResolve)
	}

	// Lock acquisition must precede every clock-based disclosure check. A
	// resolver that qualifies the ticket before waiting on this row lock could
	// otherwise return controlled intent after the signed authorization expires.
	expiryRaceTicket, expiryRaceToken := createControlledIntentTicketWithTTL(
		"controlled-intent-lock-expiry", 3*time.Second,
	)
	consentedHandoff(expiryRaceTicket, expiryRaceToken)
	locker, err := db.Begin()
	if err != nil {
		t.Fatalf("begin controlled-intent expiry lock: %v", err)
	}
	defer func() { _ = locker.Rollback() }()
	var lockedTicketID string
	if err := locker.QueryRow(`
		SELECT id::text FROM action_tickets
		WHERE id=$1::uuid FOR UPDATE`, expiryRaceTicket.ID).Scan(&lockedTicketID); err != nil {
		t.Fatalf("lock controlled-intent expiry ticket: %v", err)
	}
	if lockedTicketID != expiryRaceTicket.ID {
		t.Fatalf("locked controlled-intent ticket=%q, want %q", lockedTicketID, expiryRaceTicket.ID)
	}
	type controlledIntentResolveResult struct {
		resolution *models.ProviderControlledIntentResolution
		err        error
	}
	expiryRaceResult := make(chan controlledIntentResolveResult, 1)
	go func() {
		resolved, resolveErr := models.ResolveProviderControlledIntent(
			db, provider.key, expiryRaceTicket.ID, offer.ID, expiryRaceToken,
		)
		expiryRaceResult <- controlledIntentResolveResult{resolution: resolved, err: resolveErr}
	}()
	blockedDeadline := time.Now().Add(2 * time.Second)
	for {
		var waitingOnLock bool
		if err := db.QueryRow(`
			SELECT EXISTS (
				SELECT 1 FROM pg_stat_activity
				WHERE datname=current_database()
				  AND pid<>pg_backend_pid()
				  AND state='active'
				  AND wait_event_type='Lock'
				  AND query LIKE '%WITH locked AS MATERIALIZED%'
			)`).Scan(&waitingOnLock); err != nil {
			t.Fatalf("observe controlled-intent resolver lock wait: %v", err)
		}
		if waitingOnLock {
			break
		}
		select {
		case early := <-expiryRaceResult:
			t.Fatalf("controlled-intent resolver escaped held ticket lock: resolution=%#v err=%v", early.resolution, early.err)
		default:
		}
		if time.Now().After(blockedDeadline) {
			t.Fatal("controlled-intent resolver did not reach the held ticket lock before expiry test deadline")
		}
		time.Sleep(10 * time.Millisecond)
	}
	for {
		var expired bool
		if err := db.QueryRow(`
			SELECT expires_at<=clock_timestamp()
			FROM action_tickets WHERE id=$1::uuid`, expiryRaceTicket.ID).Scan(&expired); err != nil {
			t.Fatalf("observe controlled-intent expiry while resolver waits: %v", err)
		}
		if expired {
			break
		}
		select {
		case early := <-expiryRaceResult:
			t.Fatalf("controlled-intent resolver returned before held lock release: resolution=%#v err=%v", early.resolution, early.err)
		default:
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err := locker.Commit(); err != nil {
		t.Fatalf("release controlled-intent expiry lock: %v", err)
	}
	select {
	case afterExpiry := <-expiryRaceResult:
		if afterExpiry.resolution != nil || !errors.Is(afterExpiry.err, sql.ErrNoRows) {
			t.Fatalf("post-lock-expiry controlled-intent result=%#v err=%v, want nil/sql.ErrNoRows", afterExpiry.resolution, afterExpiry.err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("controlled-intent resolver did not finish after expiry lock release")
	}

	wrongIssuedAt := time.Now().UTC().Truncate(time.Second)
	wrongClaims, err := providerexchange.NewAttributionClaimsForKey(
		signer.ActiveKeyID(), controlledTicket.ID, offer.ID,
		wrongIssuedAt, wrongIssuedAt.Add(time.Hour),
	)
	if err != nil {
		t.Fatalf("create wrong controlled-intent bearer claims: %v", err)
	}
	wrongToken, err := signer.SignAttribution(wrongClaims)
	if err != nil {
		t.Fatalf("sign wrong controlled-intent bearer: %v", err)
	}
	if _, err := models.ResolveProviderControlledIntent(
		db, provider.key, controlledTicket.ID, offer.ID, wrongToken,
	); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("wrong controlled-intent bearer error=%v, want sql.ErrNoRows", err)
	}
	crossClaim := createPostgresProviderIdentity(t, db, "controlled-intent-cross-claim")
	if _, err := models.ResolveProviderControlledIntent(
		db, crossClaim.key, controlledTicket.ID, offer.ID, controlledToken,
	); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("cross-claim controlled-intent resolution error=%v, want sql.ErrNoRows", err)
	}

	for name, query := range map[string]string{
		"offer version": `UPDATE action_tickets SET offer_version_snapshot=offer_version_snapshot+1 WHERE id=$1::uuid`,
		"terms version": `UPDATE action_tickets SET commercial_terms_contract_version_snapshot='changed' WHERE id=$1::uuid`,
		"terms hash":    `UPDATE action_tickets SET commercial_terms_sha256_snapshot=repeat('f',64) WHERE id=$1::uuid`,
		"action type":   `UPDATE action_tickets SET action_type_snapshot='trial' WHERE id=$1::uuid`,
		"created time":  `UPDATE action_tickets SET created_at=created_at+INTERVAL '1 minute' WHERE id=$1::uuid`,
		"expiry time":   `UPDATE action_tickets SET expires_at=expires_at+INTERVAL '1 minute' WHERE id=$1::uuid`,
		"topic":         `UPDATE action_tickets SET demand_topic='commerce' WHERE id=$1::uuid`,
		"region":        `UPDATE action_tickets SET region_code='CA' WHERE id=$1::uuid`,
		"budget":        `UPDATE action_tickets SET budget_band='under_100' WHERE id=$1::uuid`,
		"urgency":       `UPDATE action_tickets SET urgency='now' WHERE id=$1::uuid`,
		"requirements":  `UPDATE action_tickets SET requirement_flags=ARRAY['mcp']::text[] WHERE id=$1::uuid`,
	} {
		if _, err := db.Exec(query, controlledTicket.ID); err == nil {
			t.Fatalf("database rewrote consent-bound controlled-intent %s", name)
		} else {
			assertPostgresConstraint(t, err, "action_ticket_controlled_intent_immutable")
		}
	}
	resolutionAfterRejectedMutation, err := models.ResolveProviderControlledIntent(
		db, provider.key, controlledTicket.ID, offer.ID, controlledToken,
	)
	if err != nil || !reflect.DeepEqual(resolution, resolutionAfterRejectedMutation) {
		t.Fatalf("controlled intent drifted after rejected direct mutation: before=%#v after=%#v err=%v", resolution, resolutionAfterRejectedMutation, err)
	}

	redactedTicket, redactedToken := createControlledIntentTicket("controlled-intent-redacted")
	consentedHandoff(redactedTicket, redactedToken)
	if _, err := db.Exec(`
		UPDATE action_tickets
		SET search_receipt_id=NULL, demand_topic='redacted', region_code='',
		    budget_band='unspecified', urgency='unspecified',
		    requirement_flags='{}', intent_redacted_at=clock_timestamp(),
		    updated_at=clock_timestamp()
		WHERE id=$1::uuid`, redactedTicket.ID); err != nil {
		t.Fatalf("apply exact one-way controlled-intent redaction: %v", err)
	}
	if _, err := models.ResolveProviderControlledIntent(
		db, provider.key, redactedTicket.ID, offer.ID, redactedToken,
	); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("redacted controlled intent resolution error=%v, want sql.ErrNoRows", err)
	}
	if _, err := db.Exec(`
		UPDATE action_tickets
		SET demand_topic='developer-tools', intent_redacted_at=NULL
		WHERE id=$1::uuid`, redactedTicket.ID); err == nil {
		t.Fatal("database restored a redacted controlled-intent bundle")
	} else {
		assertPostgresConstraint(t, err, "action_ticket_controlled_intent_immutable")
	}

	revokedIntentTicket, revokedIntentToken := createControlledIntentTicket("controlled-intent-revoked")
	consentedHandoff(revokedIntentTicket, revokedIntentToken)
	if _, err := db.Exec(`
		UPDATE action_tickets
		SET authorization_revoked_at=clock_timestamp(), updated_at=clock_timestamp()
		WHERE id=$1::uuid`, revokedIntentTicket.ID); err != nil {
		t.Fatalf("revoke controlled-intent authorization: %v", err)
	}
	if _, err := models.ResolveProviderControlledIntent(
		db, provider.key, revokedIntentTicket.ID, offer.ID, revokedIntentToken,
	); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("revoked controlled intent resolution error=%v, want sql.ErrNoRows", err)
	}
	if _, err := db.Exec(`
		UPDATE action_tickets SET authorization_revoked_at=NULL
		WHERE id=$1::uuid`, revokedIntentTicket.ID); err == nil {
		t.Fatal("database cleared a controlled-intent authorization revocation")
	} else {
		assertPostgresConstraint(t, err, "action_ticket_controlled_intent_immutable")
	}

	negativeIntentTicket, negativeIntentToken := createControlledIntentTicket("controlled-intent-negative")
	consentedHandoff(negativeIntentTicket, negativeIntentToken)
	recordPostgresOutcome(
		t, db, signer, provider.key, negativeIntentTicket.ID,
		"rejected", "controlled-intent-negative-outcome",
	)
	for _, reopenedStatus := range []string{"redirected", "accepted", "activated", "converted"} {
		if _, err := db.Exec(`
			UPDATE action_tickets SET status=$1, updated_at=clock_timestamp()
			WHERE id=$2::uuid`, reopenedStatus, negativeIntentTicket.ID); err == nil {
			t.Fatalf("database reopened negative controlled-intent ticket as %s", reopenedStatus)
		} else {
			assertPostgresConstraint(t, err, "action_ticket_status_transition")
		}
	}
	if _, err := models.ResolveProviderControlledIntent(
		db, provider.key, negativeIntentTicket.ID, offer.ID, negativeIntentToken,
	); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("negative terminal controlled intent resolution error=%v, want sql.ErrNoRows", err)
	}

	// Expiry and emergency authorization revocation are evaluated under the
	// authoritative database clock before receipt insertion.
	expiredTicket, expiredToken := createPostgresUnhandedActionTicketWithTTL(
		t, db, signer, provider.site, offer.ID, "handoff-expired", 2*time.Second,
	)
	for {
		var expired bool
		if err := db.QueryRow(`
			SELECT expires_at<=clock_timestamp()
			FROM action_tickets WHERE id=$1::uuid`, expiredTicket.ID).Scan(&expired); err != nil {
			t.Fatalf("observe handoff-boundary ticket expiry: %v", err)
		}
		if expired {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if _, _, err := models.RecordActionTicketHandoff(
		db, handoffInput(expiredTicket.ID, expiredToken),
	); !errors.Is(err, models.ErrActionTicketExpired) {
		t.Fatalf("expired handoff error = %v, want ErrActionTicketExpired", err)
	}
	assertTicketState(expiredTicket.ID, "created", 0)

	revokedTicket, revokedToken := createPostgresUnhandedActionTicket(
		t, db, signer, provider.site, offer.ID, "handoff-revoked",
	)
	if _, err := db.Exec(`
		UPDATE action_tickets
		SET authorization_revoked_at=clock_timestamp(), updated_at=clock_timestamp()
		WHERE id=$1::uuid`, revokedTicket.ID); err != nil {
		t.Fatalf("revoke handoff-boundary ticket: %v", err)
	}
	if _, _, err := models.RecordActionTicketHandoff(
		db, handoffInput(revokedTicket.ID, revokedToken),
	); !errors.Is(err, models.ErrProviderOfferRevoked) {
		t.Fatalf("revoked handoff error = %v, want ErrProviderOfferRevoked", err)
	}
	assertTicketState(revokedTicket.ID, "created", 0)

	// Database triggers independently reject bypass inserts with a wrong token
	// hash or missing handoff consent. They also reject every direct positive
	// ticket transition without the immutable observed-handoff receipt.
	directTicket, directToken := createPostgresUnhandedActionTicket(
		t, db, signer, provider.site, offer.ID, "handoff-direct-sql",
	)
	insertDirectReceipt := func(tokenHash string, consent bool, consentVersion string) error {
		_, insertErr := db.Exec(`
			INSERT INTO provider_action_handoff_receipts (
				action_ticket_id, provider_claim_id, provider_offer_id,
				offer_version_snapshot,
				commercial_terms_contract_version_snapshot,
				commercial_terms_sha256_snapshot, presented_token_hash,
				principal_handoff_consent, handoff_consent_version,
				event_contract_version
			)
			SELECT id, provider_claim_id, provider_offer_id,
			       offer_version_snapshot,
			       commercial_terms_contract_version_snapshot,
			       commercial_terms_sha256_snapshot, $2, $3, $4, $5
			FROM action_tickets WHERE id=$1::uuid`,
			directTicket.ID, tokenHash, consent, consentVersion,
			models.ProviderActionHandoffContractV1,
		)
		return insertErr
	}
	if err := insertDirectReceipt(
		postgresPayloadHash("wrong-direct-handoff-token"), true,
		models.ProviderActionHandoffConsentV1,
	); err == nil {
		t.Fatal("database accepted direct handoff receipt with wrong token hash")
	} else {
		assertPostgresConstraint(t, err, "provider_action_handoff_exact_ticket")
	}
	if err := insertDirectReceipt(
		models.HashProviderSecret(directToken), false,
		models.ProviderActionHandoffConsentV1,
	); err == nil {
		t.Fatal("database accepted direct handoff receipt without consent")
	} else {
		assertPostgresConstraint(t, err, "provider_action_handoff_exact_ticket")
	}
	if _, err := db.Exec(`
		INSERT INTO action_tickets
		SELECT (jsonb_populate_record(
			NULL::action_tickets,
			to_jsonb(ticket) || jsonb_build_object(
				'id', uuid_generate_v4(),
				'search_receipt_id', NULL,
				'token_hash', $2::text,
				'creation_request_hash', $3::text,
				'status', 'redirected'
			)
		)).*
		FROM action_tickets ticket WHERE id=$1::uuid`,
		directTicket.ID, postgresPayloadHash("direct-positive-insert-token"),
		postgresPayloadHash("direct-positive-insert-request"),
	); err == nil {
		t.Fatal("database accepted a ticket inserted directly in a positive state")
	} else {
		// Migration 024's exact pilot/returned-receipt gate now rejects this
		// forged clone before the older positive-state trigger is reached.
		assertPostgresConstraint(t, err, "action_ticket_pilot_returned_offer")
	}
	for _, positiveStatus := range []string{"redirected", "accepted", "activated", "converted"} {
		if _, err := db.Exec(`UPDATE action_tickets SET status=$1 WHERE id=$2::uuid`, positiveStatus, directTicket.ID); err == nil {
			t.Fatalf("database accepted direct %s ticket transition without handoff", positiveStatus)
		} else {
			constraint := "action_ticket_status_transition"
			if positiveStatus == "redirected" {
				constraint = "action_ticket_observed_handoff_required"
			}
			assertPostgresConstraint(t, err, constraint)
		}
		assertTicketState(directTicket.ID, "created", 0)
	}
	if _, err := db.Exec(`UPDATE action_tickets SET status='rejected' WHERE id=$1::uuid`, directTicket.ID); err != nil {
		t.Fatalf("prepare negative wrong-status handoff fixture: %v", err)
	}
	if err := insertDirectReceipt(
		models.HashProviderSecret(directToken), true,
		models.ProviderActionHandoffConsentV1,
	); err == nil {
		t.Fatal("database accepted direct handoff receipt for non-created ticket")
	} else {
		assertPostgresConstraint(t, err, "provider_action_handoff_exact_ticket")
	}
	assertTicketState(directTicket.ID, "rejected", 0)

	// Handoff and the first positive provider callback share the same
	// claim -> offer -> ticket lock order. Either the handoff wins and the
	// callback charges, or the callback fails without a write and succeeds on an
	// exact retry after the handoff. Both paths end with one receipt and charge.
	raceTicket, raceToken := createPostgresUnhandedActionTicket(
		t, db, signer, provider.site, offer.ID, "handoff-outcome-race",
	)
	recordPostgresTicketReview(t, db, raceTicket, "handoff-outcome-race")
	type handoffRaceResult struct {
		operation string
		created   bool
		err       error
	}
	start := make(chan struct{})
	results := make(chan handoffRaceResult, 2)
	go func() {
		<-start
		_, _, handoffErr := models.RecordActionTicketHandoff(
			db, handoffInput(raceTicket.ID, raceToken),
		)
		results <- handoffRaceResult{operation: "handoff", err: handoffErr}
	}()
	go func() {
		<-start
		_, created, outcomeErr := models.RecordProviderOutcome(
			db, provider.key, models.ProviderOutcomeInput{
				ActionTicketID: raceTicket.ID,
				IdempotencyKey: "pg-handoff-outcome-race-0001",
				PayloadHash:    postgresPayloadHash("handoff-outcome-race"),
				Outcome:        "accepted",
			}, signer,
		)
		results <- handoffRaceResult{operation: "outcome", created: created, err: outcomeErr}
	}()
	close(start)
	timer := time.NewTimer(15 * time.Second)
	defer timer.Stop()
	var outcomeCreated bool
	for range 2 {
		select {
		case result := <-results:
			switch result.operation {
			case "handoff":
				if result.err != nil {
					t.Fatalf("concurrent provider handoff: %v", result.err)
				}
			case "outcome":
				if result.err != nil && !errors.Is(result.err, models.ErrProviderOutcomeTransition) {
					t.Fatalf("concurrent positive outcome error = %v", result.err)
				}
				outcomeCreated = result.err == nil && result.created
			}
		case <-timer.C:
			t.Fatal("concurrent handoff/outcome did not complete within 15s")
		}
	}
	if !outcomeCreated {
		receipt, created, err := models.RecordProviderOutcome(
			db, provider.key, models.ProviderOutcomeInput{
				ActionTicketID: raceTicket.ID,
				IdempotencyKey: "pg-handoff-outcome-race-0001",
				PayloadHash:    postgresPayloadHash("handoff-outcome-race"),
				Outcome:        "accepted",
			}, signer,
		)
		if err != nil || !created || receipt.ChargeStatus != string(providerexchange.ChargeStatusCharged) {
			t.Fatalf("positive outcome retry after handoff = receipt:%#v created:%t err:%v", receipt, created, err)
		}
	}
	var receiptCount, chargedOutcomeCount, chargeLedgerCount int
	if err := db.QueryRow(`
		SELECT
		  (SELECT COUNT(*)::int FROM provider_action_handoff_receipts WHERE action_ticket_id=$1::uuid),
		  (SELECT COUNT(*)::int FROM outcome_receipts WHERE action_ticket_id=$1::uuid AND charge_status='charged'),
		  (SELECT COUNT(*)::int FROM provider_budget_ledger WHERE action_ticket_id=$1::uuid AND entry_type='charge')`,
		raceTicket.ID,
	).Scan(&receiptCount, &chargedOutcomeCount, &chargeLedgerCount); err != nil {
		t.Fatalf("read handoff/outcome race evidence: %v", err)
	}
	if receiptCount != 1 || chargedOutcomeCount != 1 || chargeLedgerCount != 1 {
		t.Fatalf("handoff/outcome race evidence = handoffs:%d charged outcomes:%d charge entries:%d, want 1/1/1", receiptCount, chargedOutcomeCount, chargeLedgerCount)
	}
	assertTicketState(raceTicket.ID, "accepted", 1)

	_, rotatedKey, err := models.RotateProviderAPIKey(
		db, provider.accountID, provider.claim.ID,
	)
	if err != nil {
		t.Fatalf("rotate controlled-intent provider key: %v", err)
	}
	if _, err := models.ResolveProviderControlledIntent(
		db, provider.key, controlledTicket.ID, offer.ID, controlledToken,
	); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("revoked provider key controlled-intent error=%v, want sql.ErrNoRows", err)
	}
	if rotatedResolution, err := models.ResolveProviderControlledIntent(
		db, rotatedKey, controlledTicket.ID, offer.ID, controlledToken,
	); err != nil || !reflect.DeepEqual(resolution, rotatedResolution) {
		t.Fatalf("rotated active key controlled-intent resolution=%#v err=%v", rotatedResolution, err)
	}
}

type postgresVerifiedProofSnapshot struct {
	companies        int
	chargedProviders map[string]int
	offerReturns     map[string]int
	acceptedHandoffs int
	activations      int
	renewals         int
	conversions      int
	settled          map[string]int64
	integrityValid   bool
	rejectedOutcomes int
	rejectedLedger   int
	pilotMet         bool
}

func postgresVerifiedProof(proof *models.ProviderExchangeProof) postgresVerifiedProofSnapshot {
	if proof == nil {
		return postgresVerifiedProofSnapshot{}
	}
	chargedProviders := make(map[string]int, len(proof.VerifiedMechanisms))
	offerReturns := make(map[string]int, len(proof.VerifiedMechanisms))
	for chargeEvent, evidence := range proof.VerifiedMechanisms {
		chargedProviders[chargeEvent] = evidence.ChargedProviderCompanies
		offerReturns[chargeEvent] = evidence.OfferReturns
	}
	return postgresVerifiedProofSnapshot{
		companies:        proof.VerifiedProviderCompanies,
		chargedProviders: chargedProviders,
		offerReturns:     offerReturns,
		acceptedHandoffs: proof.VerifiedProviderAcceptedHandoffs,
		activations:      proof.VerifiedProviderConfirmedActivations,
		renewals:         proof.VerifiedProviderRenewals,
		conversions:      proof.VerifiedProviderConfirmedConversions,
		settled:          proof.VerifiedPrepaidSettledByCurrency,
		integrityValid:   proof.OutcomeReceiptIntegrityValid,
		rejectedOutcomes: proof.RejectedOutcomeReceipts,
		rejectedLedger:   proof.RejectedOutcomeLedgerEntries,
		pilotMet:         proof.PilotThresholdsMet,
	}
}

func readPostgresVerifiedProof(
	t *testing.T,
	db *sql.DB,
	signer *providerexchange.Signer,
) postgresVerifiedProofSnapshot {
	t.Helper()
	proof, err := models.GetProviderExchangeProof(db, postgresProviderPilotEpochID, signer)
	if err != nil {
		t.Fatalf("read verified provider commercial proof: %v", err)
	}
	return postgresVerifiedProof(proof)
}

func exerciseProviderPilotReviewEvidencePostgres(
	t *testing.T,
	db *sql.DB,
	signer *providerexchange.Signer,
) {
	t.Helper()
	const bounty int64 = 1_000
	provider := createPostgresCommercialProvider(t, db, "review-evidence")
	offer := createPostgresCommercialOffer(
		t, db, provider, "review-evidence", "terms", bounty,
	)
	recordPostgresVerifiedTerms(
		t, db, provider.key, offer.ID, "review-evidence-terms", "", "",
	)
	if _, err := activatePostgresProviderOffer(
		t, db, offer.ID, "operator:review-evidence", "evidence:review-evidence",
	); err != nil {
		t.Fatalf("activate pilot-review offer: %v", err)
	}
	ticket, rawToken := createPostgresActionTicketFixture(
		t, db, signer, provider.site, offer.ID, "review-evidence-ticket",
		models.ActionTicketInput{
			DemandTopic:      "developer-tools",
			RegionCode:       "US-WA",
			BudgetBand:       "500_1999",
			Urgency:          "7_days",
			RequirementFlags: []string{"api_access", "sandbox"},
			PrincipalConsent: true,
			ConsentVersion:   models.ProviderPrincipalConsentV1,
			TTL:              time.Hour,
		},
	)
	recordPostgresTicketReview(t, db, ticket, "review-evidence-ticket")
	_, handoff, err := models.RecordActionTicketHandoff(db, models.ProviderActionHandoffInput{
		ActionTicketID:                             ticket.ID,
		AttributionToken:                           rawToken,
		PrincipalHandoffConsent:                    true,
		HandoffConsentVersion:                      models.ProviderActionHandoffConsentV1,
		PrincipalControlledIntentDisclosureConsent: true,
		ControlledIntentDisclosureConsentVersion:   models.ProviderControlledIntentDisclosureConsentV1,
	})
	if err != nil {
		t.Fatalf("record pilot-review handoff: %v", err)
	}
	callback := recordPostgresOutcome(
		t, db, signer, provider.key, ticket.ID, "accepted", "review-evidence-callback",
	)

	subjects := []struct {
		reviewType string
		subjectID  string
		assert     func(*models.ProviderPilotReviewCandidate) bool
	}{
		{
			reviewType: "provider",
			subjectID:  provider.claim.ID,
			assert: func(candidate *models.ProviderPilotReviewCandidate) bool {
				return candidate.ProviderClaimID == provider.claim.ID &&
					candidate.ProviderPilotCompanyID == provider.company.ID &&
					candidate.ProviderAcceptanceReference != "" &&
					candidate.Domain == provider.site.Domain
			},
		},
		{
			reviewType: "offer",
			subjectID:  offer.ID,
			assert: func(candidate *models.ProviderPilotReviewCandidate) bool {
				return candidate.ProviderClaimID == provider.claim.ID &&
					candidate.ProviderOfferID == offer.ID &&
					candidate.BillingMode == "terms" &&
					candidate.DisclosureLabel != "" &&
					candidate.PrincipalCurrency == "usd" &&
					candidate.CommitmentEventID != "" &&
					candidate.CommitmentProviderAcceptedAt != nil
			},
		},
		{
			reviewType: "ticket",
			subjectID:  ticket.ID,
			assert: func(candidate *models.ProviderPilotReviewCandidate) bool {
				return candidate.ProviderClaimID == provider.claim.ID &&
					candidate.ProviderOfferID == offer.ID &&
					candidate.ActionTicketID == ticket.ID &&
					candidate.PrincipalConsent &&
					candidate.DisclosureLabel != "" &&
					candidate.PrincipalCurrency == "usd" &&
					reflect.DeepEqual(candidate.RequirementFlags, []string{"api_access", "sandbox"})
			},
		},
		{
			reviewType: "handoff",
			subjectID:  handoff.ID,
			assert: func(candidate *models.ProviderPilotReviewCandidate) bool {
				return candidate.ActionTicketID == ticket.ID &&
					candidate.HandoffReceiptID == handoff.ID &&
					candidate.PrincipalHandoffConsent &&
					candidate.ControlledIntentDisclosureConsent
			},
		},
		{
			reviewType: "callback",
			subjectID:  callback.ID,
			assert: func(candidate *models.ProviderPilotReviewCandidate) bool {
				return candidate.ActionTicketID == ticket.ID &&
					candidate.HandoffReceiptID == handoff.ID &&
					candidate.OutcomeReceiptID == callback.ID &&
					candidate.OutcomeNHSEventID != "" &&
					candidate.ProviderAPIKeyID != nil &&
					candidate.OfferVersion != nil &&
					candidate.ChargeEvent == "accepted" &&
					candidate.CommercialTermsSHA256 != "" &&
					providerHashPatternForPostgresTest(candidate.OutcomeSignedReceiptSHA256) &&
					providerHashPatternForPostgresTest(candidate.OutcomeSignatureSHA256) &&
					candidate.Outcome == "accepted"
			},
		},
	}

	for _, subject := range subjects {
		subject := subject
		t.Run("pilot-review-"+subject.reviewType, func(t *testing.T) {
			candidate, err := models.GetProviderPilotReviewCandidate(
				db, postgresProviderPilotEpochID, subject.reviewType, subject.subjectID,
			)
			if err != nil {
				t.Fatalf("get %s review candidate: %v", subject.reviewType, err)
			}
			if candidate.ReviewContractVersion != models.ProviderPilotReviewContractV1 ||
				candidate.ProviderPilotEpochID != postgresProviderPilotEpochID ||
				candidate.ProviderPilotContractVersion != models.ProviderPilotEpochContractV1 ||
				candidate.PilotDemandTopic != "developer-tools" ||
				candidate.ReviewType != subject.reviewType ||
				candidate.SubjectID != subject.subjectID ||
				!providerHashPatternForPostgresTest(candidate.SubjectSnapshotSHA256) ||
				!subject.assert(candidate) {
				t.Fatalf("%s review candidate=%#v", subject.reviewType, candidate)
			}
			encoded, err := json.Marshal(candidate)
			if err != nil {
				t.Fatalf("marshal %s review candidate: %v", subject.reviewType, err)
			}
			encodedText := strings.ToLower(string(encoded))
			for _, forbidden := range []string{
				`"search_receipt`, `"query`, `"attribution_token`, `"token_hash`,
				`"company_key_hash`, `"identity`, `"contact`, `"network`,
				`"signed_receipt`, `"signature`, `"free_form_intent`,
			} {
				if strings.Contains(encodedText, forbidden) {
					t.Fatalf("%s review candidate exposed forbidden field %q: %s", subject.reviewType, forbidden, encodedText)
				}
			}
			if strings.Contains(string(encoded), rawToken) {
				t.Fatalf("%s review candidate exposed the action bearer", subject.reviewType)
			}

			hadExistingReview := candidate.ExistingReviewID != ""
			input := models.ProviderPilotReviewInput{
				ProviderPilotEpochID:   postgresProviderPilotEpochID,
				ReviewType:             subject.reviewType,
				SubjectID:              subject.subjectID,
				ExpectedSnapshotSHA256: candidate.SubjectSnapshotSHA256,
				OwnerReference:         "owner:review:" + subject.reviewType,
				EvidenceReference:      "evidence:review:" + subject.reviewType,
			}
			if hadExistingReview {
				if err := db.QueryRow(`
					SELECT owner_reference, evidence_reference
					FROM provider_pilot_review_events WHERE id=$1::uuid`,
					candidate.ExistingReviewID,
				).Scan(&input.OwnerReference, &input.EvidenceReference); err != nil {
					t.Fatalf("read existing %s review references: %v", subject.reviewType, err)
				}
			}
			event, created, err := models.RecordProviderPilotReview(db, input)
			if err != nil {
				t.Fatalf("record %s pilot review: %v", subject.reviewType, err)
			}
			if created == hadExistingReview || event.Replayed != hadExistingReview ||
				(hadExistingReview && event.ID != candidate.ExistingReviewID) ||
				event.ReviewType != subject.reviewType ||
				event.SubjectID != subject.subjectID ||
				event.SubjectSnapshotSHA256 != candidate.SubjectSnapshotSHA256 ||
				event.ReviewContractVersion != models.ProviderPilotReviewContractV1 ||
				event.ReviewedAt.IsZero() {
				t.Fatalf("record %s review event=%#v created=%t existing=%t", subject.reviewType, event, created, hadExistingReview)
			}

			replayInput := input
			replayInput.ProviderPilotEpochID = "  " + strings.ToUpper(input.ProviderPilotEpochID) + "  "
			replayInput.ReviewType = strings.ToUpper(input.ReviewType)
			replayInput.SubjectID = "  " + strings.ToUpper(input.SubjectID) + "  "
			replayInput.ExpectedSnapshotSHA256 = strings.ToUpper(input.ExpectedSnapshotSHA256)
			replayed, replayCreated, err := models.RecordProviderPilotReview(db, replayInput)
			if err != nil || replayCreated || replayed == nil || !replayed.Replayed || replayed.ID != event.ID {
				t.Fatalf("replay %s pilot review: event=%#v created=%t err=%v", subject.reviewType, replayed, replayCreated, err)
			}

			wrongHash := input
			wrongHash.ExpectedSnapshotSHA256 = strings.Repeat("0", 64)
			if wrongHash.ExpectedSnapshotSHA256 == candidate.SubjectSnapshotSHA256 {
				wrongHash.ExpectedSnapshotSHA256 = strings.Repeat("f", 64)
			}
			if _, _, err := models.RecordProviderPilotReview(db, wrongHash); !errors.Is(err, models.ErrProviderPilotReviewSnapshotChanged) {
				t.Fatalf("stale %s review hash error=%v, want ErrProviderPilotReviewSnapshotChanged", subject.reviewType, err)
			}
			conflict := input
			conflict.EvidenceReference += ":changed"
			if _, _, err := models.RecordProviderPilotReview(db, conflict); !errors.Is(err, models.ErrProviderIdempotency) {
				t.Fatalf("conflicting %s review error=%v, want ErrProviderIdempotency", subject.reviewType, err)
			}
			refetched, err := models.GetProviderPilotReviewCandidate(
				db, postgresProviderPilotEpochID, subject.reviewType, subject.subjectID,
			)
			if err != nil || refetched.ExistingReviewID != event.ID ||
				refetched.ExistingReviewedAt == nil ||
				!refetched.ExistingReviewedAt.Equal(event.ReviewedAt) {
				t.Fatalf("refetch %s reviewed candidate=%#v err=%v", subject.reviewType, refetched, err)
			}
		})
	}

	// The database derives every relational binding from the generic subject;
	// even a direct writer cannot attach a valid reviewed digest to another
	// provider or offer.
	directTicket, _ := createPostgresUnhandedActionTicket(
		t, db, signer, provider.site, offer.ID, "review-evidence-direct-binding",
	)
	directCandidate, err := models.GetProviderPilotReviewCandidate(
		db, postgresProviderPilotEpochID, "ticket", directTicket.ID,
	)
	if err != nil {
		t.Fatalf("get direct-binding review candidate: %v", err)
	}
	var fabricatedClaimID, directReviewID string
	if err := db.QueryRow(`SELECT uuid_generate_v4()::text`).Scan(&fabricatedClaimID); err != nil {
		t.Fatalf("allocate fabricated review binding: %v", err)
	}
	if err := db.QueryRow(`
		INSERT INTO provider_pilot_review_events (
			provider_pilot_epoch_id, review_contract_version, review_type,
			subject_id, provider_claim_id, subject_snapshot_sha256,
			owner_reference, evidence_reference
		) VALUES ($1::uuid,$2,'ticket',$3::uuid,$4::uuid,$5,$6,$7)
		RETURNING id::text`,
		postgresProviderPilotEpochID, models.ProviderPilotReviewContractV1,
		directTicket.ID, fabricatedClaimID, directCandidate.SubjectSnapshotSHA256,
		"owner:review:direct-binding", "evidence:review:direct-binding",
	).Scan(&directReviewID); err != nil {
		t.Fatalf("insert database-derived direct review: %v", err)
	}
	var derivedClaimID, derivedOfferID, derivedTicketID string
	if err := db.QueryRow(`
		SELECT provider_claim_id::text, provider_offer_id::text, action_ticket_id::text
		FROM provider_pilot_review_events WHERE id=$1::uuid`, directReviewID).Scan(
		&derivedClaimID, &derivedOfferID, &derivedTicketID,
	); err != nil {
		t.Fatalf("read database-derived direct review: %v", err)
	}
	if derivedClaimID != provider.claim.ID || derivedOfferID != offer.ID || derivedTicketID != directTicket.ID {
		t.Fatalf("database-derived review binding=%q/%q/%q, want %q/%q/%q",
			derivedClaimID, derivedOfferID, derivedTicketID,
			provider.claim.ID, offer.ID, directTicket.ID,
		)
	}

	wrongHashTicket, _ := createPostgresUnhandedActionTicket(
		t, db, signer, provider.site, offer.ID, "review-evidence-wrong-hash",
	)
	if _, err := db.Exec(`
		INSERT INTO provider_pilot_review_events (
			provider_pilot_epoch_id, review_contract_version, review_type,
			subject_id, provider_claim_id, subject_snapshot_sha256,
			owner_reference, evidence_reference
		) VALUES ($1::uuid,$2,'ticket',$3::uuid,$4::uuid,$5,$6,$7)`,
		postgresProviderPilotEpochID, models.ProviderPilotReviewContractV1,
		wrongHashTicket.ID, provider.claim.ID, strings.Repeat("0", 64),
		"owner:review:wrong-hash", "evidence:review:wrong-hash",
	); err == nil {
		t.Fatal("database accepted a fabricated pilot-review snapshot hash")
	} else {
		assertPostgresConstraint(t, err, "provider_pilot_review_snapshot_hash")
	}

	var wrongPilotID string
	if err := db.QueryRow(`SELECT uuid_generate_v4()::text`).Scan(&wrongPilotID); err != nil {
		t.Fatalf("allocate wrong review pilot: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO provider_pilot_review_events (
			provider_pilot_epoch_id, review_contract_version, review_type,
			subject_id, provider_claim_id, subject_snapshot_sha256,
			owner_reference, evidence_reference
		) VALUES ($1::uuid,$2,'ticket',$3::uuid,$4::uuid,$5,$6,$7)`,
		wrongPilotID, models.ProviderPilotReviewContractV1, wrongHashTicket.ID,
		provider.claim.ID, strings.Repeat("0", 64),
		"owner:review:wrong-pilot", "evidence:review:wrong-pilot",
	); err == nil {
		t.Fatal("database accepted a pilot review for a subject in another epoch")
	} else {
		assertPostgresConstraint(t, err, "provider_pilot_review_subject")
	}

	// Reviews are append-only, while privacy retention can still remove the
	// underlying controlled-intent projection from future review reads.
	providerReview, err := models.GetProviderPilotReviewCandidate(
		db, postgresProviderPilotEpochID, "provider", provider.claim.ID,
	)
	if err != nil || providerReview.ExistingReviewID == "" {
		t.Fatalf("read append-only provider review candidate=%#v err=%v", providerReview, err)
	}
	var expectedOwnerReference string
	if err := db.QueryRow(`
		SELECT owner_reference FROM provider_pilot_review_events
		WHERE id=$1::uuid`, providerReview.ExistingReviewID).Scan(&expectedOwnerReference); err != nil {
		t.Fatalf("read original append-only review owner reference: %v", err)
	}
	for operation, query := range map[string]string{
		"update": `UPDATE provider_pilot_review_events SET owner_reference='owner:mutated' WHERE id=$1::uuid`,
		"delete": `DELETE FROM provider_pilot_review_events WHERE id=$1::uuid`,
	} {
		result, err := db.Exec(query, providerReview.ExistingReviewID)
		if err != nil {
			t.Fatalf("%s append-only pilot review: %v", operation, err)
		}
		if affected, err := result.RowsAffected(); err != nil || affected != 0 {
			t.Fatalf("%s append-only pilot review affected=%d err=%v, want 0", operation, affected, err)
		}
	}
	var preservedOwnerReference string
	if err := db.QueryRow(`
		SELECT owner_reference FROM provider_pilot_review_events
		WHERE id=$1::uuid`, providerReview.ExistingReviewID).Scan(&preservedOwnerReference); err != nil {
		t.Fatalf("read preserved append-only review: %v", err)
	}
	if preservedOwnerReference != expectedOwnerReference {
		t.Fatalf("append-only review owner reference=%q", preservedOwnerReference)
	}

	redactedTicket, _ := createPostgresUnhandedActionTicket(
		t, db, signer, provider.site, offer.ID, "review-evidence-redacted",
	)
	if _, err := db.Exec(`DELETE FROM search_receipts WHERE id=$1::uuid`, redactedTicket.SearchReceiptID); err != nil {
		t.Fatalf("redact pilot-review ticket through receipt retention: %v", err)
	}
	if _, err := models.GetProviderPilotReviewCandidate(
		db, postgresProviderPilotEpochID, "ticket", redactedTicket.ID,
	); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("redacted ticket review candidate error=%v, want sql.ErrNoRows", err)
	}
}

func providerHashPatternForPostgresTest(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil && strings.ToLower(value) == value
}

// exerciseProviderCommercialProofPostgres proves the 3/5/2/1 gate from
// provider-authenticated and owner-verified events. Negative evidence is kept
// physically present where valid, while forbidden states are proved to fail at
// their write boundary.
func exerciseProviderCommercialProofPostgres(
	t *testing.T,
	db *sql.DB,
	signer *providerexchange.Signer,
) {
	t.Helper()
	const bounty int64 = 1_000

	baseline := readPostgresVerifiedProof(t, db, signer)

	// Operator-only legacy funding and a free-form terms reference are useful
	// operational notes, but neither substitutes for provider-authenticated
	// acceptance plus owner verification.
	legacyFundProvider := createPostgresProviderIdentity(t, db, "proof-legacy-fund")
	legacyFundOffer := createPostgresCommercialOffer(
		t, db, legacyFundProvider, "legacy-fund", "prepaid", bounty,
	)
	if _, err := models.FundProviderOffer(
		db, legacyFundOffer.ID, bounty, "usd", "pg-proof-legacy-operator-fund",
	); err != nil {
		t.Fatalf("record operator-only legacy funding: %v", err)
	}
	if _, err := models.ActivateProviderOffer(
		db, legacyFundOffer.ID, "operator:proof-legacy-fund", "evidence:proof-legacy-fund",
	); !errors.Is(err, models.ErrProviderCommercialEvidenceRequired) {
		t.Fatalf("operator-only funded activation error = %v, want commercial evidence required", err)
	}
	assertPostgresProviderBalance(t, db, legacyFundOffer.ID, bounty)

	legacyTermsProvider := createPostgresProviderIdentity(t, db, "proof-legacy-terms")
	legacyTermsOffer := createPostgresCommercialOffer(
		t, db, legacyTermsProvider, "legacy-terms", "terms", bounty,
	)
	if _, err := models.ActivateProviderOffer(
		db, legacyTermsOffer.ID, "operator:proof-legacy-terms", "evidence:proof-legacy-terms",
	); !errors.Is(err, models.ErrProviderCommercialEvidenceRequired) {
		t.Fatalf("operator-only terms activation error = %v, want commercial evidence required", err)
	}

	// Once a provider-authenticated pilot-company acceptance is owner verified,
	// new operator-only budget entries are rejected rather than merely excluded
	// from proof. This prevents a mixed verified/unverified commercial ledger.
	legacyMutationProvider := createPostgresCommercialProvider(t, db, "proof-legacy-mutation")
	legacyMutationOffer := createPostgresCommercialOffer(
		t, db, legacyMutationProvider, "legacy-mutation", "prepaid", bounty,
	)
	if _, err := models.FundProviderOffer(
		db, legacyMutationOffer.ID, bounty, "usd", "pg-proof-legacy-mutation",
	); !errors.Is(err, models.ErrProviderLegacyBudgetMutation) {
		t.Fatalf("legacy fund after pilot verification error = %v, want ErrProviderLegacyBudgetMutation", err)
	}
	if got := readPostgresVerifiedProof(t, db, signer); !reflect.DeepEqual(got, baseline) {
		t.Fatalf("operator-only legacy evidence changed verified proof: before=%#v after=%#v", baseline, got)
	}

	// One acceptance cannot be rebound to a different owner company digest, and
	// one canonical company digest cannot be counted through a second claim.
	hashProvider := createPostgresCommercialProvider(t, db, "proof-company-hash")
	if _, _, err := models.VerifyProviderPilotCompany(
		db,
		hashProvider.company.ProviderAcceptanceEventID,
		postgresPayloadHash("proof-mismatched-company"),
		hashProvider.company.OperatorReference,
		hashProvider.company.IdentityEvidenceReference,
	); !errors.Is(err, models.ErrProviderIdempotency) {
		t.Fatalf("pilot-company hash rewrite error = %v, want idempotency conflict", err)
	}
	duplicateIdentity := createPostgresProviderIdentity(t, db, "proof-duplicate-company")
	duplicateAcceptance := recordPostgresPilotAcceptance(
		t, db, duplicateIdentity.key, "proof-duplicate-company",
	)
	if _, _, err := models.VerifyProviderPilotCompany(
		db,
		duplicateAcceptance.ID,
		hashProvider.companyHash,
		"operator:proof-duplicate-company",
		"evidence:proof-duplicate-company",
	); !errors.Is(err, models.ErrProviderIdempotency) {
		t.Fatalf("duplicate company digest error = %v, want idempotency conflict", err)
	}
	if got := readPostgresVerifiedProof(t, db, signer); !reflect.DeepEqual(got, baseline) {
		t.Fatalf("duplicate/hash-mismatch fixtures changed verified proof: before=%#v after=%#v", baseline, got)
	}

	// A provider-reported acceptance that is later invalidated and credited is
	// not a handoff. Funding and reversal also share the claim-row then offer
	// lock order: a concurrent partial reversal and fund must both complete,
	// preserve the exact net balance, and leave truthful verified proof.
	reversalProvider := createPostgresCommercialProvider(t, db, "proof-reversal")
	reversalOffer := createPostgresCommercialOffer(
		t, db, reversalProvider, "reversal", "prepaid", bounty,
	)
	reversalFunding := recordPostgresVerifiedFunding(
		t, db, reversalOffer.ID, 4*bounty, "proof-reversal-initial", "", time.Time{},
	)
	if _, err := activatePostgresProviderOffer(
		t, db, reversalOffer.ID, "operator:proof-reversal", "evidence:proof-reversal",
	); err != nil {
		t.Fatalf("activate reversal fixture offer: %v", err)
	}
	reversalTicket := createPostgresActionTicket(
		t, db, signer, reversalProvider.site, reversalOffer.ID, "proof-reversal-ticket",
	)
	recordPostgresOutcome(
		t, db, signer, reversalProvider.key, reversalTicket.ID, "accepted", "proof-reversal-accepted",
	)
	recordPostgresOutcome(
		t, db, signer, reversalProvider.key, reversalTicket.ID, "invalid", "proof-reversal-invalid",
	)
	afterInvalid := readPostgresVerifiedProof(t, db, signer)
	if afterInvalid.acceptedHandoffs != baseline.acceptedHandoffs ||
		afterInvalid.activations != baseline.activations ||
		afterInvalid.renewals != baseline.renewals {
		t.Fatalf("invalid/credited outcome entered verified result proof: before=%#v after=%#v", baseline, afterInvalid)
	}

	type commercialRaceResult struct {
		operation string
		event     *models.ProviderCommercialCommitmentEvent
		created   bool
		err       error
	}
	raceEffectiveAt := postgresDatabaseClock(t, db)
	raceStart := make(chan struct{})
	raceResults := make(chan commercialRaceResult, 2)
	go func() {
		<-raceStart
		event, created, err := models.ReverseVerifiedProviderFunding(db, models.ProviderFundingReversalInput{
			RelatedCommitmentEventID: reversalFunding.ID,
			AmountCents:              bounty,
			SourceSystem:             "pg-settlement",
			SourceEventID:            "proof-reversal-race-partial",
			SourceEffectiveAt:        raceEffectiveAt,
			OperatorReference:        "operator:proof-reversal-race-partial",
			OwnerEvidenceReference:   "evidence:proof-reversal-race-partial",
		})
		raceResults <- commercialRaceResult{
			operation: "reversal", event: event, created: created, err: err,
		}
	}()
	go func() {
		<-raceStart
		event, created, err := models.RecordVerifiedProviderFunding(db, models.VerifiedProviderFundingInput{
			ProviderOfferID:        reversalOffer.ID,
			AmountCents:            bounty,
			Currency:               "usd",
			SourceSystem:           "pg-settlement",
			SourceEventID:          "proof-reversal-race-fund",
			SourceEffectiveAt:      raceEffectiveAt,
			OperatorReference:      "operator:proof-reversal-race-fund",
			OwnerEvidenceReference: "evidence:proof-reversal-race-fund",
		})
		raceResults <- commercialRaceResult{
			operation: "fund", event: event, created: created, err: err,
		}
	}()
	close(raceStart)
	raceTimer := time.NewTimer(15 * time.Second)
	defer raceTimer.Stop()
	var raceFund *models.ProviderCommercialCommitmentEvent
	for range 2 {
		select {
		case result := <-raceResults:
			if result.err != nil {
				t.Fatalf("concurrent verified %s: %v", result.operation, result.err)
			}
			if !result.created || result.event == nil {
				t.Fatalf("concurrent verified %s result = %#v", result.operation, result)
			}
			if result.operation == "fund" {
				raceFund = result.event
			}
		case <-raceTimer.C:
			t.Fatal("concurrent verified reversal/funding did not complete within 15s")
		}
	}
	if raceFund == nil {
		t.Fatal("concurrent verified funding result missing")
	}
	assertPostgresProviderBalance(t, db, reversalOffer.ID, 4*bounty)
	raceOffer, err := models.GetProviderOffer(db, reversalProvider.accountID, reversalOffer.ID)
	if err != nil {
		t.Fatalf("read offer after concurrent reversal/funding: %v", err)
	}
	if raceOffer.Status != "active" {
		t.Fatalf("offer status after concurrent partial reversal/funding = %q, want active", raceOffer.Status)
	}
	afterRace := readPostgresVerifiedProof(t, db, signer)
	if afterRace.companies != baseline.companies+1 ||
		afterRace.acceptedHandoffs != baseline.acceptedHandoffs ||
		afterRace.activations != baseline.activations ||
		afterRace.renewals != baseline.renewals ||
		afterRace.conversions != baseline.conversions ||
		afterRace.settled["usd"] != baseline.settled["usd"]+4*bounty {
		t.Fatalf("concurrent reversal/funding proof is not truthful: before=%#v after=%#v", baseline, afterRace)
	}

	// Fully reverse both verified source funds. This removes the offer from the
	// qualified-company and settlement aggregates and pauses it at zero balance.
	if _, created, err := models.ReverseVerifiedProviderFunding(db, models.ProviderFundingReversalInput{
		RelatedCommitmentEventID: reversalFunding.ID,
		AmountCents:              reversalFunding.AmountCents - bounty,
		SourceSystem:             "pg-settlement",
		SourceEventID:            "proof-reversal-initial-rest",
		SourceEffectiveAt:        postgresDatabaseClock(t, db),
		OperatorReference:        "operator:proof-reversal-initial-rest",
		OwnerEvidenceReference:   "evidence:proof-reversal-initial-rest",
	}); err != nil {
		t.Fatalf("record remaining initial funding reversal: %v", err)
	} else if !created {
		t.Fatal("remaining initial funding reversal unexpectedly replayed")
	}
	if _, created, err := models.ReverseVerifiedProviderFunding(db, models.ProviderFundingReversalInput{
		RelatedCommitmentEventID: raceFund.ID,
		AmountCents:              raceFund.AmountCents,
		SourceSystem:             "pg-settlement",
		SourceEventID:            "proof-reversal-race-fund-full",
		SourceEffectiveAt:        postgresDatabaseClock(t, db),
		OperatorReference:        "operator:proof-reversal-race-fund-full",
		OwnerEvidenceReference:   "evidence:proof-reversal-race-fund-full",
	}); err != nil {
		t.Fatalf("record concurrent fund reversal: %v", err)
	} else if !created {
		t.Fatal("concurrent fund reversal unexpectedly replayed")
	}
	assertPostgresProviderBalance(t, db, reversalOffer.ID, 0)
	reversedOffer, err := models.GetProviderOffer(
		db, reversalProvider.accountID, reversalOffer.ID,
	)
	if err != nil {
		t.Fatalf("read offer after full funding reversal: %v", err)
	}
	if reversedOffer.Status != "paused" {
		t.Fatalf("offer status after full reversal = %q, want paused", reversedOffer.Status)
	}
	if got := readPostgresVerifiedProof(t, db, signer); !reflect.DeepEqual(got, baseline) {
		t.Fatalf("fully reversed provider changed verified proof: before=%#v after=%#v", baseline, got)
	}

	// Historic outcomes remain in the ledger when DNS ownership ages out, but a
	// stale claim contributes no company, handoff, activation, or money proof.
	staleProvider := createPostgresCommercialProvider(t, db, "proof-stale-claim")
	staleOffer := createPostgresCommercialOffer(
		t, db, staleProvider, "stale-claim", "prepaid", bounty,
	)
	recordPostgresVerifiedFunding(
		t, db, staleOffer.ID, 4*bounty, "proof-stale-initial", "", time.Time{},
	)
	if _, err := activatePostgresProviderOffer(
		t, db, staleOffer.ID, "operator:proof-stale", "evidence:proof-stale",
	); err != nil {
		t.Fatalf("activate stale-claim fixture offer: %v", err)
	}
	staleTicket := createPostgresActionTicket(
		t, db, signer, staleProvider.site, staleOffer.ID, "proof-stale-ticket",
	)
	recordPostgresOutcome(
		t, db, signer, staleProvider.key, staleTicket.ID, "accepted", "proof-stale-accepted",
	)
	if _, err := db.Exec(`
		UPDATE provider_claims
		SET verification_last_succeeded_at=clock_timestamp()-INTERVAL '8 days'
		WHERE id=$1::uuid`, staleProvider.claim.ID); err != nil {
		t.Fatalf("age verified provider claim beyond freshness window: %v", err)
	}
	if got := readPostgresVerifiedProof(t, db, signer); !reflect.DeepEqual(got, baseline) {
		t.Fatalf("stale-claim evidence changed verified proof: before=%#v after=%#v", baseline, got)
	}

	// Build three fresh, owner-deduplicated providers. Five accepted tickets,
	// two later activations, and one source-timed replenishment must move the
	// verified counters by exactly 3/5/2/1.
	positiveSuffixes := []string{"proof-positive-a", "proof-positive-b", "proof-positive-c"}
	positiveProviders := make([]*postgresCommercialProvider, 0, len(positiveSuffixes))
	positiveOffers := make([]*models.ProviderOffer, 0, len(positiveSuffixes))
	initialFunding := make([]*models.ProviderCommercialCommitmentEvent, 0, len(positiveSuffixes))
	for _, suffix := range positiveSuffixes {
		provider := createPostgresCommercialProvider(t, db, suffix)
		offer := createPostgresCommercialOffer(t, db, provider, suffix, "prepaid", bounty)
		funding := recordPostgresVerifiedFunding(
			t, db, offer.ID, 10*bounty, suffix+"-initial", "", time.Time{},
		)
		if _, err := activatePostgresProviderOffer(
			t, db, offer.ID, "operator:"+suffix, "evidence:"+suffix,
		); err != nil {
			t.Fatalf("activate positive commercial offer %q: %v", suffix, err)
		}
		positiveProviders = append(positiveProviders, provider)
		positiveOffers = append(positiveOffers, offer)
		initialFunding = append(initialFunding, funding)
	}

	firstTicket := createPostgresActionTicket(
		t, db, signer, positiveProviders[0].site, positiveOffers[0].ID, "proof-positive-a-one",
	)
	recordPostgresOutcome(
		t, db, signer, positiveProviders[0].key, firstTicket.ID, "accepted", "proof-positive-a-one-accepted",
	)

	// Recording a replenishment late cannot rewrite its economic time. The
	// owner-verified source timestamp must be strictly later than the qualifying
	// charge, not merely inserted after it.
	beforeDelayedFund := readPostgresVerifiedProof(t, db, signer)
	if _, _, err := models.RecordVerifiedProviderFunding(db, models.VerifiedProviderFundingInput{
		ProviderOfferID:          positiveOffers[0].ID,
		AmountCents:              bounty,
		Currency:                 "usd",
		SourceSystem:             "pg-settlement",
		SourceEventID:            "proof-positive-delayed-replenishment",
		SourceEffectiveAt:        initialFunding[0].SourceEffectiveAt,
		QualifyingActionTicketID: firstTicket.ID,
		OperatorReference:        "operator:proof-positive-delayed",
		OwnerEvidenceReference:   "evidence:proof-positive-delayed",
	}); err == nil {
		t.Fatal("delayed settlement timestamp qualified as replenishment")
	} else {
		assertPostgresConstraint(t, err, "provider_commercial_commitment_replenishment")
	}
	if got := readPostgresVerifiedProof(t, db, signer); !reflect.DeepEqual(got, beforeDelayedFund) {
		t.Fatalf("rejected delayed fund changed verified proof: before=%#v after=%#v", beforeDelayedFund, got)
	}

	recordPostgresOutcome(
		t, db, signer, positiveProviders[0].key, firstTicket.ID, "activated", "proof-positive-a-one-activated",
	)
	secondTicket := createPostgresActionTicket(
		t, db, signer, positiveProviders[0].site, positiveOffers[0].ID, "proof-positive-a-two",
	)
	recordPostgresOutcome(
		t, db, signer, positiveProviders[0].key, secondTicket.ID, "accepted", "proof-positive-a-two-accepted",
	)
	recordPostgresOutcome(
		t, db, signer, positiveProviders[0].key, secondTicket.ID, "activated", "proof-positive-a-two-activated",
	)
	for _, suffix := range []string{"proof-positive-b-one", "proof-positive-b-two"} {
		ticket := createPostgresActionTicket(
			t, db, signer, positiveProviders[1].site, positiveOffers[1].ID, suffix,
		)
		recordPostgresOutcome(
			t, db, signer, positiveProviders[1].key, ticket.ID, "accepted", suffix+"-accepted",
		)
	}
	fifthTicket := createPostgresActionTicket(
		t, db, signer, positiveProviders[2].site, positiveOffers[2].ID, "proof-positive-c-one",
	)
	recordPostgresOutcome(
		t, db, signer, positiveProviders[2].key, fifthTicket.ID, "accepted", "proof-positive-c-one-accepted",
	)
	recordPostgresVerifiedFunding(
		t, db, positiveOffers[0].ID, bounty, "proof-positive-replenishment",
		firstTicket.ID, postgresDatabaseClock(t, db),
	)

	positiveProof := readPostgresVerifiedProof(t, db, signer)
	if positiveProof.companies != baseline.companies+3 ||
		positiveProof.chargedProviders["accepted"] != baseline.chargedProviders["accepted"]+3 ||
		positiveProof.offerReturns["accepted"] != baseline.offerReturns["accepted"]+5 ||
		positiveProof.acceptedHandoffs != baseline.acceptedHandoffs+5 ||
		positiveProof.activations != baseline.activations+2 ||
		positiveProof.renewals != baseline.renewals+1 {
		t.Fatalf("verified commercial proof delta = companies:%d accepted-arm providers:%d accepted-arm returns:%d handoffs:%d activations:%d renewals:%d; want 3/3/5/5/2/1 over %#v",
			positiveProof.companies-baseline.companies,
			positiveProof.chargedProviders["accepted"]-baseline.chargedProviders["accepted"],
			positiveProof.offerReturns["accepted"]-baseline.offerReturns["accepted"],
			positiveProof.acceptedHandoffs-baseline.acceptedHandoffs,
			positiveProof.activations-baseline.activations,
			positiveProof.renewals-baseline.renewals,
			baseline,
		)
	}
	if !positiveProof.pilotMet {
		t.Fatalf("truthful 3/5/2/1 proof did not satisfy pilot gate: %#v", positiveProof)
	}
	positiveAggregate, err := models.GetProviderExchangeProof(
		db, postgresProviderPilotEpochID, signer,
	)
	if err != nil {
		t.Fatalf("read proof mechanism aggregates: %v", err)
	}
	if positiveAggregate.VerifiedProviderOfferReturns < positiveAggregate.VerifiedObservedHandoffs ||
		positiveAggregate.VerifiedObservedHandoffs < positiveAggregate.VerifiedProviderAcceptedHandoffs ||
		positiveAggregate.VerifiedAcceptedLatencySamples != positiveAggregate.VerifiedProviderAcceptedHandoffs ||
		positiveAggregate.VerifiedActivatedLatencySamples != positiveAggregate.VerifiedProviderConfirmedActivations ||
		positiveAggregate.VerifiedConvertedLatencySamples != positiveAggregate.VerifiedProviderConfirmedConversions ||
		positiveAggregate.VerifiedAcceptedMedianSeconds < 0 ||
		positiveAggregate.VerifiedActivatedMedianSeconds < 0 ||
		positiveAggregate.VerifiedConvertedMedianSeconds < 0 {
		t.Fatalf("invalid proof mechanism aggregates: %#v", positiveAggregate)
	}

	// A sixth accepted receipt that is later marked duplicate/invalid and fully
	// credited must leave the financial proof unchanged. Its genuine offer
	// return remains part of the top-of-funnel exposure denominator.
	creditedTicket := createPostgresActionTicket(
		t, db, signer, positiveProviders[0].site, positiveOffers[0].ID, "proof-positive-credited",
	)
	recordPostgresOutcome(
		t, db, signer, positiveProviders[0].key, creditedTicket.ID, "accepted", "proof-positive-credited-accepted",
	)
	recordPostgresOutcome(
		t, db, signer, positiveProviders[0].key, creditedTicket.ID, "duplicate", "proof-positive-credited-duplicate",
	)
	if got := readPostgresVerifiedProof(t, db, signer); true {
		want := positiveProof
		want.offerReturns = maps.Clone(positiveProof.offerReturns)
		want.offerReturns["accepted"]++
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("duplicate/credited outcome changed truthful proof beyond one offer return: want=%#v after=%#v", want, got)
		}
	}

	// A terms click or owner verification recorded before any real charge is not
	// a renewal. Only a later provider-authenticated extension whose independent
	// evidence effective time is after a charged observed handoff may count.
	termsProvider := createPostgresCommercialProvider(t, db, "proof-terms-renewal")
	termsOffer := createPostgresCommercialOffer(
		t, db, termsProvider, "proof-terms-renewal", "terms", bounty,
	)
	initialAcceptance, initialTerms := recordPostgresVerifiedTerms(
		t, db, termsProvider.key, termsOffer.ID,
		"proof-terms-initial", "", "",
	)
	if _, err := activatePostgresProviderOffer(
		t, db, termsOffer.ID, "operator:proof-terms", "evidence:proof-terms",
	); err != nil {
		t.Fatalf("activate terms-renewal proof offer: %v", err)
	}
	prechargeAcceptance, prechargeRenewal := recordPostgresVerifiedTerms(
		t, db, termsProvider.key, termsOffer.ID,
		"proof-terms-precharge-renewal", initialAcceptance.ID, initialTerms.ID,
	)
	beforeTermsCharge := readPostgresVerifiedProof(t, db, signer)
	if beforeTermsCharge.renewals != positiveProof.renewals {
		t.Fatalf("pre-charge terms extension counted as renewal: before=%#v after=%#v", positiveProof, beforeTermsCharge)
	}
	termsTicket := createPostgresActionTicket(
		t, db, signer, termsProvider.site, termsOffer.ID, "proof-terms-charged-ticket",
	)
	termsOutcome := recordPostgresOutcome(
		t, db, signer, termsProvider.key, termsTicket.ID,
		"accepted", "proof-terms-charged-accepted",
	)
	settlementOrder, settlementCreated, err := models.PrepareProviderSettlement(db, termsOutcome.ID)
	if err != nil || !settlementCreated {
		t.Fatalf("prepare exact terms settlement = order:%#v created:%t err:%v", settlementOrder, settlementCreated, err)
	}
	const checkoutID = "cs_pgproofterms0001"
	if created, err := models.RecordProviderSettlementCheckoutSession(db, settlementOrder.ID, checkoutID); err != nil || !created {
		t.Fatalf("record exact terms checkout = created:%t err:%v", created, err)
	}
	paidAt := postgresDatabaseClock(t, db)
	paidReceipt, paidCreated, err := models.RecordProviderSettlementPayment(db, models.ProviderSettlementPaymentInput{
		OrderID:                    settlementOrder.ID,
		StripeCheckoutSessionID:    checkoutID,
		StripePaymentIntentID:      "pi_pgproofterms0001",
		StripeEventID:              "evt_pgproofterms0001",
		StripeChargeID:             "ch_pgproofterms0001",
		StripeBalanceTransactionID: "txn_pgproofterms0001",
		AmountCents:                settlementOrder.AmountCents,
		ProcessorFeeCents:          59,
		ProcessorNetCents:          settlementOrder.AmountCents - 59,
		Currency:                   settlementOrder.Currency,
		ProcessorStatus:            "available",
		ProcessorAvailableOn:       paidAt,
		ProcessorObservedAt:        paidAt,
		PaidAt:                     paidAt,
	})
	if err != nil || !paidCreated {
		t.Fatalf("record exact terms payment = receipt:%#v created:%t err:%v", paidReceipt, paidCreated, err)
	}
	paidProof, err := models.GetProviderExchangeProof(db, postgresProviderPilotEpochID, signer)
	if err != nil {
		t.Fatalf("read paid terms proof: %v", err)
	}
	if !paidProof.SettlementReceiptIntegrityValid || paidProof.VerifiedProviderPaidSettlements != 1 ||
		!paidProof.ProcessorNetReceiptIntegrityValid || paidProof.VerifiedProviderAvailableSettlements != 1 ||
		paidProof.RejectedProviderSettlementReceipts != 0 || paidProof.VerifiedPaidLatencySamples != 1 ||
		paidProof.RejectedProviderProcessorNetReceipts != 0 ||
		paidProof.VerifiedTermsPaidByCurrency["usd"] != settlementOrder.AmountCents ||
		paidProof.VerifiedProcessorFeesByCurrency["usd"] != 59 ||
		paidProof.VerifiedProcessorNetByCurrency["usd"] != settlementOrder.AmountCents-59 ||
		paidProof.VerifiedPaidMedianSeconds < 0 {
		t.Fatalf("exact paid terms proof aggregate = %#v", paidProof)
	}
	afterTermsCharge := readPostgresVerifiedProof(t, db, signer)
	if afterTermsCharge.renewals != positiveProof.renewals {
		t.Fatalf("pre-charge terms extension counted after later charge: before=%#v after=%#v", positiveProof, afterTermsCharge)
	}
	postchargeAcceptance, postchargeRenewal := recordPostgresVerifiedTerms(
		t, db, termsProvider.key, termsOffer.ID,
		"proof-terms-postcharge-renewal", prechargeAcceptance.ID, prechargeRenewal.ID,
	)
	afterTermsRenewal := readPostgresVerifiedProof(t, db, signer)
	if afterTermsRenewal.renewals != positiveProof.renewals+1 ||
		afterTermsRenewal.acceptedHandoffs != positiveProof.acceptedHandoffs+1 ||
		afterTermsRenewal.companies != positiveProof.companies+1 {
		t.Fatalf("post-charge exact terms renewal proof = before:%#v after:%#v", positiveProof, afterTermsRenewal)
	}
	replayInput := models.VerifiedProviderTermsInput{
		ProviderOfferID:           termsOffer.ID,
		ProviderAcceptanceEventID: postchargeAcceptance.ID,
		RelatedCommitmentEventID:  prechargeRenewal.ID,
		SourceSystem:              "pg-contract",
		SourceEventID:             "proof-terms-postcharge-renewal",
		SourceEffectiveAt:         postchargeRenewal.SourceEffectiveAt,
		OperatorReference:         "operator:proof-terms-postcharge-renewal",
		OwnerEvidenceReference:    "evidence:proof-terms-postcharge-renewal",
	}
	if replay, created, err := models.RecordVerifiedProviderTerms(db, replayInput); err != nil || created || !replay.Replayed || replay.ID != postchargeRenewal.ID {
		t.Fatalf("exact verified-terms replay = event:%#v created:%t err:%v", replay, created, err)
	}
	replayInput.SourceEffectiveAt = replayInput.SourceEffectiveAt.Add(-time.Second)
	if _, _, err := models.RecordVerifiedProviderTerms(db, replayInput); !errors.Is(err, models.ErrProviderIdempotency) {
		t.Fatalf("verified-terms effective-time rewrite error = %v, want ErrProviderIdempotency", err)
	}

	// A direct database writer can satisfy relational shape checks but cannot
	// forge the NHS outcome MAC. Exact-pilot proof must expose the contaminated
	// row, preserve the authentic counters, and fail the commercial threshold.
	forgedTicket := createPostgresActionTicket(
		t, db, signer, termsProvider.site, termsOffer.ID, "proof-forged-outcome",
	)
	var forgedReceiptID, forgedEventID string
	if err := db.QueryRow(`
		SELECT uuid_generate_v4()::text, uuid_generate_v4()::text`).Scan(
		&forgedReceiptID, &forgedEventID,
	); err != nil {
		t.Fatalf("allocate forged outcome IDs: %v", err)
	}
	forgedAt := postgresDatabaseClock(t, db).Truncate(time.Second)
	forgedCanonical, err := providerexchange.CanonicalOutcomeReceipt(providerexchange.OutcomeReceipt{
		Version:            providerexchange.OutcomeReceiptVersion,
		KeyID:              "pg-release-v1",
		ReceiptID:          forgedReceiptID,
		TicketID:           forgedTicket.ID,
		OfferID:            termsOffer.ID,
		NHSEventID:         forgedEventID,
		Outcome:            providerexchange.OutcomeAccepted,
		ProviderReportedAt: forgedAt.Unix(),
		RecordedAt:         forgedAt.Unix(),
		ExpiresAt:          forgedAt.Add(models.OutcomeReceiptValidity).Unix(),
		ChargedMinor:       0,
		Currency:           "usd",
		ChargeStatus:       providerexchange.ChargeStatusNone,
	})
	if err != nil {
		t.Fatalf("canonicalize forged outcome: %v", err)
	}
	insertForgedOutcome := func(canonical string, recordedAt time.Time) error {
		_, insertErr := db.Exec(`
			INSERT INTO outcome_receipts (
				id, nhs_event_id, provider_claim_id, provider_offer_id,
				action_ticket_id, provider_api_key_id, idempotency_key_hash,
				payload_hash, outcome, billed_cents, charge_status, currency,
				signed_receipt, signature, provider_reported_at, created_at
			) VALUES (
				$1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,$6,$7,$8,
				'accepted',0,'none','usd',$9,$10,$11,$11
			)`,
			forgedReceiptID, forgedEventID, termsProvider.claim.ID, termsOffer.ID,
			forgedTicket.ID, termsProvider.key.ID,
			postgresPayloadHash("proof-forged-outcome-idempotency"),
			postgresPayloadHash("proof-forged-outcome-payload"),
			canonical, strings.Repeat("A", 43), recordedAt,
		)
		return insertErr
	}
	if _, err := db.Exec(`
		UPDATE sites SET category='spam' WHERE id=$1::uuid`, termsProvider.site.ID); err != nil {
		t.Fatalf("reclassify direct-outcome provider site as spam: %v", err)
	}
	if err := insertForgedOutcome(string(forgedCanonical), forgedAt); err == nil {
		t.Fatal("database accepted a positive outcome after provider spam reclassification")
	} else {
		assertPostgresConstraint(t, err, "provider_pilot_outcome_enrollment_eligibility")
	}
	if _, err := db.Exec(`
		UPDATE sites SET category='developer' WHERE id=$1::uuid`, termsProvider.site.ID); err != nil {
		t.Fatalf("restore direct-outcome provider site category: %v", err)
	}
	if err := insertForgedOutcome(string(forgedCanonical), forgedAt.Add(-10*time.Minute)); err == nil {
		t.Fatal("database accepted a backdated pilot outcome")
	} else {
		assertPostgresConstraint(t, err, "provider_pilot_outcome_database_clock")
	}
	mismatchedCanonical := strings.Replace(
		string(forgedCanonical), forgedEventID, forgedReceiptID, 1,
	)
	if err := insertForgedOutcome(mismatchedCanonical, forgedAt); err == nil {
		t.Fatal("database accepted an outcome canonical payload for a different event")
	} else {
		assertPostgresConstraint(t, err, "provider_pilot_outcome_canonical_row")
	}
	if err := insertForgedOutcome(string(forgedCanonical), forgedAt); err != nil {
		t.Fatalf("insert relationally valid forged outcome: %v", err)
	}
	if _, err := db.Exec(`
		UPDATE action_tickets SET status='accepted', updated_at=$2
		WHERE id=$1::uuid`, forgedTicket.ID, forgedAt); err != nil {
		t.Fatalf("move forged-outcome ticket state: %v", err)
	}
	contaminatedProof := readPostgresVerifiedProof(t, db, signer)
	if contaminatedProof.integrityValid || contaminatedProof.rejectedOutcomes != 1 ||
		contaminatedProof.rejectedLedger != 0 ||
		contaminatedProof.acceptedHandoffs != afterTermsRenewal.acceptedHandoffs ||
		contaminatedProof.activations != afterTermsRenewal.activations ||
		contaminatedProof.pilotMet {
		t.Fatalf(
			"forged outcome contaminated exact proof = before:%#v after:%#v",
			afterTermsRenewal, contaminatedProof,
		)
	}

	// Receipt-to-ledger equality is not enough: an unsigned direct credit under
	// a different external reference must be discovered by the converse ledger
	// scan. The economic credit reverses this ticket for funnel/renewal counts,
	// while its lack of an authentic receipt keeps the entire proof gate red.
	if _, err := db.Exec(`
		INSERT INTO provider_budget_ledger (
			provider_claim_id, provider_offer_id, action_ticket_id,
			entry_type, amount_cents, currency, external_reference
		) VALUES ($1::uuid,$2::uuid,$3::uuid,'credit',$4,'usd',$5)`,
		positiveProviders[0].claim.ID, positiveOffers[0].ID, firstTicket.ID,
		bounty, "unsigned-credit:proof-positive-a-one"); err != nil {
		t.Fatalf("insert adversarial unsigned pilot credit: %v", err)
	}
	unsignedCreditProof := readPostgresVerifiedProof(t, db, signer)
	if unsignedCreditProof.integrityValid ||
		unsignedCreditProof.rejectedOutcomes != contaminatedProof.rejectedOutcomes ||
		unsignedCreditProof.rejectedLedger != contaminatedProof.rejectedLedger+1 ||
		unsignedCreditProof.acceptedHandoffs != contaminatedProof.acceptedHandoffs-1 ||
		unsignedCreditProof.activations != contaminatedProof.activations-1 ||
		unsignedCreditProof.renewals != contaminatedProof.renewals-1 ||
		unsignedCreditProof.pilotMet {
		t.Fatalf(
			"unsigned credit escaped exact proof = before:%#v after:%#v",
			contaminatedProof, unsignedCreditProof,
		)
	}

	beforeFuturePayment, err := models.GetProviderExchangeProof(
		db, postgresProviderPilotEpochID, signer,
	)
	if err != nil {
		t.Fatalf("read proof before future-dated payment: %v", err)
	}
	futureTicket := createPostgresActionTicket(
		t, db, signer, termsProvider.site, termsOffer.ID, "proof-future-payment",
	)
	futureOutcome := recordPostgresOutcome(
		t, db, signer, termsProvider.key, futureTicket.ID,
		"accepted", "proof-future-payment-accepted",
	)
	futureOrder, created, err := models.PrepareProviderSettlement(db, futureOutcome.ID)
	if err != nil || !created {
		t.Fatalf("prepare future-dated settlement = order:%#v created:%t err:%v", futureOrder, created, err)
	}
	const futureCheckoutID = "cs_pgprooffuture0001"
	if created, err := models.RecordProviderSettlementCheckoutSession(db, futureOrder.ID, futureCheckoutID); err != nil || !created {
		t.Fatalf("record future-dated checkout = created:%t err:%v", created, err)
	}
	futurePaidAt := postgresDatabaseClock(t, db).Add(time.Hour)
	if receipt, created, err := models.RecordProviderSettlementPayment(db, models.ProviderSettlementPaymentInput{
		OrderID:                    futureOrder.ID,
		StripeCheckoutSessionID:    futureCheckoutID,
		StripePaymentIntentID:      "pi_pgprooffuture0001",
		StripeEventID:              "evt_pgprooffuture0001",
		StripeChargeID:             "ch_pgprooffuture0001",
		StripeBalanceTransactionID: "txn_pgprooffuture0001",
		AmountCents:                futureOrder.AmountCents,
		ProcessorFeeCents:          59,
		ProcessorNetCents:          futureOrder.AmountCents - 59,
		Currency:                   futureOrder.Currency,
		ProcessorStatus:            "available",
		ProcessorAvailableOn:       futurePaidAt,
		ProcessorObservedAt:        futurePaidAt,
		PaidAt:                     futurePaidAt,
	}); err == nil || created || receipt != nil {
		t.Fatalf("future-dated processor observation was accepted = receipt:%#v created:%t err:%v", receipt, created, err)
	}
	afterFuturePayment, err := models.GetProviderExchangeProof(
		db, postgresProviderPilotEpochID, signer,
	)
	if err != nil {
		t.Fatalf("read proof after future-dated payment: %v", err)
	}
	if !afterFuturePayment.SettlementReceiptIntegrityValid || !afterFuturePayment.ProcessorNetReceiptIntegrityValid ||
		afterFuturePayment.RejectedProviderSettlementReceipts != beforeFuturePayment.RejectedProviderSettlementReceipts ||
		afterFuturePayment.VerifiedProviderPaidSettlements != beforeFuturePayment.VerifiedProviderPaidSettlements ||
		afterFuturePayment.VerifiedProviderAvailableSettlements != beforeFuturePayment.VerifiedProviderAvailableSettlements {
		t.Fatalf("rejected future-dated payment changed proof: before=%#v after=%#v", beforeFuturePayment, afterFuturePayment)
	}
}

func createPostgresProviderIdentity(
	t *testing.T,
	db *sql.DB,
	suffix string,
) *postgresCommercialProvider {
	t.Helper()
	provider := &postgresCommercialProvider{}
	email := "pg-" + suffix + "@example.invalid"
	if err := db.QueryRow(`
		INSERT INTO accounts (email, plan, status, monthly_limit)
		VALUES ($1,'provider-pilot','active',0)
		RETURNING id`, email).Scan(&provider.accountID); err != nil {
		t.Fatalf("insert commercial-proof account %q: %v", suffix, err)
	}
	provider.site = models.Site{
		Domain:       suffix + ".example",
		URL:          "https://" + suffix + ".example",
		AgenticScore: 90,
	}
	if err := db.QueryRow(`
		INSERT INTO sites (
			domain, url, name, description, has_structured_api,
			agentic_score, category, crawl_status
		) VALUES ($1,$2,$3,'Commercial proof PostgreSQL fixture',true,$4,'developer','success')
		RETURNING id::text`,
		provider.site.Domain, provider.site.URL, "Provider "+suffix,
		provider.site.AgenticScore,
	).Scan(&provider.site.ID); err != nil {
		t.Fatalf("insert commercial-proof site %q: %v", suffix, err)
	}
	claim, rawChallenge, err := models.CreateProviderClaim(
		db, provider.accountID, provider.site.ID,
	)
	if err != nil {
		t.Fatalf("create commercial-proof claim %q: %v", suffix, err)
	}
	provider.claim, err = models.VerifyProviderClaim(
		db, provider.accountID, claim.ID, rawChallenge,
	)
	if err != nil {
		t.Fatalf("verify commercial-proof claim %q: %v", suffix, err)
	}
	provider.rawKey, provider.key, err = models.CreateProviderAPIKey(
		db, provider.accountID, provider.claim.ID,
	)
	if err != nil {
		t.Fatalf("create commercial-proof API key %q: %v", suffix, err)
	}
	return provider
}

func createPostgresCommercialProvider(
	t *testing.T,
	db *sql.DB,
	suffix string,
) *postgresCommercialProvider {
	t.Helper()
	if provider, ok := postgresCommercialProviderFixtures[suffix]; ok {
		delete(postgresCommercialProviderFixtures, suffix)
		return provider
	}
	return createPostgresCommercialProviderDirect(t, db, suffix)
}

func createPostgresCommercialProviderDirect(
	t *testing.T,
	db *sql.DB,
	suffix string,
) *postgresCommercialProvider {
	t.Helper()
	provider := createPostgresProviderIdentity(t, db, suffix)
	provider.companyHash = postgresPayloadHash("company:" + suffix)
	provider.company = establishPostgresPilotCompanyWithHash(
		t, db, provider.key, suffix, provider.companyHash,
	)
	return provider
}

func preparePostgresCommercialProviderFixtures(
	t *testing.T,
	db *sql.DB,
) []*postgresCommercialProvider {
	t.Helper()
	if postgresCommercialProviderFixtures != nil {
		t.Fatal("PostgreSQL provider pilot fixture registry already initialized")
	}
	suffixes := []string{
		"queue-superseded-terms",
		"queue-expired-terms",
		"queue-valid-terms",
		"queue-review-prepaid",
		"queue-review-renewed-terms",
		"queue-review-reversed-prepaid",
		"queue-stale-callback",
		"handoff-boundary",
		"proof-legacy-mutation",
		"proof-company-hash",
		"proof-reversal",
		"proof-stale-claim",
		"proof-positive-a",
		"proof-positive-b",
		"proof-positive-c",
		"proof-terms-renewal",
		"http-loopback",
		"review-evidence",
	}
	postgresCommercialProviderFixtures = make(
		map[string]*postgresCommercialProvider, len(suffixes),
	)
	providers := make([]*postgresCommercialProvider, 0, len(suffixes))
	for _, suffix := range suffixes {
		provider := createPostgresCommercialProviderDirect(t, db, suffix)
		postgresCommercialProviderFixtures[suffix] = provider
		providers = append(providers, provider)
	}
	return providers
}

func createPostgresActiveProviderPilot(
	t *testing.T,
	db *sql.DB,
	primaryClaimID string,
	primaryCompany *models.ProviderPilotCompany,
	cohortProviders []*postgresCommercialProvider,
) string {
	t.Helper()
	mutatePostgresStage1LedgerFixture(t, db, `
		UPDATE nhs_schema_migrations
		SET applied_at=clock_timestamp()-INTERVAL '15 days'
		WHERE name='025_stage1_fact_integrity.sql'`)
	var primarySite models.Site
	if err := db.QueryRow(`
		SELECT site.id::text, site.domain, site.agentic_score
		FROM provider_claims claim
		JOIN sites site ON site.id=claim.site_id
		WHERE claim.id=$1::uuid`, primaryClaimID).
		Scan(&primarySite.ID, &primarySite.Domain, &primarySite.AgenticScore); err != nil {
		t.Fatalf("read primary pilot site for Stage 1 eligibility: %v", err)
	}
	recordPostgresStage1EligibilityReceipt(t, db, primarySite, "release")
	var cleanupEligibilityReceiptID, cleanupEligibilityClaimID string
	for index, provider := range cohortProviders {
		receiptID := recordPostgresStage1EligibilityReceipt(
			t, db, provider.site, fmt.Sprintf("cohort-%02d", index+1),
		)
		if index == len(cohortProviders)-1 {
			cleanupEligibilityReceiptID = receiptID
			cleanupEligibilityClaimID = provider.claim.ID
		}
	}
	// This otherwise valid provider is deliberately never returned in Stage 1.
	// It proves that company verification alone cannot buy entry to the pilot.
	neverReturned := createPostgresCommercialProviderDirect(t, db, "stage1-never-returned")
	spamBeforeEnrollment := createPostgresCommercialProviderDirect(t, db, "stage1-spam-before-enroll")
	recordPostgresStage1EligibilityReceipt(
		t, db, spamBeforeEnrollment.site, "stage1-spam-before-enroll",
	)
	epoch, err := models.CreateProviderPilotEpoch(db, models.ProviderPilotEpochInput{
		DemandTopic:       "developer-tools",
		CohortLimit:       20,
		ProviderTicketCap: 100,
		TotalTicketCap:    2000,
		OwnerReference:    "operator:pg-pilot",
		EvidenceReference: "evidence:pg-pilot-stage1",
	})
	if err != nil {
		t.Fatalf("create bounded provider pilot fixture: %v", err)
	}
	if _, err := models.EnrollProviderPilotCompany(db, models.ProviderPilotEnrollmentInput{
		ProviderPilotEpochID: epoch.ID,
		ProviderClaimID:      neverReturned.claim.ID,
		OwnerReference:       "operator:enroll:stage1-never-returned",
		EvidenceReference:    "evidence:enroll:stage1-never-returned",
	}); !errors.Is(err, models.ErrProviderPilotEnrollmentNotEligible) {
		t.Fatalf("enroll never-returned provider error=%v, want ErrProviderPilotEnrollmentNotEligible", err)
	}
	var neverReturnedEnrollments int
	if err := db.QueryRow(`
		SELECT COUNT(*)::int FROM provider_pilot_enrollments
		WHERE provider_pilot_epoch_id=$1::uuid
		  AND provider_claim_id=$2::uuid`, epoch.ID, neverReturned.claim.ID).
		Scan(&neverReturnedEnrollments); err != nil {
		t.Fatalf("count rejected never-returned enrollment: %v", err)
	}
	if neverReturnedEnrollments != 0 {
		t.Fatalf("never-returned provider created %d enrollment rows", neverReturnedEnrollments)
	}
	if _, err := db.Exec(`
		UPDATE sites SET category='spam' WHERE id=$1::uuid`, spamBeforeEnrollment.site.ID); err != nil {
		t.Fatalf("reclassify Stage 1-returned provider before enrollment: %v", err)
	}
	if _, err := models.EnrollProviderPilotCompany(db, models.ProviderPilotEnrollmentInput{
		ProviderPilotEpochID: epoch.ID,
		ProviderClaimID:      spamBeforeEnrollment.claim.ID,
		OwnerReference:       "operator:enroll:stage1-spam-before-enroll",
		EvidenceReference:    "evidence:enroll:stage1-spam-before-enroll",
	}); !errors.Is(err, models.ErrProviderPilotEnrollmentNotEligible) {
		t.Fatalf("enroll spam-reclassified provider error=%v, want ErrProviderPilotEnrollmentNotEligible", err)
	}
	if _, err := db.Exec(`
		UPDATE sites SET category='developer' WHERE id=$1::uuid`, spamBeforeEnrollment.site.ID); err != nil {
		t.Fatalf("restore pre-enrollment spam fixture category: %v", err)
	}
	type enrolledProvider struct {
		claimID string
		suffix  string
	}
	enrolled := make([]enrolledProvider, 0, 1+len(cohortProviders))
	enroll := func(claimID string, company *models.ProviderPilotCompany, site models.Site, suffix string) {
		t.Helper()
		if company == nil || company.ProviderClaimID != claimID {
			t.Fatalf("invalid provider pilot company fixture %q", suffix)
		}
		enrollment, err := models.EnrollProviderPilotCompany(db, models.ProviderPilotEnrollmentInput{
			ProviderPilotEpochID: epoch.ID,
			ProviderClaimID:      claimID,
			OwnerReference:       "operator:enroll:" + suffix,
			EvidenceReference:    "evidence:enroll:" + suffix,
		})
		if err != nil {
			t.Fatalf("enroll provider pilot fixture %q: %v", suffix, err)
		}
		if enrollment.Stage1EligibilityStatus != "eligible" ||
			len(enrollment.Stage1EligibilitySHA256) != 64 {
			t.Fatalf("provider pilot eligibility response %q = %#v", suffix, enrollment)
		}
		encoded, err := json.Marshal(enrollment)
		if err != nil {
			t.Fatalf("marshal provider pilot enrollment %q: %v", suffix, err)
		}
		var document map[string]any
		if err := json.Unmarshal(encoded, &document); err != nil {
			t.Fatalf("decode provider pilot enrollment %q: %v", suffix, err)
		}
		for _, forbidden := range []string{
			"stage1_eligibility_site_id_snapshot",
			"stage1_eligibility_domain_sha256",
			"stage1_eligibility_bound_at",
			"search_receipt_id",
			"organic_position",
			"eligible_domain_count",
			"eligible_domains",
		} {
			if _, exposed := document[forbidden]; exposed {
				t.Fatalf("provider pilot enrollment %q exposed internal eligibility field %q: %s", suffix, forbidden, encoded)
			}
		}
		expectedDomainSHA := sha256.Sum256([]byte(site.Domain))
		var storedSiteID, storedDomainSHA, storedEligibilitySHA string
		var eligibilityCurrent bool
		if err := db.QueryRow(`
			SELECT stage1_eligibility_site_id_snapshot::text,
			       stage1_eligibility_domain_sha256,
			       stage1_eligibility_snapshot_sha256,
			       provider_pilot_enrollment_eligibility_is_current(
			           provider_pilot_epoch_id, provider_claim_id
			       )
			FROM provider_pilot_enrollments
			WHERE id=$1::uuid`, enrollment.ID).
			Scan(&storedSiteID, &storedDomainSHA, &storedEligibilitySHA, &eligibilityCurrent); err != nil {
			t.Fatalf("read provider pilot eligibility binding %q: %v", suffix, err)
		}
		if storedSiteID != site.ID ||
			storedDomainSHA != hex.EncodeToString(expectedDomainSHA[:]) ||
			storedEligibilitySHA != enrollment.Stage1EligibilitySHA256 ||
			!eligibilityCurrent {
			t.Fatalf(
				"provider pilot eligibility binding %q = site:%q domain_sha:%q snapshot:%q current:%t",
				suffix, storedSiteID, storedDomainSHA, storedEligibilitySHA, eligibilityCurrent,
			)
		}
		enrolled = append(enrolled, enrolledProvider{claimID: claimID, suffix: suffix})
	}
	enroll(primaryClaimID, primaryCompany, primarySite, "release")
	for _, provider := range cohortProviders {
		enroll(
			provider.claim.ID, provider.company, provider.site,
			strings.TrimPrefix(provider.site.Domain, "provider-"),
		)
	}
	if cleanupEligibilityReceiptID == "" || cleanupEligibilityClaimID == "" {
		t.Fatal("provider pilot retention-cleanup eligibility fixture is missing")
	}
	if _, err := db.Exec(`
		DELETE FROM search_receipts WHERE public_id=$1`, cleanupEligibilityReceiptID); err != nil {
		t.Fatalf("delete bound Stage 1 eligibility source during retention cleanup: %v", err)
	}
	var eligibilityAfterCleanup bool
	if err := db.QueryRow(`
		SELECT provider_pilot_enrollment_eligibility_is_current($1::uuid,$2::uuid)`,
		epoch.ID, cleanupEligibilityClaimID).Scan(&eligibilityAfterCleanup); err != nil {
		t.Fatalf("read enrollment eligibility after Stage 1 retention cleanup: %v", err)
	}
	if !eligibilityAfterCleanup {
		t.Fatal("Stage 1 source retention cleanup invalidated the immutable enrollment binding")
	}
	if len(cohortProviders) == 0 {
		t.Fatal("provider pilot spam-reclassification fixture has no cohort provider")
	}
	reclassified := cohortProviders[0]
	if _, err := db.Exec(`
		UPDATE sites SET category='spam' WHERE id=$1::uuid`, reclassified.site.ID); err != nil {
		t.Fatalf("reclassify enrolled pilot site as spam: %v", err)
	}
	var eligibilityCurrent bool
	if err := db.QueryRow(`
		SELECT provider_pilot_enrollment_eligibility_is_current($1::uuid,$2::uuid)`,
		epoch.ID, reclassified.claim.ID).Scan(&eligibilityCurrent); err != nil {
		t.Fatalf("read spam-reclassified enrollment eligibility: %v", err)
	}
	if eligibilityCurrent {
		t.Fatal("spam-reclassified enrollment remained current")
	}
	if _, err := models.ActivateProviderPilotEpoch(db, models.ProviderPilotEpochActionInput{
		ProviderPilotEpochID: epoch.ID,
		OwnerReference:       "operator:activate:pg-pilot:spam-reclassified",
		EvidenceReference:    "evidence:activate:pg-pilot:spam-reclassified",
	}); !errors.Is(err, models.ErrProviderPilotCohortNotReady) {
		t.Fatalf("activate pilot with spam-reclassified enrollment error=%v, want ErrProviderPilotCohortNotReady", err)
	}
	if _, err := db.Exec(`
		UPDATE sites SET category='developer' WHERE id=$1::uuid`, reclassified.site.ID); err != nil {
		t.Fatalf("restore enrolled pilot site category: %v", err)
	}
	if _, err := models.ActivateProviderPilotEpoch(db, models.ProviderPilotEpochActionInput{
		ProviderPilotEpochID: epoch.ID,
		OwnerReference:       "operator:activate:pg-pilot:before-review",
		EvidenceReference:    "evidence:activate:pg-pilot:before-review",
	}); !errors.Is(err, models.ErrProviderPilotReviewRequired) {
		t.Fatalf("activate pilot before provider reviews error = %v, want ErrProviderPilotReviewRequired", err)
	}
	if _, err := db.Exec(`
		UPDATE provider_pilot_epochs
		SET status='active', activated_at=clock_timestamp()
		WHERE id=$1::uuid`, epoch.ID); err == nil {
		t.Fatal("database activated provider pilot before current provider reviews")
	} else {
		assertPostgresConstraint(t, err, "provider_pilot_activation_provider_reviews")
	}
	var status string
	if err := db.QueryRow(`SELECT status FROM provider_pilot_epochs WHERE id=$1::uuid`, epoch.ID).Scan(&status); err != nil {
		t.Fatalf("read provider pilot after blocked activation: %v", err)
	}
	if status != "draft" {
		t.Fatalf("provider pilot status after blocked activation = %q, want draft", status)
	}
	for _, provider := range enrolled {
		recordPostgresPilotReview(
			t, db, epoch.ID, "provider", provider.claimID, "fixture-"+provider.suffix,
		)
	}
	epoch, err = models.ActivateProviderPilotEpoch(db, models.ProviderPilotEpochActionInput{
		ProviderPilotEpochID: epoch.ID,
		OwnerReference:       "operator:activate:pg-pilot",
		EvidenceReference:    "evidence:activate:pg-pilot",
	})
	if err != nil {
		t.Fatalf("activate bounded provider pilot fixture: %v", err)
	}
	if epoch.Status != "active" {
		t.Fatalf("provider pilot fixture status=%q, want active", epoch.Status)
	}
	return epoch.ID
}

func recordPostgresPilotReview(
	t *testing.T,
	db *sql.DB,
	pilotID, reviewType, subjectID, suffix string,
) *models.ProviderPilotReviewEvent {
	t.Helper()
	candidate, err := models.GetProviderPilotReviewCandidate(
		db, pilotID, reviewType, subjectID,
	)
	if err != nil {
		t.Fatalf("get %s pilot review candidate %q: %v", reviewType, suffix, err)
	}
	ownerReference := "owner:review:" + reviewType + ":" + suffix
	evidenceReference := "evidence:review:" + reviewType + ":" + suffix
	if reviewType == "provider" && strings.Contains(suffix, "http-loopback") {
		ownerReference = "owner:loopback-review:provider"
		evidenceReference = "evidence:loopback-review:provider"
	}
	event, created, err := models.RecordProviderPilotReview(
		db,
		models.ProviderPilotReviewInput{
			ProviderPilotEpochID:   pilotID,
			ReviewType:             reviewType,
			SubjectID:              subjectID,
			ExpectedSnapshotSHA256: candidate.SubjectSnapshotSHA256,
			OwnerReference:         ownerReference,
			EvidenceReference:      evidenceReference,
		},
	)
	if err != nil || !created {
		t.Fatalf("record %s pilot review %q = event:%#v created:%t err:%v", reviewType, suffix, event, created, err)
	}
	return event
}

func activePostgresPilotIDForOffer(t *testing.T, db *sql.DB, offerID string) string {
	t.Helper()
	var pilotID string
	if err := db.QueryRow(`
		SELECT epoch.id::text
		FROM provider_offers offer
		JOIN provider_pilot_enrollments enrollment
		  ON enrollment.provider_claim_id=offer.provider_claim_id
		JOIN provider_pilot_epochs epoch
		  ON epoch.id=enrollment.provider_pilot_epoch_id
		WHERE offer.id=$1::uuid AND epoch.status='active'`, offerID).Scan(&pilotID); err != nil {
		t.Fatalf("resolve active pilot for offer %s: %v", offerID, err)
	}
	return pilotID
}

func activatePostgresProviderOffer(
	t *testing.T,
	db *sql.DB,
	offerID, operatorReference, evidenceReference string,
) (*models.ProviderOffer, error) {
	t.Helper()
	pilotID := activePostgresPilotIDForOffer(t, db, offerID)
	recordPostgresPilotReview(t, db, pilotID, "offer", offerID, offerID[:8])
	return models.ActivateProviderOffer(
		db, offerID, operatorReference, evidenceReference,
	)
}

func recordPostgresTicketReview(
	t *testing.T,
	db *sql.DB,
	ticket *models.ActionTicket,
	suffix string,
) *models.ProviderPilotReviewEvent {
	t.Helper()
	if ticket == nil || ticket.ProviderPilotEpochID == "" {
		t.Fatalf("invalid pilot ticket fixture %q: %#v", suffix, ticket)
	}
	return recordPostgresPilotReview(
		t, db, ticket.ProviderPilotEpochID, "ticket", ticket.ID, suffix,
	)
}

func recordPostgresPilotAcceptance(
	t *testing.T,
	db *sql.DB,
	key *models.ProviderAPIKey,
	suffix string,
) *models.ProviderCommercialAcceptanceEvent {
	t.Helper()
	accepted, created, err := models.RecordProviderCommercialAcceptance(
		db, key, models.ProviderCommercialAcceptanceInput{
			EventType:         "pilot_company",
			IdempotencyKey:    "pg-pilot-" + suffix + "-0001",
			ProviderReference: "provider:pilot:" + suffix,
		},
	)
	if err != nil {
		t.Fatalf("record provider-authenticated pilot acceptance %q: %v", suffix, err)
	}
	if !created {
		t.Fatalf("provider-authenticated pilot acceptance %q unexpectedly replayed", suffix)
	}
	return accepted
}

func establishPostgresPilotCompany(
	t *testing.T,
	db *sql.DB,
	key *models.ProviderAPIKey,
	suffix string,
) *models.ProviderPilotCompany {
	t.Helper()
	return establishPostgresPilotCompanyWithHash(
		t, db, key, suffix, postgresPayloadHash("company:"+suffix),
	)
}

func establishPostgresPilotCompanyWithHash(
	t *testing.T,
	db *sql.DB,
	key *models.ProviderAPIKey,
	suffix, companyHash string,
) *models.ProviderPilotCompany {
	t.Helper()
	accepted := recordPostgresPilotAcceptance(t, db, key, suffix)
	company, created, err := models.VerifyProviderPilotCompany(
		db,
		accepted.ID,
		companyHash,
		"operator:"+suffix,
		"evidence:"+suffix,
	)
	if err != nil {
		t.Fatalf("owner-verify pilot company %q: %v", suffix, err)
	}
	if !created {
		t.Fatalf("owner-verified pilot company %q unexpectedly replayed", suffix)
	}
	return company
}

func createPostgresCommercialOffer(
	t *testing.T,
	db *sql.DB,
	provider *postgresCommercialProvider,
	suffix, billingMode string,
	bounty int64,
) *models.ProviderOffer {
	t.Helper()
	zero := int64(0)
	input := models.ProviderOfferInput{
		OfferName:           "Commercial proof " + suffix,
		OfferSummary:        "Exercises provider-authenticated and owner-verified commercial proof.",
		ActionType:          "lead",
		ActionURL:           provider.site.URL + "/commercial/" + suffix,
		ChargeEvent:         "accepted",
		BountyCents:         bounty,
		Currency:            "usd",
		PrincipalPriceMode:  "free",
		PrincipalPriceCents: &zero,
		PrincipalCurrency:   "usd",
		BillingMode:         billingMode,
	}
	if billingMode == "terms" {
		limit, days := 10*bounty, 30
		input.TermsCreditLimitCents = &limit
		input.TermsPeriodDays = &days
		input.TermsEvidenceReference = "legacy:operator:" + suffix
	}
	offer, err := models.CreateProviderOffer(
		db, provider.accountID, provider.claim.ID, input,
	)
	if err != nil {
		t.Fatalf("create commercial-proof %s offer %q: %v", billingMode, suffix, err)
	}
	return offer
}

func recordPostgresVerifiedFunding(
	t *testing.T,
	db *sql.DB,
	offerID string,
	amountCents int64,
	suffix, qualifyingTicketID string,
	effectiveAt time.Time,
) *models.ProviderCommercialCommitmentEvent {
	t.Helper()
	if effectiveAt.IsZero() {
		effectiveAt = postgresDatabaseClock(t, db)
	}
	event, created, err := models.RecordVerifiedProviderFunding(
		db, models.VerifiedProviderFundingInput{
			ProviderOfferID:          offerID,
			AmountCents:              amountCents,
			Currency:                 "usd",
			SourceSystem:             "pg-settlement",
			SourceEventID:            suffix,
			SourceEffectiveAt:        effectiveAt,
			QualifyingActionTicketID: qualifyingTicketID,
			OperatorReference:        "operator:" + suffix,
			OwnerEvidenceReference:   "evidence:" + suffix,
		},
	)
	if err != nil {
		t.Fatalf("record owner-verified provider funding %q: %v", suffix, err)
	}
	if !created {
		t.Fatalf("owner-verified provider funding %q unexpectedly replayed", suffix)
	}
	return event
}

func recordPostgresUnverifiedTermsAcceptance(
	t *testing.T,
	db *sql.DB,
	key *models.ProviderAPIKey,
	offerID, suffix string,
) *models.ProviderCommercialAcceptanceEvent {
	t.Helper()
	var version int
	var exactTermsSHA256 string
	if err := db.QueryRow(`
		SELECT version, commercial_terms_sha256
		FROM provider_offers WHERE id=$1::uuid`, offerID).
		Scan(&version, &exactTermsSHA256); err != nil {
		t.Fatalf("read unverified exact terms %q: %v", suffix, err)
	}
	accepted, created, err := models.RecordProviderCommercialAcceptance(
		db, key, models.ProviderCommercialAcceptanceInput{
			EventType:                "terms_acceptance",
			ProviderOfferID:          offerID,
			ExpectedOfferVersion:     version,
			ExpectedExactTermsSHA256: exactTermsSHA256,
			IdempotencyKey:           "pg-unverified-terms-" + suffix + "-0001",
			ProviderReference:        "provider:unverified-terms:" + suffix,
		},
	)
	if err != nil {
		t.Fatalf("record unverified terms acceptance %q: %v", suffix, err)
	}
	if !created {
		t.Fatalf("unverified terms acceptance %q unexpectedly replayed", suffix)
	}
	return accepted
}

func recordPostgresVerifiedTerms(
	t *testing.T,
	db *sql.DB,
	key *models.ProviderAPIKey,
	offerID, suffix, relatedAcceptanceID, relatedCommitmentID string,
) (*models.ProviderCommercialAcceptanceEvent, *models.ProviderCommercialCommitmentEvent) {
	t.Helper()
	eventType := "terms_acceptance"
	if relatedAcceptanceID != "" {
		eventType = "terms_renewal"
	}
	var expectedOfferVersion int
	var expectedExactTermsSHA256 string
	if err := db.QueryRow(`
		SELECT version, commercial_terms_sha256
		FROM provider_offers WHERE id=$1::uuid`, offerID).
		Scan(&expectedOfferVersion, &expectedExactTermsSHA256); err != nil {
		t.Fatalf("read exact provider terms %q: %v", suffix, err)
	}
	accepted, created, err := models.RecordProviderCommercialAcceptance(
		db, key, models.ProviderCommercialAcceptanceInput{
			EventType:                eventType,
			ProviderOfferID:          offerID,
			RelatedAcceptanceEventID: relatedAcceptanceID,
			ExpectedOfferVersion:     expectedOfferVersion,
			ExpectedExactTermsSHA256: expectedExactTermsSHA256,
			IdempotencyKey:           "pg-terms-" + suffix + "-0001",
			ProviderReference:        "provider:terms:" + suffix,
		},
	)
	if err != nil {
		t.Fatalf("record exact provider terms acceptance %q: %v", suffix, err)
	}
	if !created {
		t.Fatalf("provider terms acceptance %q unexpectedly replayed", suffix)
	}
	commitment, commitmentCreated, err := models.RecordVerifiedProviderTerms(
		db, models.VerifiedProviderTermsInput{
			ProviderOfferID:           offerID,
			ProviderAcceptanceEventID: accepted.ID,
			RelatedCommitmentEventID:  relatedCommitmentID,
			SourceSystem:              "pg-contract",
			SourceEventID:             suffix,
			SourceEffectiveAt:         postgresDatabaseClock(t, db),
			OperatorReference:         "operator:" + suffix,
			OwnerEvidenceReference:    "evidence:" + suffix,
		},
	)
	if err != nil {
		t.Fatalf("owner-verify exact provider terms %q: %v", suffix, err)
	}
	if !commitmentCreated {
		t.Fatalf("owner-verified provider terms %q unexpectedly replayed", suffix)
	}
	return accepted, commitment
}

func postgresDatabaseClock(t *testing.T, db *sql.DB) time.Time {
	t.Helper()
	var now time.Time
	if err := db.QueryRow(`SELECT clock_timestamp()`).Scan(&now); err != nil {
		t.Fatalf("read PostgreSQL commercial-proof clock: %v", err)
	}
	return now
}

func assertPostgresConstraint(t *testing.T, err error, want string) {
	t.Helper()
	assertPostgresConstraintCode(t, err, "23514", want)
}

func assertPostgresConstraintCode(t *testing.T, err error, code, want string) {
	t.Helper()
	var pqErr *pq.Error
	if !errors.As(err, &pqErr) || string(pqErr.Code) != code || pqErr.Constraint != want {
		t.Fatalf("PostgreSQL error = %#v, want %s %s", err, code, want)
	}
}

func createPostgresProviderFixture(t *testing.T, db *sql.DB) (int64, models.Site) {
	t.Helper()
	var accountID int64
	if err := db.QueryRow(`
		INSERT INTO accounts (email, plan, status, monthly_limit)
		VALUES ('pg-release@example.invalid','provider-pilot','active',0)
		RETURNING id`).Scan(&accountID); err != nil {
		t.Fatalf("insert test account: %v", err)
	}
	site := models.Site{Domain: "provider.example", URL: "https://provider.example", AgenticScore: 90}
	if err := db.QueryRow(`
		INSERT INTO sites (
			domain, url, name, description, has_structured_api,
			agentic_score, category, crawl_status
		) VALUES ($1,$2,'Provider Example','PostgreSQL release fixture',true,$3,'developer','success')
		RETURNING id::text`, site.Domain, site.URL, site.AgenticScore).Scan(&site.ID); err != nil {
		t.Fatalf("insert test site: %v", err)
	}
	return accountID, site
}

func createPostgresActionTicket(
	t *testing.T,
	db *sql.DB,
	signer *providerexchange.Signer,
	site models.Site,
	offerID, suffix string,
) *models.ActionTicket {
	t.Helper()
	ticket, rawToken := createPostgresUnhandedActionTicket(
		t, db, signer, site, offerID, suffix,
	)
	recordPostgresTicketReview(t, db, ticket, suffix)
	ticket, handoff, err := models.RecordActionTicketHandoff(db, models.ProviderActionHandoffInput{
		ActionTicketID:          ticket.ID,
		AttributionToken:        rawToken,
		PrincipalHandoffConsent: true,
		HandoffConsentVersion:   models.ProviderActionHandoffConsentV1,
	})
	if err != nil {
		t.Fatalf("record action handoff %q: %v", suffix, err)
	}
	if handoff == nil || handoff.Replayed || ticket.Status != "redirected" {
		t.Fatalf("new action handoff %q = ticket:%#v receipt:%#v", suffix, ticket, handoff)
	}
	return ticket
}

func createPostgresUnhandedActionTicket(
	t *testing.T,
	db *sql.DB,
	signer *providerexchange.Signer,
	site models.Site,
	offerID, suffix string,
) (*models.ActionTicket, string) {
	t.Helper()
	return createPostgresUnhandedActionTicketWithTTL(t, db, signer, site, offerID, suffix, time.Hour)
}

func createPostgresUnhandedActionTicketWithTTL(
	t *testing.T,
	db *sql.DB,
	signer *providerexchange.Signer,
	site models.Site,
	offerID, suffix string,
	ttl time.Duration,
) (*models.ActionTicket, string) {
	t.Helper()
	return createPostgresActionTicketFixture(t, db, signer, site, offerID, suffix, models.ActionTicketInput{
		DemandTopic:      "developer-tools",
		PrincipalConsent: true,
		ConsentVersion:   models.ProviderPrincipalConsentV1,
		TTL:              ttl,
	})
}

// createPostgresActionTicketFixture uses the production writer for supported
// TTLs. For boundary-race tests only, it can insert a correctly signed ticket
// below the production one-hour minimum and its mandatory capacity reservation
// in the same transaction. The production validator remains unchanged.
func createPostgresActionTicketFixture(
	t *testing.T,
	db *sql.DB,
	signer *providerexchange.Signer,
	site models.Site,
	offerID, suffix string,
	input models.ActionTicketInput,
) (*models.ActionTicket, string) {
	t.Helper()
	searchID := recordPostgresDemandReceipt(t, db, site, suffix)
	recordPostgresReturnedOffer(t, db, site, searchID, offerID)
	input.ProviderOfferID = offerID
	input.SearchReceiptPublicID = searchID
	if input.BudgetBand == "" {
		input.BudgetBand = "unspecified"
	}
	if input.Urgency == "" {
		input.Urgency = "unspecified"
	}
	if input.RequirementFlags == nil {
		input.RequirementFlags = []string{}
	}
	if input.TTL >= time.Hour {
		ticket, _, rawToken, err := models.CreateActionTicket(db, input, signer)
		if err != nil {
			t.Fatalf("create action ticket %q: %v", suffix, err)
		}
		return ticket, rawToken
	}
	if input.TTL <= 0 || signer == nil {
		t.Fatalf("invalid short-lived action ticket fixture %q", suffix)
	}

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin short-lived action ticket %q: %v", suffix, err)
	}
	defer func() { _ = tx.Rollback() }()
	var ticketID string
	var issuedAt time.Time
	if err := tx.QueryRow(`
		SELECT uuid_generate_v4()::text,
		       date_trunc('second', clock_timestamp())`).Scan(&ticketID, &issuedAt); err != nil {
		t.Fatalf("allocate short-lived action ticket %q: %v", suffix, err)
	}
	expiresAt := issuedAt.Add(input.TTL.Truncate(time.Second))
	claims, err := providerexchange.NewAttributionClaimsForKey(
		signer.ActiveKeyID(), ticketID, offerID, issuedAt, expiresAt,
	)
	if err != nil {
		t.Fatalf("build short-lived attribution claims %q: %v", suffix, err)
	}
	rawToken, err := signer.SignAttribution(claims)
	if err != nil {
		t.Fatalf("sign short-lived action ticket %q: %v", suffix, err)
	}
	requestHash := postgresPayloadHash("short-lived-ticket:" + suffix + ":" + ticketID)
	ticket := &models.ActionTicket{}
	var requirementFlags pq.StringArray
	if err := tx.QueryRow(`
		INSERT INTO action_tickets (
			id, provider_claim_id, provider_offer_id, search_receipt_id,
			provider_pilot_epoch_id,
			source_is_synthetic, token_hash, token_nonce, creation_request_hash,
			offer_version_snapshot, offer_name_snapshot, offer_summary_snapshot,
			action_type_snapshot, action_url_snapshot, disclosure_snapshot,
			charge_event_snapshot, bounty_cents_snapshot, currency_snapshot,
			billing_mode_snapshot, terms_evidence_reference_snapshot,
			commercial_terms_contract_version_snapshot,
			commercial_terms_sha256_snapshot,
			terms_credit_limit_cents_snapshot, terms_period_days_snapshot,
			terms_period_anchor_at_snapshot, attribution_key_id_snapshot,
			principal_price_mode_snapshot, principal_price_cents_snapshot,
			principal_currency_snapshot, demand_topic, region_code, budget_band,
			urgency, requirement_flags, principal_consent, consent_version,
			expires_at, created_at, updated_at
		)
		SELECT $1::uuid, offer.provider_claim_id, offer.id, receipt.id,
		       offer.provider_pilot_epoch_id,
		       receipt.is_synthetic, $4, $5, $6,
		       offer.version, offer.offer_name, offer.offer_summary,
		       offer.action_type, offer.action_url, offer.disclosure_label,
		       offer.charge_event, offer.bounty_cents, offer.currency,
		       offer.billing_mode, offer.terms_evidence_reference,
		       offer.commercial_terms_contract_version,
		       offer.commercial_terms_sha256,
		       offer.terms_credit_limit_cents, offer.terms_period_days,
		       offer.terms_period_anchor_at, $7,
		       offer.principal_price_mode, offer.principal_price_cents,
		       offer.principal_currency, $8, $9, $10, $11, $12,
		       true, $13, $14, $15, $15
		FROM provider_offers offer
		JOIN search_receipts receipt ON receipt.public_id=$3
		WHERE offer.id=$2::uuid AND offer.status='active'
		RETURNING id::text, provider_claim_id::text, provider_offer_id::text,
		          provider_pilot_epoch_id::text,
		          search_receipt_id::text, source_is_synthetic, token_hash,
		          token_nonce, creation_request_hash, offer_version_snapshot,
		          action_type_snapshot,
		          commercial_terms_contract_version_snapshot,
		          commercial_terms_sha256_snapshot, bounty_cents_snapshot,
		          currency_snapshot, attribution_key_id_snapshot,
		          demand_topic, region_code, budget_band, urgency,
		          requirement_flags, principal_consent, consent_version,
		          status, expires_at, created_at, updated_at`,
		ticketID, offerID, searchID, models.HashProviderSecret(rawToken),
		claims.Nonce, requestHash, signer.ActiveKeyID(), input.DemandTopic,
		input.RegionCode, input.BudgetBand, input.Urgency,
		pq.Array(input.RequirementFlags), input.ConsentVersion,
		expiresAt, issuedAt,
	).Scan(
		&ticket.ID, &ticket.ProviderClaimID, &ticket.ProviderOfferID,
		&ticket.ProviderPilotEpochID,
		&ticket.SearchReceiptID, &ticket.SourceIsSynthetic, &ticket.TokenHash,
		&ticket.TokenNonce, &ticket.CreationRequestHash,
		&ticket.OfferVersionSnapshot, &ticket.ActionTypeSnapshot,
		&ticket.CommercialTermsContractVersionSnapshot,
		&ticket.CommercialTermsSHA256Snapshot, &ticket.BountyCentsSnapshot,
		&ticket.CurrencySnapshot, &ticket.AttributionKeyIDSnapshot,
		&ticket.DemandTopic, &ticket.RegionCode, &ticket.BudgetBand,
		&ticket.Urgency, &requirementFlags, &ticket.PrincipalConsent,
		&ticket.ConsentVersion, &ticket.Status, &ticket.ExpiresAt,
		&ticket.CreatedAt, &ticket.UpdatedAt,
	); err != nil {
		t.Fatalf("insert short-lived action ticket %q: %v", suffix, err)
	}
	ticket.RequirementFlags = []string(requirementFlags)
	if _, err := tx.Exec(`
		INSERT INTO provider_capacity_events (
			provider_claim_id, provider_offer_id, action_ticket_id,
			event_type, event_reason, amount_cents, currency, created_at
		) VALUES ($1::uuid,$2::uuid,$3::uuid,'reserve','ticket_created',$4,$5,$6)`,
		ticket.ProviderClaimID, ticket.ProviderOfferID, ticket.ID,
		ticket.BountyCentsSnapshot, ticket.CurrencySnapshot, issuedAt,
	); err != nil {
		t.Fatalf("reserve short-lived action ticket capacity %q: %v", suffix, err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit short-lived action ticket %q: %v", suffix, err)
	}
	return ticket, rawToken
}

func recordPostgresReturnedOffer(t *testing.T, db *sql.DB, site models.Site, searchID, offerID string) {
	t.Helper()
	offers, err := models.ListPublicProviderOffersForOrganicResults(db, searchID, []models.Site{site})
	if err != nil {
		t.Fatalf("list public provider offers for %q: %v", searchID, err)
	}
	for _, offer := range offers {
		if offer.OfferID != offerID {
			continue
		}
		if err := models.RecordProviderOffersReturned(db, searchID, []models.PublicProviderOffer{offer}); err != nil {
			t.Fatalf("record returned offer %q for %q: %v", offerID, searchID, err)
		}
		return
	}
	t.Fatalf("active offer %q was not eligible beside organic result %q", offerID, searchID)
}

func recordPostgresDemandReceipt(t *testing.T, db *sql.DB, site models.Site, suffix string) string {
	t.Helper()
	searchID := "nhs_sr_pg_release_" + suffix
	err := models.RecordDemandSearch(db, models.DemandSearchReceipt{
		PublicID: searchID, Surface: "rest", Category: "developer",
		ResultCount: 1, Page: 1, PageSize: 10,
	}, []models.Site{site})
	if err != nil {
		t.Fatalf("record demand receipt %q: %v", suffix, err)
	}
	return searchID
}

func recordPostgresStage1EligibilityReceipt(
	t *testing.T,
	db *sql.DB,
	site models.Site,
	label string,
) string {
	t.Helper()
	searchID, err := models.GenerateDemandSearchID()
	if err != nil {
		t.Fatalf("generate Stage 1 eligibility receipt %q: %v", label, err)
	}
	if err := models.RecordDemandSearch(db, models.DemandSearchReceipt{
		PublicID: searchID,
		Surface:  "rest",
		Category: "developer",
		Page:     1,
		PageSize: 10,
		// This total describes the actual one-row organic response; the
		// enrollment gate still joins the exact returned site and snapshot.
		ResultCount: 1,
	}, []models.Site{site}); err != nil {
		t.Fatalf("record Stage 1 eligibility receipt %q: %v", label, err)
	}
	return searchID
}

func recordPostgresOutcome(
	t *testing.T,
	db *sql.DB,
	signer *providerexchange.Signer,
	key *models.ProviderAPIKey,
	ticketID, outcome, suffix string,
) *models.OutcomeReceipt {
	t.Helper()
	receipt, created, err := models.RecordProviderOutcome(db, key, models.ProviderOutcomeInput{
		ActionTicketID: ticketID,
		IdempotencyKey: fmt.Sprintf("pg-%s-0001", suffix),
		PayloadHash:    postgresPayloadHash(suffix),
		Outcome:        outcome,
	}, signer)
	if err != nil {
		t.Fatalf("record %s outcome %q: %v", outcome, suffix, err)
	}
	if !created {
		t.Fatalf("record %s outcome %q unexpectedly replayed", outcome, suffix)
	}
	return receipt
}

func assertPostgresProviderBalance(t *testing.T, db *sql.DB, offerID string, want int64) {
	t.Helper()
	var got int64
	if err := db.QueryRow(`
		SELECT COALESCE(SUM(amount_cents),0)::bigint
		FROM provider_budget_ledger WHERE provider_offer_id=$1::uuid`, offerID).Scan(&got); err != nil {
		t.Fatalf("read provider balance: %v", err)
	}
	if got != want {
		t.Fatalf("provider balance = %d, want %d", got, want)
	}
	if got > models.ProviderMoneyMaximumCents {
		t.Fatalf("provider balance %d exceeds hard cap %d", got, models.ProviderMoneyMaximumCents)
	}
}

func assertPostgresOfferTicketCount(t *testing.T, db *sql.DB, offerID string, want int) {
	t.Helper()
	var got int
	if err := db.QueryRow(
		`SELECT COUNT(*)::int FROM action_tickets WHERE provider_offer_id=$1::uuid`,
		offerID,
	).Scan(&got); err != nil {
		t.Fatalf("count offer tickets: %v", err)
	}
	if got != want {
		t.Fatalf("offer %s ticket count = %d, want %d", offerID, got, want)
	}
}

func postgresPayloadHash(label string) string {
	sum := sha256.Sum256([]byte(label))
	return hex.EncodeToString(sum[:])
}

// postgresCommercialTermsHash mirrors the canonical hash only for the legacy
// Unicode-row regression above. That fixture must repair an imported draft's
// hash before migration 022 will allow either party to commit evidence.
func postgresCommercialTermsHash(
	offerID string,
	version int,
	input models.ProviderOfferInput,
) string {
	principalPrice := "null"
	if input.PrincipalPriceCents != nil {
		principalPrice = strconv.FormatInt(*input.PrincipalPriceCents, 10)
	}
	termsLimit := "null"
	if input.TermsCreditLimitCents != nil {
		termsLimit = strconv.FormatInt(*input.TermsCreditLimitCents, 10)
	}
	termsDays := "null"
	if input.TermsPeriodDays != nil {
		termsDays = strconv.Itoa(*input.TermsPeriodDays)
	}
	controlled := []string{
		models.ProviderCommercialTermsContractV1,
		strings.ToLower(strings.TrimSpace(offerID)),
		strconv.Itoa(version),
		strings.TrimSpace(input.OfferName),
		strings.TrimSpace(input.OfferSummary),
		strings.ToLower(strings.TrimSpace(input.ActionType)),
		strings.TrimSpace(input.ActionURL),
		models.ProviderDisclosureLabel,
		strings.ToLower(strings.TrimSpace(input.ChargeEvent)),
		strconv.FormatInt(input.BountyCents, 10),
		strings.ToLower(strings.TrimSpace(input.Currency)),
		strings.ToLower(strings.TrimSpace(input.PrincipalPriceMode)),
		principalPrice,
		strings.ToLower(strings.TrimSpace(input.PrincipalCurrency)),
		strings.ToLower(strings.TrimSpace(input.BillingMode)),
		termsLimit,
		termsDays,
		models.ProviderCommercialCreditRuleV1,
		models.ProviderCommercialResponseRuleV1,
		models.ProviderCommercialTermsAnchorRuleV1,
		"provider_acknowledges_merchant_of_record=true",
	}
	sum := sha256.Sum256([]byte(strings.Join(controlled, "\x00")))
	return hex.EncodeToString(sum[:])
}

func repeatPostgresRune(value rune, count int) string {
	runes := make([]rune, count)
	for i := range runes {
		runes[i] = value
	}
	return string(runes)
}
