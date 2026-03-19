package graphql

import (
	bmsvc "github.com/getlago/lago/api-go/internal/services/billablemetrics"
	customersvc "github.com/getlago/lago/api-go/internal/services/customers"
	eventsvc "github.com/getlago/lago/api-go/internal/services/events"
	invsvc "github.com/getlago/lago/api-go/internal/services/invoices"
	organizationsvc "github.com/getlago/lago/api-go/internal/services/organizations"
	plansvc "github.com/getlago/lago/api-go/internal/services/plans"
	subsvc "github.com/getlago/lago/api-go/internal/services/subscriptions"
	wesvc "github.com/getlago/lago/api-go/internal/services/webhookendpoints"
)

type Resolver struct {
	BillableMetricSvc  bmsvc.Service
	CustomerSvc        customersvc.Service
	EventSvc           eventsvc.Service
	InvoiceSvc         invsvc.Service
	OrganizationSvc    organizationsvc.Service
	PlanSvc            plansvc.Service
	SubscriptionSvc    subsvc.Service
	WebhookEndpointSvc wesvc.Service
}
