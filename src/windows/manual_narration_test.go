package main

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestManualNarrationExactlyMatchesPrintedWords(t *testing.T) {
	shown := []string{"SET ACTUAL POSITION.", "Say REFRESH ACTUAL.\r\nSay X-RAY, then YANKEE.", "No OVER or extra finishing cue is needed."}
	got := manualNarrationFromVisible(shown)
	want := []string{"SET ACTUAL POSITION.", "Say REFRESH ACTUAL. Say X-RAY, then YANKEE.", "No OVER or extra finishing cue is needed."}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("speech must contain exactly the printed words: %q", got)
	}
	got[0] = "changed"
	if strings.HasPrefix(shown[0], "changed") {
		t.Fatal("speech may not mutate paper")
	}
}

func TestManualSeekFollowsActualAudioLengths(t *testing.T) {
	durations := []time.Duration{2 * time.Second, 6 * time.Second, 2 * time.Second}
	cases := []struct {
		pos, index int
		offset     time.Duration
	}{{0, 0, 0}, {1000, 0, time.Second}, {4000, 1, 2 * time.Second}, {9000, 2, time.Second}, {10000, 2, 2 * time.Second}}
	for _, tc := range cases {
		i, offset := manualSeekPosition(durations, tc.pos)
		if i != tc.index || offset != tc.offset {
			t.Errorf("at %d expected passage %d / %v, got %d / %v", tc.pos, tc.index, tc.offset, i, offset)
		}
	}
}
