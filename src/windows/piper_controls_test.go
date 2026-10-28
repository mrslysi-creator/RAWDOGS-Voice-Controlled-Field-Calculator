package main

import (
	"math"
	"strings"
	"testing"
)

func TestPiperCommandParsing(t *testing.T) {
	good := []struct {
		input, kind string
		speed       float64
		volume      int
	}{
		{"PIPER STOP", "stop", 0, 0}, {"PIPER MUTE", "mute", 0, 0}, {"piper unmute", "unmute", 0, 0},
		{"piper faster", "faster", 0, 0}, {"piper slower", "slower", 0, 0},
		{"piper louder", "louder", 0, 0}, {"piper quieter", "quieter", 0, 0},
		{"piper speed one point eight", "speed", 1.8, 0},
		{"piper speed 1.90", "speed", 1.9, 0},
		{"piper speed two point zero", "speed", 2.0, 0},
		{"piper volume seventy", "volume", 0, 70},
		{"piper volume one hundred", "volume", 0, 100},
		{"piper volume 50", "volume", 0, 50},
	}
	for _, x := range good {
		got, ok := parsePiperCommand(x.input)
		if !ok || got.Kind != x.kind || math.Abs(got.Speed-x.speed) > 0.001 || got.Volume != x.volume {
			t.Errorf("%q -> %+v %v", x.input, got, ok)
		}
	}
	for _, s := range []string{"STOP", "Faster", "target one", "yankee three four", "piper speed 1.75", "piper volume 73", "piper volume banana", "piper stop listening"} {
		if _, ok := parsePiperCommand(s); ok {
			t.Errorf("unsafe accepted %q", s)
		}
	}
}
func TestPiperSpeedSteps(t *testing.T) {
	if stepPiperSpeed(1.5, 1) != 1.6 || stepPiperSpeed(1.9, 1) != 2.0 || stepPiperSpeed(2.0, 1) != 2.0 || stepPiperSpeed(1.6, -1) != 1.5 {
		t.Fatal("speed steps incorrect")
	}
	if len(piperSpeeds) != len(piperSpeedOptions) {
		t.Fatal("speed control diverged")
	}
	if len(piperVolumeOptions) != 11 {
		t.Fatal("volume dropdown mismatched")
	}
}
func TestTacticalPiperFilter(t *testing.T) {
	for _, s := range []string{"Roger. Actual ready. X-ray next.", "Roger. X-ray 97 understood. Yankee next.", "Target 1 confirmed.", "Y 35. Coordinates complete."} {
		if !routinePiperConfirmation(s) {
			t.Errorf("routine not filtered: %q", s)
		}
	}
	for _, s := range []string{"Roger. Repeat Actual Yankee.", "Bearing 044, Range 130, Mil 855.", "Solution unavailable.", "Incorrect coordinate"} {
		if routinePiperConfirmation(s) {
			t.Errorf("important filtered: %q", s)
		}
	}
	if !strings.Contains("Bearing 044, Range 130, Mil 855.", "Range") {
		t.Fatal("test corrupted")
	}
}
