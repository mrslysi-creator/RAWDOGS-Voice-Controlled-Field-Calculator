package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// L81 mortar calibration and geometry are ported unchanged from the project's
// src/shared/engine.js and weapons/l81.js (data version 1.1, 24 Sep 2026).
// WARDOGS XY coordinates represent 100 metres per 1.00.
// Outside photographed calibration limits: bearing/range still available,
// but no invented or extrapolated MIL value is supplied.
type L81Mark struct{ Range, Mil float64 }

var l81Marks = [...]L81Mark{
	{80, 950}, {110, 900}, {132, 850}, {187, 800}, {240, 750},
	{290, 700}, {340, 650}, {385, 600}, {430, 550}, {470, 500},
	{510, 450}, {545, 400}, {578, 350}, {609, 300}, {637, 250},
	{661, 200}, {684, 150},
}

type L81Solution struct {
	Bearing  float64
	Range    float64
	Mil      float64
	HasMil   bool
	MinRange float64
	MaxRange float64
}

func solveL81(ax, ay, tx, ty string) (L81Solution, error) {
	coords := [4]float64{}
	for i, s := range []string{ax, ay, tx, ty} {
		n, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
		if err != nil || math.IsNaN(n) || math.IsInf(n, 0) {
			return L81Solution{}, fmt.Errorf("invalid X/Y coordinate")
		}
		coords[i] = n
	}
	dx := (coords[2] - coords[0]) * 100
	dy := (coords[3] - coords[1]) * 100
	r := math.Hypot(dx, dy)
	bearing := math.Atan2(dx, dy) * 180 / math.Pi
	if bearing < 0 {
		bearing += 360
	}
	result := L81Solution{Bearing: bearing, Range: r, MinRange: l81Marks[0].Range, MaxRange: l81Marks[len(l81Marks)-1].Range}
	if r < result.MinRange || r > result.MaxRange {
		return result, nil
	}
	for i := 0; i < len(l81Marks)-1; i++ {
		lower, upper := l81Marks[i], l81Marks[i+1]
		if r >= lower.Range && r <= upper.Range {
			result.Mil = lower.Mil + (r-lower.Range)*(upper.Mil-lower.Mil)/(upper.Range-lower.Range)
			result.HasMil = true
			return result, nil
		}
	}
	return result, fmt.Errorf("range not bracketed in L81 calibration")
}

func (s L81Solution) bearingRounded() int { return int(math.Round(s.Bearing)) % 360 }
func (s L81Solution) bearingText() string { return fmt.Sprintf("%03d degrees", s.bearingRounded()) }
func (s L81Solution) rangeRounded() int   { return int(math.Round(s.Range)) }
func (s L81Solution) milRounded() int     { return int(math.Round(s.Mil)) }

func (s L81Solution) displayText() string {
	if !s.HasMil {
		return fmt.Sprintf("Bearing: %03d°\r\nRange: %d m\r\nMIL: OUT OF RANGE\r\n\r\nSTATUS: L81 80–684 m ONLY", s.bearingRounded(), s.rangeRounded())
	}
	return fmt.Sprintf("Bearing: %03d°\r\nRange: %d m\r\nMIL: %d\r\n\r\nSTATUS: L81 TABLE (80–684 m)", s.bearingRounded(), s.rangeRounded(), s.milRounded())
}

func (s L81Solution) spokenSolution() string {
	if !s.HasMil {
		return fmt.Sprintf("Bearing %03d, Range %d, Mil unavailable.", s.bearingRounded(), s.rangeRounded())
	}
	return fmt.Sprintf("Bearing %03d, Range %d, Mil %d.", s.bearingRounded(), s.rangeRounded(), s.milRounded())
}

// Echo filtering is intentionally narrow: short operator input like X-ray,
// Yankee or digits must remain possible while Piper is speaking.
// Suppress only longer recognizer phrases that occur VERBATIM in Piper's output.
func likelyPiperEcho(heard, spoken string) bool {
	clean := func(s string) string {
		s = strings.ToLower(s)
		var b strings.Builder
		for _, r := range s {
			if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
				b.WriteRune(r)
			} else {
				b.WriteRune(' ')
			}
		}
		return strings.Join(strings.Fields(b.String()), " ")
	}
	h, p := clean(heard), clean(spoken)
	if len(strings.Fields(h)) < 4 || p == "" {
		return false
	}
	// Never discard short axis/value phrases, even when Piper happens to
	// be reading back the same value. Operator voice has priority.
	if !(strings.HasPrefix(h, "roger ") || strings.HasPrefix(h, "firing solution ") || strings.HasPrefix(h, "bearing ")) {
		return false
	}
	return h == p || strings.Contains(" "+p+" ", " "+h+" ")
}
