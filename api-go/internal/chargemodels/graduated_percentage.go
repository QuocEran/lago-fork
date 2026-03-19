package chargemodels

import "github.com/shopspring/decimal"

// GraduatedPercentageStrategy bills a graduated percentage across tiers.
// Each tier applies a rate (%) on the units in that tier plus an optional flat fee.
// properties: { "graduated_percentage_ranges": [ { "from_value": 0, "to_value": 100, "rate": "1.5", "flat_amount": "0" }, ... ] }
type GraduatedPercentageStrategy struct{}

func (s *GraduatedPercentageStrategy) Compute(units decimal.Decimal, properties map[string]any) (Result, error) {
	ranges := rangesFromMap(properties, "graduated_percentage_ranges")
	hundred := decimal.NewFromInt(100)

	type rangeResult struct {
		FromValue        int64
		ToValue          *int64
		Rate             string
		FlatAmount       string
		Units            string
		PercentageAmount string
		TotalWithFlat    string
	}

	var rangeResults []rangeResult
	total := decimal.Zero
	pricedUnits := decimal.Zero

	for _, r := range ranges {
		fromValue := intFromMap(r, "from_value")
		toValueDec := toValueFromRange(r)
		rate := decimalFromMap(r, "rate")
		flat := decimalFromMap(r, "flat_amount")

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
		rangeResults = append(rangeResults, rangeResult{
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

	return Result{
		Amount:        total,
		UnitAmount:    unitAmount,
		AmountDetails: map[string]any{"graduated_percentage_ranges": rangeResults},
	}, nil
}
