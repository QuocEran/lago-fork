package graphql

import (
	bmsvc "github.com/getlago/lago/api-go/internal/services/billable_metrics"
	plansvc "github.com/getlago/lago/api-go/internal/services/plans"
)

type Resolver struct {
	BillableMetricSvc bmsvc.Service
	PlanSvc           plansvc.Service
}
