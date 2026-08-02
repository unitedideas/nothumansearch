package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestFailClosedLoginRejectsBeforeCreatingCredentialWithoutEmail(t *testing.T) {
	t.Setenv("RESEND_API_KEY", "")
	auth := NewAuthService(nil, "https://nothumansearch.ai")
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader("email=owner%40example.com"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	auth.FailClosedLogin(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503; body=%s", rr.Code, rr.Body.String())
	}
	assertPrivateAuthHeaders(t, rr)
	if strings.Contains(rr.Body.String(), "token=") || strings.Contains(rr.Body.String(), "owner@example.com") {
		t.Fatalf("fail-closed response exposed credential or account detail: %q", rr.Body.String())
	}
}

func TestAuthRecipientHashNormalizesAndDoesNotRetainRecipient(t *testing.T) {
	first := authNormalizedRecipientHash(" Owner@Example.COM ")
	second := authNormalizedRecipientHash("owner@example.com")
	if first == "" || first != second {
		t.Fatalf("normalized hashes differ: %q vs %q", first, second)
	}
	if len(first) != 64 {
		t.Fatalf("recipient digest length = %d, want 64 hex characters", len(first))
	}
	if strings.Contains(first, "owner") || strings.Contains(first, "example") || strings.Contains(first, "@") {
		t.Fatalf("recipient digest retained source detail: %q", first)
	}
}

func TestAuthMagicLinkThrottleUsesRollingHour(t *testing.T) {
	start := time.Unix(10_000, 0)
	limiter := newAuthMagicLinkThrottle(2, 2, time.Hour, 10)
	ipHash := authSHA256("203.0.113.10")
	recipientHash := authNormalizedRecipientHash("person@example.com")

	if _, ok := limiter.allow(ipHash, recipientHash, start); !ok {
		t.Fatal("first attempt was rejected")
	}
	if _, ok := limiter.allow(ipHash, recipientHash, start.Add(30*time.Minute)); !ok {
		t.Fatal("second attempt was rejected")
	}
	retry, ok := limiter.allow(ipHash, recipientHash, start.Add(45*time.Minute))
	if ok {
		t.Fatal("third attempt inside rolling hour was allowed")
	}
	if retry != 15*time.Minute {
		t.Fatalf("retry = %s, want 15m", retry)
	}
	if _, ok := limiter.allow(ipHash, recipientHash, start.Add(time.Hour)); !ok {
		t.Fatal("attempt at the exact rolling-window boundary was rejected")
	}
}

func TestAuthMagicLinkThrottleIsConcurrencySafeAndExact(t *testing.T) {
	limiter := newAuthMagicLinkThrottle(7, 7, time.Hour, 10)
	now := time.Unix(20_000, 0)
	ipHash := authSHA256("203.0.113.20")
	recipientHash := authNormalizedRecipientHash("parallel@example.com")
	var allowed atomic.Int64
	var wg sync.WaitGroup

	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, ok := limiter.allow(ipHash, recipientHash, now); ok {
				allowed.Add(1)
			}
		}()
	}
	wg.Wait()
	if got := allowed.Load(); got != 7 {
		t.Fatalf("allowed = %d, want 7", got)
	}
}

func TestAuthMagicLinkThrottleBoundsHashedBuckets(t *testing.T) {
	limiter := newAuthMagicLinkThrottle(2, 2, time.Hour, 2)
	now := time.Unix(30_000, 0)
	allowed := 0
	for i := range 20 {
		rawRecipient := fmt.Sprintf("person-%d@example.com", i)
		if _, ok := limiter.allow(authSHA256(fmt.Sprintf("ip-%d", i)), authNormalizedRecipientHash(rawRecipient), now.Add(time.Duration(i)*time.Second)); ok {
			allowed++
		}
	}
	if len(limiter.ip.buckets) > 2 || len(limiter.recipients.buckets) > 2 {
		t.Fatalf("bucket bound exceeded: ip=%d recipient=%d", len(limiter.ip.buckets), len(limiter.recipients.buckets))
	}
	if allowed != 2 {
		t.Fatalf("allowed unique buckets at capacity = %d, want 2 (new buckets must fail closed)", allowed)
	}
	for key := range limiter.recipients.buckets {
		if strings.Contains(key, "person") || strings.Contains(key, "example") || strings.Contains(key, "@") {
			t.Fatalf("recipient bucket retained raw recipient: %q", key)
		}
	}
}

func TestAuthLoginGuardReturnsSameGeneric429ForIPAndRecipientLimits(t *testing.T) {
	start := time.Unix(40_000, 0)
	ready := func() error { return nil }
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })

	ipLimiter := newAuthMagicLinkThrottle(1, 10, time.Hour, 20)
	ipGuard := authLoginGuard{next: next, throttle: ipLimiter, now: func() time.Time { return start }, emailReady: ready}
	firstIP := authLoginRequest("same-ip-first@example.com", "203.0.113.40:1000")
	ipGuard.ServeHTTP(httptest.NewRecorder(), firstIP)
	ipLimited := httptest.NewRecorder()
	ipGuard.ServeHTTP(ipLimited, authLoginRequest("same-ip-second@example.com", "203.0.113.40:2000"))

	recipientLimiter := newAuthMagicLinkThrottle(10, 1, time.Hour, 20)
	recipientGuard := authLoginGuard{next: next, throttle: recipientLimiter, now: func() time.Time { return start }, emailReady: ready}
	recipientGuard.ServeHTTP(httptest.NewRecorder(), authLoginRequest("same-recipient@example.com", "203.0.113.41:1000"))
	recipientLimited := httptest.NewRecorder()
	recipientGuard.ServeHTTP(recipientLimited, authLoginRequest(" SAME-RECIPIENT@EXAMPLE.COM ", "203.0.113.42:1000"))

	if ipLimited.Code != http.StatusTooManyRequests || recipientLimited.Code != http.StatusTooManyRequests {
		t.Fatalf("statuses = ip %d recipient %d, want both 429", ipLimited.Code, recipientLimited.Code)
	}
	if ipLimited.Body.String() != recipientLimited.Body.String() || ipLimited.Header().Get("Retry-After") != recipientLimited.Header().Get("Retry-After") {
		t.Fatalf("limiter responses differ: ip=(%q,%q) recipient=(%q,%q)",
			ipLimited.Body.String(), ipLimited.Header().Get("Retry-After"),
			recipientLimited.Body.String(), recipientLimited.Header().Get("Retry-After"))
	}
	if strings.Contains(strings.ToLower(ipLimited.Body.String()), "email") || strings.Contains(strings.ToLower(ipLimited.Body.String()), "account") {
		t.Fatalf("429 disclosed limiter/account dimension: %q", ipLimited.Body.String())
	}
	assertPrivateAuthHeaders(t, ipLimited)
}

func TestAuthLoginGuardRejectsOversizedBodyBeforeDownstream(t *testing.T) {
	var calls int
	guard := authLoginGuard{
		next:       http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls++ }),
		throttle:   newAuthMagicLinkThrottle(10, 10, time.Hour, 10),
		now:        func() time.Time { return time.Unix(50_000, 0) },
		emailReady: func() error { return nil },
	}
	req := authLoginRequest("person@example.com"+strings.Repeat("x", int(authLoginMaxBodyBytes)), "203.0.113.50:1000")
	rr := httptest.NewRecorder()
	guard.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
	if calls != 0 {
		t.Fatalf("downstream calls = %d, want 0", calls)
	}
}

func TestVerifyNoStoreRedirectsRenderedFailureToCleanLogin(t *testing.T) {
	tokenURL := "/auth/verify?token=credential-value"
	req := httptest.NewRequest(http.MethodGet, tokenURL, nil)
	rr := httptest.NewRecorder()
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, "invalid token credential-value")
	})

	serveAuthVerifyNoStore(rr, req, next)

	if rr.Code != http.StatusSeeOther || rr.Header().Get("Location") != "/login" {
		t.Fatalf("response = %d location %q, want 303 /login", rr.Code, rr.Header().Get("Location"))
	}
	assertPrivateAuthHeaders(t, rr)
	if strings.Contains(rr.Body.String(), "credential-value") || strings.Contains(rr.Header().Get("Location"), "token") {
		t.Fatalf("verification failure retained credential-bearing detail: headers=%v body=%q", rr.Header(), rr.Body.String())
	}
}

func TestVerifyNoStorePassesThroughSuccessCookieAndRedirect(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/auth/verify?token=credential-value", nil)
	rr := httptest.NewRecorder()
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Add("Set-Cookie", "nhs_session=session-value; HttpOnly; Secure")
		w.Header().Set("Location", "/")
		w.WriteHeader(http.StatusSeeOther)
	})

	serveAuthVerifyNoStore(rr, req, next)

	if rr.Code != http.StatusSeeOther || rr.Header().Get("Location") != "/" {
		t.Fatalf("response = %d location %q, want 303 /", rr.Code, rr.Header().Get("Location"))
	}
	if !strings.Contains(rr.Header().Get("Set-Cookie"), "nhs_session=session-value") {
		t.Fatalf("session cookie not preserved: %q", rr.Header().Get("Set-Cookie"))
	}
	assertPrivateAuthHeaders(t, rr)
}

func authLoginRequest(email, remoteAddr string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader("email="+email))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.RemoteAddr = remoteAddr
	return req
}

func assertPrivateAuthHeaders(t *testing.T, rr *httptest.ResponseRecorder) {
	t.Helper()
	if got := rr.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}
	if got := rr.Header().Get("Referrer-Policy"); got != "no-referrer" {
		t.Fatalf("Referrer-Policy = %q, want no-referrer", got)
	}
}
