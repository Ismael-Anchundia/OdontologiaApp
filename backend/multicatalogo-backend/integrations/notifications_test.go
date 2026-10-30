package integrations

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestWebhookSenderDeliversAuthenticatedIdempotentPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.Header.Get("Authorization") != "Bearer test-secret" {
			t.Errorf("unexpected request: %s, %s", r.Method, r.Header.Get("Authorization"))
		}
		if r.Header.Get("Idempotency-Key") != "dental-outbox-12" {
			t.Errorf("unexpected idempotency key: %s", r.Header.Get("Idempotency-Key"))
		}
		var payload Delivery
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode payload: %v", err)
		}
		if payload.Recipient != "+593999000000" || payload.Template != "appointment_reminder_24h" {
			t.Errorf("unexpected payload: %+v", payload)
		}
		w.Header().Set("X-Delivery-ID", "provider-1")
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()
	sender := &WebhookSender{Endpoint: server.URL, Secret: "test-secret", Client: server.Client()}
	ref, err := sender.Send(context.Background(), Delivery{ID: 12, Channel: "whatsapp", Recipient: "+593999000000", Template: "appointment_reminder_24h", Message: "Recordatorio", Scheduled: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	if ref != "provider-1" {
		t.Fatalf("provider reference = %q", ref)
	}
}

func TestWebhookSenderRejectsNonSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusBadGateway) }))
	defer server.Close()
	sender := &WebhookSender{Endpoint: server.URL, Secret: "test-secret", Client: server.Client()}
	if _, err := sender.Send(context.Background(), Delivery{ID: 1, Recipient: "test@example.test"}); err == nil {
		t.Fatal("expected provider error")
	}
}

func TestNewWebhookSenderRequiresEndpointAndSecret(t *testing.T) {
	t.Setenv("NOTIFICATIONS_WEBHOOK_URL", "")
	t.Setenv("NOTIFICATIONS_WEBHOOK_SECRET", "secret")
	if _, ok := NewWebhookSenderFromEnv(); ok {
		t.Fatal("sender must remain disabled without an endpoint")
	}
}
