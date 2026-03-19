// Package tiered implements charge model strategies that compute prices across
// progressive or volume-based tiers.
// Strategies: graduated, graduated_percentage, volume.
package tiered

import (
	"github.com/getlago/lago/api-go/internal/chargemodels/base"
	"github.com/shopspring/decimal"
)

// GraduatedStrategy bills units across tiered ranges, each with a per-unit rate and flat fee.
// properties: { "graduated_ranges": [ { "from_value": 0, "to_value": 10, "per_unit_amount": "1.0", "flat_amount": "0" }, ... ] }
type GraduatedStrategy struct{}

type graduatedRangeResult struct {
	FromValue     int64
	ToValue       *int64
	PerUnitAmount string
	FlatAmount    string
	Units         string
	TotalWithFlat string
}

func (s *GraduatedStrategy) Compute(units decimal.Decimal, properties map[string]any) (base.Result, error) {
	ranges := base.RangesFromMap(properties, "graduated_ranges")

	var rangeResults []graduatedRangeResult
	total := decimal.Zero

	for _, r := range ranges {
		fromValue := base.IntFromMap(r, "from_value")
		toValueDec := base.ToValueFromRange(r)
		perUnit := base.DecimalFromMap(r, "per_unit_amount")
		flat := base.DecimalFromMap(r, "flat_amount")

		var from decimal.Decimal
		if len(rangeResults) == 0 {
			from = decimal.NewFromInt(fromValue)
		} else {
			prevToValue := base.IntFromMap(ranges[len(rangeResults)-1], "to_value")
			from = decimal.NewFromInt(prevToValue)
		}

		var unitsInRange decimal.Decimal
		if toValueDec == nil {
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

		rangeResults = append(rangeResults, graduatedRangeResult{
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

	return base.Result{
		Amount:        total,
		UnitAmount:    unitAmount,
		AmountDetails: map[string]any{"graduated_ranges": rangeResults},
	}, nil
}
