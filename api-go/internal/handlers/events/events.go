package events

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/getlago/lago/api-go/internal/middleware"
	"github.com/getlago/lago/api-go/internal/models"
	eventservices "github.com/getlago/lago/api-go/internal/services/events"
)

type eventRequest struct {
	TransactionID          string         `json:"transaction_id"`
	Code                   string         `json:"code"`
	Timestamp              string         `json:"timestamp"`
	Properties             map[string]any `json:"properties"`
	ExternalCustomerID     *string        `json:"external_customer_id"`
	ExternalSubscriptionID *string        `json:"external_subscription_id"`
}

type ingestEventRequestEnvelope struct {
	Event eventRequest `json:"event" binding:"required"`
}

type ingestBatchRequestEnvelope struct {
	Events []eventRequest `json:"events" binding:"required"`
}

type eventResponse struct {
	LagoID        string `json:"lago_id"`
	TransactionID string `json:"transaction_id"`
	Code          string `json:"code"`
	Created       bool   `json:"created"`
}

func Create(svc eventservices.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		organizationID, ok := organizationIDFromContext(c)
		if !ok {
			return
		}

		var req ingestEventRequestEnvelope
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "error_code": "validation_error", "error_details": gin.H{"message": err.Error()}})
			return
		}

		input, err := toServiceInput(req.Event)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "error_code": "validation_error", "error_details": gin.H{"message": err.Error()}})
			return
		}

		ingested, err := svc.Ingest(c.Request.Context(), organizationID, input)
		if err != nil {
			handleServiceError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"event": toResponse(ingested)})
	}
}

func CreateBatch(svc eventservices.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		organizationID, ok := organizationIDFromContext(c)
		if !ok {
			return
		}

		var req ingestBatchRequestEnvelope
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "error_code": "validation_error", "error_details": gin.H{"message": err.Error()}})
			return
		}

		inputs := make([]eventservices.IngestEventInput, 0, len(req.Events))
		for _, item := range req.Events {
			input, err := toServiceInput(item)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"status": "error", "error_code": "validation_error", "error_details": gin.H{"message": err.Error()}})
				return
			}
			inputs = append(inputs, input)
		}

		results, err := svc.IngestBatch(c.Request.Context(), organizationID, inputs)
		if err != nil {
			handleServiceError(c, err)
			return
		}

		responseItems := make([]eventResponse, 0, len(results))
		for _, result := range results {
			copied := result
			responseItems = append(responseItems, toResponse(&copied))
		}

		c.JSON(http.StatusOK, gin.H{"events": responseItems})
	}
}

func organizationIDFromContext(c *gin.Context) (string, bool) {
	value, exists := c.Get(middleware.GinKeyOrganizationID)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "unauthorized", "error": "missing_organization_context"})
		return "", false
	}

	organizationID, ok := value.(string)
	if !ok || organizationID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "unauthorized", "error": "invalid_organization_context"})
		return "", false
	}

	return organizationID, true
}

func toServiceInput(input eventRequest) (eventservices.IngestEventInput, error) {
	timestamp := time.Time{}
	if input.Timestamp != "" {
		parsed, err := time.Parse(time.RFC3339, input.Timestamp)
		if err != nil {
			return eventservices.IngestEventInput{}, err
		}
		timestamp = parsed
	}

	return eventservices.IngestEventInput{
		TransactionID:          input.TransactionID,
		Code:                   input.Code,
		Timestamp:              timestamp,
		Properties:             models.JSONBMap(input.Properties),
		ExternalCustomerID:     input.ExternalCustomerID,
		ExternalSubscriptionID: input.ExternalSubscriptionID,
	}, nil
}

func handleServiceError(c *gin.Context, err error) {
	if eventservices.IsValidationError(err) {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"status": "error", "error_code": "validation_error", "error_details": gin.H{"message": err.Error()}})
		return
	}

	c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "error_code": "internal_error", "error_details": gin.H{}})
}

func toResponse(result *eventservices.IngestedEvent) eventResponse {
	return eventResponse{
		LagoID:        result.Event.ID,
		TransactionID: result.Event.TransactionID,
		Code:          result.Event.Code,
		Created:       result.Created,
	}
}

type eventListItem struct {
	LagoID                  string         `json:"lago_id"`
	TransactionID           string         `json:"transaction_id"`
	LagoCustomerID          *string        `json:"lago_customer_id"`
	Code                    string         `json:"code"`
	Timestamp               *time.Time     `json:"timestamp"`
	PreciseTotalAmountCents *string        `json:"precise_total_amount_cents"`
	Properties              map[string]any `json:"properties"`
	LagoSubscriptionID      *string        `json:"lago_subscription_id"`
	ExternalSubscriptionID  *string        `json:"external_subscription_id"`
	CreatedAt               time.Time      `json:"created_at"`
}

type paginationMeta struct {
	CurrentPage int  `json:"current_page"`
	NextPage    *int `json:"next_page"`
	PrevPage    *int `json:"prev_page"`
	TotalPages  int  `json:"total_pages"`
	TotalCount  int64 `json:"total_count"`
}

// List handles GET /api/v1/events.
func List(svc eventservices.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		organizationID, ok := organizationIDFromContext(c)
		if !ok {
			return
		}

		filter := parseListFilter(c)

		events, pagination, err := svc.List(c.Request.Context(), organizationID, filter)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "error_code": "internal_error", "error_details": gin.H{}})
			return
		}

		items := make([]eventListItem, 0, len(events))
		for _, e := range events {
			copied := e
			items = append(items, toListItem(&copied))
		}

		c.JSON(http.StatusOK, gin.H{
			"events": items,
			"meta": paginationMeta{
				CurrentPage: pagination.CurrentPage,
				NextPage:    pagination.NextPage,
				PrevPage:    pagination.PrevPage,
				TotalPages:  pagination.TotalPages,
				TotalCount:  pagination.TotalCount,
			},
		})
	}
}

func parseListFilter(c *gin.Context) eventservices.ListEventsFilter {
	filter := eventservices.ListEventsFilter{
		Code:                   c.Query("code"),
		ExternalSubscriptionID: c.Query("external_subscription_id"),
	}

	if v := c.Query("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			filter.Page = n
		}
	}
	if v := c.Query("per_page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			filter.PerPage = n
		}
	}
	if v := c.Query("timestamp_from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			filter.TimestampFrom = &t
		}
	}
	if v := c.Query("timestamp_to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			filter.TimestampTo = &t
		}
	}

	return filter
}

func toListItem(e *models.Event) eventListItem {
	properties := map[string]any{}
	if e.Properties != nil {
		properties = map[string]any(e.Properties)
	}

	return eventListItem{
		LagoID:                  e.ID,
		TransactionID:           e.TransactionID,
		LagoCustomerID:          e.CustomerID,
		Code:                    e.Code,
		Timestamp:               e.Timestamp,
		PreciseTotalAmountCents: e.PreciseTotalAmountCents,
		Properties:              properties,
		LagoSubscriptionID:      e.SubscriptionID,
		ExternalSubscriptionID:  e.ExternalSubscriptionID,
		CreatedAt:               e.CreatedAt,
	}
}

type estimateFeesRequest struct {
	Event estimateFeesEventInput `json:"event" binding:"required"`
}

type estimateFeesEventInput struct {
	Code                   string         `json:"code" binding:"required"`
	ExternalSubscriptionID string         `json:"external_subscription_id" binding:"required"`
	Timestamp              string         `json:"timestamp"`
	PreciseTotalAmountCents *string       `json:"precise_total_amount_cents"`
	TransactionID          string         `json:"transaction_id"`
	Properties             map[string]any `json:"properties"`
}

// EstimateFees handles GET /api/v1/events/estimate_fees.
// Full fee calculation requires subscription and charge models (Phase 5).
// This endpoint validates input and returns an empty fees list until then.
func EstimateFees(_ eventservices.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		_, ok := organizationIDFromContext(c)
		if !ok {
			return
		}

		var req estimateFeesRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":         "error",
				"error_code":     "validation_error",
				"error_details":  gin.H{"message": err.Error()},
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{"fees": []any{}})
	}
}
