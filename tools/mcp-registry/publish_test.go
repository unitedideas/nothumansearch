package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func response(status int, v string) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(v)), Header: make(http.Header)}
}
func fixture(t *testing.T, failure string) (*http.Client, []byte, string, *int) {
	t.Helper()
	pub, key, e := ed25519.GenerateKey(rand.Reader)
	if e != nil {
		t.Fatal("fixture generation failed")
	}
	payload := []byte(`{"name":"ai.nothumansearch/search","title":"Not Human Search","version":"1.7.2","description":"Search agent-ready websites and APIs.","websiteUrl":"https://nothumansearch.ai","remotes":[{"type":"streamable-http","url":"https://nothumansearch.ai/mcp"}]}`)
	var target server
	_ = json.Unmarshal(payload, &target)
	published := false
	authCalls := 0
	c := &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/v0.1/validate":
			if failure == "validation" {
				return response(200, `{"valid":false}`), nil
			}
			return response(200, `{"valid":true}`), nil
		case "/v0.1/servers/ai.nothumansearch/search/versions/latest":
			current := target
			if !published {
				current.Version = "1.7.1"
				current.Description = "Old description"
			}
			if failure == "readback" && published {
				current.Description = "Unexpected description"
			}
			b, _ := json.Marshal(envelope{Server: current, Meta: map[string]struct {
				Status   string `json:"status"`
				IsLatest bool   `json:"isLatest"`
			}{"io.modelcontextprotocol.registry/official": {Status: "active", IsLatest: true}}})
			return response(200, string(b)), nil
		case "/.well-known/mcp-registry-auth":
			if failure == "proof" {
				return response(200, "v=MCPv1; k=ed25519; p=invalid"), nil
			}
			return response(200, "v=MCPv1; k=ed25519; p="+base64.StdEncoding.EncodeToString(pub)+"\n"), nil
		case "/v0.1/auth/http":
			authCalls++
			var a map[string]string
			if json.NewDecoder(r.Body).Decode(&a) != nil {
				t.Fatal("invalid authentication request")
			}
			sig, e := hex.DecodeString(a["signed_timestamp"])
			if e != nil || a["domain"] != "nothumansearch.ai" || !ed25519.Verify(pub, []byte(a["timestamp"]), sig) {
				t.Fatal("authentication signature not verified")
			}
			return response(200, `{"registry_token":"synthetic-session-token"}`), nil
		case "/v0.1/publish":
			if r.Header.Get("Authorization") != "Bearer synthetic-session-token" {
				t.Fatal("authentication was not used")
			}
			published = true
			if failure == "uncertain" {
				return response(503, "sensitive-error-body-must-not-leak"), nil
			}
			return response(201, `{}`), nil
		default:
			t.Fatalf("unexpected request attempted: %s", r.URL.Path)
			return nil, nil
		}
	})}
	return c, payload, hex.EncodeToString(key.Seed()), &authCalls
}
func TestPublishVerifiedReadback(t *testing.T) {
	c, p, key, auth := fixture(t, "")
	locks := 0
	r, e := publish(c, p, key, "1.7.1", func() error { locks++; return nil })
	if e != nil || !r.Published || r.State != "published" || locks != 1 || *auth != 1 {
		t.Fatal("publication did not complete through reservation, signature and readback")
	}
}
func TestPublishFailuresFailClosed(t *testing.T) {
	for _, kind := range []string{"validation", "proof", "readback", "uncertain", "missing-key"} {
		t.Run(kind, func(t *testing.T) {
			c, p, key, auth := fixture(t, kind)
			if kind == "missing-key" {
				key = ""
			}
			r, e := publish(c, p, key, "1.7.1", func() error { return nil })
			if e == nil || r.Published {
				t.Fatal("failure falsely reported publication")
			}
			if strings.Contains(e.Error(), "sensitive-error-body") || strings.Contains(e.Error(), key) && key != "" {
				t.Fatal("sensitive material appeared in error")
			}
			if (kind == "validation" || kind == "proof" || kind == "missing-key") && *auth != 0 {
				t.Fatal("authentication occurred after a failed prerequisite")
			}
		})
	}
}
func TestValidationDoesNotAuthenticate(t *testing.T) {
	c, p, _, auth := fixture(t, "")
	r, e := publish(c, p, "", "", nil)
	if e != nil || r.Published || r.State != "validated" || *auth != 0 {
		t.Fatal("validation changed publication state")
	}
}
func TestReservationStripsCredentialBindings(t *testing.T) {
	if os.PathSeparator != '/' {
		t.Skip("POSIX fixture")
	}
	dir := t.TempDir()
	marker := filepath.Join(dir, "calls")
	script := "#!/bin/sh\n[ -z \"$MCP_PRIVATE_KEY$NHS_MCP_REGISTRY_PRIVATE_KEY$GITHUB_TOKEN$GH_TOKEN$FLY_API_TOKEN\" ] || exit 91\nprintf 'hit\\n' >> '" + marker + "'\nif [ \"$3\" = show ]; then printf '%s' '{\"status\":\"claimed\",\"claimed_by\":\"fixture-owner\",\"dedupe_key\":\"nothumansearch:official-registry:1.7.2\"}'; fi\n"
	if os.WriteFile(filepath.Join(dir, "git"), []byte(script), 0700) != nil {
		t.Fatal("fixture setup failed")
	}
	t.Setenv("PATH", dir)
	for _, k := range []string{"MCP_PRIVATE_KEY", "NHS_MCP_REGISTRY_PRIVATE_KEY", "GITHUB_TOKEN", "GH_TOKEN", "FLY_API_TOKEN"} {
		t.Setenv(k, "synthetic-reference")
	}
	if e := reserved(dir, "fixture-owner", "1.7.2"); e != nil {
		t.Fatal("reservation command exposed a binding or rejected valid lock")
	}
	b, _ := os.ReadFile(marker)
	if strings.Count(string(b), "hit") != 2 {
		t.Fatal("fixture did not intercept both git calls")
	}
}

func TestUncertainWriteReconcilesBeforeRetry(t *testing.T) {
	c, p, key, auth := fixture(t, "uncertain")
	claims := 0
	if r, e := publish(c, p, key, "1.7.1", func() error { claims++; return nil }); e == nil || r.Published {
		t.Fatal("uncertain write was reported complete")
	}
	r, e := publish(c, p, "", "1.7.1", func() error { t.Fatal("existing publication requested another reservation"); return nil })
	if e != nil || r.State != "already-published" || !r.Published || *auth != 1 || claims != 1 {
		t.Fatal("uncertain publication was not reconciled without another send")
	}
}
