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
