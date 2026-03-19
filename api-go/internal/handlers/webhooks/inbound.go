package webhooks

import (
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	webhooksvc "github.com/getlago/lago/api-go/internal/services/webhooks"
)

// StripeWebhook handles POST /webhooks/stripe/:organization_id.
// It delegates signature verification and storage to the inbound service.
func StripeWebhook(svc webhooksvc.InboundService) gin.HandlerFunc {
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

		stripeSig := c.GetHeader("Stripe-Signature")
		code := c.Query("code")

		if err := svc.HandleStripeWebhook(c.Request.Context(), orgID, stripeSig, rawBody, code); err != nil {
			if errors.Is(err, webhooksvc.ErrSignatureInvalid) {
				c.Status(http.StatusUnauthorized)
				return
			}
			c.Status(http.StatusInternalServerError)
			return
		}

		c.Status(http.StatusOK)
	}
}

// GocardlessWebhook handles POST /webhooks/gocardless/:organization_id (stub).
func GocardlessWebhook(svc webhooksvc.InboundService) gin.HandlerFunc {
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

		if err := svc.HandleGocardlessWebhook(c.Request.Context(), orgID, rawBody); err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}

		c.Status(http.StatusOK)
	}
}
