package handlers

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/hibiken/asynq"
	"gorm.io/gorm"

	"github.com/getlago/lago/api-go/internal/models"
)
const (
	TaskTypeSendHTTPWebhook = "webhooks:send_http"

	webhookHTTPTimeout    = 30 * time.Second
	webhookSignatureHeader = "X-Lago-Signature"
)

// SendHTTPWebhookPayload is the JSON payload for the send-http-webhook task.
type SendHTTPWebhookPayload struct {
	WebhookID string `json:"webhook_id"`
}

// NewSendHTTPWebhookTask creates an Asynq task to deliver one outbound webhook by its DB ID.
func NewSendHTTPWebhookTask(webhookID string) (*asynq.Task, error) {
	b, err := json.Marshal(SendHTTPWebhookPayload{WebhookID: webhookID})
	if err != nil {
		return nil, fmt.Errorf("marshal send_http_webhook payload: %w", err)
	}
	return asynq.NewTask(TaskTypeSendHTTPWebhook, b), nil
}

// HTTPPoster abstracts the HTTP transport so tests can inject a fake.
type HTTPPoster interface {
	Do(req *http.Request) (*http.Response, error)
}

// defaultHTTPPoster wraps a real http.Client.
type defaultHTTPPoster struct{ client *http.Client }

func (p *defaultHTTPPoster) Do(req *http.Request) (*http.Response, error) {
	return p.client.Do(req)
}

// HandleSendHTTPWebhook processes one outbound webhook delivery:
//  1. Load webhook + endpoint from DB
//  2. Serialize payload and sign with HMAC-SHA256
//  3. POST to endpoint URL
//  4. Record http_status + response; update status (succeeded/failed)
//
// Idempotent: if the webhook is already succeeded, this is a no-op.
// Dead-letter: malformed payload or missing endpoint returns SkipRetry.
func HandleSendHTTPWebhook(db *gorm.DB, signingSecret string) asynq.HandlerFunc {
	poster := &defaultHTTPPoster{
		client: &http.Client{Timeout: webhookHTTPTimeout},
	}
	return buildSendHTTPWebhookHandler(db, signingSecret, poster)
}

// buildSendHTTPWebhookHandler is the injectable version used by both the real handler and tests.
func buildSendHTTPWebhookHandler(db *gorm.DB, signingSecret string, poster HTTPPoster) asynq.HandlerFunc {
	return func(ctx context.Context, task *asynq.Task) error {
		var p SendHTTPWebhookPayload
		if err := json.Unmarshal(task.Payload(), &p); err != nil {
			slog.Error("send_http_webhook: unmarshal failed", slog.String("error", err.Error()))
			return fmt.Errorf("unmarshal: %w: %w", err, asynq.SkipRetry)
		}

		var wh models.Webhook
		if err := db.WithContext(ctx).First(&wh, "id = ?", p.WebhookID).Error; err != nil {
			slog.Error("send_http_webhook: webhook not found",
				slog.String("webhook_id", p.WebhookID),
				slog.String("error", err.Error()),
			)
			return fmt.Errorf("webhook not found %s: %w", p.WebhookID, asynq.SkipRetry)
		}

		// Idempotency guard — already delivered.
		if wh.Status == models.WebhookStatusSucceeded {
			slog.Info("send_http_webhook: already succeeded, skipping", slog.String("webhook_id", p.WebhookID))
			return nil
		}

		if wh.Endpoint == nil || *wh.Endpoint == "" {
			slog.Error("send_http_webhook: missing endpoint", slog.String("webhook_id", p.WebhookID))
			return fmt.Errorf("missing endpoint on webhook %s: %w", p.WebhookID, asynq.SkipRetry)
		}

		payloadBytes, err := json.Marshal(wh.Payload)
		if err != nil {
			slog.Error("send_http_webhook: marshal payload failed", slog.String("error", err.Error()))
			return fmt.Errorf("marshal payload: %w: %w", err, asynq.SkipRetry)
		}

		signature := signPayload(signingSecret, payloadBytes)

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, *wh.Endpoint, bytes.NewReader(payloadBytes))
		if err != nil {
			slog.Error("send_http_webhook: build request failed", slog.String("error", err.Error()))
			return fmt.Errorf("build request: %w: %w", err, asynq.SkipRetry)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(webhookSignatureHeader, signature)

		resp, postErr := poster.Do(req)
		now := time.Now().UTC()

		update := map[string]any{"updated_at": now, "retries": wh.Retries + 1, "last_retried_at": now}

		if postErr != nil {
			update["status"] = models.WebhookStatusFailed
			db.WithContext(ctx).Model(&wh).Updates(update)
			slog.Error("send_http_webhook: http post failed",
				slog.String("webhook_id", p.WebhookID),
				slog.String("endpoint", *wh.Endpoint),
				slog.String("error", postErr.Error()),
			)
			return fmt.Errorf("http post to %s: %w", *wh.Endpoint, postErr)
		}
		defer resp.Body.Close()

		httpStatus := resp.StatusCode
		update["http_status"] = httpStatus

		if httpStatus >= 200 && httpStatus < 300 {
			update["status"] = models.WebhookStatusSucceeded
			slog.Info("send_http_webhook: delivered",
				slog.String("webhook_id", p.WebhookID),
				slog.Int("http_status", httpStatus),
			)
		} else {
			update["status"] = models.WebhookStatusFailed
			slog.Warn("send_http_webhook: non-2xx response",
				slog.String("webhook_id", p.WebhookID),
				slog.Int("http_status", httpStatus),
			)
		}

		db.WithContext(ctx).Model(&wh).Updates(update)

		if httpStatus < 200 || httpStatus >= 300 {
			return fmt.Errorf("endpoint %s returned HTTP %d", *wh.Endpoint, httpStatus)
		}
		return nil
	}
}

// signPayload returns the hex-encoded HMAC-SHA256 signature of the payload.
func signPayload(secret string, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

// SignPayloadForTest is exported for test use only.
func SignPayloadForTest(secret string, payload []byte) string {
	return signPayload(secret, payload)
}

// VerifySignature checks that the given signature matches the payload using the shared secret.
// It reads from the reader and does NOT seek back.
func VerifySignature(secret string, body io.Reader, sig string) bool {
	payload, err := io.ReadAll(body)
	if err != nil {
		return false
	}
	expected := signPayload(secret, payload)
	return hmac.Equal([]byte(expected), []byte(sig))
}

// BuildSendHTTPWebhookHandlerForTest constructs the handler with a custom HTTPPoster, for testing.
func BuildSendHTTPWebhookHandlerForTest(db *gorm.DB, signingSecret string, poster HTTPPoster) asynq.HandlerFunc {
	return buildSendHTTPWebhookHandler(db, signingSecret, poster)
}
