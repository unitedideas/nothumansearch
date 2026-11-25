package models

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
)

type providerSigningKeyTestConnector struct {
	manifestTableExists bool
	proofRows           [][]driver.Value
}

func (c *providerSigningKeyTestConnector) Connect(context.Context) (driver.Conn, error) {
	return &providerSigningKeyTestConn{connector: c}, nil
}

func (*providerSigningKeyTestConnector) Driver() driver.Driver {
	return providerSigningKeyTestDriver{}
}

type providerSigningKeyTestDriver struct{}

func (providerSigningKeyTestDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("provider signing-key test driver requires OpenDB")
}

type providerSigningKeyTestConn struct {
	connector *providerSigningKeyTestConnector
}

func (*providerSigningKeyTestConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("provider signing-key test driver does not prepare statements")
}

func (*providerSigningKeyTestConn) Close() error { return nil }

func (*providerSigningKeyTestConn) Begin() (driver.Tx, error) {
	return nil, errors.New("provider signing-key test driver does not begin transactions")
}

func (c *providerSigningKeyTestConn) QueryContext(
	_ context.Context,
	query string,
	_ []driver.NamedValue,
) (driver.Rows, error) {
	if strings.Contains(query, "to_regclass('public.provider_commercial_proof_manifests')") {
		return &providerSigningKeyTestRows{
			columns: []string{"manifest_table_exists"},
			values:  [][]driver.Value{{c.connector.manifestTableExists}},
		}, nil
	}
	if strings.Contains(query, "SELECT DISTINCT ON (key_id, proof_kind)") {
		return &providerSigningKeyTestRows{
			columns: []string{
				"key_id", "proof_kind", "ticket_id", "offer_id",
				"issued_at", "expires_at", "token_nonce", "token_hash",
				"signed_receipt", "signature",
			},
			values: c.connector.proofRows,
		}, nil
	}
	return nil, errors.New("unexpected provider signing-key test query")
}

type providerSigningKeyTestRows struct {
	columns []string
	values  [][]driver.Value
	index   int
}

func (r *providerSigningKeyTestRows) Columns() []string { return r.columns }
func (*providerSigningKeyTestRows) Close() error        { return nil }
func (r *providerSigningKeyTestRows) Next(dest []driver.Value) error {
	if r.index >= len(r.values) {
		return io.EOF
	}
	copy(dest, r.values[r.index])
	r.index++
	return nil
}

func providerSigningKeyManifestRow(keyID, signedManifest, signature string) []driver.Value {
	return []driver.Value{
		keyID, ProviderSigningProofManifest,
		nil, nil, nil, nil, nil, nil,
		signedManifest, signature,
	}
}

func TestProviderSigningKeyProofsInUseFailsClosedWithoutStore(t *testing.T) {
	if _, err := ProviderSigningKeyProofsInUse(nil); !errors.Is(err, ErrInvalidProviderExchange) {
		t.Fatalf("nil signing-key store error = %v, want ErrInvalidProviderExchange", err)
	}
	if _, err := ProviderSigningKeyProofsInUseTx(context.Background(), nil); !errors.Is(err, ErrInvalidProviderExchange) {
		t.Fatalf("nil signing-key transaction error = %v, want ErrInvalidProviderExchange", err)
	}
}

func TestProviderSigningKeyRetentionQueryCoversTicketsReceiptsAndManifests(t *testing.T) {
	source, err := os.ReadFile("provider_signing_keys.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	for _, required := range []string{
		"attribution_key_id_snapshot",
		"signed_receipt::jsonb->>'kid'",
		"provider_commercial_proof_manifests",
		"'manifest'::text AS proof_kind",
		"signed_manifest AS signed_receipt",
		"DISTINCT ON (key_id, proof_kind)",
		"token_hash",
		"signature",
		"providerSigningKeyIDPattern",
		"ProviderSigningProofManifest",
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("signing-key retention query missing %q", required)
		}
	}
}

func TestProviderSigningKeyProofsInUseScansManifestForRetentionVerification(t *testing.T) {
	keyID := "nhs-provider-current"
	signature := strings.Repeat("A", 43)
	for _, signedManifest := range []string{
		`{"kid":"nhs-provider-current","proof":"valid"}`,
		`{"kid":"nhs-provider-current","proof":"tampered"}`,
	} {
		db := sql.OpenDB(&providerSigningKeyTestConnector{
			manifestTableExists: true,
			proofRows: [][]driver.Value{
				providerSigningKeyManifestRow(keyID, signedManifest, signature),
			},
		})
		proofs, err := ProviderSigningKeyProofsInUse(db)
		_ = db.Close()
		if err != nil {
			t.Fatalf("scan retained proof manifest: %v", err)
		}
		if len(proofs) != 1 || proofs[0].KeyID != keyID ||
			proofs[0].Kind != ProviderSigningProofManifest ||
			proofs[0].SignedManifest != signedManifest ||
			proofs[0].Signature != signature {
			t.Fatalf("retained proof manifest=%#v", proofs)
		}
		if proofs[0].SignedReceipt != "" {
			t.Fatalf("manifest proof leaked into outcome field: %#v", proofs[0])
		}
	}
}

func TestProviderSigningKeyProofsInUseRejectsMalformedManifestIdentity(t *testing.T) {
	for _, row := range [][]driver.Value{
		providerSigningKeyManifestRow("", `{"kid":"missing"}`, strings.Repeat("A", 43)),
		providerSigningKeyManifestRow("bad key id", `{"kid":"bad"}`, strings.Repeat("A", 43)),
		providerSigningKeyManifestRow("nhs-provider-current", `{"kid":"current"}`, "short"),
	} {
		db := sql.OpenDB(&providerSigningKeyTestConnector{
			manifestTableExists: true,
			proofRows:           [][]driver.Value{row},
		})
		_, err := ProviderSigningKeyProofsInUse(db)
		_ = db.Close()
		if !errors.Is(err, ErrInvalidProviderExchange) {
			t.Fatalf("malformed retained proof-manifest error=%v", err)
		}
	}
}
