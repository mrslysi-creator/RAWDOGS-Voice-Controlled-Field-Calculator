package main

import "strings"

// explicitConfirmBody splits only a complete voice phrase ending in the
// optional short "CONFIRM" suffix. Numeric parsing remains strict/independent.
func explicitConfirmBody(s string) (string, bool) {
	n := strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(s))), " ")
	if !strings.HasSuffix(n, " confirm") {
		return "", false
	}
	body := strings.TrimSpace(strings.TrimSuffix(n, " confirm"))
	return body, body != ""
}

// correctionAxis selects the current replacement, or the latest accepted
// coordinate, or the currently expected coordinate if nothing yet accepted.
func correctionAxis(replacing, last, expected string) string {
	for _, s := range []string{replacing, last, expected} {
		if s == "X" || s == "Y" {
			return s
		}
	}
	return ""
}
