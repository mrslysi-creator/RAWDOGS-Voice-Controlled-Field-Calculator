//go:build windows

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

const piperVoiceManifestURL = "https://huggingface.co/rhasspy/piper-voices/resolve/v1.0.0/voices.json"

var (
	libraryRegistryMu      sync.RWMutex
	libraryRegisteredVosk  []VoskModelSpec
	libraryRegisteredPiper []PiperVoiceSpec
	mciCommand             = syscall.NewLazyDLL("winmm.dll").NewProc("mciSendStringW")
	messageBox             = syscall.NewLazyDLL("user32.dll").NewProc("MessageBoxW")
)

type librarySavedCatalog struct {
	Vosk  []VoskCatalogEntry  `json:"vosk"`
	Piper []PiperCatalogEntry `json:"piper"`
}

func browserCatalogPath(configPath string) string {
	return filepath.Join(filepath.Dir(configPath), "libraries", "catalog_v1.json")
}
func (a *App) loadLibraryCatalogCache() {
	b, err := os.ReadFile(browserCatalogPath(a.configPath))
	if err != nil {
		return
	}
	var catalog librarySavedCatalog
	if json.Unmarshal(b, &catalog) == nil {
		a.registerCatalog(catalog)
	}
}
func (a *App) registerCatalog(cat librarySavedCatalog) {
	vs := make([]VoskModelSpec, 0, len(cat.Vosk))
	ps := make([]PiperVoiceSpec, 0, len(cat.Piper))
	for _, item := range cat.Vosk {
		if !safeVoskCatalogEntry(item) {
			continue
		}
		name := languageName(item.Language)
		if item.Region != "" {
			name += " (" + item.Region + ")"
		}
		vs = append(vs, VoskModelSpec{Key: item.Folder, Folder: item.Folder, URL: item.URL, Language: fmt.Sprintf("%s — Vosk offline", name), LibraryLabel: fmt.Sprintf("%s %s — %s", countryFlag(item.Region), name, item.Folder), ActiveDisplay: "Vosk — " + name + " — " + item.Folder + " — OFFLINE"})
	}
	for _, item := range cat.Piper {
		if !safePiperCatalogEntry(item) {
			continue
		}
		label := fmt.Sprintf("%s Piper — %s — %s (%s)", countryFlag(item.Region), item.Name, languageName(item.Language), item.Quality)
		ps = append(ps, PiperVoiceSpec{Key: item.Key, Label: label, ModelBase: item.Key, RepoPath: item.RepoPath})
	}
	libraryRegistryMu.Lock()
	libraryRegisteredVosk = vs
	libraryRegisteredPiper = ps
	libraryRegistryMu.Unlock()
}
func safeVoskCatalogEntry(x VoskCatalogEntry) bool {
	if !strings.HasPrefix(x.Folder, "vosk-model-") || strings.ContainsAny(x.Folder, "/\\'\";: ") || x.Language == "" {
		return false
	}
	return x.URL == "https://alphacephei.com/vosk/models/"+x.Folder+".zip"
}
func safePiperCatalogEntry(x PiperCatalogEntry) bool {
	if x.Key == "" || x.RepoPath == "" || x.Language == "" || strings.Contains(x.Key, "..") || strings.ContainsAny(x.Key, "/\\'\";: ") || strings.Contains(x.RepoPath, "..") || strings.ContainsAny(x.RepoPath, "'\";: ") {
		return false
	}
	return strings.HasPrefix(x.RepoPath, x.Language+"/") && !strings.HasPrefix(x.RepoPath, "/")
}
func fetchOfficialCatalog(url string, limit int64) ([]byte, error) {
	client := http.Client{Timeout: 25 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "RAWDOGS-Field-Calculator/2.43.19")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("catalog HTTP %d", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > limit {
		return nil, errors.New("catalog exceeded maximum expected size")
	}
	return b, nil
}
func (a *App) refreshLibraryCatalogAsync(force bool) {
	a.libraryStateMu.Lock()
	if a.libraryCatalogBusy || (!force && a.libraryCatalogLoaded) {
		a.libraryStateMu.Unlock()
		return
	}
	a.libraryCatalogBusy = true
	a.libraryCatalogMessage = "SEARCHING OFFICIAL VOSK / PIPER CATALOGS — THIS MAY TAKE A MOMENT"
	a.libraryStateMu.Unlock()
	a.setLibraryStatus("SEARCHING OFFICIAL VOSK / PIPER CATALOGS — THIS MAY TAKE A MOMENT")
	go func() {
		// Each provider can fail independently. Preserve the previous cache when offline.
		var cat librarySavedCatalog
		if b, err := os.ReadFile(browserCatalogPath(a.configPath)); err == nil {
			_ = json.Unmarshal(b, &cat)
		}
		notes := []string{}
		vb, ve := fetchOfficialCatalog(voskCatalogURL, 2*1024*1024)
		if ve == nil {
			if entries, e := parseVoskCatalog(vb); e == nil {
				cat.Vosk = entries
				notes = append(notes, fmt.Sprintf("Vosk: %d models", len(entries)))
			} else {
				ve = e
			}
		}
		if ve != nil {
			notes = append(notes, "Vosk unavailable: using saved options")
		}
		pb, pe := fetchOfficialCatalog(piperVoiceManifestURL, 3*1024*1024)
		if pe == nil {
			if entries, e := parsePiperCatalog(pb); e == nil {
				cat.Piper = entries
				notes = append(notes, fmt.Sprintf("Piper: %d voices", len(entries)))
			} else {
				pe = e
			}
		}
		if pe != nil {
			notes = append(notes, "Piper unavailable: using saved options")
		}
		if len(cat.Vosk) > 0 || len(cat.Piper) > 0 {
			if b, e := json.MarshalIndent(cat, "", "  "); e == nil {
				target := browserCatalogPath(a.configPath)
				_ = os.MkdirAll(filepath.Dir(target), 0755)
				tmp := target + ".tmp"
				if os.WriteFile(tmp, b, 0644) == nil {
					if e := os.Rename(tmp, target); e != nil { // Windows may not replace an existing destination.
						_ = os.Remove(target)
						_ = os.Rename(tmp, target)
					}
				}
			}
			a.registerCatalog(cat)
		}
		a.libraryStateMu.Lock()
		a.libraryCatalogMessage = strings.Join(notes, "  |  ")
		a.libraryCatalogLoaded = true
		a.libraryCatalogBusy = false
		a.libraryStateMu.Unlock()
		procPostMessageW.Call(uintptr(a.hwnd), WM_APP_LIBRARY_DONE, 0, 0)
	}()
}
func (a *App) applyLibraryCatalogResult() {
	// The GUI thread owns all Win32 controls.
	oldV, oldP := a.settings.VoiceName, a.settings.ActualVoice
	a.refreshInstalledLibrarySelectors()
	if item, ok := voskSpecByActiveDisplay(oldV); ok && a.voskModelInstalled(item) {
		a.selectComboText(ID_VOSK_LIBRARY_COMBO, item.LibraryLabel)
	}
	if item, ok := piperSpecByKey(oldP); ok && a.piperVoiceInstalled(item) {
		a.selectComboText(ID_ACTUAL_LIBRARY_COMBO, item.Label)
	}
	a.refreshLibraryBrowserList()
	a.updateVoskLibraryStatus()
	a.updateActualVoiceLibraryStatus()
	a.libraryStateMu.Lock()
	message := a.libraryCatalogMessage
	a.libraryStateMu.Unlock()
	a.setLibraryStatus(message)
}
func (a *App) buildLibraryBrowser(screen string, left, ww int) {
	a.libraryCountries = libraryCountries()
	a.libraryMode = "vosk"
	addPaper := func(id, x, y, w, h int, text string, bold bool) {
		var hw syscall.Handle
		if id == 0 {
			hw = a.addScreenStatic(screen, x, y, w, h, text, bold)
		} else {
			hw = a.addScreenStaticWithID(screen, id, x, y, w, h, text, bold)
		}
		a.manualPaper[hw] = true
		a.manualLibraries = append(a.manualLibraries, hw)
		// Keep visible Field Manual prose as the only narration source. Dynamic
		// results/details are live controls, not fixed text Piper should read.
		if id == 0 {
			a.manualPageText["libraries"] = append(a.manualPageText["libraries"], text)
			if len(a.manualPageText["libraries"]) > 2 {
				a.manualSpeechText["libraries"] = append(a.manualSpeechText["libraries"], text)
				a.manualSpeechHandles["libraries"] = append(a.manualSpeechHandles["libraries"], hw)
			}
		}
	}
	addPaper(0, left, 254, ww, 30, "LIBRARIES — FIND YOUR LISTENER OR SPEAKING VOICE", true)
	addPaper(0, left, 286, ww, 22, "PAGE 6 OF 6", false)
	addPaper(0, left, 311, ww, 55, "Choose your country flag; search the official models, and install any compatible pack you like.\r\nInstalled packs remain available in SETTINGS; SAVE SETUP TO DISK keeps your active choices.", false)
	addPaper(0, left, 374, ww, 24, "1. CHOOSE YOUR COUNTRY / REGION — FLAGS REMAIN VISIBLE", true)
	country := a.addScreenCombo(screen, ID_LIBRARY_COUNTRY, left, 405, ww, 380)
	a.manualLibraries = append(a.manualLibraries, country)
	// Emoji font in dropdown so Windows does not display ISO letter placeholders.
	a.setFont(country, syscall.Handle(createFont(-18, 400, "Segoe UI Emoji")))
	labels := make([]string, 0, len(a.libraryCountries))
	for _, c := range a.libraryCountries {
		labels = append(labels, countryFlag(c.ISO)+"  "+c.Name)
	}
	a.setComboItems(ID_LIBRARY_COUNTRY, labels, 0)
	gap := 12
	half := (ww - gap) / 2
	for _, b := range []struct {
		id, x, y, w, h int
		txt            string
	}{
		{ID_LIBRARY_VOSK_MODE, left, 452, half, 36, "🎙 VOSK — LISTENING"},
		{ID_LIBRARY_PIPER_MODE, left + half + gap, 452, half, 36, "🔊 PIPER — SPEAKING"},
	} {
		a.manualLibraries = append(a.manualLibraries, a.addScreenButton(screen, b.id, b.x, b.y, b.w, b.h, b.txt))
	}
	list := a.createControl(ID_LIBRARY_CHOICES, "LISTBOX", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_VSCROLL|LBS_NOINTEGRALHEIGHT|LBS_OWNERDRAWFIXED|LBS_HASSTRINGS|LBS_NOTIFY, left, 498, ww, 160, 0, screen)
	a.setFont(list, a.fontNormal)
	procSendMessageW.Call(uintptr(list), LB_SETITEMHEIGHT, 0, 27)
	a.manualLibraries = append(a.manualLibraries, list)
	addPaper(0, left, 669, ww, 26, "2. SELECT A MODEL — THE PANEL BELOW EXPLAINS THE CHOICE", true)
	addPaper(6350, left, 700, ww, 77, "Select a model to see its language, installation status and compatibility.", false)
	gap2 := 10
	triple := (ww - 2*gap2) / 3
	for _, b := range []struct {
		id, x, y, w, h int
		txt            string
	}{
		{ID_LIBRARY_REFRESH, left, 791, triple, 34, "REFRESH OFFICIAL LISTS"},
		{ID_LIBRARY_PREVIEW, left + triple + gap2, 791, triple, 34, "▶ PREVIEW VOICE"},
		{ID_LIBRARY_INSTALL, left + 2*(triple+gap2), 791, ww - 2*(triple+gap2), 34, "INSTALL SELECTED"},
		{ID_LIBRARY_REMOVE, left, 833, triple, 34, "REMOVE INSTALLED"},
		{ID_LIBRARY_SETTINGS, left + triple + gap2, 833, ww - triple - gap2, 34, "OPEN SETTINGS — USE / SAVE"},
	} {
		a.manualLibraries = append(a.manualLibraries, a.addScreenButton(screen, b.id, b.x, b.y, b.w, b.h, b.txt))
	}
	addPaper(6351, left, 877, ww, 53, "READY — Select a country, then choose Vosk or Piper.", false)
	a.refreshLibraryBrowserList()
}
func (a *App) onLibraryPageShown() {
	a.refreshLibraryBrowserList()
	a.refreshLibraryCatalogAsync(false)
}
func (a *App) selectedLibraryCountry() LibraryCountry {
	idx, _ := a.comboSelection(ID_LIBRARY_COUNTRY)
	if idx >= 0 && idx < len(a.libraryCountries) {
		return a.libraryCountries[idx]
	}
	return LibraryCountry{}
}
func (a *App) selectedLibraryBrowserEntry() (VoskModelSpec, PiperVoiceSpec, bool) {
	list := a.controls[ID_LIBRARY_CHOICES]
	if list == 0 {
		return VoskModelSpec{}, PiperVoiceSpec{}, false
	}
	idx, _, _ := procSendMessageW.Call(uintptr(list), 0x0188, 0, 0) // LB_GETCURSEL
	if int32(idx) < 0 {
		return VoskModelSpec{}, PiperVoiceSpec{}, false
	}
	if a.libraryMode == "piper" {
		if int(idx) < len(a.libraryShownPiper) {
			return VoskModelSpec{}, a.libraryShownPiper[int(idx)], true
		}
	}
	if a.libraryMode == "vosk" {
		if int(idx) < len(a.libraryShownVosk) {
			return a.libraryShownVosk[int(idx)], PiperVoiceSpec{}, true
		}
	}
	return VoskModelSpec{}, PiperVoiceSpec{}, false
}
func (a *App) refreshLibraryBrowserList() {
	list := a.controls[ID_LIBRARY_CHOICES]
	if list == 0 {
		return
	}
	c := a.selectedLibraryCountry()
	oldV, oldP, oldOk := a.selectedLibraryBrowserEntry()
	procSendMessageW.Call(uintptr(list), LB_RESETCONTENT, 0, 0)
	a.libraryShownVosk = nil
	a.libraryShownPiper = nil
	if a.libraryMode == "piper" {
		for _, s := range piperVoiceSpecs() {
			lang, region := piperKeyLanguageRegion(s.Key)
			if countryMatches(c, lang, region) {
				a.libraryShownPiper = append(a.libraryShownPiper, s)
			}
		}
		sort.SliceStable(a.libraryShownPiper, func(i, j int) bool {
			_, ri := piperKeyLanguageRegion(a.libraryShownPiper[i].Key)
			_, rj := piperKeyLanguageRegion(a.libraryShownPiper[j].Key)
			pi, pj := regionPriority(c.ISO, ri), regionPriority(c.ISO, rj)
			if pi != pj {
				return pi < pj
			}
			return a.libraryShownPiper[i].Label < a.libraryShownPiper[j].Label
		})
		for _, s := range a.libraryShownPiper {
			label := s.Label
			if a.piperVoiceInstalled(s) {
				label += "  ✓ INSTALLED"
				if a.settings.ActualVoice == s.Key {
					label += " — ACTIVE"
				}
			}
			procSendMessageW.Call(uintptr(list), LB_ADDSTRING, 0, uintptr(unsafe.Pointer(utf16Ptr(label))))
		}
	} else {
		for _, s := range voskModelSpecs() {
			lang, region := voskLanguageAndRegion(s.Folder)
			if countryMatches(c, lang, region) {
				a.libraryShownVosk = append(a.libraryShownVosk, s)
			}
		}
		sort.SliceStable(a.libraryShownVosk, func(i, j int) bool {
			_, ri := voskLanguageAndRegion(a.libraryShownVosk[i].Folder)
			_, rj := voskLanguageAndRegion(a.libraryShownVosk[j].Folder)
			pi, pj := regionPriority(c.ISO, ri), regionPriority(c.ISO, rj)
			if pi != pj {
				return pi < pj
			}
			return a.libraryShownVosk[i].Folder < a.libraryShownVosk[j].Folder
		})
		for _, s := range a.libraryShownVosk {
			lang, reg := voskLanguageAndRegion(s.Folder)
			label := fmt.Sprintf("%s %s — %s", countryFlag(reg), languageName(lang), s.Folder)
			if a.voskModelInstalled(s) {
				label += "  ✓ INSTALLED"
				if a.settings.VoiceName == s.ActiveDisplay {
					label += " — ACTIVE"
				}
			}
			procSendMessageW.Call(uintptr(list), LB_ADDSTRING, 0, uintptr(unsafe.Pointer(utf16Ptr(label))))
		}
	}
	selected := 0
	if oldOk {
		if a.libraryMode == "piper" {
			for i, x := range a.libraryShownPiper {
				if x.Key == oldP.Key {
					selected = i
					break
				}
			}
		} else {
			for i, x := range a.libraryShownVosk {
				if x.Folder == oldV.Folder {
					selected = i
					break
				}
			}
		}
	}
	if (a.libraryMode == "piper" && len(a.libraryShownPiper) > 0) || (a.libraryMode == "vosk" && len(a.libraryShownVosk) > 0) {
		procSendMessageW.Call(uintptr(list), 0x0186, uintptr(selected), 0)
	} // LB_SETCURSEL
	a.updateLibraryBrowserDetails()
}
func piperKeyLanguageRegion(key string) (lang, region string) {
	parts := strings.SplitN(key, "-", 2)
	if len(parts) > 0 {
		lr := strings.SplitN(parts[0], "_", 2)
		lang = lr[0]
		if len(lr) > 1 {
			region = lr[1]
		}
	}
	return
}
func (a *App) setLibraryStatus(s string) { setText(a.controls[6351], s) }
func (a *App) updateLibraryBrowserDetails() {
	c := a.selectedLibraryCountry()
	v, p, ok := a.selectedLibraryBrowserEntry()
	if !ok {
		setText(a.controls[6350], "No matches for "+c.Name+" in the available catalog. Change the flag or try All countries.\r\nSome countries have no dedicated model. Never assume a flag guarantees one.")
		return
	}
	if a.libraryMode == "piper" {
		lang, region := piperKeyLanguageRegion(p.Key)
		status := "AVAILABLE TO INSTALL"
		if a.piperVoiceInstalled(p) {
			status = "INSTALLED"
			if a.settings.ActualVoice == p.Key {
				if a.isLibraryPackSaved("piper", p.Key) {
					status = "ACTIVE — SAVED"
				} else {
					status = "ACTIVE — SAVE REQUIRED"
				}
			}
		}
		setText(a.controls[6350], fmt.Sprintf("%s %s speaking voice: %s\r\n%s | %s | %s. PREVIEW plays a published sample where available.\r\n%s — Activate and save in Settings.", countryFlag(region), languageName(lang), p.Key, region, p.RepoPath, status, "NOTE: Non-English voices do not translate RAWDOGS' English prompts."))
	} else {
		lang, reg := voskLanguageAndRegion(v.Folder)
		status := "AVAILABLE TO INSTALL"
		if a.voskModelInstalled(v) && a.voskRuntimeReady() {
			status = "INSTALLED"
			if a.activeVoskSpec().Folder == v.Folder {
				if a.isLibraryPackSaved("vosk", v.Folder) {
					status = "ACTIVE — SAVED"
				} else {
					status = "ACTIVE — SAVE REQUIRED"
				}
			}
		}
		setText(a.controls[6350], fmt.Sprintf("%s %s listening model (%s)\r\n%s  |  %s\r\n%s. Download size/memory varies; foreign-language commands are not yet translated.", countryFlag(reg), languageName(lang), v.Folder, v.Folder, status, "Activation and save are managed in Settings"))
	}
}
func (a *App) installLibraryBrowserSelection() {
	v, p, ok := a.selectedLibraryBrowserEntry()
	if !ok {
		a.setLibraryStatus("SELECT A MODEL FIRST")
		return
	}
	if a.libraryMode == "piper" {
		if a.piperVoiceInstalled(p) && a.piperRuntimeReady() {
			a.libraryAlreadyInstalled("piper", p.Key)
			return
		}
		a.piperMu.Lock()
		busy := a.piperInstallRunning
		a.piperMu.Unlock()
		if busy {
			a.setLibraryStatus("A PIPER INSTALLATION IS ALREADY IN PROGRESS")
			return
		}
		a.installPiperVoiceAsync(p)
	}
	if a.libraryMode == "vosk" {
		if a.voskModelInstalled(v) && a.voskRuntimeReady() {
			a.libraryAlreadyInstalled("vosk", v.Folder)
			return
		}
		a.voskMu.Lock()
		busy := a.voskInstallRunning
		a.voskMu.Unlock()
		if busy {
			a.setLibraryStatus("A VOSK INSTALLATION IS ALREADY IN PROGRESS")
			return
		}
		a.installVoskModelAsync(v)
	}
}
func (a *App) removeLibraryBrowserSelection() {
	a.voskMu.Lock()
	vb := a.voskInstallRunning
	a.voskMu.Unlock()
	a.piperMu.Lock()
	pb := a.piperInstallRunning
	a.piperMu.Unlock()
	if vb || pb {
		a.setLibraryStatus("REMOVE UNAVAILABLE — WAIT FOR THE CURRENT INSTALLATION")
		return
	}
	v, p, ok := a.selectedLibraryBrowserEntry()
	if !ok {
		return
	}
	var target, active string
	if a.libraryMode == "piper" {
		model, _, _ := a.piperVoicePaths(p)
		target = filepath.Dir(model)
		active = a.settings.ActualVoice
		if active == p.Key || a.savedLibraryPackOnDisk("piper", p.Key) {
			a.setLibraryStatus("CANNOT REMOVE ACTIVE / SAVED VOICE — SELECT AND SAVE A DIFFERENT VOICE FIRST")
			return
		}
	}
	if a.libraryMode == "vosk" {
		_, _, root := a.voskBasePaths()
		target = filepath.Join(root, v.Folder)
		if a.activeVoskSpec().Folder == v.Folder || a.savedLibraryPackOnDisk("vosk", v.Folder) {
			a.setLibraryStatus("CANNOT REMOVE ACTIVE / SAVED LISTENER — SELECT AND SAVE A DIFFERENT MODEL FIRST")
			return
		}
	}
	if !isContainedLibraryFolder(a.configPath, target) {
		a.setLibraryStatus("REMOVE BLOCKED — PATH OUTSIDE MANAGED LIBRARY")
		return
	}
	if _, err := os.Stat(target); err != nil {
		a.setLibraryStatus("MODEL NOT INSTALLED")
		return
	}
	caption := "Remove this downloaded pack from RAWDOGS? You can reinstall it later."
	yes, _, _ := messageBox.Call(uintptr(a.hwnd), uintptr(unsafe.Pointer(utf16Ptr(caption))), uintptr(unsafe.Pointer(utf16Ptr("RAWDOGS LIBRARY — CONFIRM REMOVAL"))), 0x00000004|0x00000030) // MB_YESNO / warning
	if yes != 6 {
		return
	} // IDYES
	if err := os.RemoveAll(target); err != nil {
		a.setLibraryStatus("REMOVE FAILED: " + err.Error())
	} else {
		a.setLibraryStatus("PACK REMOVED — OTHER LIBRARIES ARE UNCHANGED")
	}
	a.refreshLibraryBrowserList()
	a.updateActualVoiceLibraryStatus()
	a.updateVoskLibraryStatus()
}
func isContainedLibraryFolder(config, target string) bool {
	root := filepath.Dir(config)
	rel, e := filepath.Rel(root, target)
	if e != nil || filepath.IsAbs(rel) || rel == "." || strings.HasPrefix(rel, "..") {
		return false
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	if len(parts) < 3 {
		return false
	}
	return (parts[0] == "vosk" && parts[1] == "model") || (parts[0] == "piper" && parts[1] == "voices")
}
func (a *App) previewLibraryBrowserSelection() {
	if a.libraryMode != "piper" {
		a.setLibraryStatus("PREVIEW APPLIES TO PIPER SPEAKING VOICES — SWITCH TO PIPER FIRST")
		return
	}
	_, p, ok := a.selectedLibraryBrowserEntry()
	if !ok {
		return
	}
	a.libraryStateMu.Lock()
	if a.libraryPreviewBusy {
		a.libraryStateMu.Unlock()
		return
	}
	a.libraryPreviewBusy = true
	a.libraryStateMu.Unlock()
	a.setLibraryStatus("FETCHING THE OFFICIAL SAMPLE FOR " + p.Key)
	go func() {
		// Stable voice pack samples are MP3; sample_0 exists for one-speaker voices.
		// No speculative synthesis, and no download of the full voice on Preview.
		sampleURL := "https://huggingface.co/rhasspy/piper-voices/resolve/v1.0.0/" + p.RepoPath + "/samples/speaker_0.mp3"
		body, err := fetchOfficialCatalog(sampleURL, 5*1024*1024)
		if err == nil && len(body) > 500 && (strings.HasPrefix(string(body), "ID3") || (body[0] == 0xff && body[1]&0xe0 == 0xe0)) {
			cache := filepath.Join(filepath.Dir(a.configPath), "libraries", "samples")
			err = os.MkdirAll(cache, 0755)
			file := filepath.Join(cache, p.Key+".mp3")
			if err == nil {
				err = os.WriteFile(file, body, 0644)
			}
			if err == nil {
				err = playLibraryMP3(file)
			}
		} else if err == nil {
			err = errors.New("no valid audio sample was published for this voice")
		}
		a.libraryStateMu.Lock()
		if err != nil {
			a.libraryPreviewMessage = "VOICE PREVIEW UNAVAILABLE: " + err.Error()
		} else {
			a.libraryPreviewMessage = "VOICE PREVIEW FINISHED: " + p.Key
		}
		a.libraryPreviewBusy = false
		a.libraryStateMu.Unlock()
		procPostMessageW.Call(uintptr(a.hwnd), WM_APP_LIBRARY_PREVIEW, 0, 0)
	}()
}
func playLibraryMP3(path string) error {
	// MCI uses Windows native audio and doesn't require FFmpeg, browser or codecs bundled with RAWDOGS.
	// The playback runs in the worker thread and does not take over Piper narration.
	cmd := func(command string) uint32 {
		n, _, _ := mciCommand.Call(uintptr(unsafe.Pointer(utf16Ptr(command))), 0, 0, 0)
		return uint32(n)
	}
	cmd("close rawdogs_voice_sample")
	if n := cmd("open \"" + path + "\" type mpegvideo alias rawdogs_voice_sample"); n != 0 {
		return fmt.Errorf("Windows could not open MP3 preview (MCI %d)", n)
	}
	defer cmd("close rawdogs_voice_sample")
	if n := cmd("play rawdogs_voice_sample wait"); n != 0 {
		return fmt.Errorf("Windows cannot play this MP3 preview (MCI %d)", n)
	}
	return nil
}
func (a *App) applyLibraryPreviewResult() {
	a.libraryStateMu.Lock()
	message := a.libraryPreviewMessage
	a.libraryStateMu.Unlock()
	a.setLibraryStatus(message)
}
func (a *App) drawBadgeFlag(hdc syscall.Handle, x, y int, flag string) {
	if flag == "" {
		return
	}
	f := syscall.Handle(createFont(-22, 400, "Segoe UI Emoji"))
	if f == 0 {
		return
	}
	defer procDeleteObject.Call(uintptr(f))
	chars, _ := syscall.UTF16FromString(flag)
	if len(chars) < 2 {
		return
	}
	old, _, _ := procSelectObject.Call(uintptr(hdc), uintptr(f))
	procSetBkMode.Call(uintptr(hdc), TRANSPARENT)
	rect := RECT{int32(x + 132), int32(y + 3), int32(x + 169), int32(y + 28)}
	procDrawTextW.Call(uintptr(hdc), uintptr(unsafe.Pointer(&chars[0])), uintptr(len(chars)-1), uintptr(unsafe.Pointer(&rect)), DT_SINGLELINE|DT_CENTER|DT_VCENTER)
	procSelectObject.Call(uintptr(hdc), old)
}
func flagForSelectedVosk(display string) string {
	if s, ok := voskSpecByActiveDisplay(display); ok {
		_, reg := voskLanguageAndRegion(s.Folder)
		return countryFlag(reg)
	}
	return "🌐"
}
func flagForSelectedPiper(key string) string {
	if key == "" {
		return "🌐"
	}
	_, r := piperKeyLanguageRegion(key)
	return countryFlag(r)
}
