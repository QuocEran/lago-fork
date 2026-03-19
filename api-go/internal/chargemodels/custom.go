package chargemodels

import "github.com/shopspring/decimal"

// CustomStrategy is used with custom_agg billable metrics where the
// amount is computed externally. The amount is passed directly in the
// "amount" property key, so it simply returns it as-is.
type CustomStrategy struct{}

func (s *CustomStrategy) Compute(units decimal.Decimal, properties map[string]any) (Result, error) {
	amount := decimalFromMap(properties, "amount")

	unitAmount := decimal.Zero
	if !units.IsZero() {
		unitAmount = amount.Div(units)
	}

	return Result{
		Amount:        amount,
		UnitAmount:    unitAmount,
		AmountDetails: map[string]any{"amount": amount.String()},
	}, nil
}
