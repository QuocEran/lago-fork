package handlers_test

import (
	"context"
	"errors"
	"testing"

	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	invdomain "github.com/getlago/lago/api-go/internal/domain/invoices"
	"github.com/getlago/lago/api-go/internal/jobs/handlers"
	"github.com/getlago/lago/api-go/internal/models"
	invsvc "github.com/getlago/lago/api-go/internal/services/invoices"
)

// ── mock invoice service ──────────────────────────────────────────────────────

type mockInvoiceService struct {
	finalizeFn func(ctx context.Context, orgID, id string) (*models.Invoice, error)
}

func (m *mockInvoiceService) Create(_ context.Context, _ string, _ invsvc.CreateInvoiceInput) (*models.Invoice, error) {
	return nil, nil
}
func (m *mockInvoiceService) List(_ context.Context, _ string, _ invsvc.ListInvoicesFilter) ([]models.Invoice, *invsvc.Pagination, error) {
	return nil, nil, nil
}
func (m *mockInvoiceService) GetByID(_ context.Context, _, _ string) (*models.Invoice, error) {
	return nil, nil
}
func (m *mockInvoiceService) Finalize(ctx context.Context, orgID, id string) (*models.Invoice, error) {
	return m.finalizeFn(ctx, orgID, id)
}
func (m *mockInvoiceService) Void(_ context.Context, _, _ string) (*models.Invoice, error) {
	return nil, nil
}

// ── helpers ──────────────────────────────────────────────────────────────────

func makeTask(t *testing.T, orgID, invoiceID string) *asynq.Task {
	t.Helper()
	task, err := handlers.NewFinalizeInvoiceTask(orgID, invoiceID)
	require.NoError(t, err)
	return task
}

// ── TestHandleFinalizeInvoice ─────────────────────────────────────────────────

func TestHandleFinalizeInvoice_Success(t *testing.T) {
	svc := &mockInvoiceService{
		finalizeFn: func(_ context.Context, _, _ string) (*models.Invoice, error) {
			return &models.Invoice{Status: models.InvoiceStatusFinalized}, nil
		},
	}
	handler := handlers.HandleFinalizeInvoice(svc)
	task := makeTask(t, "org-1", "inv-1")
	err := handler(context.Background(), task)
	assert.NoError(t, err)
}

func TestHandleFinalizeInvoice_AlreadyFinalized_IsIdempotent(t *testing.T) {
	// An already-finalized invoice returns ErrAlreadyFinalized — the handler treats it as success.
	svc := &mockInvoiceService{
		finalizeFn: func(_ context.Context, _, _ string) (*models.Invoice, error) {
			return nil, invdomain.ErrAlreadyFinalized
		},
	}
	handler := handlers.HandleFinalizeInvoice(svc)
	task := makeTask(t, "org-1", "inv-2")
	err := handler(context.Background(), task)
	assert.NoError(t, err, "already-finalized invoice should be treated as idempotent success")
}

func TestHandleFinalizeInvoice_NotFound_SentToDeadLetter(t *testing.T) {
	svc := &mockInvoiceService{
		finalizeFn: func(_ context.Context, _, _ string) (*models.Invoice, error) {
			return nil, invsvc.ErrInvoiceNotFound
		},
	}
	handler := handlers.HandleFinalizeInvoice(svc)
	task := makeTask(t, "org-1", "missing-id")
	err := handler(context.Background(), task)
	require.Error(t, err)
	assert.ErrorIs(t, err, asynq.SkipRetry, "not-found must be sent to dead-letter (SkipRetry)")
}

func TestHandleFinalizeInvoice_TransientError_Retryable(t *testing.T) {
	transient := errors.New("connection reset by peer")
	svc := &mockInvoiceService{
		finalizeFn: func(_ context.Context, _, _ string) (*models.Invoice, error) {
			return nil, transient
		},
	}
	handler := handlers.HandleFinalizeInvoice(svc)
	task := makeTask(t, "org-1", "inv-3")
	err := handler(context.Background(), task)
	require.Error(t, err)
	assert.NotErrorIs(t, err, asynq.SkipRetry, "transient error must NOT set SkipRetry")
}

func TestHandleFinalizeInvoice_MalformedPayload_SentToDeadLetter(t *testing.T) {
	svc := &mockInvoiceService{}
	handler := handlers.HandleFinalizeInvoice(svc)
	badTask := asynq.NewTask(handlers.TaskTypeFinalizeInvoice, []byte("not-json"))
	err := handler(context.Background(), badTask)
	require.Error(t, err)
	assert.ErrorIs(t, err, asynq.SkipRetry, "malformed payload must be sent to dead-letter")
}
