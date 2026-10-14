package handlers

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/unitedideas/nothumansearch/internal/models"
	"github.com/unitedideas/nothumansearch/internal/providerexchange"
)

const (
	providerManifestTestPilotID    = "6c79ca8e-d61d-47e2-91dd-fecd9f711234"
	providerManifestTestManifestID = "7d89ca8e-d61d-47e2-91dd-fecd9f711234"
)

type providerManifestTestConnector struct {
	proofTablesExist    bool
	manifestTableExists bool
	pilotID             string
	existingManifestRow []driver.Value
	signingProofRows    [][]driver.Value
	queryCount          int
}

func (c *providerManifestTestConnector) Connect(context.Context) (driver.Conn, error) {
	return &providerManifestTestConn{connector: c}, nil
}

func (*providerManifestTestConnector) Driver() driver.Driver {
	return providerManifestTestDriver{}
}

type providerManifestTestDriver struct{}

func (providerManifestTestDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("provider manifest test driver requires OpenDB")
}

type providerManifestTestConn struct {
	connector *providerManifestTestConnector
}

func (*providerManifestTestConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("provider manifest test driver does not prepare statements")
}

func (*providerManifestTestConn) Close() error { return nil }

func (c *providerManifestTestConn) Begin() (driver.Tx, error) {
	return &providerManifestTestTx{}, nil
}

func (c *providerManifestTestConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	return &providerManifestTestTx{}, nil
}

func (c *providerManifestTestConn) QueryContext(
	_ context.Context,
	query string,
	_ []driver.NamedValue,
) (driver.Rows, error) {
	c.connector.queryCount++
	switch {
	case strings.Contains(query, "to_regclass('public.action_tickets')"):
		return &providerManifestTestRows{
			columns: []string{"proof_tables_exist"},
			values:  [][]driver.Value{{c.connector.proofTablesExist}},
		}, nil
	case strings.Contains(query, "to_regclass('public.provider_commercial_proof_manifests')"):
		return &providerManifestTestRows{
			columns: []string{"manifest_table_exists"},
			values:  [][]driver.Value{{c.connector.manifestTableExists}},
		}, nil
	case strings.Contains(query, "SELECT DISTINCT ON (key_id, proof_kind)"):
		return &providerManifestTestRows{
			columns: []string{
				"key_id", "proof_kind", "ticket_id", "offer_id",
				"issued_at", "expires_at", "token_nonce", "token_hash",
				"signed_receipt", "signature",
			},
			values: c.connector.signingProofRows,
		}, nil
	case strings.Contains(query, "SELECT id::text FROM provider_pilot_epochs"):
		values := [][]driver.Value{}
		if c.connector.pilotID != "" {
			values = append(values, []driver.Value{c.connector.pilotID})
		}
		return &providerManifestTestRows{columns: []string{"id"}, values: values}, nil
	case strings.Contains(query, "FROM provider_commercial_proof_manifests"):
		values := [][]driver.Value{}
		if c.connector.existingManifestRow != nil {
			values = append(values, c.connector.existingManifestRow)
		}
		return &providerManifestTestRows{
			columns: []string{
				"id", "provider_pilot_epoch_id", "manifest_contract_version",
				"proof_snapshot_sha256", "review_evidence_sha256",
				"key_id", "signed_manifest",
				"signature", "payload_sha256", "owner_reference",
				"evidence_reference", "issued_at",
			},
			values: values,
		}, nil
	default:
		return nil, errors.New("unexpected provider manifest test query")
	}
}

type providerManifestTestTx struct{}

func (*providerManifestTestTx) Commit() error   { return nil }
func (*providerManifestTestTx) Rollback() error { return nil }

type providerManifestTestRows struct {
	columns []string
	values  [][]driver.Value
	index   int
}

func (r *providerManifestTestRows) Columns() []string { return r.columns }
func (*providerManifestTestRows) Close() error        { return nil }
func (r *providerManifestTestRows) Next(dest []driver.Value) error {
	if r.index >= len(r.values) {
		return io.EOF
	}
	copy(dest, r.values[r.index])
	r.index++
	return nil
}

func openProviderManifestTestDB(t *testing.T, connector *providerManifestTestConnector) *sql.DB {
	t.Helper()
	db := sql.OpenDB(connector)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func providerManifestTestSigningFixture(
	t *testing.T,
	keyID, secret string,
) (*providerexchange.Signer, string, string, time.Time) {
	t.Helper()
	signer, err := providerexchange.NewSignerKeyring(keyID, secret, nil)
	if err != nil {
		t.Fatal(err)
	}
	issuedAt := time.Unix(1_700_000_000, 0).UTC()
	manifest := providerexchange.CommercialProofManifest{
		Version:                          providerexchange.CommercialProofManifestVersion,
		KeyID:                            keyID,
		SignatureVerificationScope:       providerexchange.CommercialProofVerificationScopeV1,
		ManifestContractVersion:          providerexchange.CommercialProofManifestContractV1,
		ManifestID:                       providerManifestTestManifestID,
		ProviderPilotEpochID:             providerManifestTestPilotID,
		ProviderPilotContractVersion:     "nhs-provider-pilot-v1",
		ReviewContractVersion:            "nhs-provider-pilot-review-v1",
		ReviewEvidenceContractVersion:    providerexchange.CommercialProofReviewEvidenceV1,
		MarketPolicyContractVersion:      providerexchange.CommercialProofMarketPolicyV1,
		ProofSnapshotSHA256:              strings.Repeat("a", 64),
		ReviewEvidenceSHA256:             strings.Repeat("b", 64),
		PilotDemandTopic:                 "developer-tools",
		PilotStatus:                      "closed",
		IssuedAt:                         issuedAt.Unix(),
		OutcomeReceiptIntegrityValid:     true,
		ReviewIntegrityValid:             true,
		VerifiedOutcomeReceipts:          5,
		VerifiedOutcomeLedgerEntries:     5,
		VerifiedProviderCompanies:        3,
		VerifiedProviderAcceptedHandoffs: 5,
		VerifiedProviderActivations:      2,
		VerifiedProviderRenewals:         1,
		VerifiedProviderConversions:      2,
		ReviewCoverage: providerexchange.CommercialProofReviewCoverage{
			Providers: providerexchange.CommercialProofReviewCount{Required: 3, Valid: 3},
			Offers:    providerexchange.CommercialProofReviewCount{Required: 3, Valid: 3},
			Tickets:   providerexchange.CommercialProofReviewCount{Required: 5, Valid: 5},
			Handoffs:  providerexchange.CommercialProofReviewCount{Required: 5, Valid: 5},
			Callbacks: providerexchange.CommercialProofReviewCount{Required: 5, Valid: 5},
		},
		MonetaryAmountsWithheldForPrivacy: true,
		VerifiedPrepaidSettled:            []providerexchange.CommercialProofCurrencyAmount{},
		VerifiedPrepaidNetDebited:         []providerexchange.CommercialProofCurrencyAmount{},
		VerifiedTermsNetReceivable:        []providerexchange.CommercialProofCurrencyAmount{},
		PilotThresholdsMet:                true,
		OrganicRankSold:                   false,
		RawQueriesSold:                    false,
		AgentIdentitiesSold:               false,
		EvidenceScope:                     providerexchange.CommercialProofManifestScopeV1,
	}
	canonical, signature, err := signer.SignCommercialProofManifest(manifest)
	if err != nil {
		t.Fatal(err)
	}
	return signer, canonical, signature, issuedAt
}

func providerManifestExistingRow(
	canonical, signature, keyID string,
	issuedAt time.Time,
) []driver.Value {
	payloadDigest := sha256.Sum256([]byte(canonical))
	return []driver.Value{
		providerManifestTestManifestID,
		providerManifestTestPilotID,
		providerexchange.CommercialProofManifestContractV1,
		strings.Repeat("a", 64),
		strings.Repeat("b", 64),
		keyID,
		canonical,
		signature,
		hex.EncodeToString(payloadDigest[:]),
		"owner:proof-manifest",
		"evidence:proof-manifest",
		issuedAt,
	}
}

func providerManifestAdminRequest(method, target, body string) *http.Request {
	request := httptest.NewRequest(method, target, bytes.NewBufferString(body))
	request.Header.Set("Authorization", "Bearer test-admin-key")
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	return request
}

func TestAdminProviderProofManifestAuthenticatesBeforeDatabaseAccess(t *testing.T) {
	t.Setenv("ADMIN_API_KEY", "test-admin-key")
	connector := &providerManifestTestConnector{}
	handler := &ProviderExchangeHandler{DB: openProviderManifestTestDB(t, connector)}
	requests := []*http.Request{
		httptest.NewRequest(
			http.MethodGet,
			"/api/v1/admin/provider-proof-manifest?pilot_id="+providerManifestTestPilotID,
			nil,
		),
		httptest.NewRequest(
			http.MethodPost,
			"/api/v1/admin/provider-proof-manifest",
			strings.NewReader(`{"provider_pilot_epoch_id":"`+providerManifestTestPilotID+`"}`),
		),
	}
	for _, request := range requests {
		recorder := httptest.NewRecorder()
		handler.AdminProviderProofManifest(recorder, request)
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("unauthorized %s proof-manifest status=%d body=%s",
				request.Method, recorder.Code, recorder.Body.String())
		}
	}
	if connector.queryCount != 0 {
		t.Fatalf("unauthorized proof-manifest request performed %d database queries", connector.queryCount)
	}
}

func TestAdminProviderProofManifestValidatesMethodAndParameters(t *testing.T) {
	t.Setenv("ADMIN_API_KEY", "test-admin-key")
	for _, target := range []string{
		"/api/v1/admin/provider-proof-manifest",
		"/api/v1/admin/provider-proof-manifest?pilot_id=not-a-uuid",
	} {
		recorder := httptest.NewRecorder()
		(&ProviderExchangeHandler{}).AdminProviderProofManifest(
			recorder,
			providerManifestAdminRequest(http.MethodGet, target, ""),
		)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("invalid proof-manifest GET status=%d body=%s", recorder.Code, recorder.Body.String())
		}
	}

	for _, body := range []string{
		`{`,
		`{"provider_pilot_epoch_id":"` + providerManifestTestPilotID + `","expected_snapshot_sha256":"` + strings.Repeat("a", 64) + `","owner_reference":"owner:proof-manifest","evidence_reference":"evidence:proof-manifest","query":"forbidden"}`,
		`{"provider_pilot_epoch_id":"` + providerManifestTestPilotID + `","owner_reference":"owner:proof-manifest","evidence_reference":"evidence:proof-manifest"}`,
	} {
		recorder := httptest.NewRecorder()
		(&ProviderExchangeHandler{}).AdminProviderProofManifest(
			recorder,
			providerManifestAdminRequest(http.MethodPost, "/api/v1/admin/provider-proof-manifest", body),
		)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("invalid proof-manifest POST status=%d body=%s", recorder.Code, recorder.Body.String())
		}
	}

	recorder := httptest.NewRecorder()
	(&ProviderExchangeHandler{}).AdminProviderProofManifest(
		recorder,
		providerManifestAdminRequest(http.MethodDelete, "/api/v1/admin/provider-proof-manifest", ""),
	)
	if recorder.Code != http.StatusMethodNotAllowed || recorder.Header().Get("Allow") != "GET, POST" {
		t.Fatalf("proof-manifest method status=%d allow=%q body=%s",
			recorder.Code, recorder.Header().Get("Allow"), recorder.Body.String())
	}
}

func TestAdminProviderProofManifestGETExistingAndPOSTReplayShapes(t *testing.T) {
	t.Setenv("ADMIN_API_KEY", "test-admin-key")
	keyID := "nhs-provider-current"
	signer, canonical, signature, issuedAt := providerManifestTestSigningFixture(
		t, keyID, strings.Repeat("s", 32),
	)
	connector := &providerManifestTestConnector{
		pilotID:             providerManifestTestPilotID,
		existingManifestRow: providerManifestExistingRow(canonical, signature, keyID, issuedAt),
	}
	handler := &ProviderExchangeHandler{
		DB:     openProviderManifestTestDB(t, connector),
		Signer: signer,
	}

	getRecorder := httptest.NewRecorder()
	handler.AdminProviderProofManifest(
		getRecorder,
		providerManifestAdminRequest(
			http.MethodGet,
			"/api/v1/admin/provider-proof-manifest?pilot_id="+providerManifestTestPilotID,
			"",
		),
	)
	if getRecorder.Code != http.StatusOK {
		t.Fatalf("existing proof-manifest GET status=%d body=%s", getRecorder.Code, getRecorder.Body.String())
	}
	var getBody map[string]any
	if err := json.Unmarshal(getRecorder.Body.Bytes(), &getBody); err != nil {
		t.Fatal(err)
	}
	if getBody["issued"] != true || getBody["commercial_proof_created"] != true ||
		getBody["publicly_released"] != false ||
		getBody["independently_verifiable"] != false || getBody["manifest"] == nil {
		t.Fatalf("existing proof-manifest GET body=%#v", getBody)
	}
	if strings.Contains(getRecorder.Body.String(), "owner_reference") ||
		strings.Contains(getRecorder.Body.String(), "evidence_reference") {
		t.Fatalf("existing proof-manifest exposed internal issuance references: %s", getRecorder.Body.String())
	}

	postBody := `{"provider_pilot_epoch_id":"` + providerManifestTestPilotID +
		`","expected_snapshot_sha256":"` + strings.Repeat("a", 64) +
		`","owner_reference":"owner:proof-manifest","evidence_reference":"evidence:proof-manifest"}`
	postRecorder := httptest.NewRecorder()
	handler.AdminProviderProofManifest(
		postRecorder,
		providerManifestAdminRequest(http.MethodPost, "/api/v1/admin/provider-proof-manifest", postBody),
	)
	if postRecorder.Code != http.StatusOK {
		t.Fatalf("proof-manifest POST replay status=%d body=%s", postRecorder.Code, postRecorder.Body.String())
	}
	var postResponse map[string]any
	if err := json.Unmarshal(postRecorder.Body.Bytes(), &postResponse); err != nil {
		t.Fatal(err)
	}
	if postResponse["created"] != false || postResponse["idempotent_replay"] != true ||
		postResponse["commercial_proof_created"] != true ||
		postResponse["publicly_released"] != false ||
		postResponse["independently_verifiable"] != false || postResponse["manifest"] == nil {
		t.Fatalf("proof-manifest POST replay body=%#v", postResponse)
	}

	conflictBody := strings.Replace(postBody, "owner:proof-manifest", "owner:proof-conflict", 1)
	conflictRecorder := httptest.NewRecorder()
	handler.AdminProviderProofManifest(
		conflictRecorder,
		providerManifestAdminRequest(http.MethodPost, "/api/v1/admin/provider-proof-manifest", conflictBody),
	)
	if conflictRecorder.Code != http.StatusConflict ||
		!strings.Contains(conflictRecorder.Body.String(), "different issuance evidence") {
		t.Fatalf("proof-manifest conflict status=%d body=%s", conflictRecorder.Code, conflictRecorder.Body.String())
	}
}

func TestAdminProviderProofManifestGETFailsClosedOnStoredIntegrityError(t *testing.T) {
	t.Setenv("ADMIN_API_KEY", "test-admin-key")
	keyID := "nhs-provider-current"
	signer, canonical, signature, issuedAt := providerManifestTestSigningFixture(
		t, keyID, strings.Repeat("s", 32),
	)
	tampered := strings.Replace(canonical, `"verified_provider_renewals":1`, `"verified_provider_renewals":2`, 1)
	if tampered == canonical {
		t.Fatal("proof-manifest integrity fixture did not change canonical payload")
	}
	handler := &ProviderExchangeHandler{
		DB: openProviderManifestTestDB(t, &providerManifestTestConnector{
			existingManifestRow: providerManifestExistingRow(
				tampered, signature, keyID, issuedAt,
			),
		}),
		Signer: signer,
	}
	recorder := httptest.NewRecorder()
	handler.AdminProviderProofManifest(
		recorder,
		providerManifestAdminRequest(
			http.MethodGet,
			"/api/v1/admin/provider-proof-manifest?pilot_id="+providerManifestTestPilotID,
			"",
		),
	)
	if recorder.Code != http.StatusInternalServerError ||
		!strings.Contains(recorder.Body.String(), "failed integrity verification") ||
		strings.Contains(recorder.Body.String(), canonical) ||
		strings.Contains(recorder.Body.String(), signature) {
		t.Fatalf("tampered proof-manifest GET status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestProviderProofManifestStatusMappingsAndPreviewContract(t *testing.T) {
	for _, test := range []struct {
		err     error
		status  int
		message string
	}{
		{models.ErrProviderProofManifestSnapshotChanged, http.StatusConflict, "fetch and review the current candidate"},
		{models.ErrProviderProofManifestNotIssuable, http.StatusConflict, "closed-pilot outcome and chronological review gates"},
		{models.ErrProviderProofManifestRequestConflict, http.StatusConflict, "different issuance evidence"},
		{models.ErrProviderProofManifestIntegrity, http.StatusInternalServerError, "failed integrity verification"},
	} {
		status, message := providerExchangeStatus(test.err)
		if status != test.status || !strings.Contains(message, test.message) {
			t.Fatalf("proof-manifest status mapping error=%v status=%d message=%q", test.err, status, message)
		}
	}

	source, err := os.ReadFile("provider_proof_manifest_admin.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	for _, required := range []string{
		"GetProviderCommercialProofManifest(h.DB, pilotID, h.Signer)",
		"errors.Is(err, sql.ErrNoRows)",
		"GetProviderCommercialProofManifestCandidate",
		`"manifest_candidate":`,
		`"issued":                   false`,
		`"commercial_proof_created": false`,
		`"publicly_released":        false`,
		`"independently_verifiable": false`,
		"IssueProviderCommercialProofManifest",
		"status = http.StatusCreated",
		`"idempotent_replay":        !created`,
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("provider proof-manifest handler contract missing %q", required)
		}
	}
}

func TestProviderSigningRetentionStartupCoversCommercialProofManifests(t *testing.T) {
	activeKeyID := "nhs-provider-current"
	activeSecret := strings.Repeat("k", 32)
	_, canonical, signature, _ := providerManifestTestSigningFixture(t, activeKeyID, activeSecret)
	manifestRow := func(keyID, signedManifest, signedSignature string) []driver.Value {
		return []driver.Value{
			keyID, models.ProviderSigningProofManifest,
			nil, nil, nil, nil, nil, nil,
			signedManifest, signedSignature,
		}
	}
	configureActive := func(t *testing.T) {
		t.Helper()
		t.Setenv("NHS_PROVIDER_EXCHANGE_SIGNING_KEY_ID", activeKeyID)
		t.Setenv("NHS_PROVIDER_EXCHANGE_SIGNING_KEY", activeSecret)
		t.Setenv("NHS_PROVIDER_EXCHANGE_PREVIOUS_SIGNING_KEYS_JSON", "")
	}

	t.Run("valid", func(t *testing.T) {
		configureActive(t)
		db := openProviderManifestTestDB(t, &providerManifestTestConnector{
			proofTablesExist:    true,
			manifestTableExists: true,
			signingProofRows:    [][]driver.Value{manifestRow(activeKeyID, canonical, signature)},
		})
		report, err := ValidateProviderExchangeSigningRetentionReadOnly(db, false)
		if err != nil {
			t.Fatalf("valid retained proof manifest rejected: %v", err)
		}
		if !report.SignerRequired || !report.ConfigurationValidated ||
			report.PersistedProofCount != 1 || len(report.Proofs) != 1 ||
			report.Proofs[0].KeyID != activeKeyID ||
			report.Proofs[0].Kind != models.ProviderSigningProofManifest {
			t.Fatalf("proof-manifest retention report=%#v", report)
		}
	})

	t.Run("tampered", func(t *testing.T) {
		configureActive(t)
		tampered := strings.Replace(canonical, `"verified_provider_renewals":1`, `"verified_provider_renewals":2`, 1)
		if tampered == canonical {
			t.Fatal("proof-manifest tamper fixture did not change canonical payload")
		}
		db := openProviderManifestTestDB(t, &providerManifestTestConnector{
			proofTablesExist:    true,
			manifestTableExists: true,
			signingProofRows:    [][]driver.Value{manifestRow(activeKeyID, tampered, signature)},
		})
		if _, err := ValidateProviderExchangeSigningRetentionReadOnly(db, false); err == nil ||
			!strings.Contains(err.Error(), "verification material changed") {
			t.Fatalf("tampered retained proof-manifest error=%v", err)
		}
	})

	t.Run("missing key", func(t *testing.T) {
		configureActive(t)
		retiredKeyID := "nhs-provider-retired"
		_, retiredCanonical, retiredSignature, _ := providerManifestTestSigningFixture(
			t, retiredKeyID, strings.Repeat("r", 32),
		)
		db := openProviderManifestTestDB(t, &providerManifestTestConnector{
			proofTablesExist:    true,
			manifestTableExists: true,
			signingProofRows:    [][]driver.Value{manifestRow(retiredKeyID, retiredCanonical, retiredSignature)},
		})
		if _, err := ValidateProviderExchangeSigningRetentionReadOnly(db, false); err == nil ||
			!strings.Contains(err.Error(), "verification material missing") {
			t.Fatalf("missing proof-manifest key error=%v", err)
		}
	})
}
