package billablemetrics_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/getlago/lago/api-go/internal/models"
	bmsvc "github.com/getlago/lago/api-go/internal/services/billablemetrics"
)

// --- validation tests (no DB needed) ---

func TestValidationError_Error(t *testing.T) {
	err := &bmsvc.ValidationError{Message: "some message"}
	assert.Equal(t, "some message", err.Error())
	assert.True(t, bmsvc.IsValidationError(err))
}

func TestIsValidationError_ReturnsFalseForOtherErrors(t *testing.T) {
	assert.False(t, bmsvc.IsValidationError(bmsvc.ErrBillableMetricNotFound))
}

func TestAggregationTypeRoundTrip(t *testing.T) {
	cases := []struct {
		str     string
		aggType models.AggregationType
	}{
		{"count_agg", models.AggregationTypeCount},
		{"sum_agg", models.AggregationTypeSum},
		{"max_agg", models.AggregationTypeMax},
		{"unique_count_agg", models.AggregationTypeUniqueCount},
		{"weighted_sum_agg", models.AggregationTypeWeightedSum},
		{"latest_agg", models.AggregationTypeLatest},
		{"custom_agg", models.AggregationTypeCustom},
	}

	for _, tc := range cases {
		t.Run(tc.str, func(t *testing.T) {
			got, ok := models.AggregationTypeFromString(tc.str)
			require.True(t, ok)
			assert.Equal(t, tc.aggType, got)
			assert.Equal(t, tc.str, models.AggregationTypeToString(tc.aggType))
		})
	}
}

func TestAggregationTypeFromString_InvalidReturnsNotOK(t *testing.T) {
	_, ok := models.AggregationTypeFromString("invalid_agg")
	assert.False(t, ok)
}

// TestCreateInput_ValidationRules validates that the service rejects invalid inputs
// before reaching the DB layer. Only invalid inputs that cause ValidationErrors are tested here.
// Valid inputs are tested through handler tests with a mock service.
func TestCreateInput_ValidationRules(t *testing.T) {
	tests := []struct {
		name   string
		input  bmsvc.CreateInput
		errMsg string
	}{
		{
			name:   "sum_agg without field_name fails",
			input:  bmsvc.CreateInput{Name: "m", Code: "c", AggregationType: "sum_agg"},
			errMsg: "field_name is required",
		},
		{
			name:   "weighted_sum_agg without weighted_interval fails",
			input:  bmsvc.CreateInput{Name: "m", Code: "c", AggregationType: "weighted_sum_agg", FieldName: strPtr("x")},
			errMsg: "weighted_interval is required",
		},
		{
			name:   "recurring count_agg fails",
			input:  bmsvc.CreateInput{Name: "m", Code: "c", AggregationType: "count_agg", Recurring: boolPtr(true)},
			errMsg: "recurring is not compatible",
		},
		{
			name:   "custom_agg without custom_aggregator fails",
			input:  bmsvc.CreateInput{Name: "m", Code: "c", AggregationType: "custom_agg"},
			errMsg: "custom_aggregator is required",
		},
		{
			name:   "missing name fails",
			input:  bmsvc.CreateInput{Code: "c", AggregationType: "count_agg"},
			errMsg: "name is required",
		},
		{
			name:   "missing code fails",
			input:  bmsvc.CreateInput{Name: "n", AggregationType: "count_agg"},
			errMsg: "code is required",
		},
		{
			name:   "invalid aggregation_type fails",
			input:  bmsvc.CreateInput{Name: "n", Code: "c", AggregationType: "bad_agg"},
			errMsg: "invalid aggregation_type",
		},
	}

	// Nil DB is safe here because all cases fail validation before any DB access.
	svc := bmsvc.NewService(nil)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.Create(context.Background(), "org-1", tc.input)
			require.Error(t, err)
			assert.True(t, bmsvc.IsValidationError(err), "expected ValidationError, got: %v", err)
			assert.Contains(t, err.Error(), tc.errMsg)
		})
	}
}

func strPtr(s string) *string { return &s }
func boolPtr(b bool) *bool    { return &b }
