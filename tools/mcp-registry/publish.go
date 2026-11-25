package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

const registry = "https://registry.modelcontextprotocol.io/v0.1"
const serverName = "ai.nothumansearch/search"
const proofURL = "https://nothumansearch.ai/.well-known/mcp-registry-auth"

type server struct {
	Title       string `json:"title"`
	WebsiteURL  string `json:"websiteUrl"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Remotes     []struct {
		Type string `json:"type"`
		URL  string `json:"url"`
	} `json:"remotes"`
}
type envelope struct {
	Server server `json:"server"`
	Meta   map[string]struct {
		Status   string `json:"status"`
		IsLatest bool   `json:"isLatest"`
	} `json:"_meta"`
}
type receipt struct {
	State     string `json:"state"`
	Name      string `json:"name"`
	Version   string `json:"version"`
	Published bool   `json:"publication_verified"`
}

func request(c *http.Client, method, url string, body []byte, token string) ([]byte, error) {
	r, e := http.NewRequest(method, url, bytes.NewReader(body))
	if e != nil {
		return nil, errors.New("invalid registry request")
	}
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json")
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	f, e := c.Do(r)
	if e != nil {
		return nil, errors.New("registry transport failed; reconcile version before retry")
	}
	defer f.Body.Close()
	b, e := io.ReadAll(io.LimitReader(f.Body, 128*1024+1))
	if e != nil || len(b) > 128*1024 {
		return nil, errors.New("registry response exceeded bound")
	}
	if f.StatusCode < 200 || f.StatusCode >= 300 {
		return nil, fmt.Errorf("registry request returned HTTP %d; reconcile version before retry", f.StatusCode)
	}
	return b, nil
}
func latest(c *http.Client) (envelope, error) {
	var e envelope
	b, err := request(c, "GET", registry+"/servers/ai.nothumansearch%2Fsearch/versions/latest", nil, "")
	if err != nil {
		return e, err
	}
	err = json.Unmarshal(b, &e)
	return e, err
}
func same(a, b server) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return bytes.Equal(x, y)
}
func reserved(repo, owner, version string) error {
	if repo == "" || owner == "" {
		return errors.New("shared state repository and claim owner are required")
	}
	env := []string{}
	for _, v := range os.Environ() {
		k := strings.SplitN(v, "=", 2)[0]
		if k != "MCP_PRIVATE_KEY" && k != "NHS_MCP_REGISTRY_PRIVATE_KEY" && k != "GITHUB_TOKEN" && k != "GH_TOKEN" && k != "FLY_API_TOKEN" {
			env = append(env, v)
		}
	}
	run := func(args ...string) ([]byte, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, "git", append([]string{"-C", repo}, args...)...)
		cmd.Env = env
		return cmd.Output()
	}
	if _, e := run("fetch", "origin", "main"); e != nil {
		return errors.New("shared publication reservation fetch failed")
	}
	b, e := run("show", "origin/main:public-action-locks/agentic-discovery/nhs-official-registry-"+version+".json")
	if e != nil {
		return errors.New("shared publication reservation is missing")
	}
	var lock struct {
		Status string `json:"status"`
		Owner  string `json:"claimed_by"`
		Key    string `json:"dedupe_key"`
	}
	if json.Unmarshal(b, &lock) != nil || lock.Status != "claimed" || lock.Owner != owner || lock.Key != "nothumansearch:official-registry:"+version {
		return errors.New("shared publication reservation does not match")
	}
	return nil
}
func publish(c *http.Client, payload []byte, key, expected string, checkLock func() error) (receipt, error) {
	var s server
	out := receipt{}
	if json.Unmarshal(payload, &s) != nil || s.Name != serverName || !regexp.MustCompile(`^\d+\.\d+\.\d+$`).MatchString(s.Version) || len(s.Description) == 0 || len(s.Description) > 100 || len(s.Remotes) != 1 || s.Remotes[0].Type != "streamable-http" || s.Remotes[0].URL != "https://nothumansearch.ai/mcp" {
		return out, errors.New("manifest does not match NHS registry scope")
	}
	out = receipt{State: "validated", Name: s.Name, Version: s.Version}
	validation, e := request(c, "POST", registry+"/validate", payload, "")
	if e != nil {
		return out, e
	}
	var valid struct {
		Valid bool `json:"valid"`
	}
	if json.Unmarshal(validation, &valid) != nil || !valid.Valid {
		return out, errors.New("registry validation rejected manifest")
	}
	if checkLock == nil {
		return out, nil
	}
	current, e := latest(c)
	if e != nil {
		return out, e
	}
	official := current.Meta["io.modelcontextprotocol.registry/official"]
	if current.Server.Version == s.Version {
		if same(current.Server, s) && official.Status == "active" && official.IsLatest {
			out.State = "already-published"
			out.Published = true
			return out, nil
		}
		return out, errors.New("existing registry version differs; do not overwrite")
	}
	if expected == "" || current.Server.Version != expected || official.Status != "active" || !official.IsLatest {
		return out, errors.New("live registry baseline changed; reconcile before publishing")
	}
	if e = checkLock(); e != nil {
		return out, e
	}
	seed, e := hex.DecodeString(key)
	if e != nil || len(seed) != ed25519.SeedSize {
		return out, errors.New("named signing reference is unavailable or invalid")
	}
	private := ed25519.NewKeyFromSeed(seed)
	proof, e := request(c, "GET", proofURL, nil, "")
	if e != nil {
		return out, e
	}
	match := false
	for _, p := range strings.Split(string(proof), ";") {
		if strings.TrimSpace(p) == "p="+base64.StdEncoding.EncodeToString(private.Public().(ed25519.PublicKey)) {
			match = true
		}
	}
	if !match {
		return out, errors.New("named signing reference does not match public domain proof")
	}
	timestamp := time.Now().UTC().Format(time.RFC3339)
	auth, _ := json.Marshal(map[string]string{"domain": "nothumansearch.ai", "timestamp": timestamp, "signed_timestamp": hex.EncodeToString(ed25519.Sign(private, []byte(timestamp)))})
	b, e := request(c, "POST", registry+"/auth/http", auth, "")
	if e != nil {
		return out, e
	}
	var token struct {
		Token string `json:"registry_token"`
	}
	if json.Unmarshal(b, &token) != nil || token.Token == "" {
		return out, errors.New("registry authentication response invalid")
	}
	out.State = "dispatching"
	if _, e = request(c, "POST", registry+"/publish", payload, token.Token); e != nil {
		return out, e
	}
	after, e := latest(c)
	if e != nil {
		return out, e
	}
	official = after.Meta["io.modelcontextprotocol.registry/official"]
	if !same(after.Server, s) || official.Status != "active" || !official.IsLatest {
		return out, errors.New("publication readback did not match")
	}
	out.State = "published"
	out.Published = true
	return out, nil
}
func main() {
	path := flag.String("manifest", "tools/mcp-registry/server.json", "manifest path")
	send := flag.Bool("publish", false, "publish after shared reservation and baseline checks")
	baseline := flag.String("expected-current-version", "", "verified live version")
	repo := flag.String("state-repo", "", "existing private shared-state worktree")
	owner := flag.String("claim-owner", "", "existing reservation owner")
	flag.Parse()
	payload, e := os.ReadFile(*path)
	if e != nil {
		fmt.Fprintln(os.Stderr, "registry_publish_error: manifest unavailable")
		os.Exit(1)
	}
	var check func() error
	if *send {
		var s server
		_ = json.Unmarshal(payload, &s)
		check = func() error { return reserved(*repo, *owner, s.Version) }
	}
	c := &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	out, e := publish(c, payload, os.Getenv("MCP_PRIVATE_KEY"), *baseline, check)
	_ = json.NewEncoder(os.Stdout).Encode(out)
	if e != nil {
		fmt.Fprintln(os.Stderr, "registry_publish_error:", e)
		os.Exit(1)
	}
}
