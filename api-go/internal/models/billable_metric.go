package models

// AggregationType mirrors the Rails integer enum for billable_metrics.aggregation_type.
type AggregationType int

const (
	AggregationTypeCount       AggregationType = 0
	AggregationTypeSum         AggregationType = 1
	AggregationTypeMax         AggregationType = 2
	AggregationTypeUniqueCount AggregationType = 3
	AggregationTypeWeightedSum AggregationType = 5
	AggregationTypeLatest      AggregationType = 6
	AggregationTypeCustom      AggregationType = 7
)

// AggregationTypeFromString converts a string representation (matching GraphQL enum values) to AggregationType.
func AggregationTypeFromString(s string) (AggregationType, bool) {
	m := map[string]AggregationType{
		"count_agg":        AggregationTypeCount,
		"sum_agg":          AggregationTypeSum,
		"max_agg":          AggregationTypeMax,
		"unique_count_agg": AggregationTypeUniqueCount,
		"weighted_sum_agg": AggregationTypeWeightedSum,
		"latest_agg":       AggregationTypeLatest,
		"custom_agg":       AggregationTypeCustom,
	}
	v, ok := m[s]
	return v, ok
}

// AggregationTypeToString returns the string representation of an AggregationType.
func AggregationTypeToString(a AggregationType) string {
	m := map[AggregationType]string{
		AggregationTypeCount:       "count_agg",
		AggregationTypeSum:         "sum_agg",
		AggregationTypeMax:         "max_agg",
		AggregationTypeUniqueCount: "unique_count_agg",
		AggregationTypeWeightedSum: "weighted_sum_agg",
		AggregationTypeLatest:      "latest_agg",
		AggregationTypeCustom:      "custom_agg",
	}
	if s, ok := m[a]; ok {
		return s
	}
	return "count_agg"
}

// BillableMetric maps to the billable_metrics table.
type BillableMetric struct {
	SoftDeleteModel
	OrganizationID    string          `gorm:"column:organization_id;not null;index"`
	Name              string          `gorm:"column:name;not null"`
	Code              string          `gorm:"column:code;not null;index"`
	Description       *string         `gorm:"column:description"`
	AggregationType   AggregationType `gorm:"column:aggregation_type;not null;default:0"`
	FieldName         *string         `gorm:"column:field_name"`
	Recurring         bool            `gorm:"column:recurring;not null;default:false"`
	Expression        *string         `gorm:"column:expression"`
	CustomAggregator  *string         `gorm:"column:custom_aggregator"`
	WeightedInterval  *string         `gorm:"column:weighted_interval"`
	RoundingFunction  *string         `gorm:"column:rounding_function"`
	RoundingPrecision *int            `gorm:"column:rounding_precision"`
	Filters           []BillableMetricFilter `gorm:"foreignKey:BillableMetricID"`
}

func (BillableMetric) TableName() string { return "billable_metrics" }

// BillableMetricFilter maps to the billable_metric_filters table.
type BillableMetricFilter struct {
	SoftDeleteModel
	BillableMetricID string      `gorm:"column:billable_metric_id;not null;index"`
	OrganizationID   string      `gorm:"column:organization_id;not null;index"`
	Key              string      `gorm:"column:key;not null"`
	Values           StringArray `gorm:"column:values;type:text[];not null;default:'{}'"`
}

func (BillableMetricFilter) TableName() string { return "billable_metric_filters" }
