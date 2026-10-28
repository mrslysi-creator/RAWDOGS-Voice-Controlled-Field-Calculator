package main

import "testing"

func TestLibraryCatalogLinksApproved(t *testing.T) {
	for _, u := range []string{voskCatalogURL, piperCatalogURL} {
		if !isApprovedLibraryCatalog(u) {
			t.Fatalf("official link not approved: %s", u)
		}
	}
	for _, u := range []string{"http://alphacephei.com/vosk/models", "https://example.com", "https://alphacephei.com/vosk/models/../", ""} {
		if isApprovedLibraryCatalog(u) {
			t.Fatalf("unlisted link was approved: %s", u)
		}
	}
}
