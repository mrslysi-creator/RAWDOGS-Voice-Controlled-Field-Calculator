//go:build windows

package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// The audible text is generated directly from the visible page blocks,
// NEVER from a separate summary or draft. The handles share the same indices.
func (a *App) narratorTexts(page string) []string {
	if page == "tour" {
		return guidedTourNarration()
	}
	return manualNarrationFromVisible(a.manualSpeechText[page])
}

// Always invoked from the GUI thread, never the synthesis/audio worker.
func (a *App) narratorShowSection() {
	if a == nil {
		return
	}
	a.narratorMu.Lock()
	index := a.narratorIndex
	a.narratorMu.Unlock()
	showIndex := -1
	if a.currentScreen == "manual" {
		showIndex = index
	}
	// Repaint only the old/new static paragraph, not the whole paper.
	oldHighlight := a.manualHighlighted
	a.manualHighlighted = 0
	blocks := a.manualSpeechHandles[a.manualPage]
	if showIndex >= 0 && showIndex < len(blocks) {
		a.manualHighlighted = blocks[showIndex]
	}
	// The 200 ms timer must not repeatedly ShowWindow on unchanged passages;
	// that caused needless redraws and made earlier highlighting flicker.
	if oldHighlight == a.manualHighlighted && showIndex == a.narratorVisibleSection {
		return
	}
	if oldHighlight != a.manualHighlighted {
		if oldHighlight != 0 {
			procInvalidateRect.Call(uintptr(oldHighlight), 0, 1)
		}
		if a.manualHighlighted != 0 {
			procInvalidateRect.Call(uintptr(a.manualHighlighted), 0, 1)
		}
	}
	// The indexed page-four transcript is the source of Piper's exact words.
	if a.currentScreen == "manual" && a.manualPage == "narrated" && showIndex >= 0 && showIndex < len(guidedTour) && showIndex != a.narratorVisibleSection {
		a.selectGuideIndex(showIndex)
	}
	a.narratorVisibleSection = showIndex
	for _, h := range a.manualSections {
		want := uintptr(SW_HIDE)
		if manualPageControlsVisible(a.currentScreen, a.manualPage, "narrated") {
			want = SW_SHOW
		}
		procShowWindow.Call(uintptr(h), want)
	}
}

func (a *App) narratorStatus(msg string) {
	if msg == a.manualPlayerStatusShown {
		return
	}
	a.manualPlayerStatusShown = msg
	setText(a.controls[6255], msg)
	// Update hidden page-four status as well so returning to page four cannot
	// show an earlier page's playback state.
	setText(a.controls[6204], msg)
}
func (a *App) narratorStop(reset bool) {
	a.narratorMu.Lock()
	a.narratorGeneration++
	if a.narratorCancel != nil {
		close(a.narratorCancel)
		a.narratorCancel = nil
	}
	a.narratorRunning = false
	a.narratorPaused = false
	a.manualNarrating = false
	a.narratorFinished = false
	if reset {
		a.narratorPendingSeek = -1
		a.narratorIndex = 0
		a.narratorOffset = 0
		a.narratorElapsed = 0
	}
	a.narratorMu.Unlock()
	a.narratorStatus("AUDIO: STOPPED — PRESS PLAY")
}
func (a *App) narratorPause() {
	a.narratorMu.Lock()
	if a.narratorRunning {
		a.narratorGeneration++
		if a.narratorCancel != nil {
			close(a.narratorCancel)
			a.narratorCancel = nil
		}
		a.narratorOffset = a.narratorElapsed
		a.narratorRunning = false
		a.narratorPaused = true
		a.manualNarrating = false
	}
	a.narratorMu.Unlock()
	a.narratorStatus("AUDIO: PAUSED — PLAY RESUMES POSITION")
}
func (a *App) narratorJump(step int, absolute bool) {
	a.narratorMu.Lock()
	wasPlaying := a.narratorRunning
	a.narratorGeneration++
	if a.narratorCancel != nil {
		close(a.narratorCancel)
		a.narratorCancel = nil
	}
	a.narratorRunning = false
	a.manualNarrating = false
	a.narratorFinished = false
	a.narratorPendingSeek = -1
	if absolute {
		a.narratorIndex = step
	} else {
		a.narratorIndex += step
	}
	n := len(a.narratorTexts(a.manualPage))
	if n <= 0 {
		n = 1
	}
	if a.narratorIndex < 0 {
		a.narratorIndex = 0
	}
	if a.narratorIndex >= n {
		a.narratorIndex = n - 1
	}
	a.narratorOffset = 0
	a.narratorElapsed = 0
	a.narratorMu.Unlock()
	a.narratorShowSection()
	a.narratorStatus(fmt.Sprintf("AUDIO: SECTION %d OF %d — PRESS PLAY", a.narratorIndex+1, n))
	if wasPlaying {
		a.narratorPlay()
	}
}
func (a *App) narratorRestart() {
	a.narratorMu.Lock()
	wasPlaying := a.narratorRunning
	a.narratorMu.Unlock()
	a.narratorJump(0, true)
	if !wasPlaying {
		a.narratorPlay()
	}
}
func (a *App) narratorSeek(pos int) {
	if pos < 0 {
		pos = 0
	}
	if pos > 10000 {
		pos = 10000
	}
	a.narratorMu.Lock()
	n := len(a.narratorTexts(a.manualPage))
	if len(a.narratorDurations) != n || n == 0 {
		// Never claim a guessed section matches an unknown WAV position.
		// Apply this request as soon as measured audio lengths are available.
		a.narratorPendingSeek = pos
		a.narratorMu.Unlock()
		a.narratorStatus("AUDIO: PREPARING AUDIO — SEEK QUEUED")
		return
	}
	wasPlaying := a.narratorRunning
	a.narratorGeneration++
	if a.narratorCancel != nil {
		close(a.narratorCancel)
		a.narratorCancel = nil
	}
	a.narratorRunning = false
	a.manualNarrating = false
	a.narratorFinished = false
	a.narratorIndex, a.narratorOffset = manualSeekPosition(a.narratorDurations, pos)
	a.narratorElapsed = a.narratorOffset
	a.narratorPendingSeek = -1
	a.narratorMu.Unlock()
	a.narratorShowSection()
	a.narratorStatus("AUDIO: SEEK POSITION SELECTED")
	if wasPlaying {
		a.narratorPlay()
	}
}

func (a *App) narratorToggleMute() {
	a.narratorMu.Lock()
	a.manualPlayerMute = !a.manualPlayerMute
	muted := a.manualPlayerMute
	wasPlaying := a.narratorRunning
	if wasPlaying {
		a.narratorGeneration++
		a.narratorOffset = a.narratorElapsed
		if a.narratorCancel != nil {
			close(a.narratorCancel)
			a.narratorCancel = nil
		}
		a.narratorRunning = false
		a.manualNarrating = false
	}
	a.narratorMu.Unlock()
	if h := a.controls[ID_MANUAL_MUTE]; h != 0 {
		procInvalidateRect.Call(uintptr(h), 0, 0)
	} // Icon ONLY.
	if muted {
		a.narratorStatus("AUDIO: MUTED — TIMELINE CONTINUES")
	} else {
		a.narratorStatus("AUDIO: SOUND ON")
	}
	if wasPlaying {
		a.narratorPlay()
	} // Resume same audio frame at new gain, without changing Piper operation settings.
}

func (a *App) narratorPlay() {
	if !a.tourActive && !a.manualSavedToDisk() {
		a.narratorStatus("AUDIO: SAVE SETUP TO DISK BEFORE PLAY")
		return
	}
	if !a.actualVoiceReady() || a.settings.PiperMuted {
		a.narratorStatus("AUDIO: PIPER VOICE NOT READY OR MUTED IN SETTINGS")
		return
	}
	a.captureSettingsFromUI()
	spec, ok := a.activeActualVoice()
	if !ok {
		a.narratorStatus("AUDIO: SELECT A PIPER VOICE")
		return
	}
	model, config, _ := a.piperVoicePaths(spec)
	_, runtimeDir, _ := a.piperBasePaths()
	speed := a.settings.VoiceSpeed
	if speed <= 0 {
		speed = 1
	}
	volume := a.settings.VoiceVolume
	output := a.settings.OutputIndex
	page := a.manualPage
	if a.tourActive {
		page = "tour"
	}
	sections := a.narratorTexts(page)
	if len(sections) == 0 {
		a.narratorStatus("AUDIO: NO TEXT ON THIS PAGE")
		return
	}
	a.narratorMu.Lock()
	if a.narratorRunning {
		a.narratorMu.Unlock()
		return
	}
	if page == "tour" && a.narratorIndex != a.tourIndex {
		a.narratorIndex = a.tourIndex
		a.narratorOffset = 0
		a.narratorElapsed = 0
	}
	if a.narratorPage != page {
		a.narratorPage = page
		a.narratorDurations = nil
	}
	a.narratorGeneration++
	generation := a.narratorGeneration
	cancel := make(chan struct{})
	a.narratorCancel = cancel
	a.narratorRunning = true
	a.narratorPaused = false
	a.manualNarrating = true
	if a.narratorFinished { // Replay current tour chapter, not the entire guide.
		if page == "tour" {
			a.narratorIndex = a.tourIndex
		} else if page != "narrated" {
			a.narratorIndex = 0
		}
		a.narratorOffset = 0
		a.narratorElapsed = 0
	}
	start := a.narratorIndex
	offset := a.narratorOffset
	a.narratorError = ""
	a.narratorFinished = false
	a.narratorMu.Unlock()
	a.narratorStatus("AUDIO: PREPARING PAGE NARRATION...")
	go func() {
		wavs := make([]string, len(sections))
		lengths := make([]time.Duration, len(sections))
		cache := filepath.Join(filepath.Dir(a.configPath), "narration_cache")
		if err := os.MkdirAll(cache, 0755); err != nil {
			a.narratorFailure(generation, err)
			return
		}
		for i, section := range sections {
			// An interactive tour synthesizes and plays ONE requested topic.
			// Earlier/later chapters are not synthesized until selected.
			if (page == "tour" || page == "narrated") && i != start {
				continue
			}
			select {
			case <-cancel:
				return
			default:
			}
			hash := sha256.Sum256([]byte(fmt.Sprintf("%s|%.4f|%s", spec.Key, speed, section)))
			wav := filepath.Join(cache, fmt.Sprintf("%x.wav", hash[:12]))
			if st, err := os.Stat(wav); err != nil || st.Size() < 44 {
				tmp := fmt.Sprintf("%s.%d.tmp.wav", wav, time.Now().UnixNano())
				cmd := hiddenCommand(filepath.Join(runtimeDir, "piper.exe"), "--model", model, "--config", config, "--length_scale", fmt.Sprintf("%.3f", 1/speed), "--output_file", tmp)
				cmd.Stdin = strings.NewReader(section + "\n")
				var buf bytes.Buffer
				cmd.Stdout = &buf
				cmd.Stderr = &buf
				if err := cmd.Start(); err != nil {
					a.narratorFailure(generation, err)
					return
				}
				done := make(chan error, 1)
				go func() { done <- cmd.Wait() }()
				select {
				case err := <-done:
					if err != nil {
						os.Remove(tmp)
						a.narratorFailure(generation, fmt.Errorf("Piper: %s: %w", buf.String(), err))
						return
					}
				case <-cancel:
					_ = cmd.Process.Kill()
					<-done
					os.Remove(tmp)
					return
				}
				if err := os.Rename(tmp, wav); err != nil {
					// Two render jobs may race on the cache name. A valid existing WAV is fine.
					if st, statErr := os.Stat(wav); statErr != nil || st.Size() < 44 {
						os.Remove(tmp)
						a.narratorFailure(generation, err)
						return
					}
					os.Remove(tmp)
				}
			}
			duration, err := manualWAVDuration(wav)
			if err != nil {
				a.narratorFailure(generation, err)
				return
			}
			wavs[i] = wav
			lengths[i] = duration
		}
		a.narratorMu.Lock()
		if generation != a.narratorGeneration {
			a.narratorMu.Unlock()
			return
		}
		a.narratorDurations = lengths
		if a.narratorPendingSeek >= 0 {
			start, offset = manualSeekPosition(lengths, a.narratorPendingSeek)
			a.narratorIndex = start
			a.narratorOffset = offset
			a.narratorElapsed = offset
			a.narratorPendingSeek = -1
		}
		a.narratorMu.Unlock()
		playLimit := len(sections)
		if page == "tour" || page == "narrated" {
			playLimit = start + 1
		}
		for i := start; i < playLimit; i++ {
			select {
			case <-cancel:
				return
			default:
			}
			a.narratorMu.Lock()
			if generation != a.narratorGeneration {
				a.narratorMu.Unlock()
				return
			}
			a.narratorIndex = i
			if i != start {
				a.narratorOffset = 0
				a.narratorElapsed = 0
			}
			muted := a.manualPlayerMute
			a.narratorMu.Unlock()
			startOffset := time.Duration(0)
			if i == start {
				startOffset = offset
			}
			gain := volume
			if muted {
				gain = 0
			}
			err := manualPlayWAVFrom(wavs[i], output, gain, startOffset, cancel, func(t time.Duration) {
				a.narratorMu.Lock()
				if generation == a.narratorGeneration {
					a.narratorElapsed = t
				}
				a.narratorMu.Unlock()
			})
			if err != nil {
				a.narratorFailure(generation, err)
				return
			}
		}
		a.narratorMu.Lock()
		if generation == a.narratorGeneration {
			a.narratorRunning = false
			a.manualNarrating = false
			a.narratorCancel = nil
			if page == "tour" || page == "narrated" {
				a.narratorIndex = start
				a.narratorOffset = lengths[start]
			} else {
				a.narratorIndex = len(sections) - 1
				a.narratorOffset = lengths[len(lengths)-1]
			}
			a.narratorElapsed = a.narratorOffset
			a.narratorFinished = true // Keep seek thumb at 100%, not zero, when page finishes.
		}
		a.narratorMu.Unlock()
	}()
}
func (a *App) narratorFailure(generation uint64, err error) {
	a.narratorMu.Lock()
	if generation == a.narratorGeneration {
		a.narratorRunning = false
		a.manualNarrating = false
		a.narratorCancel = nil
		a.narratorError = err.Error()
	}
	a.narratorMu.Unlock()
}
func (a *App) syncNarratorSeek() { a.narratorTickUI() }
func manualClock(t time.Duration) string {
	if t < 0 {
		t = 0
	}
	sec := int(t / time.Second)
	return fmt.Sprintf("%02d:%02d", sec/60, sec%60)
}

// Timer runs on GUI thread. Never rebuild controls, repaint paper or
// regenerate buttons while following the audio progress.
func (a *App) narratorTickUI() {
	a.narratorMu.Lock()
	index, elapsed := a.narratorIndex, a.narratorElapsed
	running, paused := a.narratorRunning, a.narratorPaused
	finished := a.narratorFinished
	durations := append([]time.Duration(nil), a.narratorDurations...)
	errMsg := a.narratorError
	page := a.narratorPage
	if page == "" {
		page = a.manualPage
	}
	pendingSeek := a.narratorPendingSeek
	a.narratorMu.Unlock()
	a.narratorShowSection()
	n := len(a.narratorTexts(page))
	if n == 0 {
		return
	}
	var total, prior time.Duration
	for i, d := range durations {
		total += d
		if i < index {
			prior += d
		}
	}
	frac := index * 10000 / n
	if page == "tour" && index >= 0 && index < len(durations) && durations[index] > 0 {
		frac = int((int64(index)*10000 + int64(elapsed)*10000/int64(durations[index])) / int64(n))
	} else if total > 0 && len(durations) == n {
		frac = int(int64(prior+elapsed) * 10000 / int64(total))
	}
	if frac < 0 {
		frac = 0
	}
	if frac > 10000 {
		frac = 10000
	}
	if !a.manualSeekDragging && !(pendingSeek >= 0 && total == 0) && frac != a.manualPlayerSeekShown {
		a.manualPlayerSeekShown = frac
		if seek := a.controls[ID_MANUAL_SEEK]; seek != 0 {
			procSendMessageW.Call(uintptr(seek), 0x0405, 1, uintptr(frac))
		}
	}
	if errMsg != "" {
		a.narratorStatus("AUDIO ERROR: " + errMsg)
		return
	}
	status := ""
	if running {
		if total == 0 {
			status = "AUDIO: PREPARING LOCAL PIPER AUDIO..."
		} else {
			status = fmt.Sprintf("AUDIO: %s — %d/%d — %s / %s", strings.ToUpper(page), index+1, n, manualClock(prior+elapsed), manualClock(total))
		}
	} else if paused {
		status = fmt.Sprintf("AUDIO: PAUSED — %d/%d — %s / %s", index+1, n, manualClock(prior+elapsed), manualClock(total))
	} else if finished {
		if page == "tour" {
			status = fmt.Sprintf("AUDIO: GUIDE SECTION %d/%d COMPLETE — PRESS NEXT", index+1, n)
		} else if page == "narrated" {
			status = fmt.Sprintf("AUDIO: GUIDE PASSAGE %d/%d COMPLETE — NEXT FOR MORE", index+1, n)
		} else {
			status = "AUDIO: PAGE COMPLETE — PRESS PLAY TO REPLAY"
		}
	}
	if status != "" {
		a.narratorStatus(status)
	}
}
