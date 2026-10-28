package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func newVoiceTestProfile() voiceProfile {
	return voiceProfile{Version: 1, Operator: "CORNWALL", Model: "vosk-en-gb"}
}
func TestVoiceTrainingHearsFreeAsThree(t *testing.T) {
	equal, c, why := voiceCompareExercise("yankee three five point seven one", "yankee free five point seven one")
	if equal || c == nil || c.Heard != "free" || c.Intended != "three" || c.Scope != "coordinate_digit" {
		t.Fatalf("candidate=%+v equal=%t why=%s", c, equal, why)
	}
	p := newVoiceTestProfile()
	if count, err := p.confirmCandidate(*c, time.Now()); err != nil || count != 1 {
		t.Fatalf("first confirm: %d %v", count, err)
	}
	if got, _ := p.adaptCoordinate("yankee free five point seven one", "Y", false); got != "yankee free five point seven one" {
		t.Fatalf("premature adaptation %q", got)
	}
	if count, err := p.confirmCandidate(*c, time.Now()); err != nil || count != 2 {
		t.Fatalf("second confirm: %d %v", count, err)
	}
	got, why := p.adaptCoordinate("yankee free five point seven one", "Y", false)
	if got != "y three five point seven one" || !strings.Contains(why, "free -> three") {
		t.Fatalf("got %q, %q", got, why)
	}
}
func TestVoiceTrainingLearnsCornishTheInStrictCoordinate(t *testing.T) {
	eq, c, _ := voiceCompareExercise("yankee three five point seven one", "yankee the five point seven one")
	if eq || c == nil || c.Heard != "the" {
		t.Fatalf("expected the->three candidate: %+v", c)
	}
	p := newVoiceTestProfile()
	for i := 0; i < 2; i++ {
		if _, e := p.confirmCandidate(*c, time.Now()); e != nil {
			t.Fatal(e)
		}
	}
	got, _ := p.adaptCoordinate("target yankee the five point seven one", "Y", false)
	if got != "target y three five point seven one" {
		t.Fatalf("got %q", got)
	}
	// A split YANKEE -> value command is safe ONLY when the axis cue
	// has already been explicitly armed in the six-step state machine.
	got, _ = p.adaptCoordinate("the five point seven one", "Y", true)
	if got != "three five point seven one" {
		t.Fatalf("armed split coordinate: %q", got)
	}
	for _, raw := range []string{"the five point seven one", "it is the five point seven one", "yankee the"} {
		got, _ = p.adaptCoordinate(raw, "Y", false)
		if got != raw {
			t.Fatalf("unarmed/ambiguous %q adapted to %q", raw, got)
		}
	}
	got, _ = p.adaptCoordinate("yankee the five point seven one", "X", false)
	if got != "yankee the five point seven one" {
		t.Fatalf("incorrect-axis conversion %q", got)
	}
}
func TestVoiceNoMissingDigitsOrWrongDigitMapping(t *testing.T) {
	cases := []string{
		"yankee four point six zero",       // omitted "three"
		"yankee two five point seven one",  // a different valid digit
		"yankee the five point seven",      // missing one
		"yankee the six point seven one",   // multiple mismatch
		"yankee five point seven one free", // insertion and reordering
	}
	for _, raw := range cases {
		eq, c, _ := voiceCompareExercise("yankee three five point seven one", raw)
		if eq || c != nil {
			t.Errorf("unsafe candidate %+v for %q", c, raw)
		}
	}
}
func TestVoiceTrainingLearnsTwoWordYankeeCue(t *testing.T) {
	eq, c, why := voiceCompareExercise("yankee three five point seven one", "young key three five point seven one")
	if eq || c == nil || c.Heard != "young key" || c.Intended != "y" || c.Scope != "axis_y" {
		t.Fatalf("cue candidate %+v; reason %s", c, why)
	}
	p := newVoiceTestProfile()
	for i := 0; i < 2; i++ {
		if _, e := p.confirmCandidate(*c, time.Now()); e != nil {
			t.Fatal(e)
		}
	}
	got, why := p.adaptCoordinate("young key three five point seven one", "Y", false)
	if got != "y three five point seven one" || why == "" {
		t.Fatalf("cue adaptation %q %s", got, why)
	}
	for _, raw := range []string{"young key", "young key three", "that young key three five point seven one"} {
		got, _ = p.adaptCoordinate(raw, "Y", false)
		if got != raw {
			t.Errorf("cue alias used unsafely on %q -> %q", raw, got)
		}
	}
	got, _ = p.adaptCoordinate("young key three five point seven one", "X", false)
	if got != "young key three five point seven one" {
		t.Fatal("cue applied to unexpected axis")
	}
}
func TestVoiceProfilePersistenceAndModelIsolation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "p.json")
	p := newVoiceTestProfile()
	c := voiceCandidate{Heard: "the", Intended: "three", Scope: "coordinate_digit"}
	for i := 0; i < 2; i++ {
		if _, err := p.confirmCandidate(c, time.Now()); err != nil {
			t.Fatal(err)
		}
	}
	if err := voiceProfileSave(path, p); err != nil {
		t.Fatal(err)
	}
	got, err := voiceProfileLoad(path, "CORNWALL", "vosk-en-gb")
	if err != nil || len(got.Aliases) != 1 || got.Aliases[0].Confirmations != 2 {
		t.Fatalf("loaded %+v err %v", got, err)
	}
	if _, err = voiceProfileLoad(path, "SCOTLAND", "vosk-en-gb"); err == nil {
		t.Fatal("cross-operator profile accepted")
	}
	if _, err = voiceProfileLoad(path, "CORNWALL", "vosk-en-us"); err == nil {
		t.Fatal("cross-model profile accepted")
	}
	if _, err = os.Stat(path); err != nil {
		t.Fatal(err)
	}
}
func TestTamperedProfileCannotSubstituteCommandsOrDigits(t *testing.T) {
	p := newVoiceTestProfile()
	p.Aliases = []voiceAlias{
		{Heard: "confirm", Intended: "three", Scope: "coordinate_digit", Confirmations: 2},
		{Heard: "four", Intended: "three", Scope: "coordinate_digit", Confirmations: 2},
		{Heard: "the", Intended: "three", Scope: "coordinate_digit", Confirmations: 2},
		{Heard: "stop training", Intended: "y", Scope: "axis_y", Confirmations: 2},
	}
	path := filepath.Join(t.TempDir(), "unsafe.json")
	if err := voiceProfileSave(path, p); err != nil {
		t.Fatal(err)
	}
	loaded, err := voiceProfileLoad(path, p.Operator, p.Model)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Aliases) != 1 || loaded.Aliases[0].Heard != "the" {
		t.Fatalf("unsafe rules retained %+v", loaded.Aliases)
	}
}
func TestVoiceExerciseExactAndCollisionSafety(t *testing.T) {
	eq, c, _ := voiceCompareExercise("x ray three six point zero two", "x ray three six point zero two")
	if !eq || c != nil {
		t.Fatalf("exact %t %+v", eq, c)
	}
	p := newVoiceTestProfile()
	if _, err := p.confirmCandidate(voiceCandidate{Heard: "four", Intended: "three", Scope: "coordinate_digit"}, time.Now()); err == nil {
		t.Fatal("wrong numeric remap allowed")
	}
	if _, err := p.confirmCandidate(voiceCandidate{Heard: "the", Intended: "three", Scope: "coordinate_digit"}, time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := p.confirmCandidate(voiceCandidate{Heard: "the", Intended: "four", Scope: "coordinate_digit"}, time.Now()); err == nil {
		t.Fatal("conflict allowed")
	}
}
