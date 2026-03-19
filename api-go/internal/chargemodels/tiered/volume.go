package tiered

import (
	"math"

	"github.com/getlago/lago/api-go/internal/chargemodels/base"
	"github.com/shopspring/decimal"
)

// VolumeStrategy finds the pricing tier that contains total units
// and bills all units at that tier's per-unit rate plus flat fee.
// properties: { "volume_ranges": [ { "from_value": 0, "to_value": 10, "per_unit_amount": "1.0", "flat_amount": "5.0" }, ... ] }
type VolumeStrategy struct{}

func (s *VolumeStrategy) Compute(units decimal.Decimal, properties map[string]any) (base.Result, error) {
	if units.IsZero() {
		return base.Result{
			Amount:     decimal.Zero,
			UnitAmount: decimal.Zero,
			AmountDetails: map[string]any{
				"flat_unit_amount":      "0",
				"per_unit_amount":       "0",
				"per_unit_total_amount": "0",
			},
		}, nil
	}

	ranges := base.RangesFromMap(properties, "volume_ranges")
	unitsCeil := int64(math.Ceil(units.InexactFloat64()))

	var matchedRange map[string]any
	for _, r := range ranges {
		from := base.IntFromMap(r, "from_value")
		toValuePtr := base.ToValueFromRange(r)

		if from <= unitsCeil {
			if toValuePtr == nil || units.LessThanOrEqual(*toValuePtr) {
				matchedRange = r
				break
			}
		}
	}

	if matchedRange == nil {
		return base.Result{Amount: decimal.Zero, UnitAmount: decimal.Zero, AmountDetails: map[string]any{}}, nil
	}

	perUnit := base.DecimalFromMap(matchedRange, "per_unit_amount")
	flat := base.DecimalFromMap(matchedRange, "flat_amount")

	perUnitTotal := units.Mul(perUnit)
	total := perUnitTotal.Add(flat)

	unitAmount := decimal.Zero
	if !units.IsZero() {
		unitAmount = total.Div(units)
	}

	return base.Result{
		Amount:     total,
		UnitAmount: unitAmount,
		AmountDetails: map[string]any{
			"flat_unit_amount":      flat.String(),
			"per_unit_amount":       perUnit.String(),
			"per_unit_total_amount": perUnitTotal.String(),
		},
	}, nil
}
