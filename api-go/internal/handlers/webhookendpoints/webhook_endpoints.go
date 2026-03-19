package webhookendpoints

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/getlago/lago/api-go/internal/handlers/shared"
	"github.com/getlago/lago/api-go/internal/models"
	wesvc "github.com/getlago/lago/api-go/internal/services/webhookendpoints"
)

var webhookEndpointErrorClassifier = shared.ServiceErrorClassifier{
	NotFoundErrors:  []error{wesvc.ErrWebhookEndpointNotFound},
	ConflictErrors:  []error{wesvc.ErrWebhookURLConflict},
	IsValidationErr: wesvc.IsValidationError,
	NotFoundCode:    "webhook_endpoint_not_found",
	ConflictCode:    "value_already_exist",
	ConflictDetails: func(err error) map[string]any {
		if errors.Is(err, wesvc.ErrWebhookURLConflict) {
			return map[string]any{"field": "webhook_url"}
		}
		return nil
	},
	CustomErrors: []shared.CustomErrorRule{
		{
			Match:  func(err error) bool { return errors.Is(err, wesvc.ErrMaxEndpointsReached) },
			Status: http.StatusUnprocessableEntity,
			Code:   "max_webhook_endpoints_reached",
			Details: func(err error) any { return gin.H{} },
		},
	},
}

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
	WebhookEndpoints []endpointBody      `json:"webhook_endpoints"`
	Meta             shared.PaginationMeta `json:"meta"`
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
		orgID, ok := shared.OrganizationIDFromContext(c)
		if !ok {
			return
		}
		var req createRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			shared.RespondError(c, http.StatusBadRequest, "validation_error", gin.H{"message": err.Error()})
			return
		}
		ep, err := svc.Create(c.Request.Context(), orgID, wesvc.CreateParams{
			ID:            req.WebhookEndpoint.ID,
			WebhookURL:    req.WebhookEndpoint.WebhookURL,
			SignatureAlgo: toSignatureAlgo(req.WebhookEndpoint.SignatureAlgo),
		})
		if err != nil {
			shared.HandleServiceError(c, err, webhookEndpointErrorClassifier)
			return
		}
		c.JSON(http.StatusCreated, endpointResponse{WebhookEndpoint: toEndpointBody(ep)})
	}
}

// Index handles GET /api/v1/webhook_endpoints.
func Index(svc wesvc.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		orgID, ok := shared.OrganizationIDFromContext(c)
		if !ok {
			return
		}
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		limit, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))

		eps, total, err := svc.List(c.Request.Context(), orgID, page, limit)
		if err != nil {
			shared.HandleServiceError(c, err, webhookEndpointErrorClassifier)
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
			Meta: shared.PaginationMeta{
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
		orgID, ok := shared.OrganizationIDFromContext(c)
		if !ok {
			return
		}
		ep, err := svc.GetByID(c.Request.Context(), orgID, c.Param("id"))
		if err != nil {
			shared.HandleServiceError(c, err, webhookEndpointErrorClassifier)
			return
		}
		c.JSON(http.StatusOK, endpointResponse{WebhookEndpoint: toEndpointBody(ep)})
	}
}

// Update handles PUT /api/v1/webhook_endpoints/:id.
func Update(svc wesvc.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		orgID, ok := shared.OrganizationIDFromContext(c)
		if !ok {
			return
		}
		var req updateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			shared.RespondError(c, http.StatusBadRequest, "validation_error", gin.H{"message": err.Error()})
			return
		}

		params := wesvc.UpdateParams{
			WebhookURL: req.WebhookEndpoint.WebhookURL,
		}
		if req.WebhookEndpoint.SignatureAlgo != nil {
			algo := toSignatureAlgo(req.WebhookEndpoint.SignatureAlgo)
			params.SignatureAlgo = &algo
		}

		ep, err := svc.Update(c.Request.Context(), orgID, c.Param("id"), params)
		if err != nil {
			shared.HandleServiceError(c, err, webhookEndpointErrorClassifier)
			return
		}
		c.JSON(http.StatusOK, endpointResponse{WebhookEndpoint: toEndpointBody(ep)})
	}
}

// Destroy handles DELETE /api/v1/webhook_endpoints/:id.
func Destroy(svc wesvc.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		orgID, ok := shared.OrganizationIDFromContext(c)
		if !ok {
			return
		}
		ep, err := svc.Delete(c.Request.Context(), orgID, c.Param("id"))
		if err != nil {
			shared.HandleServiceError(c, err, webhookEndpointErrorClassifier)
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
