//go:build windows

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unsafe"
)

const (
	ID_ADAPT_START = 7200 + iota
	ID_ADAPT_STOP
	ID_ADAPT_REPEAT
	ID_ADAPT_CONFIRM
	ID_ADAPT_SKIP
	ID_ADAPT_REFERENCE
	ID_ADAPT_REMOVE
	ID_ADAPT_LOAD_PROFILE
)
const ID_ADAPT_OPERATOR = 4221
const ID_ADAPT_RULE_LIST = 4222
const ID_ADAPT_TITLE = 4223
const ID_ADAPT_EXPECTED = 4224
const ID_ADAPT_HEARD = 4225
const ID_ADAPT_STATUS = 4226
const ID_ADAPT_PROFILE_HEADER = 4227
const ID_ADAPT_MODEL = 4228
const ID_ADAPT_COUNTS = 4229
const ID_ADAPT_RULES_HEADER = 4230
const ID_ADAPT_TRANSCRIPT_HEADER = 4231
const ID_ADAPT_PROMPT_LABEL = 4232
const ID_ADAPT_INTERPRETATION = 4233
const ID_ADAPT_LEGEND = 4234

const LB_GETCURSEL = 0x0188

type voiceTrainingState struct {
	Active         bool
	Index          int
	LastRaw        string
	LastConfidence int
	Candidate      *voiceCandidate
	Status         string
	ShowCue        bool
	PromptPending  bool
	// Display-only feedback: it never changes the recognition or learning logic.
	Interpretation string
	FeedbackClass  string // neutral, green, pink, amber, red
}

func (a *App) buildVoiceAdaptationScreen() {
	s := "training"
	// Everything y<=230 is LOCKED: original title, nav, system health,
	// listener readiness, team/Piper/Vosk badges must never be moved or restyled.
	a.addScreenButton(s, ID_TRAIN_LISTEN, 20, 238, 160, 36, "START LISTENING")
	a.addScreenButton(s, ID_TRAIN_STOP, 210, 238, 160, 36, "STOP LISTENING")
	a.addScreenButton(s, ID_TRAIN_CLEAR, 400, 238, 160, 36, "CLEAR TRANSCRIPT")
	a.addScreenButton(s, ID_ADAPT_START, 590, 238, 160, 36, "START CALIBRATION")
	a.addScreenButton(s, ID_ADAPT_STOP, 780, 238, 160, 36, "STOP CALIBRATION")
	a.addScreenButton(s, ID_ADAPT_REPEAT, 20, 280, 160, 34, "REPEAT EXERCISE")
	a.addScreenButton(s, ID_ADAPT_CONFIRM, 210, 280, 160, 34, "CONFIRM TRAINING")
	a.addScreenButton(s, ID_ADAPT_SKIP, 400, 280, 160, 34, "SKIP EXERCISE")
	a.addScreenButton(s, ID_ADAPT_REFERENCE, 590, 280, 160, 34, "CUE REFERENCE")
	a.addScreenButton(s, ID_ADAPT_REMOVE, 780, 280, 160, 34, "REMOVE SELECTED")
	a.addScreenStaticWithID(s, ID_ADAPT_TITLE, 32, 343, 500, 23, "CURRENT TRAINING EXERCISE — READY", true)
	a.addScreenStaticWithID(s, ID_ADAPT_PROMPT_LABEL, 32, 371, 520, 24, "PIPER REQUESTS — SAY THE WORDS BELOW", true)
	prompt := a.addScreenStaticWithID(s, ID_ADAPT_EXPECTED, 32, 397, 520, 41, "PRESS START CALIBRATION", true)
	a.setFont(prompt, a.fontTitle)
	a.addScreenStaticWithID(s, ID_ADAPT_HEARD, 32, 439, 520, 29, "VOSK HEARD: ---", false)
	a.addScreenStaticWithID(s, ID_ADAPT_INTERPRETATION, 32, 470, 520, 29, "RAWDOGS INTERPRETS: WAITING", true)
	a.addScreenStaticWithID(s, ID_ADAPT_STATUS, 32, 502, 520, 28, "READY — all calibration data stays offline", false)
	a.addScreenStaticWithID(s, ID_ADAPT_TRANSCRIPT_HEADER, 32, 499, 520, 24, "LIVE RECOGNITION — RAW HEARD / CONFIDENCE / INTERPRETATION", true)
	a.addScreenStaticWithID(s, 4105, 32, 528, 520, 29, "LAST RESULT: ---", true)
	transcript := a.addScreenTrainingList(s, 4102, 32, 564, 520, 100)
	procSendMessageW.Call(uintptr(transcript), LB_ADDSTRING, 0, uintptr(unsafe.Pointer(utf16Ptr("Press START LISTENING, then START CALIBRATION."))))
	a.addScreenStaticWithID(s, ID_ADAPT_PROFILE_HEADER, 750, 343, 520, 24, "OPERATOR PROFILE — LOCAL ONLY", true)
	a.addScreenStaticWithID(s, ID_ADAPT_MODEL, 750, 372, 520, 24, "VOSK MODEL: checking", false)
	operator := a.createControl(ID_ADAPT_OPERATOR, "EDIT", a.voiceProfile.Operator, WS_CHILD|WS_VISIBLE|WS_TABSTOP|WS_BORDER|ES_AUTOHSCROLL, 750, 401, 220, 31, 0, s)
	a.setFont(operator, a.fontNormal)
	a.addScreenButton(s, ID_ADAPT_LOAD_PROFILE, 980, 401, 175, 31, "LOAD / CREATE PROFILE")
	a.addScreenStaticWithID(s, ID_ADAPT_COUNTS, 750, 440, 520, 24, "CONFIRMED PRONUNCIATIONS: 0", false)
	a.addScreenStaticWithID(s, ID_ADAPT_LEGEND, 750, 480, 520, 42, "GREEN ACCEPTED | MAGENTA LEARNING\r\nAMBER REVIEW | RED REJECTED", false)
	a.addScreenStaticWithID(s, ID_ADAPT_RULES_HEADER, 750, 499, 520, 24, "PRONUNCIATION LIBRARY — CLICK A RULE TO REMOVE", true)
	rules := a.createControl(ID_ADAPT_RULE_LIST, "LISTBOX", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_VSCROLL|LBS_NOINTEGRALHEIGHT|LBS_OWNERDRAWFIXED|LBS_HASSTRINGS, 750, 530, 520, 110, 0, s)
	a.setFont(rules, a.fontNormal)
	procSendMessageW.Call(uintptr(rules), LB_SETITEMHEIGHT, 0, 24)
	a.addScreenStaticWithID(s, 4104, 32, 718, 900, 28, "TEAM: MANTICORE — GREEN | LISTENER: STOPPED", true)
	a.layoutVoiceAdaptation()
	a.refreshVoiceTrainingUI()
}

func (a *App) layoutVoiceAdaptation() {
	if a == nil || a.controls[ID_ADAPT_START] == 0 || a.hwnd == 0 {
		return
	}
	var rc RECT
	procGetClientRect.Call(uintptr(a.hwnd), uintptr(unsafe.Pointer(&rc)))
	w, h := int(rc.Right-rc.Left), int(rc.Bottom-rc.Top)
	if w < 850 {
		w = 850
	}
	if h < 680 {
		h = 680
	}
	boxes := trainingBoxes(w, h)
	// Full-width dynamically sized 5-button rows. Protected top region unchanged.
	bw := (w - 40 - 4*9) / 5
	buttons1 := []int{ID_TRAIN_LISTEN, ID_TRAIN_STOP, ID_TRAIN_CLEAR, ID_ADAPT_START, ID_ADAPT_STOP}
	buttons2 := []int{ID_ADAPT_REPEAT, ID_ADAPT_CONFIRM, ID_ADAPT_SKIP, ID_ADAPT_REFERENCE, ID_ADAPT_REMOVE}
	for i, id := range buttons1 {
		a.moveAdapt(id, 20+i*(bw+9), 238, bw, 36)
	}
	for i, id := range buttons2 {
		a.moveAdapt(id, 20+i*(bw+9), 280, bw, 34)
	}
	l, r := boxes.leftTop, boxes.rightTop
	// The operator must be able to READ the exact words Piper asked for.
	// Keep the exercise visible, including after a rejected attempt.
	compact := l.h < 247
	if compact {
		a.moveAdapt(ID_ADAPT_TITLE, l.x+12, l.y+7, l.w-24, 23)
		a.moveAdapt(ID_ADAPT_PROMPT_LABEL, l.x+12, l.y+31, l.w-24, 21)
		a.moveAdapt(ID_ADAPT_EXPECTED, l.x+12, l.y+53, l.w-24, 34)
		a.moveAdapt(ID_ADAPT_HEARD, l.x+12, l.y+91, l.w-24, 24)
		a.moveAdapt(ID_ADAPT_INTERPRETATION, l.x+12, l.y+117, l.w-24, 25)
		a.moveAdapt(ID_ADAPT_STATUS, l.x+12, l.y+147, l.w-24, maxInt(30, l.h-155))
	} else {
		a.moveAdapt(ID_ADAPT_TITLE, l.x+12, l.y+10, l.w-24, 25)
		a.moveAdapt(ID_ADAPT_PROMPT_LABEL, l.x+12, l.y+40, l.w-24, 23)
		a.moveAdapt(ID_ADAPT_EXPECTED, l.x+12, l.y+69, l.w-24, 47)
		a.moveAdapt(ID_ADAPT_HEARD, l.x+12, l.y+123, l.w-24, 31)
		a.moveAdapt(ID_ADAPT_INTERPRETATION, l.x+12, l.y+162, l.w-24, 32)
		a.moveAdapt(ID_ADAPT_STATUS, l.x+12, l.y+207, l.w-24, maxInt(30, l.h-215))
	}
	// At narrower widths, shrink ONLY the exercise font; do not make the user
	// rely on truncated words in an oversized static control.
	if prompt := a.controls[ID_ADAPT_EXPECTED]; prompt != 0 {
		chosen := a.fontTitle
		if l.w < 680 {
			chosen = a.fontBold
		}
		// Avoid re-sending WM_SETFONT while resizing: unnecessary redraws
		// cause visible flicker even if the font didn't actually change.
		current, _, _ := procSendMessageW.Call(uintptr(prompt), 0x0031 /* WM_GETFONT */, 0, 0)
		if current != uintptr(chosen) {
			a.setFont(prompt, chosen)
		}
	}
	a.moveAdapt(ID_ADAPT_PROFILE_HEADER, r.x+12, r.y+12, r.w-24, 24)
	a.moveAdapt(ID_ADAPT_MODEL, r.x+12, r.y+43, r.w-24, 24)
	loadW := minInt(166, maxInt(138, r.w*38/100))
	a.moveAdapt(ID_ADAPT_OPERATOR, r.x+12, r.y+73, r.w-loadW-35, 31)
	a.moveAdapt(ID_ADAPT_LOAD_PROFILE, r.x+r.w-loadW-12, r.y+73, loadW, 31)
	a.moveAdapt(ID_ADAPT_COUNTS, r.x+12, r.y+111, r.w-24, 29)
	a.moveAdapt(ID_ADAPT_LEGEND, r.x+12, r.y+152, r.w-24, maxInt(28, r.h-159))
	l2, r2 := boxes.leftBottom, boxes.rightBottom
	a.moveAdapt(ID_ADAPT_TRANSCRIPT_HEADER, l2.x+12, l2.y+10, l2.w-24, 24)
	a.moveAdapt(4105, l2.x+12, l2.y+38, l2.w-24, 29)
	transcriptTop := 76
	if l2.h < 120 {
		transcriptTop = 70
	}
	a.moveAdapt(4102, l2.x+12, l2.y+transcriptTop, l2.w-24, maxInt(18, l2.h-transcriptTop-10))
	a.moveAdapt(ID_ADAPT_RULES_HEADER, r2.x+12, r2.y+10, r2.w-24, 24)
	a.moveAdapt(ID_ADAPT_RULE_LIST, r2.x+12, r2.y+44, r2.w-24, maxInt(34, r2.h-56))
	a.moveAdapt(4104, boxes.footer.x+12, boxes.footer.y+7, boxes.footer.w-24, 28)
}
func (a *App) moveAdapt(id, x, y, w, h int) {
	hw := a.controls[id]
	if hw == 0 {
		return
	}
	procMoveWindow.Call(uintptr(hw), uintptr(x), uintptr(y), uintptr(maxInt(32, w)), uintptr(maxInt(20, h)), 1)
}

func (a *App) voiceProfileDir() string {
	return filepath.Join(filepath.Dir(a.configPath), "voice_profiles")
}
func voiceOperatorName(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 40 {
		s = s[:40]
	}
	if s == "" {
		s = "DEFAULT"
	}
	return s
}
func (a *App) voiceProfileFilename(operator, model string) string {
	hash := sha256.Sum256([]byte(strings.ToLower(operator) + "\x00" + model))
	return filepath.Join(a.voiceProfileDir(), "voice_"+hex.EncodeToString(hash[:12])+".json")
}
func (a *App) selectedVoiceOperator() string {
	path := filepath.Join(a.voiceProfileDir(), "active_operator.txt")
	b, err := os.ReadFile(path)
	if err != nil {
		return "DEFAULT"
	}
	return voiceOperatorName(string(b))
}
func (a *App) loadVoiceProfile(operator string) error {
	operator = voiceOperatorName(operator)
	model := strings.TrimSpace(a.settings.VoiceName)
	if model == "" {
		model = "VOSK MODEL NOT SELECTED"
	}
	p, err := voiceProfileLoad(a.voiceProfileFilename(operator, model), operator, model)
	// A missing/corrupt *new* model's profile must never leave another
	// operator/model's old rules active. Fail closed with a blank profile.
	if err != nil {
		p = voiceProfile{Version: voiceProfileVersion, Operator: operator, Model: model}
	}
	a.voiceProfile = p
	a.voiceProfileKey = operator + "\x00" + model
	return err
}
func (a *App) ensureVoiceProfile() {
	operator := a.voiceProfile.Operator
	if operator == "" {
		operator = a.selectedVoiceOperator()
	}
	model := strings.TrimSpace(a.settings.VoiceName)
	if model == "" {
		model = "VOSK MODEL NOT SELECTED"
	}
	if a.voiceProfileKey != operator+"\x00"+model {
		if err := a.loadVoiceProfile(operator); err != nil {
			a.voiceTrain.Status = "PROFILE LOAD ERROR: " + err.Error()
		}
		a.refreshVoiceTrainingUI()
	}
}
func (a *App) saveVoiceProfile() error {
	return voiceProfileSave(a.voiceProfileFilename(a.voiceProfile.Operator, a.voiceProfile.Model), a.voiceProfile)
}
func (a *App) chooseVoiceProfile() {
	if a.voiceTrain.Active {
		a.voiceTrainingStop()
	}
	operator := voiceOperatorName(windowText(a.controls[ID_ADAPT_OPERATOR]))
	if err := a.loadVoiceProfile(operator); err != nil {
		a.voiceTrain.Status = "PROFILE LOAD ERROR: " + err.Error()
		a.refreshVoiceTrainingUI()
		return
	}
	if err := os.MkdirAll(a.voiceProfileDir(), 0700); err != nil {
		a.voiceTrain.Status = err.Error()
		a.refreshVoiceTrainingUI()
		return
	}
	if err := os.WriteFile(filepath.Join(a.voiceProfileDir(), "active_operator.txt"), []byte(operator), 0600); err != nil {
		a.voiceTrain.Status = err.Error()
		a.refreshVoiceTrainingUI()
		return
	}
	a.voiceTrain.Status = "Operator " + operator + " selected. Corrections apply only to matching Vosk model."
	a.refreshVoiceTrainingUI()
}

func (a *App) voiceTrainingStart() {
	if a.guideActive || a.manualNarrating {
		a.voiceTrain.Status = "Stop Guided Setup / Field Manual narration before calibration."
		a.refreshVoiceTrainingUI()
		return
	}
	a.ensureVoiceProfile()
	a.voiceTrain.Active = true
	a.voiceTrain.Index = 0
	a.voiceTrain.Candidate = nil
	a.voiceTrain.LastRaw = ""
	a.voiceTrain.FeedbackClass = "neutral"
	a.voiceTrain.Interpretation = "WAITING FOR RECOGNITION"
	a.voiceTrain.Status = "Calibration active; operational cues are temporarily isolated."
	a.cueCoach.reset()
	a.abandonPiperReadbacks()
	a.startRecognizer()
	a.refreshVoiceTrainingUI()
	if a.currentListenerState() == "READY — SAY COMMAND" {
		a.voiceTrainingPrompt()
	} else {
		a.voiceTrain.PromptPending = true
	}
}
func (a *App) voiceTrainingPrompt() {
	if !a.voiceTrain.Active {
		return
	}
	a.voiceTrain.PromptPending = false
	exercise := voiceExercises[a.voiceTrain.Index%len(voiceExercises)]
	a.voiceTrain.Status = "Read the highlighted phrase. Say every word, including all digits."
	a.voiceTrain.FeedbackClass = "neutral"
	a.voiceTrain.Interpretation = "WAITING FOR RECOGNITION"
	a.refreshVoiceTrainingUI()
	// Short spoken prompt. If Piper is muted/unavailable, text remains visible.
	a.actualRespond("Voice calibration. Say " + exercise.Expected + ".")
}
func (a *App) voiceTrainingStop() {
	if !a.voiceTrain.Active {
		return
	}
	a.voiceTrain.Active = false
	a.voiceTrain.Candidate = nil
	a.voiceTrain.PromptPending = false
	a.voiceTrain.Status = "Calibration stopped. Normal six-step voice control restored."
	a.voiceTrain.FeedbackClass = "neutral"
	a.voiceTrain.Interpretation = "CALIBRATION STOPPED"
	a.abandonPiperReadbacks()
	a.refreshVoiceTrainingUI()
	a.actualRespond("Calibration stopped.")
}
func (a *App) voiceTrainingSkip() {
	if !a.voiceTrain.Active {
		return
	}
	a.voiceTrain.Candidate = nil
	a.voiceTrain.Index = (a.voiceTrain.Index + 1) % len(voiceExercises)
	a.voiceTrain.LastRaw = ""
	a.abandonPiperReadbacks()
	a.voiceTrainingPrompt()
}
func (a *App) voiceTrainingRepeat() {
	if !a.voiceTrain.Active {
		return
	}
	a.voiceTrain.Candidate = nil
	a.voiceTrainingPrompt()
}
func (a *App) voiceTrainingConfirm() {
	if !a.voiceTrain.Active || a.voiceTrain.Candidate == nil {
		a.voiceTrain.Status = "No candidate to confirm. Say the exercise first."
		a.refreshVoiceTrainingUI()
		return
	}
	count, err := a.voiceProfile.confirmCandidate(*a.voiceTrain.Candidate, time.Now())
	if err != nil {
		a.voiceTrain.Status = "NOT SAVED: " + err.Error()
		a.refreshVoiceTrainingUI()
		return
	}
	if err = a.saveVoiceProfile(); err != nil {
		// In-memory alias must never become a live rule without successful save.
		a.loadVoiceProfile(a.voiceProfile.Operator)
		a.voiceTrain.Status = "PROFILE SAVE FAILED: " + err.Error()
		a.refreshVoiceTrainingUI()
		return
	}
	c := *a.voiceTrain.Candidate
	a.voiceTrain.Candidate = nil
	a.addTranscript(fmt.Sprintf("TRAINING CONFIRMED: %s -> %s (%d/%d)", c.Heard, c.Intended, count, voiceAliasConfirmations))
	if count < voiceAliasConfirmations {
		a.voiceTrain.Status = "First confirmation stored; repeat once to activate the same mapping."
		a.voiceTrain.FeedbackClass = "amber"
		a.voiceTrain.Interpretation = "PENDING 1/2 — REPEAT THE SHOWN PHRASE"
		a.refreshVoiceTrainingUI()
		a.actualRespond("First confirmation saved. Repeat: " + voiceExercises[a.voiceTrain.Index%len(voiceExercises)].Expected + ".")
		return
	}
	a.voiceTrain.Status = "CONFIRMED PERSONAL MAPPING ACTIVE: " + c.Heard + " -> " + c.Intended
	a.voiceTrain.FeedbackClass = "green"
	a.voiceTrain.Interpretation = "LEARNED: " + strings.ToUpper(c.Heard) + " → " + strings.ToUpper(c.Intended)
	a.refreshVoiceTrainingUI()
	a.voiceTrainingSkip()
}

func (a *App) voiceTrainingHeard(e RecoEvent) {
	// Called on GUI thread. While calibration is active, NEVER dispatch this
	// final result to Operations. No target or ACTUAL state can be changed.
	n := voiceNormal(e.Text)
	a.voiceTrain.LastRaw = e.Text
	a.voiceTrain.LastConfidence = e.Confidence
	a.lastRecoClass = "neutral"
	a.lastRecoStatus = fmt.Sprintf("CALIBRATION RAW: %q | CONFIDENCE %d%% | NOT SENT TO OPERATIONS", e.Text, e.Confidence)
	setText(a.controls[4105], a.lastRecoStatus)
	setText(a.controls[5051], a.lastRecoStatus)
	a.addTranscript(fmt.Sprintf("TRAINING HEARD: %q (%d%%)", e.Text, e.Confidence))
	state := a.voiceDiagnosticState()
	configured := a.settings.Confidence
	if configured == 0 {
		configured = 80
	}
	// Positive/administrative commands must not be accepted at low confidence.
	if e.Confidence >= configured {
		switch n {
		case "confirm training", "save training", "training confirmed":
			a.writeVoiceDiagnostic("HEARD", e.Text, n, "TRAINING CONFIRM", "TRAINING ONLY", e.Confidence, configured, state, state)
			a.voiceTrainingConfirm()
			return
		case "repeat training", "repeat exercise", "try again training":
			a.writeVoiceDiagnostic("HEARD", e.Text, n, "TRAINING REPEAT", "TRAINING ONLY", e.Confidence, configured, state, state)
			a.voiceTrainingRepeat()
			return
		case "skip training", "skip exercise":
			a.writeVoiceDiagnostic("HEARD", e.Text, n, "TRAINING SKIP", "TRAINING ONLY", e.Confidence, configured, state, state)
			a.voiceTrainingSkip()
			return
		case "stop training", "stop calibration", "finish training":
			a.writeVoiceDiagnostic("HEARD", e.Text, n, "TRAINING STOP", "TRAINING ONLY", e.Confidence, configured, state, state)
			a.voiceTrainingStop()
			return
		}
	}
	if a.voiceTrain.Candidate != nil {
		a.voiceTrain.Status = "Waiting for CONFIRM TRAINING or REPEAT TRAINING."
		a.voiceTrain.FeedbackClass = "pink"
		a.voiceTrain.Interpretation = "REVIEW MAPPING — CONFIRM OR REPEAT"
		a.refreshVoiceTrainingUI()
		a.writeVoiceDiagnostic("HEARD", e.Text, n, "TRAINING AWAITING CONFIRM", "TRAINING ONLY", e.Confidence, configured, state, state)
		return
	}
	if e.Confidence < 65 {
		a.voiceTrain.Status = "Low-confidence practice result. Repeat the complete exercise."
		a.voiceTrain.FeedbackClass = "red"
		a.voiceTrain.Interpretation = "LOW CONFIDENCE — REPEAT THE SHOWN PHRASE"
		a.refreshVoiceTrainingUI()
		a.writeVoiceDiagnostic("HEARD", e.Text, n, "TRAINING LOW CONFIDENCE", "TRAINING ONLY", e.Confidence, 65, state, state)
		return
	}
	exercise := voiceExercises[a.voiceTrain.Index%len(voiceExercises)]
	exact, candidate, reason := voiceCompareExercise(exercise.Expected, e.Text)
	if exact {
		a.voiceTrain.Status = "GREEN: Exact transcription. Advancing to next exercise."
		a.voiceTrain.FeedbackClass = "green"
		a.voiceTrain.Interpretation = "EXACT MATCH — NO ADAPTATION NEEDED"
		a.addTranscript(fmt.Sprintf("TRAINING ACCEPTED: %s — EXACT MATCH (%d%%)", strings.ToUpper(exercise.Expected), e.Confidence))
		a.refreshVoiceTrainingUI()
		a.writeVoiceDiagnostic("HEARD", e.Text, n, "TRAINING EXACT", "TRAINING ONLY", e.Confidence, 65, state, state)
		a.voiceTrainingSkip()
		return
	}
	a.voiceTrain.Candidate = candidate
	a.voiceTrain.Status = reason
	if candidate != nil {
		a.voiceTrain.FeedbackClass = "pink"
		a.voiceTrain.Interpretation = fmt.Sprintf("PROPOSED: %s → %s — CONFIRM OR REPEAT", strings.ToUpper(candidate.Heard), strings.ToUpper(candidate.Intended))
		a.addTranscript(fmt.Sprintf("TRAINING REVIEW: %s → %s — NEEDS CONFIRMATION", strings.ToUpper(candidate.Heard), strings.ToUpper(candidate.Intended)))
	} else {
		a.voiceTrain.FeedbackClass = "amber"
		a.voiceTrain.Interpretation = "NO SAFE MATCH — REPEAT FULL PHRASE"
		a.addTranscript("TRAINING RETRY: " + reason)
	}
	a.refreshVoiceTrainingUI()
	a.writeVoiceDiagnostic("HEARD", e.Text, n, "TRAINING REVIEW: "+reason, "TRAINING ONLY", e.Confidence, 65, state, state)
	if candidate != nil {
		a.actualRespond("Vosk heard " + candidate.Heard + " for " + candidate.Intended + ". Say confirm training to teach RAWDOGS, or repeat training.")
	} else {
		a.actualRespond("Recognition unclear. Please repeat the exercise.")
	}
}

func (a *App) refreshVoiceTrainingUI() {
	if a == nil || a.controls == nil || a.controls[ID_ADAPT_TITLE] == 0 {
		return
	}
	stage := "READY"
	if a.voiceTrain.Active {
		stage = fmt.Sprintf("EXERCISE %d OF %d", a.voiceTrain.Index+1, len(voiceExercises))
	}
	setText(a.controls[ID_ADAPT_TITLE], "CURRENT TRAINING EXERCISE — "+stage)
	setText(a.controls[ID_ADAPT_PROMPT_LABEL], "PIPER REQUESTS — READ AND REPEAT THE WORDS BELOW")
	if a.voiceTrain.Active {
		// Display the spoken text, not only a numeric shorthand: prevents the
		// operator from having to remember Piper's long number sequence.
		setText(a.controls[ID_ADAPT_EXPECTED], strings.ToUpper(voiceExercises[a.voiceTrain.Index%len(voiceExercises)].Expected))
	} else {
		setText(a.controls[ID_ADAPT_EXPECTED], "PRESS START CALIBRATION")
	}
	heard := "VOSK HEARD: ---"
	if a.voiceTrain.LastRaw != "" {
		heard = fmt.Sprintf("VOSK HEARD: %s   (%d%%)", strings.ToUpper(a.voiceTrain.LastRaw), a.voiceTrain.LastConfidence)
	}
	setText(a.controls[ID_ADAPT_HEARD], heard)
	interpreted := a.voiceTrain.Interpretation
	if interpreted == "" {
		interpreted = "WAITING FOR RECOGNITION"
	}
	setText(a.controls[ID_ADAPT_INTERPRETATION], "RAWDOGS INTERPRETS: "+interpreted)
	status := a.voiceTrain.Status
	if status == "" {
		status = "Offline operator adaptation — nothing is learned without confirmation."
	}
	setText(a.controls[ID_ADAPT_STATUS], status)
	setText(a.controls[ID_ADAPT_MODEL], "MODEL: "+a.voiceProfile.Model)
	enabled, pending := 0, 0
	for _, r := range a.voiceProfile.Aliases {
		if r.Confirmations >= voiceAliasConfirmations {
			enabled++
		} else {
			pending++
		}
	}
	setText(a.controls[ID_ADAPT_COUNTS], fmt.Sprintf("OPERATOR: %s | ACTIVE RULES: %d | PENDING: %d", a.voiceProfile.Operator, enabled, pending))
	if h := a.controls[ID_ADAPT_OPERATOR]; h != 0 && windowText(h) == "" {
		setText(h, a.voiceProfile.Operator)
	}
	a.populateVoiceTrainingRules()
}
func (a *App) populateVoiceTrainingRules() {
	h := a.controls[ID_ADAPT_RULE_LIST]
	if h == 0 {
		return
	}
	procSendMessageW.Call(uintptr(h), LB_RESETCONTENT, 0, 0)
	push := func(s string) {
		procSendMessageW.Call(uintptr(h), LB_ADDSTRING, 0, uintptr(unsafe.Pointer(utf16Ptr(s))))
	}
	if a.voiceTrain.ShowCue {
		setText(a.controls[ID_ADAPT_RULES_HEADER], "ORIGINAL RAWDOGS CUE DICTIONARY — REFERENCE ONLY")
		for _, s := range masterCueTrainingRows() {
			push(s)
		}
		return
	}
	setText(a.controls[ID_ADAPT_RULES_HEADER], "PERSONAL PRONUNCIATIONS — CLICK RULE + REMOVE SELECTED")
	if len(a.voiceProfile.Aliases) == 0 {
		push("No learned pronunciations yet. Begin calibration.")
		return
	}
	for _, r := range a.voiceProfile.Aliases {
		progress := "ACTIVE"
		if r.Confirmations < voiceAliasConfirmations {
			progress = fmt.Sprintf("PENDING %d/%d", r.Confirmations, voiceAliasConfirmations)
		}
		push(fmt.Sprintf("%s -> %s | %s | %s", strings.ToUpper(r.Heard), strings.ToUpper(r.Intended), progress, r.Scope))
	}
}
func (a *App) voiceTrainingReferenceToggle() {
	a.voiceTrain.ShowCue = !a.voiceTrain.ShowCue
	a.populateVoiceTrainingRules()
}
func (a *App) voiceTrainingRemoveSelected() {
	if a.voiceTrain.ShowCue {
		a.voiceTrain.Status = "Switch to personal pronunciations to remove an entry."
		a.refreshVoiceTrainingUI()
		return
	}
	h := a.controls[ID_ADAPT_RULE_LIST]
	idx, _, _ := procSendMessageW.Call(uintptr(h), LB_GETCURSEL, 0, 0)
	if idx == ^uintptr(0) || int(idx) >= len(a.voiceProfile.Aliases) {
		a.voiceTrain.Status = "Select a learned pronunciation in the list first."
		a.refreshVoiceTrainingUI()
		return
	}
	old := append([]voiceAlias(nil), a.voiceProfile.Aliases...)
	removed := a.voiceProfile.Aliases[int(idx)]
	a.voiceProfile.Aliases = append(a.voiceProfile.Aliases[:int(idx)], a.voiceProfile.Aliases[int(idx)+1:]...)
	if err := a.saveVoiceProfile(); err != nil {
		a.voiceProfile.Aliases = old
		a.voiceTrain.Status = "Could not save profile removal: " + err.Error()
	} else {
		a.voiceTrain.Status = "Removed personal mapping: " + removed.Heard + " -> " + removed.Intended
	}
	a.refreshVoiceTrainingUI()
}
