package main

import "strings"

// normaliseActivationCommand recognises a small, explicit set of likely
// transcriptions of the two multi-word activation commands. It is invoked
// only when no transaction is in progress. This avoids reinterpreting spoken
// coordinate digits or game/team chat as an unrelated restart.
func normaliseActivationCommand(n string, pendingKind string, actualSet bool) (string, bool) {
	if pendingKind != "" {
		return n, false
	}
	n = strings.Join(strings.Fields(n), " ")
	var options []string
	canonical := "user actual"
	if !actualSet {
		options = []string{
			"user actual", "users actual", "use are actual", "use your actual",
			"use actual", "usa actual", "easier actual", "your actual",
			"set user actual", "you are actual", "use a actual",
		}
	} else {
		canonical = "new target"
		options = []string{
			"new target", "new targets", "new tar get", "new tar git",
			"new targ it", "new targe it", "new target is", "start new target",
			"new target please", "new target now",
		}
	}
	for _, v := range options {
		if n == v {
			return canonical, n != canonical
		}
		// Preserve the earlier complete one-shot "USER ACTUAL X ... Y ... OVER"
		// feature, including variants of the opening words. Never promote a
		// random sentence: a coordinate axis must immediately follow the cue.
		if strings.HasPrefix(n, v+" ") {
			rest := strings.TrimSpace(strings.TrimPrefix(n, v+" "))
			if strings.HasPrefix(rest, "x ") || rest == "x" || strings.HasPrefix(rest, "set x ") {
				return canonical + " " + rest, n != canonical+" "+rest
			}
		}
	}
	return n, false
}

// normaliseTargetRecallAlias translates only explicit, COMPLETE, numbered
// target-recall phrases into the existing saved-target command. These are
// observed Vosk near-misses of the user's UK-English "recall target".
// It never infers a target number or clears stored target data.
// A complete numbered recall is permitted during unfinished coordinate entry.
func normaliseTargetRecallAlias(raw string) (string, bool) {
	n := strings.ToLower(raw)
	n = strings.ReplaceAll(n, "’", "'")
	n = strings.ReplaceAll(n, "'", " ")
	n = strings.ReplaceAll(n, "-", " ")
	n = strings.Join(strings.Fields(n), " ")
	// Deliberate full-word form: "TARGET TWO" is a global saved-slot recall,
	// not a request to restart the current coordinate entry.
	if strings.HasPrefix(n, "target ") {
		value := strings.TrimPrefix(n, "target ")
		if _, ok := targetNumberWord(value); ok {
			return "recall target " + value, true
		}
	}
	options := []string{
		"recall target ", "recal target ", "re call target ",
		"we call target ", "we target ", "regal target ",
		"revert target ", "river target ", "we at target ",
		"with their target ", "with thier target ",
	}
	for _, prefix := range options {
		if !strings.HasPrefix(n, prefix) {
			continue
		}
		value := strings.TrimPrefix(n, prefix)
		switch value {
		case "1", "one", "2", "two", "too", "3", "three", "4", "four":
			canonical := "recall target " + value
			return canonical, n != canonical
		}
	}
	return n, false
}

// The coordinate axis by itself is a cue, NOT a coordinate. It arms the
// recogniser to accept a later digits-only utterance after a natural pause.
// Only canonical 'x' or 'y' is passed here, after context normalisation.
func loneAxisCue(canonical string) string {
	switch strings.TrimSpace(canonical) {
	case "x":
		return "X"
	case "y":
		return "Y"
	default:
		return ""
	}
}

// Keep numeric recognition at the configured confidence. Relax only the
// two-word activation cues and an already-expected X-RAY/YANKEE cue, to stop
// short command phrases being rejected while more reliable digits pass.
// Never lower below 65%, never override a user threshold below 65%.
func confidenceForVoiceCue(configured int, idleActivation bool, expectedAxisCue bool) int {
	if configured <= 0 {
		configured = 80
	}
	if (idleActivation || expectedAxisCue) && configured > 65 {
		configured -= 15
		if configured < 65 {
			configured = 65
		}
	}
	return configured
}

// sixStepActivation turns short UK-English recognition variants into the two
// explicit stages. Exact matches only: no fuzzy substring matching of game
// chatter or coordinates. "actually" is useful when Vosk misses "actual".
func sixStepActivation(text string) (string, bool) {
	n := strings.Join(strings.Fields(strings.ToLower(text)), " ")
	switch n {
	case "refresh actual", "refresh the actual", "reset actual", "actual", "actually",
		"user actual", "users actual", "use are actual", "use your actual", "use actual",
		"usa actual", "easier actual", "your actual", "set user actual",
		"you are actual", "use a actual", "actaula", "actuals":
		return "refresh actual", true
	case "refresh target", "refresh the target", "reset target", "new target",
		"new targets", "new tar get", "new tar git", "new targ it", "new targe it",
		"new target is", "start new target", "new target please", "new target now":
		return "refresh target", true
	}
	return n, false
}

// expectedSixStepCue is deliberately state-constrained: "yank" and other UK
// near-misses are YANKEE only when Y is expected, never in ordinary speech.
// Context words ACTUAL/TARGET are accepted before the axis, e.g.
// "ACTUAL YANK 110 point 37" or "TARGET X-RAY 120 point 50".
func expectedSixStepCue(text, kind, expected string) string {
	n := strings.Join(strings.Fields(strings.ToLower(text)), " ")
	if kind == "actual" || kind == "switch_actual" {
		for _, pre := range []string{"actual ", "actually ", "user actual ", "refresh actual "} {
			if strings.HasPrefix(n, pre) {
				n = strings.TrimPrefix(n, pre)
				break
			}
		}
	} else if kind == "target" {
		for _, pre := range []string{"target ", "new target ", "refresh target "} {
			if strings.HasPrefix(n, pre) {
				n = strings.TrimPrefix(n, pre)
				break
			}
		}
	}
	words := strings.Fields(n)
	if len(words) == 0 {
		return n
	}
	if expected == "Y" {
		// Whole-phrase near misses produced by different Vosk English models.
		for _, a := range []string{"yank key", "yank e", "yank he", "yank y",
			"young key", "young he", "yan key", "yang key", "y an key"} {
			if n == a {
				return "y"
			}
			if strings.HasPrefix(n, a+" ") {
				return "y" + n[len(a):]
			}
		}
		switch words[0] {
		case "y", "why", "yank", "yankee", "yankie", "yanky", "yanki", "yankey", "yanqui", "yankee's", "yonky", "yangky":
			words[0] = "y"
		}
	} else if expected == "X" {
		if len(words) >= 2 && (words[0] == "ex" || words[0] == "axe") && (words[1] == "ray" || words[1] == "rey" || words[1] == "re") {
			return "x" + n[len(words[0])+len(words[1])+1:]
		}
		switch words[0] {
		case "x", "xray", "exray", "xrey", "exrey", "ex", "axe":
			words[0] = "x"
		}
	}
	return strings.Join(words, " ")
}

// Only the command we are currently waiting for gets a relaxed threshold.
// Digits still need a parseable value AND an already-established axis cue.
// Destructive commands retain the user's configured threshold.
func sixStepThreshold(configured int, stage string) int {
	if configured <= 0 {
		configured = 80
	}
	limit := configured
	switch stage {
	case "activation", "axis":
		if limit > 40 {
			limit = 40
		}
	case "value":
		if limit > 55 {
			limit = 55
		}
	case "recall":
		if limit > 65 {
			limit = 65
		}
	case "piper":
		// Explicit Piper-prefix commands are reversible volume/speed/stop
		// controls only; never interpreted as coordinate/clear actions.
		if limit > 65 {
			limit = 65
		}
	}
	return limit
}

// targetNumberWord / parseRecallTargetPhrase are pure, unit-testable command
// rules. Exactly one spoken target number is required; no guessing.
func targetNumberWord(value string) (int, bool) {
	switch value {
	case "1", "one":
		return 0, true
	case "2", "two", "too":
		return 1, true
	case "3", "three":
		return 2, true
	case "4", "four":
		return 3, true
	}
	return -1, false
}
func parseRecallTargetPhrase(phrase string) (int, bool) {
	n := strings.Join(strings.Fields(strings.ToLower(phrase)), " ")
	for _, prefix := range []string{"recall target ", "use target ", "target "} {
		if strings.HasPrefix(n, prefix) {
			return targetNumberWord(strings.TrimPrefix(n, prefix))
		}
	}
	return -1, false
}

// A complete explicit cue in a Vosk partial can STOP old Piper immediately,
// but MUST NOT perform any state change until Vosk delivers its final result
// and a real confidence threshold is satisfied. Partial results have no %.
func likelyBargeInPartial(raw string) bool {
	n := strings.Join(strings.Fields(strings.ToLower(raw)), " ")
	switch n {
	case "new target", "refresh target", "refresh actual", "correction", "piper stop":
		return true
	}
	if _, ok := parseRecallTargetPhrase(n); ok {
		return true
	}
	if norm, changed := normaliseTargetRecallAlias(n); changed {
		_, ok := parseRecallTargetPhrase(norm)
		return ok
	}
	return false
}
