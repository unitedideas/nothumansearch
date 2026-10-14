package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"

	"github.com/unitedideas/nothumansearch/internal/models"
)

const (
	freeSearchHourlyLimit      = 240
	freeActiveProbeHourlyLimit = 20
	prioritySearchHourlyLimit  = 5000
	priorityActiveProbeLimit   = 100
	freeCheckHourlyLimit       = 10
	priorityCheckHourlyLimit   = 100
	freeRateLimitTier          = "free"
	priorityRateLimitTier      = "priority"
)

// requestRateAccess describes an abuse-control bucket, not search
// entitlement. Discovery always remains available at the free tier. An active
// legacy API key only raises the temporary hourly ceiling while its monthly
// priority allocation remains; an invalid or exhausted key silently falls back
// to the free safety limit rather than producing an auth or payment error.
type requestRateAccess struct {
	tier             string
	bucket           string
	key              *models.APIKey
	monthlyLimit     int
	monthlyRemaining int
	reservationID    int64
}

func resolveRequestRateAccess(db *sql.DB, r *http.Request) requestRateAccess {
	access := requestRateAccess{
		tier:   freeRateLimitTier,
		bucket: "unknown",
	}
	if r == nil {
		return access
	}
	access.bucket = submitHashIP(r)
	if db == nil {
		return access
	}
	rawKey := extractAPIKey(r)
	if rawKey == "" {
		return access
	}
	key, err := models.ResolveAPIKey(db, rawKey)
	if err != nil || key == nil || key.MonthlyLimit < 1 {
		return access
	}
	used, err := models.CurrentMonthUsage(db, key, "")
	if err != nil || used >= key.MonthlyLimit {
		return access
	}
	access.tier = priorityRateLimitTier
	access.bucket = "api-key:" + strconv.FormatInt(key.ID, 10)
	access.key = key
	access.monthlyLimit = key.MonthlyLimit
	access.monthlyRemaining = key.MonthlyLimit - used
	return access
}

func freeRequestRateAccess(r *http.Request) requestRateAccess {
	access := requestRateAccess{tier: freeRateLimitTier, bucket: "unknown"}
	if r != nil {
		access.bucket = submitHashIP(r)
	}
	return access
}

func (a requestRateAccess) setHeaders(w http.ResponseWriter) {
	w.Header().Set("X-RateLimit-Tier", a.tier)
	if a.key == nil {
		return
	}
	w.Header().Set("X-Priority-Quota-Limit", strconv.Itoa(a.monthlyLimit))
	w.Header().Set("X-Priority-Quota-Remaining", strconv.Itoa(a.monthlyRemaining))
}

// reservePriorityUnit intentionally stores no network or user-agent fields.
// When accounting cannot reserve the unit atomically, callers must retry this
// request through their free limiter instead of granting unmetered priority.
func reservePriorityUnit(db *sql.DB, access requestRateAccess, surface, method, path, tool string) (requestRateAccess, bool) {
	if access.key == nil {
		return access, true
	}
	reservationID, remaining, reserved, err := models.ReservePriorityUsage(db, access.key, surface, method, path, tool)
	if err != nil {
		log.Printf("priority usage reserve %s %s: %v", surface, tool, err)
		return access, false
	}
	if !reserved {
		return access, false
	}
	access.reservationID = reservationID
	access.monthlyRemaining = remaining
	return access, true
}

func releasePriorityUnit(db *sql.DB, access requestRateAccess) requestRateAccess {
	if access.key == nil || access.reservationID < 1 {
		return access
	}
	if err := models.ReleasePriorityUsage(db, access.key, access.reservationID); err != nil {
		log.Printf("priority usage refund %d: %v", access.reservationID, err)
		return access
	}
	access.reservationID = 0
	used, err := models.CurrentMonthUsage(db, access.key, "")
	if err != nil {
		log.Printf("priority usage remaining after refund %d: %v", access.key.ID, err)
		if access.monthlyRemaining < access.monthlyLimit {
			access.monthlyRemaining++
		}
		return access
	}
	access.monthlyRemaining = access.monthlyLimit - used
	if access.monthlyRemaining < 0 {
		access.monthlyRemaining = 0
	}
	return access
}
