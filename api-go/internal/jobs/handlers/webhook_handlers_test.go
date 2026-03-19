package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/getlago/lago/api-go/internal/jobs/handlers"
	"github.com/getlago/lago/api-go/internal/models"
)

// ── mock HTTP poster ──────────────────────────────────────────────────────────

type mockHTTPPoster struct {
	response *http.Response
	err      error
	called   bool
	lastURL  string
}

func (m *mockHTTPPoster) Do(req *http.Request) (*http.Response, error) {
	m.called = true
	m.lastURL = req.URL.String()
	return m.response, m.err
}

func mockResponse(code int) *http.Response {
	return &http.Response{
		StatusCode: code,
		Body:       io.NopCloser(strings.NewReader("")),
	}
}

// ── db setup ──────────────────────────────────────────────────────────────────

func newWebhookMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { sqlDB.Close() })

	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{})
	require.NoError(t, err)
	return db, mock
}

func makeWebhookTask(t *testing.T, webhookID string) *asynq.Task {
	t.Helper()
	task, err := handlers.NewSendHTTPWebhookTask(webhookID)
	require.NoError(t, err)
	return task
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestHandleSendHTTPWebhook_Success(t *testing.T) {
	db, mock := newWebhookMockDB(t)
	endpoint := "https://example.com/webhook"
	webhookType := "invoice.created"
	orgID := "org-1"
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "organization_id", "webhook_endpoint_id", "object_id", "object_type",
		"webhook_type", "status", "retries", "http_status", "endpoint",
		"payload", "response", "last_retried_at", "created_at", "updated_at",
	}).AddRow(
		"wh-1", orgID, nil, nil, nil,
		webhookType, int(models.WebhookStatusPending), 0, nil, endpoint,
		`{"event":"invoice.created"}`, nil, nil, now, now,
	)
	mock.ExpectQuery(`SELECT \* FROM "webhooks"`).WillReturnRows(rows)
	// Accept any UPDATE (status record after delivery)
	mock.ExpectExec(`UPDATE`).WillReturnResult(sqlmock.NewResult(1, 1))

	poster := &mockHTTPPoster{response: mockResponse(200)}
	handler := handlers.BuildSendHTTPWebhookHandlerForTest(db, "secret", poster)

	task := makeWebhookTask(t, "wh-1")
	err := handler(context.Background(), task)
	require.NoError(t, err)
	assert.True(t, poster.called, "HTTP poster should be called")
	assert.Equal(t, endpoint, poster.lastURL)
}

func TestHandleSendHTTPWebhook_AlreadySucceeded_IsIdempotent(t *testing.T) {
	db, mock := newWebhookMockDB(t)
	endpoint := "https://example.com/webhook"
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "organization_id", "webhook_endpoint_id", "object_id", "object_type",
		"webhook_type", "status", "retries", "http_status", "endpoint",
		"payload", "response", "last_retried_at", "created_at", "updated_at",
	}).AddRow(
		"wh-2", "org-1", nil, nil, nil,
		"invoice.created", int(models.WebhookStatusSucceeded), 1, 200, endpoint,
		`{}`, nil, nil, now, now,
	)
	mock.ExpectQuery(`SELECT \* FROM "webhooks"`).WillReturnRows(rows)
	// No UPDATE expected — idempotent skip

	poster := &mockHTTPPoster{}
	handler := handlers.BuildSendHTTPWebhookHandlerForTest(db, "secret", poster)

	task := makeWebhookTask(t, "wh-2")
	err := handler(context.Background(), task)
	require.NoError(t, err)
	assert.False(t, poster.called, "should not POST if already succeeded")
}

func TestHandleSendHTTPWebhook_NotFound_SentToDeadLetter(t *testing.T) {
	db, mock := newWebhookMockDB(t)
	mock.ExpectQuery(`SELECT \* FROM "webhooks"`).WillReturnRows(sqlmock.NewRows(nil))

	poster := &mockHTTPPoster{}
	handler := handlers.BuildSendHTTPWebhookHandlerForTest(db, "secret", poster)

	task := makeWebhookTask(t, "missing-wh")
	err := handler(context.Background(), task)
	require.Error(t, err)
	assert.ErrorIs(t, err, asynq.SkipRetry, "not-found webhook must be dead-lettered")
}

func TestHandleSendHTTPWebhook_Non2xx_Retryable(t *testing.T) {
	db, mock := newWebhookMockDB(t)
	endpoint := "https://example.com/webhook"
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "organization_id", "webhook_endpoint_id", "object_id", "object_type",
		"webhook_type", "status", "retries", "http_status", "endpoint",
		"payload", "response", "last_retried_at", "created_at", "updated_at",
	}).AddRow(
		"wh-3", "org-1", nil, nil, nil,
		"invoice.created", int(models.WebhookStatusPending), 0, nil, endpoint,
		`{}`, nil, nil, now, now,
	)
	mock.ExpectQuery(`SELECT \* FROM "webhooks"`).WillReturnRows(rows)
	mock.ExpectExec(`UPDATE`).WillReturnResult(sqlmock.NewResult(1, 1))

	poster := &mockHTTPPoster{response: mockResponse(500)}
	handler := handlers.BuildSendHTTPWebhookHandlerForTest(db, "secret", poster)

	task := makeWebhookTask(t, "wh-3")
	err := handler(context.Background(), task)
	require.Error(t, err)
	assert.NotErrorIs(t, err, asynq.SkipRetry, "HTTP 500 must be retryable")
}

func TestHandleSendHTTPWebhook_MalformedPayload_SentToDeadLetter(t *testing.T) {
	db, _ := newWebhookMockDB(t)
	poster := &mockHTTPPoster{}
	handler := handlers.BuildSendHTTPWebhookHandlerForTest(db, "secret", poster)
	badTask := asynq.NewTask(handlers.TaskTypeSendHTTPWebhook, []byte("bad-json"))
	err := handler(context.Background(), badTask)
	require.Error(t, err)
	assert.ErrorIs(t, err, asynq.SkipRetry)
}

// TestVerifySignature ensures the HMAC round-trip is consistent.
func TestVerifySignature_RoundTrip(t *testing.T) {
	secret := "my-webhook-secret"
	payload, _ := json.Marshal(map[string]string{"event": "test"})
	sig := handlers.SignPayloadForTest(secret, payload)
	assert.True(t, handlers.VerifySignature(secret, bytes.NewReader(payload), sig), "signature must verify correctly")
}
