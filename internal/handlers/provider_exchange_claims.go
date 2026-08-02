package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/unitedideas/nothumansearch/internal/models"
)

func (h *ProviderExchangeHandler) Claims(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		account := h.requireAccount(w, r, false)
		if account == nil {
			return
		}
		claims, err := models.ListProviderClaims(h.DB, account.ID)
		if err != nil {
			status, message := providerExchangeStatus(err)
			providerWriteJSON(w, status, map[string]string{"error": message})
			return
		}
		providerWriteJSON(w, http.StatusOK, map[string]any{"claims": claims})
	case http.MethodPost:
		account := h.requireAccount(w, r, true)
		if account == nil {
			return
		}
		if !h.allowClaimMutation(w, r, account.ID) {
			return
		}
		var request struct {
			Domain string `json:"domain"`
		}
		if err := decodeProviderJSON(w, r, &request); err != nil {
			providerWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid claim request"})
			return
		}
		domain := models.NormalizeProviderDomain(request.Domain)
		if !validProviderDomain(domain) {
			providerWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "valid indexed domain required"})
			return
		}
		site, err := models.GetSiteByDomain(h.DB, domain)
		if err != nil || site == nil {
			providerWriteJSON(w, http.StatusNotFound, map[string]string{
				"error":      "domain is not in the Not Human Search index",
				"submit_url": h.BaseURL + "/#submit",
			})
			return
		}
		claim, rawChallenge, err := models.CreateProviderClaim(h.DB, account.ID, site.ID)
		if err != nil {
			status, message := providerExchangeStatus(err)
			providerWriteJSON(w, status, map[string]string{"error": message})
			return
		}
		providerWriteJSON(w, http.StatusCreated, providerClaimChallengeResponse(claim, rawChallenge))
	default:
		w.Header().Set("Allow", "GET, POST")
		providerWriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET or POST required"})
	}
}

func (h *ProviderExchangeHandler) ClaimAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		providerWriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
		return
	}
	account := h.requireAccount(w, r, true)
	if account == nil {
		return
	}
	if !h.allowClaimMutation(w, r, account.ID) {
		return
	}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/provider/claims/"), "/"), "/")
	if len(parts) < 2 || parts[0] == "" {
		providerWriteJSON(w, http.StatusNotFound, map[string]string{"error": "claim action not found"})
		return
	}
	claimID, action := parts[0], strings.Join(parts[1:], "/")
	switch action {
	case "verify":
		h.verifyClaim(w, r, account.ID, claimID)
	case "challenge":
		claim, rawChallenge, err := models.RotateProviderClaimChallenge(h.DB, account.ID, claimID)
		if err != nil {
			status, message := providerExchangeStatus(err)
			providerWriteJSON(w, status, map[string]string{"error": message})
			return
		}
		providerWriteJSON(w, http.StatusOK, providerClaimChallengeResponse(claim, rawChallenge))
	case "revoke":
		if err := models.RevokeProviderClaim(h.DB, account.ID, claimID); err != nil {
			status, message := providerExchangeStatus(err)
			providerWriteJSON(w, status, map[string]string{"error": message})
			return
		}
		providerWriteJSON(w, http.StatusOK, map[string]any{"revoked": true, "claim_id": claimID})
	case "keys/rotate":
		rawKey, key, err := models.RotateProviderAPIKey(h.DB, account.ID, claimID)
		if err != nil {
			status, message := providerExchangeStatus(err)
			providerWriteJSON(w, status, map[string]string{"error": message})
			return
		}
		providerWriteJSON(w, http.StatusOK, map[string]any{
			"provider_key":          rawKey,
			"provider_key_metadata": key,
			"provider_key_returned": true,
			"save_this_key_now":     true,
			"previous_keys_revoked": true,
		})
	default:
		providerWriteJSON(w, http.StatusNotFound, map[string]string{"error": "claim action not found"})
	}
}

func (h *ProviderExchangeHandler) allowClaimMutation(w http.ResponseWriter, r *http.Request, accountID int64) bool {
	if h.claimLimit == nil {
		h.claimLimit = newMCPDiscoveryRateLimiter(20, time.Hour)
	}
	now := timeNow()
	_, accountRetry, accountOK := h.claimLimit.allow(fmt.Sprintf("account:%d", accountID), now)
	_, networkRetry, networkOK := h.claimLimit.allow("network:"+submitHashIP(r), now)
	if accountOK && networkOK {
		return true
	}
	retry := accountRetry
	if networkRetry > retry {
		retry = networkRetry
	}
	w.Header().Set("Retry-After", fmt.Sprintf("%d", max(1, int(retry.Seconds()))))
	providerWriteJSON(w, http.StatusTooManyRequests, map[string]string{"error": "provider claim safety limit exceeded"})
	return false
}

func providerClaimChallengeResponse(claim *models.ProviderClaim, rawChallenge string) map[string]any {
	return map[string]any{
		"claim": claim,
		"dns_challenge": map[string]any{
			"record_type":   "TXT",
			"record_name":   claim.VerificationRecordName,
			"record_value":  providerDNSChallengePrefix + rawChallenge,
			"expires_at":    claim.ChallengeExpiresAt,
			"returned_once": true,
		},
		"ownership_freshness": providerOwnershipFreshness(claim),
		"verify_endpoint":     "/api/v1/provider/claims/" + claim.ID + "/verify",
	}
}

type providerOwnershipFreshnessResponse struct {
	ProofMethod                                     string     `json:"proof_method"`
	RecordMustRemainPublished                       bool       `json:"record_must_remain_published"`
	StoredChallengeMaterial                         string     `json:"stored_challenge_material"`
	RawDNSAnswersRetained                           bool       `json:"raw_dns_answers_retained"`
	AutomaticReverification                         bool       `json:"automatic_reverification"`
	RecheckIntervalSeconds                          int64      `json:"recheck_interval_seconds"`
	PaidActionsStopAfterConsecutiveFailures         int        `json:"paid_actions_stop_after_consecutive_failures"`
	PaidActionsStopWhenLastSuccessAgeReachesSeconds int64      `json:"paid_actions_stop_when_last_success_age_reaches_seconds"`
	LastSucceededAt                                 *time.Time `json:"last_succeeded_at"`
	NextCheckAt                                     *time.Time `json:"next_check_at"`
	ConsecutiveFailures                             int        `json:"consecutive_failures"`
}

func providerOwnershipFreshness(claim *models.ProviderClaim) providerOwnershipFreshnessResponse {
	response := providerOwnershipFreshnessResponse{
		ProofMethod:                                     "dns_txt",
		RecordMustRemainPublished:                       true,
		StoredChallengeMaterial:                         "sha256_hash_only",
		RawDNSAnswersRetained:                           false,
		AutomaticReverification:                         true,
		RecheckIntervalSeconds:                          int64(models.ProviderClaimDNSRecheckInterval / time.Second),
		PaidActionsStopAfterConsecutiveFailures:         models.ProviderClaimDNSFailureLimit,
		PaidActionsStopWhenLastSuccessAgeReachesSeconds: int64(models.ProviderClaimVerificationFreshness / time.Second),
	}
	if claim != nil {
		response.LastSucceededAt = claim.VerificationLastSucceededAt
		response.NextCheckAt = claim.VerificationNextCheckAt
		response.ConsecutiveFailures = claim.VerificationConsecutiveFailures
	}
	return response
}

func (h *ProviderExchangeHandler) verifyClaim(w http.ResponseWriter, r *http.Request, accountID int64, claimID string) {
	claim, err := models.GetProviderClaim(h.DB, accountID, claimID)
	if err != nil {
		status, message := providerExchangeStatus(err)
		providerWriteJSON(w, status, map[string]string{"error": message})
		return
	}
	repeatVerification := claim.Status == "verified"
	resolver := h.TXTResolver
	if resolver == nil {
		resolver = netProviderTXTResolver{}
	}
	ctx, cancel := context.WithTimeout(r.Context(), providerDNSLookupTimeout)
	defer cancel()
	records, err := resolver.LookupTXT(ctx, claim.VerificationRecordName)
	if err != nil {
		if repeatVerification {
			if _, recordErr := models.RecordProviderClaimDNSFailure(h.DB, accountID, claimID, timeNow().UTC()); recordErr != nil {
				status, message := providerExchangeStatus(recordErr)
				providerWriteJSON(w, status, map[string]string{"error": message})
				return
			}
		}
		providerWriteJSON(w, http.StatusConflict, map[string]string{"error": "DNS TXT challenge not found"})
		return
	}
	var verified *models.ProviderClaim
	for _, candidate := range providerChallengeCandidates(records) {
		verified, err = models.VerifyProviderClaim(h.DB, accountID, claimID, candidate)
		if err == nil {
			break
		}
		if !errors.Is(err, models.ErrProviderChallengeMismatch) {
			status, message := providerExchangeStatus(err)
			providerWriteJSON(w, status, map[string]string{"error": message})
			return
		}
	}
	if verified == nil {
		if repeatVerification {
			if _, recordErr := models.RecordProviderClaimDNSFailure(h.DB, accountID, claimID, timeNow().UTC()); recordErr != nil {
				status, message := providerExchangeStatus(recordErr)
				providerWriteJSON(w, status, map[string]string{"error": message})
				return
			}
		}
		providerWriteJSON(w, http.StatusConflict, map[string]string{"error": "DNS TXT challenge did not match"})
		return
	}
	rawKey, key, keyErr := models.CreateProviderAPIKey(h.DB, accountID, claimID)
	if keyErr != nil {
		if errors.Is(keyErr, models.ErrProviderAPIKeyExists) {
			// Concurrent verification already issued the one active key. Never
			// rotate implicitly: doing so could invalidate the winning response
			// before its caller has saved the returned-once credential.
			providerWriteJSON(w, http.StatusOK, map[string]any{
				"claim":                 verified,
				"verified":              true,
				"provider_key_returned": false,
				"key_endpoint":          "/api/v1/provider/claims/" + claimID + "/keys/rotate",
				"ownership_freshness":   providerOwnershipFreshness(verified),
			})
			return
		}
		status, message := providerExchangeStatus(keyErr)
		providerWriteJSON(w, status, map[string]string{"error": message})
		return
	}
	providerWriteJSON(w, http.StatusOK, map[string]any{
		"claim":                 verified,
		"verified":              true,
		"provider_key":          rawKey,
		"provider_key_metadata": key,
		"provider_key_returned": true,
		"save_this_key_now":     true,
		"ownership_freshness":   providerOwnershipFreshness(verified),
	})
}

// Kept as a seam for deterministic rate-limit tests without altering the
// timestamp used by database state transitions.
var timeNow = func() time.Time { return time.Now() }
