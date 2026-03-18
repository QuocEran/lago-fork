package middleware

import (
	"net/http"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/getlago/lago/api-go/internal/models"
)

const apiPermissionsIntegration = "api_permissions"

var apiKeyResources = map[string]struct{}{
	"activity_log":           {},
	"add_on":                 {},
	"analytic":               {},
	"api_log":                {},
	"billable_metric":        {},
	"coupon":                 {},
	"applied_coupon":         {},
	"credit_note":            {},
	"customer_usage":         {},
	"customer":               {},
	"event":                  {},
	"fee":                    {},
	"invoice":                {},
	"organization":           {},
	"payment":                {},
	"payment_receipt":        {},
	"payment_request":        {},
	"payment_method":         {},
	"plan":                   {},
	"subscription":           {},
	"lifetime_usage":         {},
	"tax":                    {},
	"wallet":                 {},
	"wallet_transaction":     {},
	"webhook_endpoint":       {},
	"webhook_jwt_public_key": {},
	"invoice_custom_section": {},
	"billing_entity":         {},
	"alert":                  {},
	"feature":                {},
	"security_log":           {},
}

// RequirePermission checks API key resource/mode permissions.
// If mode is empty, it is derived from HTTP method: GET->read, others->write.
func RequirePermission(resource string, mode string) gin.HandlerFunc {
	return func(c *gin.Context) {
		actualMode := mode
		if actualMode == "" {
			actualMode = modeFromMethod(c.Request.Method)
		}

		if !isKnownResource(resource) {
			forbid(c, resource, actualMode)
			return
		}

		if !isAPIPermissionsEnabled(c) {
			c.Next()
			return
		}

		permissions, hasPermissions := getAPIKeyPermissions(c)
		if !hasPermissions {
			forbid(c, resource, actualMode)
			return
		}

		if permissions == nil {
			c.Next()
			return
		}

		if !isModeAllowed(permissions, resource, actualMode) {
			forbid(c, resource, actualMode)
			return
		}

		c.Next()
	}
}

func modeFromMethod(method string) string {
	if strings.EqualFold(method, http.MethodGet) {
		return "read"
	}
	return "write"
}

func isKnownResource(resource string) bool {
	_, exists := apiKeyResources[resource]
	return exists
}

func isAPIPermissionsEnabled(c *gin.Context) bool {
	value, exists := c.Get(GinKeyOrganizationPremiumIntegrations)
	if !exists {
		return false
	}

	switch integrations := value.(type) {
	case models.StringArray:
		return slices.Contains(integrations, apiPermissionsIntegration)
	case []string:
		return slices.Contains(integrations, apiPermissionsIntegration)
	default:
		return false
	}
}

func getAPIKeyPermissions(c *gin.Context) (models.JSONBMap, bool) {
	value, exists := c.Get(GinKeyAPIKeyPermissions)
	if !exists {
		return nil, false
	}
	if value == nil {
		return nil, true
	}

	switch permissions := value.(type) {
	case models.JSONBMap:
		return permissions, true
	case map[string]any:
		return models.JSONBMap(permissions), true
	default:
		return nil, false
	}
}

func isModeAllowed(permissions models.JSONBMap, resource string, mode string) bool {
	rawModes, exists := permissions[resource]
	if !exists {
		return false
	}

	switch modes := rawModes.(type) {
	case []string:
		return slices.Contains(modes, mode)
	case []any:
		for _, rawMode := range modes {
			modeValue, isString := rawMode.(string)
			if isString && modeValue == mode {
				return true
			}
		}
		return false
	case string:
		return modes == mode
	default:
		return false
	}
}

func forbid(c *gin.Context, resource string, mode string) {
	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
		"status": "forbidden",
		"error":  mode + "_action_not_allowed_for_" + resource,
	})
}
