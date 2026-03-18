package billablemetrics_test

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

	bmhandlers "github.com/getlago/lago/api-go/internal/handlers/billable_metrics"
	"github.com/getlago/lago/api-go/internal/middleware"
	"github.com/getlago/lago/api-go/internal/models"
	bmsvc "github.com/getlago/lago/api-go/internal/services/billable_metrics"
)

// --- mock service ---

type mockService struct {
	createFn   func(ctx context.Context, orgID string, input bmsvc.CreateInput) (*models.BillableMetric, error)
	listFn     func(ctx context.Context, orgID string, filter bmsvc.ListFilter) (*bmsvc.ListResult, error)
	getCodeFn  func(ctx context.Context, orgID string, code string) (*models.BillableMetric, error)
	getIDFn    func(ctx context.Context, orgID string, id string) (*models.BillableMetric, error)
	updateFn   func(ctx context.Context, orgID string, code string, input bmsvc.UpdateInput) (*models.BillableMetric, error)
	deleteFn   func(ctx context.Context, orgID string, code string) (*models.BillableMetric, error)
}

func (m *mockService) Create(ctx context.Context, orgID string, input bmsvc.CreateInput) (*models.BillableMetric, error) {
	return m.createFn(ctx, orgID, input)
}

func (m *mockService) List(ctx context.Context, orgID string, filter bmsvc.ListFilter) (*bmsvc.ListResult, error) {
	return m.listFn(ctx, orgID, filter)
}

func (m *mockService) GetByCode(ctx context.Context, orgID string, code string) (*models.BillableMetric, error) {
	return m.getCodeFn(ctx, orgID, code)
}

func (m *mockService) GetByID(ctx context.Context, orgID string, id string) (*models.BillableMetric, error) {
	return m.getIDFn(ctx, orgID, id)
}

func (m *mockService) Update(ctx context.Context, orgID string, code string, input bmsvc.UpdateInput) (*models.BillableMetric, error) {
	return m.updateFn(ctx, orgID, code, input)
}

func (m *mockService) Delete(ctx context.Context, orgID string, code string) (*models.BillableMetric, error) {
	return m.deleteFn(ctx, orgID, code)
}

func newTestRouter(svc bmsvc.Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.GinKeyOrganizationID, "org-123")
		c.Next()
	})
	r.POST("/billable_metrics", bmhandlers.Create(svc))
	r.GET("/billable_metrics", bmhandlers.Index(svc))
	r.GET("/billable_metrics/:code", bmhandlers.Show(svc))
	r.PUT("/billable_metrics/:code", bmhandlers.Update(svc))
	r.DELETE("/billable_metrics/:code", bmhandlers.Destroy(svc))
	return r
}

func makeMetric(id, code string) *models.BillableMetric {
	fieldName := "amount"
	return &models.BillableMetric{
		SoftDeleteModel: models.SoftDeleteModel{
			BaseModel: models.BaseModel{
				ID:        id,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
		OrganizationID:  "org-123",
		Name:            "Test Metric",
		Code:            code,
		AggregationType: models.AggregationTypeSum,
		FieldName:       &fieldName,
		Filters:         []models.BillableMetricFilter{},
	}
}

// --- tests ---

func TestCreate_RejectsMalformedPayload(t *testing.T) {
	svc := &mockService{
		createFn: func(_ context.Context, _ string, _ bmsvc.CreateInput) (*models.BillableMetric, error) {
			t.Fatal("service should not be called")
			return nil, nil
		},
	}
	r := newTestRouter(svc)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/billable_metrics", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreate_Returns200OnSuccess(t *testing.T) {
	expectedMetric := makeMetric("uuid-1", "my-metric")
	svc := &mockService{
		createFn: func(_ context.Context, _ string, _ bmsvc.CreateInput) (*models.BillableMetric, error) {
			return expectedMetric, nil
		},
	}
	r := newTestRouter(svc)

	body := `{"billable_metric": {"name": "Test Metric", "code": "my-metric", "aggregation_type": "sum_agg", "field_name": "amount"}}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/billable_metrics", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	bm := resp["billable_metric"].(map[string]any)
	assert.Equal(t, "uuid-1", bm["lago_id"])
	assert.Equal(t, "my-metric", bm["code"])
}

func TestCreate_Returns422OnValidationError(t *testing.T) {
	svc := &mockService{
		createFn: func(_ context.Context, _ string, _ bmsvc.CreateInput) (*models.BillableMetric, error) {
			return nil, &bmsvc.ValidationError{Message: "code is required"}
		},
	}
	r := newTestRouter(svc)

	body := `{"billable_metric": {"name": "Test", "code": "x", "aggregation_type": "sum_agg"}}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/billable_metrics", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestCreate_Returns422OnCodeConflict(t *testing.T) {
	svc := &mockService{
		createFn: func(_ context.Context, _ string, _ bmsvc.CreateInput) (*models.BillableMetric, error) {
			return nil, bmsvc.ErrBillableMetricCodeConflict
		},
	}
	r := newTestRouter(svc)

	body := `{"billable_metric": {"name": "Test", "code": "dup", "aggregation_type": "sum_agg", "field_name": "amount"}}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/billable_metrics", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestIndex_ReturnsPaginatedList(t *testing.T) {
	metrics := []models.BillableMetric{*makeMetric("id-1", "code-1"), *makeMetric("id-2", "code-2")}
	svc := &mockService{
		listFn: func(_ context.Context, _ string, _ bmsvc.ListFilter) (*bmsvc.ListResult, error) {
			return &bmsvc.ListResult{
				Metrics:     metrics,
				TotalCount:  2,
				TotalPages:  1,
				CurrentPage: 1,
			}, nil
		},
	}
	r := newTestRouter(svc)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/billable_metrics", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	list := resp["billable_metrics"].([]any)
	assert.Len(t, list, 2)
}

func TestShow_Returns404OnMissing(t *testing.T) {
	svc := &mockService{
		getCodeFn: func(_ context.Context, _ string, _ string) (*models.BillableMetric, error) {
			return nil, bmsvc.ErrBillableMetricNotFound
		},
	}
	r := newTestRouter(svc)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/billable_metrics/missing-code", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestShow_Returns200OnFound(t *testing.T) {
	inputMetric := makeMetric("id-1", "found-code")
	svc := &mockService{
		getCodeFn: func(_ context.Context, _ string, code string) (*models.BillableMetric, error) {
			return inputMetric, nil
		},
	}
	r := newTestRouter(svc)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/billable_metrics/found-code", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	bm := resp["billable_metric"].(map[string]any)
	assert.Equal(t, "found-code", bm["code"])
}

func TestUpdate_Returns200OnSuccess(t *testing.T) {
	updatedMetric := makeMetric("id-1", "my-metric")
	updatedMetric.Name = "Updated Name"
	svc := &mockService{
		updateFn: func(_ context.Context, _ string, _ string, _ bmsvc.UpdateInput) (*models.BillableMetric, error) {
			return updatedMetric, nil
		},
	}
	r := newTestRouter(svc)

	body := `{"billable_metric": {"name": "Updated Name"}}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/billable_metrics/my-metric", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	bm := resp["billable_metric"].(map[string]any)
	assert.Equal(t, "Updated Name", bm["name"])
}

func TestDestroy_Returns200OnSuccess(t *testing.T) {
	inputMetric := makeMetric("id-1", "to-delete")
	svc := &mockService{
		deleteFn: func(_ context.Context, _ string, _ string) (*models.BillableMetric, error) {
			return inputMetric, nil
		},
	}
	r := newTestRouter(svc)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/billable_metrics/to-delete", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestCreate_MissingOrganizationContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// No middleware setting org ID.
	r.POST("/billable_metrics", bmhandlers.Create(&mockService{
		createFn: func(_ context.Context, _ string, _ bmsvc.CreateInput) (*models.BillableMetric, error) {
			t.Fatal("should not reach service")
			return nil, nil
		},
	}))

	body := `{"billable_metric": {"name": "Test", "code": "x", "aggregation_type": "sum_agg"}}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/billable_metrics", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
