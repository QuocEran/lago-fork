package testutil

import (
	"github.com/gin-gonic/gin"

	"github.com/getlago/lago/api-go/internal/middleware"
)

// NewTestRouter returns a Gin engine with common middleware for handler tests.
// Use this when you need a minimal router that sets organization context.
func NewTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.Recovery())
	return r
}

// SetOrganizationID sets the organization ID in the request context.
// Call this in tests before invoking handlers that require auth.
func SetOrganizationID(c *gin.Context, orgID string) {
	c.Set(middleware.GinKeyOrganizationID, orgID)
}
