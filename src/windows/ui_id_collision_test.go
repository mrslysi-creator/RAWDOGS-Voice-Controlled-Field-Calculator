package main

import (
	"os"
	"regexp"
	"strconv"
	"testing"
)

// The Field Manual owns control IDs 6200..6207. Version 2.43.17
// accidentally reused them for Voice Training commands, causing tab
// control relocation, ghost overlays and broken narrated-guide scrolling.
// This regression test runs even when building on Linux.
func TestVoiceTrainingAndManualControlIDsDoNotOverlap(t *testing.T) {
	data, err := os.ReadFile("voice_adaptation_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`ID_ADAPT_START\s*=\s*(\d+)\s*\+\s*iota`).FindSubmatch(data)
	if m == nil {
		t.Fatal("cannot determine Voice Training button ID base")
	}
	start, err := strconv.Atoi(string(m[1]))
	if err != nil {
		t.Fatal(err)
	}
	for id := start; id < start+8; id++ {
		if id >= 6200 && id <= 6207 {
			t.Fatalf("Voice Training button ID %d collides with Field Manual control", id)
		}
	}
}
