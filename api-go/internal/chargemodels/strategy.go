// Package chargemodels implements pure-Go charge model calculation strategies
// that match Lago Rails output for standard/graduated/package/percentage/
// volume/graduated_percentage/custom/dynamic models.
package chargemodels

import (
	"fmt"

	"github.com/shopspring/decimal"
)

// Result holds the output of a charge model computation.
type Result struct {
	// Amount is the total charge amount.
	Amount decimal.Decimal
	// UnitAmount is the per-unit amount.
	UnitAmount decimal.Decimal
	// AmountDetails holds model-specific breakdown data.
	AmountDetails map[string]any
}

// Strategy is the interface all charge model strategies must satisfy.
type Strategy interface {
	// Compute calculates the charge for the given number of units and model properties.
	Compute(units decimal.Decimal, properties map[string]any) (Result, error)
}

// New returns the Strategy for the given charge model name.
// Returns an error for unknown model names.
func New(chargeModel string) (Strategy, error) {
	switch chargeModel {
	case "standard":
		return &StandardStrategy{}, nil
	case "graduated":
		return &GraduatedStrategy{}, nil
	case "package":
		return &PackageStrategy{}, nil
	case "percentage":
		return &PercentageStrategy{}, nil
	case "volume":
		return &VolumeStrategy{}, nil
	case "graduated_percentage":
		return &GraduatedPercentageStrategy{}, nil
	case "custom":
		return &CustomStrategy{}, nil
	case "dynamic":
		return &DynamicStrategy{}, nil
	default:
		return nil, fmt.Errorf("chargemodels: unknown charge model %q", chargeModel)
	}
}

// decimalFromMap extracts a decimal.Decimal from a map key, returning zero if absent or invalid.
func decimalFromMap(m map[string]any, key string) decimal.Decimal {
	v, ok := m[key]
	if !ok || v == nil {
		return decimal.Zero
	}
	switch val := v.(type) {
	case string:
		d, err := decimal.NewFromString(val)
		if err != nil {
			return decimal.Zero
		}
		return d
	case float64:
		return decimal.NewFromFloat(val)
	case int:
		return decimal.NewFromInt(int64(val))
	case int64:
		return decimal.NewFromInt(val)
	}
	return decimal.Zero
}

// intFromMap extracts an int64 from a map key, returning 0 if absent or invalid.
func intFromMap(m map[string]any, key string) int64 {
	v, ok := m[key]
	if !ok || v == nil {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return int64(val)
	case int:
		return int64(val)
	case int64:
		return val
	}
	return 0
}

// rangesFromMap extracts a slice of range maps from a map key.
func rangesFromMap(m map[string]any, key string) []map[string]any {
	v, ok := m[key]
	if !ok || v == nil {
		return nil
	}
	raw, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		if rm, ok := item.(map[string]any); ok {
			out = append(out, rm)
		}
	}
	return out
}

// toValueFromRange returns the to_value from a range map, or nil for open-ended.
func toValueFromRange(r map[string]any) *decimal.Decimal {
	v, ok := r["to_value"]
	if !ok || v == nil {
		return nil
	}
	d := decimalFromMap(r, "to_value")
	return &d
}
