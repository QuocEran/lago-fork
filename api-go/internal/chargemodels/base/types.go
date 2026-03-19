// Package base defines the core types shared across all charge model strategy sub-packages.
package base

import "github.com/shopspring/decimal"

// Result holds the output of a charge model computation.
type Result struct {
	Amount        decimal.Decimal
	UnitAmount    decimal.Decimal
	AmountDetails map[string]any
}

// Strategy is the interface all charge model strategies must satisfy.
type Strategy interface {
	Compute(units decimal.Decimal, properties map[string]any) (Result, error)
}
