package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type apiKeyRecord struct {
	ID             string     `gorm:"column:id"`
	OrganizationID string     `gorm:"column:organization_id"`
	ExpiresAt      *time.Time `gorm:"column:expires_at"`
}

func (apiKeyRecord) TableName() string { return "api_keys" }

func APIKeyAuth(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractBearerToken(c)
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"status": "unauthorized",
				"error":  "missing_api_key",
			})
			return
		}

		var key apiKeyRecord
		result := db.
			Where("value = ? AND (expires_at IS NULL OR expires_at > ?)", token, time.Now()).
			First(&key)

		if result.Error != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"status": "unauthorized",
				"error":  "invalid_api_key",
			})
			return
		}

		c.Set(GinKeyOrganizationID, key.OrganizationID)
		c.Set(GinKeyAPIKeyID, key.ID)
		c.Next()
	}
}

func extractBearerToken(c *gin.Context) string {
	header := c.GetHeader("Authorization")
	if header == "" {
		return ""
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}
