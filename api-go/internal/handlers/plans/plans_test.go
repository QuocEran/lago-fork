package plans_test

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

	planhandlers "github.com/getlago/lago/api-go/internal/handlers/plans"
	"github.com/getlago/lago/api-go/internal/middleware"
	"github.com/getlago/lago/api-go/internal/models"
	plansvc "github.com/getlago/lago/api-go/internal/services/plans"
)

// ── mock service ─────────────────────────────────────────────────────────────

type mockPlanService struct {
	createFn  func(ctx context.Context, orgID string, input plansvc.CreateInput) (*models.Plan, error)
	listFn    func(ctx context.Context, orgID string, filter plansvc.ListFilter) (*plansvc.ListResult, error)
	getCodeFn func(ctx context.Context, orgID string, code string) (*models.Plan, error)
	getIDFn   func(ctx context.Context, orgID string, id string) (*models.Plan, error)
	updateFn  func(ctx context.Context, orgID string, code string, input plansvc.UpdateInput) (*models.Plan, error)
	deleteFn  func(ctx context.Context, orgID string, code string) (*models.Plan, error)
}

func (m *mockPlanService) Create(ctx context.Context, orgID string, input plansvc.CreateInput) (*models.Plan, error) {
	return m.createFn(ctx, orgID, input)
}
func (m *mockPlanService) List(ctx context.Context, orgID string, filter plansvc.ListFilter) (*plansvc.ListResult, error) {
	return m.listFn(ctx, orgID, filter)
}
func (m *mockPlanService) GetByCode(ctx context.Context, orgID string, code string) (*models.Plan, error) {
	return m.getCodeFn(ctx, orgID, code)
}
func (m *mockPlanService) GetByID(ctx context.Context, orgID string, id string) (*models.Plan, error) {
	return m.getIDFn(ctx, orgID, id)
}
func (m *mockPlanService) Update(ctx context.Context, orgID string, code string, input plansvc.UpdateInput) (*models.Plan, error) {
	return m.updateFn(ctx, orgID, code, input)
}
func (m *mockPlanService) Delete(ctx context.Context, orgID string, code string) (*models.Plan, error) {
	return m.deleteFn(ctx, orgID, code)
}

// ── router setup ──────────────────────────────────────────────────────────────

func newTestRouter(svc plansvc.Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.GinKeyOrganizationID, "org-123")
		c.Next()
	})
	r.POST("/plans", planhandlers.Create(svc))
	r.GET("/plans", planhandlers.Index(svc))
	r.GET("/plans/:code", planhandlers.Show(svc))
	r.PUT("/plans/:code", planhandlers.Update(svc))
	r.DELETE("/plans/:code", planhandlers.Destroy(svc))
	return r
}

// ── helpers ───────────────────────────────────────────────────────────────────

func makePlan(id, code string) *models.Plan {
	return &models.Plan{
		SoftDeleteModel: models.SoftDeleteModel{
			BaseModel: models.BaseModel{
				ID:        id,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
		OrganizationID: "org-123",
		Name:           "Test Plan",
		Code:           code,
		Interval:       models.PlanIntervalMonthly,
		AmountCents:    1000,
		AmountCurrency: "USD",
		PayInAdvance:   false,
		Charges:        []models.Charge{},
	}
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestCreate_Success(t *testing.T) {
	inputPlan := makePlan("plan-1", "basic")
	svc := &mockPlanService{
		createFn: func(_ context.Context, orgID string, input plansvc.CreateInput) (*models.Plan, error) {
			assert.Equal(t, "org-123", orgID)
			assert.Equal(t, "basic", input.Code)
			return inputPlan, nil
		},
	}

	body, _ := json.Marshal(map[string]any{
		"plan": map[string]any{
			"name":            "Test Plan",
			"code":            "basic",
			"interval":        "monthly",
			"amount_cents":    1000,
			"amount_currency": "USD",
			"pay_in_advance":  false,
			"charges":         []any{},
		},
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/plans", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	newTestRouter(svc).ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	plan := resp["plan"].(map[string]any)
	assert.Equal(t, "plan-1", plan["lago_id"])
	assert.Equal(t, "basic", plan["code"])
	assert.Equal(t, "monthly", plan["interval"])
}

func TestCreate_ValidationError(t *testing.T) {
	svc := &mockPlanService{
		createFn: func(_ context.Context, _ string, _ plansvc.CreateInput) (*models.Plan, error) {
			return nil, &plansvc.ValidationError{Message: "code is required"}
		},
	}

	body, _ := json.Marshal(map[string]any{
		"plan": map[string]any{
			"name":            "Test Plan",
			"code":            "x",
			"interval":        "monthly",
			"amount_cents":    1000,
			"amount_currency": "USD",
		},
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/plans", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	newTestRouter(svc).ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestCreate_CodeConflict(t *testing.T) {
	svc := &mockPlanService{
		createFn: func(_ context.Context, _ string, _ plansvc.CreateInput) (*models.Plan, error) {
			return nil, plansvc.ErrPlanCodeConflict
		},
	}

	body, _ := json.Marshal(map[string]any{
		"plan": map[string]any{
			"name": "P", "code": "dup", "interval": "monthly",
			"amount_cents": 100, "amount_currency": "USD",
		},
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/plans", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	newTestRouter(svc).ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestIndex_Success(t *testing.T) {
	svc := &mockPlanService{
		listFn: func(_ context.Context, _ string, _ plansvc.ListFilter) (*plansvc.ListResult, error) {
			return &plansvc.ListResult{
				Plans:       []models.Plan{*makePlan("p1", "plan-a"), *makePlan("p2", "plan-b")},
				TotalCount:  2,
				TotalPages:  1,
				CurrentPage: 1,
			}, nil
		},
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/plans", nil)
	newTestRouter(svc).ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	plans := resp["plans"].([]any)
	assert.Len(t, plans, 2)
}

func TestShow_Success(t *testing.T) {
	svc := &mockPlanService{
		getCodeFn: func(_ context.Context, _ string, code string) (*models.Plan, error) {
			return makePlan("p1", code), nil
		},
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/plans/basic", nil)
	newTestRouter(svc).ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	plan := resp["plan"].(map[string]any)
	assert.Equal(t, "basic", plan["code"])
}

func TestShow_NotFound(t *testing.T) {
	svc := &mockPlanService{
		getCodeFn: func(_ context.Context, _ string, _ string) (*models.Plan, error) {
			return nil, plansvc.ErrPlanNotFound
		},
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/plans/missing", nil)
	newTestRouter(svc).ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdate_Success(t *testing.T) {
	svc := &mockPlanService{
		updateFn: func(_ context.Context, _ string, code string, _ plansvc.UpdateInput) (*models.Plan, error) {
			return makePlan("p1", code), nil
		},
	}

	body, _ := json.Marshal(map[string]any{
		"plan": map[string]any{"name": "Updated"},
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/plans/basic", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	newTestRouter(svc).ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestDestroy_Success(t *testing.T) {
	svc := &mockPlanService{
		deleteFn: func(_ context.Context, _ string, code string) (*models.Plan, error) {
			return makePlan("p1", code), nil
		},
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/plans/basic", nil)
	newTestRouter(svc).ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestDestroy_NotFound(t *testing.T) {
	svc := &mockPlanService{
		deleteFn: func(_ context.Context, _ string, _ string) (*models.Plan, error) {
			return nil, plansvc.ErrPlanNotFound
		},
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/plans/missing", nil)
	newTestRouter(svc).ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
