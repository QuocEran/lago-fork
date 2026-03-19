package webhookendpoints_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wehandlers "github.com/getlago/lago/api-go/internal/handlers/webhookendpoints"
	"github.com/getlago/lago/api-go/internal/middleware"
	"github.com/getlago/lago/api-go/internal/models"
	wesvc "github.com/getlago/lago/api-go/internal/services/webhookendpoints"
)

// ── mock service ──────────────────────────────────────────────────────────────

type mockService struct {
	createFn   func(ctx context.Context, orgID string, params wesvc.CreateParams) (*models.WebhookEndpoint, error)
	listFn     func(ctx context.Context, orgID string, page, limit int) ([]models.WebhookEndpoint, int64, error)
	getByIDFn  func(ctx context.Context, orgID, id string) (*models.WebhookEndpoint, error)
	updateFn   func(ctx context.Context, orgID, id string, params wesvc.UpdateParams) (*models.WebhookEndpoint, error)
	deleteFn   func(ctx context.Context, orgID, id string) (*models.WebhookEndpoint, error)
}

func (m *mockService) Create(ctx context.Context, orgID string, p wesvc.CreateParams) (*models.WebhookEndpoint, error) {
	return m.createFn(ctx, orgID, p)
}
func (m *mockService) List(ctx context.Context, orgID string, page, limit int) ([]models.WebhookEndpoint, int64, error) {
	return m.listFn(ctx, orgID, page, limit)
}
func (m *mockService) GetByID(ctx context.Context, orgID, id string) (*models.WebhookEndpoint, error) {
	return m.getByIDFn(ctx, orgID, id)
}
func (m *mockService) Update(ctx context.Context, orgID, id string, p wesvc.UpdateParams) (*models.WebhookEndpoint, error) {
	return m.updateFn(ctx, orgID, id, p)
}
func (m *mockService) Delete(ctx context.Context, orgID, id string) (*models.WebhookEndpoint, error) {
	return m.deleteFn(ctx, orgID, id)
}

// ── helpers ───────────────────────────────────────────────────────────────────

func buildRouter(svc wesvc.Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.GinKeyOrganizationID, "org-123")
		c.Next()
	})
	r.POST("/api/v1/webhook_endpoints", wehandlers.Create(svc))
	r.GET("/api/v1/webhook_endpoints", wehandlers.Index(svc))
	r.GET("/api/v1/webhook_endpoints/event_types", wehandlers.EventTypes())
	r.GET("/api/v1/webhook_endpoints/:id", wehandlers.Show(svc))
	r.PUT("/api/v1/webhook_endpoints/:id", wehandlers.Update(svc))
	r.DELETE("/api/v1/webhook_endpoints/:id", wehandlers.Destroy(svc))
	return r
}

func sampleEndpoint() *models.WebhookEndpoint {
	return &models.WebhookEndpoint{
		BaseModel:      models.BaseModel{ID: "ep-1", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		OrganizationID: "org-123",
		WebhookURL:     "https://example.com/hook",
		SignatureAlgo:  models.WebhookSignatureAlgoHMACSHA256,
	}
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestCreate_Success(t *testing.T) {
	svc := &mockService{
		createFn: func(_ context.Context, _ string, _ wesvc.CreateParams) (*models.WebhookEndpoint, error) {
			return sampleEndpoint(), nil
		},
	}
	r := buildRouter(svc)

	inputBody := `{"webhook_endpoint":{"webhook_url":"https://example.com/hook"}}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/webhook_endpoints", bytes.NewBufferString(inputBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	ep := resp["webhook_endpoint"].(map[string]any)
	assert.Equal(t, "ep-1", ep["lago_id"])
	assert.Equal(t, "hmac_sha_256", ep["signature_algo"])
}

func TestCreate_MaxReached(t *testing.T) {
	svc := &mockService{
		createFn: func(_ context.Context, _ string, _ wesvc.CreateParams) (*models.WebhookEndpoint, error) {
			return nil, wesvc.ErrMaxEndpointsReached
		},
	}
	r := buildRouter(svc)

	inputBody := `{"webhook_endpoint":{"webhook_url":"https://x.com/hook"}}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/webhook_endpoints", bytes.NewBufferString(inputBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestCreate_URLConflict(t *testing.T) {
	svc := &mockService{
		createFn: func(_ context.Context, _ string, _ wesvc.CreateParams) (*models.WebhookEndpoint, error) {
			return nil, wesvc.ErrWebhookURLConflict
		},
	}
	r := buildRouter(svc)

	inputBody := `{"webhook_endpoint":{"webhook_url":"https://x.com/hook"}}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/webhook_endpoints", bytes.NewBufferString(inputBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestIndex_Success(t *testing.T) {
	eps := []models.WebhookEndpoint{*sampleEndpoint()}
	svc := &mockService{
		listFn: func(_ context.Context, _ string, _, _ int) ([]models.WebhookEndpoint, int64, error) {
			return eps, 1, nil
		},
	}
	r := buildRouter(svc)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/webhook_endpoints", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	list := resp["webhook_endpoints"].([]any)
	assert.Len(t, list, 1)
}

func TestShow_NotFound(t *testing.T) {
	svc := &mockService{
		getByIDFn: func(_ context.Context, _, _ string) (*models.WebhookEndpoint, error) {
			return nil, wesvc.ErrWebhookEndpointNotFound
		},
	}
	r := buildRouter(svc)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/webhook_endpoints/bad-id", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdate_Success(t *testing.T) {
	ep := sampleEndpoint()
	ep.WebhookURL = "https://new.example.com/hook"
	svc := &mockService{
		updateFn: func(_ context.Context, _, _ string, _ wesvc.UpdateParams) (*models.WebhookEndpoint, error) {
			return ep, nil
		},
	}
	r := buildRouter(svc)

	inputBody := `{"webhook_endpoint":{"webhook_url":"https://new.example.com/hook"}}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/webhook_endpoints/ep-1", bytes.NewBufferString(inputBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDestroy_Success(t *testing.T) {
	svc := &mockService{
		deleteFn: func(_ context.Context, _, _ string) (*models.WebhookEndpoint, error) {
			return sampleEndpoint(), nil
		},
	}
	r := buildRouter(svc)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/webhook_endpoints/ep-1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestEventTypes_ReturnsCatalog(t *testing.T) {
	svc := &mockService{}
	r := buildRouter(svc)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/webhook_endpoints/event_types", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	types := resp["event_types"].([]any)
	assert.NotEmpty(t, types)
}
