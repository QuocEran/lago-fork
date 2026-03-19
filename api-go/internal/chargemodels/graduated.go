package chargemodels

import "github.com/shopspring/decimal"

// GraduatedStrategy bills units across tiered ranges, each with a per-unit rate and flat fee.
// properties: { "graduated_ranges": [ { "from_value": 0, "to_value": 10, "per_unit_amount": "1.0", "flat_amount": "0" }, ... ] }
type GraduatedStrategy struct{}

func (s *GraduatedStrategy) Compute(units decimal.Decimal, properties map[string]any) (Result, error) {
	ranges := rangesFromMap(properties, "graduated_ranges")

	type rangeResult struct {
		FromValue         int64
		ToValue           *int64
		PerUnitAmount     string
		FlatAmount        string
		Units             string
		TotalWithFlat     string
	}

	var rangeResults []rangeResult
	total := decimal.Zero

	for _, r := range ranges {
		fromValue := intFromMap(r, "from_value")
		toValueDec := toValueFromRange(r)
		perUnit := decimalFromMap(r, "per_unit_amount")
		flat := decimalFromMap(r, "flat_amount")

		var from decimal.Decimal
		if len(rangeResults) == 0 {
			from = decimal.NewFromInt(fromValue)
		} else {
			prevToValue := intFromMap(ranges[len(rangeResults)-1], "to_value")
			from = decimal.NewFromInt(prevToValue)
		}

		var unitsInRange decimal.Decimal
		if toValueDec == nil {
			// Last open-ended tier: all remaining units.
			unitsInRange = units.Sub(from)
		} else {
			tierEnd := *toValueDec
			if units.LessThanOrEqual(from) {
				break
			}
			if units.LessThanOrEqual(tierEnd) {
				unitsInRange = units.Sub(from)
			} else {
				unitsInRange = tierEnd.Sub(from)
			}
		}

		if unitsInRange.IsNegative() {
			unitsInRange = decimal.Zero
		}

		tierTotal := unitsInRange.Mul(perUnit).Add(flat)
		total = total.Add(tierTotal)

		var toValuePtr *int64
		if toValueDec != nil {
			v := toValueDec.IntPart()
			toValuePtr = &v
		}

		rangeResults = append(rangeResults, rangeResult{
			FromValue:     fromValue,
			ToValue:       toValuePtr,
			PerUnitAmount: perUnit.String(),
			FlatAmount:    flat.String(),
			Units:         unitsInRange.String(),
			TotalWithFlat: tierTotal.String(),
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
		AmountDetails: map[string]any{"graduated_ranges": rangeResults},
	}, nil
}
