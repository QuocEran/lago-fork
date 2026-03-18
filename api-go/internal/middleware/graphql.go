package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/getlago/lago/api-go/internal/graphql/dataloaders"
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

func GraphQLDataLoaders(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		loaders := dataloaders.NewLoaders(db)
		requestContext := dataloaders.WithLoaders(c.Request.Context(), loaders)
		c.Request = c.Request.WithContext(requestContext)
		c.Next()
	}
}

func GraphQLRequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, exists := c.Get(GinKeyOrganizationID); exists {
			c.Next()
			return
		}

		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"status": "unauthorized",
			"error":  "missing_api_key",
		})
	}
}
