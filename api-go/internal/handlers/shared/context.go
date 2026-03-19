package shared

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/getlago/lago/api-go/internal/middleware"
)

// OrganizationIDFromContext reads the organization ID from the Gin context (set by auth middleware).
// If missing or invalid, it writes the JSON error response and returns "", false.
func OrganizationIDFromContext(c *gin.Context) (string, bool) {
	value, exists := c.Get(middleware.GinKeyOrganizationID)
	if !exists {
		RespondError(c, http.StatusUnauthorized, "missing_organization_context", nil)
		return "", false
	}
	orgID, ok := value.(string)
	if !ok || orgID == "" {
		RespondError(c, http.StatusUnauthorized, "invalid_organization_context", nil)
		return "", false
	}
	return orgID, true
}
