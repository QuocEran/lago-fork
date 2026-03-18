package middleware

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/getsentry/sentry-go"
)

func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("panic recovered",
					slog.Any("error", r),
					slog.String("path", c.Request.URL.Path),
				)
				sentry.CurrentHub().Recover(r)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"status": "error",
					"code":   "internal_error",
				})
			}
		}()
		c.Next()
	}
}
