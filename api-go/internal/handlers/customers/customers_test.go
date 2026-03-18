package customers_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	customerhandlers "github.com/getlago/lago/api-go/internal/handlers/customers"
	"github.com/getlago/lago/api-go/internal/middleware"
	"github.com/getlago/lago/api-go/internal/models"
	customerservices "github.com/getlago/lago/api-go/internal/services/customers"
)

type mockCustomersService struct {
	createFn             func(ctx context.Context, organizationID string, input customerservices.CreateCustomerInput) (*models.Customer, error)
	listFn               func(ctx context.Context, organizationID string) ([]models.Customer, error)
	getByExternalIDFn    func(ctx context.Context, organizationID string, externalID string) (*models.Customer, error)
	deleteByExternalIDFn func(ctx context.Context, organizationID string, externalID string) (*models.Customer, error)
	generatePortalURLFn  func(ctx context.Context, organizationID string, externalID string) (string, error)
}

func (m *mockCustomersService) Create(ctx context.Context, organizationID string, input customerservices.CreateCustomerInput) (*models.Customer, error) {
	return m.createFn(ctx, organizationID, input)
}

func (m *mockCustomersService) List(ctx context.Context, organizationID string) ([]models.Customer, error) {
	return m.listFn(ctx, organizationID)
}

func (m *mockCustomersService) GetByExternalID(ctx context.Context, organizationID string, externalID string) (*models.Customer, error) {
	return m.getByExternalIDFn(ctx, organizationID, externalID)
}

func (m *mockCustomersService) DeleteByExternalID(ctx context.Context, organizationID string, externalID string) (*models.Customer, error) {
	return m.deleteByExternalIDFn(ctx, organizationID, externalID)
}

func (m *mockCustomersService) GeneratePortalURL(ctx context.Context, organizationID string, externalID string) (string, error) {
	return m.generatePortalURLFn(ctx, organizationID, externalID)
}

func TestCreateCustomerRejectsMalformedPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mockCustomersService{
		createFn: func(_ context.Context, _ string, _ customerservices.CreateCustomerInput) (*models.Customer, error) {
			t.Fatalf("service should not be called for malformed payload")
			return nil, nil
		},
		listFn: func(_ context.Context, _ string) ([]models.Customer, error) { return nil, nil },
		getByExternalIDFn: func(_ context.Context, _ string, _ string) (*models.Customer, error) {
			return nil, nil
		},
		deleteByExternalIDFn: func(_ context.Context, _ string, _ string) (*models.Customer, error) {
			return nil, nil
		},
		generatePortalURLFn: func(_ context.Context, _ string, _ string) (string, error) { return "", nil },
	}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(middleware.GinKeyOrganizationID, "org-1")
		c.Next()
	})
	router.POST("/api/v1/customers", customerhandlers.Create(svc))

	response := httptest.NewRecorder()
	request, err := http.NewRequest(http.MethodPost, "/api/v1/customers", bytes.NewBufferString(`{"customer":`))
	require.NoError(t, err)
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(response, request)

	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Contains(t, response.Body.String(), "validation_error")
}

func TestCreateCustomerReturnsSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mockCustomersService{
		createFn: func(_ context.Context, organizationID string, input customerservices.CreateCustomerInput) (*models.Customer, error) {
			assert.Equal(t, "org-1", organizationID)
			assert.Equal(t, "cust-ext-1", input.ExternalID)
			return &models.Customer{
				SoftDeleteModel: models.SoftDeleteModel{BaseModel: models.BaseModel{ID: "customer-1"}},
				ExternalID:      input.ExternalID,
				Name:            input.Name,
				Currency:        input.Currency,
				Metadata: []models.CustomerMetadata{
					{Key: "tier", Value: "gold", DisplayInInvoice: true},
				},
			}, nil
		},
		listFn: func(_ context.Context, _ string) ([]models.Customer, error) { return nil, nil },
		getByExternalIDFn: func(_ context.Context, _ string, _ string) (*models.Customer, error) {
			return nil, nil
		},
		deleteByExternalIDFn: func(_ context.Context, _ string, _ string) (*models.Customer, error) {
			return nil, nil
		},
		generatePortalURLFn: func(_ context.Context, _ string, _ string) (string, error) { return "", nil },
	}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(middleware.GinKeyOrganizationID, "org-1")
		c.Next()
	})
	router.POST("/api/v1/customers", customerhandlers.Create(svc))

	body := `{"customer":{"external_id":"cust-ext-1","name":"Acme","currency":"EUR","metadata":[{"key":"tier","value":"gold","display_in_invoice":true}]}}`
	response := httptest.NewRecorder()
	request, err := http.NewRequest(http.MethodPost, "/api/v1/customers", bytes.NewBufferString(body))
	require.NoError(t, err)
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(response, request)

	assert.Equal(t, http.StatusOK, response.Code)
	assert.Contains(t, response.Body.String(), `"external_id":"cust-ext-1"`)
	assert.Contains(t, response.Body.String(), `"metadata":[{"key":"tier","value":"gold","display_in_invoice":true}]`)
}

func TestPortalURLReturnsSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mockCustomersService{
		createFn: func(_ context.Context, _ string, _ customerservices.CreateCustomerInput) (*models.Customer, error) {
			return nil, nil
		},
		listFn: func(_ context.Context, _ string) ([]models.Customer, error) { return nil, nil },
		getByExternalIDFn: func(_ context.Context, _ string, _ string) (*models.Customer, error) {
			return nil, nil
		},
		deleteByExternalIDFn: func(_ context.Context, _ string, _ string) (*models.Customer, error) {
			return nil, nil
		},
		generatePortalURLFn: func(_ context.Context, organizationID string, externalID string) (string, error) {
			assert.Equal(t, "org-1", organizationID)
			assert.Equal(t, "cust-ext-1", externalID)
			return "https://portal.example.com/customer-portal/cust-ext-1?token=abc", nil
		},
	}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(middleware.GinKeyOrganizationID, "org-1")
		c.Next()
	})
	router.GET("/api/v1/customers/:external_id/portal_url", customerhandlers.PortalURL(svc))

	response := httptest.NewRecorder()
	request, err := http.NewRequest(http.MethodGet, "/api/v1/customers/cust-ext-1/portal_url", nil)
	require.NoError(t, err)
	router.ServeHTTP(response, request)

	assert.Equal(t, http.StatusOK, response.Code)
	assert.Contains(t, response.Body.String(), "/customer-portal/")
}

func TestShowCustomerNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mockCustomersService{
		createFn: func(_ context.Context, _ string, _ customerservices.CreateCustomerInput) (*models.Customer, error) {
			return nil, nil
		},
		listFn: func(_ context.Context, _ string) ([]models.Customer, error) { return nil, nil },
		getByExternalIDFn: func(_ context.Context, _ string, _ string) (*models.Customer, error) {
			return nil, customerservices.ErrCustomerNotFound
		},
		deleteByExternalIDFn: func(_ context.Context, _ string, _ string) (*models.Customer, error) {
			return nil, nil
		},
		generatePortalURLFn: func(_ context.Context, _ string, _ string) (string, error) { return "", nil },
	}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(middleware.GinKeyOrganizationID, "org-1")
		c.Next()
	})
	router.GET("/api/v1/customers/:external_id", customerhandlers.Show(svc))

	response := httptest.NewRecorder()
	request, err := http.NewRequest(http.MethodGet, "/api/v1/customers/missing-customer", nil)
	require.NoError(t, err)
	router.ServeHTTP(response, request)

	assert.Equal(t, http.StatusNotFound, response.Code)
	assert.Contains(t, response.Body.String(), "customer_not_found")
}
