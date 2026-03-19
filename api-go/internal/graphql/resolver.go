package graphql

import (
	bmsvc "github.com/getlago/lago/api-go/internal/services/billable_metrics"
	invsvc "github.com/getlago/lago/api-go/internal/services/invoices"
	plansvc "github.com/getlago/lago/api-go/internal/services/plans"
	subsvc "github.com/getlago/lago/api-go/internal/services/subscriptions"
	wesvc "github.com/getlago/lago/api-go/internal/services/webhook_endpoints"
)

type Resolver struct {
	BillableMetricSvc  bmsvc.Service
	InvoiceSvc         invsvc.Service
	PlanSvc            plansvc.Service
	SubscriptionSvc    subsvc.Service
	WebhookEndpointSvc wesvc.Service
}
