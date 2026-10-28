package main

import "testing"

func TestActivationRecognition(t *testing.T) {
	cases := []struct {
		heard, pending string
		actual         bool
		want           string
	}{
		{"use are actual", "", false, "user actual"},
		{"use are actual x nine eight point five five y one one zero point three seven over", "", false, "user actual x nine eight point five five y one one zero point three seven over"},
		{"new tar get x nine eight y one point three over", "", true, "new target x nine eight y one point three over"},
		{"usa actual", "", false, "user actual"},
		{"set user actual", "", false, "user actual"},
		{"new tar get", "", true, "new target"},
		{"new targets", "", true, "new target"},
		{"start new target", "", true, "new target"},
		{"new tar get", "actual", false, "new tar get"},
		{"new tar get", "target", true, "new tar get"},
		{"use are actual", "", true, "use are actual"},
		{"new tar get", "", false, "new tar get"},
	}
	for _, c := range cases {
		got, _ := normaliseActivationCommand(c.heard, c.pending, c.actual)
		if got != c.want {
			t.Errorf("%q, pending %q, actual %v: got %q want %q", c.heard, c.pending, c.actual, got, c.want)
		}
	}
}

func TestAxisArming(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"x", "X"}, {"y", "Y"}, {"98.55", ""}, {"new target", ""}, {"x 98.55", ""},
	} {
		if got := loneAxisCue(c.in); got != c.want {
			t.Errorf("cue %q: got %q want %q", c.in, got, c.want)
		}
	}
}

func TestConfidencePolicy(t *testing.T) {
	for _, c := range []struct {
		configured      int
		activation, cue bool
		want            int
	}{
		{80, false, false, 80}, {80, true, false, 65},
		{95, true, false, 80}, {60, true, false, 60},
		{80, false, true, 65},
	} {
		got := confidenceForVoiceCue(c.configured, c.activation, c.cue)
		if got != c.want {
			t.Errorf("confidence got %d want %d", got, c.want)
		}
	}
}

// G15 tests the actual phrases Vosk might output during each of the six steps.
// The GUI's event loop and microphone are tested separately on Windows.
func TestG15SixStepActivation(t *testing.T) {
	for _, c := range []struct {
		heard, want string
		yes         bool
	}{
		{"refresh actual", "refresh actual", true},
		{"ACTUAL", "refresh actual", true},
		{"ACTUALLY", "refresh actual", true},
		{"actaula", "refresh actual", true},
		{"use are actual", "refresh actual", true},
		{"user actual", "refresh actual", true},
		{"refresh target", "refresh target", true},
		{"new target", "refresh target", true},
		{"new tar get", "refresh target", true},
		{"actual x ray 98", "actual x ray 98", false},
		{"yank one two", "yank one two", false},
		{"clear all targets", "clear all targets", false},
		{"actually I was going to go", "actually i was going to go", false},
	} {
		got, ok := sixStepActivation(c.heard)
		if got != c.want || ok != c.yes {
			t.Errorf("%q got (%q,%v), want (%q,%v)", c.heard, got, ok, c.want, c.yes)
		}
	}
}

func TestG15ContextSixStages(t *testing.T) {
	for _, c := range []struct{ raw, kind, expected, want string }{
		{"actual x", "actual", "X", "x"},
		{"actual ex ray nine eight", "actual", "X", "x nine eight"},
		{"actual yank 110 point 37", "actual", "Y", "y 110 point 37"},
		{"ACTUAL YANKY 36 point 20", "actual", "Y", "y 36 point 20"},
		{"actual young key 34 point 60", "actual", "Y", "y 34 point 60"},
		{"actual yank e", "switch_actual", "Y", "y"},
		{"target xray", "target", "X", "x"},
		{"target axe ray 120", "target", "X", "x 120"},
		{"target yank one two point three", "target", "Y", "y one two point three"},
		{"refresh target yank key", "target", "Y", "y"},
		{"yank", "target", "X", "yank"},
		{"target yank", "actual", "Y", "target yank"},
		{"refresh actual", "actual", "Y", "refresh actual"},
	} {
		got := expectedSixStepCue(c.raw, c.kind, c.expected)
		if got != c.want {
			t.Errorf("%q, %s expecting %s -> %q (want %q)", c.raw, c.kind, c.expected, got, c.want)
		}
	}
}

func TestG15ContextConfidence(t *testing.T) {
	for _, c := range []struct {
		base  int
		stage string
		want  int
	}{
		{80, "activation", 40}, {80, "axis", 40}, {80, "value", 55},
		{80, "", 80}, {90, "", 90}, {30, "activation", 30},
		{45, "value", 45}, {0, "activation", 40},
	} {
		if got := sixStepThreshold(c.base, c.stage); got != c.want {
			t.Errorf("base=%d,stage=%s got=%d want=%d", c.base, c.stage, got, c.want)
		}
	}
}
