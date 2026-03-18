package invoices_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domain "github.com/getlago/lago/api-go/internal/domain/invoices"
	"github.com/getlago/lago/api-go/internal/models"
)

func invoiceWithStatus(status models.InvoiceStatus) *models.Invoice {
	return &models.Invoice{Status: status}
}

func TestApplyFinalize_DraftToFinalized(t *testing.T) {
	inputInvoice := invoiceWithStatus(models.InvoiceStatusDraft)

	err := domain.ApplyFinalize(inputInvoice)

	require.NoError(t, err)
	assert.Equal(t, models.InvoiceStatusFinalized, inputInvoice.Status)
	assert.NotNil(t, inputInvoice.FinalizedAt)
}

func TestApplyFinalize_GeneratingToFinalized(t *testing.T) {
	inputInvoice := invoiceWithStatus(models.InvoiceStatusGenerating)

	err := domain.ApplyFinalize(inputInvoice)

	require.NoError(t, err)
	assert.Equal(t, models.InvoiceStatusFinalized, inputInvoice.Status)
	assert.NotNil(t, inputInvoice.FinalizedAt)
}

func TestApplyFinalize_AlreadyFinalized(t *testing.T) {
	inputInvoice := invoiceWithStatus(models.InvoiceStatusFinalized)

	actualErr := domain.ApplyFinalize(inputInvoice)

	assert.ErrorIs(t, actualErr, domain.ErrAlreadyFinalized)
}

func TestApplyFinalize_AlreadyVoided(t *testing.T) {
	inputInvoice := invoiceWithStatus(models.InvoiceStatusVoided)

	actualErr := domain.ApplyFinalize(inputInvoice)

	assert.ErrorIs(t, actualErr, domain.ErrAlreadyVoided)
}

func TestApplyVoid_FinalizedToVoided(t *testing.T) {
	inputInvoice := invoiceWithStatus(models.InvoiceStatusFinalized)

	err := domain.ApplyVoid(inputInvoice)

	require.NoError(t, err)
	assert.Equal(t, models.InvoiceStatusVoided, inputInvoice.Status)
	assert.NotNil(t, inputInvoice.VoidedAt)
}

func TestApplyVoid_DraftRejected(t *testing.T) {
	inputInvoice := invoiceWithStatus(models.InvoiceStatusDraft)

	actualErr := domain.ApplyVoid(inputInvoice)

	assert.ErrorIs(t, actualErr, domain.ErrCannotVoidDraft)
}

func TestApplyVoid_AlreadyVoided(t *testing.T) {
	inputInvoice := invoiceWithStatus(models.InvoiceStatusVoided)

	actualErr := domain.ApplyVoid(inputInvoice)

	assert.ErrorIs(t, actualErr, domain.ErrAlreadyVoided)
}
