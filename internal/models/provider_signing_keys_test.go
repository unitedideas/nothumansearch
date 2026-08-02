package models

import (
	"errors"
	"os"
	"strings"
	"testing"
)

func TestProviderSigningKeyProofsInUseFailsClosedWithoutStore(t *testing.T) {
	if _, err := ProviderSigningKeyProofsInUse(nil); !errors.Is(err, ErrInvalidProviderExchange) {
		t.Fatalf("nil signing-key store error = %v, want ErrInvalidProviderExchange", err)
	}
}

func TestProviderSigningKeyRetentionQueryCoversTicketsAndReceipts(t *testing.T) {
	source, err := os.ReadFile("provider_signing_keys.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	for _, required := range []string{
		"attribution_key_id_snapshot",
		"signed_receipt::jsonb->>'kid'",
		"DISTINCT ON (key_id, proof_kind)",
		"token_hash",
		"signature",
		"providerSigningKeyIDPattern",
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("signing-key retention query missing %q", required)
		}
	}
}
