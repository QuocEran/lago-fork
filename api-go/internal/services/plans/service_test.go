package plans_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/getlago/lago/api-go/internal/services/plans"
)

func TestValidationError_Error(t *testing.T) {
	err := &plans.ValidationError{Message: "code is required"}
	assert.Equal(t, "code is required", err.Error())
	assert.True(t, plans.IsValidationError(err))
}

func TestIsValidationError_ReturnsFalseForOtherErrors(t *testing.T) {
	assert.False(t, plans.IsValidationError(plans.ErrPlanNotFound))
	assert.False(t, plans.IsValidationError(plans.ErrPlanCodeConflict))
}

func TestCreateInput_RequiredFields(t *testing.T) {
	input := plans.CreateInput{
		Name:           "Pro",
		Code:           "pro",
		Interval:       "monthly",
		AmountCents:    1000,
		AmountCurrency: "USD",
	}
	assert.Equal(t, "Pro", input.Name)
	assert.Equal(t, "pro", input.Code)
	assert.Equal(t, "monthly", input.Interval)
	assert.Equal(t, int64(1000), input.AmountCents)
	assert.Equal(t, "USD", input.AmountCurrency)
}

func TestListFilter_ZeroValue(t *testing.T) {
	var f plans.ListFilter
	assert.Equal(t, 0, f.Page)
	assert.Equal(t, 0, f.PerPage)
	assert.False(t, f.WithDeleted)
}
