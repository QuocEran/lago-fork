package chargemodels

import "github.com/shopspring/decimal"

// PercentageStrategy bills a rate% on units above free_units,
// plus an optional fixed fee per event above free_events_per_unit.
// properties: { "rate": "2.0", "fixed_amount": "0.10",
//               "free_units_per_events": "0", "free_units_per_total_aggregation": "0" }
type PercentageStrategy struct{}

func (s *PercentageStrategy) Compute(units decimal.Decimal, properties map[string]any) (Result, error) {
	rate := decimalFromMap(properties, "rate")
	fixedAmount := decimalFromMap(properties, "fixed_amount")
	freeUnitsPerTotalAggregation := decimalFromMap(properties, "free_units_per_total_aggregation")

	hundred := decimal.NewFromInt(100)

	freeUnitsValue := freeUnitsPerTotalAggregation
	if freeUnitsValue.GreaterThan(units) {
		freeUnitsValue = units
	}

	paidUnits := units.Sub(freeUnitsValue)
	if paidUnits.IsNegative() {
		paidUnits = decimal.Zero
	}

	percentageAmount := paidUnits.Mul(rate).Div(hundred)
	fixedTotal := decimal.Zero
	if !paidUnits.IsZero() && !fixedAmount.IsZero() {
		fixedTotal = fixedAmount
	}

	total := percentageAmount.Add(fixedTotal)

	unitAmount := decimal.Zero
	if !units.IsZero() {
		unitAmount = total.Div(units)
	}

	return Result{
		Amount:     total,
		UnitAmount: unitAmount,
		AmountDetails: map[string]any{
			"units":                   units.String(),
			"free_units":              freeUnitsValue.String(),
			"paid_units":              paidUnits.String(),
			"rate":                    rate.String(),
			"per_unit_total_amount":   percentageAmount.String(),
			"fixed_fee_unit_amount":   fixedAmount.String(),
			"fixed_fee_total_amount":  fixedTotal.String(),
		},
	}, nil
}
