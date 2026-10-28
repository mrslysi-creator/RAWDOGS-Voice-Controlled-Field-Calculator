package main

// The header is divided into four independent regions:
// compact RAWDOGS title, six shared Walkman controls, optional support button,
// and the existing team banner. At the operator's request, there is NO
// extra GUIDE: READY title/panel above the Walkman icons.
type voiceGuideLayout struct {
	TitleX, TitleW                           int
	ButtonX0, ButtonY, ButtonW, ButtonH, Gap int
	SupportX, SupportW                       int
	TeamX, TeamW                             int
}

func clampHeader(v, low, high int) int {
	if v < low {
		return low
	}
	if v > high {
		return high
	}
	return v
}

func voiceGuideHeaderLayoutForWidth(width int) voiceGuideLayout {
	if width < 1100 {
		width = 1100
	}
	const left, titleGap, right = 16, 20, 20
	titleW := clampHeader(width*18/100, 220, 290)
	guideW := clampHeader(width*25/100, 330, 410)
	teamW := clampHeader(width*21/100, 275, 340)
	buttonX := left + titleW + titleGap
	teamX := width - right - teamW

	// The owner's orange annotation is a compact support button, not a huge
	// gold strip. Centre it in the remaining camo header with a small bias
	// toward the team banner. Keep clear gaps at the minimum window width.
	supportW := clampHeader(width*14/100, 170, 230)
	empty := teamX - (buttonX + guideW)
	if supportW > empty-24 {
		supportW = empty - 24
	}
	supportX := buttonX + guideW + (empty-supportW)*57/100

	const gap = 5
	buttonW := (guideW - 5*gap) / 6
	return voiceGuideLayout{
		TitleX: left, TitleW: titleW,
		ButtonX0: buttonX, ButtonY: 12, ButtonW: buttonW, ButtonH: 70,
		Gap:      gap,
		SupportX: supportX, SupportW: supportW,
		TeamX: teamX, TeamW: teamW,
	}
}
func (g voiceGuideLayout) ButtonX(index int) int { return g.ButtonX0 + index*(g.ButtonW+g.Gap) }
func (g voiceGuideLayout) GuideRight() int       { return g.ButtonX(5) + g.ButtonW }
