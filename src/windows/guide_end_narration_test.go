package main

import (
	"os"
	"strings"
	"testing"
)

func TestNarrationReferencesOriginalSettingsEndGuideOnly(t *testing.T) {
	source, err := os.ReadFile("guided_tour_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	s := string(source)
	if strings.Contains(s, "Press End Guide above") || strings.Contains(s, "End Guide above") {
		t.Fatal("narrator still points at removed header End button")
	}
	if !strings.Contains(s, `scene("87 — Finish and operate normally", "settings", []int{ID_GUIDE_END}`) {
		t.Fatal("final spoken lesson must navigate to Settings and spotlight its real End Guide button")
	}
	if !strings.Contains(s, "original End Guide button") {
		t.Fatal("Walkman lesson needs to identify existing Settings End Guide")
	}
	script, err := os.ReadFile("GUIDE_INDEX_AND_SCRIPT_2_43_19.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(script), "The End Guide button is here in Settings") {
		t.Fatal("reading copy must match the final spoken lesson")
	}
}
