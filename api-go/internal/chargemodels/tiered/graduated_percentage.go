package tiered

import (
	"github.com/getlago/lago/api-go/internal/chargemodels/base"
	"github.com/shopspring/decimal"
)

// GraduatedPercentageStrategy bills a graduated percentage across tiers.
// Each tier applies a rate (%) on the units in that tier plus an optional flat fee.
// properties: { "graduated_percentage_ranges": [ { "from_value": 0, "to_value": 100, "rate": "1.5", "flat_amount": "0" }, ... ] }
type GraduatedPercentageStrategy struct{}

type graduatedPercentageRangeResult struct {
	FromValue        int64
	ToValue          *int64
	Rate             string
	FlatAmount       string
	Units            string
	PercentageAmount string
	TotalWithFlat    string
}

func (s *GraduatedPercentageStrategy) Compute(units decimal.Decimal, properties map[string]any) (base.Result, error) {
	ranges := base.RangesFromMap(properties, "graduated_percentage_ranges")
	hundred := decimal.NewFromInt(100)

	var rangeResults []graduatedPercentageRangeResult
	total := decimal.Zero
	pricedUnits := decimal.Zero

	for _, r := range ranges {
		fromValue := base.IntFromMap(r, "from_value")
		toValueDec := base.ToValueFromRange(r)
		rate := base.DecimalFromMap(r, "rate")
		flat := base.DecimalFromMap(r, "flat_amount")

		from := pricedUnits
		var unitsInRange decimal.Decimal
		if toValueDec == nil {
			unitsInRange = units.Sub(from)
		} else {
			if units.LessThanOrEqual(from) {
				break
			}
			if units.LessThanOrEqual(*toValueDec) {
				unitsInRange = units.Sub(from)
			} else {
				unitsInRange = toValueDec.Sub(from)
			}
		}

		if unitsInRange.IsNegative() {
			unitsInRange = decimal.Zero
		}

		percentageAmount := unitsInRange.Mul(rate).Div(hundred)
		tierTotal := percentageAmount.Add(flat)
		total = total.Add(tierTotal)
		pricedUnits = pricedUnits.Add(unitsInRange)

		var toValuePtr *int64
		if toValueDec != nil {
			v := toValueDec.IntPart()
			toValuePtr = &v
		}
		rangeResults = append(rangeResults, graduatedPercentageRangeResult{
			FromValue:        fromValue,
			ToValue:          toValuePtr,
			Rate:             rate.String(),
			FlatAmount:       flat.String(),
			Units:            unitsInRange.String(),
			PercentageAmount: percentageAmount.String(),
			TotalWithFlat:    tierTotal.String(),
		})

		if toValueDec != nil && units.LessThanOrEqual(*toValueDec) {
			break
		}
	}

	unitAmount := decimal.Zero
	if !units.IsZero() {
		unitAmount = total.Div(units)
	}

	return base.Result{
		Amount:        total,
		UnitAmount:    unitAmount,
		AmountDetails: map[string]any{"graduated_percentage_ranges": rangeResults},
	}, nil
}
