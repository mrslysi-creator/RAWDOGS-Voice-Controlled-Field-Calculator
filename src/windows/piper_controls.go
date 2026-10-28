package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

type PiperCommand struct {
	Kind   string
	Speed  float64
	Volume int
}

var piperSpeeds = []float64{0.75, 0.85, 1.00, 1.15, 1.30, 1.50, 1.60, 1.70, 1.80, 1.90, 2.00}
var piperSpeedOptions = []string{"0.75x", "0.85x", "1.00x", "1.15x", "1.30x", "1.50x", "1.60x", "1.70x", "1.80x", "1.90x", "2.00x"}
var piperVolumeOptions = []string{"0%", "10%", "20%", "30%", "40%", "50%", "60%", "70%", "80%", "90%", "100%"}

func piperWords(raw string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	s = strings.NewReplacer("-", " ", "’", "'", ",", " ", "!", " ", "?", " ", "%", " ").Replace(s)
	return strings.Join(strings.Fields(s), " ")
}

func spokenVolume(raw string) (int, bool) {
	words := strings.Fields(piperWords(raw))
	if len(words) == 0 {
		return 0, false
	}
	if len(words) == 1 {
		if n, e := strconv.Atoi(words[0]); e == nil && n >= 0 && n <= 100 && n%10 == 0 {
			return n, true
		}
	}
	units := map[string]int{"zero": 0, "ten": 10, "twenty": 20, "thirty": 30, "forty": 40, "fifty": 50, "sixty": 60, "seventy": 70, "eighty": 80, "ninety": 90, "hundred": 100}
	if len(words) == 1 {
		if n, ok := units[words[0]]; ok {
			return n, true
		}
	}
	if len(words) == 2 && words[0] == "one" && words[1] == "hundred" {
		return 100, true
	}
	return 0, false
}

func spokenSpeed(raw string) (float64, bool) {
	n := piperWords(raw)
	n = strings.TrimSuffix(n, "x")
	n = strings.TrimSpace(n)
	n = strings.ReplaceAll(n, "one", "1")
	n = strings.ReplaceAll(n, "two", "2")
	n = strings.ReplaceAll(n, "zero", "0")
	n = strings.ReplaceAll(n, "oh", "0")
	n = strings.ReplaceAll(n, "five", "5")
	n = strings.ReplaceAll(n, "six", "6")
	n = strings.ReplaceAll(n, "seven", "7")
	n = strings.ReplaceAll(n, "eight", "8")
	n = strings.ReplaceAll(n, "nine", "9")
	n = strings.ReplaceAll(n, "point", ".")
	n = strings.ReplaceAll(n, " ", "")
	f, err := strconv.ParseFloat(n, 64)
	if err != nil {
		return 0, false
	}
	for _, v := range piperSpeeds {
		if math.Abs(f-v) < 0.001 {
			return v, true
		}
	}
	return 0, false
}

// Only Piper-prefixed, fully recognized phrases trigger audio controls. A bare
// "faster" or "stop" must not accidentally interrupt the six-step process.
func parsePiperCommand(raw string) (PiperCommand, bool) {
	n := piperWords(raw)
	if !strings.HasPrefix(n, "piper ") {
		return PiperCommand{}, false
	}
	args := strings.TrimSpace(strings.TrimPrefix(n, "piper "))
	switch args {
	case "faster", "slower", "louder", "quieter", "mute", "unmute", "stop":
		return PiperCommand{Kind: args}, true
	}
	if strings.HasPrefix(args, "speed ") {
		if speed, ok := spokenSpeed(strings.TrimPrefix(args, "speed ")); ok {
			return PiperCommand{Kind: "speed", Speed: speed}, true
		}
	}
	if strings.HasPrefix(args, "volume ") {
		if vol, ok := spokenVolume(strings.TrimPrefix(args, "volume ")); ok {
			return PiperCommand{Kind: "volume", Volume: vol}, true
		}
	}
	return PiperCommand{}, false
}

func stepPiperSpeed(current float64, dir int) float64 {
	if dir > 0 {
		for _, v := range piperSpeeds {
			if v > current+0.001 {
				return v
			}
		}
		return piperSpeeds[len(piperSpeeds)-1]
	}
	for i := len(piperSpeeds) - 1; i >= 0; i-- {
		if piperSpeeds[i] < current-0.001 {
			return piperSpeeds[i]
		}
	}
	return piperSpeeds[0]
}

func clampPiperVolume(n int) int {
	if n < 0 {
		return 0
	}
	if n > 100 {
		return 100
	}
	return n
}
func piperVolumeLabel(v int) string { return fmt.Sprintf("%d%%", clampPiperVolume(v)) }

// G18E Tactical mode: successful steps are already communicated by steady
// green/magenta outlines, so don't make Piper narrate routine coordinates.
// Errors, correction requests, and completed firing solutions still speak.
func routinePiperConfirmation(t string) bool {
	t = strings.ToLower(strings.TrimSpace(t))
	if strings.HasPrefix(t, "roger.") && !strings.Contains(t, "repeat ") {
		return true
	}
	if strings.HasPrefix(t, "y ") && (strings.Contains(t, "x-ray") || strings.Contains(t, "coordinates complete")) {
		return true
	}
	if t == "user actual confirmed." || t == "unsaved target confirmed." {
		return true
	}
	if strings.HasPrefix(t, "target ") && strings.HasSuffix(t, " confirmed.") {
		return true
	}
	return false
}
