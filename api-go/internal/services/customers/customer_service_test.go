package customers

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateCreateInput(t *testing.T) {
	currency := "EUR"
	err := validateCreateInput("org-1", CreateCustomerInput{
		ExternalID: "cust-1",
		Currency:   &currency,
		Metadata: []MetadataInput{
			{
				Key:   "source",
				Value: "api",
			},
		},
	})

	require.NoError(t, err)
}

func TestValidateCreateInput_InvalidCurrency(t *testing.T) {
	currency := "EURO"
	err := validateCreateInput("org-1", CreateCustomerInput{
		ExternalID: "cust-1",
		Currency:   &currency,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "currency must be a 3-letter ISO code")
}

func TestValidateCreateInput_MissingMetadataKey(t *testing.T) {
	err := validateCreateInput("org-1", CreateCustomerInput{
		ExternalID: "cust-1",
		Metadata: []MetadataInput{
			{
				Key:   " ",
				Value: "v",
			},
		},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "metadata key is required")
}

func TestBuildPortalURL(t *testing.T) {
	t.Setenv("CUSTOMER_PORTAL_BASE_URL", "https://portal.example.com/customer-portal")

	actualPortalURL := buildPortalURL("customer-ext-1", "token-abc")
	assert.Equal(t, "https://portal.example.com/customer-portal/customer-ext-1?token=token-abc", actualPortalURL)
}

func TestBuildPortalToken(t *testing.T) {
	actualToken := buildPortalToken("customer-id", "hmac-key")
	assert.NotEmpty(t, actualToken)
	assert.False(t, strings.Contains(actualToken, "."))
}
