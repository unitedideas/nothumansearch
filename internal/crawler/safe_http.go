package crawler

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const publicURLValidationTimeout = 5 * time.Second

type lookupIPAddrFunc func(context.Context, string) ([]net.IPAddr, error)
type dialContextFunc func(context.Context, string, string) (net.Conn, error)

var blockedPublicTargetPrefixes = mustParsePrefixes(
	// IPv4 special-purpose ranges that IsGlobalUnicast may still classify as
	// globally routable. Private, loopback, link-local and multicast ranges are
	// rejected separately by netip.Addr methods below.
	"0.0.0.0/8",
	"100.64.0.0/10",
	"192.0.0.0/24",
	"192.0.2.0/24",
	"192.88.99.0/24",
	"198.18.0.0/15",
	"198.51.100.0/24",
	"203.0.113.0/24",
	"240.0.0.0/4",

	// IPv6 special-purpose ranges: translation/local-use, discard-only,
	// protocol assignments, documentation, 6to4 and benchmarking space.
	"64:ff9b::/96",
	"64:ff9b:1::/48",
	"100::/64",
	"2001::/23",
	"2001:db8::/32",
	"2002::/16",
	"3fff::/20",
	"5f00::/16",
)

var allocatedGlobalIPv6Prefix = netip.MustParsePrefix("2000::/3")

func mustParsePrefixes(raw ...string) []netip.Prefix {
	prefixes := make([]netip.Prefix, 0, len(raw))
	for _, value := range raw {
		prefixes = append(prefixes, netip.MustParsePrefix(value))
	}
	return prefixes
}

func defaultLookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error) {
	return net.DefaultResolver.LookupIPAddr(ctx, host)
}

// ValidatePublicURL verifies that raw is an HTTP(S) URL whose hostname
// currently resolves only to public Internet addresses. Outbound clients do
// this check again at dial time, so a DNS answer changing after this call
// cannot redirect a request into the private network.
func ValidatePublicURL(raw string) error {
	ctx, cancel := context.WithTimeout(context.Background(), publicURLValidationTimeout)
	defer cancel()
	return validatePublicURL(ctx, raw, defaultLookupIPAddr)
}

func validatePublicURL(ctx context.Context, raw string, lookup lookupIPAddrFunc) error {
	u, err := parsePublicHTTPURL(raw)
	if err != nil {
		return err
	}
	_, err = resolvePublicIPs(ctx, u.Hostname(), lookup)
	return err
}

func parsePublicHTTPURL(raw string) (*url.URL, error) {
	if strings.TrimSpace(raw) != raw || raw == "" {
		return nil, errors.New("target URL must not be empty or contain surrounding whitespace")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid target URL: %w", err)
	}
	if !strings.EqualFold(u.Scheme, "http") && !strings.EqualFold(u.Scheme, "https") {
		return nil, errors.New("target URL must use http or https")
	}
	if u.Opaque != "" || u.Host == "" || u.Hostname() == "" {
		return nil, errors.New("target URL must include a hostname")
	}
	if u.User != nil {
		return nil, errors.New("target URL must not include credentials")
	}
	if strings.Contains(u.Hostname(), "%") {
		return nil, errors.New("target URL must not include a scoped address")
	}
	if port := u.Port(); port != "" {
		value, err := strconv.Atoi(port)
		if err != nil || value < 1 || value > 65535 {
			return nil, errors.New("target URL contains an invalid port")
		}
	}
	if isLocalHostname(u.Hostname()) {
		return nil, errors.New("target URL hostname is local or non-public")
	}
	return u, nil
}

func isLocalHostname(host string) bool {
	host = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(host), "."))
	if host == "" {
		return true
	}
	if net.ParseIP(host) != nil {
		return false
	}
	if !strings.Contains(host, ".") {
		return true
	}
	for _, suffix := range []string{
		"localhost",
		"local",
		"localdomain",
		"internal",
		"lan",
		"home",
		"home.arpa",
	} {
		if host == suffix || strings.HasSuffix(host, "."+suffix) {
			return true
		}
	}
	return false
}

func resolvePublicIPs(ctx context.Context, host string, lookup lookupIPAddrFunc) ([]netip.Addr, error) {
	host = strings.TrimSuffix(strings.TrimSpace(host), ".")
	if host == "" {
		return nil, errors.New("target hostname is empty")
	}

	if literal, err := netip.ParseAddr(host); err == nil {
		literal = literal.Unmap()
		if !isPublicInternetIP(literal) {
			return nil, errors.New("target resolves to a non-public address")
		}
		return []netip.Addr{literal}, nil
	}
	if isLocalHostname(host) {
		return nil, errors.New("target hostname is local or non-public")
	}

	resolved, err := lookup(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("resolve target hostname: %w", err)
	}
	if len(resolved) == 0 {
		return nil, errors.New("target hostname returned no addresses")
	}

	addresses := make([]netip.Addr, 0, len(resolved))
	seen := make(map[netip.Addr]struct{}, len(resolved))
	for _, candidate := range resolved {
		if candidate.Zone != "" {
			return nil, errors.New("target resolves to a scoped address")
		}
		address, ok := netip.AddrFromSlice(candidate.IP)
		if !ok {
			return nil, errors.New("target returned an invalid address")
		}
		address = address.Unmap()
		// Reject the whole DNS answer set when any member is unsafe. Choosing a
		// public member from a mixed set would leave resolver-order and retry
		// behavior as a route to private infrastructure.
		if !isPublicInternetIP(address) {
			return nil, errors.New("target resolves to a non-public address")
		}
		if _, exists := seen[address]; exists {
			continue
		}
		seen[address] = struct{}{}
		addresses = append(addresses, address)
	}
	if len(addresses) == 0 {
		return nil, errors.New("target hostname returned no usable addresses")
	}
	return addresses, nil
}

func isPublicInternetIP(address netip.Addr) bool {
	if !address.IsValid() {
		return false
	}
	address = address.Unmap()
	// IPv6's IsGlobalUnicast includes large reserved/unallocated regions. The
	// public Internet currently uses 2000::/3; keeping the outbound boundary to
	// that allocation rejects deprecated site-local and other special space.
	if address.Is6() && !allocatedGlobalIPv6Prefix.Contains(address) {
		return false
	}
	if !address.IsGlobalUnicast() || address.IsPrivate() || address.IsLoopback() ||
		address.IsLinkLocalUnicast() || address.IsLinkLocalMulticast() ||
		address.IsMulticast() || address.IsUnspecified() {
		return false
	}
	for _, prefix := range blockedPublicTargetPrefixes {
		if prefix.Contains(address) {
			return false
		}
	}
	return true
}

func publicDialContext(
	ctx context.Context,
	network string,
	address string,
	lookup lookupIPAddrFunc,
	dial dialContextFunc,
) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("invalid outbound address: %w", err)
	}
	addresses, err := resolvePublicIPs(ctx, host, lookup)
	if err != nil {
		return nil, err
	}

	var dialErrors []error
	for _, target := range addresses {
		if network == "tcp4" && !target.Is4() {
			continue
		}
		if network == "tcp6" && !target.Is6() {
			continue
		}
		// Dial the exact validated address. Passing the hostname back to net.Dial
		// would perform a second, unvalidated DNS lookup and reopen rebinding.
		conn, err := dial(ctx, network, net.JoinHostPort(target.String(), port))
		if err == nil {
			return conn, nil
		}
		dialErrors = append(dialErrors, err)
	}
	if len(dialErrors) == 0 {
		return nil, errors.New("target has no address compatible with the requested network")
	}
	return nil, errors.Join(dialErrors...)
}

type publicOnlyTransport struct {
	transport *http.Transport
}

func (t *publicOnlyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req == nil || req.URL == nil {
		return nil, errors.New("outbound request URL is required")
	}
	// This syntax/hostname check runs for every request, including redirected
	// ones. DNS is deliberately enforced in DialContext so connection reuse
	// cannot create a validate-then-resolve gap.
	if _, err := parsePublicHTTPURL(req.URL.String()); err != nil {
		return nil, err
	}
	return t.transport.RoundTrip(req)
}

func newPublicHTTPClient(timeout time.Duration, maxRedirects int) *http.Client {
	return newPublicHTTPClientWithDeps(
		timeout,
		maxRedirects,
		defaultLookupIPAddr,
		(&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
	)
}

func newPublicHTTPClientWithDeps(
	timeout time.Duration,
	maxRedirects int,
	lookup lookupIPAddrFunc,
	dial dialContextFunc,
) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	// Never honor HTTP_PROXY/HTTPS_PROXY for attacker-controlled destinations.
	// A configured proxy could resolve or connect to an address outside this
	// process's validated network boundary.
	transport.Proxy = nil
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		return publicDialContext(ctx, network, address, lookup, dial)
	}

	client := &http.Client{
		Timeout:   timeout,
		Transport: &publicOnlyTransport{transport: transport},
	}
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) > maxRedirects {
			return errors.New("too many redirects")
		}
		if req == nil || req.URL == nil {
			return errors.New("redirect target URL is required")
		}
		// Validate the complete answer set at the redirect boundary. The dialer
		// repeats resolution immediately before connecting, which catches a DNS
		// answer that rebinds between these two points.
		if err := validatePublicURL(req.Context(), req.URL.String(), lookup); err != nil {
			return fmt.Errorf("unsafe redirect target: %w", err)
		}
		return nil
	}
	return client
}
