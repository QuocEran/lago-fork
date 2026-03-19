package webhookendpoints_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	wesvc "github.com/getlago/lago/api-go/internal/services/webhookendpoints"
)

func TestValidationError_Error(t *testing.T) {
	err := &wesvc.ValidationError{Field: "webhook_url", Message: "cannot be blank"}
	assert.Contains(t, err.Error(), "webhook_url")
	assert.Contains(t, err.Error(), "cannot be blank")
	assert.True(t, wesvc.IsValidationError(err))
}

func TestIsValidationError_ReturnsFalseForOtherErrors(t *testing.T) {
	assert.False(t, wesvc.IsValidationError(wesvc.ErrWebhookEndpointNotFound))
	assert.False(t, wesvc.IsValidationError(wesvc.ErrWebhookURLConflict))
	assert.False(t, wesvc.IsValidationError(wesvc.ErrMaxEndpointsReached))
}

func TestCreateParams_ZeroValue(t *testing.T) {
	var p wesvc.CreateParams
	assert.Empty(t, p.WebhookURL)
	assert.Nil(t, p.ID)
}

func TestUpdateParams_ZeroValue(t *testing.T) {
	var p wesvc.UpdateParams
	assert.Nil(t, p.WebhookURL)
	assert.Nil(t, p.SignatureAlgo)
}
