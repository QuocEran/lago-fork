package subscriptions_test

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

	subhandlers "github.com/getlago/lago/api-go/internal/handlers/subscriptions"
	"github.com/getlago/lago/api-go/internal/middleware"
	"github.com/getlago/lago/api-go/internal/models"
	subsvc "github.com/getlago/lago/api-go/internal/services/subscriptions"
)

// ── mock service ─────────────────────────────────────────────────────────────

type mockSubscriptionService struct {
	createFn        func(ctx context.Context, orgID string, input subsvc.CreateInput) (*models.Subscription, error)
	getByIDFn       func(ctx context.Context, orgID, id string) (*models.Subscription, error)
	getByExtIDFn    func(ctx context.Context, orgID, externalID string) (*models.Subscription, error)
	listFn          func(ctx context.Context, orgID string, filter subsvc.ListFilter) (*subsvc.ListResult, error)
	updateFn        func(ctx context.Context, orgID, externalID string, input subsvc.UpdateInput) (*models.Subscription, error)
	terminateFn     func(ctx context.Context, orgID, id string) (*models.Subscription, error)
}

func (m *mockSubscriptionService) Create(ctx context.Context, orgID string, input subsvc.CreateInput) (*models.Subscription, error) {
	return m.createFn(ctx, orgID, input)
}
func (m *mockSubscriptionService) GetByID(ctx context.Context, orgID, id string) (*models.Subscription, error) {
	return m.getByIDFn(ctx, orgID, id)
}
func (m *mockSubscriptionService) GetByExternalID(ctx context.Context, orgID, externalID string) (*models.Subscription, error) {
	return m.getByExtIDFn(ctx, orgID, externalID)
}
func (m *mockSubscriptionService) List(ctx context.Context, orgID string, filter subsvc.ListFilter) (*subsvc.ListResult, error) {
	return m.listFn(ctx, orgID, filter)
}
func (m *mockSubscriptionService) Update(ctx context.Context, orgID, externalID string, input subsvc.UpdateInput) (*models.Subscription, error) {
	return m.updateFn(ctx, orgID, externalID, input)
}
func (m *mockSubscriptionService) Terminate(ctx context.Context, orgID, id string) (*models.Subscription, error) {
	return m.terminateFn(ctx, orgID, id)
}

// ── router setup ──────────────────────────────────────────────────────────────

func newTestRouter(svc subsvc.Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.GinKeyOrganizationID, "org-123")
		c.Next()
	})
	r.POST("/api/v1/subscriptions", subhandlers.Create(svc))
	r.GET("/api/v1/subscriptions", subhandlers.Index(svc))
	r.GET("/api/v1/subscriptions/:external_id", subhandlers.Show(svc))
	r.PUT("/api/v1/subscriptions/:external_id", subhandlers.Update(svc))
	r.DELETE("/api/v1/subscriptions/:external_id", subhandlers.Terminate(svc))
	return r
}

func makeSubscription() *models.Subscription {
	now := time.Now().UTC()
	name := "Test Sub"
	return &models.Subscription{
		BaseModel: models.BaseModel{
			ID:        "sub-uuid-1",
			CreatedAt: now,
			UpdatedAt: now,
		},
		ExternalID:  "ext-sub-1",
		CustomerID:  "cust-uuid-1",
		PlanID:      "plan-uuid-1",
		Name:        &name,
		Status:      models.SubscriptionStatusActive,
		BillingTime: models.BillingTimeCalendar,
	}
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestCreate_Success(t *testing.T) {
	inputSub := makeSubscription()
	svc := &mockSubscriptionService{
		createFn: func(_ context.Context, orgID string, input subsvc.CreateInput) (*models.Subscription, error) {
			assert.Equal(t, "org-123", orgID)
			assert.Equal(t, "cust-uuid-1", input.CustomerID)
			assert.Equal(t, "plan-uuid-1", input.PlanID)
			assert.Equal(t, "ext-sub-1", input.ExternalID)
			return inputSub, nil
		},
	}

	body, err := json.Marshal(map[string]any{
		"subscription": map[string]any{
			"customer_id": "cust-uuid-1",
			"plan_id":     "plan-uuid-1",
			"external_id": "ext-sub-1",
		},
	})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/subscriptions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	newTestRouter(svc).ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	sub := resp["subscription"].(map[string]any)
	assert.Equal(t, "sub-uuid-1", sub["lago_id"])
	assert.Equal(t, "ext-sub-1", sub["external_id"])
	assert.Equal(t, "active", sub["status"])
}

func TestCreate_Conflict(t *testing.T) {
	svc := &mockSubscriptionService{
		createFn: func(_ context.Context, _ string, _ subsvc.CreateInput) (*models.Subscription, error) {
			return nil, subsvc.ErrExternalIDConflict
		},
	}
	body, _ := json.Marshal(map[string]any{
		"subscription": map[string]any{
			"customer_id": "cust-uuid-1",
			"plan_id":     "plan-uuid-1",
			"external_id": "ext-sub-1",
		},
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/subscriptions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	newTestRouter(svc).ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestCreate_BadRequest(t *testing.T) {
	svc := &mockSubscriptionService{}
	body, _ := json.Marshal(map[string]any{"subscription": map[string]any{}})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/subscriptions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	newTestRouter(svc).ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestShow_Success(t *testing.T) {
	sub := makeSubscription()
	svc := &mockSubscriptionService{
		getByExtIDFn: func(_ context.Context, orgID, externalID string) (*models.Subscription, error) {
			assert.Equal(t, "org-123", orgID)
			assert.Equal(t, "ext-sub-1", externalID)
			return sub, nil
		},
	}
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/subscriptions/ext-sub-1", nil)
	newTestRouter(svc).ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestShow_NotFound(t *testing.T) {
	svc := &mockSubscriptionService{
		getByExtIDFn: func(_ context.Context, _, _ string) (*models.Subscription, error) {
			return nil, subsvc.ErrSubscriptionNotFound
		},
	}
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/subscriptions/nonexistent", nil)
	newTestRouter(svc).ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestIndex_Success(t *testing.T) {
	subs := []models.Subscription{*makeSubscription()}
	svc := &mockSubscriptionService{
		listFn: func(_ context.Context, orgID string, _ subsvc.ListFilter) (*subsvc.ListResult, error) {
			assert.Equal(t, "org-123", orgID)
			return &subsvc.ListResult{Subscriptions: subs, TotalCount: 1, TotalPages: 1, CurrentPage: 1}, nil
		},
	}
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/subscriptions", nil)
	newTestRouter(svc).ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	slist := resp["subscriptions"].([]any)
	assert.Len(t, slist, 1)
}

func TestTerminate_Success(t *testing.T) {
	sub := makeSubscription()
	sub.Status = models.SubscriptionStatusTerminated
	now := time.Now().UTC()
	sub.TerminatedAt = &now

	svc := &mockSubscriptionService{
		getByExtIDFn: func(_ context.Context, _, _ string) (*models.Subscription, error) {
			return makeSubscription(), nil
		},
		terminateFn: func(_ context.Context, orgID, id string) (*models.Subscription, error) {
			assert.Equal(t, "org-123", orgID)
			return sub, nil
		},
	}
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/subscriptions/ext-sub-1", nil)
	newTestRouter(svc).ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTerminate_NotFound(t *testing.T) {
	svc := &mockSubscriptionService{
		getByExtIDFn: func(_ context.Context, _, _ string) (*models.Subscription, error) {
			return nil, subsvc.ErrSubscriptionNotFound
		},
		terminateFn: func(_ context.Context, _, _ string) (*models.Subscription, error) {
			return nil, subsvc.ErrSubscriptionNotFound
		},
	}
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/subscriptions/nonexistent", nil)
	newTestRouter(svc).ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}
