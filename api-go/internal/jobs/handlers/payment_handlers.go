package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"
)

const TaskTypeCreatePayment = "payments:create"

// CreatePaymentPayload carries the identifiers needed to create a payment for an invoice.
type CreatePaymentPayload struct {
	OrganizationID string `json:"organization_id"`
	InvoiceID      string `json:"invoice_id"`
}

// NewCreatePaymentTask creates an Asynq task to initiate payment for an invoice.
func NewCreatePaymentTask(organizationID, invoiceID string) (*asynq.Task, error) {
	b, err := json.Marshal(CreatePaymentPayload{
		OrganizationID: organizationID,
		InvoiceID:      invoiceID,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal create payment payload: %w", err)
	}
	return asynq.NewTask(TaskTypeCreatePayment, b), nil
}

// HandleCreatePayment is a stub for payment creation job.
// Full implementation depends on payment provider integration (lago-fork-4ig / lago-fork-kwx).
//
// Dead-letter: malformed payload skips retries immediately.
func HandleCreatePayment() asynq.HandlerFunc {
	return func(_ context.Context, task *asynq.Task) error {
		var payload CreatePaymentPayload
		if err := json.Unmarshal(task.Payload(), &payload); err != nil {
			slog.Error("create_payment: unmarshal failed", slog.String("error", err.Error()))
			return fmt.Errorf("unmarshal: %w: %w", err, asynq.SkipRetry)
		}

		// TODO(lago-fork-4ig): integrate with payment provider (Stripe, GoCardless, etc.)
		slog.Info("create_payment: stub executed (payment provider integration pending)",
			slog.String("invoice_id", payload.InvoiceID),
			slog.String("organization_id", payload.OrganizationID),
		)
		return nil
	}
}
