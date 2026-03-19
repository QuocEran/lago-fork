package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/getlago/lago/api-go/internal/graphql/graphcontext"
	"github.com/getlago/lago/api-go/internal/models"
)

func GraphQLAPIKeyContext(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractBearerToken(c)
		if token == "" || db == nil {
			c.Next()
			return
		}

		var key models.APIKey
		result := db.
			Preload("Organization").
			Where("value = ? AND (expires_at IS NULL OR expires_at > ?)", token, time.Now()).
			First(&key)
		if result.Error != nil {
			c.Next()
			return
		}

		c.Set(GinKeyOrganizationID, key.OrganizationID)
		c.Set(GinKeyAPIKeyID, key.ID)
		c.Set(GinKeyAPIKeyPermissions, key.Permissions)
		c.Set(GinKeyOrganizationPremiumIntegrations, key.Organization.PremiumIntegrations)

		requestContext := c.Request.Context()
		requestContext = graphcontext.WithOrganizationID(requestContext, key.OrganizationID)
		requestContext = graphcontext.WithAPIKeyID(requestContext, key.ID)
		c.Request = c.Request.WithContext(requestContext)
		c.Next()
	}
}
