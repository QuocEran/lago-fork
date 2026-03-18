package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/getlago/lago/api-go/internal/middleware"
	"github.com/getlago/lago/api-go/internal/models"
)

func newPermissionRouter(
	resource string,
	mode string,
	permissions models.JSONBMap,
	integrations []string,
) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.GinKeyAPIKeyPermissions, permissions)
		c.Set(middleware.GinKeyOrganizationPremiumIntegrations, integrations)
		c.Next()
	})
	r.Any("/resource", middleware.RequirePermission(resource, mode), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	return r
}

func TestRequirePermission_AllowsWhenFeatureDisabled(t *testing.T) {
	r := newPermissionRouter(
		"invoice",
		"",
		models.JSONBMap{"invoice": []any{"write"}},
		[]string{"security_logs"},
	)

	req, _ := http.NewRequest(http.MethodGet, "/resource", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequirePermission_RejectsMissingResourcePermission(t *testing.T) {
	r := newPermissionRouter(
		"invoice",
		"",
		models.JSONBMap{"customer": []any{"read", "write"}},
		[]string{"api_permissions"},
	)

	req, _ := http.NewRequest(http.MethodGet, "/resource", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "read_action_not_allowed_for_invoice")
}

func TestRequirePermission_AllowsReadWhenModePresent(t *testing.T) {
	r := newPermissionRouter(
		"invoice",
		"",
		models.JSONBMap{"invoice": []any{"read"}},
		[]string{"api_permissions"},
	)

	req, _ := http.NewRequest(http.MethodGet, "/resource", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequirePermission_InfersWriteModeForNonGet(t *testing.T) {
	r := newPermissionRouter(
		"invoice",
		"",
		models.JSONBMap{"invoice": []any{"read"}},
		[]string{"api_permissions"},
	)

	req, _ := http.NewRequest(http.MethodPost, "/resource", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "write_action_not_allowed_for_invoice")
}

func TestRequirePermission_NilPermissionsAllowAll(t *testing.T) {
	r := newPermissionRouter(
		"organization",
		"write",
		nil,
		[]string{"api_permissions"},
	)

	req, _ := http.NewRequest(http.MethodPut, "/resource", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
