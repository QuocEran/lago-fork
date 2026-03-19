package chargemodels

import (
	"math"

	"github.com/shopspring/decimal"
)

// VolumeStrategy finds the pricing tier that contains total units
// and bills all units at that tier's per-unit rate plus flat fee.
// properties: { "volume_ranges": [ { "from_value": 0, "to_value": 10, "per_unit_amount": "1.0", "flat_amount": "5.0" }, ... ] }
type VolumeStrategy struct{}

func (s *VolumeStrategy) Compute(units decimal.Decimal, properties map[string]any) (Result, error) {
	if units.IsZero() {
		return Result{
			Amount:     decimal.Zero,
			UnitAmount: decimal.Zero,
			AmountDetails: map[string]any{
				"flat_unit_amount":    "0",
				"per_unit_amount":     "0",
				"per_unit_total_amount": "0",
			},
		}, nil
	}

	ranges := rangesFromMap(properties, "volume_ranges")
	unitsCeil := int64(math.Ceil(units.InexactFloat64()))

	var matchedRange map[string]any
	for _, r := range ranges {
		from := intFromMap(r, "from_value")
		toValuePtr := toValueFromRange(r)

		if int64(from) <= unitsCeil {
			if toValuePtr == nil || units.LessThanOrEqual(*toValuePtr) {
				matchedRange = r
				break
			}
		}
	}

	if matchedRange == nil {
		return Result{Amount: decimal.Zero, UnitAmount: decimal.Zero, AmountDetails: map[string]any{}}, nil
	}

	perUnit := decimalFromMap(matchedRange, "per_unit_amount")
	flat := decimalFromMap(matchedRange, "flat_amount")

	perUnitTotal := units.Mul(perUnit)
	total := perUnitTotal.Add(flat)

	unitAmount := decimal.Zero
	if !units.IsZero() {
		unitAmount = total.Div(units)
	}

	return Result{
		Amount:     total,
		UnitAmount: unitAmount,
		AmountDetails: map[string]any{
			"flat_unit_amount":       flat.String(),
			"per_unit_amount":        perUnit.String(),
			"per_unit_total_amount":  perUnitTotal.String(),
		},
	}, nil
}
