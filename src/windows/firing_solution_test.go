package main

import (
	"math"
	"strconv"
	"strings"
	"testing"
)

func TestL81ExactAndInterpolation(t *testing.T) {
	cases := []struct {
		ax, ay, tx, ty    string
		rng, mil, bearing float64
		valid             bool
	}{
		{"0", "0", "0", "1.1", 110, 900, 0, true},
		{"0", "0", "1.1", "0", 110, 900, 90, true},
		{"0", "0", "0", "-1.1", 110, 900, 180, true},
		{"0", "0", "-1.1", "0", 110, 900, 270, true},
		{"0", "0", "0", "1.21", 121, 875, 0, true},
		{"0", "0", "0", "0.79", 79, 0, 0, false},
		{"0", "0", "0", "6.85", 685, 0, 0, false},
		{"98.38", "110.43", "98.38", "111.53", 110, 900, 0, true},
	}
	for _, c := range cases {
		s, e := solveL81(c.ax, c.ay, c.tx, c.ty)
		if e != nil {
			t.Fatal(e)
		}
		if math.Abs(s.Range-c.rng) > 1e-6 || math.Abs(s.Bearing-c.bearing) > 1e-6 || s.HasMil != c.valid {
			t.Errorf("%+v => %+v", c, s)
		}
		if s.HasMil && math.Abs(s.Mil-c.mil) > 1e-6 {
			t.Errorf("expected %.2f mil, got %.2f", c.mil, s.Mil)
		}
	}
}
func TestL81CalibrationBoundary(t *testing.T) {
	for _, c := range []struct{ rangeM, mil float64 }{{80, 950}, {684, 150}} {
		s, e := solveL81("0", "0", "0", fmtRangeCoord(c.rangeM))
		if e != nil || !s.HasMil || math.Abs(s.Mil-c.mil) > 1e-6 {
			t.Errorf("%.0fm: %+v, %v", c.rangeM, s, e)
		}
	}
}
func fmtRangeCoord(m float64) string {
	return strings.TrimRight(strings.TrimRight(strconvFormat(m/100), "0"), ".")
}
func strconvFormat(f float64) string { return strconv.FormatFloat(f, 'f', 3, 64) }
func TestL81RejectBadCoordinates(t *testing.T) {
	for _, bad := range []string{"", "garbage", "NaN", "Inf"} {
		if _, e := solveL81(bad, "0", "0", "1"); e == nil {
			t.Errorf("accepted %q", bad)
		}
	}
}
func TestL81NoExtrap(t *testing.T) {
	s, _ := solveL81("0", "0", "0", "8")
	if strings.Contains(s.spokenSolution(), "Elevation") || s.HasMil {
		t.Fatal("extrapolated out of range")
	}
}
func TestPiperEchoFilter(t *testing.T) {
	if !likelyPiperEcho("roger target ready x ray next", "Roger. Target ready. X-ray next.") {
		t.Fatal("missed full echo")
	}
	for _, s := range []string{"x ray", "yank", "nine eight point three", "target x ray nine eight"} {
		if likelyPiperEcho(s, "Roger. X-ray nine eight point three understood. Yankee next.") {
			t.Errorf("operator phrase blocked: %s", s)
		}
	}
}
