//go:build windows

package main

import (
	"fmt"
	"strings"
)

// Numeric CONFIRM belongs only to a live expected coordinate, never to idle,
// clear requests or unrelated speech. No extra mandatory confirmation step.
func (a *App) spokenCoordinateConfirm(s string) (string, bool) {
	if a.expectedCoordinateAxis() == "" {
		return "", false
	}
	body, ok := explicitConfirmBody(normaliseSpeech(s))
	if !ok {
		return "", false
	}
	return parseCoordinateValue(body)
}

func (a *App) recordAcceptedCoordinate(kind string, slot int, axis string) {
	if (kind != "actual" && kind != "switch_actual" && kind != "target") || (axis != "X" && axis != "Y") {
		return
	}
	a.lastCoordKind = kind
	a.lastCoordAxis = axis
	a.lastCoordSlot = slot
	a.keyboardStepOverride = 0
	a.keyboardErrorID = 0
}

// Reopen a confirmed coordinate without destroying the other coordinate or
// any saved target slots. Editing a confirmed ACTUAL pauses its old solution.
func (a *App) reopenLastConfirmedCoordinate() bool {
	axis, kind, slot := a.lastCoordAxis, a.lastCoordKind, a.lastCoordSlot
	if axis != "X" && axis != "Y" {
		return false
	}
	if kind == "actual" || kind == "switch_actual" {
		if !a.actualSet {
			return false
		}
		a.setPending("switch_actual", -1)
		a.pending.X, a.pending.Y = a.actualX, a.actualY
		a.pending.LastCoord = axis
	} else if kind == "target" {
		if !a.actualSet {
			return false
		}
		var t TargetSlot
		if slot == -2 && a.unsavedExists {
			t = a.unsaved
		} else if slot >= 0 && slot < 4 && a.targets[slot].Confirmed {
			t = a.targets[slot]
		} else {
			return false
		}
		a.activeTarget = slot // preserve/reselect the specific last edited target
		a.setPending("target", slot)
		a.pending.X, a.pending.Y = t.X, t.Y
		a.pending.LastCoord = axis
	} else {
		return false
	}
	a.pending.Replacing, a.pending.CueAwaiting = axis, axis
	a.touchPending()
	a.refreshOperationsUI()
	return true
}

// The single CORRECTION cue is recoverable even AFTER the auto-confirm on Y.
func (a *App) universalCorrection() {
	if a.pendingActive() {
		if a.pending.Kind != "actual" && a.pending.Kind != "switch_actual" && a.pending.Kind != "target" {
			a.addOperationLog("CORRECTION BLOCKED — finish or cancel the clear request.")
			return
		}
		axis := correctionAxis(a.pending.Replacing, a.pending.LastCoord, a.expectedCoordinateAxis())
		if axis == "" {
			a.addOperationLog("CORRECTION — no coordinate selected.")
			return
		}
		// The previously spoken X/Y might be wrong. Drop obsolete Piper output
		// before announcing the currently expected correction.
		a.abandonPiperReadbacks()
		// Do NOT erase the saved draft; it remains visible until replacement confirms.
		a.pending.Replacing, a.pending.CueAwaiting = axis, axis
		a.keyboardStepOverride = 0
		a.touchPending()
		a.addOperationLog("CORRECTION — repeat " + strings.ToUpper(a.pending.Kind) + " " + axis + ". Other coordinate preserved.")
		a.actualRespond("Roger. Repeat " + map[string]string{"X": "X-ray", "Y": "Yankee"}[axis] + ".")
		a.refreshOperationsUI()
		return
	}
	if a.reopenLastConfirmedCoordinate() {
		a.abandonPiperReadbacks()
		axis := a.pending.Replacing
		label := "Actual"
		if a.pending.Kind == "target" {
			label = "Target"
		}
		a.addOperationLog(fmt.Sprintf("CORRECTION — %s %s reopened AFTER confirmation. Previous coordinate protected.", strings.ToUpper(label), axis))
		a.actualRespond("Roger. Repeat " + label + " " + map[string]string{"X": "X-ray", "Y": "Yankee"}[axis] + ".")
		return
	}
	a.addOperationLog("Nothing to correct yet. Start Actual, or enter a coordinate.")
}

// Backspace rewinds the PROCEDURAL state, not merely keyboard focus. Existing
// accepted coordinates are preserved as a draft until a new Enter/voice value.
func (a *App) rewindCoordinateStep() {
	if a.pending.Kind == "clear_all" || a.pending.Kind == "clear_actual" {
		a.addOperationLog("Finish or cancel the CLEAR confirmation before rewinding.")
		return
	}
	// Keyboard rewind also makes all previous coordinate readbacks obsolete.
	a.abandonPiperReadbacks()
	if !a.pendingActive() {
		if a.reopenLastConfirmedCoordinate() {
			a.addOperationLog("BACKSPACE — previous accepted " + a.pending.Replacing + " reopened.")
			a.actualRespond("Roger. Repeat " + map[string]string{"X": "X-ray", "Y": "Yankee"}[a.pending.Replacing] + ".")
		} else if !a.actualSet {
			a.keyboardStepOverride = 1
			a.addOperationLog("BACKSPACE — STEP 1/6: Start Actual.")
			a.refreshOperationsUI()
		} else {
			a.keyboardStepOverride = 4
			a.addOperationLog("BACKSPACE — STEP 4/6: Start Target.")
			a.refreshOperationsUI()
		}
		return
	}
	kind := a.pending.Kind
	expected := a.expectedCoordinateAxis()
	switch {
	case expected == "Y":
		// Back to X; preserve X and any provisional Y, but require replacement X.
		a.pending.Replacing, a.pending.CueAwaiting = "X", "X"
		a.keyboardStepOverride = 0
		a.addOperationLog("BACKSPACE — " + strings.ToUpper(kind) + " X-RAY reopened (previous coordinates retained).")
	case expected == "X":
		// Rewind from X to its activation cue. Do not clear drafts or saved data.
		if kind == "target" {
			a.keyboardStepOverride = 4
		} else {
			a.keyboardStepOverride = 1
		}
		a.pending.CueAwaiting = ""
		a.pending.Replacing = "X" // Keep Vosk expecting X while rewound to activation.
		a.addOperationLog(fmt.Sprintf("BACKSPACE — STEP %d/6 START; press TAB for X-RAY.", a.keyboardStepOverride))
	default:
		a.pending.Replacing, a.pending.CueAwaiting = "Y", "Y"
		a.keyboardStepOverride = 0
		a.addOperationLog("BACKSPACE — " + strings.ToUpper(kind) + " YANKEE reopened.")
	}
	a.touchPending()
	a.refreshOperationsUI()
}

// TAB does not skip an unconfirmed coordinate. If the last Y committed then
// TAB selects the next target X; next field always agrees with Vosk's expectation.
func (a *App) nextCoordinateForTab() int {
	if !a.actualSet && !a.pendingActive() {
		a.keyboardStepOverride = 0
		return 5001
	}
	switch a.pending.Kind {
	case "actual", "switch_actual":
		if a.keyboardStepOverride == 1 {
			a.keyboardStepOverride = 0
		}
		if a.expectedCoordinateAxis() == "Y" {
			return 5002
		}
		return 5001
	case "target":
		if a.keyboardStepOverride == 4 {
			a.keyboardStepOverride = 0
		}
		if a.expectedCoordinateAxis() == "Y" {
			return 5012
		}
		return 5011
	}
	a.keyboardStepOverride = 0
	if a.actualSet {
		return 5011
	}
	return 5001
}
