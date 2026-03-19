package invoices_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	domain "github.com/getlago/lago/api-go/internal/domain/invoices"
)

func TestCalculateTotals(t *testing.T) {
	tests := []struct {
		name     string
		input    domain.TotalsInput
		expected domain.TotalsResult
	}{
		{
			name: "simple fees no tax no credits",
			input: domain.TotalsInput{
				Fees:      []int64{1000, 2000, 500},
				TaxesRate: 0,
			},
			expected: domain.TotalsResult{
				FeesAmountCents:             3500,
				SubTotalExcludingTaxesCents: 3500,
				TaxesAmountCents:            0,
				SubTotalIncludingTaxesCents: 3500,
				TotalAmountCents:            3500,
			},
		},
		{
			name: "20% tax rate on fees",
			input: domain.TotalsInput{
				Fees:      []int64{10000},
				TaxesRate: 20.0,
			},
			expected: domain.TotalsResult{
				FeesAmountCents:             10000,
				SubTotalExcludingTaxesCents: 10000,
				TaxesAmountCents:            2000,
				SubTotalIncludingTaxesCents: 12000,
				TotalAmountCents:            12000,
			},
		},
		{
			name: "coupon reduces pre-tax subtotal",
			input: domain.TotalsInput{
				Fees:               []int64{10000},
				CouponsAmountCents: 2000,
				TaxesRate:          20.0,
			},
			expected: domain.TotalsResult{
				FeesAmountCents:             10000,
				CouponsAmountCents:          2000,
				SubTotalExcludingTaxesCents: 8000,
				TaxesAmountCents:            1600,
				SubTotalIncludingTaxesCents: 9600,
				TotalAmountCents:            9600,
			},
		},
		{
			name: "credit note reduces total after tax",
			input: domain.TotalsInput{
				Fees:                   []int64{10000},
				TaxesRate:              20.0,
				CreditNotesAmountCents: 1000,
			},
			expected: domain.TotalsResult{
				FeesAmountCents:             10000,
				SubTotalExcludingTaxesCents: 10000,
				TaxesAmountCents:            2000,
				SubTotalIncludingTaxesCents: 12000,
				CreditNotesAmountCents:      1000,
				TotalAmountCents:            11000,
			},
		},
		{
			name: "progressive billing credit reduces pre-tax subtotal",
			input: domain.TotalsInput{
				Fees:                          []int64{10000},
				ProgressiveBillingCreditCents: 3000,
				TaxesRate:                     10.0,
			},
			expected: domain.TotalsResult{
				FeesAmountCents:                    10000,
				ProgressiveBillingCreditAmountCents: 3000,
				SubTotalExcludingTaxesCents:         7000,
				TaxesAmountCents:                    700,
				SubTotalIncludingTaxesCents:         7700,
				TotalAmountCents:                    7700,
			},
		},
		{
			name: "tax ceiling (fractional cents round up)",
			input: domain.TotalsInput{
				Fees:      []int64{333},
				TaxesRate: 10.0,
			},
			// 333 * 10% = 33.3 → ceil → 34
			expected: domain.TotalsResult{
				FeesAmountCents:             333,
				SubTotalExcludingTaxesCents: 333,
				TaxesAmountCents:            34,
				SubTotalIncludingTaxesCents: 367,
				TotalAmountCents:            367,
			},
		},
		{
			name: "credit note cannot make total negative",
			input: domain.TotalsInput{
				Fees:                   []int64{1000},
				TaxesRate:              0,
				CreditNotesAmountCents: 5000,
			},
			expected: domain.TotalsResult{
				FeesAmountCents:             1000,
				SubTotalExcludingTaxesCents: 1000,
				TaxesAmountCents:            0,
				SubTotalIncludingTaxesCents: 1000,
				CreditNotesAmountCents:      5000,
				TotalAmountCents:            0,
			},
		},
		{
			name: "zero fees produces zero totals",
			input: domain.TotalsInput{
				Fees:      []int64{},
				TaxesRate: 20.0,
			},
			expected: domain.TotalsResult{
				FeesAmountCents:             0,
				SubTotalExcludingTaxesCents: 0,
				TaxesAmountCents:            0,
				SubTotalIncludingTaxesCents: 0,
				TotalAmountCents:            0,
			},
		},
		{
			name: "combined coupons + credit notes + tax",
			input: domain.TotalsInput{
				Fees:                   []int64{10000, 5000},
				CouponsAmountCents:     2000,
				TaxesRate:              10.0,
				CreditNotesAmountCents: 500,
			},
			// fees=15000, sub_ex=13000, tax=1300, sub_inc=14300, total=13800
			expected: domain.TotalsResult{
				FeesAmountCents:             15000,
				CouponsAmountCents:          2000,
				SubTotalExcludingTaxesCents: 13000,
				TaxesAmountCents:            1300,
				SubTotalIncludingTaxesCents: 14300,
				CreditNotesAmountCents:      500,
				TotalAmountCents:            13800,
			},
		},
		{
			name: "negative coupon surplus (coupon > fees) keeps subtotal negative, zero tax",
			input: domain.TotalsInput{
				Fees:               []int64{500},
				CouponsAmountCents: 1000,
				TaxesRate:          20.0,
			},
			// fees=500, sub_ex=-500, tax=0 (not applied on negative), sub_inc=-500, total=0
			expected: domain.TotalsResult{
				FeesAmountCents:             500,
				CouponsAmountCents:          1000,
				SubTotalExcludingTaxesCents: -500,
				TaxesAmountCents:            0,
				SubTotalIncludingTaxesCents: -500,
				TotalAmountCents:            0,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := domain.CalculateTotals(tc.input)
			assert.Equal(t, tc.expected.FeesAmountCents, actual.FeesAmountCents, "FeesAmountCents")
			assert.Equal(t, tc.expected.CouponsAmountCents, actual.CouponsAmountCents, "CouponsAmountCents")
			assert.Equal(t, tc.expected.ProgressiveBillingCreditAmountCents, actual.ProgressiveBillingCreditAmountCents, "ProgressiveBillingCreditAmountCents")
			assert.Equal(t, tc.expected.SubTotalExcludingTaxesCents, actual.SubTotalExcludingTaxesCents, "SubTotalExcludingTaxesCents")
			assert.Equal(t, tc.expected.TaxesAmountCents, actual.TaxesAmountCents, "TaxesAmountCents")
			assert.Equal(t, tc.expected.SubTotalIncludingTaxesCents, actual.SubTotalIncludingTaxesCents, "SubTotalIncludingTaxesCents")
			assert.Equal(t, tc.expected.CreditNotesAmountCents, actual.CreditNotesAmountCents, "CreditNotesAmountCents")
			assert.Equal(t, tc.expected.TotalAmountCents, actual.TotalAmountCents, "TotalAmountCents")
		})
	}
}
