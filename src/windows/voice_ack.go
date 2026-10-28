package main

import "strings"

// The operational Piper guide announces ONLY the next required action.
// Vosk recognition, coordinate acceptance, firing calculations and the
// six-stage transaction engine are unchanged. The successful target Y step
// speaks the existing concise firing solution separately, not a duplicate
// confirmation. Standalone X-RAY/YANKEE cues already acknowledge through
// the on-screen active state, so don't add more audio chatter.
func sixStepAcknowledgement(stage, kind, axis, spokenValue string) string {
	kind = strings.ToLower(strings.TrimSpace(kind))
	axis = strings.ToUpper(strings.TrimSpace(axis))
	subject := "Actual"
	if kind == "target" {
		subject = "Target"
	}
	switch stage {
	case "activate":
		return subject + " X-ray next."
	case "axis":
		return "" // The operator has already said this cue; avoid double narration.
	case "coordinate":
		if axis == "X" {
			return subject + " Yankee next."
		}
	case "committed":
		if axis == "Y" && kind != "target" {
			return "Actual confirmed. New target next."
		}
	}
	return ""
}
