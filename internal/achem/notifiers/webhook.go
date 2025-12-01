package notifiers

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/daniacca/achemdb/internal/achem"
)

// WebhookNotifier sends notifications via HTTP POST to a webhook URL
type WebhookNotifier struct {
	id      string
	url     string
	client  *http.Client
	headers map[string]string
}

// NewWebhookNotifier creates a new webhook notifier
func NewWebhookNotifier(id, url string) *WebhookNotifier {
	return &WebhookNotifier{
		id:      id,
		url:     url,
		client:  &http.Client{Timeout: 5 * time.Second},
		headers: make(map[string]string),
	}
}

// SetHeader sets a custom header to include in webhook requests
func (wn *WebhookNotifier) SetHeader(key, value string) {
	if wn.headers == nil {
		wn.headers = make(map[string]string)
	}
	wn.headers[key] = value
}

// ID returns the notifier ID
func (wn *WebhookNotifier) ID() string {
	return wn.id
}

// Type returns the notifier type
func (wn *WebhookNotifier) Type() string {
	return "webhook"
}

// Notify sends the notification event to the webhook URL
func (wn *WebhookNotifier) Notify(ctx context.Context, event achem.NotificationEvent) error {
	jsonData, err := event.JSON()
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, wn.url, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for key, value := range wn.headers {
		req.Header.Set(key, value)
	}

	resp, err := wn.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// Close closes the notifier (no-op for webhook)
func (wn *WebhookNotifier) Close() error {
	return nil
}

