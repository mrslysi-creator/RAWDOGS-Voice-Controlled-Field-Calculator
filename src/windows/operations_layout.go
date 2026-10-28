package main

// All dimensions are native Win32 client-area coordinates. The original
// upper Operations cards stay in their G13/G17 registration positions; the
// transcript and LIVE RECOGNITION share the remaining *visible* height.
type opsDiagnosticsLayout struct {
	transcriptTop, transcriptHeight, transcriptTitleY, transcriptListY, transcriptListH   int
	liveTop, liveHeight, liveTitleY, liveSummaryY, liveSummaryH, liveListY, liveListH     int
	showTranscriptTitle, showTranscriptList, showLiveTitle, showLiveSummary, showLiveList bool
}

func operationDiagnosticsForHeight(clientH int) opsDiagnosticsLayout {
	const top, bottomPad, gap = 742, 16, 12
	var d opsDiagnosticsLayout
	usable := clientH - top - bottomPad
	if usable <= 0 {
		return d
	}
	d.transcriptTop = top
	switch {
	case usable >= 300:
		d.transcriptHeight = usable * 38 / 100
		if d.transcriptHeight < 112 {
			d.transcriptHeight = 112
		}
		if d.transcriptHeight > 214 {
			d.transcriptHeight = 214
		}
	case usable >= 260:
		d.transcriptHeight = 85
	case usable >= 190:
		d.transcriptHeight = 42
	case usable >= 135:
		d.transcriptHeight = 0 // compact: prioritise live recognition over transcript
	default:
		d.transcriptHeight = 0 // only the diagnosis summary fits
	}
	if d.transcriptHeight > 0 {
		d.transcriptTitleY = top + 9
		d.showTranscriptTitle = true
		if d.transcriptHeight >= 85 {
			d.transcriptListY = top + 34
			d.transcriptListH = d.transcriptHeight - 44
			d.showTranscriptList = d.transcriptListH >= 36
		}
		d.liveTop = top + d.transcriptHeight + gap
	} else {
		d.liveTop = top
	}
	d.liveHeight = clientH - bottomPad - d.liveTop
	if d.liveHeight <= 0 {
		return d
	}
	d.liveTitleY = d.liveTop + 10
	d.showLiveTitle = d.liveHeight >= 35
	d.liveSummaryY = d.liveTop + 35
	d.liveSummaryH = 28
	d.showLiveSummary = d.liveHeight >= 72
	d.liveListY = d.liveTop + 72
	d.liveListH = d.liveHeight - 82
	d.showLiveList = d.liveListH >= 36
	if !d.showLiveList {
		d.liveListH = 0
	}
	return d
}
