package providerexchange

import (
	"errors"
	"net/url"
	"strings"
)

// ActionURLMaximumBytes bounds the complete bearer-bearing provider URL
// returned to an agent. Keeping the final formatter in this package lets the
// storage transaction preflight the exact same representation before commit.
const (
	ActionURLMaximumBytes            = 2048
	ActionURLAttributionReserveBytes = 512
	ActionURLBaseMaximumBytes        = ActionURLMaximumBytes - ActionURLAttributionReserveBytes
)

// ActionURLWithAttribution adds the signed NHS capability without replacing
// any provider query values and rejects a second reserved parameter.
func ActionURLWithAttribution(rawURL, token string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || strings.TrimSpace(token) == "" {
		return "", errors.New("invalid action url or attribution token")
	}
	query := parsed.Query()
	if query.Has("nhs_attribution") {
		return "", errors.New("action url already contains reserved attribution parameter")
	}
	query.Set("nhs_attribution", token)
	parsed.RawQuery = query.Encode()
	attributed := parsed.String()
	if len(attributed) > ActionURLMaximumBytes {
		return "", errors.New("attributed action url exceeds maximum length")
	}
	return attributed, nil
}
