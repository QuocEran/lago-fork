package chargemodels_test

import (
	"testing"

	"github.com/shopspring/decimal"

	"github.com/getlago/lago/api-go/internal/chargemodels"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func dec(s string) decimal.Decimal {
	d, _ := decimal.NewFromString(s)
	return d
}

func assertAmount(t *testing.T, label string, want, got decimal.Decimal) {
	t.Helper()
	if !want.Equal(got) {
		t.Errorf("%s: want %s, got %s", label, want.String(), got.String())
	}
}

// ── standard ─────────────────────────────────────────────────────────────────

func TestStandardStrategy(t *testing.T) {
	svc, _ := chargemodels.New("standard")

	tests := []struct {
		name       string
		units      string
		amount     string
		wantAmount string
	}{
		{"zero units", "0", "0.50", "0"},
		{"5 units at 0.50", "5", "0.50", "2.50"},
		{"100 units at 1.00", "100", "1.00", "100.00"},
		{"fractional units", "2.5", "4.00", "10.00"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			inputUnits := dec(tc.units)
			props := map[string]any{"amount": tc.amount}
			result, err := svc.Compute(inputUnits, props)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertAmount(t, "amount", dec(tc.wantAmount), result.Amount)
		})
	}
}

// ── graduated ────────────────────────────────────────────────────────────────

func TestGraduatedStrategy(t *testing.T) {
	svc, _ := chargemodels.New("graduated")

	// Tiers: 0-10 @ $1.0 + $0 flat; 11-50 @ $0.5 + $5 flat; 51+ @ $0.25 + $10 flat
	props := map[string]any{
		"graduated_ranges": []any{
			map[string]any{"from_value": float64(0), "to_value": float64(10), "per_unit_amount": "1.0", "flat_amount": "0"},
			map[string]any{"from_value": float64(10), "to_value": float64(50), "per_unit_amount": "0.5", "flat_amount": "5"},
			map[string]any{"from_value": float64(50), "to_value": nil, "per_unit_amount": "0.25", "flat_amount": "10"},
		},
	}

	tests := []struct {
		name       string
		units      string
		wantAmount string
	}{
		// 0 units: no tiers hit
		{"0 units", "0", "0"},
		// 5 units: 5*1.0 + 0 flat = 5
		{"5 units (first tier only)", "5", "5"},
		// 10 units: 10*1.0 + 0 flat = 10
		{"10 units (boundary)", "10", "10"},
		// 20 units: 10*1.0 + (10*0.5 + 5) = 10 + 10 = 20
		{"20 units (two tiers)", "20", "20"},
		// 60 units: 10*1 + (40*0.5 + 5) + (10*0.25 + 10) = 10 + 25 + 12.5 = 47.5
		{"60 units (three tiers)", "60", "47.5"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := svc.Compute(dec(tc.units), props)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertAmount(t, "amount", dec(tc.wantAmount), result.Amount)
		})
	}
}

// ── package ───────────────────────────────────────────────────────────────────

func TestPackageStrategy(t *testing.T) {
	svc, _ := chargemodels.New("package")

	// $5 per package of 10, first 5 free
	props := map[string]any{
		"amount":       "5.0",
		"package_size": "10",
		"free_units":   "5",
	}

	tests := []struct {
		name       string
		units      string
		wantAmount string
	}{
		{"0 units", "0", "0"},
		{"5 units (all free)", "5", "0"},
		{"15 units (1 package)", "15", "5"},
		{"25 units (2 packages)", "25", "10"},
		// 30 - 5 = 25 paid units → ceil(25/10)=3 packages → $15
		{"30 units (3 packages)", "30", "15"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := svc.Compute(dec(tc.units), props)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertAmount(t, "amount", dec(tc.wantAmount), result.Amount)
		})
	}
}

// ── percentage ────────────────────────────────────────────────────────────────

func TestPercentageStrategy(t *testing.T) {
	svc, _ := chargemodels.New("percentage")

	// 2% rate, $0.10 fixed fee, first 10 units free
	props := map[string]any{
		"rate":                              "2.0",
		"fixed_amount":                      "0.10",
		"free_units_per_total_aggregation":  "10",
	}

	tests := []struct {
		name       string
		units      string
		wantAmount string
	}{
		{"0 units", "0", "0"},
		{"10 units (all free, no fixed)", "10", "0"},
		// 50 units: (50-10)*2%=0.8 + 0.10 fixed = 0.90
		{"50 units", "50", "0.90"},
		// 100 units: (100-10)*2% = 1.8 + 0.10 = 1.90
		{"100 units", "100", "1.90"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := svc.Compute(dec(tc.units), props)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertAmount(t, "amount", dec(tc.wantAmount), result.Amount)
		})
	}
}

// ── volume ────────────────────────────────────────────────────────────────────

func TestVolumeStrategy(t *testing.T) {
	svc, _ := chargemodels.New("volume")

	// Tiers: 1-10 @ $1.0 + $0 flat; 11-100 @ $0.5 + $5 flat; 101+ @ $0.1 + $20 flat
	props := map[string]any{
		"volume_ranges": []any{
			map[string]any{"from_value": float64(1), "to_value": float64(10), "per_unit_amount": "1.0", "flat_amount": "0"},
			map[string]any{"from_value": float64(11), "to_value": float64(100), "per_unit_amount": "0.5", "flat_amount": "5"},
			map[string]any{"from_value": float64(101), "to_value": nil, "per_unit_amount": "0.1", "flat_amount": "20"},
		},
	}

	tests := []struct {
		name       string
		units      string
		wantAmount string
	}{
		{"0 units", "0", "0"},
		// 5 units in tier 1: 5*1.0 + 0 = 5
		{"5 units (tier 1)", "5", "5"},
		// 50 units in tier 2: 50*0.5 + 5 = 30
		{"50 units (tier 2)", "50", "30"},
		// 200 units in tier 3: 200*0.1 + 20 = 40
		{"200 units (tier 3)", "200", "40"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := svc.Compute(dec(tc.units), props)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertAmount(t, "amount", dec(tc.wantAmount), result.Amount)
		})
	}
}

// ── graduated_percentage ──────────────────────────────────────────────────────

func TestGraduatedPercentageStrategy(t *testing.T) {
	svc, _ := chargemodels.New("graduated_percentage")

	// Tiers: 0-100 @ 1.5% + $0; 101-500 @ 1.0% + $2; 501+ @ 0.5% + $5
	props := map[string]any{
		"graduated_percentage_ranges": []any{
			map[string]any{"from_value": float64(0), "to_value": float64(100), "rate": "1.5", "flat_amount": "0"},
			map[string]any{"from_value": float64(100), "to_value": float64(500), "rate": "1.0", "flat_amount": "2"},
			map[string]any{"from_value": float64(500), "to_value": nil, "rate": "0.5", "flat_amount": "5"},
		},
	}

	tests := []struct {
		name       string
		units      string
		wantAmount string
	}{
		{"0 units", "0", "0"},
		// 50 units: 50*1.5% + 0 = 0.75
		{"50 units (first tier)", "50", "0.75"},
		// 200 units: 100*1.5% + (100*1.0% + 2) = 1.5 + 3 = 4.5
		{"200 units (two tiers)", "200", "4.5"},
		// 600 units: 100*1.5% + (400*1.0% + 2) + (100*0.5% + 5) = 1.5 + 6 + 5.5 = 13
		{"600 units (three tiers)", "600", "13"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := svc.Compute(dec(tc.units), props)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertAmount(t, "amount", dec(tc.wantAmount), result.Amount)
		})
	}
}

// ── custom ────────────────────────────────────────────────────────────────────

func TestCustomStrategy(t *testing.T) {
	svc, _ := chargemodels.New("custom")

	props := map[string]any{"amount": "42.50"}
	result, err := svc.Compute(dec("10"), props)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertAmount(t, "amount", dec("42.50"), result.Amount)
}

// ── dynamic ───────────────────────────────────────────────────────────────────

func TestDynamicStrategy(t *testing.T) {
	svc, _ := chargemodels.New("dynamic")

	props := map[string]any{"amount": "3.00"}
	result, err := svc.Compute(dec("7"), props)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 7 * 3.00 = 21.00
	assertAmount(t, "amount", dec("21.00"), result.Amount)
}

// ── factory ───────────────────────────────────────────────────────────────────

func TestFactory_UnknownModel(t *testing.T) {
	_, err := chargemodels.New("nonexistent_model")
	if err == nil {
		t.Fatal("expected error for unknown model, got nil")
	}
}

func TestFactory_AllModels(t *testing.T) {
	models := []string{"standard", "graduated", "package", "percentage", "volume", "graduated_percentage", "custom", "dynamic"}
	for _, m := range models {
		t.Run(m, func(t *testing.T) {
			svc, err := chargemodels.New(m)
			if err != nil {
				t.Fatalf("New(%q) returned error: %v", m, err)
			}
			if svc == nil {
				t.Fatalf("New(%q) returned nil strategy", m)
			}
		})
	}
}
