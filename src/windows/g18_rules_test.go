package main

import "testing"

func TestG18ExplicitConfirmSuffix(t *testing.T) {
	for _, tc := range []struct {
		heard, want string
		valid       bool
	}{
		{"three two one confirm", "three two one", true},
		{"THREE TWO ONE CONFIRM", "three two one", true},
		{"97 point 86 confirm", "97 point 86", true},
		{"three two one", "", false},
		{"confirm", "", false},
		{"confirmed", "", false},
		{"clear all targets", "", false},
		{"yank three confirm", "yank three", true},
	} {
		got, ok := explicitConfirmBody(tc.heard)
		if got != tc.want || ok != tc.valid {
			t.Errorf("%q => (%q,%v), want (%q,%v)", tc.heard, got, ok, tc.want, tc.valid)
		}
	}
}

func TestG18CorrectionAxisPriority(t *testing.T) {
	for _, tc := range []struct{ replace, last, expect, want string }{
		{"", "Y", "", "Y"},   // after ACTUAL Y auto-confirm
		{"", "X", "Y", "X"},  // X accepted but Y not yet entered
		{"Y", "X", "X", "Y"}, // repeated correction stays on selected axis
		{"", "", "X", "X"},   // no coordinate accepted yet
		{"", "", "", ""},
	} {
		got := correctionAxis(tc.replace, tc.last, tc.expect)
		if got != tc.want {
			t.Errorf("correctionAxis(%q,%q,%q)=%q want %q", tc.replace, tc.last, tc.expect, got, tc.want)
		}
	}
}
