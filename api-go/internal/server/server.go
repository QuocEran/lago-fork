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
	bmhandlers "github.com/getlago/lago/api-go/internal/handlers/billable_metrics"
	customerhandlers "github.com/getlago/lago/api-go/internal/handlers/customers"
	eventhandlers "github.com/getlago/lago/api-go/internal/handlers/events"
	invoicehandlers "github.com/getlago/lago/api-go/internal/handlers/invoices"
	organizationhandlers "github.com/getlago/lago/api-go/internal/handlers/organizations"
	planhandlers "github.com/getlago/lago/api-go/internal/handlers/plans"
	subhandlers "github.com/getlago/lago/api-go/internal/handlers/subscriptions"
	kafkapkg "github.com/getlago/lago/api-go/internal/kafka"
	"github.com/getlago/lago/api-go/internal/middleware"
	bmservices "github.com/getlago/lago/api-go/internal/services/billable_metrics"
	customerservices "github.com/getlago/lago/api-go/internal/services/customers"
	eventservices "github.com/getlago/lago/api-go/internal/services/events"
	invoiceservices "github.com/getlago/lago/api-go/internal/services/invoices"
	organizationservices "github.com/getlago/lago/api-go/internal/services/organizations"
	planservices "github.com/getlago/lago/api-go/internal/services/plans"
	subservices "github.com/getlago/lago/api-go/internal/services/subscriptions"
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
	invoicesSvc := invoiceservices.NewService(db)
	billableMetricsSvc := bmservices.NewService(db)
	plansSvc := planservices.NewService(db)
	subscriptionsSvc := subservices.NewService(db)
	graphQLServer := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{
		Resolvers: &graphql.Resolver{
			BillableMetricSvc: billableMetricsSvc,
			InvoiceSvc:        invoicesSvc,
			PlanSvc:           plansSvc,
			SubscriptionSvc:   subscriptionsSvc,
		},
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
		v1.POST("/invoices", middleware.RequirePermission("invoice", "write"), invoicehandlers.Create(invoicesSvc))
		v1.GET("/invoices", middleware.RequirePermission("invoice", "read"), invoicehandlers.Index(invoicesSvc))
		v1.GET("/invoices/:id", middleware.RequirePermission("invoice", "read"), invoicehandlers.Show(invoicesSvc))
		v1.PUT("/invoices/:id/finalize", middleware.RequirePermission("invoice", "write"), invoicehandlers.Finalize(invoicesSvc))
		v1.PUT("/invoices/:id/void", middleware.RequirePermission("invoice", "write"), invoicehandlers.Void(invoicesSvc))

		v1.POST("/billable_metrics", middleware.RequirePermission("billable_metric", "write"), bmhandlers.Create(billableMetricsSvc))
		v1.GET("/billable_metrics", middleware.RequirePermission("billable_metric", "read"), bmhandlers.Index(billableMetricsSvc))
		v1.GET("/billable_metrics/:code", middleware.RequirePermission("billable_metric", "read"), bmhandlers.Show(billableMetricsSvc))
		v1.PUT("/billable_metrics/:code", middleware.RequirePermission("billable_metric", "write"), bmhandlers.Update(billableMetricsSvc))
		v1.DELETE("/billable_metrics/:code", middleware.RequirePermission("billable_metric", "write"), bmhandlers.Destroy(billableMetricsSvc))

		v1.POST("/plans", middleware.RequirePermission("plan", "write"), planhandlers.Create(plansSvc))
		v1.GET("/plans", middleware.RequirePermission("plan", "read"), planhandlers.Index(plansSvc))
		v1.GET("/plans/:code", middleware.RequirePermission("plan", "read"), planhandlers.Show(plansSvc))
		v1.PUT("/plans/:code", middleware.RequirePermission("plan", "write"), planhandlers.Update(plansSvc))
		v1.DELETE("/plans/:code", middleware.RequirePermission("plan", "write"), planhandlers.Destroy(plansSvc))

		v1.POST("/subscriptions", middleware.RequirePermission("subscription", "write"), subhandlers.Create(subscriptionsSvc))
		v1.GET("/subscriptions", middleware.RequirePermission("subscription", "read"), subhandlers.Index(subscriptionsSvc))
		v1.GET("/subscriptions/:external_id", middleware.RequirePermission("subscription", "read"), subhandlers.Show(subscriptionsSvc))
		v1.PUT("/subscriptions/:external_id", middleware.RequirePermission("subscription", "write"), subhandlers.Update(subscriptionsSvc))
		v1.DELETE("/subscriptions/:external_id", middleware.RequirePermission("subscription", "write"), subhandlers.Terminate(subscriptionsSvc))
		v1.GET("/customers/:external_id/current_usage", middleware.RequirePermission("customer", "read"), subhandlers.CurrentUsage())

		// Phase 4+ routes registered here as each phase is implemented.
	}

	return r
}
