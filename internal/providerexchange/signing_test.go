package providerexchange

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

const (
	testTicketID   = "11111111-1111-4111-8111-111111111111"
	testOfferID    = "22222222-2222-4222-8222-222222222222"
	testReceiptID  = "33333333-3333-4333-8333-333333333333"
	testNHSEventID = "44444444-4444-4444-8444-444444444444"
	testNonce      = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
)

func TestAttributionRoundTripIsURLSafeAndPrivacyMinimal(t *testing.T) {
	now := time.Unix(1_800_000_000, 0).UTC()
	signer := mustSigner(t, "fixture-admin-secret-one-0123456789abcdef")
	claims := fixtureAttribution(now)

	token, err := signer.SignAttribution(claims)
	if err != nil {
		t.Fatalf("SignAttribution: %v", err)
	}
	if strings.ContainsAny(token, "+/=") {
		t.Fatalf("token is not unpadded URL-safe base64: %q", token)
	}
	if strings.Count(token, ".") != 2 {
		t.Fatalf("token must have key id, payload, and signature parts: %q", token)
	}

	verified, err := signer.VerifyAttribution(token, now)
	if err != nil {
		t.Fatalf("VerifyAttribution: %v", err)
	}
	if !reflect.DeepEqual(verified, claims) {
		t.Fatalf("verified claims mismatch\n got: %#v\nwant: %#v", verified, claims)
	}

	payload := decodeTokenPayload(t, token)
	wantPayload := `{"v":1,"kid":"nhs-provider-signing-v1","ticket_id":"11111111-1111-4111-8111-111111111111","offer_id":"22222222-2222-4222-8222-222222222222","issued_at":1800000000,"expires_at":1800000900,"nonce":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"}`
	if string(payload) != wantPayload {
		t.Fatalf("unexpected canonical attribution payload\n got: %s\nwant: %s", payload, wantPayload)
	}
	assertNoForbiddenJSONFields(t, payload, []string{
		"query", "email", "contact", "principal", "agent_id", "agent_identity",
		"ip", "network", "user_agent", "fingerprint", "referrer",
	})
}

func TestAttributionRejectsTamperWrongKeyExpiryAndFuture(t *testing.T) {
	now := time.Unix(1_800_000_000, 0).UTC()
	signer := mustSigner(t, "fixture-admin-secret-one-0123456789abcdef")
	token, err := signer.SignAttribution(fixtureAttribution(now))
	if err != nil {
		t.Fatalf("SignAttribution: %v", err)
	}

	parts := strings.Split(token, ".")
	payload := decodeRawURL(t, parts[1])
	payload[len(payload)-1] ^= 1
	tamperedPayload := parts[0] + "." + base64.RawURLEncoding.EncodeToString(payload) + "." + parts[2]
	if _, err := signer.VerifyAttribution(tamperedPayload, now); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("tampered payload error = %v, want ErrInvalidSignature", err)
	}

	signature := decodeRawURL(t, parts[2])
	signature[0] ^= 1
	tamperedSignature := parts[0] + "." + parts[1] + "." + base64.RawURLEncoding.EncodeToString(signature)
	if _, err := signer.VerifyAttribution(tamperedSignature, now); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("tampered signature error = %v, want ErrInvalidSignature", err)
	}

	other := mustSigner(t, "fixture-admin-secret-two-0123456789abcdef")
	if _, err := other.VerifyAttribution(token, now); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("wrong-key error = %v, want ErrInvalidSignature", err)
	}
	if _, err := signer.VerifyAttribution(token, now.Add(15*time.Minute)); !errors.Is(err, ErrExpired) {
		t.Fatalf("expired error = %v, want ErrExpired", err)
	}
	if claims, err := signer.VerifyAttributionSignature(token); err != nil || claims.TicketID != testTicketID {
		t.Fatalf("historical attribution signature = %#v, %v", claims, err)
	}

	futureClaims := fixtureAttribution(now)
	futureClaims.IssuedAt = now.Add(AllowedClockSkew + time.Second).Unix()
	futureClaims.ExpiresAt = now.Add(time.Hour).Unix()
	futureToken, err := signer.SignAttribution(futureClaims)
	if err != nil {
		t.Fatalf("sign future claims: %v", err)
	}
	if _, err := signer.VerifyAttribution(futureToken, now); !errors.Is(err, ErrNotYetValid) {
		t.Fatalf("future error = %v, want ErrNotYetValid", err)
	}
}

func TestAttributionRejectsUnknownAndNonCanonicalSignedPayloads(t *testing.T) {
	now := time.Unix(1_800_000_000, 0).UTC()
	signer := mustSigner(t, "fixture-admin-secret-one-0123456789abcdef")

	withPIIField := []byte(`{"v":1,"kid":"nhs-provider-signing-v1","ticket_id":"11111111-1111-4111-8111-111111111111","offer_id":"22222222-2222-4222-8222-222222222222","issued_at":1800000000,"expires_at":1800000900,"nonce":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA","email":"must-not-fit@example.com"}`)
	unknownToken := compactWithKey(DefaultSigningKeyID, withPIIField, activeMaterial(t, signer).attributionKey, attributionDomain)
	if _, err := signer.VerifyAttribution(unknownToken, now); !errors.Is(err, ErrMalformed) {
		t.Fatalf("unknown-field error = %v, want ErrMalformed", err)
	}

	nonCanonical := []byte(`{ "v":1, "kid":"nhs-provider-signing-v1", "ticket_id":"11111111-1111-4111-8111-111111111111", "offer_id":"22222222-2222-4222-8222-222222222222", "issued_at":1800000000, "expires_at":1800000900, "nonce":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" }`)
	nonCanonicalToken := compactWithKey(DefaultSigningKeyID, nonCanonical, activeMaterial(t, signer).attributionKey, attributionDomain)
	if _, err := signer.VerifyAttribution(nonCanonicalToken, now); !errors.Is(err, ErrMalformed) {
		t.Fatalf("non-canonical error = %v, want ErrMalformed", err)
	}

	validPayload, err := canonicalAttribution(fixtureAttribution(now))
	if err != nil {
		t.Fatalf("canonical attribution: %v", err)
	}
	malformedTrailing := append(append([]byte(nil), validPayload...), '!')
	trailingToken := compactWithKey(DefaultSigningKeyID, malformedTrailing, activeMaterial(t, signer).attributionKey, attributionDomain)
	if _, err := signer.VerifyAttribution(trailingToken, now); !errors.Is(err, ErrMalformed) {
		t.Fatalf("malformed-trailing error = %v, want ErrMalformed", err)
	}

	for _, malformed := range []string{"", "abc", ".", "abc.", ".abc", "abc.def", "abc.def.extra.more", "%%%.%%%.%%%"} {
		if _, err := signer.VerifyAttribution(malformed, now); !errors.Is(err, ErrMalformed) {
			t.Errorf("VerifyAttribution(%q) error = %v, want ErrMalformed", malformed, err)
		}
	}
}

func TestOutcomeReceiptCanonicalRoundTrip(t *testing.T) {
	now := time.Unix(1_800_000_000, 0).UTC()
	signer := mustSigner(t, "fixture-admin-secret-one-0123456789abcdef")
	receipt := fixtureReceipt(now)

	canonical, signature, err := signer.SignOutcomeReceipt(receipt)
	if err != nil {
		t.Fatalf("SignOutcomeReceipt: %v", err)
	}
	wantCanonical := `{"v":1,"kid":"nhs-provider-signing-v1","receipt_id":"33333333-3333-4333-8333-333333333333","ticket_id":"11111111-1111-4111-8111-111111111111","offer_id":"22222222-2222-4222-8222-222222222222","nhs_event_id":"44444444-4444-4444-8444-444444444444","outcome":"activated","provider_reported_at":1799999970,"recorded_at":1800000000,"expires_at":1831536000,"charged_minor":2500,"currency":"usd","charge_status":"charged"}`
	if canonical != wantCanonical {
		t.Fatalf("unexpected canonical receipt\n got: %s\nwant: %s", canonical, wantCanonical)
	}
	if strings.ContainsAny(signature, "+/=") || len(signature) != 43 {
		t.Fatalf("signature is not an unpadded base64url SHA-256 MAC: %q", signature)
	}

	canonicalAgain, signatureAgain, err := signer.SignOutcomeReceipt(receipt)
	if err != nil {
		t.Fatalf("SignOutcomeReceipt again: %v", err)
	}
	if canonicalAgain != canonical || signatureAgain != signature {
		t.Fatal("outcome serialization/signature is not deterministic")
	}

	verified, err := signer.VerifyOutcomeReceipt(canonical, signature, now)
	if err != nil {
		t.Fatalf("VerifyOutcomeReceipt: %v", err)
	}
	if !reflect.DeepEqual(verified, receipt) {
		t.Fatalf("verified receipt mismatch\n got: %#v\nwant: %#v", verified, receipt)
	}
	assertNoForbiddenJSONFields(t, []byte(canonical), []string{
		"query", "email", "contact", "principal", "agent_identity", "ip", "user_agent", "network",
	})
}

func TestOutcomeReceiptRejectsTamperWrongKeyExpiryAndFuture(t *testing.T) {
	now := time.Unix(1_800_000_000, 0).UTC()
	signer := mustSigner(t, "fixture-admin-secret-one-0123456789abcdef")
	receipt := fixtureReceipt(now)
	canonical, signature, err := signer.SignOutcomeReceipt(receipt)
	if err != nil {
		t.Fatalf("SignOutcomeReceipt: %v", err)
	}

	tampered := strings.Replace(canonical, `"activated"`, `"converted"`, 1)
	if _, err := signer.VerifyOutcomeReceipt(tampered, signature, now); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("tampered receipt error = %v, want ErrInvalidSignature", err)
	}

	tamperedMAC := decodeRawURL(t, signature)
	tamperedMAC[len(tamperedMAC)-1] ^= 1
	if _, err := signer.VerifyOutcomeReceipt(canonical, base64.RawURLEncoding.EncodeToString(tamperedMAC), now); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("tampered receipt signature error = %v, want ErrInvalidSignature", err)
	}

	other := mustSigner(t, "fixture-admin-secret-two-0123456789abcdef")
	if _, err := other.VerifyOutcomeReceipt(canonical, signature, now); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("wrong-key receipt error = %v, want ErrInvalidSignature", err)
	}
	if _, err := signer.VerifyOutcomeReceipt(canonical, signature, time.Unix(receipt.ExpiresAt, 0)); !errors.Is(err, ErrExpired) {
		t.Fatalf("expired receipt error = %v, want ErrExpired", err)
	}

	future := fixtureReceipt(now)
	future.ProviderReportedAt = now.Add(AllowedClockSkew + time.Second).Unix()
	future.RecordedAt = future.ProviderReportedAt
	future.ExpiresAt = future.RecordedAt + int64(time.Hour/time.Second)
	futureCanonical, futureSignature, err := signer.SignOutcomeReceipt(future)
	if err != nil {
		t.Fatalf("sign future receipt: %v", err)
	}
	if _, err := signer.VerifyOutcomeReceipt(futureCanonical, futureSignature, now); !errors.Is(err, ErrNotYetValid) {
		t.Fatalf("future receipt error = %v, want ErrNotYetValid", err)
	}

	withUnknown := strings.TrimSuffix(canonical, "}") + `,"email":"must-not-fit@example.com"}`
	unknownMAC := signMAC(activeMaterial(t, signer).outcomeKey, outcomeDomain, []byte(withUnknown))
	if _, err := signer.VerifyOutcomeReceipt(withUnknown, base64.RawURLEncoding.EncodeToString(unknownMAC[:]), now); !errors.Is(err, ErrMalformed) {
		t.Fatalf("unknown receipt field error = %v, want ErrMalformed", err)
	}
}

func TestAttributionAndOutcomeDomainsAreNotInterchangeable(t *testing.T) {
	now := time.Unix(1_800_000_000, 0).UTC()
	signer := mustSigner(t, "fixture-admin-secret-one-0123456789abcdef")

	attributionPayload, err := canonicalAttribution(fixtureAttribution(now))
	if err != nil {
		t.Fatalf("canonical attribution: %v", err)
	}
	tokenSignedWithOutcomeKey := compactWithKey(DefaultSigningKeyID, attributionPayload, activeMaterial(t, signer).outcomeKey, outcomeDomain)
	if _, err := signer.VerifyAttribution(tokenSignedWithOutcomeKey, now); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("outcome-domain attribution error = %v, want ErrInvalidSignature", err)
	}

	outcomePayload, err := CanonicalOutcomeReceipt(fixtureReceipt(now))
	if err != nil {
		t.Fatalf("canonical outcome: %v", err)
	}
	wrongMAC := signMAC(activeMaterial(t, signer).attributionKey, attributionDomain, outcomePayload)
	if _, err := signer.VerifyOutcomeReceipt(string(outcomePayload), base64.RawURLEncoding.EncodeToString(wrongMAC[:]), now); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("attribution-domain outcome error = %v, want ErrInvalidSignature", err)
	}
}

func TestSigningInputValidation(t *testing.T) {
	if _, err := NewSigner(""); !errors.Is(err, ErrSecretRequired) {
		t.Fatalf("empty secret error = %v, want ErrSecretRequired", err)
	}
	if _, err := NewSigner("   "); !errors.Is(err, ErrSecretRequired) {
		t.Fatalf("whitespace secret error = %v, want ErrSecretRequired", err)
	}
	if _, err := NewSigner("short-dedicated-secret"); !errors.Is(err, ErrSecretRequired) {
		t.Fatalf("short secret error = %v, want ErrSecretRequired", err)
	}

	now := time.Unix(1_800_000_000, 0).UTC()
	signer := mustSigner(t, "fixture-admin-secret-one-0123456789abcdef")
	claims := fixtureAttribution(now)
	claims.TicketID = "person@example.com"
	if _, err := signer.SignAttribution(claims); !errors.Is(err, ErrInvalidClaims) {
		t.Fatalf("PII-like ticket ID error = %v, want ErrInvalidClaims", err)
	}
	claims = fixtureAttribution(now)
	claims.Nonce = "raw-query-or-request-derived"
	if _, err := signer.SignAttribution(claims); !errors.Is(err, ErrInvalidClaims) {
		t.Fatalf("invalid nonce error = %v, want ErrInvalidClaims", err)
	}

	receipt := fixtureReceipt(now)
	receipt.Currency = "USD"
	if _, _, err := signer.SignOutcomeReceipt(receipt); !errors.Is(err, ErrInvalidClaims) {
		t.Fatalf("non-canonical currency error = %v, want ErrInvalidClaims", err)
	}
	receipt = fixtureReceipt(now)
	receipt.ChargeStatus = ChargeStatusNone
	if _, _, err := signer.SignOutcomeReceipt(receipt); !errors.Is(err, ErrInvalidClaims) {
		t.Fatalf("inconsistent charge error = %v, want ErrInvalidClaims", err)
	}
}

func TestSigningKeyringRetainsPreviousTokenAndReceiptVerification(t *testing.T) {
	now := time.Unix(1_800_000_000, 0).UTC()
	oldSigner, err := NewSignerKeyring("pilot-2026-08", "old-provider-secret-0123456789abcdef", nil)
	if err != nil {
		t.Fatal(err)
	}
	claims := fixtureAttribution(now)
	claims.KeyID = oldSigner.ActiveKeyID()
	oldToken, err := oldSigner.SignAttribution(claims)
	if err != nil {
		t.Fatal(err)
	}
	receipt := fixtureReceipt(now)
	receipt.KeyID = oldSigner.ActiveKeyID()
	oldCanonical, oldSignature, err := oldSigner.SignOutcomeReceipt(receipt)
	if err != nil {
		t.Fatal(err)
	}

	rotated, err := NewSignerKeyring("pilot-2026-09", "new-provider-secret-0123456789abcdef", map[string]string{
		"pilot-2026-08": "old-provider-secret-0123456789abcdef",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := rotated.VerifyAttribution(oldToken, now); err != nil {
		t.Fatalf("retained previous attribution key did not verify: %v", err)
	}
	if _, err := rotated.VerifyOutcomeReceiptSignature(oldCanonical, oldSignature); err != nil {
		t.Fatalf("retained previous receipt key did not verify: %v", err)
	}
	if !rotated.SupportsKeyID("pilot-2026-09") || !rotated.SupportsKeyID("pilot-2026-08") {
		t.Fatal("keyring did not report loaded active and retained key IDs")
	}
	if rotated.SupportsKeyID("pilot-2025-01") || (*Signer)(nil).SupportsKeyID("pilot-2026-08") {
		t.Fatal("keyring reported unsupported verification material")
	}

	withoutPrevious, err := NewSignerKeyring("pilot-2026-09", "new-provider-secret-0123456789abcdef", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := withoutPrevious.VerifyAttribution(oldToken, now); !errors.Is(err, ErrUnknownKeyID) {
		t.Fatalf("removed attribution key error = %v, want ErrUnknownKeyID", err)
	}
	if _, err := withoutPrevious.VerifyOutcomeReceiptSignature(oldCanonical, oldSignature); !errors.Is(err, ErrUnknownKeyID) {
		t.Fatalf("removed receipt key error = %v, want ErrUnknownKeyID", err)
	}
}

func TestOutcomeSignatureValidityIsSeparateFromFreshness(t *testing.T) {
	now := time.Unix(1_800_000_000, 0).UTC()
	signer := mustSigner(t, "fixture-provider-secret-0123456789abcdef")
	receipt := fixtureReceipt(now)
	canonical, signature, err := signer.SignOutcomeReceipt(receipt)
	if err != nil {
		t.Fatal(err)
	}
	afterExpiry := time.Unix(receipt.ExpiresAt+1, 0)
	if _, err := signer.VerifyOutcomeReceipt(canonical, signature, afterExpiry); !errors.Is(err, ErrExpired) {
		t.Fatalf("freshness verification error = %v, want ErrExpired", err)
	}
	if _, err := signer.VerifyOutcomeReceiptSignature(canonical, signature); err != nil {
		t.Fatalf("valid historical signature rejected after freshness expiry: %v", err)
	}
}

func TestNewNonceIsRandomURLSafeEntropy(t *testing.T) {
	first, err := NewNonce()
	if err != nil {
		t.Fatalf("NewNonce: %v", err)
	}
	second, err := NewNonce()
	if err != nil {
		t.Fatalf("NewNonce second: %v", err)
	}
	if first == second {
		t.Fatal("two independently generated nonces matched")
	}
	if !isBase64URLNonce(first) || !isBase64URLNonce(second) {
		t.Fatalf("invalid generated nonces: %q %q", first, second)
	}
}

func fixtureAttribution(now time.Time) AttributionClaims {
	return AttributionClaims{
		Version:   AttributionTokenVersion,
		KeyID:     DefaultSigningKeyID,
		TicketID:  testTicketID,
		OfferID:   testOfferID,
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(15 * time.Minute).Unix(),
		Nonce:     testNonce,
	}
}

func fixtureReceipt(now time.Time) OutcomeReceipt {
	return OutcomeReceipt{
		Version:            OutcomeReceiptVersion,
		KeyID:              DefaultSigningKeyID,
		ReceiptID:          testReceiptID,
		TicketID:           testTicketID,
		OfferID:            testOfferID,
		NHSEventID:         testNHSEventID,
		Outcome:            OutcomeActivated,
		ProviderReportedAt: now.Add(-30 * time.Second).Unix(),
		RecordedAt:         now.Unix(),
		ExpiresAt:          now.Add(365 * 24 * time.Hour).Unix(),
		ChargedMinor:       2500,
		Currency:           "usd",
		ChargeStatus:       ChargeStatusCharged,
	}
}

func mustSigner(t *testing.T, secret string) *Signer {
	t.Helper()
	signer, err := NewSigner(secret)
	if err != nil {
		t.Fatalf("NewSigner: %v", err)
	}
	return signer
}

func decodeTokenPayload(t *testing.T, token string) []byte {
	t.Helper()
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("bad compact token: %q", token)
	}
	return decodeRawURL(t, parts[1])
}

func decodeRawURL(t *testing.T, value string) []byte {
	t.Helper()
	decoded, err := base64.RawURLEncoding.Strict().DecodeString(value)
	if err != nil {
		t.Fatalf("decode base64url: %v", err)
	}
	return decoded
}

func compactWithKey(keyID string, payload []byte, key [32]byte, domain string) string {
	mac := signMAC(key, domain, payload)
	return keyID + "." + base64.RawURLEncoding.EncodeToString(payload) + "." + base64.RawURLEncoding.EncodeToString(mac[:])
}

func activeMaterial(t *testing.T, signer *Signer) signingKeyMaterial {
	t.Helper()
	material, ok := signer.keys[signer.ActiveKeyID()]
	if !ok {
		t.Fatal("active signing material missing")
	}
	return material
}

func assertNoForbiddenJSONFields(t *testing.T, payload []byte, forbidden []string) {
	t.Helper()
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(payload, &fields); err != nil {
		t.Fatalf("decode privacy field set: %v", err)
	}
	for _, field := range forbidden {
		if _, exists := fields[field]; exists {
			t.Fatalf("signed payload contains forbidden field %q: %s", field, payload)
		}
	}
}
