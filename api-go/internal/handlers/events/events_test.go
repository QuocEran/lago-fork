package events_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	eventhandlers "github.com/getlago/lago/api-go/internal/handlers/events"
	"github.com/getlago/lago/api-go/internal/middleware"
	"github.com/getlago/lago/api-go/internal/models"
	eventservices "github.com/getlago/lago/api-go/internal/services/events"
)

type mockEventsService struct {
	ingestFn      func(ctx context.Context, organizationID string, input eventservices.IngestEventInput) (*eventservices.IngestedEvent, error)
	ingestBatchFn func(ctx context.Context, organizationID string, inputs []eventservices.IngestEventInput) ([]eventservices.IngestedEvent, error)
	listFn        func(ctx context.Context, organizationID string, filter eventservices.ListEventsFilter) ([]models.Event, *eventservices.Pagination, error)
}

func (m *mockEventsService) Ingest(ctx context.Context, organizationID string, input eventservices.IngestEventInput) (*eventservices.IngestedEvent, error) {
	return m.ingestFn(ctx, organizationID, input)
}

func (m *mockEventsService) IngestBatch(ctx context.Context, organizationID string, inputs []eventservices.IngestEventInput) ([]eventservices.IngestedEvent, error) {
	return m.ingestBatchFn(ctx, organizationID, inputs)
}

func (m *mockEventsService) List(ctx context.Context, organizationID string, filter eventservices.ListEventsFilter) ([]models.Event, *eventservices.Pagination, error) {
	if m.listFn != nil {
		return m.listFn(ctx, organizationID, filter)
	}
	return nil, &eventservices.Pagination{CurrentPage: 1, TotalPages: 1}, nil
}

func TestCreateEventRejectsMalformedPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mockEventsService{
		ingestFn: func(_ context.Context, _ string, _ eventservices.IngestEventInput) (*eventservices.IngestedEvent, error) {
			t.Fatalf("service should not be called for malformed payload")
			return nil, nil
		},
		ingestBatchFn: func(_ context.Context, _ string, _ []eventservices.IngestEventInput) ([]eventservices.IngestedEvent, error) {
			return nil, nil
		},
	}

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.GinKeyOrganizationID, "org-1")
		c.Next()
	})
	r.POST("/api/v1/events", eventhandlers.Create(svc))

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewBufferString(`{"event":`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "validation_error")
}

func TestCreateEventReturnsSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mockEventsService{
		ingestFn: func(_ context.Context, organizationID string, input eventservices.IngestEventInput) (*eventservices.IngestedEvent, error) {
			assert.Equal(t, "org-1", organizationID)
			assert.Equal(t, "tx-1", input.TransactionID)
			assert.Equal(t, "usage.created", input.Code)
			timestamp := time.Now().UTC()
			return &eventservices.IngestedEvent{
				Event: &models.Event{
					SoftDeleteModel: models.SoftDeleteModel{BaseModel: models.BaseModel{ID: "evt-1"}},
					OrganizationID:  organizationID,
					TransactionID:  input.TransactionID,
					Code:           input.Code,
					Timestamp:      &timestamp,
				},
				Created: true,
			}, nil
		},
		ingestBatchFn: func(_ context.Context, _ string, _ []eventservices.IngestEventInput) ([]eventservices.IngestedEvent, error) {
			return nil, nil
		},
	}

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.GinKeyOrganizationID, "org-1")
		c.Next()
	})
	r.POST("/api/v1/events", eventhandlers.Create(svc))

	body := `{"event":{"transaction_id":"tx-1","code":"usage.created","properties":{"units":12}}}`
	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewBufferString(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "\"lago_id\":\"evt-1\"")
	assert.Contains(t, w.Body.String(), "\"created\":true")
}

func TestCreateEventSupportsIdempotentDuplicate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	callCount := 0
	svc := &mockEventsService{
		ingestFn: func(_ context.Context, organizationID string, input eventservices.IngestEventInput) (*eventservices.IngestedEvent, error) {
			callCount++
			created := callCount == 1
			return &eventservices.IngestedEvent{
				Event: &models.Event{
					SoftDeleteModel: models.SoftDeleteModel{BaseModel: models.BaseModel{ID: "evt-duplicate"}},
					OrganizationID:  organizationID,
					TransactionID:   input.TransactionID,
					Code:            input.Code,
				},
				Created: created,
			}, nil
		},
		ingestBatchFn: func(_ context.Context, _ string, _ []eventservices.IngestEventInput) ([]eventservices.IngestedEvent, error) {
			return nil, nil
		},
	}

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.GinKeyOrganizationID, "org-1")
		c.Next()
	})
	r.POST("/api/v1/events", eventhandlers.Create(svc))

	body := `{"event":{"transaction_id":"tx-dup","code":"usage.created"}}`

	first := httptest.NewRecorder()
	firstReq, err := http.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewBufferString(body))
	require.NoError(t, err)
	firstReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(first, firstReq)

	second := httptest.NewRecorder()
	secondReq, err := http.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewBufferString(body))
	require.NoError(t, err)
	secondReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(second, secondReq)

	assert.Equal(t, http.StatusCreated, first.Code)
	assert.Equal(t, http.StatusCreated, second.Code)
	assert.Contains(t, first.Body.String(), "\"created\":true")
	assert.Contains(t, second.Body.String(), "\"created\":false")
}

func TestListEventsReturnsPagedResults(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Now().UTC()
	mockEvents := []models.Event{
		{
			SoftDeleteModel: models.SoftDeleteModel{BaseModel: models.BaseModel{ID: "evt-1"}},
			OrganizationID:  "org-1",
			TransactionID:  "tx-1",
			Code:            "api_calls",
			Timestamp:       &now,
		},
	}
	mockPagination := &eventservices.Pagination{
		CurrentPage: 1,
		TotalPages:  1,
		TotalCount:  1,
	}

	svc := &mockEventsService{
		ingestFn: func(_ context.Context, _ string, _ eventservices.IngestEventInput) (*eventservices.IngestedEvent, error) {
			return nil, nil
		},
		ingestBatchFn: func(_ context.Context, _ string, _ []eventservices.IngestEventInput) ([]eventservices.IngestedEvent, error) {
			return nil, nil
		},
		listFn: func(_ context.Context, organizationID string, _ eventservices.ListEventsFilter) ([]models.Event, *eventservices.Pagination, error) {
			assert.Equal(t, "org-1", organizationID)
			return mockEvents, mockPagination, nil
		},
	}

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.GinKeyOrganizationID, "org-1")
		c.Next()
	})
	r.GET("/api/v1/events", eventhandlers.List(svc))

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/api/v1/events?code=api_calls&page=1&per_page=20", nil)
	require.NoError(t, err)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "\"lago_id\":\"evt-1\"")
	assert.Contains(t, w.Body.String(), "\"total_count\":1")
	assert.Contains(t, w.Body.String(), "\"current_page\":1")
}

func TestEstimateFeesRejectsInvalidPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mockEventsService{}

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.GinKeyOrganizationID, "org-1")
		c.Next()
	})
	r.GET("/api/v1/events/estimate_fees", eventhandlers.EstimateFees(svc))

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/api/v1/events/estimate_fees", bytes.NewBufferString(`{}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "validation_error")
}

func TestEstimateFeesReturnsEmptyFees(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mockEventsService{}

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.GinKeyOrganizationID, "org-1")
		c.Next()
	})
	r.GET("/api/v1/events/estimate_fees", eventhandlers.EstimateFees(svc))

	body := `{"event":{"code":"api_calls","external_subscription_id":"sub-123"}}`
	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/api/v1/events/estimate_fees", bytes.NewBufferString(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "\"fees\":[]")
}
