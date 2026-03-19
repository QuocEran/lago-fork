package invoices_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domain "github.com/getlago/lago/api-go/internal/domain/invoices"
)

func stateWithStatus(status domain.InvoiceStatus) *domain.InvoiceState {
	return &domain.InvoiceState{Status: status}
}

func TestApplyFinalize_DraftToFinalized(t *testing.T) {
	state := stateWithStatus(domain.InvoiceStatusDraft)

	err := domain.ApplyFinalize(state)

	require.NoError(t, err)
	assert.Equal(t, domain.InvoiceStatusFinalized, state.Status)
	assert.NotNil(t, state.FinalizedAt)
}

func TestApplyFinalize_GeneratingToFinalized(t *testing.T) {
	state := stateWithStatus(domain.InvoiceStatusGenerating)

	err := domain.ApplyFinalize(state)

	require.NoError(t, err)
	assert.Equal(t, domain.InvoiceStatusFinalized, state.Status)
	assert.NotNil(t, state.FinalizedAt)
}

func TestApplyFinalize_AlreadyFinalized(t *testing.T) {
	state := stateWithStatus(domain.InvoiceStatusFinalized)

	actualErr := domain.ApplyFinalize(state)

	assert.ErrorIs(t, actualErr, domain.ErrAlreadyFinalized)
}

func TestApplyFinalize_AlreadyVoided(t *testing.T) {
	state := stateWithStatus(domain.InvoiceStatusVoided)

	actualErr := domain.ApplyFinalize(state)

	assert.ErrorIs(t, actualErr, domain.ErrAlreadyVoided)
}

func TestApplyVoid_FinalizedToVoided(t *testing.T) {
	state := stateWithStatus(domain.InvoiceStatusFinalized)

	err := domain.ApplyVoid(state)

	require.NoError(t, err)
	assert.Equal(t, domain.InvoiceStatusVoided, state.Status)
	assert.NotNil(t, state.VoidedAt)
}

func TestApplyVoid_DraftRejected(t *testing.T) {
	state := stateWithStatus(domain.InvoiceStatusDraft)

	actualErr := domain.ApplyVoid(state)

	assert.ErrorIs(t, actualErr, domain.ErrCannotVoidDraft)
}

func TestApplyVoid_AlreadyVoided(t *testing.T) {
	state := stateWithStatus(domain.InvoiceStatusVoided)

	actualErr := domain.ApplyVoid(state)

	assert.ErrorIs(t, actualErr, domain.ErrAlreadyVoided)
}
