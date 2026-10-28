package main

// Library sources used by Field Manual page 6. The application downloads
// selected models only through the existing Settings managers.
const (
	voskCatalogURL  = "https://alphacephei.com/vosk/models"
	piperCatalogURL = "https://huggingface.co/rhasspy/piper-voices/tree/v1.0.0"
)

func isApprovedLibraryCatalog(url string) bool {
	return url == voskCatalogURL || url == piperCatalogURL
}
