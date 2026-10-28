//go:build windows

package main

import (
	"fmt"
)

// The UI only calculates from two confirmed coordinate pairs. A provisional
// target is never displayed as a confirmed solution.
func (a *App) currentL81Solution() (L81Solution, string) {
	if !a.actualSet {
		return L81Solution{}, "AWAITING ACTUAL"
	}
	if a.pending.Kind == "actual" || a.pending.Kind == "switch_actual" {
		return L81Solution{}, "ACTUAL CORRECTION PENDING"
	}
	if a.pending.Kind == "target" {
		return L81Solution{}, "TARGET ENTRY PENDING"
	}
	var tx, ty string
	switch {
	case a.activeTarget == -2 && a.unsavedExists:
		tx, ty = a.unsaved.X, a.unsaved.Y
	case a.activeTarget >= 0 && a.activeTarget < len(a.targets) && a.targets[a.activeTarget].Confirmed:
		tx, ty = a.targets[a.activeTarget].X, a.targets[a.activeTarget].Y
	default:
		return L81Solution{}, "AWAITING TARGET"
	}
	s, err := solveL81(a.actualX, a.actualY, tx, ty)
	if err != nil {
		return L81Solution{}, "INVALID COORDINATES"
	}
	return s, "READY"
}

func (a *App) firingSolutionDisplay() (string, string) {
	s, status := a.currentL81Solution()
	if status != "READY" {
		return "Bearing: ---°\r\nRange: --- m\r\nMIL: ---\r\n\r\nSTATUS: " + status, ""
	}
	if s.HasMil {
		return s.displayText(), "green"
	}
	return s.displayText(), "red"
}

func (a *App) announceCurrentSolution() {
	s, status := a.currentL81Solution()
	if status != "READY" {
		return
	}
	if !s.HasMil {
		a.addOperationLog(fmt.Sprintf("L81 OUTSIDE CALIBRATION — %.0f m (80–684 m only); bearing %.0f°", s.Range, s.Bearing))
	} else {
		a.addOperationLog(fmt.Sprintf("L81 SOLUTION — BEARING %03d° / RANGE %d m / %d MIL", s.bearingRounded(), s.rangeRounded(), s.milRounded()))
	}
	a.actualRespond(s.spokenSolution())
}

func (a *App) repeatFiringSolution(which string) {
	s, status := a.currentL81Solution()
	if status != "READY" {
		a.actualRespond("Solution unavailable. " + status + ".")
		return
	}
	switch which {
	case "repeat bearing":
		a.actualRespond(fmt.Sprintf("Bearing %03d.", s.bearingRounded()))
	case "repeat range":
		a.actualRespond(fmt.Sprintf("Range %d.", s.rangeRounded()))
	case "repeat mil", "repeat mills":
		if s.HasMil {
			a.actualRespond(fmt.Sprintf("Mil %d.", s.milRounded()))
		} else {
			a.actualRespond("Mil unavailable. Target outside verified L eighty one range.")
		}
	default:
		a.actualRespond(s.spokenSolution())
	}
}
