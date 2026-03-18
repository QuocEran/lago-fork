package handlers

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
)

type HealthResponse struct {
	Version string `json:"version"`
}

type ReadyResponse struct {
	Status string `json:"status"`
}

func Health(version string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, HealthResponse{Version: version})
	}
}

func Ready(sqlDB *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := sqlDB.Ping(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "not_ready",
				"error":  err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, ReadyResponse{Status: "ok"})
	}
}
