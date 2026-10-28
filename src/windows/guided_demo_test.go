package main

import (
	"math"
	"testing"
)

// Guide demonstration uses the production solver with real game grid units.
// This test does NOT substitute fixed narration numbers for a calculation.
func TestGuideDemonstrationFiringSolutions(t *testing.T) {
	first, err := solveL81("100.42", "114.55", "103.42", "118.55")
	if err != nil || !first.HasMil || math.Abs(first.Range-500) > 1e-6 {
		t.Fatalf("first guide target must be in range at 500 m: %+v, %v", first, err)
	}
	moved, err := solveL81("101.42", "114.55", "104.42", "118.55")
	if err != nil || !moved.HasMil || math.Abs(moved.Range-500) > 1e-6 {
		t.Fatalf("after moving actual, target must still be in range: %+v, %v", moved, err)
	}
	corrected, err := solveL81("101.42", "114.55", "104.42", "119.55")
	if err != nil || !corrected.HasMil || corrected.Range <= 500 || corrected.Range >= 684 {
		t.Fatalf("corrected target must update solution within calibration: %+v, %v", corrected, err)
	}
}
