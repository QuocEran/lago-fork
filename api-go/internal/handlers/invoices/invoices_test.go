package invoices_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domain "github.com/getlago/lago/api-go/internal/domain/invoices"
	invoicehandlers "github.com/getlago/lago/api-go/internal/handlers/invoices"
	"github.com/getlago/lago/api-go/internal/middleware"
	"github.com/getlago/lago/api-go/internal/models"
	invoiceservices "github.com/getlago/lago/api-go/internal/services/invoices"
)

type mockInvoicesService struct {
	createFn   func(ctx context.Context, organizationID string, input invoiceservices.CreateInvoiceInput) (*models.Invoice, error)
	listFn     func(ctx context.Context, organizationID string, filter invoiceservices.ListInvoicesFilter) ([]models.Invoice, *invoiceservices.Pagination, error)
	getByIDFn  func(ctx context.Context, organizationID string, id string) (*models.Invoice, error)
	finalizeFn func(ctx context.Context, organizationID string, id string) (*models.Invoice, error)
	voidFn     func(ctx context.Context, organizationID string, id string) (*models.Invoice, error)
}

func (m *mockInvoicesService) Create(ctx context.Context, organizationID string, input invoiceservices.CreateInvoiceInput) (*models.Invoice, error) {
	return m.createFn(ctx, organizationID, input)
}

func (m *mockInvoicesService) List(ctx context.Context, organizationID string, filter invoiceservices.ListInvoicesFilter) ([]models.Invoice, *invoiceservices.Pagination, error) {
	return m.listFn(ctx, organizationID, filter)
}

func (m *mockInvoicesService) GetByID(ctx context.Context, organizationID string, id string) (*models.Invoice, error) {
	return m.getByIDFn(ctx, organizationID, id)
}

func (m *mockInvoicesService) Finalize(ctx context.Context, organizationID string, id string) (*models.Invoice, error) {
	return m.finalizeFn(ctx, organizationID, id)
}

func (m *mockInvoicesService) Void(ctx context.Context, organizationID string, id string) (*models.Invoice, error) {
	return m.voidFn(ctx, organizationID, id)
}

func buildMockSvc() *mockInvoicesService {
	return &mockInvoicesService{
		createFn: func(_ context.Context, _ string, _ invoiceservices.CreateInvoiceInput) (*models.Invoice, error) {
			return nil, nil
		},
		listFn: func(_ context.Context, _ string, _ invoiceservices.ListInvoicesFilter) ([]models.Invoice, *invoiceservices.Pagination, error) {
			return nil, &invoiceservices.Pagination{CurrentPage: 1, TotalPages: 1, TotalCount: 0}, nil
		},
		getByIDFn: func(_ context.Context, _ string, _ string) (*models.Invoice, error) {
			return nil, nil
		},
		finalizeFn: func(_ context.Context, _ string, _ string) (*models.Invoice, error) {
			return nil, nil
		},
		voidFn: func(_ context.Context, _ string, _ string) (*models.Invoice, error) {
			return nil, nil
		},
	}
}

func newTestRouter(svc invoiceservices.Service) *gin.Engine {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(middleware.GinKeyOrganizationID, "org-1")
		c.Next()
	})
	router.POST("/api/v1/invoices", invoicehandlers.Create(svc))
	router.GET("/api/v1/invoices", invoicehandlers.Index(svc))
	router.GET("/api/v1/invoices/:id", invoicehandlers.Show(svc))
	router.PUT("/api/v1/invoices/:id/finalize", invoicehandlers.Finalize(svc))
	router.PUT("/api/v1/invoices/:id/void", invoicehandlers.Void(svc))
	return router
}

func TestCreateInvoiceRejectsMalformedPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := buildMockSvc()
	svc.createFn = func(_ context.Context, _ string, _ invoiceservices.CreateInvoiceInput) (*models.Invoice, error) {
		t.Fatal("service should not be called for malformed payload")
		return nil, nil
	}

	router := newTestRouter(svc)
	response := httptest.NewRecorder()
	request, err := http.NewRequest(http.MethodPost, "/api/v1/invoices", bytes.NewBufferString(`{"invoice":`))
	require.NoError(t, err)
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(response, request)

	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Contains(t, response.Body.String(), "validation_error")
}

func TestCreateInvoiceSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockInvoice := &models.Invoice{
		BaseModel:   models.BaseModel{ID: "invoice-uuid-1"},
		Status:      models.InvoiceStatusDraft,
		Currency:    "EUR",
		InvoiceType: models.InvoiceTypeOneOff,
	}

	svc := buildMockSvc()
	svc.createFn = func(_ context.Context, organizationID string, input invoiceservices.CreateInvoiceInput) (*models.Invoice, error) {
		assert.Equal(t, "org-1", organizationID)
		assert.Equal(t, "cust-1", input.CustomerID)
		return mockInvoice, nil
	}

	router := newTestRouter(svc)
	body := `{"invoice":{"customer_id":"cust-1","billing_entity_id":"billing-1","invoice_type":3,"currency":"EUR"}}`
	response := httptest.NewRecorder()
	request, err := http.NewRequest(http.MethodPost, "/api/v1/invoices", bytes.NewBufferString(body))
	require.NoError(t, err)
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(response, request)

	assert.Equal(t, http.StatusCreated, response.Code)
	assert.Contains(t, response.Body.String(), `"lago_id":"invoice-uuid-1"`)
}

func TestShowInvoiceNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := buildMockSvc()
	svc.getByIDFn = func(_ context.Context, _ string, _ string) (*models.Invoice, error) {
		return nil, invoiceservices.ErrInvoiceNotFound
	}

	router := newTestRouter(svc)
	response := httptest.NewRecorder()
	request, err := http.NewRequest(http.MethodGet, "/api/v1/invoices/missing-id", nil)
	require.NoError(t, err)
	router.ServeHTTP(response, request)

	assert.Equal(t, http.StatusNotFound, response.Code)
	assert.Contains(t, response.Body.String(), "invoice_not_found")
}

func TestFinalizeInvoiceSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockInvoice := &models.Invoice{
		BaseModel: models.BaseModel{ID: "invoice-1"},
		Status:    models.InvoiceStatusFinalized,
		Currency:  "USD",
	}

	svc := buildMockSvc()
	svc.finalizeFn = func(_ context.Context, _ string, id string) (*models.Invoice, error) {
		assert.Equal(t, "invoice-1", id)
		return mockInvoice, nil
	}

	router := newTestRouter(svc)
	response := httptest.NewRecorder()
	request, err := http.NewRequest(http.MethodPut, "/api/v1/invoices/invoice-1/finalize", nil)
	require.NoError(t, err)
	router.ServeHTTP(response, request)

	assert.Equal(t, http.StatusOK, response.Code)
	assert.Contains(t, response.Body.String(), `"status":"finalized"`)
}

func TestFinalizeInvoiceAlreadyFinalized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := buildMockSvc()
	svc.finalizeFn = func(_ context.Context, _ string, _ string) (*models.Invoice, error) {
		return nil, domain.ErrAlreadyFinalized
	}

	router := newTestRouter(svc)
	response := httptest.NewRecorder()
	request, err := http.NewRequest(http.MethodPut, "/api/v1/invoices/invoice-1/finalize", nil)
	require.NoError(t, err)
	router.ServeHTTP(response, request)

	assert.Equal(t, http.StatusUnprocessableEntity, response.Code)
	assert.Contains(t, response.Body.String(), "transition_error")
}

func TestVoidInvoiceSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockInvoice := &models.Invoice{
		BaseModel: models.BaseModel{ID: "invoice-1"},
		Status:    models.InvoiceStatusVoided,
		Currency:  "USD",
	}

	svc := buildMockSvc()
	svc.voidFn = func(_ context.Context, _ string, id string) (*models.Invoice, error) {
		assert.Equal(t, "invoice-1", id)
		return mockInvoice, nil
	}

	router := newTestRouter(svc)
	response := httptest.NewRecorder()
	request, err := http.NewRequest(http.MethodPut, "/api/v1/invoices/invoice-1/void", nil)
	require.NoError(t, err)
	router.ServeHTTP(response, request)

	assert.Equal(t, http.StatusOK, response.Code)
	assert.Contains(t, response.Body.String(), `"status":"voided"`)
}

func TestVoidDraftInvoiceRejected(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := buildMockSvc()
	svc.voidFn = func(_ context.Context, _ string, _ string) (*models.Invoice, error) {
		return nil, domain.ErrCannotVoidDraft
	}

	router := newTestRouter(svc)
	response := httptest.NewRecorder()
	request, err := http.NewRequest(http.MethodPut, "/api/v1/invoices/invoice-1/void", nil)
	require.NoError(t, err)
	router.ServeHTTP(response, request)

	assert.Equal(t, http.StatusUnprocessableEntity, response.Code)
	assert.Contains(t, response.Body.String(), "transition_error")
}
