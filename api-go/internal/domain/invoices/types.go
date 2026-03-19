package invoices

import "time"

// InvoiceStatus represents the lifecycle status of an invoice in the domain.
// Values match models.InvoiceStatus for easy mapping.
type InvoiceStatus int

const (
	InvoiceStatusDraft      InvoiceStatus = 0
	InvoiceStatusFinalized  InvoiceStatus = 1
	InvoiceStatusVoided     InvoiceStatus = 2
	InvoiceStatusGenerating InvoiceStatus = 3
	InvoiceStatusFailed     InvoiceStatus = 4
)

// InvoiceState holds the fields required for invoice state machine logic.
type InvoiceState struct {
	Status      InvoiceStatus
	FinalizedAt *time.Time
	VoidedAt    *time.Time
}
