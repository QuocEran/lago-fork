package chargemodels

import (
	"math"

	"github.com/shopspring/decimal"
)

// PackageStrategy bills by packages of units.
// After subtracting free units, units are grouped into packages of per_package_size;
// each package costs per_package_unit_amount.
// properties: { "amount": "5.0", "package_size": "10", "free_units": "0" }
type PackageStrategy struct{}

func (s *PackageStrategy) Compute(units decimal.Decimal, properties map[string]any) (Result, error) {
	perPackageAmount := decimalFromMap(properties, "amount")
	packageSize := decimalFromMap(properties, "package_size")
	freeUnits := decimalFromMap(properties, "free_units")

	if packageSize.IsZero() {
		packageSize = decimal.NewFromInt(1)
	}

	paidUnits := units.Sub(freeUnits)

	if paidUnits.IsNegative() || paidUnits.IsZero() {
		return Result{
			Amount:     decimal.Zero,
			UnitAmount: decimal.Zero,
			AmountDetails: map[string]any{
				"free_units":             freeUnits.String(),
				"paid_units":             "0.0",
				"per_package_size":       packageSize.String(),
				"per_package_unit_amount": perPackageAmount.String(),
			},
		}, nil
	}

	packageCount := math.Ceil(paidUnits.Div(packageSize).InexactFloat64())
	total := decimal.NewFromFloat(packageCount).Mul(perPackageAmount)

	unitAmount := decimal.Zero
	if !paidUnits.IsZero() {
		unitAmount = total.Div(paidUnits)
	}

	return Result{
		Amount:     total,
		UnitAmount: unitAmount,
		AmountDetails: map[string]any{
			"free_units":              freeUnits.String(),
			"paid_units":              paidUnits.String(),
			"per_package_size":        packageSize.String(),
			"per_package_unit_amount": perPackageAmount.String(),
		},
	}, nil
}
