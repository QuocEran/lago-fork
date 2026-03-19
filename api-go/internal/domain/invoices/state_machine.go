package invoices

import (
	"errors"
	"time"
)

var (
	ErrInvalidTransition = errors.New("invalid_state_transition")
	ErrAlreadyFinalized  = errors.New("invoice_already_finalized")
	ErrAlreadyVoided     = errors.New("invoice_already_voided")
	ErrCannotVoidDraft   = errors.New("cannot_void_draft_invoice")
)

// CanFinalize reports whether the invoice may transition to Finalized.
// Allowed source states: Draft, Generating.
func CanFinalize(state *InvoiceState) bool {
	return state.Status == InvoiceStatusDraft ||
		state.Status == InvoiceStatusGenerating
}

// CanVoid reports whether the invoice may transition to Voided.
// Only Finalized invoices can be voided.
func CanVoid(state *InvoiceState) bool {
	return state.Status == InvoiceStatusFinalized
}

// ApplyFinalize transitions the state to Finalized and stamps FinalizedAt.
// Returns a domain error if the transition is not allowed.
func ApplyFinalize(state *InvoiceState) error {
	switch state.Status {
	case InvoiceStatusFinalized:
		return ErrAlreadyFinalized
	case InvoiceStatusVoided:
		return ErrAlreadyVoided
	}

	if !CanFinalize(state) {
		return ErrInvalidTransition
	}

	now := time.Now()
	state.Status = InvoiceStatusFinalized
	state.FinalizedAt = &now
	return nil
}

// ApplyVoid transitions the state to Voided and stamps VoidedAt.
// Returns a domain error if the transition is not allowed.
func ApplyVoid(state *InvoiceState) error {
	switch state.Status {
	case InvoiceStatusVoided:
		return ErrAlreadyVoided
	case InvoiceStatusDraft:
		return ErrCannotVoidDraft
	}

	if !CanVoid(state) {
		return ErrInvalidTransition
	}

	now := time.Now()
	state.Status = InvoiceStatusVoided
	state.VoidedAt = &now
	return nil
}
