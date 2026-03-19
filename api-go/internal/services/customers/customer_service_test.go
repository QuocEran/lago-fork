package customers_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/getlago/lago/api-go/internal/services/customers"
)

// Black-box tests for the customers service use the public API only.

func TestCreateCustomerInput_ValidStruct(t *testing.T) {
	currency := "EUR"
	input := customers.CreateCustomerInput{
		ExternalID: "ext-1",
		Currency:   &currency,
	}
	assert.Equal(t, "ext-1", input.ExternalID)
	assert.NotNil(t, input.Currency)
	assert.Equal(t, "EUR", *input.Currency)
}
