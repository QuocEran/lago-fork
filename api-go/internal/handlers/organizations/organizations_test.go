package organizations_test

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

	organizationhandlers "github.com/getlago/lago/api-go/internal/handlers/organizations"
	"github.com/getlago/lago/api-go/internal/middleware"
	"github.com/getlago/lago/api-go/internal/models"
	organizationservices "github.com/getlago/lago/api-go/internal/services/organizations"
)

type mockOrganizationService struct {
	getFn    func(ctx context.Context, organizationID string) (*models.Organization, error)
	updateFn func(ctx context.Context, organizationID string, input organizationservices.UpdateOrganizationInput) (*models.Organization, error)
}

func (m *mockOrganizationService) Get(ctx context.Context, organizationID string) (*models.Organization, error) {
	return m.getFn(ctx, organizationID)
}

func (m *mockOrganizationService) Update(ctx context.Context, organizationID string, input organizationservices.UpdateOrganizationInput) (*models.Organization, error) {
	return m.updateFn(ctx, organizationID, input)
}

func TestShowOrganizationReturnsOrganization(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mockOrganizationService{
		getFn: func(_ context.Context, organizationID string) (*models.Organization, error) {
			assert.Equal(t, "org-1", organizationID)
			webhookURL := "https://example.com/webhook"
			country := "fr"
			return &models.Organization{
				BaseModel:                 models.BaseModel{ID: "org-1"},
				Name:                      "Acme",
				DefaultCurrency:           "EUR",
				WebhookURL:                &webhookURL,
				Country:                   &country,
				Timezone:                  "Europe/Paris",
				NetPaymentTerm:            15,
				FinalizeZeroAmountInvoice: true,
			}, nil
		},
		updateFn: func(_ context.Context, _ string, _ organizationservices.UpdateOrganizationInput) (*models.Organization, error) {
			return nil, errors.New("unexpected call")
		},
	}

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.GinKeyOrganizationID, "org-1")
		c.Next()
	})
	r.GET("/api/v1/organizations", organizationhandlers.Show(svc))

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/api/v1/organizations", nil)
	require.NoError(t, err)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "\"organization\"")
	assert.Contains(t, w.Body.String(), "\"lago_id\":\"org-1\"")
	assert.Contains(t, w.Body.String(), "\"default_currency\":\"EUR\"")
}

func TestUpdateOrganizationReturnsValidationError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mockOrganizationService{
		getFn: func(_ context.Context, _ string) (*models.Organization, error) {
			return nil, errors.New("unexpected call")
		},
		updateFn: func(_ context.Context, _ string, _ organizationservices.UpdateOrganizationInput) (*models.Organization, error) {
			return nil, &organizationservices.ValidationError{Message: "default_currency must be a 3-letter ISO code"}
		},
	}

	body := map[string]any{
		"organization": map[string]any{
			"default_currency": "EURO",
		},
	}
	payload, err := json.Marshal(body)
	require.NoError(t, err)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.GinKeyOrganizationID, "org-1")
		c.Next()
	})
	r.PUT("/api/v1/organizations", organizationhandlers.Update(svc))

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPut, "/api/v1/organizations", bytes.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), "validation_error")
}

func TestOrganizationsRoutesBlockUnauthorizedWrite(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mockOrganizationService{
		getFn: func(_ context.Context, _ string) (*models.Organization, error) {
			return &models.Organization{BaseModel: models.BaseModel{ID: "org-1"}, Name: "Acme", DefaultCurrency: "USD", Timezone: "UTC"}, nil
		},
		updateFn: func(_ context.Context, _ string, _ organizationservices.UpdateOrganizationInput) (*models.Organization, error) {
			return &models.Organization{BaseModel: models.BaseModel{ID: "org-1"}, Name: "Acme", DefaultCurrency: "USD", Timezone: "UTC"}, nil
		},
	}

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.GinKeyOrganizationID, "org-1")
		c.Set(middleware.GinKeyOrganizationPremiumIntegrations, []string{"api_permissions"})
		c.Set(middleware.GinKeyAPIKeyPermissions, models.JSONBMap{"organization": []any{"read"}})
		c.Next()
	})
	r.GET("/api/v1/organizations", middleware.RequirePermission("organization", ""), organizationhandlers.Show(svc))
	r.PUT("/api/v1/organizations", middleware.RequirePermission("organization", ""), organizationhandlers.Update(svc))

	getResponse := httptest.NewRecorder()
	getRequest, err := http.NewRequest(http.MethodGet, "/api/v1/organizations", nil)
	require.NoError(t, err)
	r.ServeHTTP(getResponse, getRequest)
	assert.Equal(t, http.StatusOK, getResponse.Code)

	payload := []byte(`{"organization":{"default_currency":"USD"}}`)
	putResponse := httptest.NewRecorder()
	putRequest, err := http.NewRequest(http.MethodPut, "/api/v1/organizations", bytes.NewReader(payload))
	require.NoError(t, err)
	putRequest.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(putResponse, putRequest)

	assert.Equal(t, http.StatusForbidden, putResponse.Code)
	assert.Contains(t, putResponse.Body.String(), "write_action_not_allowed_for_organization")
}
