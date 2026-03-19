package webhooks

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/getlago/lago/api-go/internal/jobs/handlers"
	"github.com/getlago/lago/api-go/internal/models"
)

// StripeWebhook handles POST /webhooks/stripe/:organization_id.
// It validates the Stripe signature and stores an InboundWebhook record for later processing.
func StripeWebhook(db *gorm.DB, webhookSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		orgID := c.Param("organization_id")
		if orgID == "" {
			c.Status(http.StatusBadRequest)
			return
		}

		rawBody, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.Status(http.StatusBadRequest)
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewReader(rawBody))

		stripeSig := c.GetHeader("Stripe-Signature")
		if webhookSecret != "" && stripeSig != "" {
			if !handlers.VerifySignature(webhookSecret, bytes.NewReader(rawBody), stripeSig) {
				c.Status(http.StatusUnauthorized)
				return
			}
		}

		var payloadMap models.JSONBMap
		if err := json.Unmarshal(rawBody, &payloadMap); err != nil {
			payloadMap = models.JSONBMap{}
		}

		eventType, _ := payloadMap["type"].(string)
		code := c.Query("code")
		var codePtr *string
		if code != "" {
			codePtr = &code
		}
		var sigPtr *string
		if stripeSig != "" {
			sigPtr = &stripeSig
		}

		inbound := &models.InboundWebhook{
			OrganizationID: orgID,
			Source:         "stripe",
			EventType:      eventType,
			Payload:        payloadMap,
			Status:         models.InboundWebhookStatusPending,
			Code:           codePtr,
			Signature:      sigPtr,
		}
		if err := db.Create(inbound).Error; err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}

		c.Status(http.StatusOK)
	}
}

// GocardlessWebhook handles POST /webhooks/gocardless/:organization_id (stub).
func GocardlessWebhook(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		orgID := c.Param("organization_id")
		if orgID == "" {
			c.Status(http.StatusBadRequest)
			return
		}

		rawBody, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.Status(http.StatusBadRequest)
			return
		}

		var payloadMap models.JSONBMap
		if err := json.Unmarshal(rawBody, &payloadMap); err != nil {
			payloadMap = models.JSONBMap{}
		}

		inbound := &models.InboundWebhook{
			OrganizationID: orgID,
			Source:         "gocardless",
			EventType:      "",
			Payload:        payloadMap,
			Status:         models.InboundWebhookStatusPending,
		}
		if err := db.Create(inbound).Error; err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}

		c.Status(http.StatusOK)
	}
}
