package handlers

import (
	"net/http"
	"time"

	"github.com/unitedideas/nothumansearch/internal/models"
)

type providerPageData struct {
	SignedIn                     bool
	Email                        string
	BaseURL                      string
	DNSFailureLimit              int
	DNSRecheckHours              int
	DNSVerificationFreshnessDays int
}

func newProviderPageData(baseURL string) providerPageData {
	return providerPageData{
		BaseURL:                      baseURL,
		DNSFailureLimit:              models.ProviderClaimDNSFailureLimit,
		DNSRecheckHours:              int(models.ProviderClaimDNSRecheckInterval / time.Hour),
		DNSVerificationFreshnessDays: int(models.ProviderClaimVerificationFreshness / (24 * time.Hour)),
	}
}

func (h *ProviderExchangeHandler) ProvidersPage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/providers" || r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	data := newProviderPageData(h.BaseURL)
	if h.Auth != nil {
		if account := h.Auth.CurrentAccount(r); account != nil {
			data.SignedIn = true
			data.Email = account.Email
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "private, no-store")
	if h.PageTemplate == nil {
		http.Error(w, "provider page unavailable", http.StatusServiceUnavailable)
		return
	}
	if err := h.PageTemplate.ExecuteTemplate(w, "providers.html", data); err != nil {
		http.Error(w, "provider page unavailable", http.StatusInternalServerError)
	}
}

func (h *ProviderExchangeHandler) PrivacyPage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/privacy" || r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	if h.PageTemplate == nil {
		http.Error(w, "privacy page unavailable", http.StatusServiceUnavailable)
		return
	}
	if err := h.PageTemplate.ExecuteTemplate(w, "privacy.html", newProviderPageData(h.BaseURL)); err != nil {
		http.Error(w, "privacy page unavailable", http.StatusInternalServerError)
	}
}
