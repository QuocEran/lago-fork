package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"
	"gorm.io/gorm"

	"github.com/getlago/lago/api-go/internal/models"
	invsvc "github.com/getlago/lago/api-go/internal/services/invoices"
)

const (
	TaskTypeFinalizeInvoice      = "invoice:finalize"
	TaskTypeMarkPaymentOverdue   = "invoice:mark_payment_overdue"
)

// FinalizeInvoicePayload is the JSON payload for the finalize-invoice task.
type FinalizeInvoicePayload struct {
	OrganizationID string `json:"organization_id"`
	InvoiceID      string `json:"invoice_id"`
}

// NewFinalizeInvoiceTask creates an Asynq task to finalize a single draft invoice.
func NewFinalizeInvoiceTask(organizationID, invoiceID string) (*asynq.Task, error) {
	b, err := json.Marshal(FinalizeInvoicePayload{
		OrganizationID: organizationID,
		InvoiceID:      invoiceID,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal finalize invoice payload: %w", err)
	}
	return asynq.NewTask(TaskTypeFinalizeInvoice, b), nil
}

// HandleFinalizeInvoice processes a single draft-to-finalized invoice transition.
// Idempotent: if the invoice is already finalized the service returns a transition error which is treated as success.
// Dead-letter: on unrecoverable errors (not-found, validation) returns asynq.SkipRetry.
func HandleFinalizeInvoice(svc invsvc.Service) asynq.HandlerFunc {
	return func(ctx context.Context, task *asynq.Task) error {
		var payload FinalizeInvoicePayload
		if err := json.Unmarshal(task.Payload(), &payload); err != nil {
			slog.Error("finalize_invoice: unmarshal failed",
				slog.String("error", err.Error()),
				slog.String("task_id", task.Type()),
			)
			// Malformed payload — no value in retrying, skip to dead-letter.
			return fmt.Errorf("unmarshal: %w: %w", err, asynq.SkipRetry)
		}

		_, err := svc.Finalize(ctx, payload.OrganizationID, payload.InvoiceID)
		if err == nil {
			return nil
		}

		// Idempotency: already-finalized / already-voided is not an error.
		if invsvc.IsTransitionError(err) {
			slog.Info("finalize_invoice: skipping already-transitioned invoice",
				slog.String("invoice_id", payload.InvoiceID),
				slog.String("reason", err.Error()),
			)
			return nil
		}

		// Not found — no value in retrying.
		if invsvc.IsNotFoundError(err) {
			slog.Warn("finalize_invoice: invoice not found, sending to dead-letter",
				slog.String("invoice_id", payload.InvoiceID),
			)
			return fmt.Errorf("invoice not found %s: %w", payload.InvoiceID, asynq.SkipRetry)
		}

		// Transient error — let Asynq retry.
		slog.Error("finalize_invoice: transient error, will retry",
			slog.String("invoice_id", payload.InvoiceID),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("finalize invoice %s: %w", payload.InvoiceID, err)
	}
}

// HandleMarkPaymentOverdue scans all finalized invoices whose payment_due_date has passed
// and marks them payment_overdue=true. This is a clock/batch job — idempotent by WHERE clause.
func HandleMarkPaymentOverdue(db *gorm.DB) asynq.HandlerFunc {
	return func(ctx context.Context, _ *asynq.Task) error {
		now := time.Now().UTC()

		result := db.WithContext(ctx).
			Model(&models.Invoice{}).
			Where("status = ? AND payment_overdue = false AND payment_dispute_lost_at IS NULL AND payment_due_date < ?",
				models.InvoiceStatusFinalized, now).
			Where("payment_status != ?", models.InvoicePaymentStatusSucceeded).
			Updates(map[string]any{
				"payment_overdue": true,
				"updated_at":      now,
			})

		if result.Error != nil {
			slog.Error("mark_payment_overdue: db update failed",
				slog.String("error", result.Error.Error()),
			)
			return fmt.Errorf("mark payment overdue: %w", result.Error)
		}

		slog.Info("mark_payment_overdue: completed",
			slog.Int64("rows_updated", result.RowsAffected),
		)
		return nil
	}
}
