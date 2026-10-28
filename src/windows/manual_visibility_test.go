package main

import "testing"

func TestManualPageControlsVisible_OnlyOnFieldManualTab(t *testing.T) {
	cases := []struct {
		name, screen, selected, control string
		want                            bool
	}{
		{"save on Settings cannot reveal narrated guide", "settings", "narrated", "narrated", false},
		{"saving again after page four selected", "settings", "narrated", "quick", false},
		{"operations cannot show narrated guide", "operations", "narrated", "narrated", false},
		{"voice training cannot show page", "training", "narrated", "narrated", false},
		{"startup cannot show page", "", "quick", "quick", false},
		{"narrated page visible when opened", "manual", "narrated", "narrated", true},
		{"other pages hidden in narrated mode", "manual", "narrated", "quick", false},
		{"quick reference visible when opened", "manual", "quick", "quick", true},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := manualPageControlsVisible(tt.screen, tt.selected, tt.control)
			if got != tt.want {
				t.Fatalf("visible(%q, %q, %q) = %t, want %t", tt.screen, tt.selected, tt.control, got, tt.want)
			}
		})
	}
}
