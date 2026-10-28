package main

// Shared, testable layout geometry for Voice Training.
// The protected top area is unchanged by this calculation.

type trainingBox struct{ x, y, w, h int }
type voiceTrainingBoxes struct{ leftTop, leftBottom, rightTop, rightBottom, footer trainingBox }

func trainingBoxes(w, h int) voiceTrainingBoxes {
	if w < 850 {
		w = 850
	}
	if h < 680 {
		h = 680
	}
	x := 20
	gap := 18
	usable := w - 40 - gap
	leftW := usable * 54 / 100
	rightX := x + leftW + gap
	rightW := w - rightX - 20
	// Make the current phrase the principal training element without touching
	// any of the locked top controls (y<=230). Panels grow with the window and
	// compress proportionally at a short window height.
	topH := h / 4
	if topH < 185 {
		topH = 185
	}
	if topH > 255 {
		topH = 255
	}
	bottomY := 330 + topH + 8
	footY := h - 52
	if footY < 628 {
		footY = 628
	}
	bottomH := footY - bottomY - 7
	if bottomH < 55 {
		bottomH = 55
	}
	return voiceTrainingBoxes{
		leftTop:     trainingBox{x, 330, leftW, topH},
		leftBottom:  trainingBox{x, bottomY, leftW, bottomH},
		rightTop:    trainingBox{rightX, 330, rightW, topH},
		rightBottom: trainingBox{rightX, bottomY, rightW, bottomH},
		footer:      trainingBox{20, footY + 4, w - 40, 42},
	}
}
