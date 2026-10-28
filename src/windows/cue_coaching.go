package main

import (
	"strings"
	"time"
)

// cueCoach is advisory only. It does not parse, accept, replace, or commit
// any coordinates. It is kept separate from the frozen six-stage transaction
// engine and requires two consecutive *recognisable cue attempts* before
// Piper offers a short reminder. Ordinary game chat never counts as an attempt.
type cueCoach struct {
	stage       string
	failures    int
	lastFailure time.Time
	lastHint    time.Time
}

func (c *cueCoach) reset() {
	c.stage = ""
	c.failures = 0
	c.lastFailure = time.Time{}
}

// cueAttempt reports only explicit axis cues and, when an axis was already
// armed by X-RAY/YANKEE, number-like phrases. It must not infer a voice cue
// from stray words in game chat or from unknown Vosk transcripts.
func cueAttempt(interpreted string, axisArmed bool) bool {
	words := strings.Fields(strings.ToLower(strings.TrimSpace(interpreted)))
	if len(words) == 0 {
		return false
	}
	if words[0] == "x" || words[0] == "y" {
		if len(words) == 1 {
			return true
		} // a deliberate stand-alone axis cue
		words = words[1:]
		if len(words) > 0 && (words[0] == "is" || words[0] == "equals" || words[0] == "equal") {
			words = words[1:]
		}
		if len(words) == 0 {
			return true
		}
		// A cue followed by plausible coordinate words is an attempted entry.
		// Don't interpret "x is where the action is" as an operator error.
		return coordinateWordPhrase(words)
	}
	return axisArmed && coordinateWordPhrase(words)
}

func coordinateWordPhrase(words []string) bool {
	if len(words) == 0 {
		return false
	}
	// With an already-armed axis, a numeric-sounding phrase may be a bad
	// coordinate, e.g. "seven forty" after POINT. Require the entire phrase
	// to comprise coordinate words; never coach on ordinary commentary.
	allowed := map[string]bool{
		"zero": true, "oh": true, "one": true, "two": true, "too": true, "three": true,
		"four": true, "for": true, "five": true, "six": true, "seven": true, "eight": true,
		"ate": true, "nine": true, "ten": true, "eleven": true, "twelve": true,
		"thirteen": true, "fourteen": true, "fifteen": true, "sixteen": true,
		"seventeen": true, "eighteen": true, "nineteen": true,
		"twenty": true, "thirty": true, "forty": true, "fifty": true,
		"sixty": true, "seventy": true, "eighty": true, "ninety": true,
		"hundred": true, "thousand": true, "point": true, "dot": true,
	}
	for _, word := range words {
		if allowed[word] {
			continue
		}
		numeric := true
		for _, r := range word {
			if !(r >= '0' && r <= '9') && r != '.' {
				numeric = false
				break
			}
		}
		if !numeric {
			return false
		}
	}
	return true
}

// note is called once per FINAL Vosk utterance. accepted means the coordinate
// was saved or the correct axis cue was armed. Each different axis is a new
// stage, so stale failures cannot leak into the next step.
func (c *cueCoach) note(stage string, attempted, accepted bool, confidence int, now time.Time) bool {
	if stage == "" || accepted {
		c.reset()
		return false
	}
	if stage != c.stage {
		c.stage = stage
		c.failures = 0
		c.lastFailure = time.Time{}
	}
	if !attempted || confidence < 65 {
		c.failures = 0
		return false
	}
	if !c.lastFailure.IsZero() && now.Sub(c.lastFailure) > 45*time.Second {
		c.failures = 0
	}
	c.lastFailure = now
	c.failures++
	if c.failures < 2 || (!c.lastHint.IsZero() && now.Sub(c.lastHint) < 30*time.Second) {
		return false
	}
	c.failures = 0
	c.lastHint = now
	return true
}

func cueCoachPrompt(kind, axis string, cueAlreadyArmed bool) string {
	subject := "Actual"
	if kind == "target" {
		subject = "Target"
	}
	if cueAlreadyArmed {
		return subject + " " + axis + " coordinate still needed. Say the digits clearly, one at a time after point."
	}
	cue := "X-ray"
	if axis == "Y" {
		cue = "Yankee"
	}
	return subject + " " + cue + " required. Say " + cue + ", then your " + axis + " coordinate."
}
