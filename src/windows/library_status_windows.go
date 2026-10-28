//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// The Settings model managers show only installed, usable packs. The entire
// public catalog is intentionally displayed only in Field Manual > Libraries.
func (a *App) installedVoskLabels() []string {
	if !a.voskRuntimeReady() {
		return nil
	}
	result := []string{}
	for _, s := range voskModelSpecs() {
		if a.voskModelInstalled(s) {
			result = append(result, s.LibraryLabel)
		}
	}
	return result
}
func (a *App) installedPiperLabels() []string {
	if !a.piperRuntimeReady() {
		return nil
	}
	result := []string{}
	for _, s := range piperVoiceSpecs() {
		if a.piperVoiceInstalled(s) {
			result = append(result, s.Label)
		}
	}
	return result
}

func (a *App) refreshInstalledLibrarySelectors() {
	_, oldV := a.comboSelection(ID_VOSK_LIBRARY_COMBO)
	_, oldP := a.comboSelection(ID_ACTUAL_LIBRARY_COMBO)
	vs, ps := a.installedVoskLabels(), a.installedPiperLabels()
	a.setComboItems(ID_VOSK_LIBRARY_COMBO, vs, -1)
	a.setComboItems(ID_ACTUAL_LIBRARY_COMBO, ps, -1)
	// Preserve the user's uncommitted selection where possible; fall back to
	// the active pack, then the first usable pack. None means no selection.
	vPick, pPick := oldV, oldP
	if !containsLibraryLabel(vs, vPick) {
		if spec, ok := voskSpecByActiveDisplay(a.settings.VoiceName); ok {
			vPick = spec.LibraryLabel
		}
	}
	if !containsLibraryLabel(ps, pPick) {
		if spec, ok := piperSpecByKey(a.settings.ActualVoice); ok {
			pPick = spec.Label
		}
	}
	if !containsLibraryLabel(vs, vPick) && len(vs) > 0 {
		vPick = vs[0]
	}
	if !containsLibraryLabel(ps, pPick) && len(ps) > 0 {
		pPick = ps[0]
	}
	if containsLibraryLabel(vs, vPick) {
		a.selectComboText(ID_VOSK_LIBRARY_COMBO, vPick)
	}
	if containsLibraryLabel(ps, pPick) {
		a.selectComboText(ID_ACTUAL_LIBRARY_COMBO, pPick)
	}
	a.updateVoskLibraryStatus()
	a.updateActualVoiceLibraryStatus()
}

// Compare the ACTIVE pack with the settings file, rather than treating a
// download as a saved preference. Do not modify the user's existing settings.
func (a *App) savedLibraryPackOnDisk(kind, identifier string) bool {
	data, err := os.ReadFile(a.configPath)
	if err != nil {
		return false
	}
	var settings Settings
	if json.Unmarshal(data, &settings) != nil {
		return false
	}
	switch kind {
	case "piper":
		return settings.ActualVoice == identifier
	case "vosk":
		saved, ok := voskSpecByActiveDisplay(settings.VoiceName)
		return ok && saved.Folder == identifier
	}
	return false
}
func (a *App) isLibraryPackSaved(kind, identifier string) bool {
	if !a.savedLibraryPackOnDisk(kind, identifier) {
		return false
	}
	if kind == "piper" {
		return a.settings.ActualVoice == identifier
	}
	if kind == "vosk" {
		active, ok := voskSpecByActiveDisplay(a.settings.VoiceName)
		return ok && active.Folder == identifier
	}
	return false
}

// No Piper requests during Field Manual narration or calibration. Announcements
// occur once per action, never on a timer, and cannot change the voice workflow.
func (a *App) announceLibraryEvent(message string) {
	if !a.actualVoiceReady() || a.settings.PiperMuted || a.manualNarrating || a.voiceTrain.Active {
		return
	}
	a.speakActualAsync(message)
}

func (a *App) libraryAlreadyInstalled(kind, identifier string) {
	name := "Listening model"
	if kind == "piper" {
		name = "Voice"
	}
	if a.isLibraryPackSaved(kind, identifier) {
		a.setLibraryStatus(strings.ToUpper(name) + " ALREADY INSTALLED — ACTIVE AND SAVED")
		a.announceLibraryEvent("This " + strings.ToLower(name) + " is already installed and saved as your active selection.")
	} else {
		a.setLibraryStatus(strings.ToUpper(name) + " ALREADY INSTALLED — SELECT AND SAVE IN SETTINGS")
		a.announceLibraryEvent("This " + strings.ToLower(name) + " is already installed. Open Settings to select and save it.")
	}
}

func (a *App) libraryBeginInstall(kind, name string) {
	// UI thread only. The downloader's worker sends subsequent stage updates.
	noun := "LISTENING MODEL"
	spoken := "Listening model"
	if kind == "piper" {
		noun, spoken = "PIPER VOICE", "Voice"
	}
	a.setLibraryStatus(noun + " — INSTALLING: " + name + "\r\nPLEASE WAIT — DO NOT CLOSE RAWDOGS")
	a.announceLibraryEvent(spoken + " download started. This may take a moment.")
}

func (a *App) libraryFinishInstall(kind, message string, success bool) {
	// UI thread only. Keep existing active settings: installing != activating.
	if success {
		if kind == "piper" {
			a.announceLibraryEvent("Voice installed. Open Settings, choose your voice, click Use Selected, then Save Setup to Disk.")
		} else {
			a.announceLibraryEvent("Listening model installed. Open Settings, choose your model, click Use Selected, then Save Setup to Disk.")
		}
	} else {
		if kind == "piper" {
			a.announceLibraryEvent("Voice installation failed. Your previous voice is unchanged.")
		} else {
			a.announceLibraryEvent("Listening model installation failed. Your previous model is unchanged.")
		}
	}
}

// Reports *real stages* emitted by the PowerShell installer, not synthetic
// percentages. No repaint animation and no spinner. A separate status file is
// used per installer type and removed when the worker finishes.
func (a *App) libraryProgressPath(kind string) string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("rawdogs_%s_install_%d_%d.stage", kind, os.Getpid(), time.Now().UnixNano()))
}
func (a *App) watchLibraryInstallProgress(stagePath, kind string, stopped <-chan struct{}) {
	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()
	previous := ""
	for {
		select {
		case <-stopped:
			return
		case <-ticker.C:
			raw, err := os.ReadFile(stagePath)
			if err != nil {
				continue
			}
			stage := strings.TrimSpace(strings.TrimPrefix(string(raw), "\ufeff"))
			if stage == "" || stage == previous {
				continue
			}
			previous = stage
			a.libraryStateMu.Lock()
			a.libraryStageMessage = kind + " — " + stage
			a.libraryStateMu.Unlock()
			procPostMessageW.Call(uintptr(a.hwnd), WM_APP_LIBRARY_STAGE, 0, 0)
		}
	}
}
func (a *App) applyLibraryInstallProgress() {
	a.libraryStateMu.Lock()
	message := a.libraryStageMessage
	a.libraryStateMu.Unlock()
	if message == "" {
		return
	}
	label := strings.ToUpper(message)
	a.setLibraryStatus("INSTALLING — " + label + "\r\nPLEASE WAIT — ACTIVE PACK IS UNCHANGED")
	if strings.HasPrefix(message, "vosk") {
		setText(a.controls[3010], "LIBRARY STATUS: "+label)
	} else {
		setText(a.controls[3020], "VOICE STATUS: "+label)
	}
}
