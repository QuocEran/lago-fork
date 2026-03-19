package base

import "github.com/shopspring/decimal"

// DecimalFromMap extracts a decimal.Decimal from a map key, returning zero if absent or invalid.
func DecimalFromMap(m map[string]any, key string) decimal.Decimal {
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

// IntFromMap extracts an int64 from a map key, returning 0 if absent or invalid.
func IntFromMap(m map[string]any, key string) int64 {
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

// RangesFromMap extracts a slice of range maps from a map key.
func RangesFromMap(m map[string]any, key string) []map[string]any {
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

// ToValueFromRange returns the to_value from a range map, or nil for open-ended tiers.
func ToValueFromRange(r map[string]any) *decimal.Decimal {
	v, ok := r["to_value"]
	if !ok || v == nil {
		return nil
	}
	d := DecimalFromMap(r, "to_value")
	return &d
}
