package handlers

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/unitedideas/nothumansearch/internal/email"
)

const (
	authLoginIPHourlyLimit              = 20
	authLoginRecipientHourlyLimit       = 5
	authLoginMaxBuckets                 = 16_384
	authLoginMaxBodyBytes         int64 = 8 << 10
)

var defaultAuthMagicLinkThrottle = newAuthMagicLinkThrottle(
	authLoginIPHourlyLimit,
	authLoginRecipientHourlyLimit,
	time.Hour,
	authLoginMaxBuckets,
)

// AuthService cannot retain request-scoped recipient data. The process-local
// throttle below stores only an edge-normalized IP hash and a SHA-256 digest of
// the normalized recipient. Buckets are both time-pruned and capacity-bounded.
type authMagicLinkThrottle struct {
	mu         sync.Mutex
	ip         authRollingBuckets
	recipients authRollingBuckets
}

type authRollingBuckets struct {
	limit      int
	window     time.Duration
	maxBuckets int
	buckets    map[string]authRollingBucket
}

type authRollingBucket struct {
	attempts []time.Time
}

func newAuthMagicLinkThrottle(ipLimit, recipientLimit int, window time.Duration, maxBuckets int) *authMagicLinkThrottle {
	if ipLimit < 1 {
		ipLimit = 1
	}
	if recipientLimit < 1 {
		recipientLimit = 1
	}
	if window <= 0 {
		window = time.Hour
	}
	if maxBuckets < 1 {
		maxBuckets = 1
	}
	return &authMagicLinkThrottle{
		ip: authRollingBuckets{
			limit:      ipLimit,
			window:     window,
			maxBuckets: maxBuckets,
			buckets:    make(map[string]authRollingBucket),
		},
		recipients: authRollingBuckets{
			limit:      recipientLimit,
			window:     window,
			maxBuckets: maxBuckets,
			buckets:    make(map[string]authRollingBucket),
		},
	}
}

// allow applies both limits under one lock. A recipient-limited request still
// consumes its network allowance, which prevents cheap probing of whether a
// particular recipient bucket is full. A network-limited request does not burn
// another recipient's allowance.
func (l *authMagicLinkThrottle) allow(ipHash, recipientHash string, now time.Time) (time.Duration, bool) {
	if l == nil {
		return time.Hour, false
	}
	if ipHash == "" {
		ipHash = authSHA256("unknown")
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	ipRetry, ipBlocked := l.ip.retryAfter(ipHash, now)
	recipientRetry, recipientBlocked := time.Duration(0), false
	if recipientHash != "" {
		recipientRetry, recipientBlocked = l.recipients.retryAfter(recipientHash, now)
	}

	if ipBlocked || recipientBlocked {
		if !ipBlocked {
			l.ip.record(ipHash, now)
		}
		if recipientRetry > ipRetry {
			ipRetry = recipientRetry
		}
		if ipRetry <= 0 {
			ipRetry = time.Second
		}
		return ipRetry, false
	}

	l.ip.record(ipHash, now)
	if recipientHash != "" {
		l.recipients.record(recipientHash, now)
	}
	return 0, true
}

func (b *authRollingBuckets) retryAfter(key string, now time.Time) (time.Duration, bool) {
	bucket, exists := b.buckets[key]
	if !exists {
		if len(b.buckets) < b.maxBuckets {
			return 0, false
		}
		b.pruneExpired(now)
		if len(b.buckets) < b.maxBuckets {
			return 0, false
		}
		return b.capacityRetryAfter(now), true
	}
	bucket = b.pruned(bucket, now)
	if len(bucket.attempts) == 0 {
		delete(b.buckets, key)
		return 0, false
	}
	b.buckets[key] = bucket
	if len(bucket.attempts) < b.limit {
		return 0, false
	}
	retry := bucket.attempts[0].Add(b.window).Sub(now)
	if retry < time.Second {
		retry = time.Second
	}
	return retry, true
}

func (b *authRollingBuckets) record(key string, now time.Time) {
	bucket := b.buckets[key]
	bucket = b.pruned(bucket, now)
	bucket.attempts = append(bucket.attempts, now)
	b.buckets[key] = bucket
}

func (b *authRollingBuckets) pruned(bucket authRollingBucket, now time.Time) authRollingBucket {
	cutoff := now.Add(-b.window)
	firstLive := 0
	for firstLive < len(bucket.attempts) && !bucket.attempts[firstLive].After(cutoff) {
		firstLive++
	}
	if firstLive > 0 {
		live := make([]time.Time, len(bucket.attempts)-firstLive)
		copy(live, bucket.attempts[firstLive:])
		bucket.attempts = live
	}
	return bucket
}

func (b *authRollingBuckets) pruneExpired(now time.Time) {
	for key, bucket := range b.buckets {
		bucket = b.pruned(bucket, now)
		if len(bucket.attempts) == 0 {
			delete(b.buckets, key)
			continue
		}
		b.buckets[key] = bucket
	}
}

func (b *authRollingBuckets) capacityRetryAfter(now time.Time) time.Duration {
	var retry time.Duration
	for _, bucket := range b.buckets {
		if len(bucket.attempts) == 0 {
			continue
		}
		candidate := bucket.attempts[0].Add(b.window).Sub(now)
		if retry == 0 || candidate < retry {
			retry = candidate
		}
	}
	if retry < time.Second {
		return time.Second
	}
	return retry
}

func authNormalizedRecipientHash(raw string) string {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	if normalized == "" || !strings.Contains(normalized, "@") {
		return ""
	}
	return authSHA256(normalized)
}

func authSHA256(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

type authLoginGuard struct {
	next       http.Handler
	throttle   *authMagicLinkThrottle
	now        func() time.Time
	emailReady func() error
}

func (g authLoginGuard) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	setAuthPrivateResponseHeaders(w.Header())
	if r.Method != http.MethodPost {
		g.next.ServeHTTP(w, r)
		return
	}

	if g.emailReady == nil || g.emailReady() != nil {
		log.Printf("auth: login unavailable because email transport is not configured")
		http.Error(w, "Sign-in email is temporarily unavailable.", http.StatusServiceUnavailable)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, authLoginMaxBodyBytes)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid sign-in request.", http.StatusBadRequest)
		return
	}

	now := time.Now()
	if g.now != nil {
		now = g.now()
	}
	recipientHash := authNormalizedRecipientHash(r.Form.Get("email"))
	retry, ok := g.throttle.allow(submitHashIP(r), recipientHash, now)
	if !ok {
		retrySeconds := int64((retry + time.Second - 1) / time.Second)
		if retrySeconds < 1 {
			retrySeconds = 1
		}
		w.Header().Set("Retry-After", fmt.Sprintf("%d", retrySeconds))
		http.Error(w, "Too many sign-in attempts. Try again later.", http.StatusTooManyRequests)
		return
	}

	g.next.ServeHTTP(w, r)
}

// FailClosedLogin prevents issuance of a magic-link credential when email is
// unavailable and applies rolling network + recipient throttles before Login
// can create or send a credential.
func (a *AuthService) FailClosedLogin(w http.ResponseWriter, r *http.Request) {
	guard := authLoginGuard{
		next:     http.HandlerFunc(a.Login),
		throttle: defaultAuthMagicLinkThrottle,
		now:      time.Now,
		emailReady: func() error {
			_, err := email.NewClientFromEnv()
			return err
		},
	}
	guard.ServeHTTP(w, r)
}

// VerifyNoStore keeps the credential-bearing verification URL out of caches
// and referrers. Verify's success redirect and session cookie pass through;
// any rendered failure is replaced with a clean redirect to /login.
func (a *AuthService) VerifyNoStore(w http.ResponseWriter, r *http.Request) {
	serveAuthVerifyNoStore(w, r, http.HandlerFunc(a.Verify))
}

func serveAuthVerifyNoStore(w http.ResponseWriter, r *http.Request, next http.Handler) {
	buffer := newAuthResponseBuffer()
	next.ServeHTTP(buffer, r)

	if buffer.status >= http.StatusMultipleChoices && buffer.status < http.StatusBadRequest && buffer.header.Get("Location") != "" {
		copyAuthResponseHeaders(w.Header(), buffer.header)
		setAuthPrivateResponseHeaders(w.Header())
		w.WriteHeader(buffer.status)
		_, _ = w.Write(buffer.body.Bytes())
		return
	}

	setAuthPrivateResponseHeaders(w.Header())
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func setAuthPrivateResponseHeaders(header http.Header) {
	header.Set("Cache-Control", "no-store")
	header.Set("Referrer-Policy", "no-referrer")
	header.Set("X-Robots-Tag", "noindex, nofollow")
}

func copyAuthResponseHeaders(dst, src http.Header) {
	for key, values := range src {
		dst.Del(key)
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

type authResponseBuffer struct {
	header      http.Header
	body        bytes.Buffer
	status      int
	wroteHeader bool
}

func newAuthResponseBuffer() *authResponseBuffer {
	return &authResponseBuffer{header: make(http.Header), status: http.StatusOK}
}

func (b *authResponseBuffer) Header() http.Header { return b.header }

func (b *authResponseBuffer) WriteHeader(status int) {
	if b.wroteHeader {
		return
	}
	b.status = status
	b.wroteHeader = true
}

func (b *authResponseBuffer) Write(value []byte) (int, error) {
	if !b.wroteHeader {
		b.WriteHeader(http.StatusOK)
	}
	return b.body.Write(value)
}
