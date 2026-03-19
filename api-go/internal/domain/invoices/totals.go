package invoices

import "math"

// TotalsInput holds the raw inputs for invoice total calculation.
// All values are in the smallest currency unit (cents).
type TotalsInput struct {
	// Fees is the list of fee amounts in cents to sum.
	Fees []int64
	// CouponsAmountCents is the total coupon credit applied before taxes (from credits table, coupon kind).
	CouponsAmountCents int64
	// ProgressiveBillingCreditCents is the progressive-billing credit applied before taxes.
	ProgressiveBillingCreditCents int64
	// CreditNotesAmountCents is the total credit-note credit applied after taxes.
	CreditNotesAmountCents int64
	// TaxesRate is the percentage tax rate (e.g. 20.0 for 20%).
	TaxesRate float64
}

// TotalsResult holds all computed invoice monetary fields.
type TotalsResult struct {
	FeesAmountCents                    int64
	CouponsAmountCents                 int64
	ProgressiveBillingCreditAmountCents int64
	SubTotalExcludingTaxesCents        int64
	TaxesAmountCents                   int64
	SubTotalIncludingTaxesCents        int64
	CreditNotesAmountCents             int64
	TotalAmountCents                   int64
}

// CalculateTotals computes all invoice monetary fields from the given inputs.
// The formula mirrors Rails Invoices::ComputeAmountsFromFees.
//
//	fees_amount_cents         = SUM(fees)
//	sub_total_ex_taxes        = fees - progressive_billing_credit - coupons
//	taxes_amount_cents        = CEIL(sub_total_ex_taxes * taxes_rate / 100)   [never negative]
//	sub_total_inc_taxes       = sub_total_ex_taxes + taxes_amount_cents
//	total_amount_cents        = sub_total_inc_taxes - credit_notes             [floor at 0]
func CalculateTotals(input TotalsInput) TotalsResult {
	var feesTotal int64
	for _, f := range input.Fees {
		feesTotal += f
	}

	subTotalExcludingTaxes := feesTotal - input.ProgressiveBillingCreditCents - input.CouponsAmountCents

	// Tax is computed on the subtotal excluding taxes; never negative.
	var taxesAmount int64
	if subTotalExcludingTaxes > 0 && input.TaxesRate > 0 {
		taxesAmount = int64(math.Ceil(float64(subTotalExcludingTaxes) * input.TaxesRate / 100.0))
	}

	subTotalIncludingTaxes := subTotalExcludingTaxes + taxesAmount

	totalAmount := subTotalIncludingTaxes - input.CreditNotesAmountCents
	if totalAmount < 0 {
		totalAmount = 0
	}

	return TotalsResult{
		FeesAmountCents:                    feesTotal,
		CouponsAmountCents:                 input.CouponsAmountCents,
		ProgressiveBillingCreditAmountCents: input.ProgressiveBillingCreditCents,
		SubTotalExcludingTaxesCents:         subTotalExcludingTaxes,
		TaxesAmountCents:                    taxesAmount,
		SubTotalIncludingTaxesCents:         subTotalIncludingTaxes,
		CreditNotesAmountCents:              input.CreditNotesAmountCents,
		TotalAmountCents:                    totalAmount,
	}
}
