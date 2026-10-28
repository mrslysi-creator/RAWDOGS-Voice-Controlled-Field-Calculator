package main

import "testing"

func TestG18CRecallAliases(t *testing.T) {
	tests := []struct {
		heard, want string
		changed     bool
	}{
		{"recall target one", "recall target one", false},
		{"recal target one", "recall target one", true},
		{"RECAL TARGET 1", "recall target 1", true},
		{"we call target one", "recall target one", true},
		{"we target one", "recall target one", true},
		{"regal target one", "recall target one", true},
		{"re call target 2", "recall target 2", true},
		{"we call target four", "recall target four", true},
		{"regal target too", "recall target too", true},
		{"revert target one", "recall target one", true},
		{"river target 2", "recall target 2", true},
		{"with thier target three", "recall target three", true},
		{"with their target four", "recall target four", true},
		{"we'at target one", "recall target one", true},
		{"we at target one", "recall target one", true},
		{"re-call target three", "recall target three", true},
		// Don't guess missing target number, incidental speech or CLEAR.
		{"regal target", "regal target", false},
		{"we call target five", "we call target five", false},
		{"we call target one please", "we call target one please", false},
		{"we target is one", "we target is one", false},
		{"target one", "recall target one", true}, // G18F: user-approved bare numbered recall
		{"we call the target one", "we call the target one", false},
		{"clear target one", "clear target one", false},
		{"yankee three", "yankee three", false},
		{"correction", "correction", false},
		{"use target three", "use target three", false},
		{"bull target one", "bull target one", false},
		{"river target", "river target", false},
		{"we target", "we target", false},
		{"revert target ten", "revert target ten", false},
		{"river target 3 please", "river target 3 please", false},
	}
	for _, tc := range tests {
		got, changed := normaliseTargetRecallAlias(tc.heard)
		if got != tc.want || changed != tc.changed {
			t.Errorf("%q => (%q,%v), want (%q,%v)", tc.heard, got, changed, tc.want, tc.changed)
		}
	}
}
