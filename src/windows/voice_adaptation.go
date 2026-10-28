package main

// v2.43.13: Offline, operator- and Vosk-model-scoped pronunciation adaptation.
// This is an interpretation layer, NOT Vosk acoustic retraining. Nothing here
// changes existing confidence thresholds, the six-step state machine or maths.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const voiceProfileVersion = 1
const voiceAliasConfirmations = 2

// Learn only highly contextual coordinate-digit and X/Y cue variants.
// Neither type may override an existing command or invent missing digits.
type voiceAlias struct {
	Heard         string `json:"heard"`
	Intended      string `json:"intended"`
	Scope         string `json:"scope"`
	Confirmations int    `json:"confirmations"`
	UpdatedUTC    string `json:"updated_utc"`
}

type voiceProfile struct {
	Version  int          `json:"version"`
	Operator string       `json:"operator"`
	Model    string       `json:"vosk_model"`
	Aliases  []voiceAlias `json:"aliases"`
}

type voiceCandidate struct{ Heard, Intended, Scope string }

type voiceExercise struct{ Display, Expected string }

var voiceExercises = []voiceExercise{
	{"YANKEE 35.71", "yankee three five point seven one"},
	{"X-RAY 36.02", "x ray three six point zero two"},
	{"YANKEE 34.60", "yankee three four point six zero"},
	{"X-RAY 95.24", "x ray nine five point two four"},
	{"YANKEE 48.30", "yankee four eight point three zero"},
	{"X-RAY 73.25", "x ray seven three point two five"},
	{"YANKEE 63.18", "yankee six three point one eight"},
	{"X-RAY 26.49", "x ray two six point four nine"},
}

var voiceNonwordRE = regexp.MustCompile(`[^a-z0-9.]+`)

func voiceNormal(s string) string {
	s = strings.ToLower(strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(s, "’", "'"), "-", " ")))
	s = voiceNonwordRE.ReplaceAllString(s, " ")
	return strings.Join(strings.Fields(s), " ")
}
func voiceNormalAxis(s string) string {
	s = " " + voiceNormal(s) + " "
	// Exact bounded replacements; these are transcription comparisons only.
	for _, kv := range [][2]string{{"x ray", "x"}, {"xray", "x"}, {"yankee", "y"}, {"yankie", "y"}, {"yanky", "y"}} {
		s = strings.ReplaceAll(s, " "+kv[0]+" ", " "+kv[1]+" ")
	}
	return strings.Join(strings.Fields(s), " ")
}

var voiceDigits = map[string]bool{
	"zero": true, "oh": true, "one": true, "two": true, "too": true,
	"three": true, "free": true, "tree": true, "four": true, "for": true,
	"five": true, "six": true, "seven": true, "eight": true, "ate": true, "nine": true,
	"0": true, "1": true, "2": true, "3": true, "4": true, "5": true,
	"6": true, "7": true, "8": true, "9": true,
}
var voiceCanonicalDigits = map[string]bool{
	"zero": true, "one": true, "two": true, "three": true, "four": true,
	"five": true, "six": true, "seven": true, "eight": true, "nine": true,
}

func voiceEquivalentDigitAlias(heard, intended string) bool {
	return map[string]string{"free": "three", "tree": "three", "too": "two", "for": "four", "ate": "eight", "oh": "zero"}[heard] == intended
}

var voiceProtectedWords = map[string]bool{
	"x": true, "y": true, "point": true, "dot": true, "confirm": true,
	"cancel": true, "correction": true, "actual": true, "target": true,
	"refresh": true, "clear": true, "restart": true, "save": true,
	"start": true, "stop": true, "training": true, "repeat": true, "skip": true,
}

// Training is anchored to a known spoken exercise. Only one exact mismatch
// (a digit word or the X/Y cue) may be proposed. Multiword X/Y mishearings
// are supported when all following coordinate words are exactly aligned.
// A missing number NEVER becomes a training rule.
func voiceCompareExercise(expected, heard string) (bool, *voiceCandidate, string) {
	a, b := strings.Fields(voiceNormalAxis(expected)), strings.Fields(voiceNormalAxis(heard))
	if len(a) < 4 || (a[0] != "x" && a[0] != "y") || len(b) == 0 {
		return false, nil, "Not a coordinate exercise. Repeat."
	}
	if len(b) == len(a)+1 && len(b) >= 2 {
		eq := true
		for i := 1; i < len(a); i++ {
			if a[i] != b[i+1] {
				eq = false
				break
			}
		}
		if eq && validVoiceAxisAlias(b[:2], a[0]) {
			c := &voiceCandidate{Heard: strings.Join(b[:2], " "), Intended: a[0], Scope: "axis_" + a[0]}
			return false, c, fmt.Sprintf("Vosk heard %q in place of %s-RAY/YANKEE. Confirm or repeat.", c.Heard, strings.ToUpper(a[0]))
		}
	}
	if len(a) != len(b) {
		return false, nil, "Word count differs. No guessing of missing or extra digits. Repeat."
	}
	diff := -1
	for i := range a {
		if a[i] != b[i] {
			if diff >= 0 {
				return false, nil, "More than one word differs. Repeat."
			}
			diff = i
		}
	}
	if diff < 0 {
		return true, nil, "Exact match. No adaptation needed."
	}
	if diff == 0 {
		if validVoiceAxisAlias(b[:1], a[0]) {
			c := &voiceCandidate{Heard: b[0], Intended: a[0], Scope: "axis_" + a[0]}
			return false, c, fmt.Sprintf("Vosk heard %q instead of the expected axis cue. Confirm or repeat.", c.Heard)
		}
		return false, nil, "Ambiguous or protected X/Y cue. Repeat."
	}
	if !voiceCanonicalDigits[a[diff]] || (voiceDigits[b[diff]] && !voiceEquivalentDigitAlias(b[diff], a[diff])) || voiceProtectedWords[b[diff]] || b[diff] == "" {
		return false, nil, "Ambiguous digit recognition. No unsafe substitution. Repeat."
	}
	c := &voiceCandidate{Heard: b[diff], Intended: a[diff], Scope: "coordinate_digit"}
	return false, c, fmt.Sprintf("Vosk heard %q where the prompted digit was %q. Confirm or repeat.", c.Heard, c.Intended)
}
func validVoiceAxisAlias(words []string, axis string) bool {
	if axis != "x" && axis != "y" || len(words) < 1 || len(words) > 2 {
		return false
	}
	for _, w := range words {
		if w == "" || len(w) > 24 || voiceDigits[w] || voiceProtectedWords[w] || strings.ContainsAny(w, ".\\/:") {
			return false
		}
	}
	return true
}
func validVoiceCandidate(c voiceCandidate) bool {
	if c.Heard == "" || len(c.Heard) > 50 || c.Heard != voiceNormal(c.Heard) {
		return false
	}
	if c.Scope == "coordinate_digit" {
		return voiceCanonicalDigits[c.Intended] && len(strings.Fields(c.Heard)) == 1 &&
			!voiceProtectedWords[c.Heard] && (!voiceDigits[c.Heard] || voiceEquivalentDigitAlias(c.Heard, c.Intended))
	}
	if c.Scope == "axis_x" && c.Intended == "x" || c.Scope == "axis_y" && c.Intended == "y" {
		return validVoiceAxisAlias(strings.Fields(c.Heard), c.Intended)
	}
	return false
}
func (p *voiceProfile) confirmCandidate(c voiceCandidate, now time.Time) (int, error) {
	if p == nil || !validVoiceCandidate(c) {
		return 0, errors.New("unsafe voice-alias candidate")
	}
	for i := range p.Aliases {
		r := &p.Aliases[i]
		if r.Scope != c.Scope || r.Heard != c.Heard {
			continue
		}
		if r.Intended != c.Intended {
			return 0, errors.New("conflict with existing pronunciation: remove it first")
		}
		if r.Confirmations < voiceAliasConfirmations {
			r.Confirmations++
		}
		r.UpdatedUTC = now.UTC().Format(time.RFC3339)
		return r.Confirmations, nil
	}
	p.Aliases = append(p.Aliases, voiceAlias{Heard: c.Heard, Intended: c.Intended, Scope: c.Scope, Confirmations: 1, UpdatedUTC: now.UTC().Format(time.RFC3339)})
	return 1, nil
}
func validVoiceNumberSequence(words []string, p *voiceProfile) bool {
	points, known := 0, 0
	for _, word := range words {
		if word == "point" || word == "dot" {
			points++
			continue
		}
		if voiceDigits[word] {
			known++
			continue
		}
		found := false
		for _, r := range p.Aliases {
			if r.Scope == "coordinate_digit" && r.Confirmations >= voiceAliasConfirmations && r.Heard == word {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return points == 1 && known >= 2
}

// Only interpret in an ALREADY active six-step coordinate stage. A saved
// mispronunciation is usable only if an X/Y cue and a complete decimal-digit
// sequence are present. No word is inserted and no number is inferred.
func (p *voiceProfile) adaptCoordinate(raw, expectedAxis string, armed bool) (string, string) {
	if p == nil || (expectedAxis != "X" && expectedAxis != "Y") {
		return raw, ""
	}
	tokens := strings.Fields(voiceNormalAxis(raw))
	if len(tokens) < 3 {
		return raw, ""
	}
	axis := strings.ToLower(expectedAxis)
	// Safe two-utterance input: once the operator has explicitly armed X-RAY
	// or YANKEE, a complete value-only decimal may use confirmed digit rules.
	// Without an armed cue the identical words are never promoted to data.
	if armed && validVoiceNumberSequence(tokens, p) {
		changes := []string{}
		for i := range tokens {
			for _, rule := range p.Aliases {
				if rule.Scope == "coordinate_digit" && rule.Confirmations >= voiceAliasConfirmations && tokens[i] == rule.Heard {
					tokens[i] = rule.Intended
					changes = append(changes, rule.Heard+" -> "+rule.Intended)
					break
				}
			}
		}
		if len(changes) > 0 {
			return strings.Join(tokens, " "), "ARMED PERSONAL VALUE PROFILE (" + strings.Join(changes, ", ") + ")"
		}
	}
	pos := 0
	if tokens[0] == "target" || tokens[0] == "actual" {
		pos = 1
	}
	if pos >= len(tokens) {
		return raw, ""
	}
	converted := []string{}
	// No global alias. Match an operator-confirmed cue only at the exact
	// coordinate-cue position, and only for the axis expected by the state.
	if tokens[pos] != axis {
		for _, rule := range p.Aliases {
			if rule.Scope != "axis_"+axis || rule.Intended != axis || rule.Confirmations < voiceAliasConfirmations {
				continue
			}
			aliasWords := strings.Fields(rule.Heard)
			if pos+len(aliasWords) >= len(tokens) {
				continue
			}
			same := true
			for i, word := range aliasWords {
				if tokens[pos+i] != word {
					same = false
					break
				}
			}
			if !same || !validVoiceNumberSequence(tokens[pos+len(aliasWords):], p) {
				continue
			}
			tokens = append(append([]string{}, tokens[:pos]...), append([]string{axis}, tokens[pos+len(aliasWords):]...)...)
			converted = append(converted, rule.Heard+" -> "+axis)
			break
		}
	}
	if pos >= len(tokens) || tokens[pos] != axis {
		return raw, ""
	}
	start := pos + 1
	if !validVoiceNumberSequence(tokens[start:], p) {
		return raw, ""
	}
	for i := start; i < len(tokens); i++ {
		for _, rule := range p.Aliases {
			if rule.Scope == "coordinate_digit" && rule.Confirmations >= voiceAliasConfirmations && tokens[i] == rule.Heard {
				tokens[i] = rule.Intended
				converted = append(converted, rule.Heard+" -> "+rule.Intended)
				break
			}
		}
	}
	if len(converted) == 0 {
		return raw, ""
	}
	// For explicitly cued phrases the state machine still decides acceptance.
	return strings.Join(tokens, " "), "CONFIRMED PERSONAL VOICE PROFILE (" + strings.Join(converted, ", ") + ")"
}

func voiceProfileLoad(path, operator, model string) (voiceProfile, error) {
	base := voiceProfile{Version: voiceProfileVersion, Operator: operator, Model: model}
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return base, nil
	}
	if err != nil {
		return base, err
	}
	var p voiceProfile
	if err = json.Unmarshal(b, &p); err != nil {
		return base, err
	}
	if p.Version != voiceProfileVersion || p.Operator != operator || p.Model != model {
		return base, errors.New("voice profile schema/operator/model mismatch")
	}
	if len(p.Aliases) > 256 {
		return base, errors.New("voice profile exceeds rule limit")
	}
	// Profile files are user data: validate before applying; never allow a
	// malformed/edited file to override operational commands or digits.
	found := map[string]bool{}
	good := make([]voiceAlias, 0, len(p.Aliases))
	for _, r := range p.Aliases {
		c := voiceCandidate{Heard: r.Heard, Intended: r.Intended, Scope: r.Scope}
		if !validVoiceCandidate(c) || r.Confirmations < 1 || r.Confirmations > voiceAliasConfirmations || found[c.Scope+":"+c.Heard] {
			continue
		}
		found[c.Scope+":"+c.Heard] = true
		good = append(good, r)
	}
	p.Aliases = good
	return p, nil
}
func voiceProfileSave(path string, p voiceProfile) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err = os.WriteFile(tmp, b, 0600); err != nil {
		return err
	}
	// On Windows rename over an existing file may fail. We favor keeping the
	// old profile intact on failure, rather than deleting it before rename.
	if err = os.Rename(tmp, path); err != nil {
		// Existing target: direct write as fallback after tmp is complete.
		if e := os.WriteFile(path, b, 0600); e != nil {
			return e
		}
		_ = os.Remove(tmp)
	}
	return nil
}
