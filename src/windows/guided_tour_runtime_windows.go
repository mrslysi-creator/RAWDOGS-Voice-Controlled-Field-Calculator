//go:build windows

package main

import (
	"fmt"
	"strings"
	"syscall"
	"unsafe"
)

const (
	TOUR_VOSK_MONKEY  = -8101 // painted badge, not a child control
	TOUR_PIPER_MONKEY = -8102
	ID_TOUR_PREV      = 8100 + iota
	ID_TOUR_PLAY
	ID_TOUR_PAUSE
	ID_TOUR_STOP
	ID_TOUR_NEXT
	ID_TOUR_RESTART
	ID_TOUR_CAPTION
	ID_TOUR_INDEX
	ID_TOUR_CURRENT
)

var (
	tourGetWindowRect   = user32.NewProc("GetWindowRect")
	tourScreenToClient  = user32.NewProc("ScreenToClient")
	tourIsWindowVisible = user32.NewProc("IsWindowVisible")
)

type tourPoint struct{ X, Y int32 }

type tourSnapshot struct {
	actualX, actualY                                                      string
	actualSet                                                             bool
	targets                                                               [4]TargetSlot
	unsaved                                                               TargetSlot
	unsavedExists                                                         bool
	activeTarget                                                          int
	pending                                                               PendingState
	lastClear                                                             *clearSnapshot
	operationLog                                                          []string
	transcript                                                            []string
	lastCoordKind, lastCoordAxis                                          string
	lastCoordSlot, keyboardDirtyID, keyboardErrorID, keyboardStepOverride int
	coach                                                                 cueCoach
}

func (a *App) captureTourSnapshot() *tourSnapshot {
	return &tourSnapshot{
		actualX: a.actualX, actualY: a.actualY, actualSet: a.actualSet,
		targets: a.targets, unsaved: a.unsaved, unsavedExists: a.unsavedExists,
		activeTarget: a.activeTarget, pending: a.pending, lastClear: a.lastClear,
		operationLog:  append([]string(nil), a.operationLog...),
		transcript:    append([]string(nil), a.transcript...),
		lastCoordKind: a.lastCoordKind, lastCoordAxis: a.lastCoordAxis, lastCoordSlot: a.lastCoordSlot,
		keyboardDirtyID: a.keyboardDirtyID, keyboardErrorID: a.keyboardErrorID,
		keyboardStepOverride: a.keyboardStepOverride,
		coach:                a.cueCoach,
	}
}
func (a *App) restoreTourSnapshot(s *tourSnapshot) {
	if s == nil {
		return
	}
	a.actualX, a.actualY, a.actualSet = s.actualX, s.actualY, s.actualSet
	a.targets, a.unsaved, a.unsavedExists = s.targets, s.unsaved, s.unsavedExists
	a.activeTarget, a.pending, a.lastClear = s.activeTarget, s.pending, s.lastClear
	a.operationLog = append([]string(nil), s.operationLog...)
	a.transcript = append([]string(nil), s.transcript...)
	a.lastCoordKind, a.lastCoordAxis, a.lastCoordSlot = s.lastCoordKind, s.lastCoordAxis, s.lastCoordSlot
	a.keyboardDirtyID, a.keyboardErrorID, a.keyboardStepOverride = s.keyboardDirtyID, s.keyboardErrorID, s.keyboardStepOverride
	a.cueCoach = s.coach
	a.firingLastSignature = ""
	a.refreshOperationsUI()
	for _, pair := range []struct {
		id   int
		rows []string
	}{{5031, a.operationLog}, {4102, a.transcript}, {5052, a.transcript}} {
		if h := a.controls[pair.id]; h != 0 {
			procSendMessageW.Call(uintptr(h), LB_RESETCONTENT, 0, 0)
			for _, row := range pair.rows {
				procSendMessageW.Call(uintptr(h), LB_ADDSTRING, 0, uintptr(unsafe.Pointer(utf16Ptr(row))))
			}
		}
	}
}

func (a *App) startTour() {
	if a == nil || len(guidedTour) == 0 {
		return
	}
	if !a.actualVoiceReady() {
		a.showScreen("settings")
		setText(a.controls[3004], "GUIDE WAITING — DEFAULT PIPER VOICE NOT READY. REPAIR THE DEFAULT PACK AND RETRY.")
		if !a.provisionActive {
			a.beginStartupProvisioning()
		}
		return
	}
	if a.tourActive {
		a.tourGoTo(0, false)
		return
	}
	a.narratorStop(true)
	a.stopRecognizer() // Scripted demonstrations must never be microphone input.
	a.tourOriginalScreen, a.tourOriginalPage = a.currentScreen, a.manualPage
	a.tourSnapshot = a.captureTourSnapshot()
	a.tourActive = true
	a.tourHold = false
	a.tourIndex = 0
	a.tourDemoMode = false
	a.tourOutputTested = false
	a.tourOutputConfirmed = false
	a.tourDemoShown = false
	a.showScreen("settings")
	setText(a.controls[ID_GUIDE_NEXT], "NEXT SECTION")
	a.tourGoTo(0, false)
}

func (a *App) endTour() {
	if !a.tourActive {
		return
	}
	a.narratorStop(true)
	a.tourActive = false
	a.tourHold = false
	a.tourDemoMode = false
	a.restoreTourSnapshot(a.tourSnapshot)
	a.tourSnapshot = nil
	a.tourDemoShown = false
	a.tourOutputTested = false
	a.tourOutputConfirmed = false
	setText(a.controls[ID_GUIDE_NEXT], "START GUIDE")
	if a.tourOriginalScreen != "" {
		a.showScreen(a.tourOriginalScreen)
	} else {
		a.showScreen("settings")
	}
	if a.currentScreen == "manual" {
		a.setManualPage(a.tourOriginalPage)
	}
	a.updateSetupStatus()
	procInvalidateRect.Call(uintptr(a.hwnd), 0, 1)
}

func (a *App) tourGoTo(index int, play bool) {
	if !a.tourActive {
		return
	}
	if index < 0 {
		index = 0
	}
	if index >= len(guidedTour) {
		index = len(guidedTour) - 1
	}
	a.narratorStop(true)
	a.tourIndex = index
	s := guidedTour[index]
	a.tourReplayDemo(index)
	a.showScreen(s.Tab)
	if s.Tab == "manual" && s.Page != "" {
		a.setManualPage(s.Page)
	}
	a.narratorMu.Lock()
	a.narratorPage = "tour"
	a.narratorIndex = index
	a.narratorDurations = nil
	a.narratorFinished = false
	a.narratorMu.Unlock()
	// Page four owns the searchable, verbatim master text. The current
	// chapter selection is the same index as the spoken WAV segment.
	a.selectGuideIndex(index)
	a.narratorShowSection()
	a.tourTrainingExample()
	if s.Tab == "settings" {
		setText(a.controls[3004], fmt.Sprintf("GUIDE %d/%d — %s\r\n%s", index+1, len(guidedTour), s.Title, s.Narration))
	}
	if index == 0 && !a.tourOutputConfirmed {
		setText(a.controls[3004], "GUIDE 1 — FIRST SELECT PIPER OUTPUT, PRESS TEST VOICE, THEN PRESS NEXT IF YOU HEARD IT. THE REST OF THE TOUR IS SPOKEN.")
	}
	procInvalidateRect.Call(uintptr(a.hwnd), 0, 1)
	if play && (index > 0 || a.tourOutputConfirmed) {
		a.narratorPlay()
	}
}

func (a *App) tourNext() {
	if !a.tourActive {
		return
	}
	if a.tourIndex == 0 && a.isTtsSpeaking() {
		setText(a.controls[3004], "WAIT FOR THE TEST VOICE TO FINISH, THEN PRESS NEXT IF YOU HEARD IT.")
		return
	}
	if a.tourIndex == 0 && !a.tourOutputConfirmed {
		if !a.tourOutputTested {
			setText(a.controls[3004], "FIRST PRESS TEST VOICE. WHEN YOU HEAR PIPER, PRESS NEXT. IF SILENT, CHANGE OUTPUT AND RETRY.")
			return
		}
		if idx, _ := a.comboSelection(ID_OUTPUT_COMBO); idx < 0 {
			setText(a.controls[3004], "SELECT A VALID OUTPUT DEVICE BEFORE CONTINUING.")
			return
		}
		a.tourOutputConfirmed = true
	}
	if a.tourIndex+1 >= len(guidedTour) {
		a.endTour()
		return
	}
	a.tourGoTo(a.tourIndex+1, !a.tourHold)
}
func (a *App) tourPrev() {
	if a.tourActive {
		a.tourGoTo(a.tourIndex-1, !a.tourHold)
	}
}
func (a *App) tourRestart() {
	if a.tourActive {
		a.tourGoTo(0, a.tourOutputConfirmed && !a.tourHold)
	}
}
func (a *App) tourPlay() {
	if !a.tourActive {
		return
	}
	if a.tourIndex == 0 && !a.tourOutputConfirmed {
		setText(a.controls[3004], "AUDIO FIRST: SELECT OUTPUT, PRESS TEST VOICE, THEN PRESS NEXT.")
		return
	}
	a.tourHold = false
	a.narratorPlay()
}
func (a *App) tourPause() {
	if a.tourActive {
		a.tourHold = true
		a.narratorPause()
	}
}
func (a *App) tourStop() {
	if a.tourActive {
		a.tourHold = true
		a.narratorStop(true)
	}
}

func (a *App) selectGuideIndex(i int) {
	if h := a.controls[ID_TOUR_INDEX]; h != 0 {
		procSendMessageW.Call(uintptr(h), CB_SETCURSEL, uintptr(i), 0)
	}
	if h := a.controls[ID_TOUR_CURRENT]; h != 0 && i >= 0 && i < len(guidedTour) {
		setText(h, guidedTour[i].Narration)
	}
	// EM_SETSEL, EM_SCROLLCARET: highlight the real spoken paragraph, not an
	// independent paraphrase. Never send a full text redraw on every timer tick.
	if h := a.controls[6206]; h != 0 && i >= 0 && i < len(guidedTour) {
		start := 0
		for n := 0; n < i; n++ {
			start += len([]rune(guidedTour[n].Title)) + 2 + len([]rune(guidedTour[n].Narration)) + 4
		}
		start += len([]rune(guidedTour[i].Title)) + 2
		end := start + len([]rune(guidedTour[i].Narration))
		procSendMessageW.Call(uintptr(h), EM_SETSEL, uintptr(start), uintptr(end))
		procSendMessageW.Call(uintptr(h), 0x00B7, 0, 0) // EM_SCROLLCARET
	}
}

func (a *App) tourReplayDemo(upto int) {
	if a.tourSnapshot == nil {
		return
	}
	anyDemo := false
	for i := 0; i <= upto && i < len(guidedTour); i++ {
		if guidedTour[i].Demo != "" {
			anyDemo = true
			break
		}
	}
	if !anyDemo && !a.tourDemoShown {
		return
	}
	a.tourDemoShown = anyDemo
	a.tourDemoMode = true
	a.restoreTourSnapshot(a.tourSnapshot)
	// Only the named demonstration events mutate the temporary workspace.
	// Replaying from the original snapshot makes Previous/Next deterministic.
	for i := 0; i <= upto && i < len(guidedTour); i++ {
		switch guidedTour[i].Demo {
		case "begin":
			a.actualX, a.actualY, a.actualSet = "", "", false
			a.targets = [4]TargetSlot{}
			a.unsaved = TargetSlot{}
			a.unsavedExists = false
			a.activeTarget = -1
			a.pending = PendingState{Slot: -1}
			a.operationLog = []string{"GUIDED DEMONSTRATION — NOT LIVE COORDINATES"}
			a.lastClear = nil
		case "actual_start":
			a.refreshActualSixStep()
		case "actual_x":
			a.acceptVoiceCoordinate("X", "100.42")
		case "actual_y":
			a.acceptVoiceCoordinate("Y", "114.55")
		case "target_start":
			a.startNewTarget()
		case "target_x":
			a.acceptVoiceCoordinate("X", "103.42")
		case "target_y":
			a.acceptVoiceCoordinate("Y", "118.55")
		case "recall":
			a.recallTarget(0)
		case "clear_target":
			a.clearOneTarget(0)
		case "move_actual":
			a.refreshActualSixStep()
			a.acceptVoiceCoordinate("X", "101.42")
			a.acceptVoiceCoordinate("Y", "114.55")
		case "new_target_start":
			a.startNewTarget()
		case "new_target_x":
			a.acceptVoiceCoordinate("X", "104.42")
		case "new_target_y":
			a.acceptVoiceCoordinate("Y", "118.55")
		case "correct_y":
			a.universalCorrection()
			a.acceptVoiceCoordinate("Y", "119.55")
		case "cancel_draft":
			a.startNewTarget()
			a.cancelPending()
		}
	}
	a.tourDemoMode = false
	a.refreshOperationsUI()
}

func (a *App) drawTourHighlights(hdc syscall.Handle) {
	if !a.tourActive || a.tourIndex < 0 || a.tourIndex >= len(guidedTour) {
		return
	}
	s := guidedTour[a.tourIndex]
	if s.Tab != a.currentScreen || (s.Tab == "manual" && s.Page != "" && s.Page != a.manualPage) {
		return
	}
	var client RECT
	procGetClientRect.Call(uintptr(a.hwnd), uintptr(unsafe.Pointer(&client)))
	for _, id := range s.Focus {
		if id == TOUR_VOSK_MONKEY || id == TOUR_PIPER_MONKEY {
			baseW := a.layoutBaseW
			if baseW <= 0 {
				baseW = int(client.Right)
			}
			px, pw := a.registerX(baseW-388, int(client.Right)), a.registerWidth(368, int(client.Right))
			x := px + 9
			if id == TOUR_PIPER_MONKEY {
				x += pw / 2
			}
			for pad := int32(2); pad <= 3; pad++ {
				frameColor(hdc, RECT{int32(x) - pad, 123 - pad, int32(x+56) + pad, 180 + pad}, 255, 0, 170)
			}
			continue
		}
		h := a.controls[id]
		if h == 0 {
			continue
		}
		visible, _, _ := tourIsWindowVisible.Call(uintptr(h))
		if visible == 0 {
			continue
		}
		var rect RECT
		result, _, _ := tourGetWindowRect.Call(uintptr(h), uintptr(unsafe.Pointer(&rect)))
		if result == 0 {
			continue
		}
		leftTop := tourPoint{rect.Left, rect.Top}
		rightBottom := tourPoint{rect.Right, rect.Bottom}
		tourScreenToClient.Call(uintptr(a.hwnd), uintptr(unsafe.Pointer(&leftTop)))
		tourScreenToClient.Call(uintptr(a.hwnd), uintptr(unsafe.Pointer(&rightBottom)))
		// Win32 COMBOBOX includes an invisible dropdown. Outline only the
		// CLOSED control, never its 200px concealed list or neighbouring card.
		if a.tourIsCombo(id) && rightBottom.Y-leftTop.Y > 32 {
			rightBottom.Y = leftTop.Y + 30
		}
		if leftTop.X < 3 || leftTop.Y < 3 || rightBottom.X > client.Right-3 || rightBottom.Y > client.Bottom-3 {
			continue
		}
		if rightBottom.X <= leftTop.X || rightBottom.Y <= leftTop.Y {
			continue
		}
		for pad := int32(2); pad <= 3; pad++ {
			frameColor(hdc, RECT{leftTop.X - pad, leftTop.Y - pad, rightBottom.X + pad, rightBottom.Y + pad}, 255, 0, 170)
		}
	}
}
func (a *App) tourIsCombo(id int) bool {
	switch id {
	case ID_OUTPUT_COMBO, ID_MIC_COMBO, ID_GAIN_COMBO, ID_LANG_COMBO, ID_CONF_COMBO,
		ID_VOICE_COMBO, ID_SPEED_COMBO, ID_VOLUME_COMBO, ID_THEME_COMBO, ID_TEAM_COMBO,
		ID_VOSK_LIBRARY_COMBO, ID_ACTUAL_LIBRARY_COMBO, ID_LIBRARY_COUNTRY, ID_TOUR_INDEX:
		return true
	}
	return false
}

// The current index, narration and walkthrough action are one unit. All
// checks are pure: no second audio thread and no second recognition system.
func tourTextForIndex(index int) string {
	if index < 0 || index >= len(guidedTour) {
		return ""
	}
	return strings.TrimSpace(guidedTour[index].Narration)
}

// Demonstrate a real example without writing to the operator's profile.
// Voice Training refresh restores the user's real results when the tour exits.
func (a *App) tourTrainingExample() {
	if !a.tourActive || a.currentScreen != "training" {
		return
	}
	name := guidedTour[a.tourIndex].Title
	if !strings.Contains(name, "Voice Training: why") && !strings.Contains(name, "Vosk Heard and RAWDOGS") && !strings.Contains(name, "Personal pronunciations and colors") {
		return
	}
	setText(a.controls[ID_ADAPT_EXPECTED], "EXAMPLE: SAY THREE")
	setText(a.controls[ID_ADAPT_HEARD], "VOSK HEARD: FREE — SCRIPTED EXAMPLE")
	setText(a.controls[ID_ADAPT_INTERPRETATION], "RAWDOGS INTERPRETS: THREE — EXAMPLE")
	setText(a.controls[ID_ADAPT_STATUS], "GUIDE ONLY — NO PRONUNCIATION DATA WAS LEARNED")
}
