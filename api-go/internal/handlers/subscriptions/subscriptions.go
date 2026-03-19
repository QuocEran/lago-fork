package subscriptions

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/getlago/lago/api-go/internal/middleware"
	"github.com/getlago/lago/api-go/internal/models"
	subsvc "github.com/getlago/lago/api-go/internal/services/subscriptions"
)

// ── request types ─────────────────────────────────────────────────────────────

type createSubscriptionRequest struct {
	CustomerID     string     `json:"customer_id"     binding:"required"`
	PlanID         string     `json:"plan_id"         binding:"required"`
	ExternalID     string     `json:"external_id"     binding:"required"`
	Name           *string    `json:"name"`
	BillingTime    string     `json:"billing_time"`
	SubscriptionAt *time.Time `json:"subscription_at"`
	EndingAt       *time.Time `json:"ending_at"`
}

type createEnvelope struct {
	Subscription createSubscriptionRequest `json:"subscription" binding:"required"`
}

type updateSubscriptionRequest struct {
	Name           *string    `json:"name"`
	SubscriptionAt *time.Time `json:"subscription_at"`
	EndingAt       *time.Time `json:"ending_at"`
}

type updateEnvelope struct {
	Subscription updateSubscriptionRequest `json:"subscription" binding:"required"`
}

// ── response types ─────────────────────────────────────────────────────────────

type subscriptionResponse struct {
	LagoID         string     `json:"lago_id"`
	ExternalID     string     `json:"external_id"`
	CustomerID     string     `json:"lago_customer_id"`
	PlanID         string     `json:"lago_plan_id"`
	Name           *string    `json:"name"`
	Status         string     `json:"status"`
	BillingTime    string     `json:"billing_time"`
	SubscriptionAt *time.Time `json:"subscription_at"`
	StartedAt      *time.Time `json:"started_at"`
	EndingAt       *time.Time `json:"ending_at"`
	CanceledAt     *time.Time `json:"canceled_at"`
	TerminatedAt   *time.Time `json:"terminated_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type paginationMeta struct {
	CurrentPage int   `json:"current_page"`
	NextPage    *int  `json:"next_page"`
	PrevPage    *int  `json:"prev_page"`
	TotalPages  int   `json:"total_pages"`
	TotalCount  int64 `json:"total_count"`
}

// ── handlers ──────────────────────────────────────────────────────────────────

// Create handles POST /api/v1/subscriptions.
func Create(svc subsvc.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		orgID, ok := organizationIDFromContext(c)
		if !ok {
			return
		}
		var req createEnvelope
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "error_code": "validation_error", "error_details": gin.H{"message": err.Error()}})
			return
		}
		input := toCreateInput(req.Subscription)
		sub, err := svc.Create(c.Request.Context(), orgID, input)
		if err != nil {
			handleServiceError(c, err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{"subscription": toResponse(sub)})
	}
}

// Index handles GET /api/v1/subscriptions.
func Index(svc subsvc.Service) gin.HandlerFunc {
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
		items := make([]subscriptionResponse, 0, len(result.Subscriptions))
		for i := range result.Subscriptions {
			items = append(items, toResponse(&result.Subscriptions[i]))
		}
		c.JSON(http.StatusOK, gin.H{
			"subscriptions": items,
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

// Show handles GET /api/v1/subscriptions/:external_id.
func Show(svc subsvc.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		orgID, ok := organizationIDFromContext(c)
		if !ok {
			return
		}
		externalID := c.Param("external_id")
		sub, err := svc.GetByExternalID(c.Request.Context(), orgID, externalID)
		if err != nil {
			handleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"subscription": toResponse(sub)})
	}
}

// Update handles PUT /api/v1/subscriptions/:external_id.
func Update(svc subsvc.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		orgID, ok := organizationIDFromContext(c)
		if !ok {
			return
		}
		var req updateEnvelope
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "error_code": "validation_error", "error_details": gin.H{"message": err.Error()}})
			return
		}
		externalID := c.Param("external_id")
		input := subsvc.UpdateInput{
			Name:           req.Subscription.Name,
			SubscriptionAt: req.Subscription.SubscriptionAt,
			EndingAt:       req.Subscription.EndingAt,
		}
		sub, err := svc.Update(c.Request.Context(), orgID, externalID, input)
		if err != nil {
			handleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"subscription": toResponse(sub)})
	}
}

// Terminate handles DELETE /api/v1/subscriptions/:external_id.
// Looks up the active/pending subscription by external_id, then terminates by internal ID.
func Terminate(svc subsvc.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		orgID, ok := organizationIDFromContext(c)
		if !ok {
			return
		}
		externalID := c.Param("external_id")
		// Resolve by external_id first, then terminate by internal ID.
		sub, err := svc.GetByExternalID(c.Request.Context(), orgID, externalID)
		if err != nil {
			handleServiceError(c, err)
			return
		}
		terminated, err := svc.Terminate(c.Request.Context(), orgID, sub.ID)
		if err != nil {
			handleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"subscription": toResponse(terminated)})
	}
}

// CurrentUsage handles GET /api/v1/customers/:external_id/current_usage.
// Returns a stub response — full aggregation is out of scope for Phase 7.
func CurrentUsage() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"customer_usage": gin.H{
				"from_datetime":      nil,
				"to_datetime":        nil,
				"issuing_date":       nil,
				"currency":           nil,
				"amount_cents":       0,
				"taxes_amount_cents": 0,
				"total_amount_cents": 0,
				"charges_usage":      []any{},
			},
		})
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

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
	case errors.Is(err, subsvc.ErrSubscriptionNotFound):
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "error_code": "subscription_not_found", "error_details": gin.H{}})
	case errors.Is(err, subsvc.ErrExternalIDConflict):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"status": "error", "error_code": "value_already_exist", "error_details": gin.H{"external_id": []string{"value_already_exist"}}})
	case subsvc.IsValidationError(err):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"status": "error", "error_code": "validation_error", "error_details": gin.H{"message": err.Error()}})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "error_code": "internal_error", "error_details": gin.H{}})
	}
}

func toCreateInput(req createSubscriptionRequest) subsvc.CreateInput {
	billingTime := req.BillingTime
	if billingTime == "" {
		billingTime = "calendar"
	}
	return subsvc.CreateInput{
		CustomerID:     req.CustomerID,
		PlanID:         req.PlanID,
		ExternalID:     req.ExternalID,
		Name:           req.Name,
		BillingTime:    billingTime,
		SubscriptionAt: req.SubscriptionAt,
		EndingAt:       req.EndingAt,
	}
}

func toResponse(sub *models.Subscription) subscriptionResponse {
	return subscriptionResponse{
		LagoID:         sub.ID,
		ExternalID:     sub.ExternalID,
		CustomerID:     sub.CustomerID,
		PlanID:         sub.PlanID,
		Name:           sub.Name,
		Status:         models.SubscriptionStatusToString(sub.Status),
		BillingTime:    models.BillingTimeToString(sub.BillingTime),
		SubscriptionAt: sub.SubscriptionAt,
		StartedAt:      sub.StartedAt,
		EndingAt:       sub.EndingAt,
		CanceledAt:     sub.CanceledAt,
		TerminatedAt:   sub.TerminatedAt,
		CreatedAt:      sub.CreatedAt,
		UpdatedAt:      sub.UpdatedAt,
	}
}

func parseListFilter(c *gin.Context) subsvc.ListFilter {
	f := subsvc.ListFilter{
		CustomerExternalID: c.Query("external_customer_id"),
	}
	if v := c.QueryArray("status[]"); len(v) > 0 {
		f.Status = v
	} else if v := c.Query("status"); v != "" {
		f.Status = []string{v}
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
	return f
}
