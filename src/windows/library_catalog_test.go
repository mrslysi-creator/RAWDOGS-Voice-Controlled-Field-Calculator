package main

import (
	"strings"
	"testing"
)

func TestCountriesAllEUAndFlag(t *testing.T) {
	eu := []string{"AT", "BE", "BG", "HR", "CY", "CZ", "DK", "EE", "FI", "FR", "DE", "GR", "HU", "IE", "IT", "LV", "LT", "LU", "MT", "NL", "PL", "PT", "RO", "SK", "SI", "ES", "SE"}
	seen := map[string]bool{}
	for _, c := range libraryCountries() {
		seen[c.ISO] = true
	}
	for _, iso := range eu {
		if !seen[iso] {
			t.Fatal("missing EU", iso)
		}
	}
	if countryFlag("GB") != "🇬🇧" || countryFlag("invalid") != "🌐" {
		t.Fatal("emoji flag mismatch")
	}
}
func TestCatalogParsers(t *testing.T) {
	page := []byte(`<a href="https://alphacephei.com/vosk/models/vosk-model-small-en-in-0.4.zip">India</a> <a href="https://evil.invalid/vosk-model-small-en-us-0.15.zip">bad</a>`)
	xs, err := parseVoskCatalog(page)
	if err != nil || len(xs) != 1 || xs[0].Language != "en" || xs[0].Region != "IN" {
		t.Fatalf("Vosk %+v %v", xs, err)
	}
	js := `{"en_GB-alba-medium":{"key":"en_GB-alba-medium","name":"alba","language":{"family":"en","region":"GB"},"quality":"medium","num_speakers":1,"files":{"en/en_GB/alba/medium/en_GB-alba-medium.onnx":{"size_bytes":63201294},"en/en_GB/alba/medium/en_GB-alba-medium.onnx.json":{"size_bytes":4500}}}}`
	ys, err := parsePiperCatalog([]byte(js))
	if err != nil || len(ys) != 1 || ys[0].RepoPath != "en/en_GB/alba/medium" {
		t.Fatalf("Piper %+v %v", ys, err)
	}
	if !countryMatches(libraryCountries()[0], "en", "") {
		t.Fatal("all must match")
	}
	if !strings.HasPrefix(countryFlag("JP"), "🇯") {
		t.Fatal("flag")
	}
}
func TestRejectEmptyCatalog(t *testing.T) {
	if _, err := parseVoskCatalog([]byte("empty")); err == nil {
		t.Fatal("no results")
	}
	if _, err := parsePiperCatalog([]byte("{}")); err == nil {
		t.Fatal("no results")
	}
}
