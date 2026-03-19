package webhooks_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/getlago/lago/api-go/internal/services/webhooks"
)

func TestErrWebhookNotFound(t *testing.T) {
	assert.Equal(t, "webhook not found", webhooks.ErrWebhookNotFound.Error())
}
