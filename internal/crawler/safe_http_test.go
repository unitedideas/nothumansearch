package crawler

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

func noDNSLookup(t *testing.T) lookupIPAddrFunc {
	t.Helper()
	return func(context.Context, string) ([]net.IPAddr, error) {
		t.Fatal("DNS lookup was not expected for this target")
		return nil, nil
	}
}

func TestValidatePublicURLRejectsUnsafeTargets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		url  string
	}{
		{"non-http-scheme", "file:///etc/passwd"},
		{"embedded-userinfo", "https://account@8.8.8.8/path"},
		{"missing-host", "https:///path"},
		{"single-label-host", "https://intranet/path"},
		{"localhost", "http://localhost:8080"},
		{"localhost-subdomain", "http://api.localhost"},
		{"mdns-host", "http://printer.local"},
		{"internal-host", "https://metadata.google.internal"},
		{"unspecified-v4", "http://0.1.2.3"},
		{"loopback-v4", "http://127.0.0.1"},
		{"private-v4", "http://10.20.30.40"},
		{"link-local-v4", "http://169.254.169.254/latest/meta-data"},
		{"carrier-nat-v4", "http://100.64.0.1"},
		{"protocol-v4", "http://192.0.0.1"},
		{"documentation-v4", "http://192.0.2.10"},
		{"benchmark-v4", "http://198.18.0.1"},
		{"multicast-v4", "http://224.0.0.1"},
		{"reserved-v4", "http://240.0.0.1"},
		{"loopback-v6", "http://[::1]"},
		{"private-v6", "http://[fd00::1]"},
		{"link-local-v6", "http://[fe80::1]"},
		{"deprecated-site-local-v6", "http://[fec0::1]"},
		{"reserved-v6", "http://[4000::1]"},
		{"documentation-v6", "http://[2001:db8::1]"},
		{"nat64-v6", "http://[64:ff9b::808:808]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validatePublicURL(context.Background(), tt.url, noDNSLookup(t)); err == nil {
				t.Fatalf("validatePublicURL(%q) succeeded, want rejection", tt.url)
			}
		})
	}
}

func TestValidatePublicURLAllowsPublicLiteralsWithoutNetwork(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{
		"http://8.8.8.8",
		"https://1.1.1.1:8443/path",
		"https://[2606:4700:4700::1111]/dns-query",
	} {
		if err := validatePublicURL(context.Background(), raw, noDNSLookup(t)); err != nil {
			t.Errorf("validatePublicURL(%q) returned %v, want success", raw, err)
		}
	}
}

func TestValidatePublicURLRejectsMixedDNSAnswer(t *testing.T) {
	t.Parallel()

	lookup := func(context.Context, string) ([]net.IPAddr, error) {
		return []net.IPAddr{
			{IP: net.ParseIP("8.8.8.8")},
			{IP: net.ParseIP("127.0.0.1")},
		}, nil
	}
	if err := validatePublicURL(context.Background(), "https://mixed.example/path", lookup); err == nil {
		t.Fatal("mixed public/private DNS answer succeeded, want rejection")
	}
}

func TestValidatePublicURLAllowsPublicDNSAnswer(t *testing.T) {
	t.Parallel()

	lookup := func(context.Context, string) ([]net.IPAddr, error) {
		return []net.IPAddr{
			{IP: net.ParseIP("8.8.8.8")},
			{IP: net.ParseIP("2606:4700:4700::1111")},
		}, nil
	}
	if err := validatePublicURL(context.Background(), "https://public.example/path", lookup); err != nil {
		t.Fatalf("public DNS answer was rejected: %v", err)
	}
}

func TestPublicDialContextPinsValidatedAddress(t *testing.T) {
	t.Parallel()

	lookup := func(_ context.Context, host string) ([]net.IPAddr, error) {
		if host != "public.example" {
			t.Fatalf("lookup host = %q, want public.example", host)
		}
		return []net.IPAddr{{IP: net.ParseIP("8.8.8.8")}}, nil
	}
	wantErr := errors.New("stop before network")
	var dialedAddress string
	dial := func(_ context.Context, network, address string) (net.Conn, error) {
		if network != "tcp" {
			t.Errorf("network = %q, want tcp", network)
		}
		dialedAddress = address
		return nil, wantErr
	}

	_, err := publicDialContext(context.Background(), "tcp", "public.example:443", lookup, dial)
	if !errors.Is(err, wantErr) {
		t.Fatalf("publicDialContext error = %v, want %v", err, wantErr)
	}
	if dialedAddress != "8.8.8.8:443" {
		t.Fatalf("dial address = %q, want validated literal 8.8.8.8:443", dialedAddress)
	}
}

func TestPublicDialContextDoesNotDialUnsafeAnswer(t *testing.T) {
	t.Parallel()

	lookup := func(context.Context, string) ([]net.IPAddr, error) {
		return []net.IPAddr{{IP: net.ParseIP("169.254.169.254")}}, nil
	}
	dial := func(context.Context, string, string) (net.Conn, error) {
		t.Fatal("dial called for unsafe DNS answer")
		return nil, nil
	}
	if _, err := publicDialContext(context.Background(), "tcp", "metadata.example:80", lookup, dial); err == nil {
		t.Fatal("unsafe dial target succeeded, want rejection")
	}
}

func TestPublicHTTPClientDisablesProxyAndValidatesRedirects(t *testing.T) {
	t.Parallel()

	lookup := func(context.Context, string) ([]net.IPAddr, error) {
		return []net.IPAddr{{IP: net.ParseIP("8.8.8.8")}}, nil
	}
	dial := func(context.Context, string, string) (net.Conn, error) {
		return nil, errors.New("network disabled in test")
	}
	client := newPublicHTTPClientWithDeps(time.Second, 2, lookup, dial)
	wrapped, ok := client.Transport.(*publicOnlyTransport)
	if !ok {
		t.Fatalf("transport type = %T, want *publicOnlyTransport", client.Transport)
	}
	if wrapped.transport.Proxy != nil {
		t.Fatal("outbound transport proxy is configured, want direct-only")
	}

	unsafe, _ := http.NewRequest(http.MethodGet, "http://127.0.0.1/admin", nil)
	if err := client.CheckRedirect(unsafe, nil); err == nil || !strings.Contains(err.Error(), "unsafe redirect") {
		t.Fatalf("private redirect error = %v, want unsafe redirect rejection", err)
	}

	public, _ := http.NewRequest(http.MethodGet, "https://public.example/next", nil)
	if err := client.CheckRedirect(public, []*http.Request{{}, {}}); err != nil {
		t.Fatalf("redirect at configured cap was rejected: %v", err)
	}
	if err := client.CheckRedirect(public, []*http.Request{{}, {}, {}}); err == nil || !strings.Contains(err.Error(), "too many redirects") {
		t.Fatalf("redirect above configured cap error = %v, want too many redirects", err)
	}
}
