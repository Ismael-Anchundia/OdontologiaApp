package integrations

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Delivery struct {
	ID        int64     `json:"id"`
	Channel   string    `json:"channel"`
	Recipient string    `json:"recipient"`
	Template  string    `json:"template"`
	Message   string    `json:"message"`
	Scheduled time.Time `json:"scheduledAt"`
}

type NotificationSender interface {
	Send(context.Context, Delivery) (string, error)
}

// WebhookSender adapts the outbox to a clinic-selected email/SMS/WhatsApp gateway.
// The endpoint and secret are supplied by the operator and are never compiled into the app.
type WebhookSender struct {
	Endpoint string
	Secret   string
	Client   *http.Client
}

func NewWebhookSenderFromEnv() (NotificationSender, bool) {
	endpoint := strings.TrimSpace(os.Getenv("NOTIFICATIONS_WEBHOOK_URL"))
	secret := strings.TrimSpace(os.Getenv("NOTIFICATIONS_WEBHOOK_SECRET"))
	if endpoint == "" || secret == "" {
		return nil, false
	}
	parsed, err := url.ParseRequestURI(endpoint)
	if err != nil || parsed.Host == "" || (os.Getenv("APP_ENV") == "production" && parsed.Scheme != "https") {
		return nil, false
	}
	return &WebhookSender{Endpoint: endpoint, Secret: secret, Client: &http.Client{Timeout: 10 * time.Second}}, true
}

func (s *WebhookSender) Send(ctx context.Context, delivery Delivery) (string, error) {
	if s == nil || s.Endpoint == "" || s.Secret == "" {
		return "", errors.New("notification webhook is not configured")
	}
	payload, err := json.Marshal(delivery)
	if err != nil {
		return "", err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, s.Endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+s.Secret)
	request.Header.Set("Idempotency-Key", deliveryKey(delivery))
	client := s.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", errors.New("notification provider returned a non-success status")
	}
	return response.Header.Get("X-Delivery-ID"), nil
}

func deliveryKey(delivery Delivery) string {
	return "dental-outbox-" + strconv.FormatInt(delivery.ID, 10)
}
