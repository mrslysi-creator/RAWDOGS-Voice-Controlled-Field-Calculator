package main

import "testing"

func TestMarkedHeaderRegionsAtTypicalWidths(t *testing.T) {
	for _, width := range []int{1100, 1200, 1280, 1390, 1400, 1484, 1640, 1920} {
		g := voiceGuideHeaderLayoutForWidth(width)
		if g.TitleW >= g.TeamW {
			t.Errorf("width %d: title should not exceed team banner", width)
		}
		if g.TitleX+g.TitleW >= g.ButtonX0 {
			t.Errorf("width %d: title overlaps Walkman", width)
		}
		if g.GuideRight()+12 > g.SupportX {
			t.Errorf("width %d: Walkman overlaps coffee", width)
		}
		if g.SupportX+g.SupportW+12 > g.TeamX {
			t.Errorf("width %d: coffee overlaps team", width)
		}
		if g.TeamX+g.TeamW > width-15 {
			t.Errorf("width %d: team extends off right", width)
		}
		if g.SupportW < 150 || g.SupportW > 230 {
			t.Errorf("width %d: coffee width %d", width, g.SupportW)
		}
		if g.ButtonY < 10 || g.ButtonY+g.ButtonH > 83 {
			t.Errorf("width %d: buttons overflow top header", width)
		}
		if g.ButtonH < 64 || g.ButtonW < 39 {
			t.Errorf("width %d: buttons insufficient size: %dx%d", width, g.ButtonW, g.ButtonH)
		}
		for i := 0; i < 5; i++ {
			if g.ButtonX(i)+g.ButtonW+g.Gap != g.ButtonX(i+1) {
				t.Errorf("width %d: button %d spacing", width, i)
			}
		}
		if g.ButtonX(5)+g.ButtonW != g.GuideRight() {
			t.Errorf("width %d: buttons do not fill Walkman area", width)
		}
	}
}
