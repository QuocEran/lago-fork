package server

import (
	"database/sql"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/getlago/lago/api-go/internal/graphql"
	"github.com/getlago/lago/api-go/internal/graphql/generated"
	"github.com/getlago/lago/api-go/internal/handlers"
	authhandlers "github.com/getlago/lago/api-go/internal/handlers/auth"
	customerhandlers "github.com/getlago/lago/api-go/internal/handlers/customers"
	eventhandlers "github.com/getlago/lago/api-go/internal/handlers/events"
	organizationhandlers "github.com/getlago/lago/api-go/internal/handlers/organizations"
	kafkapkg "github.com/getlago/lago/api-go/internal/kafka"
	"github.com/getlago/lago/api-go/internal/middleware"
	customerservices "github.com/getlago/lago/api-go/internal/services/customers"
	eventservices "github.com/getlago/lago/api-go/internal/services/events"
	organizationservices "github.com/getlago/lago/api-go/internal/services/organizations"
	"github.com/getlago/lago/api-go/internal/services/users"
)

func New(db *gorm.DB, sqlDB *sql.DB, version string, jwtSecret string, eventPublisher kafkapkg.EventPublisher) *gin.Engine {
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

	authSvc := users.NewAuthService(db, jwtSecret)
	customersSvc := customerservices.NewService(db)
	eventsSvc := eventservices.NewService(db, eventPublisher)
	organizationSvc := organizationservices.NewService(db)
	graphQLServer := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{
		Resolvers: &graphql.Resolver{},
	}))

	r.POST("/users/login", authhandlers.Login(authSvc))
	r.POST("/users/register", authhandlers.Register(authSvc))
	graphQLServer.SetErrorPresenter(graphql.ErrorPresenter)
	r.POST("/graphql",
		middleware.GraphQLAPIKeyContext(db),
		middleware.GraphQLDataLoaders(db),
		gin.WrapH(graphQLServer),
	)

	v1 := r.Group("/api/v1")
	v1.Use(middleware.APIKeyAuth(db))
	{
		v1.GET("/organizations", middleware.RequirePermission("organization", ""), organizationhandlers.Show(organizationSvc))
		v1.PUT("/organizations", middleware.RequirePermission("organization", ""), organizationhandlers.Update(organizationSvc))
		v1.POST("/customers", middleware.RequirePermission("customer", "write"), customerhandlers.Create(customersSvc))
		v1.GET("/customers", middleware.RequirePermission("customer", "read"), customerhandlers.Index(customersSvc))
		v1.GET("/customers/:external_id", middleware.RequirePermission("customer", "read"), customerhandlers.Show(customersSvc))
		v1.DELETE("/customers/:external_id", middleware.RequirePermission("customer", "write"), customerhandlers.Delete(customersSvc))
		v1.GET("/customers/:external_id/portal_url", middleware.RequirePermission("customer", "read"), customerhandlers.PortalURL(customersSvc))
		v1.POST("/events", middleware.RequirePermission("event", ""), eventhandlers.Create(eventsSvc))
		v1.POST("/events/batch", middleware.RequirePermission("event", ""), eventhandlers.CreateBatch(eventsSvc))
		v1.GET("/events", middleware.RequirePermission("event", ""), eventhandlers.List(eventsSvc))
		v1.GET("/events/estimate_fees", middleware.RequirePermission("event", ""), eventhandlers.EstimateFees(eventsSvc))

		// Phase 4+ routes registered here as each phase is implemented.
	}

	return r
}
