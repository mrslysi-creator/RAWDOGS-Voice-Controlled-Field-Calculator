package main

import (
	"strings"
	"time"
)

// The written Field Manual is the ONLY source of narration text. The caller
// supplies exactly the visible instructional blocks in their reading order.
// Only whitespace is collapsed for Piper: no rewritten or omitted words.
func manualNarrationFromVisible(blocks []string) []string {
	out := make([]string, 0, len(blocks))
	for _, printed := range blocks {
		spoken := strings.Join(strings.Fields(printed), " ")
		if spoken != "" {
			out = append(out, spoken)
		}
	}
	return out
}

// Map an exact WAV-time fraction to its displayed passage; never guess
// based on number of characters or on the number of sections.
func manualSeekPosition(durations []time.Duration, position int) (int, time.Duration) {
	if len(durations) == 0 {
		return 0, 0
	}
	if position < 0 {
		position = 0
	}
	if position > 10000 {
		position = 10000
	}
	var total time.Duration
	for _, d := range durations {
		total += d
	}
	if total <= 0 {
		return 0, 0
	}
	remaining := time.Duration(int64(total) * int64(position) / 10000)
	for i, d := range durations {
		if remaining < d || i == len(durations)-1 {
			if remaining > d {
				remaining = d
			}
			return i, remaining
		}
		remaining -= d
	}
	return 0, 0
}
