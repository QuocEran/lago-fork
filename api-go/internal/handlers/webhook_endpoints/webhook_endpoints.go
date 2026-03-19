package webhook_endpoints

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/getlago/lago/api-go/internal/middleware"
	"github.com/getlago/lago/api-go/internal/models"
	wesvc "github.com/getlago/lago/api-go/internal/services/webhook_endpoints"
)

// ── request types ─────────────────────────────────────────────────────────────

type createRequest struct {
	WebhookEndpoint createBody `json:"webhook_endpoint" binding:"required"`
}

type createBody struct {
	ID            *string `json:"id"`
	WebhookURL    string  `json:"webhook_url" binding:"required"`
	SignatureAlgo *int    `json:"signature_algo"`
}

type updateRequest struct {
	WebhookEndpoint updateBody `json:"webhook_endpoint" binding:"required"`
}

type updateBody struct {
	WebhookURL    *string `json:"webhook_url"`
	SignatureAlgo *int    `json:"signature_algo"`
}

// ── response types ─────────────────────────────────────────────────────────────

type endpointResponse struct {
	WebhookEndpoint endpointBody `json:"webhook_endpoint"`
}

type endpointBody struct {
	LagoID             string `json:"lago_id"`
	LagoOrganizationID string `json:"lago_organization_id"`
	WebhookURL         string `json:"webhook_url"`
	SignatureAlgo      string `json:"signature_algo"`
	CreatedAt          string `json:"created_at"`
}

type listResponse struct {
	WebhookEndpoints []endpointBody `json:"webhook_endpoints"`
	Meta             pageMeta       `json:"meta"`
}

type pageMeta struct {
	CurrentPage int   `json:"current_page"`
	NextPage    *int  `json:"next_page"`
	PrevPage    *int  `json:"prev_page"`
	TotalPages  int   `json:"total_pages"`
	TotalCount  int64 `json:"total_count"`
}

// SupportedWebhookEventTypes is the catalog of all known outbound event types.
var SupportedWebhookEventTypes = []string{
	"customer.created",
	"customer.updated",
	"subscription.started",
	"subscription.terminated",
	"invoice.created",
	"invoice.finalized",
	"invoice.payment_status_updated",
	"invoice.paid_credit_added",
	"invoice.void",
	"invoice.payment_failure",
	"fee.created",
	"credit_note.created",
	"credit_note.generated",
	"payment_request.created",
	"payment_request.payment_failure",
}

// ── helpers ────────────────────────────────────────────────────────────────────

func toEndpointBody(ep *models.WebhookEndpoint) endpointBody {
	algoStr := "hmac_sha_256"
	if ep.SignatureAlgo == models.WebhookSignatureAlgoJWT {
		algoStr = "jwt_es512"
	}
	return endpointBody{
		LagoID:             ep.ID,
		LagoOrganizationID: ep.OrganizationID,
		WebhookURL:         ep.WebhookURL,
		SignatureAlgo:      algoStr,
		CreatedAt:          ep.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

func organizationIDFromContext(c *gin.Context) (string, bool) {
	v, ok := c.Get(middleware.GinKeyOrganizationID)
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

func handleServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, wesvc.ErrWebhookEndpointNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "webhook_endpoint_not_found"})
	case errors.Is(err, wesvc.ErrWebhookURLConflict):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "value_already_exist", "field": "webhook_url"})
	case errors.Is(err, wesvc.ErrMaxEndpointsReached):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "max_webhook_endpoints_reached"})
	case wesvc.IsValidationError(err):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
	}
}

func toSignatureAlgo(v *int) models.WebhookSignatureAlgo {
	if v == nil {
		return models.WebhookSignatureAlgoHMACSHA256
	}
	if *v == 1 {
		return models.WebhookSignatureAlgoJWT
	}
	return models.WebhookSignatureAlgoHMACSHA256
}

// ── handlers ───────────────────────────────────────────────────────────────────

// Create handles POST /api/v1/webhook_endpoints.
func Create(svc wesvc.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		orgID, ok := organizationIDFromContext(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		var req createRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ep, err := svc.Create(orgID, wesvc.CreateParams{
			ID:            req.WebhookEndpoint.ID,
			WebhookURL:    req.WebhookEndpoint.WebhookURL,
			SignatureAlgo: toSignatureAlgo(req.WebhookEndpoint.SignatureAlgo),
		})
		if err != nil {
			handleServiceError(c, err)
			return
		}
		c.JSON(http.StatusCreated, endpointResponse{WebhookEndpoint: toEndpointBody(ep)})
	}
}

// Index handles GET /api/v1/webhook_endpoints.
func Index(svc wesvc.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		orgID, ok := organizationIDFromContext(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		limit, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))

		eps, total, err := svc.List(orgID, page, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
			return
		}

		bodies := make([]endpointBody, len(eps))
		for i := range eps {
			bodies[i] = toEndpointBody(&eps[i])
		}

		totalPages := int((total + int64(limit) - 1) / int64(limit))
		var nextPage, prevPage *int
		if page < totalPages {
			n := page + 1
			nextPage = &n
		}
		if page > 1 {
			p := page - 1
			prevPage = &p
		}

		c.JSON(http.StatusOK, listResponse{
			WebhookEndpoints: bodies,
			Meta: pageMeta{
				CurrentPage: page,
				NextPage:    nextPage,
				PrevPage:    prevPage,
				TotalPages:  totalPages,
				TotalCount:  total,
			},
		})
	}
}

// Show handles GET /api/v1/webhook_endpoints/:id.
func Show(svc wesvc.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		orgID, ok := organizationIDFromContext(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		ep, err := svc.GetByID(orgID, c.Param("id"))
		if err != nil {
			handleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, endpointResponse{WebhookEndpoint: toEndpointBody(ep)})
	}
}

// Update handles PUT /api/v1/webhook_endpoints/:id.
func Update(svc wesvc.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		orgID, ok := organizationIDFromContext(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		var req updateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		params := wesvc.UpdateParams{
			WebhookURL: req.WebhookEndpoint.WebhookURL,
		}
		if req.WebhookEndpoint.SignatureAlgo != nil {
			algo := toSignatureAlgo(req.WebhookEndpoint.SignatureAlgo)
			params.SignatureAlgo = &algo
		}

		ep, err := svc.Update(orgID, c.Param("id"), params)
		if err != nil {
			handleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, endpointResponse{WebhookEndpoint: toEndpointBody(ep)})
	}
}

// Destroy handles DELETE /api/v1/webhook_endpoints/:id.
func Destroy(svc wesvc.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		orgID, ok := organizationIDFromContext(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		ep, err := svc.Delete(orgID, c.Param("id"))
		if err != nil {
			handleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, endpointResponse{WebhookEndpoint: toEndpointBody(ep)})
	}
}

// EventTypes handles GET /api/v1/webhook_endpoints/event_types.
func EventTypes() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"event_types": SupportedWebhookEventTypes})
	}
}
