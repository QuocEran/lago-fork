package flat

import (
	"github.com/getlago/lago/api-go/internal/chargemodels/base"
	"github.com/shopspring/decimal"
)

// CustomStrategy is used with custom_agg billable metrics where the
// amount is computed externally and passed directly in the "amount" property.
type CustomStrategy struct{}

func (s *CustomStrategy) Compute(units decimal.Decimal, properties map[string]any) (base.Result, error) {
	amount := base.DecimalFromMap(properties, "amount")

	unitAmount := decimal.Zero
	if !units.IsZero() {
		unitAmount = amount.Div(units)
	}

	return base.Result{
		Amount:        amount,
		UnitAmount:    unitAmount,
		AmountDetails: map[string]any{"amount": amount.String()},
	}, nil
}
