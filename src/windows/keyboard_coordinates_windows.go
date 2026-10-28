//go:build windows

package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

// G17: Only the four existing X/Y display rectangles are editable. Tab/Enter
// work without disturbing existing tabs/buttons or the six-step voice state.
func (a *App) isCoordinateID(id int) bool {
	return id == 5001 || id == 5002 || id == 5011 || id == 5012
}
func coordinateIDs() [4]int { return [4]int{5001, 5002, 5011, 5012} }

func (a *App) syncCoordinateEdit(id int, value string) {
	if !a.keyboardCommitInProgress && a.keyboardDirtyID == id {
		return
	}
	h := a.controls[id]
	if h == 0 {
		return
	}
	if value == "---" {
		value = ""
	}
	if windowText(h) == value {
		return
	}
	a.keyboardProgrammatic = true
	setText(h, value)
	a.keyboardProgrammatic = false
}

func windowText(h syscall.Handle) string {
	if h == 0 {
		return ""
	}
	n, _, _ := procGetWindowTextLengthW.Call(uintptr(h))
	buf := make([]uint16, int(n)+2)
	procGetWindowTextW.Call(uintptr(h), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return syscall.UTF16ToString(buf)
}

func (a *App) coordinateFocusChanged(id, code int) {
	if code == EN_SETFOCUS {
		a.keyboardFocusID = id
		a.keyboardErrorID = 0
		// Select the previous value ready for direct numeric replacement.
		procSendMessageW.Call(uintptr(a.controls[id]), EM_SETSEL, 0, ^uintptr(0))
	} else if code == EN_KILLFOCUS && a.keyboardFocusID == id {
		a.keyboardFocusID = 0
	}
	if a.currentScreen == "operations" {
		procInvalidateRect.Call(uintptr(a.hwnd), 0, 1)
	}
}

func (a *App) handleCoordinateKey(msg *MSG) bool {
	if a == nil || msg == nil || msg.Message != WM_KEYDOWN || a.currentScreen != "operations" {
		return false
	}
	id := 0
	for _, n := range coordinateIDs() {
		if syscall.Handle(msg.HWnd) == a.controls[n] {
			id = n
			break
		}
	}
	// The window is not a Win32 dialog, so native Tab normally does not
	// change focus. Support Tab from any control on Operations into the
	// currently expected coordinate, then cycle through the four fields.
	if id == 0 && msg.WParam == VK_TAB {
		next := a.nextCoordinateForTab()
		procSetFocus.Call(uintptr(a.controls[next]))
		a.refreshOperationsUI()
		return true
	}
	// Ctrl+Backspace always rewinds a procedural stage, including when editing.
	// Plain Backspace does so outside a currently modified input field.
	if msg.WParam == VK_BACK {
		ctrl, _, _ := procGetKeyState.Call(VK_CONTROL)
		forced := (uint16(ctrl) & 0x8000) != 0
		if id == 0 || forced || a.keyboardDirtyID != id {
			a.keyboardDirtyID = 0
			a.rewindCoordinateStep()
			next := a.coordinateStepHighlight()
			if a.isCoordinateID(next) {
				procSetFocus.Call(uintptr(a.controls[next]))
				// Rewind may return to the SAME focused field; ensure the
				// existing value is selected ready for complete replacement.
				procSendMessageW.Call(uintptr(a.controls[next]), EM_SETSEL, 0, ^uintptr(0))
			}
			return true
		}
	}
	if id == 0 {
		return false
	}
	switch msg.WParam {
	case VK_TAB:
		next := a.nextCoordinateForTab()
		// Once a pair is confirmed, click/edit fields remain naturally
		// navigable in X→Y→target X→target Y order.
		if !a.pendingActive() && a.actualSet {
			switch id {
			case 5001:
				next = 5002
			case 5002:
				next = 5011
			case 5011:
				next = 5012
			}
		}
		procSetFocus.Call(uintptr(a.controls[next]))
		a.refreshOperationsUI()
		return true
	case VK_RETURN:
		a.commitKeyboardCoordinate(id)
		return true
	}
	return false
}

// Stage help occupies the EXISTING context box; no new text/controls are laid out.
// Voice and keyboard use the same pending state, making the next expected step
// unambiguous even when the user has not yet set an ACTUAL position.
func (a *App) keyboardStageHelp() string {
	switch a.pending.Kind {
	case "actual", "switch_actual":
		if a.keyboardStepOverride == 1 {
			return "STEP 1/6 — REFRESH ACTUAL; TAB to edit MORTAR X-RAY. Confirmed coordinates are protected."
		}
		if a.expectedCoordinateAxis() == "X" {
			return "STEP 2/6 — USER ACTUAL X-RAY; ENTER X then TAB to Y. (STEP 1 ACTUAL STARTED)"
		}
		if a.expectedCoordinateAxis() == "Y" {
			return "STEP 3/6 — USER ACTUAL YANKEE; ENTER Y to CONFIRM mortar position"
		}
		return "ACTUAL X/Y received — confirming position"
	case "target":
		if a.keyboardStepOverride == 4 {
			return "STEP 4/6 — REFRESH TARGET; TAB to edit TARGET X-RAY. Confirmed targets are protected."
		}
		if a.expectedCoordinateAxis() == "X" {
			return "STEP 5/6 — TARGET X-RAY; ENTER X then TAB to Y. (STEP 4 TARGET STARTED)"
		}
		if a.expectedCoordinateAxis() == "Y" {
			return "STEP 6/6 — TARGET YANKEE; ENTER Y to CONFIRM target and calculate solution"
		}
		return "TARGET X/Y received — calculating solution"
	case "clear_all", "clear_actual":
		return "CONFIRM OR CANCEL THE CLEAR REQUEST FIRST"
	}
	if !a.actualSet {
		return "STEP 1/6 — MORTAR POSITION NOT SET! Say ACTUAL / ACTUALLY or select USER ACTUAL X, type, ENTER; TAB to Y, ENTER. SET ACTUAL FIRST."
	}
	return "STEP 4/6 — ACTUAL CONFIRMED. Say NEW TARGET or select TARGET X, type, ENTER; TAB to Y, ENTER. Existing target fields are editable."
}

func (a *App) coordinateStepHighlight() int {
	switch a.pending.Kind {
	case "actual", "switch_actual":
		if a.keyboardStepOverride == 1 {
			return 5001
		}
		if a.expectedCoordinateAxis() == "Y" {
			return 5002
		}
		return 5001
	case "target":
		if a.keyboardStepOverride == 4 {
			return 5011
		}
		if a.expectedCoordinateAxis() == "Y" {
			return 5012
		}
		return 5011
	case "clear_all", "clear_actual":
		return 0
	}
	if !a.actualSet {
		return 5001
	}
	if a.keyboardDirtyID != 0 {
		return a.keyboardDirtyID
	}
	if a.isCoordinateID(a.keyboardFocusID) {
		return a.keyboardFocusID
	}
	return 5011
}

func (a *App) drawCoordinateStepOutline(hdc syscall.Handle, width int) {
	if a.currentScreen != "operations" {
		return
	}
	id := a.coordinateStepHighlight()
	h := a.controls[id]
	rc, ok := a.controlBase[h]
	if id == 0 || h == 0 || !ok {
		return
	}
	x := a.registerX(int(rc.Left), width)
	ww := a.registerWidth(int(rc.Right-rc.Left), width)
	y := int(rc.Top)
	hh := int(rc.Bottom - rc.Top)
	// Steady (no animation), BLUE focus marker; respects original G6 registration.
	for off := 1; off <= 3; off++ {
		frameColor(hdc, RECT{int32(x - off), int32(y - off), int32(x + ww + off), int32(y + hh + off)}, 64, 153, 255)
	}
}

func parseKeyboardCoordinate(s string) (string, error) {
	s = strings.TrimSpace(strings.ReplaceAll(s, ",", "."))
	if s == "" || s == "---" {
		return "", fmt.Errorf("coordinate is empty")
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) || math.Abs(f) > 99999 {
		return "", fmt.Errorf("enter a valid numeric X/Y coordinate (for example 97.86)")
	}
	return strconv.FormatFloat(f, 'f', -1, 64), nil
}

func (a *App) beginKeyboardPosition(isTarget bool) bool {
	if isTarget && !a.actualSet {
		a.addOperationLog("ACTUAL FIRST — set both USER ACTUAL coordinates before entering target.")
		procSetFocus.Call(uintptr(a.controls[5001]))
		return false
	}
	if a.pendingActive() {
		permitted := (!isTarget && (a.pending.Kind == "actual" || a.pending.Kind == "switch_actual")) || (isTarget && a.pending.Kind == "target")
		if !permitted {
			a.addOperationLog("FINISH CURRENT STEP FIRST — use its second coordinate or CANCEL before switching position/target.")
			return false
		}
		return true
	}
	if !isTarget {
		if a.actualSet {
			a.setPending("switch_actual", -1)
			a.pending.X, a.pending.Y = a.actualX, a.actualY
			a.addOperationLog("KEYBOARD — EDITING USER ACTUAL; old confirmed position protected until ENTER.")
		} else {
			a.setPending("actual", -1)
			a.addOperationLog("KEYBOARD STEP 1/6 — USER ACTUAL entry started.")
		}
		return true
	}
	slot := a.activeTarget
	switch {
	case slot == -2 && a.unsavedExists:
		a.setPending("target", -2)
		a.pending.X, a.pending.Y = a.unsaved.X, a.unsaved.Y
		a.addOperationLog("KEYBOARD — EDITING ACTIVE UNSAVED TARGET; old values protected until ENTER.")
	case slot >= 0 && slot < 4 && a.targets[slot].Confirmed:
		a.setPending("target", slot)
		a.pending.X, a.pending.Y = a.targets[slot].X, a.targets[slot].Y
		a.addOperationLog(fmt.Sprintf("KEYBOARD — EDITING TARGET %d (existing values protected until ENTER).", slot+1))
	default:
		next := a.nextTargetSlot()
		if next < 0 {
			a.addOperationLog("TARGET MEMORY FULL — clear a saved target to enter another.")
			return false
		}
		a.setPending("target", next)
		a.addOperationLog(fmt.Sprintf("KEYBOARD STEP 4/6 — TARGET %d started.", next+1))
	}
	return true
}

func (a *App) commitKeyboardCoordinate(id int) {
	isTarget := id == 5011 || id == 5012
	axis := "X"
	if id == 5002 || id == 5012 {
		axis = "Y"
	}
	raw := windowText(a.controls[id])
	v, err := parseKeyboardCoordinate(raw)
	if err != nil {
		a.writeInteractionDiagnostic("KEYBOARD_REJECT", fmt.Sprintf("field=%d; input=%q; error=%v", id, raw, err), "REJECTED", a.voiceDiagnosticState(), a.voiceDiagnosticState())
		a.keyboardErrorID = id
		a.refreshOperationsUI()
		a.addOperationLog("KEYBOARD REJECTED — " + strings.ToUpper(axis) + ": " + err.Error())
		return
	}
	if !a.beginKeyboardPosition(isTarget) {
		return
	}
	// New chains require X before Y. Existing confirmed values can be edited
	// independently; other axis is prefilled and never silently discarded.
	if a.pending.X == "" && axis == "Y" {
		a.addOperationLog("STEP ORDER — enter X-RAY first, then TAB to YANKEE.")
		return
	}
	before := a.voiceDiagnosticState()
	a.keyboardCommitInProgress = true
	a.keyboardDirtyID = 0
	// The shared voice state machine handles input, auto-confirmation,
	// transcription, Piper responses and the L81 firing solution.
	if (axis == "X" && a.pending.X != "") || (axis == "Y" && a.pending.Y != "") {
		a.pending.Replacing = axis
	}
	a.acceptVoiceCoordinate(axis, v)
	if axis == "Y" && !a.pendingActive() {
		a.keyboardFocusID = 0
	}
	a.keyboardCommitInProgress = false
	a.writeInteractionDiagnostic("KEYBOARD_ACCEPT", fmt.Sprintf("field=%d; axis=%s; input=%q; normalized=%q", id, axis, raw, v), "ACCEPTED", before, a.voiceDiagnosticState())
	a.addOperationLog("KEYBOARD ENTER — " + map[bool]string{true: "TARGET ", false: "ACTUAL "}[isTarget] + axis + " = " + v + " accepted.")
	a.refreshOperationsUI()
}

// Record only keys delivered to this app; never hook system-wide keyboard activity.
func (a *App) handleCoordinateKeyLogged(msg *MSG) bool {
	if msg == nil || msg.Message != WM_KEYDOWN || a.currentScreen != "operations" {
		return a.handleCoordinateKey(msg)
	}
	key := ""
	switch msg.WParam {
	case VK_TAB:
		key = "TAB"
	case VK_RETURN:
		key = "ENTER"
	case VK_BACK:
		key = "BACKSPACE"
	case 0x1B:
		key = "ESCAPE"
	case 0x25:
		key = "LEFT"
	case 0x26:
		key = "UP"
	case 0x27:
		key = "RIGHT"
	case 0x28:
		key = "DOWN"
	}
	if key == "" {
		return a.handleCoordinateKey(msg)
	}
	before := a.voiceDiagnosticState()
	focus := a.keyboardFocusID
	handled := a.handleCoordinateKey(msg)
	a.writeInteractionDiagnostic("KEYBOARD", fmt.Sprintf("key=%s; focus_before=%d; focus_after=%d; handled=%t", key, focus, a.keyboardFocusID, handled), "OBSERVED", before, a.voiceDiagnosticState())
	return handled
}
