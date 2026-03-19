package shared

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrorResponse is the standard error envelope for API responses.
type ErrorResponse struct {
	Status       string `json:"status"`
	ErrorCode    string `json:"error_code"`
	ErrorDetails any    `json:"error_details"`
}

// RespondJSON writes a JSON response with the given status, root key, and data.
func RespondJSON(c *gin.Context, status int, key string, data any) {
	c.JSON(status, gin.H{key: data})
}

// RespondList writes a paginated list JSON response with the given key, data slice, and meta.
func RespondList(c *gin.Context, key string, data any, meta PaginationMeta) {
	c.JSON(http.StatusOK, gin.H{key: data, "meta": meta})
}

// RespondError writes the standard error envelope and sets the HTTP status.
// For 401, body status is "unauthorized"; otherwise "error". errorDetails may be nil (sent as empty object).
func RespondError(c *gin.Context, status int, errorCode string, errorDetails any) {
	statusStr := "error"
	if status == http.StatusUnauthorized {
		statusStr = "unauthorized"
	}
	if errorDetails == nil {
		errorDetails = gin.H{}
	}
	c.JSON(status, ErrorResponse{
		Status:       statusStr,
		ErrorCode:    errorCode,
		ErrorDetails: errorDetails,
	})
}
