package subscriptions_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/getlago/lago/api-go/internal/services/subscriptions"
)

func TestValidationError_Error(t *testing.T) {
	err := &subscriptions.ValidationError{Message: "customer_id is required"}
	assert.Equal(t, "customer_id is required", err.Error())
	assert.True(t, subscriptions.IsValidationError(err))
}

func TestIsValidationError_ReturnsFalseForOtherErrors(t *testing.T) {
	assert.False(t, subscriptions.IsValidationError(subscriptions.ErrSubscriptionNotFound))
	assert.False(t, subscriptions.IsValidationError(subscriptions.ErrExternalIDConflict))
}

func TestCreateInput_RequiredFields(t *testing.T) {
	input := subscriptions.CreateInput{
		CustomerID:  "cust-1",
		PlanID:      "plan-1",
		ExternalID:  "ext-sub-1",
		BillingTime: "calendar",
	}
	assert.Equal(t, "cust-1", input.CustomerID)
	assert.Equal(t, "plan-1", input.PlanID)
	assert.Equal(t, "ext-sub-1", input.ExternalID)
}

func TestListFilter_ZeroValue(t *testing.T) {
	var f subscriptions.ListFilter
	assert.Equal(t, 0, f.Page)
	assert.Equal(t, 0, f.PerPage)
	assert.Nil(t, f.Status)
}
