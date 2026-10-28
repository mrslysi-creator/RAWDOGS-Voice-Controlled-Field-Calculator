package main

import "testing"

func TestG18FGlobalTargetRecallExactness(t *testing.T) {
	good := map[string]int{
		"target one": 0, "TARGET TWO": 1, "target three": 2, "target 4": 3,
		"recall target 1": 0, "use target four": 3,
	}
	for in, want := range good {
		got, ok := parseRecallTargetPhrase(in)
		if !ok || got != want {
			t.Errorf("%q got %d ok %v want %d", in, got, ok, want)
		}
	}
	bad := []string{
		"target", "target zero", "target 5", "target one please", "target x ray nine",
		"new target", "clear target one", "recall target", "target one more", "repeat target two",
	}
	for _, in := range bad {
		if slot, ok := parseRecallTargetPhrase(in); ok {
			t.Errorf("incorrect global recall for %q -> %d", in, slot)
		}
	}
}

func TestG18FEarlyPiperBargeInRequiresCompleteCue(t *testing.T) {
	accepted := []string{"new target", "refresh target", "correction", "recall target two", "target one", "we call target three", "piper stop"}
	for _, in := range accepted {
		if !likelyBargeInPartial(in) {
			t.Errorf("should preempt only old speech for complete cue %q", in)
		}
	}
	rejected := []string{"new", "targ", "target", "yankee", "x ray nine", "clear all targets", "hello", "target six", "we call target", "one"}
	for _, in := range rejected {
		if likelyBargeInPartial(in) {
			t.Errorf("unsafe partial should not preempt %q", in)
		}
	}
}
