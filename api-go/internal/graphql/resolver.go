package graphql

import (
	bmsvc "github.com/getlago/lago/api-go/internal/services/billable_metrics"
)

type Resolver struct {
	BillableMetricSvc bmsvc.Service
}
