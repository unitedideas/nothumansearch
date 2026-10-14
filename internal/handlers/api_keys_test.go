package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/unitedideas/nothumansearch/internal/models"
)

func TestAPIKeySubscribeGetDocumentsPlans(t *testing.T) {
	h := NewAPIKeyHandler(nil, "https://nothumansearch.ai")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/api-keys/subscribe", nil)
	rr := httptest.NewRecorder()

	h.Subscribe(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("GET subscribe status = %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	var payload struct {
		Plans []struct {
			ID           string `json:"id"`
			Plan         string `json:"plan"`
			MonthlyLimit int    `json:"monthly_limit"`
		} `json:"plans"`
		Subscribe struct {
			Method         string   `json:"method"`
			Endpoint       string   `json:"endpoint"`
			RequiredFields []string `json:"required_fields"`
			AllowedPlans   []string `json:"allowed_plans"`
			ExampleBody    struct {
				Plan string `json:"plan"`
			} `json:"example_body"`
		} `json:"subscribe"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode subscribe document: %v", err)
	}
	if len(payload.Plans) != 1 {
		t.Fatalf("plans count = %d, want 1", len(payload.Plans))
	}
	want := map[string]int{"unlimited": 50000}
	for _, plan := range payload.Plans {
		if got := want[plan.Plan]; got != plan.MonthlyLimit {
			t.Fatalf("plan %q monthly_limit = %d, want %d", plan.Plan, plan.MonthlyLimit, got)
		}
		if plan.ID != "nhs_api_"+plan.Plan {
			t.Fatalf("plan %q id = %q", plan.Plan, plan.ID)
		}
	}
	if payload.Subscribe.Method != http.MethodPost {
		t.Fatalf("subscribe method = %q, want POST", payload.Subscribe.Method)
	}
	if payload.Subscribe.Endpoint != "https://nothumansearch.ai/api/v1/api-keys/subscribe" {
		t.Fatalf("subscribe endpoint = %q", payload.Subscribe.Endpoint)
	}
	if len(payload.Subscribe.RequiredFields) != 2 {
		t.Fatalf("required fields = %v, want email and plan", payload.Subscribe.RequiredFields)
	}
	if len(payload.Subscribe.AllowedPlans) != 1 || payload.Subscribe.AllowedPlans[0] != "unlimited" || payload.Subscribe.ExampleBody.Plan != "unlimited" {
		t.Fatalf("subscribe plan contract = allowed %v example %q, want unlimited", payload.Subscribe.AllowedPlans, payload.Subscribe.ExampleBody.Plan)
	}
}

func TestAPIKeySubscribeMetadataIncludesAttribution(t *testing.T) {
	metadata := apiSubscribeMetadata("buyer@example.com", models.APIPlanFor("unlimited"), map[string]string{
		"qc":           "campaign-123",
		"utm_source":   "linkedin",
		"utm_medium":   "qlimit",
		"utm_campaign": "campaign-123",
	})

	for key, want := range map[string]string{
		"tenant":        "nothumansearch",
		"product":       "nhs_api_subscription",
		"product_id":    "nhs_api_unlimited",
		"source":        "api_key_subscribe",
		"plan":          "unlimited",
		"monthly_limit": "50000",
		"email":         "buyer@example.com",
		"qc":            "campaign-123",
		"utm_source":    "linkedin",
		"utm_medium":    "qlimit",
		"utm_campaign":  "campaign-123",
	} {
		if metadata[key] != want {
			t.Fatalf("metadata[%s] = %q, want %q; metadata=%#v", key, metadata[key], want, metadata)
		}
	}
}

func TestAPIKeySubscribeCancelURLPreservesAttribution(t *testing.T) {
	h := NewAPIKeyHandler(nil, "https://nothumansearch.ai")
	got := h.subscribeCancelURL("buyer@example.com", map[string]string{
		"qc":           "campaign-123",
		"utm_source":   "linkedin",
		"utm_medium":   "qlimit",
		"utm_campaign": "campaign-123",
	})
	for _, want := range []string{
		"https://nothumansearch.ai/subscribe?",
		"email=buyer%40example.com",
		"qc=campaign-123",
		"utm_source=linkedin",
		"utm_medium=qlimit",
		"utm_campaign=campaign-123",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("cancel URL %q missing %q", got, want)
		}
	}
}
