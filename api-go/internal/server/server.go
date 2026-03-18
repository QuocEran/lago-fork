package server

import (
	"database/sql"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/getlago/lago/api-go/internal/handlers"
	"github.com/getlago/lago/api-go/internal/middleware"
)

func New(db *gorm.DB, sqlDB *sql.DB, version string) *gin.Engine {
	r := gin.New()

	r.Use(middleware.Recovery())
	r.Use(middleware.Logging())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Lago-Organization-Id"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
	}))

	r.GET("/health", handlers.Health(version))
	r.GET("/ready", handlers.Ready(sqlDB))

	v1 := r.Group("/api/v1")
	v1.Use(middleware.APIKeyAuth(db))
	{
		// Phase 4+ routes registered here as each phase is implemented.
	}

	return r
}
