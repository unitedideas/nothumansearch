package models

import "testing"

func TestNormalizeDomain(t *testing.T) {
	tests := []struct {
		in      string
		wantOut string
		wantErr bool
	}{
		{"example.com", "example.com", false},
		{"https://example.com/path", "example.com", false},
		{"  WWW.Example.COM  ", "example.com", false},
		{"http://sub.example.com:8080/x", "sub.example.com", false},
		{"", "", true},
		{"notadomain", "", true},
		// SSRF targets must be rejected.
		{"localhost", "", true},
		{"127.0.0.1", "", true},
		{"10.0.0.5", "", true},
		{"192.168.1.1", "", true},
		{"169.254.169.254", "", true}, // cloud metadata
		{"0.0.0.0", "", true},
		// 172.16.0.0/12
		{"172.16.0.1", "", true},
		{"172.20.5.5", "", true},
		{"172.31.255.255", "", true},
		// 172 outside the private range must pass.
		{"172.15.0.1", "172.15.0.1", false},
		{"172.32.0.1", "172.32.0.1", false},
		{"172.217.14.206", "172.217.14.206", false}, // google.com
	}
	for _, tc := range tests {
		got, err := NormalizeDomain(tc.in)
		if tc.wantErr && err == nil {
			t.Errorf("NormalizeDomain(%q) = %q, want error", tc.in, got)
		}
		if !tc.wantErr && err != nil {
			t.Errorf("NormalizeDomain(%q) returned error: %v", tc.in, err)
		}
		if !tc.wantErr && got != tc.wantOut {
			t.Errorf("NormalizeDomain(%q) = %q, want %q", tc.in, got, tc.wantOut)
		}
	}
}

func TestValidateEmail(t *testing.T) {
	good := []string{"a@b.co", "foo.bar+tag@example.com", "USER@EXAMPLE.COM"}
	bad := []string{"", "noatsign", "@no-local", "no-domain@", "a@b", "a"}
	for _, e := range good {
		if _, err := ValidateEmail(e); err != nil {
			t.Errorf("ValidateEmail(%q) unexpected error: %v", e, err)
		}
	}
	for _, e := range bad {
		if _, err := ValidateEmail(e); err == nil {
			t.Errorf("ValidateEmail(%q) should have errored", e)
		}
	}
}

// TestNewToken locks in the monitor-token contract:
//   - 48 hex chars (24 bytes = 192 bits of entropy)
//   - Only lowercase hex characters
//   - No collisions across many calls (crypto/rand)
//
// A regression that shortens the random part weakens unsubscribe-link
// security (tokens are the ONLY auth for unsubscribe endpoints per
// SECURITY.md — no login flow).
func TestNewToken(t *testing.T) {
	tok, err := newToken()
	if err != nil {
		t.Fatalf("newToken returned error: %v", err)
	}
	if len(tok) != 48 {
		t.Errorf("newToken len = %d, want 48 (24 bytes hex-encoded)", len(tok))
	}
	// Must be pure hex (lowercase).
	for _, c := range tok {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("newToken contains non-hex char %q in %q", c, tok)
			break
		}
	}

	// 100 calls, zero collisions. 24 bytes → collision probability effectively 0.
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		tok, err := newToken()
		if err != nil {
			t.Fatalf("newToken iter %d: %v", i, err)
		}
		if seen[tok] {
			t.Errorf("newToken collision at iter %d: %q", i, tok)
		}
		seen[tok] = true
	}
}

// TestIs172Private_Boundaries covers the 172.16.0.0/12 range. This range is
// the nastiest private-IP boundary to get right — not a simple prefix match —
// and SSRF blocking depends on it. Included ranges: 172.16 through 172.31.
// Excluded: 172.0–15 (public) and 172.32+ (public).
func TestIs172Private_Boundaries(t *testing.T) {
	private := []string{
		"172.16.0.1",
		"172.17.255.255",
		"172.20.0.0",
		"172.25.1.1",
		"172.31.255.255",
		// Exact-match boundaries
		"172.16.0.0",
		"172.31.0.0",
	}
	public := []string{
		"172.0.0.0",
		"172.1.1.1",
		"172.15.255.255",  // one below range
		"172.32.0.0",      // one above range
		"172.33.1.1",
		"172.100.0.0",
		"172.255.255.255",
		// Edge cases the routine must not mis-classify
		"172.1",            // no dot after
		"172.",             // empty second octet
		"172..1",           // double dot
		"172.16",           // range-valid first-two-octets but no third
		// 3-digit octet must reject regardless of the first-2-digits trick
		"172.160.0.0",
		"172.300.0.0",
	}
	for _, host := range private {
		if !is172Private(host) {
			t.Errorf("is172Private(%q) = false, want true (in 172.16.0.0/12 range)", host)
		}
	}
	for _, host := range public {
		if is172Private(host) {
			t.Errorf("is172Private(%q) = true, want false (outside 172.16.0.0/12 range)", host)
		}
	}
}

// TestSanitizeUTF8 locks in the NUL-byte + invalid-UTF-8 stripping contract.
// Prevents regression of the SAVE FAIL bug fixed in commit a3fd496 where
// favicon URLs containing 0x00 would reach Postgres and trigger:
//   pq: invalid byte sequence for encoding "UTF8": 0x00 (22021)
// 0x00 is valid UTF-8 (passes utf8.ValidString) but Postgres rejects it in
// text columns — so sanitizeUTF8 must strip it unconditionally.
func TestSanitizeUTF8_NulByte(t *testing.T) {
	if got := sanitizeUTF8("zero\x00byte"); got != "zerobyte" {
		t.Errorf("sanitizeUTF8(NUL) = %q, want %q", got, "zerobyte")
	}
	// Multiple NULs + mixed with other content.
	if got := sanitizeUTF8("\x00leading"); got != "leading" {
		t.Errorf("sanitizeUTF8(leading NUL) = %q", got)
	}
	if got := sanitizeUTF8("trailing\x00"); got != "trailing" {
		t.Errorf("sanitizeUTF8(trailing NUL) = %q", got)
	}
	if got := sanitizeUTF8("a\x00b\x00c"); got != "abc" {
		t.Errorf("sanitizeUTF8(interspersed NUL) = %q", got)
	}
	// Valid UTF-8 with no NUL passes through unchanged.
	in := "https://avatars.githubusercontent.com/u/12345"
	if got := sanitizeUTF8(in); got != in {
		t.Errorf("sanitizeUTF8 mutated valid input: %q → %q", in, got)
	}
	// Invalid UTF-8 still gets cleaned (0xFF is never a legal start byte).
	bad := "ok" + string([]byte{0xFF}) + "fine"
	got := sanitizeUTF8(bad)
	if !isValidASCIIish(got) {
		t.Errorf("sanitizeUTF8 left invalid byte: %q", got)
	}
}

func isValidASCIIish(s string) bool {
	for _, b := range []byte(s) {
		if b == 0xFF {
			return false
		}
	}
	return true
}
