package main

import (
	"testing"
	"time"
)

func TestCueAttemptOnlyWithExplicitCoordinateIntent(t *testing.T) {
	for _, c := range []struct {
		phrase      string
		armed, want bool
	}{
		{"x nine eight point seven forty", false, true},
		{"y 104.52", false, true},
		{"y", false, true},
		{"seven forty", true, true},
		{"98.44", true, true},
		{"seven forty", false, false},
		{"i was talking about the game", true, false},
		{"x is where the action is", false, false}, // game chat is never a coordinate attempt
		{"repeat solution", true, false},
		{"clear all targets", true, false},
		{"my teammates are talking", false, false},
	} {
		if got := cueAttempt(c.phrase, c.armed); got != c.want {
			t.Errorf("phrase %q armed %v got %v expected %v", c.phrase, c.armed, got, c.want)
		}
	}
}

func TestCueCoachStageConfidenceAndCooldown(t *testing.T) {
	var c cueCoach
	start := time.Date(2026, 10, 25, 19, 0, 0, 0, time.UTC)
	if c.note("actual:X", true, false, 90, start) {
		t.Fatal("should not prompt after one miss")
	}
	if !c.note("actual:X", true, false, 90, start.Add(2*time.Second)) {
		t.Fatal("two clear misses should prompt")
	}
	if c.note("actual:X", true, false, 90, start.Add(3*time.Second)) {
		t.Fatal("cooldown expected")
	}
	if c.note("actual:X", true, false, 90, start.Add(4*time.Second)) {
		t.Fatal("cooldown expected")
	}
	if c.note("actual:X", true, true, 95, start.Add(5*time.Second)) {
		t.Fatal("success must clear failures")
	}
	if c.note("actual:Y", true, false, 90, start.Add(6*time.Second)) {
		t.Fatal("new axis must start over")
	}
	if !c.note("actual:Y", true, false, 90, start.Add(32*time.Second)) {
		t.Fatal("new axis after cooldown should prompt")
	}
}
func TestCueCoachChatDoesNotNudge(t *testing.T) {
	var c cueCoach
	start := time.Date(2026, 10, 25, 19, 0, 0, 0, time.UTC)
	if c.note("target:Y", true, false, 85, start) {
		t.Fatal("only one miss")
	}
	if c.note("target:Y", false, false, 90, start.Add(time.Second)) {
		t.Fatal("chat must not prompt")
	}
	if c.note("target:Y", true, false, 90, start.Add(2*time.Second)) {
		t.Fatal("interrupted streak should reset")
	}
	if c.note("target:Y", true, false, 50, start.Add(3*time.Second)) {
		t.Fatal("low-confidence audio must not prompt")
	}
	if c.note("target:Y", true, false, 90, start.Add(60*time.Second)) {
		t.Fatal("old attempts expire")
	}
}
func TestCueCoachPrompt(t *testing.T) {
	for _, c := range []struct {
		kind, axis string
		armed      bool
		want       string
	}{
		{"actual", "X", false, "Actual X-ray required. Say X-ray, then your X coordinate."},
		{"target", "Y", false, "Target Yankee required. Say Yankee, then your Y coordinate."},
		{"target", "X", true, "Target X coordinate still needed. Say the digits clearly, one at a time after point."},
	} {
		if got := cueCoachPrompt(c.kind, c.axis, c.armed); got != c.want {
			t.Fatalf("got %q want %q", got, c.want)
		}
	}
}

func TestComboGuideHeightNotDropdownHeight(t *testing.T) {
	for _, h := range []int{160, 220, 260, 300} {
		if got := guideVisibleHeight(true, h); got != 30 {
			t.Fatalf("combo popup height %d produced outline %d", h, got)
		}
	}
	if got := guideVisibleHeight(false, 34); got != 34 {
		t.Fatalf("button cropped: %d", got)
	}
}
