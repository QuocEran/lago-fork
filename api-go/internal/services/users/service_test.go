package users_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/getlago/lago/api-go/internal/services/users"
)

func TestLoginResult_Fields(t *testing.T) {
	r := &users.LoginResult{Token: "jwt-abc"}
	assert.Equal(t, "jwt-abc", r.Token)
}

func TestRegisterResult_Fields(t *testing.T) {
	r := &users.RegisterResult{Token: "jwt-xyz"}
	assert.Equal(t, "jwt-xyz", r.Token)
}

func TestSentinelErrors(t *testing.T) {
	assert.NotEmpty(t, users.ErrInvalidCredentials.Error())
	assert.NotEmpty(t, users.ErrLoginMethodNotAuthorized.Error())
	assert.NotEmpty(t, users.ErrSignupDisabled.Error())
	assert.NotEmpty(t, users.ErrUserAlreadyExists.Error())
}
