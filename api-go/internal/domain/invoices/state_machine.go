package invoices

import (
	"errors"
	"time"

	"github.com/getlago/lago/api-go/internal/models"
)

var (
	ErrInvalidTransition = errors.New("invalid_state_transition")
	ErrAlreadyFinalized  = errors.New("invoice_already_finalized")
	ErrAlreadyVoided     = errors.New("invoice_already_voided")
	ErrCannotVoidDraft   = errors.New("cannot_void_draft_invoice")
)

// CanFinalize reports whether the invoice may transition to Finalized.
// Allowed source states: Draft, Generating.
func CanFinalize(invoice *models.Invoice) bool {
	return invoice.Status == models.InvoiceStatusDraft ||
		invoice.Status == models.InvoiceStatusGenerating
}

// CanVoid reports whether the invoice may transition to Voided.
// Only Finalized invoices can be voided.
func CanVoid(invoice *models.Invoice) bool {
	return invoice.Status == models.InvoiceStatusFinalized
}

// ApplyFinalize transitions the invoice to Finalized and stamps FinalizedAt.
// Returns a domain error if the transition is not allowed.
func ApplyFinalize(invoice *models.Invoice) error {
	switch invoice.Status {
	case models.InvoiceStatusFinalized:
		return ErrAlreadyFinalized
	case models.InvoiceStatusVoided:
		return ErrAlreadyVoided
	}

	if !CanFinalize(invoice) {
		return ErrInvalidTransition
	}

	now := time.Now()
	invoice.Status = models.InvoiceStatusFinalized
	invoice.FinalizedAt = &now
	return nil
}

// ApplyVoid transitions the invoice to Voided and stamps VoidedAt.
// Returns a domain error if the transition is not allowed.
func ApplyVoid(invoice *models.Invoice) error {
	switch invoice.Status {
	case models.InvoiceStatusVoided:
		return ErrAlreadyVoided
	case models.InvoiceStatusDraft:
		return ErrCannotVoidDraft
	}

	if !CanVoid(invoice) {
		return ErrInvalidTransition
	}

	now := time.Now()
	invoice.Status = models.InvoiceStatusVoided
	invoice.VoidedAt = &now
	return nil
}
