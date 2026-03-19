// Package flat implements charge model strategies that compute a flat per-unit price.
// Strategies: standard, dynamic, custom.
package flat

import (
	"github.com/getlago/lago/api-go/internal/chargemodels/base"
	"github.com/shopspring/decimal"
)

// StandardStrategy bills units * amount.
// properties: { "amount": "0.50" }
type StandardStrategy struct{}

func (s *StandardStrategy) Compute(units decimal.Decimal, properties map[string]any) (base.Result, error) {
	amount := base.DecimalFromMap(properties, "amount")
	total := units.Mul(amount)

	unitAmount := decimal.Zero
	if !units.IsZero() {
		unitAmount = total.Div(units)
	}

	return base.Result{
		Amount:     total,
		UnitAmount: unitAmount,
		AmountDetails: map[string]any{
			"amount": amount.String(),
		},
	}, nil
}
