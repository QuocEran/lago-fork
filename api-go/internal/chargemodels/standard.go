package chargemodels

import "github.com/shopspring/decimal"

// StandardStrategy bills units * amount.
// properties: { "amount": "0.50" }
type StandardStrategy struct{}

func (s *StandardStrategy) Compute(units decimal.Decimal, properties map[string]any) (Result, error) {
	amount := decimalFromMap(properties, "amount")
	total := units.Mul(amount)

	unitAmount := decimal.Zero
	if !units.IsZero() {
		unitAmount = total.Div(units)
	}

	return Result{
		Amount:     total,
		UnitAmount: unitAmount,
		AmountDetails: map[string]any{
			"amount": amount.String(),
		},
	}, nil
}
