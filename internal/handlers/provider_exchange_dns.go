package handlers

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

type ProviderDNSReverificationStats struct {
	Leased           int
	Matched          int
	Failed           int
	Revoked          int
	CompletionErrors int
}

const (
	providerDNSChallengePrefix = "nhs-verification="
	providerDNSLookupTimeout   = 5 * time.Second
)

type providerTXTResolver interface {
	LookupTXT(context.Context, string) ([]string, error)
}

type netProviderTXTResolver struct {
	resolver *net.Resolver
}

func (r netProviderTXTResolver) LookupTXT(ctx context.Context, name string) ([]string, error) {
	resolver := r.resolver
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	return resolver.LookupTXT(ctx, name)
}

func providerChallengeCandidates(records []string) []string {
	seen := map[string]bool{}
	candidates := make([]string, 0, min(len(records), 8))
	for _, record := range records {
		record = strings.TrimSpace(record)
		if !strings.HasPrefix(record, providerDNSChallengePrefix) {
			continue
		}
		candidate := strings.TrimSpace(strings.TrimPrefix(record, providerDNSChallengePrefix))
		// Raw claim challenges are intentionally high-entropy URL-safe values.
		// The bounds also prevent oversized TXT answers entering comparisons.
		if len(candidate) < 32 || len(candidate) > 128 || seen[candidate] {
			continue
		}
		seen[candidate] = true
		candidates = append(candidates, candidate)
		if len(candidates) == 8 {
			break
		}
	}
	return candidates
}

// ReverifyDueProviderClaims leases due work from PostgreSQL before doing DNS.
// The lease model makes this safe across multiple NHS server instances. DNS
// errors produce an empty observation (a failed proof), never a successful one.
func (h *ProviderExchangeHandler) ReverifyDueProviderClaims(ctx context.Context, limit int) (ProviderDNSReverificationStats, error) {
	stats := ProviderDNSReverificationStats{}
	if h == nil || h.DB == nil || h.leaseDNSChecks == nil || h.completeDNSCheck == nil {
		return stats, errors.New("provider DNS reverification store is unavailable")
	}
	nowFn := h.dnsNow
	if nowFn == nil {
		nowFn = time.Now
	}
	now := nowFn().UTC()
	leases, err := h.leaseDNSChecks(h.DB, now, limit)
	if err != nil {
		return stats, fmt.Errorf("lease due provider DNS checks: %w", err)
	}
	stats.Leased = len(leases)
	resolver := h.TXTResolver
	if resolver == nil {
		resolver = netProviderTXTResolver{}
	}
	var firstCompletionErr error
	for _, lease := range leases {
		if err := ctx.Err(); err != nil {
			return stats, err
		}
		lookupCtx, cancel := context.WithTimeout(ctx, providerDNSLookupTimeout)
		records, lookupErr := resolver.LookupTXT(lookupCtx, lease.VerificationRecordName)
		cancel()
		if err := ctx.Err(); err != nil {
			// A process shutdown or caller cancellation is not domain evidence. Leave
			// the row leased so another instance can retry after lease expiry.
			return stats, err
		}
		candidates := []string(nil)
		if lookupErr == nil {
			candidates = providerChallengeCandidates(records)
		}
		result, completeErr := h.completeDNSCheck(
			h.DB, lease.ClaimID, lease.LeaseID, candidates, nowFn().UTC(),
		)
		if completeErr != nil {
			stats.CompletionErrors++
			if firstCompletionErr == nil {
				firstCompletionErr = fmt.Errorf("complete provider DNS check for claim %s: %w", lease.ClaimID, completeErr)
			}
			continue
		}
		if result.Matched {
			stats.Matched++
		} else {
			stats.Failed++
		}
		if result.Revoked {
			stats.Revoked++
		}
	}
	return stats, firstCompletionErr
}
