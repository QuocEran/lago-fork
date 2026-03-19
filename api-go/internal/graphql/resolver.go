package graphql

import (
	bmsvc "github.com/getlago/lago/api-go/internal/services/billable_metrics"
	plansvc "github.com/getlago/lago/api-go/internal/services/plans"
	subsvc "github.com/getlago/lago/api-go/internal/services/subscriptions"
)

type Resolver struct {
	BillableMetricSvc bmsvc.Service
	PlanSvc           plansvc.Service
	SubscriptionSvc   subsvc.Service
}
