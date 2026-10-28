package main

import (
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strings"
)

// The country chooser is navigation, not a promise of model availability.
// All 27 EU countries remain visible even when a provider has no model.
type LibraryCountry struct {
	Name, ISO string
	Languages []string
}

func libraryCountries() []LibraryCountry {
	return []LibraryCountry{
		{"All countries / all languages", "", nil},
		{"Austria", "AT", []string{"de"}}, {"Belgium", "BE", []string{"nl", "fr", "de"}}, {"Bulgaria", "BG", []string{"bg"}},
		{"Croatia", "HR", []string{"hr"}}, {"Cyprus", "CY", []string{"el", "tr"}}, {"Czechia", "CZ", []string{"cs"}},
		{"Denmark", "DK", []string{"da"}}, {"Estonia", "EE", []string{"et"}}, {"Finland", "FI", []string{"fi", "sv"}},
		{"France", "FR", []string{"fr", "br"}}, {"Germany", "DE", []string{"de"}}, {"Greece", "GR", []string{"el"}},
		{"Hungary", "HU", []string{"hu"}}, {"Ireland", "IE", []string{"en", "ga"}}, {"Italy", "IT", []string{"it"}},
		{"Latvia", "LV", []string{"lv"}}, {"Lithuania", "LT", []string{"lt"}}, {"Luxembourg", "LU", []string{"lb", "fr", "de"}},
		{"Malta", "MT", []string{"mt", "en"}}, {"Netherlands", "NL", []string{"nl"}}, {"Poland", "PL", []string{"pl"}},
		{"Portugal", "PT", []string{"pt"}}, {"Romania", "RO", []string{"ro"}}, {"Slovakia", "SK", []string{"sk"}},
		{"Slovenia", "SI", []string{"sl"}}, {"Spain", "ES", []string{"es", "ca"}}, {"Sweden", "SE", []string{"sv"}},
		{"United Kingdom", "GB", []string{"en"}}, {"United States", "US", []string{"en"}},
		{"India", "IN", []string{"en", "hi", "gu", "te"}}, {"Japan", "JP", []string{"ja"}}, {"China", "CN", []string{"zh"}},
		{"Jamaica", "JM", []string{"en"}}, {"Trinidad & Tobago", "TT", []string{"en"}},
		{"Canada", "CA", []string{"en", "fr"}}, {"Australia", "AU", []string{"en"}}, {"New Zealand", "NZ", []string{"en"}},
		{"South Africa", "ZA", []string{"en"}}, {"Brazil", "BR", []string{"pt"}},
		{"Ukraine", "UA", []string{"uk"}}, {"Russia", "RU", []string{"ru"}}, {"Turkey", "TR", []string{"tr"}},
		{"Vietnam", "VN", []string{"vi"}}, {"South Korea", "KR", []string{"ko"}}, {"Philippines", "PH", []string{"tl", "en"}},
		{"United Arab Emirates", "AE", []string{"ar", "en"}}, {"Iran", "IR", []string{"fa"}},
	}
}
func countryFlag(iso string) string {
	iso = strings.ToUpper(strings.TrimSpace(iso))
	if len(iso) != 2 || iso[0] < 'A' || iso[0] > 'Z' || iso[1] < 'A' || iso[1] > 'Z' {
		return "🌐"
	}
	return string([]rune{rune(0x1F1E6) + rune(iso[0]-'A'), rune(0x1F1E6) + rune(iso[1]-'A')})
}
func languageName(code string) string {
	names := map[string]string{
		"en": "English", "de": "German", "fr": "French", "es": "Spanish", "it": "Italian", "pt": "Portuguese", "nl": "Dutch", "pl": "Polish", "sv": "Swedish", "cs": "Czech", "el": "Greek", "hi": "Hindi", "ja": "Japanese", "zh": "Chinese (Mandarin)", "ru": "Russian", "tr": "Turkish", "vi": "Vietnamese", "ca": "Catalan", "uk": "Ukrainian", "ko": "Korean", "ar": "Arabic", "fa": "Persian", "gu": "Gujarati", "te": "Telugu", "tl": "Filipino (Tagalog)", "br": "Breton", "eo": "Esperanto", "bg": "Bulgarian", "hr": "Croatian", "da": "Danish", "et": "Estonian", "fi": "Finnish", "hu": "Hungarian", "ga": "Irish", "lv": "Latvian", "lt": "Lithuanian", "lb": "Luxembourgish", "mt": "Maltese", "ro": "Romanian", "sk": "Slovak", "sl": "Slovenian", "ka": "Georgian", "ky": "Kyrgyz", "kz": "Kazakh", "uz": "Uzbek", "tg": "Tajik"}
	if s := names[code]; s != "" {
		return s
	}
	return strings.ToUpper(code)
}
func voskLanguageAndRegion(folder string) (lang, region string) {
	s := strings.TrimPrefix(folder, "vosk-model-")
	s = strings.TrimPrefix(s, "small-")
	chunks := strings.Split(s, "-")
	if len(chunks) == 0 {
		return "", ""
	}
	lang = chunks[0]
	switch lang {
	case "cn":
		lang = "zh"
		region = "CN"
	case "vn":
		lang = "vi"
		region = "VN"
	case "kz":
		region = "KZ"
	}
	if len(chunks) > 1 {
		regionToken := strings.ToUpper(chunks[1])
		known := map[string]bool{"GB": true, "US": true, "IN": true, "GR": true, "TN": true, "PH": true, "CN": true, "CA": true, "BR": true}
		if known[regionToken] {
			region = regionToken
		}
	}
	return
}
func countryMatches(c LibraryCountry, lang, region string) bool {
	if c.ISO == "" {
		return true
	}
	for _, l := range c.Languages {
		if l == lang {
			return true
		}
	}
	return false
}
func regionPriority(countryISO, region string) int {
	if countryISO != "" && strings.EqualFold(countryISO, region) {
		return 0
	}
	return 1
}

type VoskCatalogEntry struct{ Folder, URL, Language, Region, Size string }

var voskArchivePattern = regexp.MustCompile(`(?i)(?:href\s*=\s*["'])([^"']*vosk-model-[a-z0-9_.-]+\.zip(?:\?[^"']*)?)["']`)

func parseVoskCatalog(page []byte) ([]VoskCatalogEntry, error) {
	found := map[string]bool{}
	out := []VoskCatalogEntry{}
	for _, m := range voskArchivePattern.FindAllSubmatch(page, -1) {
		raw := html.UnescapeString(string(m[1]))
		u, err := url.Parse(raw)
		if err != nil {
			continue
		}
		if u.Host == "" {
			u, err = url.Parse("https://alphacephei.com/vosk/models/" + strings.TrimLeft(raw, "/"))
			if err != nil {
				continue
			}
		}
		if u.Scheme != "https" || !strings.EqualFold(u.Host, "alphacephei.com") {
			continue
		}
		u.RawQuery = ""
		name := strings.TrimSuffix(path.Base(u.Path), ".zip")
		if !strings.HasPrefix(name, "vosk-model-") || found[name] {
			continue
		}
		lang, region := voskLanguageAndRegion(name)
		if lang == "spk" || lang == "" {
			continue
		}
		found[name] = true
		out = append(out, VoskCatalogEntry{Folder: name, URL: u.String(), Language: lang, Region: region})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("official Vosk catalog returned no compatible model archives")
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Folder < out[j].Folder })
	return out, nil
}

type PiperCatalogEntry struct {
	Key, Name, Language, Region, Quality, RepoPath string
	Bytes                                          int64
	Speakers                                       int
}

func parsePiperCatalog(b []byte) ([]PiperCatalogEntry, error) {
	type voice struct {
		Key      string `json:"key"`
		Name     string `json:"name"`
		Language struct {
			Family string `json:"family"`
			Region string `json:"region"`
		} `json:"language"`
		Quality  string `json:"quality"`
		Speakers int    `json:"num_speakers"`
		Files    map[string]struct {
			Size int64 `json:"size_bytes"`
		} `json:"files"`
	}
	all := map[string]voice{}
	if err := json.Unmarshal(b, &all); err != nil {
		return nil, err
	}
	out := []PiperCatalogEntry{}
	for _, v := range all {
		if v.Key == "" || v.Language.Family == "" {
			continue
		}
		for p, f := range v.Files {
			if !strings.HasSuffix(p, ".onnx") || path.Base(p) != v.Key+".onnx" {
				continue
			}
			base := strings.TrimSuffix(p, "/"+v.Key+".onnx")
			if strings.Contains(base, "..") || strings.HasPrefix(base, "/") {
				continue
			}
			if _, ok := v.Files[p+".json"]; !ok {
				continue
			}
			out = append(out, PiperCatalogEntry{Key: v.Key, Name: v.Name, Language: v.Language.Family, Region: v.Language.Region, Quality: v.Quality, RepoPath: base, Bytes: f.Size, Speakers: v.Speakers})
			break
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("Piper voice manifest has no verified model/config pairs")
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out, nil
}
func catalogSize(bytes int64) string {
	if bytes <= 0 {
		return "Size not published"
	}
	if bytes > 1024*1024*1024 {
		return fmt.Sprintf("%.1f GB", float64(bytes)/(1024*1024*1024))
	}
	return fmt.Sprintf("%.0f MB", float64(bytes)/(1024*1024))
}
