package main

import "testing"

func TestOperationsDiagnosticsNeverLeaveVisibleClient(t *testing.T) {
	for h := 760; h <= 2400; h++ {
		d := operationDiagnosticsForHeight(h)
		if d.transcriptHeight > 0 && d.transcriptTop+d.transcriptHeight > h-16 {
			t.Fatalf("h=%d transcript exceeds viewport: %+v", h, d)
		}
		if d.liveHeight > 0 && d.liveTop+d.liveHeight > h-16 {
			t.Fatalf("h=%d live exceeds viewport: %+v", h, d)
		}
		if d.showLiveList && d.liveListY+d.liveListH > h-16 {
			t.Fatalf("h=%d list exceeds viewport: %+v", h, d)
		}
		if d.showTranscriptList && d.transcriptListY+d.transcriptListH > h-16 {
			t.Fatalf("h=%d transcript list exceeds viewport: %+v", h, d)
		}
	}
}
func TestOperationsDiagnosticAccessibility(t *testing.T) {
	for _, h := range []int{1010, 1100, 1280, 1440, 1700} {
		d := operationDiagnosticsForHeight(h)
		if !d.showLiveList {
			t.Fatalf("h=%d should have live list: %+v", h, d)
		}
		if d.liveListH < 36 {
			t.Fatalf("h=%d list height %+v", h, d)
		}
	}
}
