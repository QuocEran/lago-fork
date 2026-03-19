package organizations_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/getlago/lago/api-go/internal/services/organizations"
)

func TestValidationError_Error(t *testing.T) {
	err := &organizations.ValidationError{Message: "invalid country"}
	assert.Equal(t, "invalid country", err.Error())
	assert.True(t, organizations.IsValidationError(err))
}

func TestIsValidationError_ReturnsFalseForOtherErrors(t *testing.T) {
	assert.False(t, organizations.IsValidationError(organizations.ErrOrganizationNotFound))
}

func TestUpdateOrganizationInput_ZeroValue(t *testing.T) {
	var input organizations.UpdateOrganizationInput
	assert.Nil(t, input.Country)
	assert.Nil(t, input.DefaultCurrency)
}

func TestBillingConfigurationInput_ZeroValue(t *testing.T) {
	var input organizations.BillingConfigurationInput
	assert.Nil(t, input.InvoiceFooter)
	assert.Nil(t, input.InvoiceGracePeriod)
}
