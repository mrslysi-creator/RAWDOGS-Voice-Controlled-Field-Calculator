package main

import "testing"

func TestLibrarySelectionOnlyExistingEntries(t *testing.T) {
	labels := []string{"Piper — Alba", "Vosk UK Small"}
	if !containsLibraryLabel(labels, "piper — alba") {
		t.Fatal("case insensitive selected entry should survive")
	}
	if containsLibraryLabel(labels, "Piper — US Large") {
		t.Fatal("uninstalled entry should not match")
	}
	if containsLibraryLabel(nil, "Piper — Alba") {
		t.Fatal("empty installed list must not select a default")
	}
}
