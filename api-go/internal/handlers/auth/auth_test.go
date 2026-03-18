package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authhandlers "github.com/getlago/lago/api-go/internal/handlers/auth"
	"github.com/getlago/lago/api-go/internal/models"
	"github.com/getlago/lago/api-go/internal/services/users"
)

// mockAuthService implements users.AuthService for unit tests.
type mockAuthService struct {
	loginFn    func(ctx context.Context, email, password string) (*users.LoginResult, error)
	registerFn func(ctx context.Context, email, password, orgName string) (*users.RegisterResult, error)
}

func (m *mockAuthService) Login(ctx context.Context, email, password string) (*users.LoginResult, error) {
	return m.loginFn(ctx, email, password)
}

func (m *mockAuthService) Register(ctx context.Context, email, password, orgName string) (*users.RegisterResult, error) {
	return m.registerFn(ctx, email, password, orgName)
}

func newRouter(svc users.AuthService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/users/login", authhandlers.Login(svc))
	r.POST("/users/register", authhandlers.Register(svc))
	return r
}

func postJSON(t *testing.T, r *gin.Engine, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	b, err := json.Marshal(body)
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// ── Login tests ──────────────────────────────────────────────────────────────

func TestLogin_ValidCredentials(t *testing.T) {
	svc := &mockAuthService{
		loginFn: func(_ context.Context, _, _ string) (*users.LoginResult, error) {
			return &users.LoginResult{
				Token: "signed.jwt.token",
				User:  models.User{Email: "alice@example.com"},
			}, nil
		},
	}
	r := newRouter(svc)
	w := postJSON(t, r, "/users/login", map[string]string{"email": "alice@example.com", "password": "secret"})

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "signed.jwt.token")
	assert.Contains(t, w.Body.String(), "alice@example.com")
}

func TestLogin_WrongPassword(t *testing.T) {
	svc := &mockAuthService{
		loginFn: func(_ context.Context, _, _ string) (*users.LoginResult, error) {
			return nil, users.ErrInvalidCredentials
		},
	}
	r := newRouter(svc)
	w := postJSON(t, r, "/users/login", map[string]string{"email": "alice@example.com", "password": "wrong"})

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "incorrect_login_or_password")
}

func TestLogin_LoginMethodNotAuthorized(t *testing.T) {
	svc := &mockAuthService{
		loginFn: func(_ context.Context, _, _ string) (*users.LoginResult, error) {
			return nil, users.ErrLoginMethodNotAuthorized
		},
	}
	r := newRouter(svc)
	w := postJSON(t, r, "/users/login", map[string]string{"email": "alice@example.com", "password": "secret"})

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), "login_method_not_authorized")
}

func TestLogin_MissingFields(t *testing.T) {
	svc := &mockAuthService{
		loginFn: func(_ context.Context, _, _ string) (*users.LoginResult, error) {
			return nil, nil
		},
	}
	r := newRouter(svc)
	w := postJSON(t, r, "/users/login", map[string]string{"email": "alice@example.com"})

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLogin_InternalError(t *testing.T) {
	svc := &mockAuthService{
		loginFn: func(_ context.Context, _, _ string) (*users.LoginResult, error) {
			return nil, errors.New("db connection lost")
		},
	}
	r := newRouter(svc)
	w := postJSON(t, r, "/users/login", map[string]string{"email": "a@b.com", "password": "pw"})

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "internal_error")
}

// ── Register tests ───────────────────────────────────────────────────────────

func TestRegister_HappyPath(t *testing.T) {
	svc := &mockAuthService{
		registerFn: func(_ context.Context, _, _, orgName string) (*users.RegisterResult, error) {
			return &users.RegisterResult{
				Token:        "register.jwt.token",
				User:         models.User{Email: "bob@example.com"},
				Organization: models.Organization{Name: orgName},
				Membership:   models.Membership{},
			}, nil
		},
	}
	r := newRouter(svc)
	w := postJSON(t, r, "/users/register", map[string]string{
		"email":             "bob@example.com",
		"password":          "secret123",
		"organization_name": "Acme Inc",
	})

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "register.jwt.token")
	assert.Contains(t, w.Body.String(), "Acme Inc")
}

func TestRegister_SignupDisabled(t *testing.T) {
	svc := &mockAuthService{
		registerFn: func(_ context.Context, _, _, _ string) (*users.RegisterResult, error) {
			return nil, users.ErrSignupDisabled
		},
	}
	r := newRouter(svc)
	w := postJSON(t, r, "/users/register", map[string]string{
		"email": "bob@example.com", "password": "pw", "organization_name": "Acme",
	})

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "signup_disabled")
}

func TestRegister_DuplicateEmail(t *testing.T) {
	svc := &mockAuthService{
		registerFn: func(_ context.Context, _, _, _ string) (*users.RegisterResult, error) {
			return nil, users.ErrUserAlreadyExists
		},
	}
	r := newRouter(svc)
	w := postJSON(t, r, "/users/register", map[string]string{
		"email": "bob@example.com", "password": "pw", "organization_name": "Acme",
	})

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), "user_already_exists")
}

func TestRegister_MissingFields(t *testing.T) {
	svc := &mockAuthService{
		registerFn: func(_ context.Context, _, _, _ string) (*users.RegisterResult, error) {
			return nil, nil
		},
	}
	r := newRouter(svc)
	w := postJSON(t, r, "/users/register", map[string]string{"email": "a@b.com"})

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRegister_InternalError(t *testing.T) {
	svc := &mockAuthService{
		registerFn: func(_ context.Context, _, _, _ string) (*users.RegisterResult, error) {
			return nil, errors.New("db unavailable")
		},
	}
	r := newRouter(svc)
	w := postJSON(t, r, "/users/register", map[string]string{
		"email": "a@b.com", "password": "pw", "organization_name": "Org",
	})

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "internal_error")
}
