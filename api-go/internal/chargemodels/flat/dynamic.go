package flat

import (
	"github.com/getlago/lago/api-go/internal/chargemodels/base"
	"github.com/shopspring/decimal"
)

// DynamicStrategy is used when pricing is determined externally at event time.
// Like standard, it bills units * amount, but marks the charge as dynamically priced.
// The per-event amount is stored in the "amount" property.
type DynamicStrategy struct{}

func (s *DynamicStrategy) Compute(units decimal.Decimal, properties map[string]any) (base.Result, error) {
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
