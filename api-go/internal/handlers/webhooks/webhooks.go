package webhooks

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/getlago/lago/api-go/internal/middleware"
	"github.com/getlago/lago/api-go/internal/models"
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
	Webhooks []webhookBody `json:"webhooks"`
	Meta     pageMeta      `json:"meta"`
}

type pageMeta struct {
	CurrentPage int   `json:"current_page"`
	NextPage    *int  `json:"next_page"`
	PrevPage    *int  `json:"prev_page"`
	TotalPages  int   `json:"total_pages"`
	TotalCount  int64 `json:"total_count"`
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

func organizationIDFromContext(c *gin.Context) (string, bool) {
	v, ok := c.Get(middleware.GinKeyOrganizationID)
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

// ── handlers ───────────────────────────────────────────────────────────────────

// Index handles GET /api/v1/webhooks — list outbound webhooks for the org.
func Index(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		orgID, ok := organizationIDFromContext(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
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
		offset := (page - 1) * limit

		var total int64
		if err := db.Model(&models.Webhook{}).Where("organization_id = ?", orgID).Count(&total).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
			return
		}

		var ws []models.Webhook
		if err := db.Where("organization_id = ?", orgID).
			Order("created_at DESC").
			Limit(limit).Offset(offset).
			Find(&ws).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
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

// Show handles GET /api/v1/webhooks/:id.
func Show(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		orgID, ok := organizationIDFromContext(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		var wh models.Webhook
		err := db.Where("id = ? AND organization_id = ?", c.Param("id"), orgID).First(&wh).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "webhook_not_found"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"webhook": toWebhookBody(&wh)})
	}
}
