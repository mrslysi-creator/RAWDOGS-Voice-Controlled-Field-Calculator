//go:build windows

package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var voiceDiagnosticMu sync.Mutex
var voiceDiagnosticSession string

// Files are append-only within a session. CLEAR BUFFER rotates the session,
// preserving earlier evidence and never affecting recognition or target state.
func (a *App) rotateVoiceDiagnosticLog() {
	voiceDiagnosticMu.Lock()
	voiceDiagnosticSession = ""
	voiceDiagnosticMu.Unlock()
}
func (a *App) voiceDiagnosticState() string {
	return fmt.Sprintf("kind=%s; expected=%s; cue=%s; actualSet=%t", a.pending.Kind, a.expectedCoordinateAxis(), a.pending.CueAwaiting, a.actualSet)
}
func (a *App) writeVoiceDiagnostic(event, raw, interpreted, reason, result string, confidence, threshold int, before, after string) {
	if a.tourDemoMode {
		return /* Examples must never contaminate the operator diagnostic CSV. */
	}
	voiceDiagnosticMu.Lock()
	defer voiceDiagnosticMu.Unlock()
	if voiceDiagnosticSession == "" {
		voiceDiagnosticSession = time.Now().Format("20060102_150405.000000000")
	}
	root := filepath.Join(os.Getenv("LOCALAPPDATA"), "RAWDOGS_Field_Calculator", "diagnostics")
	if strings.TrimSpace(os.Getenv("LOCALAPPDATA")) == "" {
		root = filepath.Join(os.TempDir(), "RAWDOGS_Field_Calculator", "diagnostics")
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		return
	}
	path := filepath.Join(root, "RAWDOGS_VOICE_"+voiceDiagnosticSession+".csv")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		return
	}
	writer := csv.NewWriter(f)
	if stat.Size() == 0 {
		_ = writer.Write([]string{"timestamp", "event", "raw_heard", "interpreted", "interpretation_detail", "result", "confidence_percent", "threshold_percent", "state_before", "state_after"})
	}
	_ = writer.Write([]string{time.Now().Format(time.RFC3339Nano), event, raw, interpreted, reason, result, fmt.Sprint(confidence), fmt.Sprint(threshold), before, after})
	writer.Flush()
}

// Diagnostic-only event stream: uses the same CSV and does not change any control behavior.
func (a *App) writeInteractionDiagnostic(event, detail, result, before, after string) {
	a.writeVoiceDiagnostic(event, "", "", detail, result, 0, 0, before, after)
}
