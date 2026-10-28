//go:build windows

package main

import (
	"fmt"
	"strings"
)

func (a *App) syncPiperMuteButton() {
	if a.controls[ID_PIPER_MUTE] == 0 {
		return
	}
	if a.settings.PiperMuted {
		setText(a.controls[ID_PIPER_MUTE], "UNMUTE")
	} else {
		setText(a.controls[ID_PIPER_MUTE], "MUTE")
	}
	procInvalidateRect.Call(uintptr(a.controls[ID_PIPER_MUTE]), 0, 1)
}

func (a *App) persistPiperControls() {
	// Avoid falsely marking an unfinished first-run setup as completed.
	if !a.firstRun && !a.guideActive {
		if err := a.saveSettings(); err != nil {
			a.addTranscript("PIPER SETTINGS SAVE FAILED: " + err.Error())
		}
	}
}

func (a *App) stopPiperSpeech(reason string) {
	a.manualNarrating = false
	a.abandonPiperReadbacks()
	a.ttsMu.Lock()
	a.ttsStatus = "SILENT — " + reason
	a.ttsMu.Unlock()
	setText(a.controls[3020], "VOICE STATUS: SILENT — "+reason)
}

func (a *App) setPiperMute(mute bool) {
	a.settings.PiperMuted = mute
	if mute {
		a.stopPiperSpeech("MUTED")
	}
	a.syncPiperMuteButton()
	if !mute {
		setText(a.controls[3020], "VOICE STATUS: READY — UNMUTED")
	}
	a.persistPiperControls()
}

func (a *App) applyPiperCommand(cmd PiperCommand) {
	switch cmd.Kind {
	case "stop":
		a.stopPiperSpeech("OPERATOR STOP")
	case "mute":
		a.setPiperMute(true)
	case "unmute":
		a.setPiperMute(false)
	case "speed", "faster", "slower":
		speed := cmd.Speed
		if cmd.Kind == "faster" {
			speed = stepPiperSpeed(a.settings.VoiceSpeed, 1)
		}
		if cmd.Kind == "slower" {
			speed = stepPiperSpeed(a.settings.VoiceSpeed, -1)
		}
		a.settings.VoiceSpeed = speed
		a.selectComboText(ID_SPEED_COMBO, fmt.Sprintf("%.2fx", speed))
		a.addOperationLog(fmt.Sprintf("PIPER SPEED — %.2fx", speed))
		a.persistPiperControls()
	case "volume", "louder", "quieter":
		volume := cmd.Volume
		if cmd.Kind == "louder" {
			volume = a.settings.VoiceVolume + 10
		}
		if cmd.Kind == "quieter" {
			volume = a.settings.VoiceVolume - 10
		}
		volume = clampPiperVolume(volume)
		a.settings.VoiceVolume = volume
		a.selectComboText(ID_VOLUME_COMBO, piperVolumeLabel(volume))
		if volume > 0 && a.settings.PiperMuted {
			a.settings.PiperMuted = false
			a.syncPiperMuteButton()
		}
		a.addOperationLog(fmt.Sprintf("PIPER VOLUME — %d%%", volume))
		a.persistPiperControls()
	}
	// All commands are silent and don't affect the six-stage coordinator.
	a.addTranscript("PIPER CONTROL: " + strings.ToUpper(cmd.Kind))
}
