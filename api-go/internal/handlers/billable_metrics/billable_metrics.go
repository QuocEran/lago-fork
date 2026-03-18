package billablemetrics

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/getlago/lago/api-go/internal/middleware"
	"github.com/getlago/lago/api-go/internal/models"
	bmsvc "github.com/getlago/lago/api-go/internal/services/billable_metrics"
)

type filterRequest struct {
	Key    string   `json:"key"`
	Values []string `json:"values"`
}

type createBillableMetricRequest struct {
	Name              string          `json:"name" binding:"required"`
	Code              string          `json:"code" binding:"required"`
	Description       *string         `json:"description"`
	AggregationType   string          `json:"aggregation_type" binding:"required"`
	FieldName         *string         `json:"field_name"`
	Recurring         *bool           `json:"recurring"`
	Expression        *string         `json:"expression"`
	CustomAggregator  *string         `json:"custom_aggregator"`
	WeightedInterval  *string         `json:"weighted_interval"`
	RoundingFunction  *string         `json:"rounding_function"`
	RoundingPrecision *int            `json:"rounding_precision"`
	Filters           []filterRequest `json:"filters"`
}

type createRequestEnvelope struct {
	BillableMetric createBillableMetricRequest `json:"billable_metric" binding:"required"`
}

type updateBillableMetricRequest struct {
	Name              *string         `json:"name"`
	Description       *string         `json:"description"`
	AggregationType   *string         `json:"aggregation_type"`
	FieldName         *string         `json:"field_name"`
	Recurring         *bool           `json:"recurring"`
	Expression        *string         `json:"expression"`
	CustomAggregator  *string         `json:"custom_aggregator"`
	WeightedInterval  *string         `json:"weighted_interval"`
	RoundingFunction  *string         `json:"rounding_function"`
	RoundingPrecision *int            `json:"rounding_precision"`
	Filters           *[]filterRequest `json:"filters"`
}

type updateRequestEnvelope struct {
	BillableMetric updateBillableMetricRequest `json:"billable_metric" binding:"required"`
}

type filterResponse struct {
	LagoID string   `json:"lago_id"`
	Key    string   `json:"key"`
	Values []string `json:"values"`
}

type billableMetricResponse struct {
	LagoID            string           `json:"lago_id"`
	Name              string           `json:"name"`
	Code              string           `json:"code"`
	Description       *string          `json:"description"`
	AggregationType   string           `json:"aggregation_type"`
	FieldName         *string          `json:"field_name"`
	Recurring         bool             `json:"recurring"`
	Expression        *string          `json:"expression"`
	WeightedInterval  *string          `json:"weighted_interval"`
	RoundingFunction  *string          `json:"rounding_function"`
	RoundingPrecision *int             `json:"rounding_precision"`
	Filters           []filterResponse `json:"filters"`
	CreatedAt         time.Time        `json:"created_at"`
	UpdatedAt         time.Time        `json:"updated_at"`
}

type paginationMeta struct {
	CurrentPage int   `json:"current_page"`
	NextPage    *int  `json:"next_page"`
	PrevPage    *int  `json:"prev_page"`
	TotalPages  int   `json:"total_pages"`
	TotalCount  int64 `json:"total_count"`
}

// Create handles POST /api/v1/billable_metrics.
func Create(svc bmsvc.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		orgID, ok := organizationIDFromContext(c)
		if !ok {
			return
		}

		var req createRequestEnvelope
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "error_code": "validation_error", "error_details": gin.H{"message": err.Error()}})
			return
		}

		input := toCreateInput(req.BillableMetric)
		metric, err := svc.Create(c.Request.Context(), orgID, input)
		if err != nil {
			handleServiceError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"billable_metric": toResponse(metric)})
	}
}

// Index handles GET /api/v1/billable_metrics.
func Index(svc bmsvc.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		orgID, ok := organizationIDFromContext(c)
		if !ok {
			return
		}

		filter := parseListFilter(c)
		result, err := svc.List(c.Request.Context(), orgID, filter)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "error_code": "internal_error", "error_details": gin.H{}})
			return
		}

		items := make([]billableMetricResponse, 0, len(result.Metrics))
		for i := range result.Metrics {
			items = append(items, toResponse(&result.Metrics[i]))
		}

		c.JSON(http.StatusOK, gin.H{
			"billable_metrics": items,
			"meta": paginationMeta{
				CurrentPage: result.CurrentPage,
				NextPage:    result.NextPage,
				PrevPage:    result.PrevPage,
				TotalPages:  result.TotalPages,
				TotalCount:  result.TotalCount,
			},
		})
	}
}

// Show handles GET /api/v1/billable_metrics/:code.
func Show(svc bmsvc.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		orgID, ok := organizationIDFromContext(c)
		if !ok {
			return
		}

		code := c.Param("code")
		metric, err := svc.GetByCode(c.Request.Context(), orgID, code)
		if err != nil {
			handleServiceError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"billable_metric": toResponse(metric)})
	}
}

// Update handles PUT /api/v1/billable_metrics/:code.
func Update(svc bmsvc.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		orgID, ok := organizationIDFromContext(c)
		if !ok {
			return
		}

		var req updateRequestEnvelope
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "error_code": "validation_error", "error_details": gin.H{"message": err.Error()}})
			return
		}

		code := c.Param("code")
		input := toUpdateInput(req.BillableMetric)
		metric, err := svc.Update(c.Request.Context(), orgID, code, input)
		if err != nil {
			handleServiceError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"billable_metric": toResponse(metric)})
	}
}

// Destroy handles DELETE /api/v1/billable_metrics/:code.
func Destroy(svc bmsvc.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		orgID, ok := organizationIDFromContext(c)
		if !ok {
			return
		}

		code := c.Param("code")
		metric, err := svc.Delete(c.Request.Context(), orgID, code)
		if err != nil {
			handleServiceError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"billable_metric": toResponse(metric)})
	}
}

func organizationIDFromContext(c *gin.Context) (string, bool) {
	value, exists := c.Get(middleware.GinKeyOrganizationID)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "unauthorized", "error": "missing_organization_context"})
		return "", false
	}

	orgID, ok := value.(string)
	if !ok || orgID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "unauthorized", "error": "invalid_organization_context"})
		return "", false
	}

	return orgID, true
}

func handleServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, bmsvc.ErrBillableMetricNotFound):
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "error_code": "billable_metric_not_found", "error_details": gin.H{}})
	case errors.Is(err, bmsvc.ErrBillableMetricCodeConflict):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"status": "error", "error_code": "value_already_exist", "error_details": gin.H{"code": []string{"value_already_exist"}}})
	case bmsvc.IsValidationError(err):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"status": "error", "error_code": "validation_error", "error_details": gin.H{"message": err.Error()}})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "error_code": "internal_error", "error_details": gin.H{}})
	}
}

func toCreateInput(req createBillableMetricRequest) bmsvc.CreateInput {
	filters := make([]bmsvc.FilterInput, 0, len(req.Filters))
	for _, f := range req.Filters {
		filters = append(filters, bmsvc.FilterInput{Key: f.Key, Values: f.Values})
	}

	return bmsvc.CreateInput{
		Name:              req.Name,
		Code:              req.Code,
		Description:       req.Description,
		AggregationType:   req.AggregationType,
		FieldName:         req.FieldName,
		Recurring:         req.Recurring,
		Expression:        req.Expression,
		CustomAggregator:  req.CustomAggregator,
		WeightedInterval:  req.WeightedInterval,
		RoundingFunction:  req.RoundingFunction,
		RoundingPrecision: req.RoundingPrecision,
		Filters:           filters,
	}
}

func toUpdateInput(req updateBillableMetricRequest) bmsvc.UpdateInput {
	var filters *[]bmsvc.FilterInput
	if req.Filters != nil {
		mapped := make([]bmsvc.FilterInput, 0, len(*req.Filters))
		for _, f := range *req.Filters {
			mapped = append(mapped, bmsvc.FilterInput{Key: f.Key, Values: f.Values})
		}
		filters = &mapped
	}

	return bmsvc.UpdateInput{
		Name:              req.Name,
		Description:       req.Description,
		AggregationType:   req.AggregationType,
		FieldName:         req.FieldName,
		Recurring:         req.Recurring,
		Expression:        req.Expression,
		CustomAggregator:  req.CustomAggregator,
		WeightedInterval:  req.WeightedInterval,
		RoundingFunction:  req.RoundingFunction,
		RoundingPrecision: req.RoundingPrecision,
		Filters:           filters,
	}
}

func toResponse(m *models.BillableMetric) billableMetricResponse {
	filters := make([]filterResponse, 0, len(m.Filters))
	for i := range m.Filters {
		f := m.Filters[i]
		values := []string(f.Values)
		if values == nil {
			values = []string{}
		}
		filters = append(filters, filterResponse{
			LagoID: f.ID,
			Key:    f.Key,
			Values: values,
		})
	}

	return billableMetricResponse{
		LagoID:            m.ID,
		Name:              m.Name,
		Code:              m.Code,
		Description:       m.Description,
		AggregationType:   models.AggregationTypeToString(m.AggregationType),
		FieldName:         m.FieldName,
		Recurring:         m.Recurring,
		Expression:        m.Expression,
		WeightedInterval:  m.WeightedInterval,
		RoundingFunction:  m.RoundingFunction,
		RoundingPrecision: m.RoundingPrecision,
		Filters:           filters,
		CreatedAt:         m.CreatedAt,
		UpdatedAt:         m.UpdatedAt,
	}
}

func parseListFilter(c *gin.Context) bmsvc.ListFilter {
	f := bmsvc.ListFilter{
		Search: c.Query("search_term"),
	}

	if v := c.Query("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			f.Page = n
		}
	}
	if v := c.Query("per_page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			f.PerPage = n
		}
	}
	if v := c.Query("recurring"); v != "" {
		b := v == "true"
		f.Recurring = &b
	}

	return f
}
