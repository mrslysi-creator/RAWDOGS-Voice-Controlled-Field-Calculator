package main

import (
	"strings"
	"testing"
)

func TestSixStepAcknowledgements(t *testing.T) {
	tests := []struct{ stage, kind, axis, value, want string }{
		{"activate", "actual", "", "", "Actual X-ray next."},
		{"axis", "actual", "X", "", ""},
		{"coordinate", "actual", "X", "97 point 86", "Actual Yankee next."},
		{"committed", "actual", "Y", "34 point 60", "Actual confirmed. New target next."},
		{"activate", "target", "", "", "Target X-ray next."},
		{"axis", "target", "Y", "", ""},
		{"coordinate", "target", "X", "98 point 34", "Target Yankee next."},
		{"committed", "target", "Y", "35 point 10", ""},
	}
	for _, tt := range tests {
		got := sixStepAcknowledgement(tt.stage, tt.kind, tt.axis, tt.value)
		if got != tt.want {
			t.Errorf("%q %q %q: got %q, want %q", tt.stage, tt.kind, tt.axis, got, tt.want)
		}
		if strings.Contains(strings.ToLower(got), "roger") || strings.Contains(strings.ToLower(got), "over") {
			t.Errorf("unnecessary speech in %q", got)
		}
	}
}
