// Package chargemodels is the public entry point for all charge model strategies.
// It re-exports the Strategy interface and Result type from the base sub-package,
// and provides the New factory for looking up a strategy by its model name.
package chargemodels

import (
	"fmt"

	"github.com/getlago/lago/api-go/internal/chargemodels/aggregated"
	"github.com/getlago/lago/api-go/internal/chargemodels/base"
	"github.com/getlago/lago/api-go/internal/chargemodels/flat"
	"github.com/getlago/lago/api-go/internal/chargemodels/tiered"
)

// Strategy is the interface all charge model strategies must satisfy.
type Strategy = base.Strategy

// Result holds the output of a charge model computation.
type Result = base.Result

// New returns the Strategy for the given charge model name.
// Returns an error for unknown model names.
func New(chargeModel string) (Strategy, error) {
	switch chargeModel {
	case "standard":
		return &flat.StandardStrategy{}, nil
	case "dynamic":
		return &flat.DynamicStrategy{}, nil
	case "custom":
		return &flat.CustomStrategy{}, nil
	case "graduated":
		return &tiered.GraduatedStrategy{}, nil
	case "graduated_percentage":
		return &tiered.GraduatedPercentageStrategy{}, nil
	case "volume":
		return &tiered.VolumeStrategy{}, nil
	case "package":
		return &aggregated.PackageStrategy{}, nil
	case "percentage":
		return &aggregated.PercentageStrategy{}, nil
	default:
		return nil, fmt.Errorf("chargemodels: unknown charge model %q", chargeModel)
	}
}
