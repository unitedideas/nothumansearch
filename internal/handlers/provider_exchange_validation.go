package handlers

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/unitedideas/nothumansearch/internal/providerexchange"
)

const (
	providerExchangeMaxBodyBytes             int64 = 32 << 10
	providerActionURLMaximumBytes                  = providerexchange.ActionURLMaximumBytes
	providerActionURLAttributionReserveBytes       = providerexchange.ActionURLAttributionReserveBytes
	providerActionURLBaseMaximumBytes              = providerexchange.ActionURLBaseMaximumBytes
)

var (
	providerDomainLabelPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)
	providerEvidenceRefPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]{7,199}$`)
)

// decodeProviderJSON keeps every exchange write small, rejects misspelled or
// unexpected fields, and accepts exactly one JSON value. That makes the public
// privacy contract enforceable instead of merely documentary.
func decodeProviderJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	if r == nil || r.Body == nil {
		return errors.New("json body required")
	}
	if mediaType := strings.ToLower(strings.TrimSpace(strings.Split(r.Header.Get("Content-Type"), ";")[0])); mediaType != "application/json" {
		return errors.New("content-type must be application/json")
	}
	r.Body = http.MaxBytesReader(w, r.Body, providerExchangeMaxBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("exactly one json value required")
		}
		return err
	}
	return nil
}

func validProviderDomain(raw string) bool {
	domain := strings.ToLower(strings.TrimSpace(raw))
	domain = strings.TrimSuffix(domain, ".")
	if domain == "" || len(domain) > 253 || net.ParseIP(domain) != nil || !strings.Contains(domain, ".") {
		return false
	}
	for _, label := range strings.Split(domain, ".") {
		if len(label) > 63 || !providerDomainLabelPattern.MatchString(label) {
			return false
		}
	}
	return true
}

// normalizeProviderActionURL limits a claimed provider to its own HTTPS origin
// (or a subdomain). NHS never makes this request; the restriction prevents a
// verified claim from laundering an unrelated redirect or phishing target.
func normalizeProviderActionURL(raw, claimedDomain string) (string, error) {
	raw = strings.TrimSpace(raw)
	claimedDomain = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(claimedDomain), "."))
	if len(raw) == 0 || len(raw) > providerActionURLBaseMaximumBytes || !validProviderDomain(claimedDomain) {
		return "", errors.New("invalid action url or claimed domain")
	}
	parsed, err := url.Parse(raw)
	if err != nil || !strings.EqualFold(parsed.Scheme, "https") || parsed.Host == "" {
		return "", errors.New("action_url must be an absolute https url")
	}
	if parsed.User != nil || parsed.Fragment != "" || parsed.Opaque != "" {
		return "", errors.New("action_url cannot contain credentials, fragments, or opaque data")
	}
	if parsed.RawQuery != "" {
		return "", errors.New("action_url cannot contain provider query parameters")
	}
	host := strings.ToLower(strings.TrimSuffix(parsed.Hostname(), "."))
	if !validProviderDomain(host) || (host != claimedDomain && !strings.HasSuffix(host, "."+claimedDomain)) {
		return "", errors.New("action_url must use the verified provider domain or its subdomain")
	}
	if port := parsed.Port(); port != "" && port != "443" {
		return "", errors.New("action_url may only use the default https port")
	}
	parsed.Scheme = "https"
	parsed.Host = host
	if parsed.Path == "" {
		parsed.Path = "/"
	}
	normalized := parsed.String()
	if len(normalized) > providerActionURLBaseMaximumBytes {
		return "", errors.New("normalized action_url exceeds maximum length")
	}
	return normalized, nil
}

func actionURLWithAttribution(rawURL, token string) (string, error) {
	return providerexchange.ActionURLWithAttribution(rawURL, token)
}

func validProviderEvidenceReference(value string) bool {
	return providerEvidenceRefPattern.MatchString(strings.TrimSpace(value))
}

func constantTimeStringEqual(left, right string) bool {
	if len(left) != len(right) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(left), []byte(right)) == 1
}

func requestOriginAllowed(r *http.Request, baseURL string) bool {
	if r == nil {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(r.Header.Get("Sec-Fetch-Site")), "cross-site") {
		return false
	}
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return true
	}
	want, wantErr := url.Parse(baseURL)
	got, gotErr := url.Parse(origin)
	if wantErr != nil || gotErr != nil {
		return false
	}
	return strings.EqualFold(want.Scheme, got.Scheme) && strings.EqualFold(want.Host, got.Host)
}
