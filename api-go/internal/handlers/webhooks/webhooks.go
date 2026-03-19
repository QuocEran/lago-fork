package webhooks

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/getlago/lago/api-go/internal/handlers/shared"
	"github.com/getlago/lago/api-go/internal/models"
	webhooksvc "github.com/getlago/lago/api-go/internal/services/webhooks"
)

// ── response types ─────────────────────────────────────────────────────────────

type webhookBody struct {
	LagoID             string  `json:"lago_id"`
	LagoOrganizationID string  `json:"lago_organization_id"`
	WebhookEndpointID  *string `json:"lago_webhook_endpoint_id,omitempty"`
	WebhookType        *string `json:"webhook_type,omitempty"`
	ObjectID           *string `json:"object_lago_id,omitempty"`
	ObjectType         *string `json:"object_type,omitempty"`
	Status             string  `json:"status"`
	Retries            int     `json:"retries"`
	HTTPStatus         *int    `json:"http_status,omitempty"`
	Endpoint           *string `json:"endpoint,omitempty"`
}

type listResponse struct {
	Webhooks []webhookBody         `json:"webhooks"`
	Meta     shared.PaginationMeta `json:"meta"`
}

// ── helpers ────────────────────────────────────────────────────────────────────

func toWebhookBody(w *models.Webhook) webhookBody {
	statusStr := map[models.WebhookStatus]string{
		models.WebhookStatusPending:   "pending",
		models.WebhookStatusSucceeded: "succeeded",
		models.WebhookStatusFailed:    "failed",
	}[w.Status]

	return webhookBody{
		LagoID:             w.ID,
		LagoOrganizationID: w.OrganizationID,
		WebhookEndpointID:  w.WebhookEndpointID,
		WebhookType:        w.WebhookType,
		ObjectID:           w.ObjectID,
		ObjectType:         w.ObjectType,
		Status:             statusStr,
		Retries:            w.Retries,
		HTTPStatus:         w.HTTPStatus,
		Endpoint:           w.Endpoint,
	}
}

// ── handlers ───────────────────────────────────────────────────────────────────

// Index handles GET /api/v1/webhooks — list outbound webhooks for the org.
func Index(svc webhooksvc.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		orgID, ok := shared.OrganizationIDFromContext(c)
		if !ok {
			return
		}

		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		limit, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
		if limit <= 0 {
			limit = 20
		}
		if page <= 0 {
			page = 1
		}

		ws, total, err := svc.List(c.Request.Context(), orgID, page, limit)
		if err != nil {
			shared.RespondError(c, http.StatusInternalServerError, "internal_error", gin.H{})
			return
		}

		bodies := make([]webhookBody, len(ws))
		for i := range ws {
			bodies[i] = toWebhookBody(&ws[i])
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
			Webhooks: bodies,
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

// Show handles GET /api/v1/webhooks/:id.
func Show(svc webhooksvc.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		orgID, ok := shared.OrganizationIDFromContext(c)
		if !ok {
			return
		}
		wh, err := svc.GetByID(c.Request.Context(), orgID, c.Param("id"))
		if err != nil {
			if errors.Is(err, webhooksvc.ErrWebhookNotFound) {
				shared.RespondError(c, http.StatusNotFound, "webhook_not_found", gin.H{})
				return
			}
			shared.RespondError(c, http.StatusInternalServerError, "internal_error", gin.H{})
			return
		}
		c.JSON(http.StatusOK, gin.H{"webhook": toWebhookBody(wh)})
	}
}
