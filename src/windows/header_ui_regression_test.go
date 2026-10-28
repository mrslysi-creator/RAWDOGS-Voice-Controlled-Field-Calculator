package main

import (
	"os"
	"strings"
	"testing"
)

func TestHeaderHasSixCleanWalkmanButtonsAndNoDuplicateEnd(t *testing.T) {
	s, err := os.ReadFile("source_main_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	ui := string(s)
	for _, q := range []string{`"V.C.F.C."`, `a.addScreenButton(s, ID_GUIDE_END`, `"END GUIDE"`, `for i, id := range []int{ID_TOUR_PREV, ID_TOUR_PLAY, ID_TOUR_PAUSE,`} {
		if !strings.Contains(ui, q) {
			t.Errorf("missing expected header/guide content: %q", q)
		}
	}
	if strings.Contains(ui, `a.addButton(ID_TOUR_END,`) {
		t.Fatal("duplicate header END control is still being created")
	}
	if strings.Contains(ui, `"END", "nav"`) {
		t.Fatal("a duplicate END nav button remains")
	}
	if strings.Contains(ui, `"V.C.F.C. — v`) {
		t.Fatal("version inserted into header name")
	}
}
