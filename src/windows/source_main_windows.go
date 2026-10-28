//go:build windows

package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	_ "embed"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image/png"
	"io"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	winmm    = syscall.NewLazyDLL("winmm.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")

	procShellExecuteW        = shell32.NewProc("ShellExecuteW")
	procSupportMessageBoxW   = user32.NewProc("MessageBoxW")
	procCreateWindowExW      = user32.NewProc("CreateWindowExW")
	procDefWindowProcW       = user32.NewProc("DefWindowProcW")
	procRegisterClassExW     = user32.NewProc("RegisterClassExW")
	procShowWindow           = user32.NewProc("ShowWindow")
	procUpdateWindow         = user32.NewProc("UpdateWindow")
	procGetMessageW          = user32.NewProc("GetMessageW")
	procTranslateMessage     = user32.NewProc("TranslateMessage")
	procDispatchMessageW     = user32.NewProc("DispatchMessageW")
	procPostQuitMessage      = user32.NewProc("PostQuitMessage")
	procSendMessageW         = user32.NewProc("SendMessageW")
	procSetWindowTextW       = user32.NewProc("SetWindowTextW")
	procGetWindowTextW       = user32.NewProc("GetWindowTextW")
	procGetWindowTextLengthW = user32.NewProc("GetWindowTextLengthW")
	procEnableWindow         = user32.NewProc("EnableWindow")
	procGetClientRect        = user32.NewProc("GetClientRect")
	procMoveWindow           = user32.NewProc("MoveWindow")
	procInvalidateRect       = user32.NewProc("InvalidateRect")
	procSetTimer             = user32.NewProc("SetTimer")
	procKillTimer            = user32.NewProc("KillTimer")
	procSetFocus             = user32.NewProc("SetFocus")
	procGetKeyState          = user32.NewProc("GetKeyState")
	procLoadCursorW          = user32.NewProc("LoadCursorW")
	procLoadImageW           = user32.NewProc("LoadImageW")
	procGetSysColorBrush     = user32.NewProc("GetSysColorBrush")
	procFillRect             = user32.NewProc("FillRect")
	procDrawTextW            = user32.NewProc("DrawTextW")
	procFrameRect            = user32.NewProc("FrameRect")
	procGetDC                = user32.NewProc("GetDC")
	procReleaseDC            = user32.NewProc("ReleaseDC")
	procSetWindowPos         = user32.NewProc("SetWindowPos")
	procBeginPaint           = user32.NewProc("BeginPaint")
	procEndPaint             = user32.NewProc("EndPaint")

	procCreateSolidBrush      = gdi32.NewProc("CreateSolidBrush")
	procDeleteObject          = gdi32.NewProc("DeleteObject")
	procSetTextColor          = gdi32.NewProc("SetTextColor")
	procSetBkColor            = gdi32.NewProc("SetBkColor")
	procSetBkMode             = gdi32.NewProc("SetBkMode")
	procCreateFontW           = gdi32.NewProc("CreateFontW")
	procSelectObject          = gdi32.NewProc("SelectObject")
	procCreatePen             = gdi32.NewProc("CreatePen")
	procMoveToEx              = gdi32.NewProc("MoveToEx")
	procLineTo                = gdi32.NewProc("LineTo")
	procStretchDIBits         = gdi32.NewProc("StretchDIBits")
	procEllipse               = gdi32.NewProc("Ellipse")
	procGetTextExtentPoint32W = gdi32.NewProc("GetTextExtentPoint32W")

	procGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")
	procPostMessageW     = user32.NewProc("PostMessageW")

	procWaveInGetNumDevs = winmm.NewProc("waveInGetNumDevs")
	procWaveInGetDevCaps = winmm.NewProc("waveInGetDevCapsW")
	procWaveInOpen       = winmm.NewProc("waveInOpen")
	procWaveInPrepare    = winmm.NewProc("waveInPrepareHeader")
	procWaveInUnprepare  = winmm.NewProc("waveInUnprepareHeader")
	procWaveInAddBuffer  = winmm.NewProc("waveInAddBuffer")
	procWaveInStart      = winmm.NewProc("waveInStart")
	procWaveInStop       = winmm.NewProc("waveInStop")
	procWaveInReset      = winmm.NewProc("waveInReset")
	procWaveInClose      = winmm.NewProc("waveInClose")

	procWaveOutGetNumDevs  = winmm.NewProc("waveOutGetNumDevs")
	procWaveOutGetDevCaps  = winmm.NewProc("waveOutGetDevCapsW")
	procWaveOutOpen        = winmm.NewProc("waveOutOpen")
	procWaveOutPrepare     = winmm.NewProc("waveOutPrepareHeader")
	procWaveOutUnprepare   = winmm.NewProc("waveOutUnprepareHeader")
	procWaveOutWrite       = winmm.NewProc("waveOutWrite")
	procWaveOutReset       = winmm.NewProc("waveOutReset")
	procWaveOutGetPosition = winmm.NewProc("waveOutGetPosition")
	procWaveOutClose       = winmm.NewProc("waveOutClose")
)

const (
	WS_OVERLAPPEDWINDOW  = 0x00CF0000
	WS_VISIBLE           = 0x10000000
	WS_CHILD             = 0x40000000
	WS_TABSTOP           = 0x00010000
	WS_BORDER            = 0x00800000
	WS_VSCROLL           = 0x00200000
	BS_PUSHBUTTON        = 0x00000000
	BS_OWNERDRAW         = 0x0000000B
	BS_AUTOCHECKBOX      = 0x00000003
	CBS_DROPDOWNLIST     = 0x00000003
	CBS_HASSTRINGS       = 0x00000200
	ES_READONLY          = 0x00000800
	ES_AUTOHSCROLL       = 0x00000080
	ES_MULTILINE         = 0x00000004
	ES_AUTOVSCROLL       = 0x00000040
	LBS_NOINTEGRALHEIGHT = 0x00000100
	LBS_NOSEL            = 0x00004000
	LBS_OWNERDRAWFIXED   = 0x00000010
	LBS_HASSTRINGS       = 0x00000040
	LBS_NOTIFY           = 0x00000001
	SS_LEFT              = 0x00000000
	SS_CENTER            = 0x00000001
	SW_SHOW              = 5
	SW_HIDE              = 0
	CREATE_NO_WINDOW     = 0x08000000

	WM_DESTROY             = 0x0002
	WM_ERASEBKGND          = 0x0014
	WM_PAINT               = 0x000F
	WM_DRAWITEM            = 0x002B
	WM_SETFONT             = 0x0030
	WM_CLOSE               = 0x0010
	WM_KEYDOWN             = 0x0100
	WM_COMMAND             = 0x0111
	WM_CTLCOLORSTATIC      = 0x0138
	WM_CTLCOLOREDIT        = 0x0133
	WM_CTLCOLORLISTBOX     = 0x0134
	WM_CTLCOLORBTN         = 0x0135
	WM_TIMER               = 0x0113
	WM_SIZE                = 0x0005
	WM_APP                 = 0x8000
	WM_APP_SCAN_DONE       = WM_APP + 1
	WM_APP_MIC_LEVEL       = WM_APP + 2
	WM_APP_VOICE_DONE      = WM_APP + 3
	WM_APP_RECO_EVENT      = WM_APP + 4
	WM_APP_HEALTH          = WM_APP + 5
	WM_APP_VOSK_DONE       = WM_APP + 6
	WM_APP_PIPER_DONE      = WM_APP + 7
	WM_APP_TTS_DONE        = WM_APP + 8
	WM_APP_PROVISION_STAGE = WM_APP + 9
	WM_APP_PROVISION_DONE  = WM_APP + 10
	WM_APP_LIBRARY_DONE    = WM_APP + 11
	WM_APP_LIBRARY_PREVIEW = WM_APP + 12
	WM_APP_LIBRARY_STAGE   = WM_APP + 13

	EN_SETFOCUS      = 0x0100
	EN_KILLFOCUS     = 0x0200
	EN_CHANGE        = 0x0300
	EM_SETSEL        = 0x00B1
	VK_TAB           = 0x09
	VK_BACK          = 0x08
	VK_CONTROL       = 0x11
	VK_RETURN        = 0x0D
	CB_ADDSTRING     = 0x0143
	CB_RESETCONTENT  = 0x014B
	CB_SETCURSEL     = 0x014E
	CB_GETCURSEL     = 0x0147
	CB_GETLBTEXT     = 0x0148
	CB_GETLBTEXTLEN  = 0x0149
	CBN_SELCHANGE    = 1
	LB_ADDSTRING     = 0x0180
	LB_RESETCONTENT  = 0x0184
	LB_GETTEXT       = 0x0189
	LB_GETTEXTLEN    = 0x018A
	LB_SETITEMHEIGHT = 0x01A0
	LB_SETTOPINDEX   = 0x0197
	BN_CLICKED       = 0
	ODT_LISTBOX      = 2
	ODT_BUTTON       = 4
	ODS_SELECTED     = 0x0001
	ODS_DISABLED     = 0x0004
	DT_CENTER        = 0x0001
	DT_VCENTER       = 0x0004
	DT_SINGLELINE    = 0x0020
	DT_END_ELLIPSIS  = 0x8000
	DIB_RGB_COLORS   = 0
	SRCCOPY          = 0x00CC0020

	COLOR_WINDOW = 5
	TRANSPARENT  = 1
	OPAQUE       = 2

	CALLBACK_NULL   = 0x00000000
	WAVE_FORMAT_PCM = 1
	WHDR_DONE       = 0x00000001
)

const (
	ID_NAV_OPS = 1001 + iota
	ID_NAV_SETTINGS
	ID_NAV_TRAINING
	ID_NAV_MANUAL
	ID_PERMISSION_RUN
	ID_PERMISSION_ALWAYS
	ID_PERMISSION_NOTNOW
	ID_OPEN_PRIVACY
	ID_MIC_COMBO
	ID_TEST_MIC
	ID_GAIN_COMBO
	ID_OUTPUT_COMBO
	ID_RESCAN_AUDIO
	ID_LANG_COMBO
	ID_CONF_COMBO
	ID_VOICE_COMBO
	ID_SPEED_COMBO
	ID_VOLUME_COMBO
	ID_TEST_VOICE
	ID_THEME_COMBO
	ID_TEAM_COMBO
	ID_SAVE_SETUP
	ID_REFRESH_VOICE
	ID_VOSK_LIBRARY_COMBO
	ID_VOSK_INSTALL_SELECTED
	ID_VOSK_USE_SELECTED
	ID_ACTUAL_LIBRARY_COMBO
	ID_ACTUAL_INSTALL_SELECTED
	ID_ACTUAL_USE_SELECTED
	ID_SETTINGS_LISTEN
	ID_SETTINGS_STOP
	ID_TRAIN_LISTEN
	ID_TRAIN_STOP
	ID_TRAIN_CLEAR
	ID_OP_LISTEN
	ID_OP_STOP
	ID_OP_NEW_TARGET
	ID_OP_CONFIRM
	ID_OP_CANCEL
	ID_OP_CLEAR_ALL
	ID_OP_CLEAR_ACTUAL
	ID_VOICE_LIBRARY
	ID_GUIDE_NEXT
	ID_GUIDE_END
	ID_MANUAL_QUICK
	ID_MANUAL_CUES
	ID_MANUAL_VOICES
	ID_PIPER_MUTE
	ID_PIPER_STOP
	ID_OP_UNDO_CLEAR
	ID_CLEAR_TARGET_1
	ID_CLEAR_TARGET_2
	ID_CLEAR_TARGET_3
	ID_CLEAR_TARGET_4
	ID_CLEAR_UNSAVED
	ID_CLEAR_ACTUAL_X
	ID_CLEAR_ACTUAL_Y
	ID_CLEAR_TARGET_X
	ID_CLEAR_TARGET_Y
	ID_MANUAL_NARRATED
	ID_MANUAL_READ
	ID_MANUAL_END
	ID_MANUAL_PAUSE
	ID_MANUAL_STOP
	ID_MANUAL_RESTART
	ID_MANUAL_PREV
	ID_MANUAL_NEXT
	ID_MANUAL_MODELS
	ID_MANUAL_LIBRARIES
	ID_MANUAL_SEEK
	ID_MANUAL_MUTE
	ID_OP_CLEAR_BUFFER
	ID_MANUAL_LIB_SETTINGS
	ID_MANUAL_LIB_VOSK_SOURCE
	ID_MANUAL_LIB_PIPER_SOURCE
	ID_LIBRARY_COUNTRY
	ID_LIBRARY_VOSK_MODE
	ID_LIBRARY_PIPER_MODE
	ID_LIBRARY_CHOICES
	ID_LIBRARY_REFRESH
	ID_LIBRARY_INSTALL
	ID_LIBRARY_PREVIEW
	ID_LIBRARY_SETTINGS
	ID_LIBRARY_REMOVE
)

type WNDCLASSEX struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     syscall.Handle
	HIcon         syscall.Handle
	HCursor       syscall.Handle
	HbrBackground syscall.Handle
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       syscall.Handle
}

type MSG struct {
	HWnd    syscall.Handle
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      struct{ X, Y int32 }
}

type RECT struct{ Left, Top, Right, Bottom int32 }
type SIZE struct{ Cx, Cy int32 }
type PAINTSTRUCT struct {
	Hdc         syscall.Handle
	FErase      int32
	RcPaint     RECT
	FRestore    int32
	FIncUpdate  int32
	RgbReserved [32]byte
}

type DRAWITEMSTRUCT struct {
	CtlType    uint32
	CtlID      uint32
	ItemID     uint32
	ItemAction uint32
	ItemState  uint32
	HwndItem   syscall.Handle
	HDC        syscall.Handle
	RcItem     RECT
	ItemData   uintptr
}

type BITMAPINFOHEADER struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

type BITMAPINFO struct {
	Header BITMAPINFOHEADER
	Colors [1]uint32
}

type CamoImage struct {
	W, H int
	BGRA []byte
}

type WAVEINCAPS struct {
	Mid           uint16
	Pid           uint16
	DriverVersion uint32
	Pname         [32]uint16
	Formats       uint32
	Channels      uint16
	Reserved1     uint16
}

type WAVEOUTCAPS struct {
	Mid           uint16
	Pid           uint16
	DriverVersion uint32
	Pname         [32]uint16
	Formats       uint32
	Channels      uint16
	Reserved1     uint16
	Support       uint32
}

type WAVEFORMATEX struct {
	FormatTag      uint16
	Channels       uint16
	SamplesPerSec  uint32
	AvgBytesPerSec uint32
	BlockAlign     uint16
	BitsPerSample  uint16
	CbSize         uint16
}

type WAVEHDR struct {
	Data          *byte
	BufferLength  uint32
	BytesRecorded uint32
	User          uintptr
	Flags         uint32
	Loops         uint32
	Next          *WAVEHDR
	Reserved      uintptr
}

type Settings struct {
	Version     string  `json:"version"`
	Permission  string  `json:"permission"`
	MicName     string  `json:"mic_name"`
	MicIndex    int     `json:"mic_index"`
	GainDB      float64 `json:"gain_db"`
	OutputName  string  `json:"output_name"`
	OutputIndex int     `json:"output_index"`
	Language    string  `json:"language"`
	Confidence  int     `json:"confidence"`
	VoiceName   string  `json:"voice_name"`
	VoiceIndex  int     `json:"voice_index"`
	VoiceSpeed  float64 `json:"voice_speed"`
	VoiceVolume int     `json:"voice_volume"`
	PiperMuted  bool    `json:"piper_muted"`
	ActualVoice string  `json:"actual_voice,omitempty"`
	Theme       string  `json:"theme"`
	Team        string  `json:"team"`
}

type RecoEvent struct {
	Kind       string
	Text       string
	Confidence int
	Detail     string
}

type TargetSlot struct {
	X         string
	Y         string
	Confirmed bool
}

// One-step recovery snapshot for CLEAR ALL, numbered slots and individual coordinates.
// It stores only the working state; clearing never changes saved settings or the L81 table.
type clearSnapshot struct {
	actualX, actualY string
	actualSet        bool
	targets          [4]TargetSlot
	unsaved          TargetSlot
	unsavedExists    bool
	activeTarget     int
	pending          PendingState
}

type PendingState struct {
	CueAwaiting string // X or Y: operator spoke cue, waiting for digits after a pause
	Kind        string // actual, target, switch_actual, clear_all, clear_actual
	Slot        int    // 0..3 saved targets, -2 unsaved, -1 not applicable
	X           string
	Y           string
	LastCoord   string
	Replacing   string
	StartedAt   time.Time
	UpdatedAt   time.Time
}

type VoskModelSpec struct {
	Key           string
	ActiveDisplay string
	LibraryLabel  string
	Language      string
	Folder        string
	URL           string
}

func voskModelSpecs() []VoskModelSpec {
	base := []VoskModelSpec{
		{Key: "en-gb-small", ActiveDisplay: "Vosk 0.3.38 — English UK small model — OFFLINE", LibraryLabel: "English UK — Small model (en-GB)", Language: "English (United Kingdom) [en-GB] — Vosk offline", Folder: "vosk-model-small-en-gb-0.15", URL: "https://alphacephei.com/vosk/models/vosk-model-small-en-gb-0.15.zip"},
		{Key: "en-us-small", ActiveDisplay: "Vosk — English US small model — OFFLINE", LibraryLabel: "English US — Small model (en-US)", Language: "English (United States) [en-US] — Vosk offline", Folder: "vosk-model-small-en-us-0.15", URL: "https://alphacephei.com/vosk/models/vosk-model-small-en-us-0.15.zip"},
		{Key: "en-in-small", ActiveDisplay: "Vosk — English India small model — OFFLINE", LibraryLabel: "English India — Small model (en-IN)", Language: "English (India) [en-IN] — Vosk offline", Folder: "vosk-model-small-en-in-0.4", URL: "https://alphacephei.com/vosk/models/vosk-model-small-en-in-0.4.zip"},
	}
	libraryRegistryMu.RLock()
	defer libraryRegistryMu.RUnlock()
	for _, extra := range libraryRegisteredVosk {
		found := false
		for _, old := range base {
			if old.Folder == extra.Folder {
				found = true
				break
			}
		}
		if !found {
			base = append(base, extra)
		}
	}
	return base
}

func voskLibraryLabels() []string {
	items := make([]string, 0, len(voskModelSpecs()))
	for _, s := range voskModelSpecs() {
		items = append(items, s.LibraryLabel)
	}
	return items
}

func voskSpecByLibraryLabel(label string) (VoskModelSpec, bool) {
	for _, s := range voskModelSpecs() {
		if strings.EqualFold(strings.TrimSpace(label), s.LibraryLabel) {
			return s, true
		}
	}
	return VoskModelSpec{}, false
}

func voskSpecByActiveDisplay(display string) (VoskModelSpec, bool) {
	for _, s := range voskModelSpecs() {
		if strings.EqualFold(strings.TrimSpace(display), s.ActiveDisplay) {
			return s, true
		}
	}
	return VoskModelSpec{}, false
}

type PiperVoiceSpec struct {
	Key       string
	Label     string
	ModelBase string
	RepoPath  string
}

func piperVoiceSpecs() []PiperVoiceSpec {
	// G10 starts with the stable offline UK voices previously auditioned for ACTUAL.
	// The manager is deliberately data-driven so more accents/providers can be added later
	// without changing the active-output controls above it.
	base := []PiperVoiceSpec{
		{Key: "en_GB-alba-medium", Label: "Piper — Alba — UK female (medium)", ModelBase: "en_GB-alba-medium", RepoPath: "en/en_GB/alba/medium"},
		{Key: "en_GB-northern_english_male-medium", Label: "Piper — Northern English male (medium)", ModelBase: "en_GB-northern_english_male-medium", RepoPath: "en/en_GB/northern_english_male/medium"},
		{Key: "en_GB-cori-medium", Label: "Piper — Cori — UK female (medium)", ModelBase: "en_GB-cori-medium", RepoPath: "en/en_GB/cori/medium"},
		{Key: "en_GB-alan-medium", Label: "Piper — Alan — UK male (medium)", ModelBase: "en_GB-alan-medium", RepoPath: "en/en_GB/alan/medium"},
	}
	libraryRegistryMu.RLock()
	defer libraryRegistryMu.RUnlock()
	for _, extra := range libraryRegisteredPiper {
		found := false
		for _, old := range base {
			if old.Key == extra.Key {
				found = true
				break
			}
		}
		if !found {
			base = append(base, extra)
		}
	}
	return base
}

func piperLibraryLabels() []string {
	out := make([]string, 0, len(piperVoiceSpecs()))
	for _, v := range piperVoiceSpecs() {
		out = append(out, v.Label)
	}
	return out
}

func piperSpecByLabel(label string) (PiperVoiceSpec, bool) {
	for _, v := range piperVoiceSpecs() {
		if strings.EqualFold(strings.TrimSpace(label), v.Label) {
			return v, true
		}
	}
	return PiperVoiceSpec{}, false
}

func piperSpecByKey(key string) (PiperVoiceSpec, bool) {
	for _, v := range piperVoiceSpecs() {
		if strings.EqualFold(strings.TrimSpace(key), v.Key) {
			return v, true
		}
	}
	return PiperVoiceSpec{}, false
}

type App struct {
	hwnd       syscall.Handle
	settings   Settings
	configPath string
	firstRun   bool

	controls       map[int]syscall.Handle
	screenControls map[string][]syscall.Handle
	currentScreen  string
	micNames       []string
	outputNames    []string
	voices         []string
	languages      []string

	micRunning bool
	micStop    chan struct{}
	micMu      sync.Mutex
	micPeak    int
	micStatus  string
	micTested  bool

	scanInProgress         bool
	voiceRefreshInProgress bool

	recoMu              sync.Mutex
	recoCmd             *exec.Cmd
	recoStdin           io.WriteCloser
	recoAudioStop       chan struct{}
	recoRunning         bool
	recoStatus          string
	recoAudioLast       time.Time // last completed mic->Vosk stdin transfer
	recoAudioAttempt    time.Time // last attempted feed, for stall diagnostics
	recoResultLast      time.Time // last recognizer output (partial or final)
	recoPending         []RecoEvent
	transcript          []string
	voiceTestAlt        int
	voskMu              sync.Mutex
	voskInstallRunning  bool
	voskInstallMsg      string
	voskInstallModel    string
	piperMu             sync.Mutex
	piperInstallRunning bool
	piperInstallMsg     string
	piperInstallVoice   string
	ttsMu               sync.Mutex
	ttsSpeaking         bool
	ttsAudioPlaying     bool // TRUE only while WinMM is actually playing Piper WAV
	ttsInterruptedAt    time.Time
	ttsCurrentText      string // currently spoken phrase; for narrow Piper-echo filtering
	ttsStatus           string
	ttsQueue            []string      // Piper responses heard in fast succession must not be dropped
	ttsCancel           chan struct{} // closed to abandon an outdated playback immediately
	ttsGeneration       uint64        // a cancelled Piper worker must not modify a newer response
	lastRecoClass       string
	lastRecoStatus      string // shared native status for Training and Operations
	lastDiagPaint       time.Time
	firingLastSignature string
	firingBorderState   string // ready, out_of_range, or empty
	firingBlinkPhase    int    // legacy field retained for binary compatibility; no animation in G18H
	healthStop          chan struct{}
	healthMu            sync.Mutex
	healthMicOK         bool
	healthOutOK         bool

	actualX                  string
	actualY                  string
	actualSet                bool
	lastClear                *clearSnapshot
	targets                  [4]TargetSlot
	unsaved                  TargetSlot
	unsavedExists            bool
	activeTarget             int // -1 none, 0..3 saved, -2 unsaved
	pending                  PendingState
	autoVoiceConfirm         bool // suppress duplicate legacy readbacks on automatic step-3/6 commit
	keyboardFocusID          int  // keyboard field with focus; does not override voice stage expectation
	keyboardCommitInProgress bool
	keyboardDirtyID          int
	keyboardProgrammatic     bool
	keyboardStepOverride     int    // activation cue (1/4) after BACKSPACE; no layout change
	keyboardErrorID          int    // bad manual input is red until corrected
	lastCoordKind            string // last successfully accepted voice OR keyboard stage
	lastCoordAxis            string
	lastCoordSlot            int
	operationLog             []string
	manualPage               string
	manualPageText           map[string][]string         // Page copy: exact text visible in the manual.
	manualSpeechText         map[string][]string         // The SINGLE source for both print and Piper.
	manualSpeechHandles      map[string][]syscall.Handle // Visible paragraphs indexed to the actual WAV.
	manualHighlighted        syscall.Handle              // Steady highlight: current narrated paragraph only.
	manualHighlightBrush     syscall.Handle
	narratorPendingSeek      int              // -1 unless a seek is waiting for measured WAV durations.
	manualPlayerControls     []syscall.Handle // Persistent on all six Field Manual pages.
	manualPlayerMute         bool             // Narration-only; never alters operational Piper setting.
	manualPlayerStatusShown  string
	manualPlayerSeekShown    int
	manualSeekDragging       bool
	manualQuick              []syscall.Handle
	manualCues               []syscall.Handle
	manualVoices             []syscall.Handle
	manualNarrated           []syscall.Handle
	manualSections           []syscall.Handle
	manualModels             []syscall.Handle
	manualLibraries          []syscall.Handle
	libraryMode              string // "vosk" or "piper", independent of Settings activation
	libraryCountries         []LibraryCountry
	libraryShownVosk         []VoskModelSpec
	libraryShownPiper        []PiperVoiceSpec
	libraryCatalogLoaded     bool
	libraryCatalogBusy       bool
	libraryCatalogMessage    string
	libraryPreviewBusy       bool
	libraryPreviewAlias      string
	libraryPreviewMessage    string
	libraryStageMessage      string
	libraryStateMu           sync.Mutex
	manualNarrating          bool
	narratorMu               sync.Mutex
	narratorCancel           chan struct{}
	narratorGeneration       uint64
	narratorIndex            int
	narratorPaused           bool
	narratorRunning          bool
	narratorOffset           time.Duration
	narratorElapsed          time.Duration
	narratorDurations        []time.Duration
	narratorPage             string
	narratorVisibleSection   int
	narratorError            string
	narratorFinished         bool

	// G11 guided setup/familiarisation assistant. This is additive to every existing manual control.
	guideActive             bool
	guideStep               int
	guidePulse              bool
	guideMarker             string
	guideSaved              bool
	guideOutputTesting      bool
	guideOutputAwaiting     bool
	guideOutputHeard        bool
	guideInfoSpeechStep     int
	guideSaveReminderSpoken bool
	// 2.43.19: indexed, user-paced full-application narrated tour.
	tourActive          bool
	tourHold            bool
	tourIndex           int
	tourOutputTested    bool
	tourOutputConfirmed bool
	tourDemoMode        bool
	tourDemoShown       bool
	tourSnapshot        *tourSnapshot
	tourOriginalScreen  string
	tourOriginalPage    string
	cueCoach            cueCoach     // advisory only, never changes Vosk/coordinate acceptance
	voiceProfile        voiceProfile // operator/model-scoped confirmed offline pronunciations
	voiceProfileKey     string
	voiceTrain          voiceTrainingState

	// G13 startup provisioning overlay. It runs BEFORE the interactive magenta guide
	// and is used only while RAWDOGS detects/installs/verifies the default offline
	// Vosk + Piper stack. The mortar target is the real provisioning phase.
	loaderMu             sync.Mutex
	loaderHwnd           syscall.Handle
	loaderActive         bool
	loaderPhase          int
	loaderProgress       int
	loaderImpactTicks    int
	loaderStageDone      bool
	loaderVisualComplete bool
	loaderComponent      string
	loaderAction         string
	loaderTarget         string
	loaderStatus         string
	loaderError          string
	loaderActivityTick   int
	provisionActive      bool
	provisionDone        bool
	provisionSucceeded   bool
	audioReady           bool

	darkBrush       syscall.Handle
	lightBrush      syscall.Handle
	darkEditBrush   syscall.Handle
	lightEditBrush  syscall.Handle
	darkLabelBrush  syscall.Handle
	lightLabelBrush syscall.Handle
	paperBrush      syscall.Handle
	manualPaper     map[syscall.Handle]bool
	labelPlate      map[syscall.Handle]bool
	fontNormal      syscall.Handle
	fontBold        syscall.Handle
	fontTitle       syscall.Handle
	fontSmall       syscall.Handle
	fontButton      syscall.Handle
	fontEmoji       syscall.Handle
	fontNoEntry     syscall.Handle
	camos           map[string]*CamoImage
	micMeterMap     []int

	// Registration-layout engine. G9 preserves the G6 rule: every child control in the same
	// baseline coordinate space used to paint the panel borders. On WM_SIZE we
	// apply one horizontal transform to both layers, preventing the G5 failure
	// where borders moved but text/buttons stayed behind.
	layoutBaseW int
	layoutBaseH int
	controlBase map[syscall.Handle]RECT
	textClass   map[syscall.Handle]string // G9 semantic colour: pink=cue/active, green=confirmed, red=rejected
}

// Exact team camo assets carried forward from the 2.4/browser design.
//
//go:embed camo_blue.png
var camoBluePNG []byte

//go:embed camo_red.png
var camoRedPNG []byte

//go:embed camo_green.png
var camoGreenPNG []byte

var app *App

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func utf16Ptr(s string) *uint16 { p, _ := syscall.UTF16PtrFromString(s); return p }
func utf16ToString(a []uint16) string {
	n := 0
	for n < len(a) && a[n] != 0 {
		n++
	}
	return syscall.UTF16ToString(a[:n])
}

func rgb(r, g, b byte) uintptr { return uintptr(uint32(r) | uint32(g)<<8 | uint32(b)<<16) }

func main() {
	runtime.LockOSThread()
	a := &App{controls: map[int]syscall.Handle{}, screenControls: map[string][]syscall.Handle{}, manualPaper: map[syscall.Handle]bool{}, labelPlate: map[syscall.Handle]bool{}, controlBase: map[syscall.Handle]RECT{}, textClass: map[syscall.Handle]string{}, activeTarget: -1, healthMicOK: true, healthOutOK: true}
	a.pending.Slot = -1
	app = a
	a.camos = loadCamos()
	a.settings = defaultSettings()
	a.configPath = settingsPath()
	a.loadLibraryCatalogCache()
	a.loadSettings()
	_ = a.loadVoiceProfile(a.selectedVoiceOperator())
	a.createWindow()
	a.layoutBaseW, a.layoutBaseH = a.clientSize()
	a.createUI()
	// A completed setup skips the provisioning path. The Field Manual audio
	// timeline must update on EVERY launch, not only the very first launch.
	// This GUI-thread timer advances the real WAV progress bar and its current
	// printed passage; it does not start Piper or change any narration text.
	procSetTimer.Call(uintptr(a.hwnd), 80, 100, 0)
	a.createLoaderOverlay()
	a.updateAllFromSettings()
	a.applyTheme()
	a.showScreen("settings")
	a.updateSetupStatus()
	a.guideMarker = filepath.Join(filepath.Dir(a.configPath), "g13_guided_setup_complete.txt")
	// Device discovery and default voice-stack provisioning happen only after the
	// main window is visible. The magenta setup guide does not start until both
	// provisioning and audio discovery are complete.
	go a.scanAudioAsync(false)
	go a.refreshVoiceLanguageAsync(false)
	if _, err := os.Stat(a.guideMarker); err != nil {
		a.beginStartupProvisioning()
	}
	a.healthStop = make(chan struct{})
	go a.healthLoop(a.healthStop)
	a.messageLoop()
}

func defaultSettings() Settings {
	return Settings{Version: "RAWDOGS-2.42-WINDOWS-NATIVE-G5-SETTINGS-v1", MicIndex: -1, GainDB: 0, OutputIndex: -1, Confidence: 80, VoiceIndex: -1, VoiceSpeed: 1.0, VoiceVolume: 100, Theme: "Dark", Team: "MANTICORE — GREEN"}
}

func settingsPath() string {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		base = os.TempDir()
	}
	dir := filepath.Join(base, "RAWDOGS Field Calculator", "NativeTestG5")
	os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "settings.json")
}

func (a *App) loadSettings() {
	// Test builds deliberately start as a clean first-user installation.
	// G9 keeps the G6 registered-layout baseline and the G5 settings schema/data folder so audio/Vosk setup is preserved.
	if b, err := os.ReadFile(a.configPath); err == nil {
		var st Settings
		if json.Unmarshal(b, &st) == nil && st.Version == "RAWDOGS-2.42-WINDOWS-NATIVE-G5-SETTINGS-v1" {
			if st.Permission == "this_run" || st.Permission == "not_now" {
				st.Permission = ""
			}
			a.settings = st
			a.firstRun = false
			return
		}
	}
	a.firstRun = true
}

func (a *App) saveSettingsToPath(path string, s Settings) error {
	s.Version = "RAWDOGS-2.42-WINDOWS-NATIVE-G5-SETTINGS-v1"
	b, _ := json.MarshalIndent(s, "", "  ")
	return os.WriteFile(path, b, 0644)
}

func (a *App) saveSettings() error {
	a.captureSettingsFromUI()
	disk := a.settings
	disk.Version = "RAWDOGS-2.42-WINDOWS-NATIVE-G5-SETTINGS-v1"
	if disk.Permission == "this_run" || disk.Permission == "not_now" {
		disk.Permission = ""
	}
	b, _ := json.MarshalIndent(disk, "", "  ")
	if err := os.WriteFile(a.configPath, b, 0644); err != nil {
		return err
	}
	a.firstRun = false
	setText(a.controls[3005], "SAVED SETUP LOADED FROM DISK")
	return nil
}

func (a *App) createWindow() {
	hInst, _, _ := procGetModuleHandleW.Call(0)
	cursor, _, _ := procLoadCursorW.Call(0, 32512)
	a.darkBrush = syscall.Handle(createSolidBrush(22, 27, 29))
	a.lightBrush = syscall.Handle(createSolidBrush(239, 239, 233))
	a.darkEditBrush = syscall.Handle(createSolidBrush(12, 17, 19))
	a.lightEditBrush = syscall.Handle(createSolidBrush(255, 255, 255))
	// Label plates deliberately contrast with the page in BOTH themes.
	// This keeps labels visually distinct without making them larger.
	a.darkLabelBrush = syscall.Handle(createSolidBrush(15, 21, 23))
	a.lightLabelBrush = syscall.Handle(createSolidBrush(220, 220, 212))
	a.paperBrush = syscall.Handle(createSolidBrush(239, 226, 190))
	a.fontNormal = syscall.Handle(createFont(-18, 400, "Segoe UI"))
	a.fontBold = syscall.Handle(createFont(-19, 700, "Segoe UI Semibold"))
	a.fontTitle = syscall.Handle(createFont(-30, 800, "Segoe UI"))
	a.fontSmall = syscall.Handle(createFont(-16, 400, "Segoe UI"))
	a.fontButton = syscall.Handle(createFont(-18, 700, "Segoe UI Semibold"))
	a.fontEmoji = syscall.Handle(createFont(-43, 400, "Segoe UI Emoji"))
	a.fontNoEntry = syscall.Handle(createFont(-30, 700, "Segoe UI Emoji"))
	cls := utf16Ptr("RAWDOGSNativeD")
	// RT_GROUP_ICON #101 is embedded from the original RAWDOGS_2_42_TEST.ico.
	// Set both window sizes so the title bar, taskbar and Alt+Tab use the
	// same identity as Explorer (rather than the generic default program icon).
	const iconResourceID = 101
	const imageIcon = 1
	const lrDefaultSize = 0x0040
	largeIcon, _, _ := procLoadImageW.Call(hInst, iconResourceID, imageIcon, 0, 0, lrDefaultSize)
	smallIcon, _, _ := procLoadImageW.Call(hInst, iconResourceID, imageIcon, 16, 16, 0)
	wc := WNDCLASSEX{CbSize: uint32(unsafe.Sizeof(WNDCLASSEX{})), LpfnWndProc: syscall.NewCallback(wndProc), HInstance: syscall.Handle(hInst), HIcon: syscall.Handle(largeIcon), HIconSm: syscall.Handle(smallIcon), HCursor: syscall.Handle(cursor), HbrBackground: a.darkBrush, LpszClassName: cls}
	procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	title := utf16Ptr("RAWDOGS Voice Controlled Field Calculator — Preview build 0.9.4")
	hwnd, _, err := procCreateWindowExW.Call(0, uintptr(unsafe.Pointer(cls)), uintptr(unsafe.Pointer(title)), WS_OVERLAPPEDWINDOW|WS_VISIBLE, 30, 25, 1500, 1120, 0, 0, hInst, 0)
	if hwnd == 0 {
		panic(err)
	}
	a.hwnd = syscall.Handle(hwnd)
	procShowWindow.Call(hwnd, SW_SHOW)
	procUpdateWindow.Call(hwnd)
}

func procGetSysColorBrushCall(idx uintptr) uintptr {
	r, _, _ := procGetSysColorBrush.Call(idx)
	return r
}
func createSolidBrush(r, g, b byte) uintptr {
	v, _, _ := procCreateSolidBrush.Call(rgb(r, g, b))
	return v
}

func (a *App) clientSize() (int, int) {
	var rc RECT
	procGetClientRect.Call(uintptr(a.hwnd), uintptr(unsafe.Pointer(&rc)))
	w := int(rc.Right - rc.Left)
	h := int(rc.Bottom - rc.Top)
	if w < 1100 {
		w = 1100
	}
	if h < 760 {
		h = 760
	}
	return w, h
}

// registerX applies the same baseline-to-current transform to both painted
// panel rectangles and child controls. Think of it as a print-registration
// mark: panel centre and content centre cannot drift apart because they share
// the same reference coordinate system. Vertical positions remain fixed; G6
// is correcting the horizontal registration fault seen in G5.
func (a *App) registerX(v, currentW int) int {
	base := a.layoutBaseW
	if base <= 0 || currentW <= 0 || currentW == base {
		return v
	}
	return (v*currentW + base/2) / base
}

func (a *App) registerWidth(v, currentW int) int {
	base := a.layoutBaseW
	if base <= 0 || currentW <= 0 || currentW == base {
		return v
	}
	r := (v*currentW + base/2) / base
	if r < 1 {
		r = 1
	}
	return r
}

func (a *App) relayoutRegisteredControls() {
	if a == nil || a.hwnd == 0 || a.layoutBaseW <= 0 {
		return
	}
	var rc RECT
	procGetClientRect.Call(uintptr(a.hwnd), uintptr(unsafe.Pointer(&rc)))
	currentW := int(rc.Right - rc.Left)
	if currentW < 1 {
		return
	}
	for h, br := range a.controlBase {
		if h == 0 {
			continue
		}
		x := a.registerX(int(br.Left), currentW)
		w := a.registerWidth(int(br.Right-br.Left), currentW)
		y := int(br.Top)
		hh := int(br.Bottom - br.Top)
		procMoveWindow.Call(uintptr(h), uintptr(x), uintptr(y), uintptr(w), uintptr(hh), 1)
	}
	a.resizeOperationalDiagnostics()
	a.layoutVoiceAdaptation()
}

// G18G: lower Operations panes share only the available *visible* client
// height. This prevents the G18F live feed from falling under the taskbar.
// The original upper cards (and all non-Operations tabs) remain untouched.
func (a *App) resizeOperationalDiagnostics() {
	if a.controls[5052] == 0 {
		return
	}
	var rc RECT
	procGetClientRect.Call(uintptr(a.hwnd), uintptr(unsafe.Pointer(&rc)))
	h, w := int(rc.Bottom-rc.Top), int(rc.Right-rc.Left)
	d := operationDiagnosticsForHeight(h)
	move := func(id, y, hh int, visible bool) {
		handle := a.controls[id]
		if handle == 0 {
			return
		}
		if visible {
			br := a.controlBase[handle]
			procMoveWindow.Call(uintptr(handle), uintptr(a.registerX(int(br.Left), w)), uintptr(y),
				uintptr(a.registerWidth(int(br.Right-br.Left), w)), uintptr(hh), 1)
		}
		if a.currentScreen == "operations" {
			command := uintptr(SW_HIDE)
			if visible {
				command = SW_SHOW
			}
			procShowWindow.Call(uintptr(handle), command)
		}
	}
	move(5034, d.transcriptTitleY, 24, d.showTranscriptTitle)
	move(ID_OP_CLEAR_BUFFER, d.transcriptTitleY-5, 30, d.showTranscriptTitle)
	move(5031, d.transcriptListY, d.transcriptListH, d.showTranscriptList)
	move(5050, d.liveTitleY, 24, d.showLiveTitle)
	move(5051, d.liveSummaryY, d.liveSummaryH, d.showLiveSummary)
	move(5052, d.liveListY, d.liveListH, d.showLiveList)
}

func (a *App) threeColumnGeometry() (margin, gap, colW, x1, x2, x3 int) {
	w, _ := a.clientSize()
	margin = 20
	gap = 18
	usable := w - margin*2 - gap*2
	colW = usable / 3
	x1 = margin
	x2 = x1 + colW + gap
	x3 = x2 + colW + gap
	return
}

func (a *App) operationGeometry() (margin, gap, c1, c2, c3, cueW, x1, x2, x3, x4 int) {
	w, _ := a.clientSize()
	margin = 20
	gap = 16
	usable := w - margin*2
	cueW = usable * 27 / 100
	if cueW < 330 {
		cueW = 330
	}
	left := usable - cueW - gap
	c1 = (left - gap*2) / 3
	c2 = c1
	c3 = left - gap*2 - c1 - c2
	x1 = margin
	x2 = x1 + c1 + gap
	x3 = x2 + c2 + gap
	x4 = x3 + c3 + gap
	return
}

func (a *App) trainingGeometry() (margin, gap, leftW, rightW, x1, x2 int) {
	w, _ := a.clientSize()
	margin = 20
	gap = 18
	usable := w - margin*2 - gap
	leftW = usable * 49 / 100
	rightW = usable - leftW
	x1 = margin
	x2 = x1 + leftW + gap
	return
}

const ID_BUY_ME_A_COFFEE = 8192
const buyMeACoffeeURL = "https://buymeacoffee.com/silvesta"

func (a *App) createUI() {
	w, _ := a.clientSize()
	// The three header regions follow the user's marked screenshot. Reuse
	// the same proportional registration during resize; no overlap with
	// the original team banner, even at the 1100px logical minimum.
	guide := voiceGuideHeaderLayoutForWidth(w)
	a.addStatic(0, guide.TitleX, 12, guide.TitleW, 36, "RAWDOGS", true)
	subtitle := a.addStatic(0, guide.TitleX, 48, guide.TitleW, 24,
		"V.C.F.C.", true)
	a.setFont(subtitle, a.fontSmall)
	a.addStatic(9001, guide.TeamX, 24, guide.TeamW, 28, "MANTICORE  /  GREEN TEAM", true)
	// Whole gold area is one real, clickable Windows button. Never launch
	// an external page without the operator clicking it.
	a.addButton(ID_BUY_ME_A_COFFEE, guide.SupportX, 18, guide.SupportW, 58,
		"★  BUY ME A COFFEE", "nav")
	a.addButton(ID_NAV_OPS, 20, 88, 190, 40, "OPERATIONS", "nav")
	a.addButton(ID_NAV_SETTINGS, 220, 88, 190, 40, "SETTINGS", "nav")
	a.addButton(ID_NAV_TRAINING, 420, 88, 210, 40, "VOICE TRAINING", "nav")
	a.addButton(ID_NAV_MANUAL, 640, 88, 190, 40, "FIELD MANUAL", "nav")
	// The user requested removal of the GUIDE READY panel altogether.
	// Six clean, larger Walkman buttons fill the freed header area.
	// The former header END was a duplicate: END GUIDE remains in Settings.
	// Controls remain synchronized across all four tabs.
	for i, id := range []int{ID_TOUR_PREV, ID_TOUR_PLAY, ID_TOUR_PAUSE,
		ID_TOUR_STOP, ID_TOUR_NEXT, ID_TOUR_RESTART} {
		a.addButton(id, guide.ButtonX(i), guide.ButtonY, guide.ButtonW, guide.ButtonH, "", "nav")
	}

	a.buildSettingsScreen()
	a.buildOperationsScreen()
	a.buildTrainingScreen()
	a.buildFieldManualScreen()
}

func (a *App) loaderGeometry() (x, y, w, h int) {
	var rc RECT
	procGetClientRect.Call(uintptr(a.hwnd), uintptr(unsafe.Pointer(&rc)))
	cw := int(rc.Right - rc.Left)
	ch := int(rc.Bottom - rc.Top)
	w = cw * 58 / 100
	if w < 720 {
		w = 720
	}
	if w > 980 {
		w = 980
	}
	h = w * 54 / 100
	if h < 390 {
		h = 390
	}
	if h > 500 {
		h = 500
	}
	x = (cw - w) / 2
	y = 185
	if y+h > ch-30 {
		y = maxInt(145, (ch-h)/2)
	}
	return
}

func (a *App) createLoaderOverlay() {
	hInst, _, _ := procGetModuleHandleW.Call(0)
	cls := utf16Ptr("RAWDOGSProvisionLoaderG13")
	wc := WNDCLASSEX{
		CbSize:        uint32(unsafe.Sizeof(WNDCLASSEX{})),
		LpfnWndProc:   syscall.NewCallback(loaderWndProc),
		HInstance:     syscall.Handle(hInst),
		HbrBackground: 0,
		LpszClassName: cls,
	}
	procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	x, y, w, h := a.loaderGeometry()
	hwnd, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(cls)),
		uintptr(unsafe.Pointer(utf16Ptr(""))),
		WS_CHILD,
		uintptr(x), uintptr(y), uintptr(w), uintptr(h),
		uintptr(a.hwnd), 0, hInst, 0,
	)
	a.loaderHwnd = syscall.Handle(hwnd)
	if a.loaderHwnd != 0 {
		procShowWindow.Call(uintptr(a.loaderHwnd), SW_HIDE)
	}
}

func (a *App) layoutLoaderOverlay() {
	if a.loaderHwnd == 0 {
		return
	}
	x, y, w, h := a.loaderGeometry()
	procMoveWindow.Call(uintptr(a.loaderHwnd), uintptr(x), uintptr(y), uintptr(w), uintptr(h), 1)
}

func (a *App) beginStartupProvisioning() {
	if a == nil || a.loaderHwnd == 0 || a.provisionActive {
		return
	}
	a.provisionActive = true
	a.provisionDone = false
	a.provisionSucceeded = false
	a.loaderMu.Lock()
	a.loaderActive = true
	a.loaderPhase = 0
	a.loaderProgress = 0
	a.loaderImpactTicks = 0
	a.loaderStageDone = false
	a.loaderVisualComplete = false
	a.loaderComponent = "DEFAULT OFFLINE COMMS PACKS"
	a.loaderAction = "CHECKING LOCAL ASSETS"
	a.loaderTarget = "PREPARING"
	a.loaderStatus = "Stand by — RAWDOGS is preparing the default listener and ACTUAL voice packs."
	a.loaderError = ""
	a.loaderActivityTick = 0
	a.loaderMu.Unlock()
	a.layoutLoaderOverlay()
	procShowWindow.Call(uintptr(a.loaderHwnd), SW_SHOW)
	const SWP_NOSIZE = 0x0001
	const SWP_NOMOVE = 0x0002
	const SWP_SHOWWINDOW = 0x0040
	procSetWindowPos.Call(uintptr(a.loaderHwnd), 0, 0, 0, 0, 0, SWP_NOMOVE|SWP_NOSIZE|SWP_SHOWWINDOW)
	procSetTimer.Call(uintptr(a.hwnd), 78, 30, 0)
	procInvalidateRect.Call(uintptr(a.loaderHwnd), 0, 1)
	go a.provisionDefaultVoiceStack()
}

func (a *App) setProvisionPhase(index int, component, phase, action, status string) {
	a.loaderMu.Lock()
	a.loaderPhase = index
	a.loaderProgress = 0
	a.loaderImpactTicks = 0
	a.loaderStageDone = false
	a.loaderVisualComplete = false
	a.loaderComponent = component
	a.loaderAction = action
	a.loaderTarget = phase
	a.loaderStatus = status
	a.loaderError = ""
	a.loaderMu.Unlock()
	procPostMessageW.Call(uintptr(a.hwnd), WM_APP_PROVISION_STAGE, 0, 0)
}

func (a *App) completeProvisionPhase(status string) {
	a.loaderMu.Lock()
	if status != "" {
		a.loaderStatus = status
	}
	a.loaderStageDone = true
	a.loaderMu.Unlock()
	procPostMessageW.Call(uintptr(a.hwnd), WM_APP_PROVISION_STAGE, 0, 0)
}

func (a *App) waitProvisionVisual(max time.Duration) {
	deadline := time.Now().Add(max)
	for time.Now().Before(deadline) {
		a.loaderMu.Lock()
		done := a.loaderVisualComplete
		a.loaderMu.Unlock()
		if done {
			return
		}
		time.Sleep(30 * time.Millisecond)
	}
}

func (a *App) failStartupProvisioning(err error) {
	if err == nil {
		return
	}
	a.loaderMu.Lock()
	a.loaderError = err.Error()
	a.loaderStatus = "Automatic setup could not finish. The manual library controls remain available in Settings."
	a.loaderStageDone = true
	a.loaderMu.Unlock()
	a.provisionSucceeded = false
	a.provisionActive = false
	procPostMessageW.Call(uintptr(a.hwnd), WM_APP_PROVISION_DONE, 0, 0)
}

func (a *App) finishStartupProvisioning() {
	a.provisionSucceeded = true
	a.provisionActive = false
	procPostMessageW.Call(uintptr(a.hwnd), WM_APP_PROVISION_DONE, 0, 0)
}

func (a *App) hideCommsLoader() {
	if a == nil {
		return
	}
	a.loaderMu.Lock()
	a.loaderActive = false
	a.loaderMu.Unlock()
	procKillTimer.Call(uintptr(a.hwnd), 78)
	if a.loaderHwnd != 0 {
		procShowWindow.Call(uintptr(a.loaderHwnd), SW_HIDE)
	}
}

func (a *App) tickCommsLoader() {
	a.loaderMu.Lock()
	if !a.loaderActive || a.loaderHwnd == 0 {
		a.loaderMu.Unlock()
		return
	}
	a.loaderActivityTick++
	if a.loaderError != "" {
		a.loaderMu.Unlock()
		procInvalidateRect.Call(uintptr(a.loaderHwnd), 0, 1)
		return
	}
	if a.loaderProgress < 100 {
		if a.loaderStageDone {
			a.loaderProgress += 18
			if a.loaderProgress > 100 {
				a.loaderProgress = 100
			}
		} else if a.loaderProgress < 82 {
			// Keep the shell moving while a real download/unpack/init operation is active,
			// but never allow an unfinished phase to reach the target.
			a.loaderProgress++
		}
	} else if !a.loaderVisualComplete {
		if a.loaderImpactTicks == 0 {
			a.loaderImpactTicks = 8
		} else {
			a.loaderImpactTicks--
			if a.loaderImpactTicks == 0 {
				a.loaderVisualComplete = true
			}
		}
	}
	a.loaderMu.Unlock()
	procInvalidateRect.Call(uintptr(a.loaderHwnd), 0, 1)
}

func (a *App) loaderPhaseText() (step, action, target, component, status string) {
	a.loaderMu.Lock()
	defer a.loaderMu.Unlock()
	step = fmt.Sprintf("%d/8", a.loaderPhase+1)
	action = a.loaderAction
	target = a.loaderTarget
	component = a.loaderComponent
	status = a.loaderStatus
	if a.loaderError != "" {
		action = "AUTOMATIC PROVISIONING FAILED"
		target = "FAULT"
		status = a.loaderError
	}
	return
}

func drawLoaderText(hdc syscall.Handle, font syscall.Handle, text string, rc RECT, r, g, b byte, flags uintptr) {
	if text == "" {
		return
	}
	procSetBkMode.Call(uintptr(hdc), TRANSPARENT)
	procSetTextColor.Call(uintptr(hdc), rgb(r, g, b))
	old, _, _ := procSelectObject.Call(uintptr(hdc), uintptr(font))
	defer procSelectObject.Call(uintptr(hdc), old)
	u := utf16Ptr(text)
	procDrawTextW.Call(uintptr(hdc), uintptr(unsafe.Pointer(u)), uintptr(int32(len([]rune(text)))), uintptr(unsafe.Pointer(&rc)), flags)
}

func (a *App) drawCommsLoader(hdc syscall.Handle, width, height int) {
	if hdc == 0 || width <= 0 || height <= 0 {
		return
	}
	fillColor(hdc, RECT{0, 0, int32(width), int32(height)}, 7, 12, 14)
	frameColor(hdc, RECT{1, 1, int32(width - 1), int32(height - 1)}, 197, 160, 91)
	frameColor(hdc, RECT{5, 5, int32(width - 5), int32(height - 5)}, 50, 70, 67)

	step, action, target, component, status := a.loaderPhaseText()
	drawLoaderText(hdc, a.fontBold, "RAWDOGS — DEFAULT COMMS PROVISIONING", RECT{20, 14, int32(width - 20), 48}, 245, 228, 193, DT_CENTER|DT_VCENTER|DT_SINGLELINE)
	drawLoaderText(hdc, a.fontSmall, component, RECT{24, 48, int32(width - 24), 76}, 197, 160, 91, DT_CENTER|DT_VCENTER|DT_SINGLELINE|DT_END_ELLIPSIS)
	drawLoaderText(hdc, a.fontSmall, step+"  "+action, RECT{24, 76, int32(width - 24), 104}, 255, 0, 170, DT_CENTER|DT_VCENTER|DT_SINGLELINE|DT_END_ELLIPSIS)

	leftX := width * 12 / 100
	rightX := width * 84 / 100
	baseY := height * 70 / 100
	apexLift := height * 34 / 100

	fillColor(hdc, RECT{int32(leftX - 32), int32(baseY + 24), int32(leftX + 38), int32(baseY + 31)}, 100, 105, 100)
	for i := 0; i < 7; i++ {
		x := leftX - 9 + i*4
		y := baseY + 18 - i*9
		fillColor(hdc, RECT{int32(x), int32(y), int32(x + 13), int32(y + 22)}, 145, 150, 142)
	}
	drawLoaderText(hdc, a.fontSmall, "MORTAR", RECT{int32(leftX - 65), int32(baseY + 36), int32(leftX + 70), int32(baseY + 66)}, 197, 160, 91, DT_CENTER|DT_VCENTER|DT_SINGLELINE)

	dotBrush := syscall.Handle(createSolidBrush(78, 96, 91))
	oldBrush, _, _ := procSelectObject.Call(uintptr(hdc), uintptr(dotBrush))
	for i := 0; i <= 28; i++ {
		t := float64(i) / 28.0
		x := float64(leftX) + float64(rightX-leftX)*t
		y := float64(baseY) - 4*float64(apexLift)*t*(1-t)
		procEllipse.Call(uintptr(hdc), uintptr(int(x)-2), uintptr(int(y)-2), uintptr(int(x)+2), uintptr(int(y)+2))
	}
	procSelectObject.Call(uintptr(hdc), oldBrush)
	procDeleteObject.Call(uintptr(dotBrush))

	targetW := width * 20 / 100
	if targetW < 155 {
		targetW = 155
	}
	tr, tg, tb := teamColor(a.teamKey())
	targetRC := RECT{int32(rightX - targetW/2), int32(baseY - 42), int32(rightX + targetW/2), int32(baseY + 42)}
	fillColor(hdc, targetRC, 13, 24, 22)
	frameColor(hdc, targetRC, tr, tg, tb)
	frameColor(hdc, RECT{targetRC.Left + 5, targetRC.Top + 5, targetRC.Right - 5, targetRC.Bottom - 5}, 197, 160, 91)
	drawLoaderText(hdc, a.fontSmall, target, targetRC, 245, 228, 193, DT_CENTER|DT_VCENTER|DT_SINGLELINE|DT_END_ELLIPSIS)

	a.loaderMu.Lock()
	progress := a.loaderProgress
	impactTicks := a.loaderImpactTicks
	activityTick := a.loaderActivityTick
	a.loaderMu.Unlock()
	if progress > 100 {
		progress = 100
	}
	t := float64(progress) / 100.0
	sx := float64(leftX) + float64(rightX-leftX)*t
	sy := float64(baseY) - 4*float64(apexLift)*t*(1-t)
	shellBrush := syscall.Handle(createSolidBrush(240, 190, 70))
	oldBrush, _, _ = procSelectObject.Call(uintptr(hdc), uintptr(shellBrush))
	procEllipse.Call(uintptr(hdc), uintptr(int(sx)-8), uintptr(int(sy)-5), uintptr(int(sx)+9), uintptr(int(sy)+6))
	procSelectObject.Call(uintptr(hdc), oldBrush)
	procDeleteObject.Call(uintptr(shellBrush))

	if impactTicks > 0 {
		radius := 12 + (8-impactTicks)*5
		for n := 0; n < 3; n++ {
			r := radius + n*7
			frameColor(hdc, RECT{int32(rightX - r), int32(baseY - r), int32(rightX + r), int32(baseY + r)}, 240, byte(170+n*20), 55)
		}
		drawLoaderText(hdc, a.fontBold, "BOOM", RECT{int32(rightX - 70), int32(baseY - 95), int32(rightX + 70), int32(baseY - 55)}, 240, 190, 70, DT_CENTER|DT_VCENTER|DT_SINGLELINE)
	}

	barLeft := width * 12 / 100
	barRight := width * 88 / 100
	barTop := height * 86 / 100
	fillColor(hdc, RECT{int32(barLeft), int32(barTop), int32(barRight), int32(barTop + 10)}, 31, 42, 40)
	filled := barLeft + (barRight-barLeft)*progress/100
	if filled > barLeft {
		fillColor(hdc, RECT{int32(barLeft), int32(barTop), int32(filled), int32(barTop + 10)}, tr, tg, tb)
	}
	footer := status
	if footer == "" {
		footer = "Stand by — background services are coming online" + strings.Repeat(".", (activityTick/8)%4)
	}
	drawLoaderText(hdc, a.fontSmall, footer, RECT{20, int32(barTop + 18), int32(width - 20), int32(height - 10)}, 245, 228, 193, DT_CENTER|DT_VCENTER|DT_SINGLELINE|DT_END_ELLIPSIS)
}

func fileUsable(path string, minBytes int64) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir() && st.Size() >= minBytes
}

func downloadFile(url, path string, minBytes int64) error {
	if fileUsable(path, minBytes) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	tmp := path + ".part"
	_ = os.Remove(tmp)
	client := &http.Client{Timeout: 20 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("download failed: %s", resp.Status)
	}
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	n, copyErr := io.Copy(f, resp.Body)
	closeErr := f.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}
	if n < minBytes {
		_ = os.Remove(tmp)
		return fmt.Errorf("download unexpectedly small: %d bytes", n)
	}
	_ = os.Remove(path)
	return os.Rename(tmp, path)
}

func extractZipFile(zipPath, dest string) error {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer zr.Close()
	cleanDest, err := filepath.Abs(dest)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(cleanDest, 0755); err != nil {
		return err
	}
	for _, zf := range zr.File {
		target := filepath.Join(cleanDest, filepath.FromSlash(zf.Name))
		cleanTarget, err := filepath.Abs(target)
		if err != nil {
			return err
		}
		if cleanTarget != cleanDest && !strings.HasPrefix(cleanTarget, cleanDest+string(os.PathSeparator)) {
			return fmt.Errorf("unsafe archive path: %s", zf.Name)
		}
		if zf.FileInfo().IsDir() {
			if err := os.MkdirAll(cleanTarget, 0755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(cleanTarget), 0755); err != nil {
			return err
		}
		r, err := zf.Open()
		if err != nil {
			return err
		}
		w, err := os.Create(cleanTarget)
		if err != nil {
			r.Close()
			return err
		}
		_, copyErr := io.Copy(w, r)
		closeW := w.Close()
		r.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeW != nil {
			return closeW
		}
	}
	return nil
}

func findFileByName(root, name string) (string, error) {
	var matches []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.EqualFold(info.Name(), name) {
			matches = append(matches, path)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("%s not found", name)
	}
	// Vosk NuGet packages can contain more than one Windows architecture.
	// Prefer the x64 runtime required by this x64 RAWDOGS build.
	for _, path := range matches {
		lower := strings.ToLower(filepath.ToSlash(path))
		if strings.Contains(lower, "win-x64") || strings.Contains(lower, "win64") || strings.Contains(lower, "x64") {
			return path, nil
		}
	}
	return matches[0], nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		return copyFile(path, target)
	})
}

func (a *App) chooseProvisionVosk() VoskModelSpec {
	if spec, ok := voskSpecByActiveDisplay(a.settings.VoiceName); ok && a.voskRuntimeReady() && a.voskModelInstalled(spec) {
		return spec
	}
	for _, spec := range voskModelSpecs() {
		if a.voskRuntimeReady() && a.voskModelInstalled(spec) {
			return spec
		}
	}
	return voskModelSpecs()[0]
}

func (a *App) chooseProvisionPiper() PiperVoiceSpec {
	if spec, ok := a.activeActualVoice(); ok && a.piperRuntimeReady() && a.piperVoiceInstalled(spec) {
		return spec
	}
	for _, spec := range piperVoiceSpecs() {
		if a.piperRuntimeReady() && a.piperVoiceInstalled(spec) {
			return spec
		}
	}
	return piperVoiceSpecs()[0]
}

func (a *App) provisionDefaultVoiceStack() {
	voskSpec := a.chooseProvisionVosk()
	if err := a.provisionVoskPack(voskSpec); err != nil {
		a.failStartupProvisioning(fmt.Errorf("Vosk provisioning: %w", err))
		return
	}
	piperSpec := a.chooseProvisionPiper()
	if err := a.provisionPiperPack(piperSpec); err != nil {
		a.failStartupProvisioning(fmt.Errorf("Piper provisioning: %w", err))
		return
	}

	// Defaults are only applied when the saved choice is missing or unusable. Existing
	// working user selections survive this first-run provisioning pass.
	if current, ok := voskSpecByActiveDisplay(a.settings.VoiceName); !ok || !a.voskModelInstalled(current) {
		a.settings.VoiceName = voskSpec.ActiveDisplay
		a.settings.Language = voskSpec.Language
	}
	if current, ok := a.activeActualVoice(); !ok || !a.piperVoiceInstalled(current) {
		a.settings.ActualVoice = piperSpec.Key
	}
	a.finishStartupProvisioning()
}

func (a *App) provisionVoskPack(spec VoskModelSpec) error {
	base, runtimeDir, modelRoot := a.voskBasePaths()
	if err := os.MkdirAll(base, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(runtimeDir, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(modelRoot, 0755); err != nil {
		return err
	}
	runtimeReady := a.voskRuntimeReady()
	modelReady := a.voskModelInstalled(spec)
	runtimePkg := filepath.Join(base, "vosk.0.3.38.nupkg")
	modelZip := filepath.Join(base, spec.Folder+".zip")

	a.setProvisionPhase(0, "VOSK RECOGNITION PACK", "DOWNLOADING", "VOSK — DOWNLOADING", "Checking for the UK offline recognition pack on disk.")
	if !runtimeReady {
		if err := downloadFile("https://api.nuget.org/v3-flatcontainer/vosk/0.3.38/vosk.0.3.38.nupkg", runtimePkg, 100000); err != nil {
			return err
		}
	}
	if !modelReady {
		if err := downloadFile(spec.URL, modelZip, 1000000); err != nil {
			return err
		}
	}
	if runtimeReady && modelReady {
		a.completeProvisionPhase("Local Vosk pack found — download not required.")
	} else {
		a.completeProvisionPhase("Vosk download complete.")
	}
	a.waitProvisionVisual(2 * time.Second)

	a.setProvisionPhase(1, "VOSK RECOGNITION PACK", "UNPACKING", "VOSK — UNPACKING", "Preparing the recognition runtime and model files.")
	if !runtimeReady {
		tmp := filepath.Join(base, "vosk_runtime_unpack")
		_ = os.RemoveAll(tmp)
		if err := extractZipFile(runtimePkg, tmp); err != nil {
			return err
		}
		for _, name := range []string{"libvosk.dll", "libgcc_s_seh-1.dll", "libstdc++-6.dll", "libwinpthread-1.dll"} {
			src, err := findFileByName(tmp, name)
			if err != nil {
				return err
			}
			if err := copyFile(src, filepath.Join(runtimeDir, name)); err != nil {
				return err
			}
		}
		_ = os.RemoveAll(tmp)
	}
	if !modelReady {
		target := filepath.Join(modelRoot, spec.Folder)
		_ = os.RemoveAll(target)
		if err := extractZipFile(modelZip, modelRoot); err != nil {
			return err
		}
	}
	a.completeProvisionPhase("Vosk files unpacked and staged.")
	a.waitProvisionVisual(2 * time.Second)

	a.setProvisionPhase(2, "VOSK RECOGNITION PACK", "INITIALISING", "VOSK — INITIALISING", "Verifying the runtime and English UK recognition model.")
	if !a.voskRuntimeReady() || !a.voskModelInstalled(spec) {
		return fmt.Errorf("Vosk verification failed")
	}
	a.completeProvisionPhase("Vosk recognition pack verified.")
	a.waitProvisionVisual(2 * time.Second)

	a.setProvisionPhase(3, "VOSK RECOGNITION PACK", "ONLINE", "VOSK — ONLINE", "Default recognition pack is ready for the setup tour.")
	a.completeProvisionPhase("Vosk recognition pack ONLINE.")
	a.waitProvisionVisual(2 * time.Second)
	_ = os.Remove(runtimePkg)
	_ = os.Remove(modelZip)
	return nil
}

func (a *App) provisionPiperPack(spec PiperVoiceSpec) error {
	base, runtimeDir, voicesDir := a.piperBasePaths()
	model, config, card := a.piperVoicePaths(spec)
	voiceDir := filepath.Dir(model)
	for _, d := range []string{base, runtimeDir, voicesDir, voiceDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			return err
		}
	}
	runtimeReady := a.piperRuntimeReady()
	voiceReady := a.piperVoiceInstalled(spec)
	runtimeZip := filepath.Join(base, "piper_windows_amd64.zip")
	baseURL := "https://huggingface.co/rhasspy/piper-voices/resolve/v1.0.0/" + spec.RepoPath

	a.setProvisionPhase(4, "PIPER / ACTUAL VOICE PACK", "DOWNLOADING", "PIPER — DOWNLOADING", "Checking for the default ACTUAL voice on disk.")
	if !runtimeReady {
		if err := downloadFile("https://github.com/rhasspy/piper/releases/download/2023.11.14-2/piper_windows_amd64.zip", runtimeZip, 1000000); err != nil {
			return err
		}
	}
	if !voiceReady {
		if err := downloadFile(baseURL+"/"+spec.ModelBase+".onnx", model, 1000000); err != nil {
			return err
		}
		if err := downloadFile(baseURL+"/"+spec.ModelBase+".onnx.json", config, 500); err != nil {
			return err
		}
		// Model card is useful for attribution but non-critical to playback.
		_ = downloadFile(baseURL+"/MODEL_CARD", card, 1)
	}
	if runtimeReady && voiceReady {
		a.completeProvisionPhase("Local Piper / ACTUAL pack found — download not required.")
	} else {
		a.completeProvisionPhase("Piper / ACTUAL download complete.")
	}
	a.waitProvisionVisual(2 * time.Second)

	a.setProvisionPhase(5, "PIPER / ACTUAL VOICE PACK", "UNPACKING", "PIPER — UNPACKING", "Preparing the ACTUAL voice runtime.")
	if !runtimeReady {
		tmp := filepath.Join(base, "piper_runtime_unpack")
		_ = os.RemoveAll(tmp)
		if err := extractZipFile(runtimeZip, tmp); err != nil {
			return err
		}
		pex, err := findFileByName(tmp, "piper.exe")
		if err != nil {
			return err
		}
		if err := copyDir(filepath.Dir(pex), runtimeDir); err != nil {
			return err
		}
		_ = os.RemoveAll(tmp)
	}
	a.completeProvisionPhase("Piper runtime unpacked and voice files staged.")
	a.waitProvisionVisual(2 * time.Second)

	a.setProvisionPhase(6, "PIPER / ACTUAL VOICE PACK", "INITIALISING", "PIPER — INITIALISING", "Verifying ACTUAL voice runtime and model.")
	if !a.piperRuntimeReady() || !a.piperVoiceInstalled(spec) {
		return fmt.Errorf("Piper / ACTUAL verification failed")
	}
	a.completeProvisionPhase("ACTUAL voice pack verified.")
	a.waitProvisionVisual(2 * time.Second)

	a.setProvisionPhase(7, "PIPER / ACTUAL VOICE PACK", "ONLINE", "PIPER — ONLINE", "ACTUAL is ready. Interactive setup will begin next.")
	a.completeProvisionPhase("Piper / ACTUAL voice pack ONLINE — setup tour ready.")
	a.waitProvisionVisual(2 * time.Second)
	_ = os.Remove(runtimeZip)
	return nil
}

func (a *App) buildSettingsScreen() {
	s := "settings"
	w, _ := a.clientSize()
	margin, _, colW, x1, x2, x3 := a.threeColumnGeometry()
	inner := 14
	iw := colW - inner*2

	a.addScreenStatic(s, margin, 142, 620, 26, "SETTINGS — HARDWARE / RECOGNITION / ACTUAL'S VOICE", true)
	setup := "SETUP STATUS: FIRST RUN"
	if !a.firstRun {
		setup = "SETUP STATUS: SAVED"
	}
	a.addScreenStaticWithID(s, 3005, margin, 174, 360, 24, setup, true)

	// PLAYER'S VOICE — action buttons are deliberately stacked full-width.
	// Important captions are never shortened just to preserve a compact row.
	x := x1 + inner
	a.addScreenStatic(s, x, 222, iw, 26, "PLAYER'S VOICE — INPUT", true)
	a.addScreenStatic(s, x, 260, iw, 24, "RAWDOGS MICROPHONE CONSENT", false)
	a.addScreenButton(s, ID_PERMISSION_RUN, x, 290, iw, 34, "ALLOW THIS RUN")
	a.addScreenButton(s, ID_PERMISSION_ALWAYS, x, 330, iw, 34, "ALWAYS ALLOW")
	a.addScreenButton(s, ID_PERMISSION_NOTNOW, x, 370, iw, 34, "NOT NOW")
	a.addScreenButton(s, ID_OPEN_PRIVACY, x, 414, iw, 34, "WINDOWS MIC PRIVACY")

	a.addScreenStatic(s, x, 462, iw, 24, "MICROPHONE / INPUT DEVICE", false)
	a.addScreenCombo(s, ID_MIC_COMBO, x, 492, iw, 260)
	a.addScreenButton(s, ID_TEST_MIC, x, 534, iw, 34, "TEST MIC")
	a.addScreenButton(s, ID_RESCAN_AUDIO, x, 574, iw, 34, "RESCAN WINDOWS AUDIO")

	gainW := (iw - 24) / 2
	meterX := x + gainW + 24
	a.addScreenStatic(s, x, 626, gainW, 24, "RAWDOGS INPUT GAIN", false)
	a.addScreenCombo(s, ID_GAIN_COMBO, x, 656, gainW, 220)
	a.addScreenStatic(s, meterX, 626, iw-gainW-24, 24, "MIC LEVEL — GAINED", false)
	a.addScreenStaticWithID(s, 3001, meterX, 656, iw-gainW-24, 58, "[------------------]\r\n0%   STOPPED", false)
	a.addScreenStaticWithID(s, 3002, x, 724, iw, 54, "STATUS: SELECT A MIC AND PRESS TEST MIC", true)

	// SPEECH RECOGNITION
	x = x2 + inner
	a.addScreenStatic(s, x, 222, iw, 26, "SPEECH RECOGNITION", true)
	a.addScreenStatic(s, x, 260, iw, 24, "RECOGNITION LANGUAGE", false)
	a.addScreenCombo(s, ID_LANG_COMBO, x, 290, iw, 260)
	a.addScreenStatic(s, x, 344, iw, 24, "CONFIDENCE THRESHOLD", false)
	a.addScreenCombo(s, ID_CONF_COMBO, x, 374, minInt(210, iw), 220)
	a.addScreenStatic(s, x, 430, iw, 24, "ACTIVE RECOGNITION MODEL", false)
	a.addScreenCombo(s, ID_VOICE_COMBO, x, 460, iw, 260)
	a.addScreenButton(s, ID_REFRESH_VOICE, x, 506, iw, 36, "REFRESH INSTALLED MODELS")
	a.addScreenStaticWithID(s, 3006, x, 552, iw, 64, "STATUS: OFFLINE RECOGNITION", true)

	// G9: the lower half is a library/manager, not a duplicate active-model selector.
	// Models installed here feed the ACTIVE RECOGNITION MODEL selector above.
	a.addScreenStatic(s, x, 632, iw, 26, "VOSK MODEL LIBRARY / MANAGER", true)
	a.addScreenStatic(s, x, 664, iw, 24, "AVAILABLE ENGLISH MODEL / ACCENT", false)
	a.addScreenCombo(s, ID_VOSK_LIBRARY_COMBO, x, 692, iw, 260)
	libGap := 10
	libBW := (iw - libGap) / 2
	a.addScreenButton(s, ID_VOSK_INSTALL_SELECTED, x, 734, libBW, 34, "INSTALL / REPAIR")
	a.addScreenButton(s, ID_VOSK_USE_SELECTED, x+libBW+libGap, 734, iw-libBW-libGap, 34, "USE SELECTED")
	a.addScreenStaticWithID(s, 3010, x, 778, iw, 30, "LIBRARY STATUS: READY", false)

	// ACTUAL'S VOICE
	x = x3 + inner
	a.addScreenStatic(s, x, 222, iw, 26, "ACTUAL'S VOICE — OUTPUT / LIBRARY", true)
	a.addScreenStatic(s, x, 260, iw, 24, "ACTUAL'S VOICE / OUTPUT DEVICE", false)
	a.addScreenCombo(s, ID_OUTPUT_COMBO, x, 290, iw, 260)
	voiceHalf := (iw - 24) / 2
	volX := x + voiceHalf + 24
	a.addScreenStatic(s, x, 346, voiceHalf, 24, "SPEECH SPEED", false)
	a.addScreenCombo(s, ID_SPEED_COMBO, x, 376, voiceHalf, 220)
	a.addScreenStatic(s, volX, 346, iw-voiceHalf-24, 24, "VOICE VOLUME", false)
	a.addScreenCombo(s, ID_VOLUME_COMBO, volX, 376, iw-voiceHalf-24, 220)
	// G18E: only subdivide the existing TEST VOICE row. No panels, card borders,
	// tab positions or responsive registration geometry are moved.
	smallGap := 8
	testWidth := (iw * 45) / 100
	muteWidth := (iw * 23) / 100
	stopWidth := iw - testWidth - muteWidth - smallGap*2
	a.addScreenButton(s, ID_TEST_VOICE, x, 430, testWidth, 36, "TEST VOICE")
	a.addScreenButton(s, ID_PIPER_MUTE, x+testWidth+smallGap, 430, muteWidth, 36, "MUTE")
	a.addScreenButton(s, ID_PIPER_STOP, x+testWidth+smallGap+muteWidth+smallGap, 430, stopWidth, 36, "STOP SPEECH")
	a.addScreenButton(s, ID_VOICE_LIBRARY, x, 476, iw, 38, "OPEN ACTUAL'S VOICE LIBRARY")
	a.addScreenStatic(s, x, 538, voiceHalf, 24, "THEME", false)
	a.addScreenCombo(s, ID_THEME_COMBO, x, 568, voiceHalf, 180)
	a.addScreenStatic(s, volX, 538, iw-voiceHalf-24, 24, "TEAM", false)
	a.addScreenCombo(s, ID_TEAM_COMBO, volX, 568, iw-voiceHalf-24, 180)
	a.addScreenButton(s, ID_SAVE_SETUP, x, 626, iw, 40, "SAVE SETUP TO DISK")

	// G10 ACTUAL output voice library. This is additive: the existing output device,
	// speed/volume, TEST and manual Voice Library button remain unchanged above.
	a.addScreenStatic(s, x, 684, iw, 26, "ACTUAL VOICE LIBRARY / MANAGER", true)
	a.addScreenCombo(s, ID_ACTUAL_LIBRARY_COMBO, x, 714, iw, 220)
	actualGap := 10
	actualBW := (iw - actualGap) / 2
	a.addScreenButton(s, ID_ACTUAL_INSTALL_SELECTED, x, 750, actualBW, 34, "INSTALL / REPAIR")
	a.addScreenButton(s, ID_ACTUAL_USE_SELECTED, x+actualBW+actualGap, 750, iw-actualBW-actualGap, 34, "USE SELECTED")
	a.addScreenStaticWithID(s, 3020, x, 792, iw, 24, "VOICE STATUS: NOT INSTALLED", false)

	// Full-width system health / guided-setup row below the three equal-height cards.
	// G13 keeps setup anchored in SETTINGS. START/STOP here mirror the same single
	// listener used by Operations and Voice Training; no second recogniser is created.
	a.addScreenStatic(s, margin+14, 832, 520, 26, "SYSTEM HEALTH / COMMS CONTROL / GUIDED SETUP", true)
	controlsW := 660
	statusW := w - margin*2 - 28 - controlsW - 16
	if statusW < 500 {
		statusW = 500
	}
	a.addScreenStaticWithID(s, 3004, margin+14, 864, statusW, 70, "SYSTEM STATUS: WINDOW ACTIVE — SAFE TO CONFIGURE", true)
	controlX := margin + 14 + statusW + 16
	a.addScreenStaticWithID(s, 3021, controlX, 836, controlsW, 24, "STATUS: VOSK STOPPED", true)
	buttonGap := 10
	buttonW := (controlsW - buttonGap*3) / 4
	a.addScreenButton(s, ID_SETTINGS_LISTEN, controlX, 868, buttonW, 34, "START LISTENING")
	a.addScreenButton(s, ID_SETTINGS_STOP, controlX+buttonW+buttonGap, 868, buttonW, 34, "STOP LISTENING")
	a.addScreenButton(s, ID_GUIDE_NEXT, controlX+(buttonW+buttonGap)*2, 868, buttonW, 34, "START GUIDE")
	a.addScreenButton(s, ID_GUIDE_END, controlX+(buttonW+buttonGap)*3, 868, controlsW-(buttonW+buttonGap)*3, 34, "END GUIDE")

	a.populateStaticCombos()
	a.syncPiperMuteButton()
}

func (a *App) buildOperationsScreen() {
	s := "operations"
	w, _ := a.clientSize()
	margin, gap, c1, c2, c3, cueW, x1, x2, x3, x4 := a.operationGeometry()
	pad := 12

	a.addScreenStatic(s, margin, 142, 520, 28, "OPERATIONS — TARGET WORKSPACE / VOICE CONTEXT", true)
	a.addScreenStaticWithID(s, 5000, margin, 174, 520, 24, "SYSTEM HEALTH: AMBER — STOPPED", true)
	a.addScreenButton(s, ID_OP_LISTEN, margin, 208, 180, 34, "START LISTENING")
	a.addScreenButton(s, ID_OP_STOP, margin+190, 208, 180, 34, "STOP LISTENING")
	a.addScreenStaticWithID(s, 5033, margin+390, 214, minInt(565, maxInt(360, w-830)), 24, "TEAM: MANTICORE — GREEN | LISTENER: STOPPED", true)

	// USER ACTUAL
	x := x1 + pad
	iw := c1 - pad*2
	a.addScreenStatic(s, x, 270, iw, 24, "USER ACTUAL", true)
	a.addScreenStatic(s, x, 306, 20, 24, "X", false)
	actXW := (iw - 60) / 2
	a.addScreenCoordinateEdit(s, 5001, x+28, 302, actXW-24, 30)
	a.addScreenButton(s, ID_CLEAR_ACTUAL_X, x+28+actXW-24, 305, 23, 24, "X")
	yx := x + 40 + (iw-60)/2
	a.addScreenStatic(s, yx, 306, 20, 24, "Y", false)
	actYW := iw - (yx + 28 - x)
	a.addScreenCoordinateEdit(s, 5002, yx+28, 302, actYW-24, 30)
	a.addScreenButton(s, ID_CLEAR_ACTUAL_Y, yx+28+actYW-24, 305, 23, 24, "X")
	a.addScreenStatic(s, x, 344, 80, 24, "STATUS", false)
	a.addScreenStaticWithID(s, 5003, x, 374, iw, 30, "NOT SET", false)
	a.addScreenButton(s, ID_OP_CLEAR_ACTUAL, x, 416, iw, 34, "CLEAR ACTUAL")

	// TARGET / COMMAND BUFFER
	x = x2 + pad
	iw = c2 - pad*2
	a.addScreenStatic(s, x, 270, iw, 24, "TARGET / COMMAND BUFFER", true)
	a.addScreenStaticWithID(s, 5010, x, 302, iw, 28, "NO ACTIVE TARGET", true)
	a.addScreenStatic(s, x, 344, 20, 24, "X", false)
	targetXW := (iw - 60) / 2
	a.addScreenCoordinateEdit(s, 5011, x+28, 340, targetXW-24, 30)
	a.addScreenButton(s, ID_CLEAR_TARGET_X, x+28+targetXW-24, 343, 23, 24, "X")
	yx = x + 40 + (iw-60)/2
	a.addScreenStatic(s, yx, 344, 20, 24, "Y", false)
	targetYW := iw - (yx + 28 - x)
	a.addScreenCoordinateEdit(s, 5012, yx+28, 340, targetYW-24, 30)
	a.addScreenButton(s, ID_CLEAR_TARGET_Y, yx+28+targetYW-24, 343, 23, 24, "X")
	a.addScreenStatic(s, x, 380, 80, 24, "STATUS", false)
	a.addScreenStaticWithID(s, 5013, x, 410, iw, 30, "IDLE", false)
	bGap := 8
	bW := (iw - bGap*2) / 3
	a.addScreenButton(s, ID_OP_NEW_TARGET, x, 452, bW, 34, "NEW TARGET")
	a.addScreenButton(s, ID_OP_CONFIRM, x+bW+bGap, 452, bW, 34, "CONFIRM")
	a.addScreenButton(s, ID_OP_CANCEL, x+(bW+bGap)*2, 452, iw-(bW+bGap)*2, 34, "CANCEL")

	// TARGET MEMORY
	x = x3 + pad
	iw = c3 - pad*2
	a.addScreenStatic(s, x, 270, iw, 24, "TARGET MEMORY", true)
	a.addScreenStaticWithID(s, 5020, x, 302, iw-31, 28, "TARGET 1 — EMPTY", false)
	a.addScreenButton(s, ID_CLEAR_TARGET_1, x+iw-29, 304, 24, 23, "X")
	a.addScreenStaticWithID(s, 5021, x, 336, iw-31, 28, "TARGET 2 — EMPTY", false)
	a.addScreenButton(s, ID_CLEAR_TARGET_2, x+iw-29, 338, 24, 23, "X")
	a.addScreenStaticWithID(s, 5022, x, 370, iw-31, 28, "TARGET 3 — EMPTY", false)
	a.addScreenButton(s, ID_CLEAR_TARGET_3, x+iw-29, 372, 24, 23, "X")
	a.addScreenStaticWithID(s, 5023, x, 404, iw-31, 28, "TARGET 4 — EMPTY", false)
	a.addScreenButton(s, ID_CLEAR_TARGET_4, x+iw-29, 406, 24, 23, "X")
	a.addScreenStaticWithID(s, 5024, x, 438, iw-31, 32, "UNSAVED TARGET — EMPTY", false)
	a.addScreenButton(s, ID_CLEAR_UNSAVED, x+iw-29, 442, 24, 23, "X")
	a.addScreenButton(s, ID_OP_CLEAR_ALL, x, 480, (iw-8)*2/3, 34, "CLEAR ALL TARGETS")
	a.addScreenButton(s, ID_OP_UNDO_CLEAR, x+(iw-8)*2/3+8, 480, iw-(iw-8)*2/3-8, 34, "UNDO CLEAR")

	// CUE REFERENCE
	x = x4 + pad
	iw = cueW - pad*2
	a.addScreenStatic(s, x, 270, iw, 24, "VOICE CUE REFERENCE — ALWAYS VISIBLE", true)
	cueList := a.addScreenTrainingList(s, 5040, x, 302, iw, 414)
	a.populateTrainingList(cueList)

	// CURRENT CONTEXT spans first two columns
	contextW := c1 + gap + c2
	a.addScreenStatic(s, x1+pad, 536, contextW-pad*2, 24, "CURRENT VOICE CONTEXT / COMMAND BUFFER", true)
	a.addScreenStaticWithID(s, 5030, x1+pad, 568, contextW-pad*2, 142, "IDLE — say USER ACTUAL or NEW TARGET", true)

	// FIRING SOLUTION contains only user-facing information.
	x = x3 + pad
	iw = c3 - pad*2
	a.addScreenStatic(s, x, 536, iw, 24, "FIRING SOLUTION", true)
	a.addScreenStaticWithID(s, 5032, x, 568, iw, 142, "Bearing: ---°\r\nRange: --- m\r\nMIL: ---\r\n\r\nSTATUS: AWAITING TARGET", true)

	// Original operational transcript remains where users know it. Slightly
	// enlarge the list; the extra *shared* speech diagnosis goes below it.
	a.addScreenStaticWithID(s, 5034, margin+12, 806, 610, 24, "OPERATIONS TRANSCRIPT — RAW HEARD + INTERPRETED COMMANDS", true)
	a.addScreenButton(s, ID_OP_CLEAR_BUFFER, w-margin-54, 801, 34, 30, "X")
	opsFeed := a.addScreenTrainingList(s, 5031, margin+12, 838, w-margin*2-24, 115)
	procSendMessageW.Call(uintptr(opsFeed), LB_ADDSTRING, 0, uintptr(unsafe.Pointer(utf16Ptr("No operational commands yet."))))
	// G18F expansion is exclusively in the agreed unused space beneath
	// the existing panels. Maximising on a second screen reveals the full feed.
	a.addScreenStaticWithID(s, 5050, margin+12, 982, 690, 24, "LIVE RECOGNITION — RAW HEARD / CONFIDENCE / INTERPRETATION", true)
	a.addScreenStaticWithID(s, 5051, margin+12, 1014, w-margin*2-24, 55, "RAW HEARD: ---   |   CONFIDENCE: ---   |   INTERPRETATION: WAITING", true)
	feed := a.addScreenTrainingList(s, 5052, margin+12, 1084, w-margin*2-24, 90)
	procSendMessageW.Call(uintptr(feed), LB_ADDSTRING, 0, uintptr(unsafe.Pointer(utf16Ptr("One shared Vosk recogniser — speak to see raw, confidence and interpretation."))))
	a.resizeOperationalDiagnostics()
	a.refreshOperationsUI()
}

func (a *App) buildTrainingScreen() {
	s := "training"
	w, _ := a.clientSize()
	margin := 20
	// LOCKED 2.43.12 header: exact same controls, positions and styling.
	a.addScreenStatic(s, margin, 142, 520, 28, "VOICE TRAINING — MASTER RAWDOGS CUE DICTIONARY", true)
	a.addScreenStaticWithID(s, 4100, margin, 174, 520, 24, "SYSTEM HEALTH: AMBER — LISTENER STOPPED", true)
	a.addScreenStaticWithID(s, 4101, margin, 204, minInt(900, w-margin*2), 24, "READY CHECK: select/save microphone + language in Settings, then START LISTENING", false)
	// All redesigned/added controls start below the protected header (y>=238).
	a.buildVoiceAdaptationScreen()
}

// G17: replaces the four non-interactive coordinate labels with native single-line
// edit boxes in their EXACT original rectangles; no layout geometry changes.
func (a *App) addScreenCoordinateEdit(screen string, id, x, y, w, h int) syscall.Handle {
	hw := a.createControl(id, "EDIT", "", WS_CHILD|WS_VISIBLE|WS_TABSTOP|WS_BORDER|ES_AUTOHSCROLL, x, y, w, h, 0, screen)
	a.setFont(hw, a.fontBold)
	return hw
}

func (a *App) addScreenEdit(screen string, x, y, w, h int, text string) syscall.Handle {
	hw := a.createControl(0, "EDIT", text, WS_CHILD|WS_VISIBLE|WS_BORDER|WS_VSCROLL|ES_MULTILINE|ES_AUTOVSCROLL|ES_READONLY, x, y, w, h, 0, screen)
	a.setFont(hw, a.fontNormal)
	return hw
}

func masterCueTrainingRows() []string {
	return []string{
		"1 — LISTENER CONTROL",
		"START LISTENING — use the on-screen button",
		"STOP LISTENING — use the on-screen button",
		"",
		"2 — G18 SIX-STEP FLOW — ACTUAL (VOICE OR KEYBOARD)",
		"STEP 1: REFRESH ACTUAL (or ACTUAL / ACTUALLY)",
		"STEP 2: X-RAY 97.86 / TAB TO X, TYPE, ENTER",
		"STEP 3: ACTUAL YANKEE 34.60 / TAB TO Y, TYPE, ENTER",
		"Y automatically confirms the ACTUAL",
		"",
		"3 — SET USER ACTUAL — ONE SHOT",
		"User actual X-RAY 97.86 YANKEE 34.60 Over",
		"Set user actual X-RAY 97.86 YANKEE 34.60 Over",
		"",
		"4 — G18 SIX-STEP FLOW — TARGET (VOICE OR KEYBOARD)",
		"STEP 4: REFRESH TARGET (or NEW TARGET)",
		"Restart = restart unfinished target",
		"Start again",
		"STEP 5: X-RAY 98.34",
		"STEP 6: TARGET YANKEE 35.10 (YANK accepted)",
		"Y automatically saves the target",
		"",
		"5 — CREATE NEW TARGET — ONE SHOT",
		"New target X-RAY 98.34 YANKEE 35.10 Over",
		"Target X-RAY 98.34 YANKEE 35.10 Over",
		"",
		"6 — COORDINATE WORDING PRACTICE",
		"X-RAY is 97.86",
		"X-RAY equals 97.86",
		"Set X-RAY 97.86",
		"X ninety-seven point eight six",
		"X nine seven point eight six",
		"YANKEE is 34.60",
		"YANKEE equals 34.60",
		"Set YANKEE 34.60",
		"Y thirty-four point six zero",
		"Y three four point six zero",
		"",
		"7 — CORRECT / FINISH / ABANDON",
		"CORRECTION — reopen last accepted X/Y, including after confirmation",
		"After CORRECTION say new digits; repeat X-RAY/YANKEE if desired",
		"X-RAY / YANKEE does not need repeating after Correction (but may be repeated)",
		"OVER / CONFIRM — needed for one-shot, not the six-step Y",
		"Cancel",
		"Negative",
		"",
		"8 — READ BACK FIRING SOLUTION",
		"Repeat solution",
		"Repeat bearing",
		"Repeat range",
		"Repeat mil",
		"",
		"9 — USE STORED TARGETS",
		"Recall Target 1",
		"Recall Target 2",
		"Recall Target 3",
		"Recall Target 4",
		"Use Target 1",
		"Use Target 2",
		"Use Target 3",
		"Use Target 4",
		"",
		"10 — CHANGE OR CLEAR",
		"Switching actual",
		"Clear Target 1",
		"Clear Target 2",
		"Clear Target 3",
		"Clear Target 4",
		"Clear all targets — acts immediately, preserves User Actual",
		"Undo clear — restore most recent clear",
		"Clear unsaved target",
		"Clear actual — separate confirmation",
		"CLEAR ACTUAL X / Y — erase individual coordinate",
		"CLEAR TARGET X / Y — erase individual coordinate",
		"SAVE TO DISK / SAVE SETTINGS / SAVE SETUP",
		"STOP — stops the narrated manual",
		"",
		"11 — RECOGNISER ALIASES / SPECIAL CASES",
		"Target one",
		"Target two",
		"Target three",
		"Target four",
		"Target 1",
		"Target 2",
		"Target 3",
		"Target 4",
		"Target too — recognised as Target 2 only inside a valid target command",
		"Your actual X-RAY 97.86 YANKEE 34.60 Over — protected full-command alias only",
		"",
		"TRAINING NOTE",
		"LISTENER STOPPED means recognition is off. IDLE / READY means recognition is already live.",
		"Bare X or Y is ignored while idle. In the Y step, YANK / YANKIE / YANKY count as YANKEE.",
		"During a workflow: cue first, then digits. CORRECTION reopens the most recent X/Y for replacement.",
		"Voice Training and Operations share one live listener. Vosk continues listening while Piper speaks.",
		"G18I: Narrated manual only after Save Setup To Disk; STOP stops narration.",
	}
}

func (a *App) addScreenTrainingList(screen string, id, x, y, w, h int) syscall.Handle {
	hw := a.createControl(id, "LISTBOX", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_VSCROLL|LBS_NOINTEGRALHEIGHT|LBS_NOSEL|LBS_OWNERDRAWFIXED|LBS_HASSTRINGS, x, y, w, h, 0, screen)
	a.setFont(hw, a.fontNormal)
	procSendMessageW.Call(uintptr(hw), LB_SETITEMHEIGHT, 0, 24)
	return hw
}

func (a *App) populateTrainingList(hwnd syscall.Handle) {
	if hwnd == 0 {
		return
	}
	procSendMessageW.Call(uintptr(hwnd), LB_RESETCONTENT, 0, 0)
	for _, row := range masterCueTrainingRows() {
		procSendMessageW.Call(uintptr(hwnd), LB_ADDSTRING, 0, uintptr(unsafe.Pointer(utf16Ptr(row))))
	}
}

func (a *App) buildFieldManualScreen() {
	s := "manual"
	w, _ := a.clientSize()
	a.manualPage = "quick"
	a.manualPageText = make(map[string][]string)
	a.manualSpeechText = make(map[string][]string)
	a.manualSpeechHandles = make(map[string][]syscall.Handle)
	a.narratorPendingSeek = -1
	if a.manualHighlightBrush == 0 {
		a.manualHighlightBrush = syscall.Handle(createSolidBrush(255, 247, 169))
	}
	a.narratorVisibleSection = -1
	margin := 20
	gap := 16
	paperW := (w - margin*2 - gap) * 68 / 100
	if paperW < 760 {
		paperW = 760
	}
	sideX := margin + paperW + gap
	sideW := w - margin - sideX
	if sideW < 330 {
		sideW = 330
	}
	paperLeft := margin + 38
	paperInnerW := paperW - 76
	colGap := 28
	leftCol := (paperInnerW - colGap) / 2
	rightColX := paperLeft + leftCol + colGap
	rightColW := paperInnerW - leftCol - colGap

	a.addScreenStatic(s, margin, 146, 600, 30, "RAWDOGS FIELD MANUAL — LIVE TRACTOR-FEED REFERENCE", true)
	a.addScreenButton(s, ID_MANUAL_QUICK, margin, 188, 160, 36, "QUICK RADIO FLOW")
	a.addScreenButton(s, ID_MANUAL_CUES, margin+170, 188, 150, 36, "VOICE CUES")
	a.addScreenButton(s, ID_MANUAL_VOICES, margin+330, 188, 185, 36, "ACTUAL'S VOICE")
	a.addScreenButton(s, ID_MANUAL_NARRATED, margin+525, 188, 175, 36, "NARRATED GUIDE")
	a.addScreenButton(s, ID_MANUAL_MODELS, margin+710, 188, 140, 36, "VOSK / PIPER")
	a.addScreenButton(s, ID_MANUAL_LIBRARIES, margin+860, 188, 120, 36, "LIBRARIES")
	// END GUIDE removed from Field Manual page navigation; Page 4 has STOP.

	addPaper := func(page string, x, y, ww, h int, text string, bold bool) syscall.Handle {
		hw := a.addScreenStatic(s, x, y, ww, h, text, bold)
		a.manualPageText[page] = append(a.manualPageText[page], text)
		// One-to-one mapping: Piper speaks ONLY text displayed in this
		// instructional block. Skip page counters/cover titles, not instructions.
		eligible := len(a.manualPageText[page]) > 2
		if page == "narrated" {
			eligible = false
			for _, paragraph := range narratedFieldManual {
				if text == paragraph {
					eligible = true
					break
				}
			}
		}
		if eligible {
			a.manualSpeechText[page] = append(a.manualSpeechText[page], text)
			a.manualSpeechHandles[page] = append(a.manualSpeechHandles[page], hw)
		}
		a.manualPaper[hw] = true
		switch page {
		case "quick":
			a.manualQuick = append(a.manualQuick, hw)
		case "cues":
			a.manualCues = append(a.manualCues, hw)
		case "voices":
			a.manualVoices = append(a.manualVoices, hw)
		case "narrated":
			a.manualNarrated = append(a.manualNarrated, hw)
		case "models":
			a.manualModels = append(a.manualModels, hw)
		case "libraries":
			a.manualLibraries = append(a.manualLibraries, hw)
		}
		return hw
	}

	// Version 2.43.9: each page has a different purpose. Every instructional
	// block is read VERBATIM (whitespace normalised) from its printed text.
	// No R&D diary, obsolete OVER command or spoken ordinal section labels.
	addPaper("quick", paperLeft, 254, paperInnerW, 30, "QUICK RADIO FLOW — SIX STEPS", true)
	addPaper("quick", paperLeft, 286, paperInnerW, 22, "PAGE 1 OF 6", false)
	addPaper("quick", paperLeft, 326, leftCol, 26, "LISTENER CONTROL.", true)
	addPaper("quick", paperLeft, 358, leftCol, 94, "Select START LISTENING before speaking. Select STOP LISTENING to turn voice recognition off.", false)
	addPaper("quick", rightColX, 326, rightColW, 26, "SET ACTUAL POSITION.", true)
	addPaper("quick", rightColX, 358, rightColW, 130, "Say REFRESH ACTUAL.\r\nSay X-RAY, followed by the actual X coordinate.\r\nSay YANKEE, followed by the actual Y coordinate.\r\nThe accepted Actual position is retained.", false)
	addPaper("quick", paperLeft, 495, leftCol, 26, "CREATE A TARGET.", true)
	addPaper("quick", paperLeft, 527, leftCol, 142, "Say NEW TARGET or REFRESH TARGET.\r\nSay X-RAY, followed by the target X coordinate.\r\nSay YANKEE, followed by the target Y coordinate.\r\nThe accepted target produces the firing solution.", false)
	addPaper("quick", rightColX, 495, rightColW, 26, "CONFIRMATION AND READBACK.", true)
	addPaper("quick", rightColX, 527, rightColW, 140, "Each accepted YANKEE completes its position automatically. No extra finishing cue is needed. Piper gives short next-step prompts and the calculated bearing, range and MIL when available.", false)
	addPaper("quick", paperLeft, 705, paperInnerW, 26, "OPERATING PRINCIPLE.", true)
	addPaper("quick", paperLeft, 737, paperInnerW, 113, "Use one cue at a time. Wait for acceptance before giving the next coordinate. Refresh Actual only when the gun position changes; subsequent targets use the confirmed Actual. For corrections, recalls or clear commands, open VOICE CUES.", false)

	addPaper("cues", paperLeft, 254, paperInnerW, 30, "VOICE CUES — COMMAND REFERENCE", true)
	addPaper("cues", paperLeft, 286, paperInnerW, 22, "PAGE 2 OF 6", false)
	addPaper("cues", paperLeft, 326, leftCol, 26, "COORDINATE WORDS.", true)
	addPaper("cues", paperLeft, 358, leftCol, 125, "X-RAY identifies an X coordinate. YANKEE identifies Y. When the listener is awaiting Y, YANK and YANKIE are accepted variants. Enter the expected coordinate after its cue. Whole numbers are valid.", false)
	addPaper("cues", rightColX, 326, rightColW, 26, "EDIT AND ABANDON.", true)
	addPaper("cues", rightColX, 358, rightColW, 142, "CORRECTION reopens the last accepted coordinate. Speak the replacement number. BACKSPACE moves to the preceding field; TAB advances. RESTART or START AGAIN resets an unfinished target. CANCEL abandons provisional input.", false)
	addPaper("cues", paperLeft, 515, leftCol, 26, "RECALL AND READBACK.", true)
	addPaper("cues", paperLeft, 547, leftCol, 143, "RECALL TARGET ONE, TWO, THREE or FOUR selects a filled slot. REPEAT SOLUTION gives the firing values again. REPEAT BEARING, REPEAT RANGE and REPEAT MIL request their individual readings.", false)
	addPaper("cues", rightColX, 515, rightColW, 26, "CLEAR AND UNDO.", true)
	addPaper("cues", rightColX, 547, rightColW, 158, "CLEAR TARGET followed by ONE to FOUR clears that slot. CLEAR ALL TARGETS removes stored and unsaved targets but keeps Actual. UNDO CLEAR restores the most recent deletion. CLEAR ACTUAL is separate and requires confirmation.", false)
	addPaper("cues", paperLeft, 738, paperInnerW, 26, "PIPER AND SAVING.", true)
	addPaper("cues", paperLeft, 771, paperInnerW, 82, "PIPER STOP interrupts spoken output. PIPER MUTE and PIPER UNMUTE control speech. SAVE TO DISK saves the current setup. Empty slots cannot be recalled. Check the live transcript if a command is not accepted.", false)

	addPaper("voices", paperLeft, 254, paperInnerW, 30, "ACTUAL'S VOICE — PIPER OUTPUT", true)
	addPaper("voices", paperLeft, 286, paperInnerW, 22, "PAGE 3 OF 6", false)
	addPaper("voices", paperLeft, 330, paperInnerW, 26, "CHOOSE A VOICE.", true)
	addPaper("voices", paperLeft, 362, paperInnerW, 106, "Open SETTINGS. Choose the installed Piper voice, or use INSTALL / REPAIR and USE SELECTED. TEST ACTUAL'S VOICE plays a sample through the chosen output device.", false)
	addPaper("voices", paperLeft, 491, paperInnerW, 26, "ADJUST SPEECH.", true)
	addPaper("voices", paperLeft, 523, paperInnerW, 100, "Set Piper speech speed and volume in SETTINGS. SAVE SETUP TO DISK keeps these choices. The Field Manual uses the selected voice and output when PLAY is pressed.", false)
	addPaper("voices", paperLeft, 649, paperInnerW, 26, "KEEP PROMPTS SHORT.", true)
	addPaper("voices", paperLeft, 681, paperInnerW, 145, "Piper gives brief prompts for the next expected input. Operational readback uses the calculator's accepted coordinates and firing solution. Speak a new command to interrupt outdated speech; PIPER STOP also stops output. The manual's blue-and-silver controls affect its narration independently.", false)

	addPaper("narrated", paperLeft, 254, paperInnerW, 30, "NARRATED GUIDE — READ AND LISTEN TOGETHER", true)
	addPaper("narrated", paperLeft, 286, paperInnerW, 22, "PAGE 4 OF 6", false)
	// Page-four status is dynamic UI, not instructional prose to read aloud.
	status := a.addScreenStaticWithID(s, 6204, paperLeft, 333, paperInnerW, 42, "PRESS PLAY AFTER SAVING SETTINGS", true)
	a.manualPaper[status] = true
	a.manualNarrated = append(a.manualNarrated, status)
	addPaper("narrated", paperLeft, 382, paperInnerW, 28, "INDEX — SELECT ANY GUIDE TOPIC", true)
	addPaper("narrated", paperLeft, 460, paperInnerW, 28, "NOW READING — CURRENT PASSAGE BELOW", true)
	// One index and one live passage, not dozens of overlapping static controls.
	// This transcript is the authoritative source Piper reads in guide mode.
	index := a.addScreenCombo(s, ID_TOUR_INDEX, paperLeft, 413, paperInnerW, 300)
	a.manualNarrated = append(a.manualNarrated, index)
	for _, chapter := range guidedTour {
		procSendMessageW.Call(uintptr(index), CB_ADDSTRING, 0, uintptr(unsafe.Pointer(utf16Ptr(chapter.Title))))
	}
	procSendMessageW.Call(uintptr(index), CB_SETCURSEL, 0, 0)
	current := a.addScreenStaticWithID(s, ID_TOUR_CURRENT, paperLeft, 496, paperInnerW, 129, guidedTour[0].Narration, false)
	a.manualNarrated = append(a.manualNarrated, current)
	a.manualSections = append(a.manualSections, current)
	a.manualPaper[current] = true
	a.manualSpeechText["narrated"] = guidedTourNarration()
	for range guidedTour {
		a.manualSpeechHandles["narrated"] = append(a.manualSpeechHandles["narrated"], current)
	}
	fullText := guidedTourIndex()
	transcript := a.createControl(6206, "EDIT", fullText, WS_CHILD|WS_VISIBLE|WS_BORDER|WS_VSCROLL|0x0004|0x0800|0x0100, paperLeft, 639, paperInnerW, 222, 0, s) // ES_MULTILINE, ES_READONLY, ES_NOHIDESEL
	a.manualNarrated = append(a.manualNarrated, transcript)
	a.manualPaper[transcript] = true
	a.setFont(transcript, a.fontNormal)
	addPaper("narrated", paperLeft, 875, paperInnerW, 29, "PLAY, PAUSE, STOP, RESTART, PREVIOUS, NEXT OR DRAG TO SEEK.", true)

	addPaper("models", paperLeft, 254, paperInnerW, 30, "VOSK AND PIPER — HOW THEY WORK", true)
	addPaper("models", paperLeft, 286, paperInnerW, 22, "PAGE 5 OF 6", false)
	addPaper("models", paperLeft, 326, paperInnerW, 26, "VOSK LISTENS.", true)
	addPaper("models", paperLeft, 358, paperInnerW, 103, "Vosk converts microphone audio into recognised words locally. START LISTENING activates the recogniser; STOP LISTENING turns it off. IDLE or READY means the listener is live, awaiting a useful cue.", false)
	addPaper("models", paperLeft, 492, paperInnerW, 26, "PIPER SPEAKS.", true)
	addPaper("models", paperLeft, 524, paperInnerW, 99, "Piper generates speech from the chosen offline voice. Vosk can listen while Piper speaks. The monkey status indicators show the two systems separately; using a headset can reduce audio feedback.", false)
	addPaper("models", paperLeft, 655, paperInnerW, 26, "CHECK A PROBLEM.", true)
	addPaper("models", paperLeft, 687, paperInnerW, 155, "If a cue does not work, check the listener status and raw-heard transcript before repeating it. Confirm the microphone, audio output, active Vosk model and Piper voice in SETTINGS. The calculator's firing calculations do not depend on narration.", false)

	// Page 6 is an in-app guided discovery and installation browser. Its controls
	// are registered with the existing manual-page visibility and resize logic.
	// Model activation and persistence remain in Settings, not in the browser.
	a.buildLibraryBrowser(s, paperLeft, paperInnerW)
	// Player layout never affects any printed-page text, and is registered
	// together with the rest of the UI for width/monitor resizing.
	syscall.NewLazyDLL("comctl32.dll").NewProc("InitCommonControls").Call()
	icons := []struct {
		id    int
		title string
	}{
		{ID_MANUAL_PREV, "Previous section"},
		{ID_MANUAL_READ, "Play / Resume"},
		{ID_MANUAL_PAUSE, "Pause"},
		{ID_MANUAL_STOP, "Stop"},
		{ID_MANUAL_NEXT, "Next section"},
		{ID_MANUAL_RESTART, "Restart narration"},
		{ID_MANUAL_MUTE, "Narration mute / unmute"},
	}
	for i, icon := range icons {
		h := a.addScreenButton(s, icon.id, paperLeft+i*59, 945, 56, 58, icon.title)
		a.manualPlayerControls = append(a.manualPlayerControls, h)
	}
	seekX := paperLeft + 7*59 + 8
	seekW := paperInnerW - 7*59 - 16
	if seekW < 120 {
		seekW = 120
	}
	seek := a.createControl(ID_MANUAL_SEEK, "msctls_trackbar32", "", WS_CHILD|WS_VISIBLE|WS_TABSTOP, seekX, 957, seekW, 34, 0, s)
	a.manualPlayerControls = append(a.manualPlayerControls, seek)
	procSendMessageW.Call(uintptr(seek), 0x0406, 1, uintptr(10000<<16)) // TBM_SETRANGE: actual timeline 0..10000.
	statusLine := a.addScreenStaticWithID(s, 6255, paperLeft+4, 1005, paperInnerW-8, 26, "AUDIO: READY — SAVE FIRST, THEN PLAY", false)
	a.manualPlayerControls = append(a.manualPlayerControls, statusLine)
	// Live sidebar remains visible while the printed reference page changes.
	a.addScreenStatic(s, sideX+12, 252, 270, 24, "LISTENER STATUS — LIVE", true)
	a.addScreenStaticWithID(s, 6200, sideX+12, 288, sideW-24, 34, "LISTENER: STOPPED", true)
	a.addScreenStaticWithID(s, 6201, sideX+12, 330, sideW-24, 76, "CURRENT CONTEXT: IDLE", false)
	a.addScreenStatic(s, sideX+12, 432, 270, 24, "COMMAND FLOW / PROGRESS", true)
	a.addScreenStaticWithID(s, 6203, sideX+12, 466, sideW-24, 130, "○ Listener live\r\n○ Step 1 — ACTUAL\r\n○ X / ○ Y — Steps 2–3\r\n○ Step 4 — TARGET\r\n○ X / ○ Y — Steps 5–6", false)
	a.addScreenStatic(s, sideX+12, 622, 250, 24, "LIVE COMMAND TRACK", true)
	a.addScreenStaticWithID(s, 6202, sideX+12, 656, sideW-24, 142, "No operational commands yet.", false)
	a.addScreenStatic(s, sideX+12, 820, 130, 24, "QUICK TIPS", true)
	a.addScreenStatic(s, sideX+12, 852, sideW-24, 70, "RAWDOGS keeps listening when you change tabs.\r\nRead the procedure sheet aloud and substitute your live X / Y values.\r\nReturn to Operations to see the resulting state / solution.", false)

	a.setManualPage("quick")
	a.updateManualLive()
	if !a.manualSavedToDisk() {
		setText(a.controls[6204], "SAVE SETUP TO DISK FIRST — THEN PIPER CAN NARRATE")
	}
}

// G18I spoken user guide: called only after a successful Save Setup To Disk.
// Short sections are synthesised separately so END GUIDE can cut audio promptly.
var narratedFieldManual = guidedTourNarration()

func (a *App) manualSavedToDisk() bool {
	if a == nil || a.configPath == "" {
		return false
	}
	_, err := os.Stat(a.configPath)
	return err == nil && !a.firstRun
}

func (a *App) endNarratedManual()   { a.narratorStop(true) }
func (a *App) startNarratedManual() { a.narratorPlay() }

// Open only our two fixed, vetted catalog links in the system browser.
// The external pages are references, not an alternative app installer.
func (a *App) openOfficialLibraryPage(url string) {
	if !isApprovedLibraryCatalog(url) {
		return
	}
	ret, _, _ := procShellExecuteW.Call(uintptr(a.hwnd),
		uintptr(unsafe.Pointer(utf16Ptr("open"))),
		uintptr(unsafe.Pointer(utf16Ptr(url))),
		0, 0, SW_SHOW)
	if ret <= 32 {
		setText(a.controls[6255], "LIBRARY LINK COULD NOT OPEN — CHECK YOUR BROWSER")
	}
}

// The project already publishes this optional support URL in its original
// GitHub README. Open it in the user's default browser only on a real click.
// No login, purchase, or donation is required to use the calculator.
func (a *App) openBuyMeACoffee() {
	ret, _, _ := procShellExecuteW.Call(uintptr(a.hwnd),
		uintptr(unsafe.Pointer(utf16Ptr("open"))),
		uintptr(unsafe.Pointer(utf16Ptr(buyMeACoffeeURL))),
		0, 0, SW_SHOW)
	if ret <= 32 {
		procSupportMessageBoxW.Call(uintptr(a.hwnd),
			uintptr(unsafe.Pointer(utf16Ptr("Could not open the support page.\r\nVisit: "+buyMeACoffeeURL))),
			uintptr(unsafe.Pointer(utf16Ptr("RAWDOGS — Buy Me a Coffee"))), 0)
	}
}

func (a *App) setManualPage(page string) {
	if page == "" {
		page = "quick"
	}
	if a.manualPage != page {
		a.narratorStop(true) // Page switches silence old Piper audio; never auto-play next page.
		a.narratorMu.Lock()
		a.narratorPage = page
		a.narratorDurations = nil // Never show a different page's timing information.
		a.narratorPendingSeek = -1
		a.narratorMu.Unlock()
	}
	a.manualPage = page
	a.narratorVisibleSection = -1
	show := func(hs []syscall.Handle, on bool) {
		cmd := uintptr(SW_HIDE)
		if on {
			cmd = SW_SHOW
		}
		for _, h := range hs {
			if h != 0 {
				procShowWindow.Call(uintptr(h), cmd)
			}
		}
	}
	// A Field Manual page may be selected while Guided Setup is still on
	// SETTINGS (notably immediately after SAVE TO DISK). Selecting a page is
	// NOT permission to show its controls over another screen.
	show(a.manualQuick, manualPageControlsVisible(a.currentScreen, page, "quick"))
	show(a.manualCues, manualPageControlsVisible(a.currentScreen, page, "cues"))
	show(a.manualVoices, manualPageControlsVisible(a.currentScreen, page, "voices"))
	show(a.manualNarrated, manualPageControlsVisible(a.currentScreen, page, "narrated"))
	show(a.manualModels, manualPageControlsVisible(a.currentScreen, page, "models"))
	show(a.manualLibraries, manualPageControlsVisible(a.currentScreen, page, "libraries"))
	if page == "libraries" && a.currentScreen == "manual" {
		a.onLibraryPageShown()
	}
	a.narratorShowSection()
	a.syncNarratorSeek()
	procInvalidateRect.Call(uintptr(a.hwnd), 0, 1)
}

func (a *App) updateManualLive() {
	if a.controls[6200] == 0 {
		return
	}
	state := a.currentListenerState()
	setText(a.controls[6200], "LISTENER: "+state)

	ctx := "CURRENT CONTEXT: IDLE — say REFRESH ACTUAL / ACTUALLY"
	switch a.pending.Kind {
	case "actual":
		ctx = fmt.Sprintf("CURRENT CONTEXT: USER ACTUAL — X %s / Y %s", emptyDash(a.pending.X), emptyDash(a.pending.Y))
	case "switch_actual":
		ctx = fmt.Sprintf("CURRENT CONTEXT: SWITCHING ACTUAL — X %s / Y %s", emptyDash(a.pending.X), emptyDash(a.pending.Y))
	case "target":
		label := "UNSAVED TARGET"
		if a.pending.Slot >= 0 {
			label = fmt.Sprintf("TARGET %d", a.pending.Slot+1)
		}
		ctx = fmt.Sprintf("CURRENT CONTEXT: %s — X %s / Y %s", label, emptyDash(a.pending.X), emptyDash(a.pending.Y))
	case "clear_all":
		ctx = "CURRENT CONTEXT: TARGETS CLEARED — say UNDO CLEAR to restore"
	case "clear_actual":
		ctx = "CURRENT CONTEXT: CLEAR ACTUAL — waiting for OVER or CANCEL"
	}
	setText(a.controls[6201], ctx)

	mark := func(ok bool) string {
		if ok {
			return "✓"
		}
		return "○"
	}
	listenerOK := state == "READY — SAY COMMAND" || state == "STARTING"
	actualPending := a.pending.Kind == "actual" || a.pending.Kind == "switch_actual"
	actualStarted := actualPending || a.actualSet
	actualXOK := a.actualSet || (actualPending && a.pending.X != "")
	actualYOK := a.actualSet || (actualPending && a.pending.Y != "")
	targetStarted := a.pending.Kind == "target" || a.activeTarget != -1
	targetXOK := a.activeTarget != -1 || (a.pending.Kind == "target" && a.pending.X != "")
	targetYOK := a.activeTarget != -1 || (a.pending.Kind == "target" && a.pending.Y != "")
	progress := fmt.Sprintf("%s Listener live\r\n%s STEP 1 — ACTUAL\r\n%s X / %s Y — STEPS 2–3\r\n%s STEP 4 — TARGET\r\n%s X / %s Y — STEPS 5–6",
		mark(listenerOK), mark(actualStarted), mark(actualXOK), mark(actualYOK), mark(targetStarted), mark(targetXOK), mark(targetYOK))
	setText(a.controls[6203], progress)

	track := "No operational commands yet."
	if len(a.operationLog) > 0 {
		rows := a.operationLog
		if len(rows) > 7 {
			rows = rows[len(rows)-7:]
		}
		track = strings.Join(rows, "\r\n")
	}
	setText(a.controls[6202], track)
}

func (a *App) addStatic(id int, x, y, w, h int, text string, bold bool) syscall.Handle {
	hw := a.createControl(id, "STATIC", text, WS_CHILD|WS_VISIBLE|SS_LEFT, x, y, w, h, 0, "")
	if y <= 20 {
		a.setFont(hw, a.fontTitle)
	} else if bold {
		a.setFont(hw, a.fontBold)
	} else {
		a.setFont(hw, a.fontNormal)
	}
	return hw
}
func (a *App) measureTextWidth(text string, bold bool) int {
	if text == "" {
		return 0
	}
	hdc, _, _ := procGetDC.Call(uintptr(a.hwnd))
	if hdc == 0 {
		return len([]rune(text))*10 + 12
	}
	font := a.fontNormal
	if bold {
		font = a.fontBold
	}
	old, _, _ := procSelectObject.Call(hdc, uintptr(font))
	u := utf16Ptr(text)
	uu, _ := syscall.UTF16FromString(text)
	var sz SIZE
	procGetTextExtentPoint32W.Call(hdc, uintptr(unsafe.Pointer(u)), uintptr(len(uu)-1), uintptr(unsafe.Pointer(&sz)))
	procSelectObject.Call(hdc, old)
	procReleaseDC.Call(uintptr(a.hwnd), hdc)
	return int(sz.Cx)
}

func (a *App) addScreenStatic(screen string, x, y, w, h int, text string, bold bool) syscall.Handle {
	// One-line labels use their measured rendered width plus a small pad. The
	// caller's width is the hard panel-safe maximum, so labels can neither
	// create giant backplates nor cross a card border.
	if h <= 30 && !strings.Contains(text, "\r") && !strings.Contains(text, "\n") {
		need := a.measureTextWidth(text, bold) + 12
		if need > 24 && need < w {
			w = need
		}
	}
	hwnd := a.createControl(0, "STATIC", text, WS_CHILD|WS_VISIBLE|SS_LEFT, x, y, w, h, 0, screen)
	if h <= 30 && !strings.Contains(text, "\r") && !strings.Contains(text, "\n") {
		a.labelPlate[hwnd] = true
	}
	if bold {
		a.setFont(hwnd, a.fontBold)
	} else {
		a.setFont(hwnd, a.fontNormal)
	}
	return hwnd
}
func (a *App) addScreenStaticWithID(screen string, id, x, y, w, h int, text string, bold bool) syscall.Handle {
	style := uintptr(WS_CHILD | WS_VISIBLE | SS_LEFT)
	if id >= 5020 && id <= 5024 {
		style |= 0x0100 /* SS_NOTIFY */
	}
	hw := a.createControl(id, "STATIC", text, style, x, y, w, h, 0, screen)
	if bold {
		a.setFont(hw, a.fontBold)
	} else {
		a.setFont(hw, a.fontNormal)
	}
	return hw
}
func (a *App) addButton(id, x, y, w, h int, text, screen string) syscall.Handle {
	hw := a.createControl(id, "BUTTON", text, WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_OWNERDRAW, x, y, w, h, 0, screen)
	a.setFont(hw, a.fontButton)
	return hw
}
func (a *App) addScreenButton(screen string, id, x, y, w, h int, text string) syscall.Handle {
	return a.addButton(id, x, y, w, h, text, screen)
}
func (a *App) addScreenCombo(screen string, id, x, y, w, h int) syscall.Handle {
	hw := a.createControl(id, "COMBOBOX", "", WS_CHILD|WS_VISIBLE|WS_TABSTOP|CBS_DROPDOWNLIST|CBS_HASSTRINGS|WS_VSCROLL, x, y, w, h, 0, screen)
	a.setFont(hw, a.fontNormal)
	return hw
}

func (a *App) createControl(id int, class, text string, style uintptr, x, y, w, h int, exStyle uintptr, screen string) syscall.Handle {
	hwnd, _, _ := procCreateWindowExW.Call(exStyle, uintptr(unsafe.Pointer(utf16Ptr(class))), uintptr(unsafe.Pointer(utf16Ptr(text))), style,
		uintptr(x), uintptr(y), uintptr(w), uintptr(h), uintptr(a.hwnd), uintptr(id), 0, 0)
	hnd := syscall.Handle(hwnd)
	if hnd != 0 && a.controlBase != nil {
		a.controlBase[hnd] = RECT{Left: int32(x), Top: int32(y), Right: int32(x + w), Bottom: int32(y + h)}
	}
	if id != 0 {
		a.controls[id] = hnd
	}
	if screen != "" && screen != "nav" {
		a.screenControls[screen] = append(a.screenControls[screen], hnd)
	}
	return hnd
}

func (a *App) populateStaticCombos() {
	a.setComboItems(ID_GAIN_COMBO, []string{"0 dB (OFF)", "+3 dB", "+6 dB", "+9 dB", "+12 dB"}, 0)
	conf := []string{"60%", "65%", "70%", "75%", "80%", "85%", "90%", "95%"}
	a.setComboItems(ID_CONF_COMBO, conf, 4)
	a.setComboItems(ID_SPEED_COMBO, piperSpeedOptions, 2)
	a.setComboItems(ID_VOLUME_COMBO, piperVolumeOptions, 10)
	a.setComboItems(ID_THEME_COMBO, []string{"Dark", "Light"}, 0)
	a.setComboItems(ID_TEAM_COMBO, []string{"MANTICORE — GREEN", "LONESTAR — BLUE", "VALKYRA — RED"}, 0)
	a.refreshInstalledLibrarySelectors()
	a.updateVoskLibraryStatus()
	a.updateActualVoiceLibraryStatus()
}

func (a *App) setComboItems(id int, items []string, sel int) {
	h := a.controls[id]
	if h == 0 {
		return
	}
	procSendMessageW.Call(uintptr(h), CB_RESETCONTENT, 0, 0)
	for _, s := range items {
		procSendMessageW.Call(uintptr(h), CB_ADDSTRING, 0, uintptr(unsafe.Pointer(utf16Ptr(s))))
	}
	if len(items) > 0 && sel >= 0 {
		if sel >= len(items) {
			sel = 0
		}
		procSendMessageW.Call(uintptr(h), CB_SETCURSEL, uintptr(sel), 0)
	}
}

func (a *App) comboSelection(id int) (int, string) {
	h := a.controls[id]
	if h == 0 {
		return -1, ""
	}
	idx, _, _ := procSendMessageW.Call(uintptr(h), CB_GETCURSEL, 0, 0)
	if int32(idx) < 0 {
		return -1, ""
	}
	n, _, _ := procSendMessageW.Call(uintptr(h), CB_GETLBTEXTLEN, idx, 0)
	buf := make([]uint16, n+2)
	procSendMessageW.Call(uintptr(h), CB_GETLBTEXT, idx, uintptr(unsafe.Pointer(&buf[0])))
	return int(idx), syscall.UTF16ToString(buf)
}

func (a *App) selectComboText(id int, text string) {
	h := a.controls[id]
	if h == 0 {
		return
	}
	// iterate known list via combobox GETCOUNT/GETLBTEXT could be used; easier based on IDs.
	var items []string
	switch id {
	case ID_MIC_COMBO:
		items = a.micNames
	case ID_OUTPUT_COMBO:
		items = a.outputNames
	case ID_VOICE_COMBO:
		items = a.voices
	case ID_LANG_COMBO:
		items = a.languages
	case ID_VOSK_LIBRARY_COMBO:
		items = a.installedVoskLabels()
	case ID_ACTUAL_LIBRARY_COMBO:
		items = a.installedPiperLabels()
	case ID_GAIN_COMBO:
		items = []string{"0 dB (OFF)", "+3 dB", "+6 dB", "+9 dB", "+12 dB"}
	case ID_CONF_COMBO:
		items = []string{"60%", "65%", "70%", "75%", "80%", "85%", "90%", "95%"}
	case ID_SPEED_COMBO:
		items = piperSpeedOptions
	case ID_VOLUME_COMBO:
		items = piperVolumeOptions
	case ID_THEME_COMBO:
		items = []string{"Dark", "Light"}
	case ID_TEAM_COMBO:
		items = []string{"MANTICORE — GREEN", "LONESTAR — BLUE", "VALKYRA — RED"}
	}
	for i, s := range items {
		if strings.EqualFold(s, text) {
			procSendMessageW.Call(uintptr(h), CB_SETCURSEL, uintptr(i), 0)
			return
		}
	}
}

func (a *App) updateAllFromSettings() {
	a.selectComboText(ID_THEME_COMBO, a.settings.Theme)
	a.selectComboText(ID_TEAM_COMBO, a.settings.Team)
	gainMap := map[float64]string{0: "0 dB (OFF)", 3: "+3 dB", 6: "+6 dB", 9: "+9 dB", 12: "+12 dB"}
	if s := gainMap[a.settings.GainDB]; s != "" {
		a.selectComboText(ID_GAIN_COMBO, s)
	}
	a.selectComboText(ID_CONF_COMBO, fmt.Sprintf("%d%%", a.settings.Confidence))
	a.selectComboText(ID_SPEED_COMBO, fmt.Sprintf("%.2fx", a.settings.VoiceSpeed))
	a.selectComboText(ID_VOLUME_COMBO, fmt.Sprintf("%d%%", a.settings.VoiceVolume))
	a.syncPiperMuteButton()
	if spec, ok := piperSpecByKey(a.settings.ActualVoice); ok {
		a.selectComboText(ID_ACTUAL_LIBRARY_COMBO, spec.Label)
	}
	a.updateActualVoiceLibraryStatus()
}

func (a *App) captureSettingsFromUI() {
	idx, s := a.comboSelection(ID_MIC_COMBO)
	if idx >= 0 {
		a.settings.MicIndex = idx
		a.settings.MicName = s
	}
	idx, s = a.comboSelection(ID_OUTPUT_COMBO)
	if idx >= 0 {
		a.settings.OutputIndex = idx
		a.settings.OutputName = s
	}
	idx, s = a.comboSelection(ID_LANG_COMBO)
	if idx >= 0 {
		a.settings.Language = s
	}
	idx, s = a.comboSelection(ID_VOICE_COMBO)
	if idx >= 0 {
		a.settings.VoiceIndex = idx
		a.settings.VoiceName = s
	}
	_, s = a.comboSelection(ID_GAIN_COMBO)
	a.settings.GainDB = parseGain(s)
	_, s = a.comboSelection(ID_CONF_COMBO)
	a.settings.Confidence = parsePct(s, 80)
	_, s = a.comboSelection(ID_SPEED_COMBO)
	a.settings.VoiceSpeed = parseSpeed(s, 1)
	_, s = a.comboSelection(ID_VOLUME_COMBO)
	a.settings.VoiceVolume = parsePct(s, 100)
	_, s = a.comboSelection(ID_THEME_COMBO)
	if s != "" {
		a.settings.Theme = s
	}
	_, s = a.comboSelection(ID_TEAM_COMBO)
	if s != "" {
		a.settings.Team = s
	}
}

func parseGain(s string) float64 {
	s = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(s, "+"), " dB"))
	if strings.HasPrefix(s, "0 dB") {
		return 0
	}
	f, _ := strconv.ParseFloat(s, 64)
	return f
}
func parsePct(s string, def int) int {
	n, err := strconv.Atoi(strings.TrimSuffix(strings.TrimSpace(s), "%"))
	if err != nil {
		return def
	}
	return n
}
func parseSpeed(s string, def float64) float64 {
	f, err := strconv.ParseFloat(strings.TrimSuffix(strings.TrimSpace(s), "x"), 64)
	if err != nil {
		return def
	}
	return f
}

func (a *App) showScreen(name string) {
	// Leaving the Training tab ends calibration so an operator cannot
	// unknowingly play with operational voice commands intercepted.
	// The app may still be off-screen behind WARDOGS during a training session.
	if name != "training" && a.voiceTrain.Active {
		a.voiceTrainingStop()
	}
	for screen, hs := range a.screenControls {
		cmd := uintptr(SW_HIDE)
		if screen == name {
			cmd = SW_SHOW
		}
		for _, h := range hs {
			procShowWindow.Call(uintptr(h), cmd)
		}
	}
	a.currentScreen = name
	procInvalidateRect.Call(uintptr(a.hwnd), 0, 1)
	if name == "settings" {
		a.refreshInstalledLibrarySelectors()
		a.updateSetupStatus()
	}
	if name == "operations" {
		a.resizeOperationalDiagnostics()
		a.refreshOperationsUI()
	}
	if name == "training" {
		a.ensureVoiceProfile()
		a.layoutVoiceAdaptation()
		a.refreshVoiceTrainingUI()
	}
	if name == "manual" {
		a.setManualPage(a.manualPage)
		a.updateManualLive()
	}
	a.syncListenerControls()
}

func (a *App) updateSetupStatus() {
	if a.currentScreen != "settings" {
		return
	}
	if a.tourActive && a.tourIndex >= 0 && a.tourIndex < len(guidedTour) {
		sc := guidedTour[a.tourIndex]
		setText(a.controls[3004], fmt.Sprintf("GUIDE %d/%d — %s\r\n%s", a.tourIndex+1, len(guidedTour), sc.Title, sc.Narration))
		return
	}
	if a.guideActive {
		a.applyGuideStep(false)
		return
	}
	if a.firstRun {
		setText(a.controls[3004], "SYSTEM STATUS: FIRST RUN — CHOOSE PERMISSION, DEVICES AND VOICE SETTINGS")
	} else {
		setText(a.controls[3004], "SYSTEM STATUS: SAVED SETUP LOADED — SAFE TO TEST / CHANGE")
	}
}

func (a *App) startGuidedSetup() {
	a.startTour()
}

func (a *App) finishGuidedSetup(completed bool) {
	if a == nil {
		return
	}
	if a.manualNarrating {
		a.endNarratedManual()
	}
	a.guideActive = false
	a.guidePulse = false
	a.guideOutputTesting = false
	a.guideOutputAwaiting = false
	a.guideInfoSpeechStep = -1
	procKillTimer.Call(uintptr(a.hwnd), 77)
	setText(a.controls[ID_GUIDE_NEXT], "START GUIDE")
	procEnableWindow.Call(uintptr(a.controls[ID_GUIDE_NEXT]), 1)
	setText(a.controls[ID_SETTINGS_LISTEN], "START LISTENING")
	setText(a.controls[ID_SETTINGS_STOP], "STOP LISTENING")
	if completed && a.guideMarker != "" {
		_ = os.WriteFile(a.guideMarker, []byte("RAWDOGS G13 guided setup completed\r\n"), 0644)
	}
	if a.currentScreen == "settings" {
		if completed {
			setText(a.controls[3004], "GUIDED SETUP COMPLETE — COMMS READY. SETTINGS CAN BE CHANGED LATER; SAVE TO DISK REPLACES THE PREVIOUS SAVED SETUP.")
		} else {
			a.updateSetupStatus()
		}
	}
	a.syncListenerControls()
	procInvalidateRect.Call(uintptr(a.hwnd), 0, 1)
}

func (a *App) guideStepSatisfied() bool {
	a.captureSettingsFromUI()
	switch a.guideStep {
	case 0:
		return a.settings.Permission == "this_run" || a.settings.Permission == "always"
	case 1:
		idx, _ := a.comboSelection(ID_MIC_COMBO)
		return idx >= 0 && a.micTested
	case 2:
		idx, _ := a.comboSelection(ID_OUTPUT_COMBO)
		return idx >= 0 && a.guideOutputHeard
	case 3, 4:
		return false // explanatory steps advance when ACTUAL finishes speaking
	case 5:
		return a.guideSaved
	case 7:
		a.recoMu.Lock()
		ok := a.recoRunning && a.recoStatus == "READY"
		a.recoMu.Unlock()
		return ok
	}
	return false
}

func (a *App) guideWaitingText() string {
	switch a.guideStep {
	case 0:
		return "GUIDE WAITING — CHOOSE ALLOW THIS RUN OR ALWAYS ALLOW. NOT NOW LEAVES VOICE SETUP FOR LATER."
	case 1:
		return "GUIDE WAITING — SELECT A MICROPHONE, PRESS TEST MIC, THEN SPEAK SO THE LEVEL METER CAN CONFIRM INPUT."
	case 2:
		if a.guideOutputAwaiting {
			return "DID YOU HEAR ACTUAL? CHOOSE HEARD ACTUAL OR TRY AGAIN."
		}
		return "GUIDE WAITING — SELECT ACTUAL'S OUTPUT DEVICE. THE FIRST-RUN WELCOME PHRASE WILL PLAY AUTOMATICALLY."
	case 5:
		return "GUIDE WAITING — PRESS SAVE SETUP TO DISK. THIS REPLACES THE PREVIOUS SAVED CONFIGURATION."
	case 6:
		return "GUIDE WAITING — PRESS START LISTENING HERE IN SETTINGS."
	}
	return "GUIDE ACTIVE"
}

func (a *App) advanceGuideAutomatic() {
	if !a.guideActive {
		return
	}
	a.guideStep++
	if a.guideStep > 7 {
		a.finishGuidedSetup(true)
		return
	}
	a.applyGuideStep(true)
}

func (a *App) guidePrimaryTargets() []int {
	switch a.guideStep {
	case 0:
		return []int{ID_PERMISSION_RUN, ID_PERMISSION_ALWAYS}
	case 1:
		return []int{ID_MIC_COMBO, ID_TEST_MIC}
	case 2:
		if a.guideOutputAwaiting {
			return []int{ID_SETTINGS_LISTEN, ID_SETTINGS_STOP}
		}
		return []int{ID_OUTPUT_COMBO}
	case 3:
		return []int{ID_ACTUAL_LIBRARY_COMBO, ID_ACTUAL_INSTALL_SELECTED, ID_ACTUAL_USE_SELECTED, ID_TEST_VOICE}
	case 4:
		return []int{ID_VOSK_LIBRARY_COMBO, ID_VOSK_INSTALL_SELECTED, ID_VOSK_USE_SELECTED}
	case 5:
		return []int{ID_SAVE_SETUP}
	case 6:
		return []int{ID_GUIDE_END}
	case 7:
		return []int{ID_SETTINGS_LISTEN}
	}
	return nil
}

func (a *App) guideInstruction() (string, string) {
	switch a.guideStep {
	case 0:
		return "GUIDED SETUP 1/8 — MICROPHONE PERMISSION\r\nChoose ALLOW THIS RUN or ALWAYS ALLOW. The selected button completes this step automatically.", ""
	case 1:
		return "GUIDED SETUP 2/8 — YOUR MICROPHONE\r\nSelect your microphone, press TEST MIC, then speak normally. When RAWDOGS detects live input this step completes automatically.", ""
	case 2:
		if a.guideOutputAwaiting {
			return "GUIDED SETUP 3/8 — ACTUAL OUTPUT CONFIRMATION\r\nDid you hear ACTUAL? Use HEARD ACTUAL or TRY AGAIN below. You can change the output device before retrying.", ""
		}
		return "GUIDED SETUP 3/8 — ACTUAL OUTPUT\r\nSelect the speakers or headset where ACTUAL should speak. RAWDOGS will immediately play the dedicated first-run welcome phrase.", ""
	case 3:
		return "GUIDED SETUP 4/8 — PIPER VOICE\r\nPiper controls the voice RAWDOGS uses to speak. The installed default is already active; this area can be changed later.", "This chooses the voice RAWDOGS uses to speak. Piper is the ACTUAL voice system."
	case 4:
		return "GUIDED SETUP 5/8 — VOSK MODEL\r\nVosk controls how RAWDOGS listens to and interprets your speech. The installed default is already active; other models can be chosen later.", "This chooses how RAWDOGS listens to and interprets your speech. Vosk is the recognition system."
	case 5:
		return "GUIDED SETUP 6/8 — SAVE TO DISK\r\nPress SAVE SETUP TO DISK. Later changes to microphone, output, Piper or Vosk should be re-saved; saving replaces the previous configuration.", "Voice and recognition settings can be changed at any time in Settings. Remember to Save to Disk after making changes. Saving replaces the previous saved configuration."
	case 6:
		return "GUIDED SETUP 7/8 — NARRATED FIELD MANUAL\r\nSetup saved. Open FIELD MANUAL, select NARRATED GUIDE, then press PLAY. Piper reads the text there, not over SETTINGS. Return here and choose END GUIDE when finished.", ""
	case 7:
		return "GUIDED SETUP 8/8 — BRING LISTENER ONLINE\r\nPress START LISTENING here in Settings. When Vosk reports READY the guide completes automatically.", ""
	}
	return "GUIDED SETUP", ""
}

func (a *App) syncGuideOutputButtons() {
	if !a.guideActive || a.guideStep != 2 {
		return
	}
	if a.guideOutputTesting {
		setText(a.controls[ID_SETTINGS_LISTEN], "ACTUAL SPEAKING...")
		setText(a.controls[ID_SETTINGS_STOP], "PLEASE WAIT")
		procEnableWindow.Call(uintptr(a.controls[ID_SETTINGS_LISTEN]), 0)
		procEnableWindow.Call(uintptr(a.controls[ID_SETTINGS_STOP]), 0)
		return
	}
	if a.guideOutputAwaiting {
		setText(a.controls[ID_SETTINGS_LISTEN], "HEARD ACTUAL")
		setText(a.controls[ID_SETTINGS_STOP], "TRY AGAIN")
		procEnableWindow.Call(uintptr(a.controls[ID_SETTINGS_LISTEN]), 1)
		procEnableWindow.Call(uintptr(a.controls[ID_SETTINGS_STOP]), 1)
		return
	}
	setText(a.controls[ID_SETTINGS_LISTEN], "WAITING FOR OUTPUT")
	setText(a.controls[ID_SETTINGS_STOP], "TRY AGAIN")
	procEnableWindow.Call(uintptr(a.controls[ID_SETTINGS_LISTEN]), 0)
	procEnableWindow.Call(uintptr(a.controls[ID_SETTINGS_STOP]), 1)
}

func (a *App) startFirstRunActualTest() {
	if !a.guideActive || a.guideStep != 2 || a.guideOutputTesting {
		return
	}
	a.captureSettingsFromUI()
	idx, _ := a.comboSelection(ID_OUTPUT_COMBO)
	if idx < 0 {
		setText(a.controls[3004], "GUIDE WAITING — SELECT ACTUAL'S OUTPUT DEVICE.")
		return
	}
	if !a.actualVoiceReady() {
		setText(a.controls[3004], "ACTUAL DEFAULT VOICE IS NOT READY — REPLAY START GUIDE TO REPAIR THE DEFAULT PIPER PACK.")
		return
	}
	a.guideOutputTesting = true
	a.guideOutputAwaiting = false
	a.guideOutputHeard = false
	a.syncGuideOutputButtons()
	setText(a.controls[3004], "ACTUAL OUTPUT TEST — LISTEN FOR THE FIRST-RUN WELCOME PHRASE.")
	a.speakActualAsync("Welcome. This is ACTUAL. Stand by for a quick tour of duty.")
	procInvalidateRect.Call(uintptr(a.hwnd), 0, 1)
}

func (a *App) applyGuideStep(speak bool) {
	if !a.guideActive || a.currentScreen != "settings" {
		return
	}
	msg, voice := a.guideInstruction()
	setText(a.controls[3004], msg)
	setText(a.controls[ID_GUIDE_NEXT], "GUIDE ACTIVE")
	procEnableWindow.Call(uintptr(a.controls[ID_GUIDE_NEXT]), 0)
	if a.guideStep == 2 {
		a.syncGuideOutputButtons()
		if !a.guideOutputTesting && !a.guideOutputAwaiting {
			if idx, _ := a.comboSelection(ID_OUTPUT_COMBO); idx >= 0 {
				a.startFirstRunActualTest()
			}
		}
	} else {
		setText(a.controls[ID_SETTINGS_LISTEN], "START LISTENING")
		setText(a.controls[ID_SETTINGS_STOP], "STOP LISTENING")
		a.syncListenerControls()
	}
	procInvalidateRect.Call(uintptr(a.hwnd), 0, 1)
	if a.guideStep == 6 && speak {
		// Preselect the narrated page for later navigation, but never display
		// its controls on top of Settings after Save to Disk.
		a.setManualPage("narrated")
		setText(a.controls[6204], "PRESS PLAY TO BEGIN NARRATION")
		return
	}
	if speak && voice != "" && a.actualVoiceReady() {
		if a.guideStep == 3 || a.guideStep == 4 {
			a.guideInfoSpeechStep = a.guideStep
		} else {
			a.guideInfoSpeechStep = -1
		}
		if a.guideStep != 5 || !a.guideSaveReminderSpoken {
			a.speakActualAsync(voice)
			if a.guideStep == 5 {
				a.guideSaveReminderSpoken = true
			}
		}
	}
}

func (a *App) drawGuideHighlight(hdc syscall.Handle, width int) {
	if !a.guideActive || a.currentScreen != "settings" {
		return
	}
	var client RECT
	procGetClientRect.Call(uintptr(a.hwnd), uintptr(unsafe.Pointer(&client)))
	clipped := false
	for _, id := range a.guidePrimaryTargets() {
		h := a.controls[id]
		br, ok := a.controlBase[h]
		if h == 0 || !ok {
			continue
		}
		x := a.registerX(int(br.Left), width)
		ww := a.registerWidth(int(br.Right-br.Left), width)
		y := int(br.Top)
		hh := int(br.Bottom - br.Top)
		// Native COMBOBOX rectangles include the *hidden dropdown list*.
		// Highlight only the 30px closed selection field. Previously this
		// outlined the 220-260px list area behind neighbouring panels.
		isCombo := id == ID_MIC_COMBO || id == ID_OUTPUT_COMBO ||
			id == ID_ACTUAL_LIBRARY_COMBO || id == ID_VOSK_LIBRARY_COMBO
		hh = guideVisibleHeight(isCombo, hh)
		if y+hh+4 > int(client.Bottom) {
			clipped = true
			continue
		}
		for pad := 2; pad <= 3; pad++ {
			rc := RECT{int32(x - pad), int32(y - pad), int32(x + ww + pad), int32(y + hh + pad)}
			frameColor(hdc, rc, 255, 0, 170)
		}
	}
	if clipped {
		// G18: keep the guide visible on smaller windows without moving a
		// single established panel or control. The marker reports what is offscreen.
		x := a.registerX(406, width)
		ww := a.registerWidth(680, width)
		r := RECT{int32(x), 172, int32(x + ww), 200}
		frameColor(hdc, r, 255, 0, 170)
		procSetBkMode.Call(uintptr(hdc), TRANSPARENT)
		procSetTextColor.Call(uintptr(hdc), rgb(255, 60, 190))
		msg := fmt.Sprintf("GUIDE %d/7: HIGHLIGHTED CONTROLS BELOW — ENLARGE WINDOW HEIGHT", a.guideStep+1)
		u := utf16Ptr(msg)
		procDrawTextW.Call(uintptr(hdc), uintptr(unsafe.Pointer(u)), ^uintptr(0), uintptr(unsafe.Pointer(&r)), DT_VCENTER|DT_SINGLELINE|DT_END_ELLIPSIS)
	}
}

func setText(h syscall.Handle, s string) {
	if h == 0 {
		return
	}
	// G18H: repaint a text control only if the *visible* text has changed.
	// A Vosk feed tick must not repeatedly erase/recreate unchanged labels.
	n, _, _ := procGetWindowTextLengthW.Call(uintptr(h))
	if n < 2048 {
		buf := make([]uint16, int(n)+1)
		procGetWindowTextW.Call(uintptr(h), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
		if syscall.UTF16ToString(buf) == s {
			return
		}
	}
	procSetWindowTextW.Call(uintptr(h), uintptr(unsafe.Pointer(utf16Ptr(s))))
}

func (a *App) scanAudioAsync(user bool) {
	if a.scanInProgress {
		return
	}
	a.scanInProgress = true
	if user {
		setText(a.controls[3004], "SYSTEM STATUS: RESCANNING WINDOWS AUDIO...")
	}
	inputs := enumerateWaveIn()
	outputs := enumerateWaveOut()
	a.micNames = inputs
	a.outputNames = outputs
	a.micMeterMap = make([]int, len(inputs))
	for i := range a.micMeterMap {
		a.micMeterMap[i] = i
	}
	procPostMessageW.Call(uintptr(a.hwnd), WM_APP_SCAN_DONE, 0, 0)
}

func enumerateWaveIn() []string {
	n, _, _ := procWaveInGetNumDevs.Call()
	out := []string{}
	for i := uintptr(0); i < n; i++ {
		var c WAVEINCAPS
		r, _, _ := procWaveInGetDevCaps.Call(i, uintptr(unsafe.Pointer(&c)), unsafe.Sizeof(c))
		if r == 0 {
			name := utf16ToString(c.Pname[:])
			if name == "" {
				name = fmt.Sprintf("Input device %d", i)
			}
			out = append(out, name)
		}
	}
	return out
}

func psEsc(s string) string { return strings.ReplaceAll(s, "'", "''") }

func hiddenCommand(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: CREATE_NO_WINDOW}
	return cmd
}

func enumerateSapiInputs() []string {
	ps := `$ErrorActionPreference='SilentlyContinue'; $r=New-Object -ComObject LEGACY-SPEECH.SpInprocRecognizer; if($r){ for($i=0;$i -lt $r.GetAudioInputs().Count;$i++){ $r.GetAudioInputs().Item($i).GetDescription() } }`
	out, err := hiddenCommand("powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", ps).Output()
	if err != nil {
		return nil
	}
	result := []string{}
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	for sc.Scan() {
		if v := strings.TrimSpace(sc.Text()); v != "" {
			result = appendUnique(result, v)
		}
	}
	return result
}

func normDevice(s string) string {
	s = strings.ToLower(s)
	r := regexp.MustCompile(`[^a-z0-9]+`)
	return r.ReplaceAllString(s, "")
}

func mapMicToWave(sapi, wave []string) []int {
	out := make([]int, len(sapi))
	for i := range out {
		out[i] = -1
	}
	for i, s := range sapi {
		ns := normDevice(s)
		best := -1
		for j, w := range wave {
			nw := normDevice(w)
			if ns == nw || strings.Contains(ns, nw) || strings.Contains(nw, ns) {
				best = j
				break
			}
		}
		out[i] = best
	}
	return out
}

func enumerateSapiOutputs() []string {
	ps := `$ErrorActionPreference='SilentlyContinue'; $sp=New-Object -ComObject LEGACY-SPEECH.SpVoice; if($sp){ for($i=0;$i -lt $sp.GetAudioOutputs().Count;$i++){ $sp.GetAudioOutputs().Item($i).GetDescription() } }`
	out, err := hiddenCommand("powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", ps).Output()
	if err != nil {
		return nil
	}
	result := []string{}
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	for sc.Scan() {
		v := strings.TrimSpace(sc.Text())
		if v != "" {
			result = appendUnique(result, v)
		}
	}
	return result
}

func enumerateWaveOut() []string {
	n, _, _ := procWaveOutGetNumDevs.Call()
	out := []string{}
	for i := uintptr(0); i < n; i++ {
		var c WAVEOUTCAPS
		r, _, _ := procWaveOutGetDevCaps.Call(i, uintptr(unsafe.Pointer(&c)), unsafe.Sizeof(c))
		if r == 0 {
			name := utf16ToString(c.Pname[:])
			if name == "" {
				name = fmt.Sprintf("Output device %d", i)
			}
			out = append(out, name)
		}
	}
	return out
}

func (a *App) applyAudioScan() {
	oldMic := a.settings.MicName
	if _, s := a.comboSelection(ID_MIC_COMBO); s != "" {
		oldMic = s
	}
	oldOut := a.settings.OutputName
	if _, s := a.comboSelection(ID_OUTPUT_COMBO); s != "" {
		oldOut = s
	}
	micSel, outSel := 0, 0
	if a.firstRun && oldMic == "" {
		micSel = -1
	}
	if a.firstRun && oldOut == "" {
		outSel = -1
	}
	a.setComboItems(ID_MIC_COMBO, a.micNames, micSel)
	a.setComboItems(ID_OUTPUT_COMBO, a.outputNames, outSel)
	if oldMic != "" {
		a.selectComboText(ID_MIC_COMBO, oldMic)
	}
	if oldOut != "" {
		a.selectComboText(ID_OUTPUT_COMBO, oldOut)
	}
	if a.currentScreen == "settings" {
		setText(a.controls[3004], fmt.Sprintf("SYSTEM STATUS: AUDIO READY — %d INPUT / %d OUTPUT", len(a.micNames), len(a.outputNames)))
	}
	a.scanInProgress = false
	a.audioReady = true
	if a.provisionDone && a.provisionSucceeded && !a.guideActive {
		if _, err := os.Stat(a.guideMarker); err != nil {
			a.startGuidedSetup()
		}
	}
}

func (a *App) refreshVoiceLanguageAsync(user bool) {
	if a.voiceRefreshInProgress {
		return
	}
	a.voiceRefreshInProgress = true
	if user {
		setText(a.controls[3004], "SYSTEM STATUS: REFRESHING WINDOWS VOICES / RECOGNITION LANGUAGES...")
	}
	voices, langs := a.queryVoiceLanguage()
	a.voices = voices
	a.languages = langs
	procPostMessageW.Call(uintptr(a.hwnd), WM_APP_VOICE_DONE, 0, 0)
}

func (a *App) queryVoiceLanguage() ([]string, []string) {
	voices := []string{}
	langs := []string{}
	for _, spec := range voskModelSpecs() {
		if a.voskModelInstalled(spec) {
			voices = append(voices, spec.ActiveDisplay)
			langs = appendUnique(langs, spec.Language)
		}
	}
	return voices, langs
}
func appendUnique(a []string, s string) []string {
	for _, v := range a {
		if v == s {
			return a
		}
	}
	return append(a, s)
}

func (a *App) applyVoiceLanguage() {
	oldV := a.settings.VoiceName
	if _, s := a.comboSelection(ID_VOICE_COMBO); s != "" {
		oldV = s
	}
	oldL := a.settings.Language
	if _, s := a.comboSelection(ID_LANG_COMBO); s != "" {
		oldL = s
	}
	voiceSel, langSel := 0, 0
	if a.firstRun && oldV == "" {
		voiceSel = -1
	}
	if a.firstRun && oldL == "" {
		langSel = -1
	}
	a.setComboItems(ID_VOICE_COMBO, a.voices, voiceSel)
	a.setComboItems(ID_LANG_COMBO, a.languages, langSel)
	if oldV != "" {
		a.selectComboText(ID_VOICE_COMBO, oldV)
	}
	if oldL != "" {
		a.selectComboText(ID_LANG_COMBO, oldL)
	}
	if a.voskReady() {
		_, active := a.comboSelection(ID_VOICE_COMBO)
		if active == "" {
			active = a.settings.VoiceName
		}
		setText(a.controls[3004], "SYSTEM STATUS: VOSK OFFLINE STT READY — "+active)
		setText(a.controls[3006], "STATUS: ACTIVE MODEL READY — "+active)
	} else if len(a.voices) > 0 {
		setText(a.controls[3004], "SYSTEM STATUS: INSTALLED VOSK MODEL AVAILABLE — SELECT ACTIVE MODEL ABOVE")
		setText(a.controls[3006], "STATUS: SELECT AN INSTALLED MODEL")
	} else {
		setText(a.controls[3004], "SYSTEM STATUS: NO VOSK MODEL INSTALLED — USE MODEL LIBRARY / MANAGER")
		setText(a.controls[3006], "STATUS: INSTALL A MODEL FROM THE LIBRARY BELOW")
	}
	a.voiceRefreshInProgress = false
	a.updateVoskLibraryStatus()
}

func (a *App) startStopMic() {
	if a.micRunning {
		a.stopMicTest()
		return
	}
	if a.settings.Permission == "" || a.settings.Permission == "not_now" {
		setText(a.controls[3002], "STATUS: CHOOSE ALLOW THIS RUN OR ALWAYS ALLOW FIRST")
		return
	}
	idx, name := a.comboSelection(ID_MIC_COMBO)
	if idx < 0 {
		setText(a.controls[3002], "STATUS: SELECT A MICROPHONE FIRST")
		return
	}
	_, gainText := a.comboSelection(ID_GAIN_COMBO)
	gain := parseGain(gainText)
	a.micStop = make(chan struct{})
	a.micRunning = true
	setText(a.controls[ID_TEST_MIC], "STOP MIC")
	waveIdx := idx
	if idx >= 0 && idx < len(a.micMeterMap) && a.micMeterMap[idx] >= 0 {
		waveIdx = a.micMeterMap[idx]
	}
	if idx >= 0 && idx < len(a.micMeterMap) && a.micMeterMap[idx] < 0 {
		a.micTested = true
		setText(a.controls[3002], fmt.Sprintf("STATUS: %s IS RECOGNISER-READY — LEVEL METER COULD NOT MAP TO LEGACY WAVE INPUT", name))
		return
	}
	setText(a.controls[3002], fmt.Sprintf("STATUS: TESTING %s — SPEAK NORMALLY", name))
	go a.runMicMeter(waveIdx, gain, a.micStop)
}

func (a *App) stopMicTest() {
	a.micMu.Lock()
	if !a.micRunning {
		a.micMu.Unlock()
		return
	}
	ch := a.micStop
	a.micRunning = false
	a.micMu.Unlock()
	if ch != nil {
		close(ch)
	}
	setText(a.controls[ID_TEST_MIC], "TEST MIC")
}

func (a *App) runMicMeter(device int, gainDB float64, stop <-chan struct{}) {
	var hwi uintptr
	format := WAVEFORMATEX{FormatTag: WAVE_FORMAT_PCM, Channels: 1, SamplesPerSec: 16000, BitsPerSample: 16, BlockAlign: 2, AvgBytesPerSec: 32000}
	r, _, _ := procWaveInOpen.Call(uintptr(unsafe.Pointer(&hwi)), uintptr(device), uintptr(unsafe.Pointer(&format)), 0, 0, CALLBACK_NULL)
	if r != 0 || hwi == 0 {
		a.micMu.Lock()
		a.micStatus = fmt.Sprintf("MIC OPEN FAILED — ERROR %d", r)
		a.micRunning = false
		a.micMu.Unlock()
		procPostMessageW.Call(uintptr(a.hwnd), WM_APP_MIC_LEVEL, 0, 0)
		return
	}
	defer procWaveInClose.Call(hwi)
	samples := make([]int16, 1600) // 100ms
	hdr := WAVEHDR{Data: (*byte)(unsafe.Pointer(&samples[0])), BufferLength: uint32(len(samples) * 2)}
	r, _, _ = procWaveInPrepare.Call(hwi, uintptr(unsafe.Pointer(&hdr)), unsafe.Sizeof(hdr))
	if r != 0 {
		a.micMu.Lock()
		a.micStatus = "MIC BUFFER PREPARE FAILED"
		a.micRunning = false
		a.micMu.Unlock()
		procPostMessageW.Call(uintptr(a.hwnd), WM_APP_MIC_LEVEL, 0, 0)
		return
	}
	defer procWaveInUnprepare.Call(hwi, uintptr(unsafe.Pointer(&hdr)), unsafe.Sizeof(hdr))
	procWaveInStart.Call(hwi)
	a.queueReco(RecoEvent{Kind: "MICREADY", Detail: "MICROPHONE LINK OPEN"})
	gain := math.Pow(10, gainDB/20)
	for {
		select {
		case <-stop:
			procWaveInStop.Call(hwi)
			procWaveInReset.Call(hwi)
			a.micMu.Lock()
			a.micStatus = "STOPPED"
			a.micPeak = 0
			a.micRunning = false
			a.micMu.Unlock()
			procPostMessageW.Call(uintptr(a.hwnd), WM_APP_MIC_LEVEL, 0, 0)
			return
		default:
		}
		hdr.BytesRecorded = 0
		hdr.Flags &^= WHDR_DONE
		r, _, _ := procWaveInAddBuffer.Call(hwi, uintptr(unsafe.Pointer(&hdr)), unsafe.Sizeof(hdr))
		if r != 0 {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		for hdr.Flags&WHDR_DONE == 0 {
			select {
			case <-stop:
				procWaveInStop.Call(hwi)
				procWaveInReset.Call(hwi)
				return
			default:
				time.Sleep(10 * time.Millisecond)
			}
		}
		count := int(hdr.BytesRecorded / 2)
		if count > len(samples) {
			count = len(samples)
		}
		peak := 0.0
		for i := 0; i < count; i++ {
			v := math.Abs(float64(samples[i])) / 32768.0 * gain
			if v > peak {
				peak = v
			}
		}
		pct := int(math.Min(100, peak*100))
		if pct >= 2 {
			a.micMu.Lock()
			a.micTested = true
			a.micMu.Unlock()
		}
		status := "GOOD INPUT"
		if pct < 8 {
			status = "MICROPHONE QUIET — TRY +3 OR +6 dB"
		} else if pct > 95 {
			status = "CLIPPING RISK — REDUCE GAIN"
		}
		a.micMu.Lock()
		a.micPeak = pct
		a.micStatus = status
		a.micMu.Unlock()
		procPostMessageW.Call(uintptr(a.hwnd), WM_APP_MIC_LEVEL, 0, 0)
		runtime.KeepAlive(samples)
		runtime.KeepAlive(hdr)
	}
}

func (a *App) applyMicLevel() {
	if a.currentScreen == "operations" && time.Since(a.lastDiagPaint) >= 250*time.Millisecond {
		a.lastDiagPaint = time.Now()
		procInvalidateRect.Call(uintptr(a.hwnd), 0, 0)
	}
	a.micMu.Lock()
	p := a.micPeak
	st := a.micStatus
	running := a.micRunning
	a.micMu.Unlock()
	bars := 30
	filled := p * bars / 100
	if filled > bars {
		filled = bars
	}
	meter := "[" + strings.Repeat("#", filled) + strings.Repeat("-", bars-filled) + "]"
	suffix := "STOPPED"
	if running {
		suffix = "LIVE"
	}
	setText(a.controls[3001], fmt.Sprintf("%s  %3d%%  %s", meter, p, suffix))
	if st != "" {
		setText(a.controls[3002], "STATUS: "+st)
	}
	if !running {
		setText(a.controls[ID_TEST_MIC], "TEST MIC")
	}
	if a.guideActive && a.guideStep == 1 && a.micTested {
		if a.micRunning {
			a.stopMicTest()
		}
		setText(a.controls[3004], "MICROPHONE CHECK PASSED — MOVING TO ACTUAL OUTPUT.")
		a.advanceGuideAutomatic()
	}
}

func (a *App) testVoice() {
	if !a.actualVoiceReady() {
		setText(a.controls[3004], "SYSTEM STATUS: ACTUAL'S VOICE IS NOT INSTALLED / ACTIVE YET")
		a.updateActualVoiceLibraryStatus()
		return
	}
	a.actualRespond("ACTUAL online. RAWDOGS voice return is ready.")
}

func (a *App) openVoiceLibrary() {
	// Keep the existing Field Manual voice page: G10 adds the real installer below
	// Settings without removing this working help/navigation feature.
	a.showScreen("manual")
	a.setManualPage("voices")
}

func (a *App) openPrivacy() {
	hiddenCommand("cmd.exe", "/c", "start", "ms-settings:privacy-microphone").Start()
}

func normaliseSpeech(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "’", "'")
	s = strings.ReplaceAll(s, "-", " ")
	r := regexp.MustCompile(`[^a-z0-9.]+`)
	s = r.ReplaceAllString(s, " ")
	return strings.Join(strings.Fields(s), " ")
}

func normaliseAxisCues(s string) string {
	n := normaliseSpeech(s)
	// Deliberate, high-signal cue words. X-RAY is normalised by normaliseSpeech to "x ray".
	repls := []struct{ from, to string }{
		{"x ray", "x"}, {"xray", "x"},
		{"yankee", "y"}, {"yankie", "y"}, {"yanky", "y"},
	}
	for _, r := range repls {
		n = regexp.MustCompile(`\b`+regexp.QuoteMeta(r.from)+`\b`).ReplaceAllString(n, r.to)
	}
	return strings.Join(strings.Fields(n), " ")
}

func coordinateAxisPrefix(text string) (string, string, bool) {
	n := normaliseAxisCues(text)
	parts := strings.Fields(n)
	if len(parts) < 2 {
		return "", "", false
	}
	if parts[0] == "set" {
		parts = parts[1:]
	}
	if len(parts) < 2 || (parts[0] != "x" && parts[0] != "y") {
		return "", "", false
	}
	axis := strings.ToUpper(parts[0])
	parts = parts[1:]
	if len(parts) > 0 && (parts[0] == "is" || parts[0] == "equals" || parts[0] == "equal") {
		parts = parts[1:]
	}
	return axis, strings.Join(parts, " "), len(parts) > 0
}

// interpretSpeechForContext keeps Vosk's raw transcript untouched, then applies
// narrow RAWDOGS-only phonetic corrections when intent/context makes them sensible.
// It deliberately does NOT turn every ordinary "why" into Y while idle.
func (a *App) interpretSpeechForContext(raw string) (string, string) {
	n := normaliseAxisCues(raw)
	if n == "" {
		return n, ""
	}
	original := n

	// G18F: Explicit, fully numbered recall commands have priority in EVERY
	// six-step state. Exact words/number only, never fuzzy coordinate guessing.
	if recall, ok := normaliseTargetRecallAlias(n); ok {
		n = recall
	}

	// G15: exact REFRESH ACTUAL / REFRESH TARGET stages, including ACTUAL / ACTUALLY
	// on their own. Never replace a longer utterance containing coordinate digits.
	if activation, ok := sixStepActivation(n); ok {
		n = activation
	} else if activation, changed := normaliseActivationCommand(n, a.pending.Kind, a.actualSet); changed {
		n = activation
	}

	// State-driven cue steering: after a transaction opens, RAWDOGS narrows its
	// interpretation to the coordinate it logically expects next. Near-matches
	// are therefore local to the state, never global substitutions in ordinary speech.
	expected := a.expectedCoordinateAxis()
	oneShotIntent := strings.HasPrefix(n, "user actual ") || strings.HasPrefix(n, "set user actual ") || strings.HasPrefix(n, "your actual ") || strings.HasPrefix(n, "new target ") || strings.HasPrefix(n, "target ")
	if expected != "" {
		// G15: interpret ACTUAL/TARGET + X-RAY/YANKEE, and the YANK/YANKY
		// misrecognitions, only inside the coordinate step that expects them.
		n = expectedSixStepCue(n, a.pending.Kind, expected)
		toks := strings.Fields(n)
		if len(toks) > 0 {
			if expected == "X" {
				// Vosk can split or reshape X-RAY. Only while X is expected do these
				// forms become the X coordinate cue.
				if len(toks) >= 2 && ((toks[0] == "ex" || toks[0] == "axe") && (toks[1] == "ray" || toks[1] == "rey" || toks[1] == "re")) {
					toks = append([]string{"x"}, toks[2:]...)
				} else if toks[0] == "exray" || toks[0] == "xrey" || toks[0] == "exrey" {
					toks[0] = "x"
				} else if toks[0] == "axe" || toks[0] == "ex" {
					toks[0] = "x"
				}
			} else if expected == "Y" {
				// YANKEE/YANKIE/YANKY are already normalised above. A bare "why"
				// is only promoted to Y here, after X has been accepted.
				if toks[0] == "why" {
					toks[0] = "y"
				}
			}
		}
		n = strings.Join(toks, " ")
	} else if oneShotIntent {
		// One-shot phrases carry both axes in the same utterance, so retain the
		// narrow phonetic help used by the previous test build.
		toks := strings.Fields(n)
		for i, t := range toks {
			switch t {
			case "axe", "ex":
				toks[i] = "x"
			case "why":
				toks[i] = "y"
			}
		}
		n = strings.Join(toks, " ")
	}
	if n != original {
		return n, "CONTEXT PHONETIC NORMALISATION"
	}
	return n, ""
}

func classifyCue(text string) (bool, string) {
	n := normaliseAxisCues(text)
	if n == "" {
		return false, ""
	}
	exacts := map[string]string{
		"refresh actual": "REFRESH ACTUAL — STEP 1", "refresh target": "REFRESH TARGET — STEP 4",
		"user actual": "USER ACTUAL", "new target": "NEW TARGET", "correction": "CORRECTION",
		"over": "OVER / CONFIRM", "confirm": "OVER / CONFIRM", "cancel": "CANCEL", "negative": "CANCEL",
		"restart": "RESTART TARGET", "start again": "RESTART TARGET",
		"switching actual": "SWITCHING ACTUAL", "clear all targets": "CLEAR ALL TARGETS", "undo clear": "UNDO CLEAR", "clear unsaved target": "CLEAR UNSAVED TARGET", "clear actual": "CLEAR ACTUAL", "clear actual x": "CLEAR ACTUAL X", "clear actual y": "CLEAR ACTUAL Y", "clear target x": "CLEAR TARGET X", "clear target y": "CLEAR TARGET Y",
		"repeat bearing": "REPEAT BEARING", "repeat range": "REPEAT RANGE", "repeat mil": "REPEAT MIL", "repeat mills": "REPEAT MIL", "repeat solution": "REPEAT SOLUTION",
		"save to disk": "SAVE TO DISK", "save settings": "SAVE TO DISK", "save setup": "SAVE TO DISK",
	}
	if c, ok := exacts[n]; ok {
		return true, c
	}
	targetNum := `(?:1|one|2|two|too|3|three|4|four)`
	if regexp.MustCompile(`^(recall|use|clear) target `+targetNum+`$`).MatchString(n) ||
		regexp.MustCompile(`^target `+targetNum+`$`).MatchString(n) {
		return true, strings.ToUpper(n)
	}
	if axis, _, ok := parseCoordinateCue(n); ok {
		return true, axis + " COORDINATE"
	}
	if axis, _, ok := coordinateAxisPrefix(n); ok {
		return true, axis + " COORDINATE — VALUE UNCLEAR"
	}
	if kind, _, _, ok := parseOneShot(n); ok {
		if kind == "actual" {
			return true, "ONE-SHOT USER ACTUAL"
		}
		return true, "ONE-SHOT TARGET"
	}
	return false, ""
}

func parseTargetNumberWord(s string) (int, bool) {
	s = normaliseSpeech(s)
	switch s {
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

func parseNumberWords(tokens []string) (int, bool) {
	if len(tokens) == 0 {
		return 0, false
	}
	digits := map[string]int{"zero": 0, "oh": 0, "one": 1, "two": 2, "too": 2, "three": 3, "free": 3, "tree": 3, "four": 4, "for": 4, "five": 5, "six": 6, "seven": 7, "eight": 8, "ate": 8, "nine": 9}
	allDigits := true
	buf := ""
	for _, t := range tokens {
		if t == "and" {
			continue
		}
		if d, ok := digits[t]; ok {
			buf += strconv.Itoa(d)
			continue
		}
		if len(t) == 1 && t[0] >= '0' && t[0] <= '9' {
			buf += t
			continue
		}
		allDigits = false
		break
	}
	if allDigits && buf != "" {
		n, err := strconv.Atoi(buf)
		return n, err == nil
	}
	units := map[string]int{"zero": 0, "one": 1, "two": 2, "three": 3, "four": 4, "five": 5, "six": 6, "seven": 7, "eight": 8, "nine": 9, "ten": 10, "eleven": 11, "twelve": 12, "thirteen": 13, "fourteen": 14, "fifteen": 15, "sixteen": 16, "seventeen": 17, "eighteen": 18, "nineteen": 19}
	tens := map[string]int{"twenty": 20, "thirty": 30, "forty": 40, "fifty": 50, "sixty": 60, "seventy": 70, "eighty": 80, "ninety": 90}
	total, current := 0, 0
	seen := false
	for _, t := range tokens {
		if t == "and" {
			continue
		}
		if n, err := strconv.Atoi(t); err == nil {
			current += n
			seen = true
			continue
		}
		if v, ok := units[t]; ok {
			current += v
			seen = true
			continue
		}
		if v, ok := tens[t]; ok {
			current += v
			seen = true
			continue
		}
		if t == "hundred" {
			if current == 0 {
				current = 1
			}
			current *= 100
			seen = true
			continue
		}
		if t == "thousand" {
			if current == 0 {
				current = 1
			}
			total += current * 1000
			current = 0
			seen = true
			continue
		}
		return 0, false
	}
	if !seen {
		return 0, false
	}
	return total + current, true
}

func parseCoordinateValue(raw string) (string, bool) {
	raw = strings.TrimSpace(strings.ReplaceAll(raw, ",", "."))
	if raw == "" {
		return "", false
	}
	if f, err := strconv.ParseFloat(raw, 64); err == nil {
		return strconv.FormatFloat(f, 'f', -1, 64), true
	}
	n := normaliseSpeech(raw)
	if f, err := strconv.ParseFloat(n, 64); err == nil {
		return strconv.FormatFloat(f, 'f', -1, 64), true
	}
	parts := strings.Fields(n)
	point := -1
	for i, t := range parts {
		if t == "point" || t == "dot" {
			point = i
			break
		}
	}
	intTokens := parts
	var fracTokens []string
	if point >= 0 {
		intTokens = parts[:point]
		fracTokens = parts[point+1:]
	}
	iv, ok := parseNumberWords(intTokens)
	if !ok {
		return "", false
	}
	if point < 0 {
		return strconv.Itoa(iv), true
	}
	if len(fracTokens) == 0 {
		return "", false
	}
	digitMap := map[string]string{"zero": "0", "oh": "0", "one": "1", "two": "2", "too": "2", "three": "3", "free": "3", "tree": "3", "four": "4", "for": "4", "five": "5", "six": "6", "seven": "7", "eight": "8", "ate": "8", "nine": "9"}
	frac := ""
	allSingle := true
	for _, t := range fracTokens {
		if d, ok := digitMap[t]; ok {
			frac += d
			continue
		}
		if len(t) == 1 && t[0] >= '0' && t[0] <= '9' {
			frac += t
			continue
		}
		allSingle = false
		break
	}
	if !allSingle {
		// G9 safety rule: after POINT, accept only individual digits.
		// Example: Vosk hearing "seven forty" must be rejected rather than guessed as 7.40/7.42.
		return "", false
	}
	return fmt.Sprintf("%d.%s", iv, frac), true
}

func parseCoordinateCue(text string) (string, string, bool) {
	axis, rawValue, ok := coordinateAxisPrefix(text)
	if !ok {
		return "", "", false
	}
	v, ok := parseCoordinateValue(rawValue)
	return axis, v, ok
}

func parseOneShot(text string) (string, string, string, bool) {
	n := normaliseAxisCues(text)
	kind := ""
	prefixes := []struct{ p, k string }{{"set user actual ", "actual"}, {"user actual ", "actual"}, {"your actual ", "actual"}, {"new target ", "target"}, {"target ", "target"}}
	for _, x := range prefixes {
		if strings.HasPrefix(n, x.p) {
			kind = x.k
			n = strings.TrimSpace(strings.TrimPrefix(n, x.p))
			break
		}
	}
	if kind == "" {
		return "", "", "", false
	}
	if strings.HasSuffix(n, " over") {
		n = strings.TrimSpace(strings.TrimSuffix(n, " over"))
	} else if strings.HasSuffix(n, " confirm") {
		n = strings.TrimSpace(strings.TrimSuffix(n, " confirm"))
	} else {
		return "", "", "", false
	}
	parts := strings.Fields(n)
	xi, yi := -1, -1
	for i, t := range parts {
		if t == "x" && xi < 0 {
			xi = i
		}
		if t == "y" && yi < 0 {
			yi = i
		}
	}
	if xi < 0 || yi < 0 || yi <= xi+1 {
		return "", "", "", false
	}
	xv, okx := parseCoordinateValue(strings.Join(parts[xi+1:yi], " "))
	yv, oky := parseCoordinateValue(strings.Join(parts[yi+1:], " "))
	return kind, xv, yv, okx && oky
}

func (a *App) addOperationLog(line string) {
	stamp := time.Now().Format("15:04:05")
	a.operationLog = append(a.operationLog, stamp+"  "+line)
	if len(a.operationLog) > 40 {
		a.operationLog = a.operationLog[len(a.operationLog)-40:]
	}
	if h := a.controls[5031]; h != 0 {
		procSendMessageW.Call(uintptr(h), LB_RESETCONTENT, 0, 0)
		for _, v := range a.operationLog {
			procSendMessageW.Call(uintptr(h), LB_ADDSTRING, 0, uintptr(unsafe.Pointer(utf16Ptr(v))))
		}
		if len(a.operationLog) > 0 {
			top := maxInt(0, len(a.operationLog)-4)
			procSendMessageW.Call(uintptr(h), LB_SETTOPINDEX, uintptr(top), 0)
		}
	}
	a.updateManualLive()
}

func (a *App) pendingActive() bool { return a.pending.Kind != "" }

func (a *App) setPending(kind string, slot int) {
	a.cueCoach.reset()
	now := time.Now()
	a.pending = PendingState{Kind: kind, Slot: slot, StartedAt: now, UpdatedAt: now}
	a.keyboardStepOverride = 0
	a.refreshOperationsUI()
}

func (a *App) touchPending() {
	if a.pendingActive() {
		a.pending.UpdatedAt = time.Now()
	}
}

func (a *App) startUserActual() {
	if a.pendingActive() {
		a.addOperationLog("Finish or cancel the current transaction first.")
		return
	}
	if a.actualSet {
		a.addOperationLog("Actual already set. Say Switching Actual.")
		return
	}
	a.setPending("actual", -1)
	a.addOperationLog("G18 STEP 1/6 — ACTUAL STARTED — say X-RAY + coordinates.")
	a.actualRespond(sixStepAcknowledgement("activate", "actual", "", ""))
}

// G15 stage 1: start/restart the ACTUAL capture, protecting a confirmed
// previous position until both new coordinates have been spoken and accepted.
func (a *App) refreshActualSixStep() {
	if a.pending.Kind == "actual" || a.pending.Kind == "switch_actual" {
		kind := a.pending.Kind
		a.setPending(kind, -1)
		a.addOperationLog("G18 STEP 1/6 — REFRESH ACTUAL — X-RAY NEXT.")
		a.actualRespond(sixStepAcknowledgement("activate", "actual", "", ""))
		return
	}
	if a.pendingActive() {
		// REFRESH ACTUAL is an explicit restart cue. Abandon only the
		// unconfirmed transaction; confirmed targets and ACTUAL stay intact.
		before := a.voiceDiagnosticState()
		a.writeInteractionDiagnostic("WORKFLOW", "REFRESH ACTUAL replaced unfinished transaction", "RESTART", before, "actual X next")
		a.addOperationLog("REFRESH ACTUAL — unfinished entry discarded; confirmed data preserved.")
	}
	if a.actualSet {
		a.setPending("switch_actual", -1)
		a.addOperationLog("G18 STEP 1/6 — REFRESH ACTUAL — previous ACTUAL protected until new Y accepted.")
		a.actualRespond(sixStepAcknowledgement("activate", "actual", "", ""))
	} else {
		a.startUserActual()
	}
}

func (a *App) nextTargetSlot() int {
	for i := 0; i < 4; i++ {
		if !a.targets[i].Confirmed {
			return i
		}
	}
	return -1
}

func (a *App) startNewTarget() {
	// G9 global target-chain restart: saying NEW TARGET again while a target is
	// provisional abandons only that unfinished target and starts that target slot fresh.
	// Confirmed USER ACTUAL and saved targets are never touched.
	if a.pending.Kind == "target" {
		slot := a.pending.Slot
		a.setPending("target", slot)
		if slot == -2 {
			a.addOperationLog("NEW TARGET RESTART — UNSAVED TARGET reset. Awaiting X-RAY and YANKEE.")
			a.actualRespond("New target restarted. X-ray.")
		} else {
			a.addOperationLog(fmt.Sprintf("NEW TARGET RESTART — TARGET %d reset. Awaiting X-RAY and YANKEE.", slot+1))
			a.actualRespond("New target restarted. X-ray.")
		}
		return
	}
	if a.pendingActive() {
		a.addOperationLog("Finish or cancel the current non-target transaction first.")
		return
	}
	if !a.actualSet {
		a.addOperationLog("Set USER ACTUAL first.")
		return
	}
	slot := a.nextTargetSlot()
	if slot >= 0 {
		a.setPending("target", slot)
		a.addOperationLog(fmt.Sprintf("G18 STEP 4/6 — REFRESH TARGET — TARGET %d armed; X-RAY next.", slot+1))
		a.actualRespond(sixStepAcknowledgement("activate", "target", "", ""))
		return
	}
	if !a.unsavedExists {
		a.setPending("target", -2)
		a.addOperationLog("UNSAVED TARGET armed — clear a saved target to keep it.")
		a.actualRespond(sixStepAcknowledgement("activate", "target", "", ""))
		return
	}
	a.addOperationLog("TARGET MEMORY FULL — clear a saved target before creating another target.")
}

func (a *App) restartCurrentTarget() {
	if a.pending.Kind != "target" {
		if a.pendingActive() {
			a.addOperationLog("RESTART is reserved for the active NEW TARGET chain. Current transaction unchanged.")
		} else {
			a.addOperationLog("No NEW TARGET chain is active to restart.")
		}
		return
	}
	slot := a.pending.Slot
	a.setPending("target", slot)
	if slot == -2 {
		a.addOperationLog("TARGET RESTART — UNSAVED TARGET reset. Awaiting X-RAY and YANKEE.")
		a.actualRespond("Target restarted. X-ray.")
	} else {
		a.addOperationLog(fmt.Sprintf("TARGET RESTART — TARGET %d reset. Awaiting X-RAY and YANKEE.", slot+1))
		a.actualRespond("Target restarted. X-ray.")
	}
}

func (a *App) startSwitchActual() {
	if a.pendingActive() {
		a.addOperationLog("Finish or cancel the current transaction first.")
		return
	}
	if !a.actualSet {
		a.addOperationLog("No Actual is set. Say User Actual.")
		return
	}
	a.setPending("switch_actual", -1)
	a.addOperationLog("SWITCHING ACTUAL — old Actual protected until Confirm/Over.")
}

func (a *App) expectedCoordinateAxis() string {
	if !a.pendingActive() || (a.pending.Kind != "actual" && a.pending.Kind != "target" && a.pending.Kind != "switch_actual") {
		return ""
	}
	if a.pending.Replacing != "" {
		return a.pending.Replacing
	}
	if a.pending.X == "" {
		return "X"
	}
	if a.pending.Y == "" {
		return "Y"
	}
	return ""
}

func (a *App) acceptCoordinate(axis, value string) {
	if !a.pendingActive() || (a.pending.Kind != "actual" && a.pending.Kind != "target" && a.pending.Kind != "switch_actual") {
		a.addOperationLog("Specify User Actual or New Target.")
		return
	}
	expected := a.expectedCoordinateAxis()
	if expected == "" {
		a.addOperationLog("COORDINATES READY — SAY OVER / CONFIRM, OR CORRECTION. Existing values preserved.")
		return
	}
	if expected != "" && axis != expected {
		if a.pending.Replacing != "" {
			a.addOperationLog("CORRECTION — awaiting replacement " + a.pending.Replacing + " coordinate.")
		} else if expected == "X" {
			a.addOperationLog("EXPECTED X-RAY FIRST — repeat X-RAY followed by the X coordinate.")
		} else {
			a.addOperationLog("EXPECTED YANKEE — X accepted; repeat YANKEE followed by the Y coordinate.")
		}
		return
	}
	if axis == "X" {
		a.pending.X = value
	} else {
		a.pending.Y = value
	}
	a.pending.LastCoord = axis
	a.recordAcceptedCoordinate(a.pending.Kind, a.pending.Slot, axis)
	a.pending.Replacing = ""
	a.pending.CueAwaiting = ""
	a.touchPending()
	if next := a.expectedCoordinateAxis(); next == "Y" {
		a.addOperationLog(fmt.Sprintf("ACCEPTED — %s %s — NOW AWAITING YANKEE", axis, value))
		a.actualRespond(sixStepAcknowledgement("coordinate", a.pending.Kind, "X", coordinateSpeech(value)))
	} else if next == "X" {
		a.addOperationLog(fmt.Sprintf("ACCEPTED — %s %s — NOW AWAITING X-RAY", axis, value))
		a.actualRespond("Y " + coordinateSpeech(value) + ". X-ray.")
	} else {
		a.addOperationLog(fmt.Sprintf("G18 ACCEPTED — %s %s — COORDINATES COMPLETE", axis, value))
		if !a.autoVoiceConfirm {
			a.actualRespond("Y " + coordinateSpeech(value) + ". Coordinates complete.")
		}
	}
	a.refreshOperationsUI()
}

// G15: after the Y coordinate, the six-step voice transaction commits
// automatically. No seventh OVER/CONFIRM is required. Guard against
// auto-committing half-entered, cancelled or non-coordinate transactions.
func (a *App) acceptVoiceCoordinate(axis, value string) {
	kind := a.pending.Kind
	coordinateKind := kind == "actual" || kind == "switch_actual" || kind == "target"
	// Y can commit only after valid X, only while the state machine expects Y.
	// Do not announce implementation for a rejected or incomplete coordinate.
	autoCompleting := coordinateKind && axis == "Y" && a.pending.X != "" && a.expectedCoordinateAxis() == "Y"
	a.autoVoiceConfirm = autoCompleting
	a.acceptCoordinate(axis, value)
	if coordinateKind && a.pending.Kind == kind && a.pending.X != "" && a.pending.Y != "" {
		a.addOperationLog("G18 AUTO-CONFIRM — stage completed (OVER not required).")
		a.confirmPending()
		if autoCompleting && !a.pendingActive() {
			a.actualRespond(sixStepAcknowledgement("committed", kind, "Y", coordinateSpeech(value)))
			if kind == "target" {
				a.announceCurrentSolution()
			}
		}
	}
	a.autoVoiceConfirm = false
}

func (a *App) correctPending() {
	// G18: one command at every stage, including AFTER an auto-committed Y.
	a.universalCorrection()
}

func (a *App) confirmPending() {
	if !a.pendingActive() {
		a.addOperationLog("Nothing awaiting confirmation.")
		return
	}
	switch a.pending.Kind {
	case "clear_all":
		a.targets = [4]TargetSlot{}
		a.unsaved = TargetSlot{}
		a.unsavedExists = false
		a.activeTarget = -1
		a.pending = PendingState{Slot: -1}
		a.addOperationLog("CONFIRMED — ALL TARGETS CLEARED. USER ACTUAL PRESERVED.")
	case "clear_actual":
		a.actualX, a.actualY, a.actualSet = "", "", false
		a.pending = PendingState{Slot: -1}
		a.addOperationLog("CONFIRMED — USER ACTUAL CLEARED. TARGET MEMORIES PRESERVED.")
	case "actual", "switch_actual":
		if a.pending.X == "" || a.pending.Y == "" {
			missing := "X and Y"
			if a.pending.X != "" {
				missing = "Y"
			}
			if a.pending.Y != "" {
				missing = "X"
			}
			a.addOperationLog("INCOMPLETE ACTUAL — awaiting " + missing + " coordinate.")
			return
		}
		a.actualX, a.actualY, a.actualSet = a.pending.X, a.pending.Y, true
		wasSwitch := a.pending.Kind == "switch_actual"
		a.pending = PendingState{Slot: -1}
		if wasSwitch {
			a.addOperationLog("CONFIRMED — USER ACTUAL SWITCHED. Saved targets preserved; L81 solution recalculated.")
		} else {
			a.addOperationLog("CONFIRMED — USER ACTUAL SET.")
			if !a.autoVoiceConfirm {
				a.actualRespond("User actual confirmed.")
			}
		}
	case "target":
		if a.pending.X == "" || a.pending.Y == "" {
			missing := "X and Y"
			if a.pending.X != "" {
				missing = "Y"
			}
			if a.pending.Y != "" {
				missing = "X"
			}
			a.addOperationLog("INCOMPLETE TARGET — awaiting " + missing + " coordinate.")
			return
		}
		t := TargetSlot{X: a.pending.X, Y: a.pending.Y, Confirmed: true}
		slot := a.pending.Slot
		if slot == -2 {
			a.unsaved, a.unsavedExists, a.activeTarget = t, true, -2
			a.addOperationLog("CONFIRMED — UNSAVED TARGET active. Clear a saved target to keep it.")
			if !a.autoVoiceConfirm {
				a.actualRespond("Unsaved target confirmed.")
			}
		} else if slot >= 0 && slot < 4 {
			a.targets[slot], a.activeTarget = t, slot
			a.addOperationLog(fmt.Sprintf("CONFIRMED — TARGET %d stored and active.", slot+1))
			if !a.autoVoiceConfirm {
				a.actualRespond(fmt.Sprintf("Target %d confirmed.", slot+1))
			}
		}
		a.pending = PendingState{Slot: -1}
		if !a.autoVoiceConfirm {
			a.announceCurrentSolution()
		}
	}
	a.refreshOperationsUI()
}

func (a *App) cancelPending() {
	if !a.pendingActive() {
		a.addOperationLog("Nothing awaiting cancellation.")
		return
	}
	kind := a.pending.Kind
	a.pending = PendingState{Slot: -1}
	a.addOperationLog("CANCELLED — " + strings.ToUpper(strings.ReplaceAll(kind, "_", " ")) + " unchanged.")
	a.actualRespond("Cancelled.")
	a.refreshOperationsUI()
}

func (a *App) saveClearSnapshot() {
	a.lastClear = &clearSnapshot{actualX: a.actualX, actualY: a.actualY, actualSet: a.actualSet,
		targets: a.targets, unsaved: a.unsaved, unsavedExists: a.unsavedExists,
		activeTarget: a.activeTarget, pending: a.pending}
}

func (a *App) undoClear() {
	if a.lastClear == nil {
		a.addOperationLog("NOTHING TO UNDO — no recent clear.")
		return
	}
	snap := a.lastClear
	a.actualX, a.actualY, a.actualSet = snap.actualX, snap.actualY, snap.actualSet
	a.targets, a.unsaved, a.unsavedExists = snap.targets, snap.unsaved, snap.unsavedExists
	a.activeTarget, a.pending = snap.activeTarget, snap.pending
	a.lastClear = nil // one-step undo, not a repeating toggle
	a.keyboardDirtyID, a.keyboardErrorID, a.keyboardStepOverride = 0, 0, 0
	a.addOperationLog("UNDO CLEAR — previous targets and coordinates restored.")
	a.refreshOperationsUI()
	a.actualRespond("Restored.")
}

func (a *App) clearAllTargets() {
	a.saveClearSnapshot()
	a.abandonPiperReadbacks()
	a.targets = [4]TargetSlot{}
	a.unsaved = TargetSlot{}
	a.unsavedExists = false
	a.activeTarget = -1
	if a.pending.Kind == "target" {
		a.pending = PendingState{Slot: -1}
	}
	a.addOperationLog("ALL TARGETS CLEARED — USER ACTUAL PRESERVED. Say UNDO CLEAR to restore.")
	a.refreshOperationsUI()
	a.actualRespond("Targets cleared.")
}

func (a *App) clearOneTarget(slot int) {
	if slot < 0 || slot >= 4 {
		return
	}
	if !a.targets[slot].Confirmed && !(a.pending.Kind == "target" && a.pending.Slot == slot) {
		a.addOperationLog(fmt.Sprintf("Target %d is empty.", slot+1))
		return
	}
	a.saveClearSnapshot()
	a.abandonPiperReadbacks()
	a.targets[slot] = TargetSlot{} // never auto-promote UNSAVED into deleted slot
	if a.activeTarget == slot {
		a.activeTarget = -1
	}
	if a.pending.Kind == "target" && a.pending.Slot == slot {
		a.pending = PendingState{Slot: -1}
	}
	a.addOperationLog(fmt.Sprintf("TARGET %d cleared. UNDO CLEAR restores it.", slot+1))
	a.refreshOperationsUI()
	a.actualRespond(fmt.Sprintf("Target %d cleared.", slot+1))
}

func (a *App) clearUnsavedTarget() {
	if !a.unsavedExists && !(a.pending.Kind == "target" && a.pending.Slot == -2) {
		a.addOperationLog("Unsaved target is empty.")
		return
	}
	a.saveClearSnapshot()
	a.unsaved = TargetSlot{}
	a.unsavedExists = false
	if a.activeTarget == -2 {
		a.activeTarget = -1
	}
	if a.pending.Kind == "target" && a.pending.Slot == -2 {
		a.pending = PendingState{Slot: -1}
	}
	a.addOperationLog("UNSAVED TARGET cleared. UNDO CLEAR restores it.")
	a.refreshOperationsUI()
}

// Clear one X/Y value without deleting its partner. The incomplete position
// goes through the established six-step flow before it can be used for firing.
func (a *App) clearCoordinateValue(kind, axis string) {
	if axis != "X" && axis != "Y" {
		return
	}
	if kind == "actual" {
		if a.pending.Kind != "actual" && a.pending.Kind != "switch_actual" && !a.actualSet {
			a.addOperationLog("USER ACTUAL not set.")
			return
		}
		a.saveClearSnapshot()
		if a.pending.Kind != "actual" && a.pending.Kind != "switch_actual" {
			a.pending = PendingState{Kind: "actual", Slot: -1, X: a.actualX, Y: a.actualY, StartedAt: time.Now()}
		}
		// To avoid stale firing solutions the previous confirmed position is invalid.
		a.actualSet = false
		if axis == "X" {
			a.pending.X = ""
		} else {
			a.pending.Y = ""
		}
	} else if kind == "target" {
		if a.pending.Kind != "target" {
			var t TargetSlot
			slot := a.activeTarget
			if slot >= 0 && slot < 4 && a.targets[slot].Confirmed {
				t = a.targets[slot]
			} else if slot == -2 && a.unsavedExists {
				t = a.unsaved
			} else {
				a.addOperationLog("No active target coordinate to clear.")
				return
			}
			a.saveClearSnapshot()
			a.pending = PendingState{Kind: "target", Slot: slot, X: t.X, Y: t.Y, StartedAt: time.Now()}
			if slot == -2 {
				a.unsaved = TargetSlot{}
				a.unsavedExists = false
			} else {
				a.targets[slot] = TargetSlot{}
			}
			a.activeTarget = -1 // old solution is now invalid and must not remain active
		} else {
			a.saveClearSnapshot()
		}
		if axis == "X" {
			a.pending.X = ""
		} else {
			a.pending.Y = ""
		}
	} else {
		return
	}
	a.pending.CueAwaiting = ""
	a.pending.Replacing = ""
	a.keyboardDirtyID, a.keyboardErrorID, a.keyboardStepOverride = 0, 0, 0
	a.addOperationLog(kind + " " + axis + " cleared — say " + map[string]string{"X": "X-RAY", "Y": "YANKEE"}[axis] + " or type a replacement. UNDO CLEAR available.")
	a.refreshOperationsUI()
}

func (a *App) recallTarget(slot int) {
	if slot < 0 || slot >= 4 || !a.targets[slot].Confirmed {
		a.addOperationLog(fmt.Sprintf("TARGET %d NOT SAVED — current entry preserved.", slot+1))
		return
	}
	// Explicit recall overrides incomplete coordinate entry, but the previously
	// confirmed USER ACTUAL and all four SAVED targets remain untouched.
	if a.pendingActive() {
		a.addOperationLog("RECALL: abandoned unconfirmed entry; confirmed actual preserved.")
		a.pending = PendingState{Slot: -1}
		a.keyboardStepOverride, a.keyboardDirtyID, a.keyboardErrorID = 0, 0, 0
		a.lastCoordKind, a.lastCoordAxis = "", ""
	}
	a.activeTarget = slot
	a.addOperationLog(fmt.Sprintf("TARGET %d recalled and active.", slot+1))
	a.refreshOperationsUI()
	a.announceCurrentSolution()
}

// Selecting a memory row displays its existing solution without changing any saved coordinates.
// Voice entry is left untouched if it is in progress.
func (a *App) selectMemoryTargetByMouse(slot int) {
	if a.pendingActive() {
		a.addOperationLog("Finish or cancel the current coordinate entry before recalling a target.")
		return
	}
	if slot == 4 {
		if !a.unsavedExists {
			a.addOperationLog("UNSAVED TARGET EMPTY — no solution to display.")
			return
		}
		a.activeTarget = -2
		a.addOperationLog("UNSAVED TARGET selected by mouse.")
		a.refreshOperationsUI()
		a.announceCurrentSolution()
		return
	}
	if slot >= 0 && slot < 4 {
		a.recallTarget(slot)
	}
}

func (a *App) beginClearAll() { a.clearAllTargets() }

func (a *App) beginClearActual() {
	if a.pendingActive() {
		a.addOperationLog("Finish or cancel the current transaction first.")
		return
	}
	if !a.actualSet {
		a.addOperationLog("USER ACTUAL is already empty.")
		return
	}
	a.setPending("clear_actual", -1)
	a.addOperationLog("CONFIRM CLEAR ACTUAL — targets will be preserved.")
}

func (a *App) saveSetupOnCommand() {
	if err := a.saveSettings(); err != nil {
		setText(a.controls[3004], "SYSTEM STATUS: SAVE FAILED — "+err.Error())
		a.addOperationLog("SAVE TO DISK FAILED — " + err.Error())
		return
	}
	a.guideSaved = true
	setText(a.controls[3004], "SYSTEM STATUS: SETUP SAVED TO DISK — PREVIOUS SAVED CONFIGURATION REPLACED")
	a.addOperationLog("SAVED: SETUP TO DISK")
	a.updateVoskLibraryStatus()
	a.updateActualVoiceLibraryStatus()
	a.actualRespond("Saved.")
	if a.guideActive && a.guideStep == 5 {
		a.advanceGuideAutomatic()
	}
}

func (a *App) processOperationalSpeech(text string) {
	// G18H clear commands have priority at every stage; incomplete phrases do nothing.
	nClear := normaliseSpeech(text)
	if nClear == "end guide" && a.manualNarrating {
		a.endNarratedManual()
		if a.guideActive && a.guideStep == 6 {
			a.finishGuidedSetup(false)
		}
		return
	}
	switch nClear {
	case "clear all targets":
		a.clearAllTargets()
		return
	case "undo clear":
		a.undoClear()
		return
	case "clear unsaved target":
		a.clearUnsavedTarget()
		return
	case "clear actual x", "clear actual x ray":
		a.clearCoordinateValue("actual", "X")
		return
	case "clear actual y", "clear actual yankee":
		a.clearCoordinateValue("actual", "Y")
		return
	case "clear target x", "clear target x ray":
		a.clearCoordinateValue("target", "X")
		return
	case "clear target y", "clear target yankee":
		a.clearCoordinateValue("target", "Y")
		return
	}
	if m := regexp.MustCompile(`^clear target (1|one|2|two|too|3|three|4|four)$`).FindStringSubmatch(nClear); len(m) == 2 {
		if slot, ok := parseTargetNumberWord(m[1]); ok {
			a.clearOneTarget(slot)
		}
		return
	}
	// G18F: numbered saved-target recall outranks correction, pending X/Y,
	// and even an unfinished six-step transaction. Empty slots never cancel.
	if slot, ok := parseRecallTargetPhrase(text); ok {
		a.recallTarget(slot)
		return
	}
	// Direct verbal access to the EXISTING Save Setup To Disk action.
	switch normaliseSpeech(text) {
	case "save to disk", "save settings", "save setup":
		a.saveSetupOnCommand()
		return
	}
	// Handle Piper independently of ACTUAL / TARGET procedure state.
	if cmd, ok := parsePiperCommand(text); ok {
		a.applyPiperCommand(cmd)
		return
	}
	if normaliseSpeech(text) == "correction" {
		a.universalCorrection()
		return
	}
	if value, ok := a.spokenCoordinateConfirm(text); ok {
		axis := a.expectedCoordinateAxis()
		a.acceptVoiceCoordinate(axis, value)
		return
	}
	if kind, x, y, ok := parseOneShot(text); ok {
		if kind == "actual" {
			if a.actualSet {
				a.addOperationLog("Actual already set. Say Switching Actual.")
				return
			}
			a.refreshActualSixStep()
			if a.pending.Kind == "actual" {
				a.acceptVoiceCoordinate("X", x)
				a.acceptVoiceCoordinate("Y", y)
			}
		} else {
			a.startNewTarget()
			if a.pending.Kind == "target" {
				a.acceptVoiceCoordinate("X", x)
				a.acceptVoiceCoordinate("Y", y)
			}
		}
		return
	}
	n := normaliseAxisCues(text)
	// G14: "X-RAY" [pause] "nine eight point five five" is valid.
	// The digits-only phrase cannot be consumed until the CURRENT expected axis
	// has actually been armed by its cue (or a deliberate CORRECTION).
	if expected := a.expectedCoordinateAxis(); expected != "" {
		if axis := loneAxisCue(n); axis != "" {
			if axis == expected {
				a.pending.CueAwaiting = axis
				a.touchPending()
				a.addOperationLog("HEARD " + axis + " CUE — waiting for digits; take your time.")
				a.actualRespond(sixStepAcknowledgement("axis", a.pending.Kind, axis, ""))
			} else {
				a.addOperationLog("STILL WAITING FOR " + expected + " — previous coordinate preserved.")
			}
			return
		}
		if a.pending.CueAwaiting == expected || a.pending.Replacing == expected {
			if value, ok := parseCoordinateValue(text); ok {
				a.acceptVoiceCoordinate(expected, value)
				return
			}
		}
	}
	if axis, val, ok := parseCoordinateCue(text); ok {
		a.acceptVoiceCoordinate(axis, val)
		return
	}
	if a.pending.Replacing == "X" || a.pending.Replacing == "Y" {
		// After CORRECTION the cue has already established which block is being
		// replaced, so a clean digit/value phrase can be accepted directly.
		if val, ok := parseCoordinateValue(text); ok {
			a.acceptVoiceCoordinate(a.pending.Replacing, val)
			return
		}
	}
	if axis, _, ok := coordinateAxisPrefix(text); ok {
		cue := "X-RAY"
		if axis == "Y" {
			cue = "YANKEE"
		}
		a.addOperationLog("REJECTED: " + cue + " VALUE UNCLEAR — repeat the coordinate using individual digits after POINT.")
		return
	}
	switch n {
	case "refresh actual", "user actual":
		a.refreshActualSixStep()
		return
	case "refresh target", "new target":
		a.startNewTarget()
		return
	case "restart", "start again":
		a.restartCurrentTarget()
		return
	case "switching actual":
		a.startSwitchActual()
		return
	case "correction":
		a.correctPending()
		return
	case "over", "confirm":
		a.confirmPending()
		return
	case "cancel", "negative":
		a.cancelPending()
		return
	case "clear all targets":
		a.beginClearAll()
		return
	case "clear actual":
		a.beginClearActual()
		return
	case "repeat bearing", "repeat range", "repeat mil", "repeat mills", "repeat solution":
		a.repeatFiringSolution(n)
		return
	}
	r := regexp.MustCompile(`^(recall|use|clear) target (1|one|2|two|too|3|three|4|four)$`)
	m := r.FindStringSubmatch(n)
	if len(m) == 3 {
		slot, ok := parseTargetNumberWord(m[2])
		if !ok {
			return
		}
		if m[1] == "clear" {
			a.clearOneTarget(slot)
		} else {
			a.recallTarget(slot)
		}
	}
}

func (a *App) activeTargetDisplay() (string, string, string) {
	if a.pending.Kind == "target" {
		name := "UNSAVED TARGET — PENDING"
		if a.pending.Slot >= 0 {
			name = fmt.Sprintf("TARGET %d — PENDING", a.pending.Slot+1)
		}
		return name, emptyDash(a.pending.X), emptyDash(a.pending.Y)
	}
	if a.activeTarget == -2 && a.unsavedExists {
		return "UNSAVED TARGET — ACTIVE", a.unsaved.X, a.unsaved.Y
	}
	if a.activeTarget >= 0 && a.activeTarget < 4 && a.targets[a.activeTarget].Confirmed {
		t := a.targets[a.activeTarget]
		return fmt.Sprintf("TARGET %d — ACTIVE", a.activeTarget+1), t.X, t.Y
	}
	return "NO ACTIVE TARGET", "---", "---"
}
func emptyDash(s string) string {
	if s == "" {
		return "---"
	}
	return s
}

func (a *App) setControlClass(id int, class string) {
	h := a.controls[id]
	if h == 0 {
		return
	}
	old := a.textClass[h]
	if old == class {
		return
	}
	if class == "" {
		delete(a.textClass, h)
	} else {
		a.textClass[h] = class
	}
	procInvalidateRect.Call(uintptr(h), 0, 0)
}

func (a *App) pendingCoordinateStatus(prefix string) string {
	switch a.expectedCoordinateAxis() {
	case "X":
		return "G18 — AWAITING X-RAY (TAB / ENTER ALSO ACCEPTED)"
	case "Y":
		return "G18 — X ACCEPTED / AWAITING YANKEE (YANK OK)"
	default:
		if a.pendingActive() {
			return "PENDING — COORDINATES READY / SAY OVER OR CONFIRM"
		}
	}
	return prefix + " READY"
}

func (a *App) refreshOperationsUI() {
	if a.controls[5001] == 0 {
		return
	}
	ax, ay := "---", "---"
	astate := "NOT SET"
	if a.actualSet {
		ax, ay, astate = a.actualX, a.actualY, "CONFIRMED"
	} else if a.actualX != "" || a.actualY != "" {
		ax, ay, astate = emptyDash(a.actualX), emptyDash(a.actualY), "INCOMPLETE — REPLACE MISSING COORDINATE"
	}
	if a.pending.Kind == "actual" {
		ax, ay = emptyDash(a.pending.X), emptyDash(a.pending.Y)
		astate = a.pendingCoordinateStatus("USER ACTUAL")
	}
	if a.pending.Kind == "switch_actual" {
		astate = "CONFIRMED — NEW ACTUAL IN COMMAND BUFFER"
	}
	a.syncCoordinateEdit(5001, ax)
	a.syncCoordinateEdit(5002, ay)
	setText(a.controls[5003], astate)
	name, tx, ty := a.activeTargetDisplay()
	setText(a.controls[5010], name)
	a.syncCoordinateEdit(5011, tx)
	a.syncCoordinateEdit(5012, ty)
	tstate := "IDLE"
	if a.pending.Kind == "target" {
		tstate = a.pendingCoordinateStatus("TARGET")
	}
	setText(a.controls[5013], tstate)
	for i := 0; i < 4; i++ {
		line := fmt.Sprintf("TARGET %d — EMPTY", i+1)
		if a.targets[i].Confirmed {
			active := ""
			if a.activeTarget == i {
				active = " — ACTIVE"
			}
			line = fmt.Sprintf("TARGET %d — X %s  Y %s%s", i+1, a.targets[i].X, a.targets[i].Y, active)
		}
		setText(a.controls[5020+i], line)
	}
	uline := "UNSAVED TARGET — EMPTY"
	if a.unsavedExists {
		active := ""
		if a.activeTarget == -2 {
			active = " — ACTIVE"
		}
		uline = fmt.Sprintf("UNSAVED TARGET — X %s  Y %s%s\r\nCLEAR A SAVED TARGET TO KEEP THIS", a.unsaved.X, a.unsaved.Y, active)
	}
	setText(a.controls[5024], uline)
	ctx := a.keyboardStageHelp()
	switch a.pending.Kind {
	case "actual":
		ctx = fmt.Sprintf("%s | X %s | Y %s | %s", a.keyboardStageHelp(), emptyDash(a.pending.X), emptyDash(a.pending.Y), replacementText(a.pending))
	case "switch_actual":
		ctx = fmt.Sprintf("%s | NEW X %s | NEW Y %s | old Actual protected | %s", a.keyboardStageHelp(), emptyDash(a.pending.X), emptyDash(a.pending.Y), replacementText(a.pending))
	case "target":
		label := "UNSAVED TARGET"
		if a.pending.Slot >= 0 {
			label = fmt.Sprintf("TARGET %d", a.pending.Slot+1)
		}
		ctx = fmt.Sprintf("%s | %s PENDING | X %s | Y %s | %s", a.keyboardStageHelp(), label, emptyDash(a.pending.X), emptyDash(a.pending.Y), replacementText(a.pending))
	case "clear_all":
		ctx = "ALL TARGETS CLEARED — say UNDO CLEAR to restore"
	case "clear_actual":
		ctx = "CONFIRM CLEAR ACTUAL — say OVER or CANCEL / NEGATIVE"
	}
	setText(a.controls[5030], ctx)
	firingText, firingClass := a.firingSolutionDisplay()
	setText(a.controls[5032], firingText)
	// G18H: steady NAVY border. No flash timer and no whole-window repaint per phase.
	if firingClass == "green" {
		a.firingBorderState = "ready"
	} else if firingClass == "red" {
		a.firingBorderState = "out_of_range"
	} else {
		a.firingBorderState = ""
	}
	procKillTimer.Call(uintptr(a.hwnd), 79)

	// G18H: calculate FINAL colour per child and apply each once. Previous
	// builds briefly reset green/pink to neutral on every Vosk update; that
	// caused visible text flicker (especially on a second screen).
	classes := make(map[int]string, 14)
	if a.actualSet {
		for _, id := range []int{5001, 5002, 5003} {
			classes[id] = "green"
		}
	}
	if a.pending.Kind == "actual" || a.pending.Kind == "switch_actual" {
		if a.pending.X != "" {
			classes[5001] = "green"
		}
		if a.pending.Y != "" {
			classes[5002] = "green"
		}
		classes[5003], classes[5030] = "pink", "pink"
	}
	if a.pending.Kind == "target" {
		classes[5010], classes[5013], classes[5030] = "pink", "pink", "pink"
		if a.pending.X != "" {
			classes[5011] = "green"
		}
		if a.pending.Y != "" {
			classes[5012] = "green"
		}
	} else if a.activeTarget != -1 {
		for _, id := range []int{5010, 5011, 5012, 5013} {
			classes[id] = "green"
		}
	}
	for i := 0; i < 4; i++ {
		if a.targets[i].Confirmed {
			classes[5020+i] = "green"
		}
	}
	if a.unsavedExists {
		classes[5024] = "green"
	}
	if firingClass != "" {
		classes[5032] = firingClass
	}
	focus := a.coordinateStepHighlight()
	if a.isCoordinateID(focus) {
		classes[focus] = "pink"
	}
	if a.isCoordinateID(a.keyboardDirtyID) {
		classes[a.keyboardDirtyID] = "pink"
	}
	if a.isCoordinateID(a.keyboardErrorID) {
		classes[a.keyboardErrorID] = "red"
	}
	for _, id := range []int{5001, 5002, 5003, 5010, 5011, 5012, 5013, 5020, 5021, 5022, 5023, 5024, 5030, 5032} {
		a.setControlClass(id, classes[id])
	}

	a.updateOpsListenerStatus()
	if a.currentScreen == "operations" {
		// Actual coordinate/solution changes are infrequent and need a complete
		// panel refresh; regular microphone / TTS status ticks stay targeted.
		procInvalidateRect.Call(uintptr(a.hwnd), 0, 1)
	}
}

func replacementText(p PendingState) string {
	if p.Replacing != "" {
		return "CORRECTION — REPLACE " + p.Replacing
	}
	if p.X == "" || p.Y == "" {
		return "AWAITING COORDINATE"
	}
	return "READY FOR OVER (CONFIRM button = same action)"
}

func (a *App) syncListenerControls() {
	if a.guideActive && a.guideStep == 2 {
		a.syncGuideOutputButtons()
		return
	}
	state := a.currentListenerState()
	running := state == "STARTING" || state == "READY — SAY COMMAND"
	status := "STATUS: VOSK STOPPED"
	if state == "STARTING" {
		status = "STATUS: VOSK STARTING"
	} else if state == "READY — SAY COMMAND" {
		status = "STATUS: VOSK LISTENING"
	} else if strings.Contains(state, "ERROR") || strings.Contains(state, "STOPPED —") {
		status = "STATUS: VOSK STOPPED — " + state
	}
	setText(a.controls[3021], status)
	setText(a.controls[ID_SETTINGS_LISTEN], "START LISTENING")
	setText(a.controls[ID_SETTINGS_STOP], "STOP LISTENING")

	startEnabled := uintptr(1)
	stopEnabled := uintptr(0)
	if running {
		startEnabled, stopEnabled = 0, 1
	}
	for _, id := range []int{ID_SETTINGS_LISTEN, ID_TRAIN_LISTEN, ID_OP_LISTEN} {
		if h := a.controls[id]; h != 0 {
			procEnableWindow.Call(uintptr(h), startEnabled)
		}
	}
	for _, id := range []int{ID_SETTINGS_STOP, ID_TRAIN_STOP, ID_OP_STOP} {
		if h := a.controls[id]; h != 0 {
			procEnableWindow.Call(uintptr(h), stopEnabled)
		}
	}
}

func (a *App) updateOpsListenerStatus() {
	a.syncListenerControls()
	if a.controls[5033] == 0 {
		return
	}
	team := a.settings.Team
	if team == "" {
		team = "MANTICORE — GREEN"
	}
	state := a.currentListenerState()
	setText(a.controls[5033], "TEAM: "+strings.ToUpper(team)+" | LISTENER: "+state)
	a.updateManualLive()
	if a.controls[5000] != 0 {
		a.healthMu.Lock()
		micOK, outOK := a.healthMicOK, a.healthOutOK
		a.healthMu.Unlock()
		if !micOK {
			setText(a.controls[5000], "SYSTEM HEALTH: RED — SELECTED MICROPHONE LOST")
		} else if !outOK {
			setText(a.controls[5000], "SYSTEM HEALTH: RED — SELECTED AUDIO OUTPUT LOST")
		} else if state == "READY — SAY COMMAND" {
			setText(a.controls[5000], "SYSTEM HEALTH: GREEN — READY FOR OPERATIONAL VOICE COMMANDS")
		} else if strings.Contains(state, "ERROR") || strings.Contains(state, "STOPPED —") {
			setText(a.controls[5000], "SYSTEM HEALTH: RED — "+state)
		} else {
			setText(a.controls[5000], "SYSTEM HEALTH: AMBER — "+state)
		}
	}
}

func (a *App) checkPendingTimeout() {
	// G14: silence NEVER cancels an unfinished coordinate entry. Delays while
	// checking game pings are normal, and X must survive while waiting for Y.
	if a.pending.Kind == "actual" || a.pending.Kind == "target" || a.pending.Kind == "switch_actual" {
		return
	}
	// Preserve the G13 safety timeout for unrelated destructive confirmation
	// transactions (CLEAR ALL / CLEAR ACTUAL).
	if !a.pendingActive() || a.pending.UpdatedAt.IsZero() {
		return
	}
	if time.Since(a.pending.UpdatedAt) >= 30*time.Second {
		kind := a.pending.Kind
		a.pending = PendingState{Slot: -1}
		a.addOperationLog("TRANSACTION TIMEOUT — " + strings.ToUpper(strings.ReplaceAll(kind, "_", " ")) + " cancelled.")
		a.refreshOperationsUI()
	}
}

func (a *App) voskBasePaths() (base, runtimeDir, modelRoot string) {
	base = filepath.Join(filepath.Dir(a.configPath), "vosk")
	runtimeDir = filepath.Join(base, "runtime")
	modelRoot = filepath.Join(base, "model")
	return
}

func (a *App) voskRuntimeReady() bool {
	_, runtimeDir, _ := a.voskBasePaths()
	need := []string{"libvosk.dll", "libgcc_s_seh-1.dll", "libstdc++-6.dll", "libwinpthread-1.dll"}
	for _, n := range need {
		if _, err := os.Stat(filepath.Join(runtimeDir, n)); err != nil {
			return false
		}
	}
	return true
}

func (a *App) voskModelInstalled(spec VoskModelSpec) bool {
	_, _, root := a.voskBasePaths()
	modelDir := filepath.Join(root, spec.Folder)
	if _, err := os.Stat(filepath.Join(modelDir, "am")); err != nil {
		return false
	}
	if _, err := os.Stat(filepath.Join(modelDir, "conf")); err != nil {
		return false
	}
	return true
}

func (a *App) activeVoskSpec() VoskModelSpec {
	name := a.settings.VoiceName
	if _, s := a.comboSelection(ID_VOICE_COMBO); s != "" {
		name = s
	}
	if spec, ok := voskSpecByActiveDisplay(name); ok {
		return spec
	}
	for _, spec := range voskModelSpecs() {
		if a.voskModelInstalled(spec) {
			return spec
		}
	}
	return voskModelSpecs()[0]
}

func (a *App) voskPaths() (base, runtimeDir, modelDir string) {
	base, runtimeDir, root := a.voskBasePaths()
	spec := a.activeVoskSpec()
	modelDir = filepath.Join(root, spec.Folder)
	return
}

func (a *App) voskReady() bool {
	if !a.voskRuntimeReady() {
		return false
	}
	return a.voskModelInstalled(a.activeVoskSpec())
}

func (a *App) selectedLibraryModel() (VoskModelSpec, bool) {
	_, label := a.comboSelection(ID_VOSK_LIBRARY_COMBO)
	return voskSpecByLibraryLabel(label)
}

func (a *App) updateVoskLibraryStatus() {
	if a.controls[3010] == 0 {
		return
	}
	spec, ok := a.selectedLibraryModel()
	if !ok {
		setText(a.controls[3010], "LIBRARY STATUS: NO USABLE PACKS — FIELD MANUAL > LIBRARIES")
		return
	}
	if a.voskModelInstalled(spec) {
		active := a.activeVoskSpec()
		if strings.EqualFold(active.Folder, spec.Folder) && a.voskReady() {
			if a.isLibraryPackSaved("vosk", spec.Folder) {
				setText(a.controls[3010], "LIBRARY STATUS: ACTIVE — SAVED")
			} else {
				setText(a.controls[3010], "LIBRARY STATUS: ACTIVE — SAVE REQUIRED")
			}
		} else {
			setText(a.controls[3010], "LIBRARY STATUS: INSTALLED — USE SELECTED")
		}
	} else {
		setText(a.controls[3010], "LIBRARY STATUS: NOT INSTALLED")
	}
}

func (a *App) syncLanguageToModel(spec VoskModelSpec) {
	a.selectComboText(ID_LANG_COMBO, spec.Language)
	a.settings.Language = spec.Language
}

func (a *App) syncModelToLanguage(language string) {
	for _, spec := range voskModelSpecs() {
		if strings.EqualFold(spec.Language, language) && a.voskModelInstalled(spec) {
			a.selectComboText(ID_VOICE_COMBO, spec.ActiveDisplay)
			a.settings.VoiceName = spec.ActiveDisplay
			a.settings.VoiceIndex, _ = a.comboSelection(ID_VOICE_COMBO)
			return
		}
	}
}

func (a *App) useSelectedLibraryModel() {
	spec, ok := a.selectedLibraryModel()
	if !ok {
		a.updateVoskLibraryStatus()
		return
	}
	if !a.voskModelInstalled(spec) {
		setText(a.controls[3010], "LIBRARY STATUS: NOT INSTALLED — INSTALL / REPAIR FIRST")
		return
	}
	if a.recoRunning {
		a.stopRecognizer()
	}
	// Local discovery is instant: refresh the installed-model selectors in place.
	a.voices, a.languages = a.queryVoiceLanguage()
	a.setComboItems(ID_VOICE_COMBO, a.voices, 0)
	a.setComboItems(ID_LANG_COMBO, a.languages, 0)
	a.selectComboText(ID_VOICE_COMBO, spec.ActiveDisplay)
	a.syncLanguageToModel(spec)
	a.settings.VoiceName = spec.ActiveDisplay
	a.settings.VoiceIndex, _ = a.comboSelection(ID_VOICE_COMBO)
	setText(a.controls[3006], "STATUS: ACTIVE MODEL — "+spec.ActiveDisplay)
	setText(a.controls[3004], "SYSTEM STATUS: MODEL SWITCHED — SAVE SETUP TO KEEP THIS CHOICE")
	a.updateVoskLibraryStatus()
	a.announceLibraryEvent("Listening model selected. Click Save Setup to Disk to remember your choice.")
}

func (a *App) installVoskAsync() {
	spec, ok := a.selectedLibraryModel()
	if !ok {
		setText(a.controls[3010], "LIBRARY STATUS: SELECT A MODEL FIRST")
		return
	}
	a.installVoskModelAsync(spec)
}

func (a *App) installVoskModelAsync(spec VoskModelSpec) {
	// Repairing the active model while the recogniser is using its files is unsafe.
	// Stop the listener first; the operator can restart it after install/use completes.
	if a.recoRunning {
		a.stopRecognizer()
	}
	a.voskMu.Lock()
	if a.voskInstallRunning {
		a.voskMu.Unlock()
		return
	}
	a.voskInstallRunning = true
	a.voskInstallMsg = ""
	a.voskInstallModel = spec.LibraryLabel
	a.voskMu.Unlock()
	setText(a.controls[3010], "LIBRARY STATUS: INSTALLING — "+spec.LibraryLabel)
	setText(a.controls[3004], "SYSTEM STATUS: VOSK MODEL INSTALL — PLEASE WAIT")
	a.libraryBeginInstall("vosk", spec.LibraryLabel)
	go func() {
		base, runtimeDir, modelRoot := a.voskBasePaths()
		_ = os.MkdirAll(base, 0755)
		progressPath := a.libraryProgressPath("vosk")
		defer os.Remove(progressPath)
		ps := fmt.Sprintf(`$ErrorActionPreference='Stop';
$progress='%s'; function Report-Stage { param([string]$name) try { [IO.File]::WriteAllText($progress, $name) } catch {} };
$stage='PREPARE INSTALL'; Report-Stage $stage;
try {
  $base='%s'; $rt='%s'; $mroot='%s';
  New-Item -ItemType Directory -Force -Path $base,$rt,$mroot | Out-Null;

  $runtimeOK=$true;
  foreach($n in 'libvosk.dll','libgcc_s_seh-1.dll','libstdc++-6.dll','libwinpthread-1.dll'){ if(-not (Test-Path (Join-Path $rt $n))){$runtimeOK=$false} }
  if(-not $runtimeOK){
    $stage='DOWNLOAD VOSK RUNTIME'; Report-Stage $stage;
    $pkg=Join-Path $base 'vosk.0.3.38.nupkg';
    $pkgzip=Join-Path $base 'vosk.0.3.38.zip';
    $unz=Join-Path $base 'nuget';
    Invoke-WebRequest -UseBasicParsing 'https://api.nuget.org/v3-flatcontainer/vosk/0.3.38/vosk.0.3.38.nupkg' -OutFile $pkg;
    if(-not (Test-Path $pkg) -or (Get-Item $pkg).Length -lt 100000){ throw 'Downloaded Vosk runtime package is missing or unexpectedly small' }
    if(Test-Path $pkgzip){Remove-Item $pkgzip -Force}; Copy-Item $pkg $pkgzip -Force;
    $stage='EXTRACT VOSK RUNTIME'; Report-Stage $stage;
    if(Test-Path $unz){Remove-Item $unz -Recurse -Force};
    Expand-Archive -LiteralPath $pkgzip -DestinationPath $unz -Force;
    $stage='INSTALL VOSK RUNTIME FILES'; Report-Stage $stage;
    foreach($n in 'libvosk.dll','libgcc_s_seh-1.dll','libstdc++-6.dll','libwinpthread-1.dll'){
      $f=Get-ChildItem $unz -Recurse -File -Filter $n | Where-Object {$_.FullName -match 'win-x64|win64|runtimes\win-x64'} | Select-Object -First 1;
      if(-not $f){$f=Get-ChildItem $unz -Recurse -File -Filter $n | Select-Object -First 1};
      if(-not $f){throw "Missing $n in Vosk package"}; Copy-Item $f.FullName (Join-Path $rt $n) -Force;
    }
    Remove-Item $pkg,$pkgzip -Force -ErrorAction SilentlyContinue; if(Test-Path $unz){Remove-Item $unz -Recurse -Force -ErrorAction SilentlyContinue};
  }

  $stage='DOWNLOAD SELECTED VOSK MODEL'; Report-Stage $stage;
  $folder='%s'; $url='%s'; $mzip=Join-Path $base ($folder+'.zip');
  Invoke-WebRequest -UseBasicParsing $url -OutFile $mzip;
  if(-not (Test-Path $mzip) -or (Get-Item $mzip).Length -lt 1000000){ throw 'Downloaded Vosk model is missing or unexpectedly small' }

  $stage='EXTRACT SELECTED VOSK MODEL'; Report-Stage $stage;
  $target=Join-Path $mroot $folder;
  $pending=Join-Path $mroot ($folder+'.pending');
  $backup=Join-Path $mroot ($folder+'.backup');
  if(Test-Path $backup){throw 'A previous model backup still exists — repair must not overwrite it'};
  if(Test-Path $pending){Remove-Item $pending -Recurse -Force};
  Expand-Archive -LiteralPath $mzip -DestinationPath $pending -Force;
  $candidate=Join-Path $pending $folder;

  $stage='VERIFY SELECTED VOSK MODEL'; Report-Stage $stage;
  if(-not (Test-Path (Join-Path $candidate 'am'))){throw 'Model verification failed: am folder not found'};
  if(-not (Test-Path (Join-Path $candidate 'conf'))){throw 'Model verification failed: conf folder not found'};
  $stage='ACTIVATE INSTALLED VOSK FILES'; Report-Stage $stage;
  if(Test-Path $target){Move-Item -LiteralPath $target -Destination $backup};
  try {
    Move-Item -LiteralPath $candidate -Destination $target;
    if(Test-Path $backup){Remove-Item $backup -Recurse -Force};
    if(Test-Path $pending){Remove-Item $pending -Recurse -Force};
  } catch {
    if(Test-Path $target){Remove-Item $target -Recurse -Force};
    if(Test-Path $backup){Move-Item -LiteralPath $backup -Destination $target};
    throw;
  }
  Remove-Item $mzip -Force -ErrorAction SilentlyContinue;
  Write-Output 'OK|VOSK MODEL READY';
} catch {
  Write-Error ($stage + ' FAILED: ' + $_.Exception.Message); exit 1;
}`, psEsc(progressPath), psEsc(base), psEsc(runtimeDir), psEsc(modelRoot), psEsc(spec.Folder), psEsc(spec.URL))
		stopProgress := make(chan struct{})
		go a.watchLibraryInstallProgress(progressPath, "vosk", stopProgress)
		out, err := hiddenCommand("powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", ps).CombinedOutput()
		close(stopProgress)
		a.voskMu.Lock()
		if err != nil {
			detail := strings.TrimSpace(string(out))
			if detail == "" {
				detail = err.Error()
			}
			a.voskInstallMsg = "VOSK INSTALL FAILED: " + detail
		} else if !a.voskRuntimeReady() || !a.voskModelInstalled(spec) {
			a.voskInstallMsg = "VOSK INSTALL FINISHED BUT FILE VERIFICATION FAILED"
		} else {
			a.voskInstallMsg = "INSTALLED — " + spec.LibraryLabel + " — READY TO USE"
		}
		a.voskInstallRunning = false
		a.voskMu.Unlock()
		procPostMessageW.Call(uintptr(a.hwnd), WM_APP_VOSK_DONE, 0, 0)
	}()
}

func (a *App) applyVoskInstallResult() {
	a.voskMu.Lock()
	msg := a.voskInstallMsg
	a.voskMu.Unlock()
	if msg == "" {
		msg = "VOSK INSTALL STATUS UNKNOWN"
	}
	setText(a.controls[3004], "SYSTEM STATUS: "+msg)
	setText(a.controls[3010], "LIBRARY STATUS: "+msg)
	a.voices, a.languages = a.queryVoiceLanguage()
	a.applyVoiceLanguage()
	a.refreshInstalledLibrarySelectors()
	a.updateVoskLibraryStatus()
	a.libraryFinishInstall("vosk", msg, strings.HasPrefix(msg, "INSTALLED"))
	if a.currentScreen == "manual" && a.manualPage == "libraries" {
		a.refreshLibraryBrowserList()
		if strings.HasPrefix(msg, "INSTALLED") {
			a.setLibraryStatus("INSTALLED — READY TO SELECT IN SETTINGS\r\nCHOOSE USE SELECTED, THEN SAVE SETUP TO DISK")
		} else {
			a.setLibraryStatus(msg)
		}
	}
	if a.guideActive {
		a.applyGuideStep(false)
	}
}

func coordinateSpeech(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return v
	}
	if strings.Contains(v, ".") {
		parts := strings.SplitN(v, ".", 2)
		frac := ""
		for i, r := range parts[1] {
			if i > 0 {
				frac += " "
			}
			frac += string(r)
		}
		return strings.TrimSpace(parts[0] + " point " + frac)
	}
	return v
}

func (a *App) piperBasePaths() (base, runtimeDir, voicesDir string) {
	root := filepath.Dir(a.configPath)
	base = filepath.Join(root, "piper")
	runtimeDir = filepath.Join(base, "runtime")
	voicesDir = filepath.Join(base, "voices")
	return
}

func (a *App) piperVoicePaths(spec PiperVoiceSpec) (model, config, card string) {
	_, _, voicesDir := a.piperBasePaths()
	dir := filepath.Join(voicesDir, spec.Key)
	return filepath.Join(dir, spec.ModelBase+".onnx"), filepath.Join(dir, spec.ModelBase+".onnx.json"), filepath.Join(dir, "MODEL_CARD")
}

func (a *App) piperRuntimeReady() bool {
	_, runtimeDir, _ := a.piperBasePaths()
	st, err := os.Stat(filepath.Join(runtimeDir, "piper.exe"))
	return err == nil && !st.IsDir() && st.Size() > 100000
}

func (a *App) piperVoiceInstalled(spec PiperVoiceSpec) bool {
	model, config, _ := a.piperVoicePaths(spec)
	ms, me := os.Stat(model)
	cs, ce := os.Stat(config)
	return me == nil && ce == nil && !ms.IsDir() && !cs.IsDir() && ms.Size() > 1000000 && cs.Size() > 500
}

func (a *App) selectedActualLibraryVoice() (PiperVoiceSpec, bool) {
	_, label := a.comboSelection(ID_ACTUAL_LIBRARY_COMBO)
	return piperSpecByLabel(label)
}

func (a *App) activeActualVoice() (PiperVoiceSpec, bool) {
	if a.settings.ActualVoice == "" {
		return PiperVoiceSpec{}, false
	}
	return piperSpecByKey(a.settings.ActualVoice)
}

func (a *App) actualVoiceReady() bool {
	spec, ok := a.activeActualVoice()
	return ok && a.piperRuntimeReady() && a.piperVoiceInstalled(spec)
}

func (a *App) updateActualVoiceLibraryStatus() {
	if a.controls[3020] == 0 {
		return
	}
	spec, ok := a.selectedActualLibraryVoice()
	if !ok {
		setText(a.controls[3020], "VOICE STATUS: NO USABLE PACKS — FIELD MANUAL > LIBRARIES")
		return
	}
	installed := a.piperVoiceInstalled(spec) && a.piperRuntimeReady()
	active, activeOK := a.activeActualVoice()
	switch {
	case installed && activeOK && strings.EqualFold(active.Key, spec.Key):
		if a.isLibraryPackSaved("piper", spec.Key) {
			setText(a.controls[3020], "VOICE STATUS: ACTIVE — SAVED")
		} else {
			setText(a.controls[3020], "VOICE STATUS: ACTIVE — SAVE REQUIRED")
		}
	case installed:
		setText(a.controls[3020], "VOICE STATUS: INSTALLED — USE SELECTED")
	default:
		setText(a.controls[3020], "VOICE STATUS: NOT INSTALLED")
	}
}

func (a *App) useSelectedActualVoice() {
	spec, ok := a.selectedActualLibraryVoice()
	if !ok {
		a.updateActualVoiceLibraryStatus()
		return
	}
	if !a.piperRuntimeReady() || !a.piperVoiceInstalled(spec) {
		setText(a.controls[3020], "VOICE STATUS: NOT INSTALLED — INSTALL / REPAIR FIRST")
		return
	}
	a.settings.ActualVoice = spec.Key
	setText(a.controls[3004], "SYSTEM STATUS: ACTUAL VOICE SELECTED — "+spec.Label+" — SAVE SETUP TO KEEP")
	a.updateActualVoiceLibraryStatus()
	a.announceLibraryEvent("Voice selected. Click Save Setup to Disk to remember your choice.")
}

func (a *App) installSelectedActualVoice() {
	spec, ok := a.selectedActualLibraryVoice()
	if !ok {
		setText(a.controls[3020], "VOICE STATUS: SELECT A VOICE FIRST")
		return
	}
	a.installPiperVoiceAsync(spec)
}

func (a *App) installPiperVoiceAsync(spec PiperVoiceSpec) {
	a.piperMu.Lock()
	if a.piperInstallRunning {
		a.piperMu.Unlock()
		return
	}
	a.piperInstallRunning = true
	a.piperInstallMsg = ""
	a.piperInstallVoice = spec.Label
	a.piperMu.Unlock()

	setText(a.controls[3020], "VOICE STATUS: INSTALLING — "+spec.Label)
	setText(a.controls[3004], "SYSTEM STATUS: ACTUAL VOICE INSTALL — PLEASE WAIT")
	a.libraryBeginInstall("piper", spec.Label)
	go func() {
		base, runtimeDir, voicesDir := a.piperBasePaths()
		model, _, _ := a.piperVoicePaths(spec)
		voiceDir := filepath.Dir(model)
		_ = os.MkdirAll(base, 0755)
		_ = os.MkdirAll(runtimeDir, 0755)
		_ = os.MkdirAll(voicesDir, 0755)
		// The installer stages new files separately and only replaces the model
		// directory after verification. Never pre-create the live voice directory.
		baseURL := "https://huggingface.co/rhasspy/piper-voices/resolve/v1.0.0/" + spec.RepoPath
		progressPath := a.libraryProgressPath("piper")
		defer os.Remove(progressPath)
		ps := fmt.Sprintf(`$ErrorActionPreference='Stop';
$progress='%s'; function Report-Stage { param([string]$name) try { [IO.File]::WriteAllText($progress, $name) } catch {} };
$stage='PREPARE ACTUAL VOICE'; Report-Stage $stage;
try {
  $base='%s'; $rt='%s'; $voices='%s'; $vdir='%s';
  New-Item -ItemType Directory -Force -Path $base,$rt,$voices | Out-Null;

  $piper=Join-Path $rt 'piper.exe';
  if(-not (Test-Path $piper)){
    $stage='DOWNLOAD PIPER RUNTIME'; Report-Stage $stage;
    $zip=Join-Path $base 'piper_windows_amd64.zip';
    $unpack=Join-Path $base 'piper_runtime_unpack';
    Invoke-WebRequest -UseBasicParsing 'https://github.com/rhasspy/piper/releases/download/2023.11.14-2/piper_windows_amd64.zip' -OutFile $zip;
    if(-not (Test-Path $zip) -or (Get-Item $zip).Length -lt 1000000){throw 'Piper runtime download is missing or unexpectedly small'};
    $stage='EXTRACT PIPER RUNTIME'; Report-Stage $stage;
    if(Test-Path $unpack){Remove-Item $unpack -Recurse -Force};
    Expand-Archive -LiteralPath $zip -DestinationPath $unpack -Force;
    $pex=Get-ChildItem $unpack -Recurse -File -Filter 'piper.exe' | Select-Object -First 1;
    if(-not $pex){throw 'piper.exe not found in runtime archive'};
    $src=$pex.Directory.FullName;
    Copy-Item (Join-Path $src '*') $rt -Recurse -Force;
    Remove-Item $zip -Force -ErrorAction SilentlyContinue;
    if(Test-Path $unpack){Remove-Item $unpack -Recurse -Force -ErrorAction SilentlyContinue};
    try { Invoke-WebRequest -UseBasicParsing 'https://raw.githubusercontent.com/rhasspy/piper/master/LICENSE.md' -OutFile (Join-Path $rt 'PIPER_LICENSE_MIT.txt') } catch {}
  }

  $pending=$vdir+'.pending'; $backup=$vdir+'.backup';
  if(Test-Path $backup){throw 'A previous voice backup still exists — repair must not overwrite it'};
  if(Test-Path $pending){Remove-Item $pending -Recurse -Force};
  New-Item -ItemType Directory -Force -Path $pending | Out-Null;
  $pendingModel=Join-Path $pending '%s.onnx';
  $pendingConfig=Join-Path $pending '%s.onnx.json';
  $pendingCard=Join-Path $pending 'MODEL_CARD';
  $stage='DOWNLOAD ACTUAL VOICE MODEL'; Report-Stage $stage;
  Invoke-WebRequest -UseBasicParsing '%s/%s.onnx' -OutFile $pendingModel;
  $stage='DOWNLOAD ACTUAL VOICE CONFIG'; Report-Stage $stage;
  Invoke-WebRequest -UseBasicParsing '%s/%s.onnx.json' -OutFile $pendingConfig;
  $stage='DOWNLOAD ACTUAL VOICE MODEL CARD'; Report-Stage $stage;
  try { Invoke-WebRequest -UseBasicParsing '%s/MODEL_CARD' -OutFile $pendingCard } catch {}

  $stage='VERIFY ACTUAL VOICE'; Report-Stage $stage;
  if(-not (Test-Path $pendingModel) -or (Get-Item $pendingModel).Length -lt 1000000){throw 'Voice model verification failed'};
  if(-not (Test-Path $pendingConfig) -or (Get-Item $pendingConfig).Length -lt 500){throw 'Voice config verification failed'};
  if(-not (Test-Path (Join-Path $rt 'piper.exe'))){throw 'Piper runtime verification failed'};
  $stage='ACTIVATE INSTALLED VOICE FILES'; Report-Stage $stage;
  if(Test-Path $vdir){Move-Item -LiteralPath $vdir -Destination $backup};
  try {
    Move-Item -LiteralPath $pending -Destination $vdir;
    if(Test-Path $backup){Remove-Item $backup -Recurse -Force};
  } catch {
    if(Test-Path $vdir){Remove-Item $vdir -Recurse -Force};
    if(Test-Path $backup){Move-Item -LiteralPath $backup -Destination $vdir};
    throw;
  }
  Write-Output 'OK|ACTUAL VOICE READY';
} catch {
  Write-Error ($stage + ' FAILED: ' + $_.Exception.Message); exit 1;
}`,
			psEsc(progressPath), psEsc(base), psEsc(runtimeDir), psEsc(voicesDir), psEsc(voiceDir),
			psEsc(spec.ModelBase), psEsc(spec.ModelBase),
			psEsc(baseURL), psEsc(spec.ModelBase),
			psEsc(baseURL), psEsc(spec.ModelBase),
			psEsc(baseURL))
		stopProgress := make(chan struct{})
		go a.watchLibraryInstallProgress(progressPath, "piper", stopProgress)
		out, err := hiddenCommand("powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", ps).CombinedOutput()
		close(stopProgress)
		a.piperMu.Lock()
		if err != nil {
			detail := strings.TrimSpace(string(out))
			if detail == "" {
				detail = err.Error()
			}
			a.piperInstallMsg = "ACTUAL VOICE INSTALL FAILED: " + detail
		} else if !a.piperRuntimeReady() || !a.piperVoiceInstalled(spec) {
			a.piperInstallMsg = "ACTUAL VOICE INSTALL FINISHED BUT VERIFICATION FAILED"
		} else {
			a.piperInstallMsg = "INSTALLED — " + spec.Label + " — READY TO USE"
		}
		a.piperInstallRunning = false
		a.piperMu.Unlock()
		procPostMessageW.Call(uintptr(a.hwnd), WM_APP_PIPER_DONE, 0, 0)
	}()
}

func (a *App) applyPiperInstallResult() {
	a.piperMu.Lock()
	msg := a.piperInstallMsg
	a.piperMu.Unlock()
	if msg == "" {
		msg = "ACTUAL VOICE INSTALL STATUS UNKNOWN"
	}
	setText(a.controls[3004], "SYSTEM STATUS: "+msg)
	setText(a.controls[3020], "VOICE STATUS: "+msg)
	a.refreshInstalledLibrarySelectors()
	a.updateActualVoiceLibraryStatus()
	a.libraryFinishInstall("piper", msg, strings.HasPrefix(msg, "INSTALLED"))
	if a.currentScreen == "manual" && a.manualPage == "libraries" {
		a.refreshLibraryBrowserList()
		if strings.HasPrefix(msg, "INSTALLED") {
			a.setLibraryStatus("INSTALLED — READY TO SELECT IN SETTINGS\r\nCHOOSE USE SELECTED, THEN SAVE SETUP TO DISK")
		} else {
			a.setLibraryStatus(msg)
		}
	}
	if a.guideActive {
		a.applyGuideStep(false)
	}
}

func (a *App) isTtsSpeaking() bool {
	a.ttsMu.Lock()
	defer a.ttsMu.Unlock()
	return a.ttsSpeaking
}

func (a *App) actualRespond(text string) {
	if a.tourDemoMode || a.tourActive {
		return // guide narration owns Piper; no mixed operational speech
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	// The same response is visible in both diagnostic and operational views.
	a.addTranscript("ACTUAL: " + text)
	a.addOperationLog("ACTUAL: " + text)
	if !a.actualVoiceReady() || a.settings.PiperMuted || routinePiperConfirmation(text) {
		return
	}
	a.speakActualAsync(text)
}

// Cancel only obsolete operational speech. When a coordinate is corrected,
// old queued X/Y readbacks and any currently playing phrase must not delay or
// talk over the new short "Roger. Repeat ..." response. No Vosk state changes.
func (a *App) abandonPiperReadbacks() {
	a.ttsMu.Lock()
	if a.ttsCancel != nil {
		close(a.ttsCancel)
		a.ttsCancel = nil
	}
	a.ttsGeneration++
	a.ttsQueue = nil
	a.ttsSpeaking = false
	a.ttsAudioPlaying = false
	a.ttsInterruptedAt = time.Now()
	a.ttsCurrentText = ""
	a.ttsStatus = "INTERRUPTED — OLD READBACK CANCELLED"
	a.ttsMu.Unlock()
}

func (a *App) speakActualAsync(text string) {
	text = strings.TrimSpace(text)
	if text == "" || a.settings.PiperMuted {
		if text != "" {
			a.writeInteractionDiagnostic("PIPER_REQUEST", "muted=true; skipped", "SKIPPED", a.voiceDiagnosticState(), a.voiceDiagnosticState())
		}
		return
	}
	a.ttsMu.Lock()
	if a.ttsSpeaking {
		// G15 voice acknowledgements are short, but may arrive during an earlier
		// Piper WAV. Preserve them in order; never silently drop a valid step.
		a.ttsQueue = append(a.ttsQueue, text)
		a.ttsMu.Unlock()
		return
	}
	a.ttsSpeaking = true
	a.ttsAudioPlaying = false
	a.ttsGeneration++
	generation := a.ttsGeneration
	cancel := make(chan struct{})
	a.ttsCancel = cancel
	a.ttsCurrentText = text
	a.ttsStatus = "SPEAKING"
	a.ttsMu.Unlock()
	a.writeInteractionDiagnostic("PIPER_REQUEST", fmt.Sprintf("muted=false; phrase=%q", text), "STARTED", a.voiceDiagnosticState(), a.voiceDiagnosticState())

	a.captureSettingsFromUI()
	spec, ok := a.activeActualVoice()
	if !ok {
		a.ttsMu.Lock()
		a.ttsSpeaking = false
		a.ttsCurrentText = ""
		a.ttsCancel = nil
		a.ttsStatus = "NO ACTIVE VOICE"
		a.ttsMu.Unlock()
		return
	}
	_, runtimeDir, _ := a.piperBasePaths()
	model, config, _ := a.piperVoicePaths(spec)
	outputIndex := a.settings.OutputIndex
	speed := a.settings.VoiceSpeed
	if speed <= 0 {
		speed = 1
	}
	volume := a.settings.VoiceVolume
	if volume < 0 {
		volume = 0
	}
	tmp := filepath.Join(os.TempDir(), fmt.Sprintf("rawdogs_actual_%d.wav", time.Now().UnixNano()))
	setText(a.controls[3020], "VOICE STATUS: SPEAKING — "+spec.Label)

	go func() {
		defer os.Remove(tmp)
		lengthScale := 1.0 / speed
		exe := filepath.Join(runtimeDir, "piper.exe")
		cmd := hiddenCommand(exe, "--model", model, "--config", config, "--length_scale", fmt.Sprintf("%.3f", lengthScale), "--output_file", tmp)
		cmd.Stdin = strings.NewReader(text + "\n")
		var output bytes.Buffer
		cmd.Stdout, cmd.Stderr = &output, &output
		err := cmd.Start()
		if err == nil {
			wait := make(chan error, 1)
			go func() { wait <- cmd.Wait() }()
			select {
			case err = <-wait:
			case <-cancel:
				// Stop synthesis itself, not merely future WAV playback.
				_ = cmd.Process.Kill()
				<-wait
				return
			}
		}
		out := output.Bytes()
		status := "READY — " + spec.Label
		if err == nil {
			select {
			case <-cancel:
				// Correction arrived during synthesis. Never play this old phrase.
				return
			default:
				a.ttsMu.Lock()
				if a.ttsGeneration == generation {
					a.ttsAudioPlaying = true
				}
				a.ttsMu.Unlock()
				procInvalidateRect.Call(uintptr(a.hwnd), 0, 0)
				a.writeInteractionDiagnostic("PIPER_PLAYBACK", "audio started", "STARTED", a.voiceDiagnosticState(), a.voiceDiagnosticState())
				err = playWavOnDevice(tmp, outputIndex, volume, cancel)
				a.writeInteractionDiagnostic("PIPER_PLAYBACK", "audio ended", "FINISHED", a.voiceDiagnosticState(), a.voiceDiagnosticState())
			}
		}
		if err != nil {
			detail := strings.TrimSpace(string(out))
			if detail == "" {
				detail = err.Error()
			}
			status = "ACTUAL PLAYBACK FAILED — " + detail
		}
		a.ttsMu.Lock()
		if a.ttsGeneration != generation {
			a.ttsMu.Unlock()
			return // correction replaced this speech: don't affect new Piper response
		}
		a.ttsSpeaking = false
		a.ttsAudioPlaying = false
		a.ttsCurrentText = ""
		a.ttsCancel = nil
		a.ttsStatus = status
		a.ttsMu.Unlock()
		procPostMessageW.Call(uintptr(a.hwnd), WM_APP_TTS_DONE, uintptr(generation), 0)
	}()
}

func (a *App) applyTtsDone(completedGeneration uint64) {
	a.ttsMu.Lock()
	if a.ttsGeneration != completedGeneration {
		a.ttsMu.Unlock()
		return // A stale completion cannot drain the new response queue.
	}
	status := a.ttsStatus
	var next string
	if len(a.ttsQueue) > 0 {
		next = a.ttsQueue[0]
		a.ttsQueue = a.ttsQueue[1:]
	}
	a.ttsMu.Unlock()
	if next != "" {
		// Speech output is serialized; do not mistake a busy Piper process
		// for a missed cue. The last reply still owns the normal guide callback.
		a.speakActualAsync(next)
		return
	}
	if status == "" {
		status = "READY"
	}
	setText(a.controls[3020], "VOICE STATUS: "+status)
	procInvalidateRect.Call(uintptr(a.hwnd), 0, 0)

	if !a.guideActive {
		return
	}
	if a.guideStep == 2 && a.guideOutputTesting {
		a.guideOutputTesting = false
		if strings.HasPrefix(status, "READY") {
			a.guideOutputAwaiting = true
			setText(a.controls[3004], "DID YOU HEAR ACTUAL? CHOOSE HEARD ACTUAL OR TRY AGAIN.")
		} else {
			a.guideOutputAwaiting = false
			setText(a.controls[3004], "ACTUAL PLAYBACK DID NOT COMPLETE — CHECK THE OUTPUT DEVICE AND TRY AGAIN.")
		}
		a.syncGuideOutputButtons()
		procInvalidateRect.Call(uintptr(a.hwnd), 0, 1)
		return
	}
	if (a.guideStep == 3 || a.guideStep == 4) && a.guideInfoSpeechStep == a.guideStep {
		a.guideInfoSpeechStep = -1
		a.advanceGuideAutomatic()
	}
}

func playWavOnDevice(path string, deviceIndex int, volume int, cancel <-chan struct{}) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if len(b) < 44 || string(b[0:4]) != "RIFF" || string(b[8:12]) != "WAVE" {
		return fmt.Errorf("invalid WAV output")
	}
	var fmtData []byte
	var pcm []byte
	for off := 12; off+8 <= len(b); {
		id := string(b[off : off+4])
		n := int(binary.LittleEndian.Uint32(b[off+4 : off+8]))
		start := off + 8
		end := start + n
		if n < 0 || start < 0 || end > len(b) {
			break
		}
		switch id {
		case "fmt ":
			fmtData = append([]byte(nil), b[start:end]...)
		case "data":
			pcm = append([]byte(nil), b[start:end]...)
		}
		off = end
		if off%2 != 0 {
			off++
		}
	}
	if len(fmtData) < 16 || len(pcm) == 0 {
		return fmt.Errorf("WAV missing fmt/data")
	}
	format := WAVEFORMATEX{
		FormatTag:      binary.LittleEndian.Uint16(fmtData[0:2]),
		Channels:       binary.LittleEndian.Uint16(fmtData[2:4]),
		SamplesPerSec:  binary.LittleEndian.Uint32(fmtData[4:8]),
		AvgBytesPerSec: binary.LittleEndian.Uint32(fmtData[8:12]),
		BlockAlign:     binary.LittleEndian.Uint16(fmtData[12:14]),
		BitsPerSample:  binary.LittleEndian.Uint16(fmtData[14:16]),
	}
	if format.FormatTag != WAVE_FORMAT_PCM || format.BitsPerSample != 16 {
		return fmt.Errorf("unsupported WAV format: tag=%d bits=%d", format.FormatTag, format.BitsPerSample)
	}
	if volume < 0 {
		volume = 0
	}
	if volume > 100 {
		volume = 100
	}
	if volume != 100 {
		for i := 0; i+1 < len(pcm); i += 2 {
			v := int16(binary.LittleEndian.Uint16(pcm[i : i+2]))
			scaled := int32(v) * int32(volume) / 100
			binary.LittleEndian.PutUint16(pcm[i:i+2], uint16(int16(scaled)))
		}
	}

	var hwo uintptr
	device := uintptr(^uint32(0)) // WAVE_MAPPER fallback.
	if deviceIndex >= 0 {
		device = uintptr(deviceIndex)
	}
	r, _, _ := procWaveOutOpen.Call(uintptr(unsafe.Pointer(&hwo)), device, uintptr(unsafe.Pointer(&format)), 0, 0, CALLBACK_NULL)
	if r != 0 || hwo == 0 {
		return fmt.Errorf("waveOutOpen failed: %d", r)
	}
	defer procWaveOutClose.Call(hwo)

	hdr := WAVEHDR{Data: &pcm[0], BufferLength: uint32(len(pcm))}
	r, _, _ = procWaveOutPrepare.Call(hwo, uintptr(unsafe.Pointer(&hdr)), unsafe.Sizeof(hdr))
	if r != 0 {
		return fmt.Errorf("waveOutPrepareHeader failed: %d", r)
	}
	defer procWaveOutUnprepare.Call(hwo, uintptr(unsafe.Pointer(&hdr)), unsafe.Sizeof(hdr))

	r, _, _ = procWaveOutWrite.Call(hwo, uintptr(unsafe.Pointer(&hdr)), unsafe.Sizeof(hdr))
	if r != 0 {
		return fmt.Errorf("waveOutWrite failed: %d", r)
	}
	deadline := time.Now().Add(45 * time.Second)
	for hdr.Flags&WHDR_DONE == 0 {
		select {
		case <-cancel:
			// WinMM returns WHDR_DONE after reset, interrupting the audible
			// stale data immediately instead of waiting for the WAV to finish.
			procWaveOutReset.Call(hwo)
			return nil
		default:
		}
		if time.Now().After(deadline) {
			procWaveOutReset.Call(hwo)
			return fmt.Errorf("audio playback timeout")
		}
		time.Sleep(10 * time.Millisecond)
	}
	runtime.KeepAlive(pcm)
	runtime.KeepAlive(hdr)
	runtime.KeepAlive(format)
	return nil
}

func voskPowerShell(runtimeDir, modelDir string) string {
	return fmt.Sprintf(`$ErrorActionPreference='Stop'; $env:PATH='%s;'+$env:PATH;
$code=@'
using System;
using System.Runtime.InteropServices;
public static class RVosk {
 [DllImport("libvosk.dll", CallingConvention=CallingConvention.Cdecl)] public static extern void vosk_set_log_level(int l);
 [DllImport("libvosk.dll", CallingConvention=CallingConvention.Cdecl)] public static extern IntPtr vosk_model_new([MarshalAs(UnmanagedType.LPStr)] string p);
 [DllImport("libvosk.dll", CallingConvention=CallingConvention.Cdecl)] public static extern void vosk_model_free(IntPtr m);
 [DllImport("libvosk.dll", CallingConvention=CallingConvention.Cdecl)] public static extern IntPtr vosk_recognizer_new(IntPtr m, float r);
 [DllImport("libvosk.dll", CallingConvention=CallingConvention.Cdecl)] public static extern void vosk_recognizer_set_words(IntPtr r, int v);
 [DllImport("libvosk.dll", CallingConvention=CallingConvention.Cdecl)] public static extern int vosk_recognizer_accept_waveform(IntPtr r, byte[] d, int n);
 [DllImport("libvosk.dll", CallingConvention=CallingConvention.Cdecl)] public static extern IntPtr vosk_recognizer_result(IntPtr r);
 [DllImport("libvosk.dll", CallingConvention=CallingConvention.Cdecl)] public static extern IntPtr vosk_recognizer_partial_result(IntPtr r);
 [DllImport("libvosk.dll", CallingConvention=CallingConvention.Cdecl)] public static extern IntPtr vosk_recognizer_final_result(IntPtr r);
 [DllImport("libvosk.dll", CallingConvention=CallingConvention.Cdecl)] public static extern void vosk_recognizer_free(IntPtr r);
}
'@; Add-Type -TypeDefinition $code -Language CSharp;
[RVosk]::vosk_set_log_level(-1); $m=[RVosk]::vosk_model_new('%s'); if($m -eq [IntPtr]::Zero){Write-Output 'ERROR|VOSK MODEL FAILED TO LOAD'; exit 2}; $r=[RVosk]::vosk_recognizer_new($m,[single]16000); if($r -eq [IntPtr]::Zero){Write-Output 'ERROR|VOSK RECOGNISER FAILED TO START'; exit 3}; [RVosk]::vosk_recognizer_set_words($r,1); Write-Output 'READY|VOSK OFFLINE'; [Console]::Out.Flush();
$stdin=[Console]::OpenStandardInput(); $buf=New-Object byte[] 3200; $last=''; try { while(($n=$stdin.Read($buf,0,$buf.Length)) -gt 0){ $ok=[RVosk]::vosk_recognizer_accept_waveform($r,$buf,$n); if($ok -eq 1){$j=[Runtime.InteropServices.Marshal]::PtrToStringAnsi([RVosk]::vosk_recognizer_result($r)); if($j){Write-Output ('FINALJSON|'+($j.Replace([char]13,' ').Replace([char]10,' '))); [Console]::Out.Flush()}} else {$j=[Runtime.InteropServices.Marshal]::PtrToStringAnsi([RVosk]::vosk_recognizer_partial_result($r)); if($j -and $j -ne $last){$last=$j; Write-Output ('PARTIALJSON|'+($j.Replace([char]13,' ').Replace([char]10,' '))); [Console]::Out.Flush()}} } } finally { [RVosk]::vosk_recognizer_free($r); [RVosk]::vosk_model_free($m) }`, psEsc(runtimeDir), psEsc(modelDir))
}

func (a *App) startRecognizer() {
	a.recoMu.Lock()
	if a.recoRunning {
		a.recoMu.Unlock()
		return
	}
	a.recoMu.Unlock()
	a.captureSettingsFromUI()
	if a.settings.Permission == "" || a.settings.Permission == "not_now" {
		setText(a.controls[4100], "SYSTEM HEALTH: RED — MICROPHONE CONSENT NOT GIVEN")
		return
	}
	idx, mic := a.comboSelection(ID_MIC_COMBO)
	if idx < 0 || mic == "" {
		setText(a.controls[4101], "NOT READY: select a microphone first")
		return
	}
	if !a.voskReady() {
		setText(a.controls[4100], "SYSTEM HEALTH: RED — VOSK NOT INSTALLED")
		setText(a.controls[4101], "Open Settings and install a Vosk model from MODEL LIBRARY / MANAGER")
		return
	}
	if a.micRunning {
		a.stopMicTest()
	}
	_, runtimeDir, modelDir := a.voskPaths()
	cmd := hiddenCommand("powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", voskPowerShell(runtimeDir, modelDir))
	stdin, err := cmd.StdinPipe()
	if err != nil {
		setText(a.controls[4101], "VOSK START FAILED: "+err.Error())
		return
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		setText(a.controls[4101], "VOSK START FAILED: "+err.Error())
		return
	}
	cmd.Stderr = cmd.Stdout
	if err := cmd.Start(); err != nil {
		setText(a.controls[4101], "VOSK START FAILED: "+err.Error())
		return
	}
	stop := make(chan struct{})
	a.recoMu.Lock()
	a.recoCmd = cmd
	a.recoStdin = stdin
	a.recoAudioStop = stop
	a.recoRunning = true
	a.recoStatus = "STARTING"
	a.recoAudioLast, a.recoAudioAttempt, a.recoResultLast = time.Time{}, time.Time{}, time.Time{}
	a.recoMu.Unlock()
	a.syncListenerControls()
	setText(a.controls[4100], "SYSTEM HEALTH: AMBER — VOSK LISTENER STARTING")
	a.addTranscript("LISTENER STATE: STARTING | ENGINE: VOSK OFFLINE | INPUT: " + mic)
	go a.runVoskAudio(idx, a.settings.GainDB, stdin, stop)
	go func() {
		sc := bufio.NewScanner(stdout)
		sc.Buffer(make([]byte, 0, 65536), 1024*1024)
		for sc.Scan() {
			a.handleRecognizerLine(sc.Text())
		}
		err := cmd.Wait()
		a.recoMu.Lock()
		was := a.recoRunning
		a.recoRunning = false
		a.recoCmd = nil
		a.recoStdin = nil
		if was {
			if err != nil {
				a.recoStatus = "STOPPED — " + err.Error()
				a.recoPending = append(a.recoPending, RecoEvent{Kind: "EXIT", Detail: "VOSK LISTENER EXITED — " + err.Error()})
			} else {
				a.recoStatus = "STOPPED"
				a.recoPending = append(a.recoPending, RecoEvent{Kind: "EXIT", Detail: "VOSK LISTENER STOPPED"})
			}
		}
		a.recoMu.Unlock()
		procPostMessageW.Call(uintptr(a.hwnd), WM_APP_RECO_EVENT, 0, 0)
	}()
}

func (a *App) runVoskAudio(device int, gainDB float64, w io.WriteCloser, stop <-chan struct{}) {
	defer w.Close()
	var hwi uintptr
	format := WAVEFORMATEX{FormatTag: WAVE_FORMAT_PCM, Channels: 1, SamplesPerSec: 16000, BitsPerSample: 16, BlockAlign: 2, AvgBytesPerSec: 32000}
	r, _, _ := procWaveInOpen.Call(uintptr(unsafe.Pointer(&hwi)), uintptr(device), uintptr(unsafe.Pointer(&format)), 0, 0, CALLBACK_NULL)
	if r != 0 || hwi == 0 {
		a.queueReco(RecoEvent{Kind: "ERROR", Detail: fmt.Sprintf("MIC OPEN FAILED — ERROR %d", r)})
		return
	}
	defer procWaveInClose.Call(hwi)
	samples := make([]int16, 1600)
	hdr := WAVEHDR{Data: (*byte)(unsafe.Pointer(&samples[0])), BufferLength: uint32(len(samples) * 2)}
	r, _, _ = procWaveInPrepare.Call(hwi, uintptr(unsafe.Pointer(&hdr)), unsafe.Sizeof(hdr))
	if r != 0 {
		a.queueReco(RecoEvent{Kind: "ERROR", Detail: "MIC BUFFER PREPARE FAILED"})
		return
	}
	defer procWaveInUnprepare.Call(hwi, uintptr(unsafe.Pointer(&hdr)), unsafe.Sizeof(hdr))
	procWaveInStart.Call(hwi)
	a.micMu.Lock()
	a.micTested = true
	a.micMu.Unlock()
	procPostMessageW.Call(uintptr(a.hwnd), WM_APP_MIC_LEVEL, 0, 0)
	gain := math.Pow(10, gainDB/20)
	for {
		select {
		case <-stop:
			procWaveInStop.Call(hwi)
			procWaveInReset.Call(hwi)
			return
		default:
		}
		hdr.BytesRecorded = 0
		hdr.Flags &^= WHDR_DONE
		r, _, _ = procWaveInAddBuffer.Call(hwi, uintptr(unsafe.Pointer(&hdr)), unsafe.Sizeof(hdr))
		if r != 0 {
			time.Sleep(50 * time.Millisecond)
			continue
		}
		for hdr.Flags&WHDR_DONE == 0 {
			select {
			case <-stop:
				procWaveInStop.Call(hwi)
				procWaveInReset.Call(hwi)
				return
			default:
				time.Sleep(5 * time.Millisecond)
			}
		}
		count := int(hdr.BytesRecorded / 2)
		if count > len(samples) {
			count = len(samples)
		}
		peak := 0.0
		for i := 0; i < count; i++ {
			v := float64(samples[i]) * gain
			if v > 32767 {
				v = 32767
			}
			if v < -32768 {
				v = -32768
			}
			samples[i] = int16(v)
			av := math.Abs(v) / 32768
			if av > peak {
				peak = av
			}
		}
		a.micMu.Lock()
		a.micPeak = int(math.Min(100, peak*100))
		if a.isTtsSpeaking() {
			a.micStatus = "VOSK LISTENING — PIPER ALSO SPEAKING"
		} else {
			a.micStatus = "VOSK LISTENING"
		}
		a.micMu.Unlock()
		procPostMessageW.Call(uintptr(a.hwnd), WM_APP_MIC_LEVEL, 0, 0)
		// G16 FULL DUPLEX: audio continues to Vosk during Piper playback.
		// Headphones are recommended to prevent feedback into the microphone.
		b := unsafe.Slice((*byte)(unsafe.Pointer(&samples[0])), count*2)
		a.recoMu.Lock()
		a.recoAudioAttempt = time.Now()
		a.recoMu.Unlock()
		_, err := w.Write(b)
		if err == nil {
			a.recoMu.Lock()
			a.recoAudioLast = time.Now()
			a.recoMu.Unlock()
		}
		if err != nil {
			a.queueReco(RecoEvent{Kind: "ERROR", Detail: "VOSK AUDIO PIPE FAILED — " + err.Error()})
			return
		}
		runtime.KeepAlive(samples)
		runtime.KeepAlive(hdr)
	}
}

func (a *App) queueReco(e RecoEvent) {
	a.recoMu.Lock()
	a.recoPending = append(a.recoPending, e)
	a.recoMu.Unlock()
	procPostMessageW.Call(uintptr(a.hwnd), WM_APP_RECO_EVENT, 0, 0)
}

func (a *App) stopRecognizer() {
	a.hideCommsLoader()
	a.recoMu.Lock()
	cmd := a.recoCmd
	w := a.recoStdin
	stop := a.recoAudioStop
	a.recoRunning = false
	a.recoCmd = nil
	a.recoStdin = nil
	a.recoAudioStop = nil
	a.recoStatus = "STOPPED"
	a.recoAudioLast = time.Time{}
	a.recoMu.Unlock()
	a.lastRecoStatus = "VOSK STOPPED BY USER"
	setText(a.controls[5051], a.lastRecoStatus)
	a.setControlClass(5051, "red")
	procInvalidateRect.Call(uintptr(a.hwnd), 0, 0)
	if stop != nil {
		close(stop)
	}
	if w != nil {
		w.Close()
	}
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	if a.controls[4100] != 0 {
		setText(a.controls[4100], "SYSTEM HEALTH: AMBER — LISTENER STOPPED")
	}
	if a.controls[4101] != 0 {
		setText(a.controls[4101], "LISTENER STOPPED BY USER")
		a.addTranscript("LISTENER STATE: STOPPED BY USER")
	}
	a.updateOpsListenerStatus()
}

func (a *App) handleRecognizerLine(line string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}
	if strings.HasPrefix(line, "PARTIALJSON|") || strings.HasPrefix(line, "FINALJSON|") {
		a.recoMu.Lock()
		a.recoResultLast = time.Now()
		a.recoMu.Unlock()
	}
	if strings.HasPrefix(line, "READY|") {
		a.queueReco(RecoEvent{Kind: "READY", Detail: strings.TrimPrefix(line, "READY|")})
		return
	}
	if strings.HasPrefix(line, "ERROR|") {
		a.queueReco(RecoEvent{Kind: "ERROR", Detail: strings.TrimPrefix(line, "ERROR|")})
		return
	}
	if strings.HasPrefix(line, "PARTIALJSON|") {
		var v struct {
			Partial string `json:"partial"`
		}
		if json.Unmarshal([]byte(strings.TrimPrefix(line, "PARTIALJSON|")), &v) == nil && strings.TrimSpace(v.Partial) != "" {
			a.queueReco(RecoEvent{Kind: "PARTIAL", Text: v.Partial})
		}
		return
	}
	if strings.HasPrefix(line, "FINALJSON|") {
		var v struct {
			Text   string `json:"text"`
			Result []struct {
				Conf float64 `json:"conf"`
				Word string  `json:"word"`
			} `json:"result"`
		}
		if json.Unmarshal([]byte(strings.TrimPrefix(line, "FINALJSON|")), &v) == nil && strings.TrimSpace(v.Text) != "" {
			pct := 0
			if len(v.Result) > 0 {
				sum := 0.0
				for _, w := range v.Result {
					sum += w.Conf
				}
				pct = int(math.Round(sum / float64(len(v.Result)) * 100))
			}
			a.queueReco(RecoEvent{Kind: "HEARD", Text: v.Text, Confidence: pct})
		}
		return
	}
}

func (a *App) applyRecoEvents() {
	a.recoMu.Lock()
	events := append([]RecoEvent(nil), a.recoPending...)
	a.recoPending = nil
	running := a.recoRunning
	status := a.recoStatus
	a.recoMu.Unlock()
	for _, e := range events {
		switch e.Kind {
		case "READY":
			a.lastRecoStatus = "VOSK READY — microphone feed awaiting speech"
			setText(a.controls[5051], a.lastRecoStatus)
			a.setControlClass(5051, "green")
			a.recoMu.Lock()
			a.recoStatus = "READY"
			a.recoMu.Unlock()
			setText(a.controls[4100], "SYSTEM HEALTH: GREEN — VOSK OFFLINE LISTENER READY")
			setText(a.controls[4101], "READY — SAY COMMAND | "+e.Detail)
			a.addTranscript("LISTENER STATE: READY — SAY COMMAND | VOSK OFFLINE")
			if a.voiceTrain.Active && a.voiceTrain.PromptPending {
				a.voiceTrainingPrompt()
			}
			if a.guideActive && a.guideStep == 7 {
				a.finishGuidedSetup(true)
			}
		case "MICREADY":
			if a.currentScreen == "settings" && !a.guideActive {
				setText(a.controls[3004], "SYSTEM STATUS: MICROPHONE LINK ESTABLISHED — VOSK STARTUP CONTINUES")
			}
		case "ERROR":
			a.lastRecoClass = "red"
			a.lastRecoStatus = "LISTENER ERROR — " + e.Detail
			setText(a.controls[5051], a.lastRecoStatus)
			a.setControlClass(5051, "red")
			setText(a.controls[4105], "ERROR — "+e.Detail)
			setText(a.controls[4100], "SYSTEM HEALTH: RED — VOSK RECOGNITION OFFLINE")
			setText(a.controls[4101], e.Detail)
			a.addTranscript("LISTENER ERROR — " + e.Detail)
		case "EXIT":
			a.lastRecoClass = "red"
			a.lastRecoStatus = "VOSK STOPPED — " + e.Detail
			setText(a.controls[5051], a.lastRecoStatus)
			a.setControlClass(5051, "red")
			setText(a.controls[4105], "LISTENER EXITED — "+e.Detail)
			setText(a.controls[4100], "SYSTEM HEALTH: RED — LISTENER STOPPED UNEXPECTEDLY")
			a.addTranscript(e.Detail)
		case "PARTIAL":
			a.lastRecoClass = "neutral"
			a.lastRecoStatus = "RAW PARTIAL: " + e.Text + " | CONFIDENCE: pending Vosk final"
			setText(a.controls[4105], "PARTIAL: "+e.Text)
			setText(a.controls[5051], a.lastRecoStatus)
			a.setControlClass(5051, "pink")
			// Vosk partials have NO trustworthy confidence yet: we may silence
			// old Piper on a complete harmless command, but NEVER execute it
			// or change coordinate state until a confidence-accepted FINAL.
			if a.isTtsSpeaking() && likelyBargeInPartial(e.Text) {
				a.ttsMu.Lock()
				spoken := a.ttsCurrentText
				a.ttsMu.Unlock()
				if !likelyPiperEcho(e.Text, spoken) {
					a.abandonPiperReadbacks()
					a.addOperationLog("PIPER INTERRUPTED — Vosk partial command; awaiting final confidence")
				}
			}
			procInvalidateRect.Call(uintptr(a.controls[4105]), 0, 1)
		case "HEARD":
			a.ttsMu.Lock()
			piperPhrase := a.ttsCurrentText
			a.ttsMu.Unlock()
			if likelyPiperEcho(e.Text, piperPhrase) {
				a.writeVoiceDiagnostic("HEARD", e.Text, "", "PIPER ECHO", "IGNORED ECHO", e.Confidence, 0, a.voiceDiagnosticState(), a.voiceDiagnosticState())
				a.addTranscript("IGNORED PIPER ECHO: " + e.Text)
				continue
			}
			if a.voiceTrain.Active {
				a.voiceTrainingHeard(e)
				continue // no calibration utterance may alter firing data
			}
			a.ensureVoiceProfile()
			stateBefore := a.voiceDiagnosticState()
			expectedAxis := a.expectedCoordinateAxis()
			armedAxis := expectedAxis != "" && (a.pending.CueAwaiting == expectedAxis || a.pending.Replacing == expectedAxis)
			adapted, adaptationWhy := a.voiceProfile.adaptCoordinate(e.Text, expectedAxis, armedAxis)
			interpreted, why := a.interpretSpeechForContext(adapted)
			if adaptationWhy != "" {
				if why != "" {
					why += "; "
				}
				why += adaptationWhy
			}
			matched, canonical := classifyCue(interpreted)
			piperCmd, isPiperControl := parsePiperCommand(interpreted)
			if isPiperControl {
				matched, canonical = true, "PIPER "+strings.ToUpper(piperCmd.Kind)
			}
			expected := a.expectedCoordinateAxis()
			pendingKindBefore := a.pending.Kind
			armedBefore := expected != "" && (a.pending.CueAwaiting == expected || a.pending.Replacing == expected)
			axisOnly := loneAxisCue(interpreted)
			axisCueReady := expected != "" && axisOnly == expected
			if axisCueReady {
				matched, canonical = true, expected+" CUE — WAITING FOR DIGITS"
			}
			if expected != "" && (a.pending.CueAwaiting == expected || a.pending.Replacing == expected) {
				if _, ok := parseCoordinateValue(interpreted); ok {
					matched, canonical = true, expected+" DIGITS"
				}
			}
			if _, ok := a.spokenCoordinateConfirm(interpreted); ok {
				matched, canonical = true, expected+" VALUE + CONFIRM"
			}
			configuredThreshold := a.settings.Confidence
			if configuredThreshold == 0 {
				configuredThreshold = 80
			}
			// G15: lower confidence only for a command in the EXPECTED six-step stage.
			// Values require a valid parsed number and an axis cue; dangerous commands
			// (CLEAR ALL, CLEAR ACTUAL etc.) keep the configured threshold.
			activation := (interpreted == "refresh actual" &&
				(a.pending.Kind == "" || a.pending.Kind == "actual" || a.pending.Kind == "switch_actual")) ||
				(interpreted == "refresh target" && a.actualSet &&
					(a.pending.Kind == "" || a.pending.Kind == "target"))
			stage := ""
			if activation {
				stage = "activation"
			}
			if axisCueReady {
				stage = "axis"
			}
			if expected != "" && a.pending.CueAwaiting == expected {
				if _, ok := parseCoordinateValue(interpreted); ok {
					stage = "value"
				}
			}
			if expected != "" {
				if axis, val, ok := parseCoordinateCue(interpreted); ok && axis == expected && val != "" {
					stage = "value"
				}
			}
			if _, ok := a.spokenCoordinateConfirm(interpreted); ok {
				stage = "value"
			}
			if interpreted == "correction" && (expected != "" || a.lastCoordAxis != "") {
				stage = "axis"
			} // recovery cue
			// G18F: exact numbered recall is globally prioritised across
			// coordinate stages; the confidence relaxation applies only when
			// the referenced target slot is ALREADY SAVED. Never to CLEAR.
			if isPiperControl {
				// These are explicit, specific Piper cues. Keep conservative
				// recognition confidence; do not change ACTUAL/TARGET rules.
				stage = "piper"
			}
			if matched {
				if slot, ok := parseRecallTargetPhrase(interpreted); ok && a.targets[slot].Confirmed {
					stage = "recall"
				}
			}
			threshold := sixStepThreshold(configuredThreshold, stage)
			class := "red"
			verdict := "REJECTED CONFIDENCE"
			if e.Confidence >= threshold {
				if matched {
					class, verdict = "pink", "VALID RAWDOGS CUE"
				} else {
					class, verdict = "green", "ACCEPTED CONFIDENCE"
				}
			}
			a.lastRecoClass = class
			cue := "NOT A RAWDOGS COMMAND — IGNORED"
			if matched && e.Confidence >= threshold {
				cue = "VALID RAWDOGS CUE: " + canonical
			}
			setText(a.controls[4105], fmt.Sprintf("%s — %d%% — %s", verdict, e.Confidence, e.Text))
			a.lastRecoStatus = fmt.Sprintf("RAW HEARD: %q    CONFIDENCE: %d%% / %d%%    INTERPRETED: %q    %s",
				e.Text, e.Confidence, threshold, interpreted, verdict)
			setText(a.controls[5051], a.lastRecoStatus)
			a.setControlClass(5051, class)
			procInvalidateRect.Call(uintptr(a.controls[4105]), 0, 1)
			if e.Confidence >= threshold {
				a.addTranscript(fmt.Sprintf("CONFIDENCE PASS: %d%%", e.Confidence))
			} else {
				a.addTranscript(fmt.Sprintf("REJECTED: confidence %d%% below %d%% threshold", e.Confidence, threshold))
			}
			if matched && e.Confidence >= threshold {
				a.addTranscript(fmt.Sprintf("CMD: Heard %q | %s", e.Text, cue))
			} else {
				a.addTranscript(fmt.Sprintf("Heard: %q — %d%% | %s", e.Text, e.Confidence, cue))
			}
			if why != "" {
				a.addTranscript(fmt.Sprintf("Interpreted: %q | %s", interpreted, why))
			}
			// G9 universal voice engine: one listener/state machine survives tab changes.
			if e.Confidence >= threshold && matched {
				// G18E: an accepted operator cue takes priority over the old
				// Piper readback, even when spoken during playback.
				// Not triggered by ordinary noise or low-confidence guesses.
				a.abandonPiperReadbacks()
				a.addOperationLog(fmt.Sprintf("CMD: %s — HEARD %q — %d%%", canonical, e.Text, e.Confidence))
				if why != "" {
					a.addOperationLog("CMD: INTERPRETED — " + strings.ToUpper(interpreted))
				}
				a.processOperationalSpeech(interpreted)
			} else if matched {
				a.addOperationLog(fmt.Sprintf("REJECTED: %s — %d%% BELOW %d%% THRESHOLD", canonical, e.Confidence, threshold))
			}
			// Optional cue coaching: consider only clearly attempted coordinates
			// while the existing state machine awaits X or Y. Ordinary chat,
			// accepted inputs and low-confidence noise cannot trigger advice.
			// This never alters Vosk thresholds, pending data or firing maths.
			if expected != "" && !a.guideActive && !a.manualNarrating {
				stageKey := pendingKindBefore + ":" + expected
				// A correct axis can be armed without advancing to the next
				// X/Y step; that's successful progress, not a failed attempt.
				correctCueArmed := matched && axisOnly == expected && e.Confidence >= threshold
				accepted := a.expectedCoordinateAxis() != expected || a.pending.Kind != pendingKindBefore || correctCueArmed
				if a.cueCoach.note(stageKey, cueAttempt(interpreted, armedBefore), accepted, e.Confidence, time.Now()) {
					hint := cueCoachPrompt(pendingKindBefore, expected, armedBefore)
					a.addOperationLog("VOICE HELP — " + hint)
					a.actualRespond(hint)
				}
			} else {
				a.cueCoach.reset()
			}
			outcome := "IGNORED — NOT MATCHED"
			if e.Confidence < threshold {
				outcome = "REJECTED — LOW CONFIDENCE"
			} else if matched {
				outcome = "DISPATCHED"
			}
			a.writeVoiceDiagnostic("HEARD", e.Text, interpreted, canonical+" | "+why, outcome, e.Confidence, threshold, stateBefore, a.voiceDiagnosticState())
		}
	}
	if !running && status == "STOPPED" {
		setText(a.controls[4100], "SYSTEM HEALTH: AMBER — LISTENER STOPPED")
	}
	a.updateTeamTrainingStatus()
	a.updateOpsListenerStatus()
}

func (a *App) addTranscript(line string) {
	a.transcript = append(a.transcript, line)
	if len(a.transcript) > 40 {
		a.transcript = a.transcript[len(a.transcript)-40:]
	}
	for _, listID := range []int{4102, 5052} {
		if h := a.controls[listID]; h != 0 {
			procSendMessageW.Call(uintptr(h), LB_RESETCONTENT, 0, 0)
			for _, v := range a.transcript {
				procSendMessageW.Call(uintptr(h), LB_ADDSTRING, 0, uintptr(unsafe.Pointer(utf16Ptr(v))))
			}
			var rc RECT
			procGetClientRect.Call(uintptr(h), uintptr(unsafe.Pointer(&rc)))
			visibleRows := maxInt(2, int(rc.Bottom-rc.Top)/22)
			if len(a.transcript) > 0 {
				top := maxInt(0, len(a.transcript)-visibleRows)
				procSendMessageW.Call(uintptr(h), LB_SETTOPINDEX, uintptr(top), 0)
			}
		}
	}
}

func (a *App) currentListenerState() string {
	a.recoMu.Lock()
	defer a.recoMu.Unlock()
	if a.recoRunning {
		if a.recoStatus == "READY" {
			return "READY — SAY COMMAND"
		}
		return "STARTING"
	}
	if strings.HasPrefix(a.recoStatus, "ERROR") || strings.HasPrefix(a.recoStatus, "STOPPED —") {
		return a.recoStatus
	}
	return "STOPPED"
}

func (a *App) updateTeamTrainingStatus() {
	team := a.settings.Team
	if team == "" {
		team = "MANTICORE — GREEN"
	}
	a.recoMu.Lock()
	running := a.recoRunning
	a.recoMu.Unlock()
	ls := "LISTENER STOPPED"
	if running {
		ls = "LISTENER ACTIVE"
	}
	if h := a.controls[4104]; h != 0 {
		setText(h, "TEAM: "+strings.ToUpper(team)+" | "+ls)
	}
}

func (a *App) healthLoop(stop <-chan struct{}) {
	t := time.NewTicker(4 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-stop:
			return
		case <-t.C:
			a.recoMu.Lock()
			micName := a.settings.MicName
			outName := a.settings.OutputName
			a.recoMu.Unlock()
			ins := enumerateWaveIn()
			outs := enumerateWaveOut()
			a.healthMu.Lock()
			a.healthMicOK = micName == "" || containsFold(ins, micName)
			a.healthOutOK = outName == "" || containsFold(outs, outName)
			a.healthMu.Unlock()
			procPostMessageW.Call(uintptr(a.hwnd), WM_APP_HEALTH, 0, 0)
		}
	}
}

func containsFold(list []string, want string) bool {
	for _, v := range list {
		if strings.EqualFold(v, want) || strings.Contains(strings.ToLower(v), strings.ToLower(want)) || strings.Contains(strings.ToLower(want), strings.ToLower(v)) {
			return true
		}
	}
	return false
}
func (a *App) applyHealth() {
	a.healthMu.Lock()
	micOK, outOK := a.healthMicOK, a.healthOutOK
	a.healthMu.Unlock()
	a.recoMu.Lock()
	rr := a.recoRunning
	rs := a.recoStatus
	a.recoMu.Unlock()
	if rr && rs == "READY" && micOK && outOK {
		setText(a.controls[4100], "SYSTEM HEALTH: GREEN — MIC / OUTPUT / LISTENER READY")
	} else if !micOK {
		setText(a.controls[4100], "SYSTEM HEALTH: RED — SELECTED MICROPHONE LOST")
	} else if !outOK {
		setText(a.controls[4100], "SYSTEM HEALTH: RED — SELECTED AUDIO OUTPUT LOST")
	} else if rr {
		setText(a.controls[4100], "SYSTEM HEALTH: AMBER — LISTENER STARTING / CHECKING")
	}
	a.checkPendingTimeout()
	a.updateOpsListenerStatus()
}

func (a *App) applyTheme() {
	a.captureSettingsFromUI()
	isLight := strings.EqualFold(a.settings.Theme, "Light")
	var brush syscall.Handle
	if isLight {
		brush = a.lightBrush
	} else {
		brush = a.darkBrush
	}
	_ = brush
	procInvalidateRect.Call(uintptr(a.hwnd), 0, 1)
	team := strings.ToUpper(a.settings.Team)
	if team == "" {
		team = "MANTICORE — GREEN"
	}
	title := fmt.Sprintf("RAWDOGS Voice Controlled Field Calculator — Preview build 0.9.4 — %s / %s", strings.ToUpper(a.settings.Theme), team)
	procSetWindowTextW.Call(uintptr(a.hwnd), uintptr(unsafe.Pointer(utf16Ptr(title))))
	setText(a.controls[9001], strings.ReplaceAll(strings.ToUpper(team), " — ", "  /  ")+" TEAM")
	a.updateTeamTrainingStatus()
	a.updateOpsListenerStatus()
}

func createFont(height int32, weight int32, face string) uintptr {
	v, _, _ := procCreateFontW.Call(uintptr(height), 0, 0, 0, uintptr(weight), 0, 0, 0, 1, 0, 0, 5, 0, uintptr(unsafe.Pointer(utf16Ptr(face))))
	return v
}
func (a *App) setFont(h syscall.Handle, f syscall.Handle) {
	if h != 0 && f != 0 {
		procSendMessageW.Call(uintptr(h), WM_SETFONT, uintptr(f), 1)
	}
}

func decodeCamo(data []byte) *CamoImage {
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	px := make([]byte, w*h*4)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, bb, _ := img.At(b.Min.X+x, b.Min.Y+(h-1-y)).RGBA()
			i := (y*w + x) * 4
			px[i] = byte(bb >> 8)
			px[i+1] = byte(g >> 8)
			px[i+2] = byte(r >> 8)
			px[i+3] = 0
		}
	}
	return &CamoImage{W: w, H: h, BGRA: px}
}
func loadCamos() map[string]*CamoImage {
	return map[string]*CamoImage{"blue": decodeCamo(camoBluePNG), "red": decodeCamo(camoRedPNG), "green": decodeCamo(camoGreenPNG)}
}
func (a *App) teamKey() string {
	t := strings.ToLower(a.settings.Team)
	if strings.Contains(t, "blue") {
		return "blue"
	}
	if strings.Contains(t, "red") {
		return "red"
	}
	return "green"
}
func teamColor(key string) (byte, byte, byte) {
	switch key {
	case "blue":
		return 30, 112, 194
	case "red":
		return 188, 55, 55
	default:
		return 39, 148, 77
	}
}
func fillColor(hdc syscall.Handle, rc RECT, r, g, b byte) {
	br := syscall.Handle(createSolidBrush(r, g, b))
	procFillRect.Call(uintptr(hdc), uintptr(unsafe.Pointer(&rc)), uintptr(br))
	procDeleteObject.Call(uintptr(br))
}
func frameColor(hdc syscall.Handle, rc RECT, r, g, b byte) {
	br := syscall.Handle(createSolidBrush(r, g, b))
	procFrameRect.Call(uintptr(hdc), uintptr(unsafe.Pointer(&rc)), uintptr(br))
	procDeleteObject.Call(uintptr(br))
}
func (a *App) drawCamoTile(hdc syscall.Handle, ci *CamoImage, x, y, w, h int) {
	if ci == nil || len(ci.BGRA) == 0 {
		return
	}
	bmi := BITMAPINFO{Header: BITMAPINFOHEADER{Size: uint32(unsafe.Sizeof(BITMAPINFOHEADER{})), Width: int32(ci.W), Height: int32(ci.H), Planes: 1, BitCount: 32, Compression: 0, SizeImage: uint32(len(ci.BGRA))}}
	procStretchDIBits.Call(uintptr(hdc), uintptr(x), uintptr(y), uintptr(w), uintptr(h), 0, 0, uintptr(ci.W), uintptr(ci.H), uintptr(unsafe.Pointer(&ci.BGRA[0])), uintptr(unsafe.Pointer(&bmi)), DIB_RGB_COLORS, SRCCOPY)
}
func (a *App) drawTacticalChrome(hdc syscall.Handle, width, height int) {
	key := a.teamKey()
	ci := a.camos[key]
	// Native-size TILE/REPEAT across the *entire parent background*. Panels
	// painted below are fully opaque so camouflage is never inside a workspace.
	if ci != nil && ci.W > 0 && ci.H > 0 {
		for y := 0; y < height; y += ci.H {
			for x := 0; x < width; x += ci.W {
				a.drawCamoTile(hdc, ci, x, y, ci.W, ci.H)
			}
		}
	}
	fillColor(hdc, RECT{0, 134, int32(width), 138}, 205, 172, 103)
	light := strings.EqualFold(a.settings.Theme, "Light")
	panelR, panelG, panelB := byte(11), byte(17), byte(18)
	if light {
		panelR, panelG, panelB = 248, 247, 242
	}
	goldR, goldG, goldB := byte(197), byte(160), byte(91)
	drawPanel := func(x, y, ww, hh int) {
		rc := RECT{int32(x), int32(y), int32(x + ww), int32(y + hh)}
		fillColor(hdc, rc, panelR, panelG, panelB)
		frameColor(hdc, rc, goldR, goldG, goldB)
		fillColor(hdc, RECT{int32(x), int32(y), int32(x + ww), int32(y + 4)}, goldR, goldG, goldB)
	}

	// G6 registration rule: compute panel geometry in the SAME baseline width
	// used when the controls were created, then transform panel X/width with
	// registerX/registerWidth. This is the single source of truth that G5 lacked.
	baseW := a.layoutBaseW
	if baseW <= 0 {
		baseW = width
	}
	drawRegisteredPanel := func(x, y, ww, hh int) {
		drawPanel(a.registerX(x, width), y, a.registerWidth(ww, width), hh)
	}

	switch a.currentScreen {
	case "settings":
		margin := 20
		gap := 18
		usable := baseW - margin*2 - gap*2
		colW := usable / 3
		x1 := margin
		x2 := x1 + colW + gap
		x3 := x2 + colW + gap
		drawRegisteredPanel(x1, 206, colW, 610)
		drawRegisteredPanel(x2, 206, colW, 610)
		drawRegisteredPanel(x3, 206, colW, 610)
		// G9 Vosk + G10 ACTUAL libraries use the same registration transform as their controls.
		drawRegisteredPanel(x2+10, 622, colW-20, 190)
		drawRegisteredPanel(x3+10, 674, colW-20, 138)
		drawRegisteredPanel(margin, 824, baseW-margin*2, 120)
	case "operations":
		margin := 20
		gap := 16
		usable := baseW - margin*2
		cueW := usable * 27 / 100
		if cueW < 330 {
			cueW = 330
		}
		left := usable - cueW - gap
		c1 := (left - gap*2) / 3
		c2 := c1
		c3 := left - gap*2 - c1 - c2
		x1 := margin
		x2 := x1 + c1 + gap
		x3 := x2 + c2 + gap
		x4 := x3 + c3 + gap
		drawRegisteredPanel(x1, 258, c1, 246)
		drawRegisteredPanel(x2, 258, c2, 246)
		drawRegisteredPanel(x3, 258, c3, 270)
		drawRegisteredPanel(x4, 258, cueW, 466)
		drawRegisteredPanel(x1, 524, c1+gap+c2, 200)
		drawRegisteredPanel(x3, 524, c3, 200)
		// Both panes are bounded by the REAL client height, including the
		// short-window compact mode. The panel and its children use one layout.
		d := operationDiagnosticsForHeight(height)
		if d.transcriptHeight > 0 {
			drawRegisteredPanel(20, d.transcriptTop, baseW-40, d.transcriptHeight)
		}
		if d.liveHeight > 0 {
			drawRegisteredPanel(20, d.liveTop, baseW-40, d.liveHeight)
		}
		a.drawFiringSolutionOutline(hdc, width, x3, c3)
	case "training":
		// Panels follow exactly the same dynamic vertical/horizontal rectangles as
		// native controls, and use all available client height.
		boxes := trainingBoxes(width, height)
		for _, b := range []trainingBox{boxes.leftTop, boxes.leftBottom, boxes.rightTop, boxes.rightBottom, boxes.footer} {
			drawPanel(b.x, b.y, b.w, b.h)
		}
	case "manual":
		margin := 20
		gap := 16
		paperW := (baseW - margin*2 - gap) * 68 / 100
		if paperW < 760 {
			paperW = 760
		}
		sideX := margin + paperW + gap
		sideW := baseW - margin - sideX
		rx := a.registerX(margin, width)
		rpw := a.registerWidth(paperW, width)
		drawRegisteredPanel(margin, 938, paperW, 98) // Persistent Walkman-style player strip.
		paper := RECT{int32(rx), 236, int32(rx + rpw), 930}
		fillColor(hdc, paper, 239, 226, 190)
		frameColor(hdc, paper, goldR, goldG, goldB)
		bandW := a.registerWidth(32, width)
		fillColor(hdc, RECT{int32(rx), 236, int32(rx + bandW), 930}, 226, 209, 170)
		fillColor(hdc, RECT{int32(rx + rpw - bandW), 236, int32(rx + rpw), 930}, 226, 209, 170)
		holeBrush := syscall.Handle(createSolidBrush(22, 27, 29))
		oldBrush, _, _ := procSelectObject.Call(uintptr(hdc), uintptr(holeBrush))
		for yy := 252; yy < 912; yy += 36 {
			l1 := a.registerX(margin+9, width)
			l2 := a.registerX(margin+23, width)
			r1 := a.registerX(margin+paperW-23, width)
			r2 := a.registerX(margin+paperW-9, width)
			procEllipse.Call(uintptr(hdc), uintptr(l1), uintptr(yy), uintptr(l2), uintptr(yy+14))
			procEllipse.Call(uintptr(hdc), uintptr(r1), uintptr(yy), uintptr(r2), uintptr(yy+14))
		}
		procSelectObject.Call(uintptr(hdc), oldBrush)
		procDeleteObject.Call(uintptr(holeBrush))
		drawRegisteredPanel(sideX, 236, sideW, 694)
	}
	// Two views share the same Vosk / Piper status, no duplicate listeners.
	if a.currentScreen == "operations" || a.currentScreen == "settings" || a.currentScreen == "training" || a.currentScreen == "manual" {
		a.drawVoiceBadges(hdc, width)
	}
	// G17: steady blue outline ONLY around the currently expected or focused X/Y.
	a.drawCoordinateStepOutline(hdc, width)
	// Existing guide outline is inactive during the new tour.
	a.drawGuideHighlight(hdc, width)
	a.drawTourHighlights(hdc)
}

// A navy solution border is a visual acknowledgement, not the green
// "accepted" control state and not a change to the calculator maths.
func (a *App) drawFiringSolutionOutline(hdc syscall.Handle, width, x3, col3 int) {
	if a.firingBorderState == "" {
		return
	}

	x, ww := a.registerX(x3, width), a.registerWidth(col3, width)
	rr, gg, bb := byte(22), byte(74), byte(148) // NAVY BLUE, also visible in LIGHT theme
	if a.firingBorderState == "out_of_range" {
		rr, gg, bb = 185, 54, 54
	}
	for inset := 0; inset < 7; inset++ {
		frameColor(hdc, RECT{int32(x + inset), int32(524 + inset), int32(x + ww - inset), int32(724 - inset)}, rr, gg, bb)
	}
}

// G18F: native GDI badges do not add focusable child controls or alter layout.
// A real no-entry glyph is superimposed on the monkey when not active.
func (a *App) drawVoiceBadges(hdc syscall.Handle, width int) {
	a.recoMu.Lock()
	run, state, fedAt, resultAt := a.recoRunning, a.recoStatus, a.recoAudioLast, a.recoResultLast
	a.recoMu.Unlock()
	a.ttsMu.Lock()
	busy, speaking, interruptedAt := a.ttsSpeaking, a.ttsAudioPlaying, a.ttsInterruptedAt
	a.ttsMu.Unlock()
	a.narratorMu.Lock()
	manualNarrating := a.narratorRunning
	manualMuted := a.manualPlayerMute
	a.narratorMu.Unlock()
	voxText := "STOPPED"
	voxActive := false
	voxR, voxG, voxB := byte(235), byte(80), byte(80)
	if run && state == "READY" {
		if fedAt.IsZero() || time.Since(fedAt) > 1300*time.Millisecond {
			voxText = "FEED STALLED"
		} else {
			voxActive = true
			voxR, voxG, voxB = 70, 210, 105
			voxText = "FEED ACTIVE"
			if !resultAt.IsZero() && time.Since(resultAt) < 650*time.Millisecond {
				voxText = "RECOGNISING"
				voxR, voxG, voxB = 255, 0, 180
			}
		}
	} else if run {
		voxText = "STARTING"
		voxR, voxG, voxB = 240, 190, 70
	}
	pipText, pipActive := "IDLE", false
	pipR, pipG, pipB := byte(180), byte(180), byte(180)
	if a.settings.PiperMuted {
		pipText = "MUTED"
	} else if manualNarrating && !manualMuted {
		pipText = "NARRATING"
		pipActive = true
		pipR, pipG, pipB = 70, 210, 105
	} else if manualNarrating && manualMuted {
		pipText = "FM MUTED"
	} else if busy && speaking {
		pipText = "SPEAKING"
		pipActive = true
		pipR, pipG, pipB = 70, 210, 105
	} else if busy {
		pipText = "PREPARING"
		pipR, pipG, pipB = 240, 190, 70
	} else if !interruptedAt.IsZero() && time.Since(interruptedAt) < 3*time.Second {
		pipText = "INTERRUPTED"
		pipR, pipG, pipB = 255, 0, 180
	}
	baseW := a.layoutBaseW
	if baseW <= 0 {
		baseW = width
	}
	// User-approved G18G framed panel: transparent camo NEVER passes through
	// the diagnostic workspace. The frame scales horizontally with the same
	// G6 registration rules; monkey glyphs keep their readable 43px size.
	px, pw := a.registerX(baseW-388, width), a.registerWidth(368, width)
	light := strings.EqualFold(a.settings.Theme, "Light")
	cr, cg, cb := byte(11), byte(17), byte(18)
	if light {
		cr, cg, cb = 248, 247, 242
	}
	panelY := 112 // raised: entirely above 206px Settings cards and 258px Operations cards
	panel := RECT{int32(px), int32(panelY), int32(px + pw), int32(panelY + 87)}
	fillColor(hdc, panel, cr, cg, cb)
	frameColor(hdc, panel, 197, 160, 91)
	fillColor(hdc, RECT{int32(px), int32(panelY), int32(px + pw), int32(panelY + 3)}, 197, 160, 91)
	mid := px + pw/2
	fillColor(hdc, RECT{int32(mid), int32(panelY + 10), int32(mid + 1), int32(panelY + 78)}, 197, 160, 91)
	a.drawMonkeyBadge(hdc, px+9, panelY+11, "🙉", "VOSK", voxText, voxActive, voxR, voxG, voxB)
	a.drawMonkeyBadge(hdc, mid+9, panelY+11, "🙊", "PIPER", pipText, pipActive, pipR, pipG, pipB)
	// Compact actual emoji flags; leave the registered badge geometry untouched.
	// Selected model and voice are independent and may use different regions.
	a.drawBadgeFlag(hdc, px+9, panelY+11, flagForSelectedVosk(a.settings.VoiceName))
	a.drawBadgeFlag(hdc, mid+9, panelY+11, flagForSelectedPiper(a.settings.ActualVoice))
}
func (a *App) drawMonkeyBadge(hdc syscall.Handle, x, y int, monkey, label, state string, active bool, r, g, b byte) {
	// Light panels wash out the original pale emoji. A small charcoal backing
	// behind the glyph only keeps Dark Theme and all other cards unchanged.
	if strings.EqualFold(a.settings.Theme, "Light") {
		glyph := RECT{int32(x), int32(y), int32(x + 56), int32(y + 57)}
		fillColor(hdc, glyph, 26, 31, 32)
		frameColor(hdc, glyph, 197, 160, 91)
	}
	procSetBkMode.Call(uintptr(hdc), TRANSPARENT)
	write := func(msg string, font syscall.Handle, rc RECT, rr, gg, bb byte) {
		textBuf, _ := syscall.UTF16FromString(msg)
		old, _, _ := procSelectObject.Call(uintptr(hdc), uintptr(font))
		procSetTextColor.Call(uintptr(hdc), rgb(rr, gg, bb))
		procDrawTextW.Call(uintptr(hdc), uintptr(unsafe.Pointer(&textBuf[0])),
			uintptr(len(textBuf)-1), uintptr(unsafe.Pointer(&rc)), DT_SINGLELINE|DT_CENTER|DT_VCENTER)
		procSelectObject.Call(uintptr(hdc), old)
	}
	write(monkey, a.fontEmoji, RECT{int32(x), int32(y), int32(x + 53), int32(y + 51)}, 242, 221, 180)
	if !active {
		// Overlay, not a text symbol appended beside the monkey.
		write("🚫", a.fontNoEntry, RECT{int32(x + 19), int32(y + 18), int32(x + 56), int32(y + 55)}, 230, 55, 55)
	}
	lr, lg, lb := byte(245), byte(228), byte(193)
	if strings.EqualFold(a.settings.Theme, "Light") {
		lr, lg, lb = 29, 34, 36
	}
	write(label, a.fontButton, RECT{int32(x + 57), int32(y + 3), int32(x + 133), int32(y + 27)}, lr, lg, lb)
	write(state, a.fontSmall, RECT{int32(x + 57), int32(y + 29), int32(x + 166), int32(y + 50)}, r, g, b)
}

func (a *App) drawButton(di *DRAWITEMSTRUCT) {
	if di == nil || di.CtlType != ODT_BUTTON {
		return
	}
	id := int(di.CtlID)
	if a.drawManualIconButton(di, id) {
		return
	}
	key := a.teamKey()
	tr, tg, tb := teamColor(key)
	br, bg, bb := byte(205), byte(174), byte(111)
	fr, fg, fb := byte(238), byte(217), byte(165)
	txR, txG, txB := byte(18), byte(20), byte(18)
	isNav := id == ID_NAV_OPS || id == ID_NAV_SETTINGS || id == ID_NAV_TRAINING || id == ID_NAV_MANUAL
	active := (id == ID_NAV_OPS && a.currentScreen == "operations") || (id == ID_NAV_SETTINGS && a.currentScreen == "settings") || (id == ID_NAV_TRAINING && a.currentScreen == "training") || (id == ID_NAV_MANUAL && a.currentScreen == "manual")
	if a.currentScreen == "manual" && ((id == ID_MANUAL_QUICK && a.manualPage == "quick") || (id == ID_MANUAL_CUES && a.manualPage == "cues") || (id == ID_MANUAL_VOICES && a.manualPage == "voices") || (id == ID_MANUAL_NARRATED && a.manualPage == "narrated") || (id == ID_MANUAL_MODELS && a.manualPage == "models") || (id == ID_MANUAL_LIBRARIES && a.manualPage == "libraries")) {
		br, bg, bb = 38, 120, 61
		fr, fg, fb = 105, 205, 116
		txR, txG, txB = 255, 255, 255
	}
	if isNav {
		br, bg, bb = 18, 25, 25
		fr, fg, fb = 70, 82, 78
		txR, txG, txB = 238, 238, 232
		if active {
			br, bg, bb = tr, tg, tb
			fr, fg, fb = 225, 225, 220
		}
	}
	if id == ID_BUY_ME_A_COFFEE {
		// High-contrast gold support button; no donation required to use RAWDOGS.
		br, bg, bb = 227, 178, 62
		fr, fg, fb = 255, 222, 115
		txR, txG, txB = 28, 25, 15
	}
	if id == ID_SETTINGS_LISTEN || id == ID_TRAIN_LISTEN || id == ID_OP_LISTEN || id == ID_OP_CONFIRM || id == ID_SAVE_SETUP {
		br, bg, bb = 45, 174, 79
		fr, fg, fb = 111, 240, 137
		txR, txG, txB = 5, 24, 10
	}
	if id == ID_SETTINGS_STOP || id == ID_TRAIN_STOP || id == ID_OP_STOP || id == ID_OP_CANCEL || id == ID_OP_CLEAR_ALL || id == ID_OP_CLEAR_ACTUAL || id == ID_PIPER_STOP || (id == ID_PIPER_MUTE && a.settings.PiperMuted) {
		br, bg, bb = 151, 48, 48
		fr, fg, fb = 233, 105, 105
		txR, txG, txB = 255, 245, 245
	}
	// G18 traffic-light semantics for EXISTING buttons only; positions unchanged.
	needsGuide := a.firstRun
	if !needsGuide && a.guideMarker != "" && !a.guideActive {
		_, err := os.Stat(a.guideMarker)
		needsGuide = err != nil
	}
	if (id == ID_GUIDE_NEXT && (needsGuide || a.guideActive)) ||
		(id == ID_OP_NEW_TARGET && a.currentScreen == "operations" && a.actualSet && !a.pendingActive()) {
		br, bg, bb = 190, 0, 135 // steady magenta — expected action
		fr, fg, fb = 255, 100, 210
		txR, txG, txB = 255, 255, 255
	}
	if id == ID_PIPER_MUTE && !a.settings.PiperMuted {
		br, bg, bb = 45, 174, 79
		fr, fg, fb = 111, 240, 137
		txR, txG, txB = 5, 24, 10
	}
	if id == ID_OP_CONFIRM && (!a.pendingActive() ||
		((a.pending.Kind == "actual" || a.pending.Kind == "switch_actual" || a.pending.Kind == "target") &&
			(a.pending.X == "" || a.pending.Y == ""))) {
		br, bg, bb = 45, 52, 51 // neutral until both X and Y exist
		fr, fg, fb = 102, 114, 107
		txR, txG, txB = 234, 230, 220
	}
	if id >= ID_CLEAR_TARGET_1 && id <= ID_CLEAR_TARGET_Y {
		br, bg, bb = 24, 19, 20
		fr, fg, fb = 225, 50, 50
		txR, txG, txB = 255, 65, 65
	}
	if id == ID_OP_UNDO_CLEAR {
		br, bg, bb = 73, 66, 46
		fr, fg, fb = 215, 178, 100
		txR, txG, txB = 250, 235, 200
	}
	if di.ItemState&ODS_SELECTED != 0 {
		br = byte(int(br) * 85 / 100)
		bg = byte(int(bg) * 85 / 100)
		bb = byte(int(bb) * 85 / 100)
	}
	fillColor(di.HDC, di.RcItem, br, bg, bb)
	frameColor(di.HDC, di.RcItem, fr, fg, fb)
	if di.ItemState&ODS_DISABLED != 0 {
		txR, txG, txB = 130, 130, 130
	}
	procSetBkMode.Call(uintptr(di.HDC), TRANSPARENT)
	procSetTextColor.Call(uintptr(di.HDC), rgb(txR, txG, txB))
	buttonFont := a.fontButton
	buttonTextFlags := uintptr(DT_CENTER | DT_VCENTER | DT_SINGLELINE | DT_END_ELLIPSIS)
	if id == ID_BUY_ME_A_COFFEE {
		buttonFont = a.fontButton
		buttonTextFlags = DT_CENTER | DT_VCENTER | DT_SINGLELINE
	}
	old, _, _ := procSelectObject.Call(uintptr(di.HDC), uintptr(buttonFont))
	defer procSelectObject.Call(uintptr(di.HDC), old)
	ln, _, _ := procGetWindowTextLengthW.Call(uintptr(di.HwndItem))
	buf := make([]uint16, int(ln)+1)
	procGetWindowTextW.Call(uintptr(di.HwndItem), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	r := di.RcItem
	procDrawTextW.Call(uintptr(di.HDC), uintptr(unsafe.Pointer(&buf[0])), uintptr(int32(ln)), uintptr(unsafe.Pointer(&r)), buttonTextFlags)
}

func (a *App) drawListItem(di *DRAWITEMSTRUCT) {
	if di == nil || di.CtlType != ODT_LISTBOX || di.ItemID == 0xffffffff {
		return
	}
	light := strings.EqualFold(a.settings.Theme, "Light")
	if di.CtlID == ID_LIBRARY_CHOICES && di.ItemState&ODS_SELECTED != 0 {
		fillColor(di.HDC, di.RcItem, 125, 0, 103) // steady magenta, selected pack
		frameColor(di.HDC, di.RcItem, 255, 102, 207)
		procSetTextColor.Call(uintptr(di.HDC), rgb(255, 255, 255))
	} else if di.CtlID == ID_ADAPT_RULE_LIST && di.ItemState&ODS_SELECTED != 0 {
		fillColor(di.HDC, di.RcItem, 61, 45, 84)
		procSetTextColor.Call(uintptr(di.HDC), rgb(255, 245, 255))
	} else if light {
		fillColor(di.HDC, di.RcItem, 255, 255, 255)
		procSetTextColor.Call(uintptr(di.HDC), rgb(20, 20, 20))
	} else {
		fillColor(di.HDC, di.RcItem, 12, 17, 19)
		procSetTextColor.Call(uintptr(di.HDC), rgb(245, 228, 193))
	}
	procSetBkMode.Call(uintptr(di.HDC), TRANSPARENT)
	old, _, _ := procSelectObject.Call(uintptr(di.HDC), uintptr(a.fontNormal))
	defer procSelectObject.Call(uintptr(di.HDC), old)
	ln, _, _ := procSendMessageW.Call(uintptr(di.HwndItem), LB_GETTEXTLEN, uintptr(di.ItemID), 0)
	if int(ln) < 0 || int(ln) > 4096 {
		return
	}
	buf := make([]uint16, int(ln)+1)
	procSendMessageW.Call(uintptr(di.HwndItem), LB_GETTEXT, uintptr(di.ItemID), uintptr(unsafe.Pointer(&buf[0])))
	text := utf16ToString(buf)
	lower := strings.ToLower(text)
	pink := strings.HasPrefix(text, "PINK:") || strings.HasPrefix(text, "CMD:") || strings.Contains(text, "  CMD:") || strings.Contains(text, "VALID RAWDOGS CUE") || strings.Contains(lower, "accepted — x") || strings.Contains(lower, "accepted — y")
	green := strings.HasPrefix(text, "GREEN:") || strings.HasPrefix(text, "CONFIDENCE PASS:") || strings.Contains(text, "CONFIRMED —") || strings.Contains(text, "SAVED") || strings.HasPrefix(text, "TRAINING ACCEPTED:") || strings.HasPrefix(text, "TRAINING CONFIRMED:") || (di.CtlID == ID_ADAPT_RULE_LIST && strings.Contains(text, " | ACTIVE | "))
	red := strings.HasPrefix(text, "RED:") || strings.HasPrefix(text, "REJECTED:") || strings.Contains(text, "LISTENER ERROR") || strings.Contains(text, "ERROR —") || strings.HasPrefix(text, "TRAINING LOW CONFIDENCE:")
	dim := strings.HasPrefix(text, "NEUTRAL:") || strings.Contains(text, "NOT A RAWDOGS COMMAND")
	amber := strings.HasPrefix(text, "TRAINING RETRY:") || strings.HasPrefix(text, "TRAINING HEARD:") || (di.CtlID == ID_ADAPT_RULE_LIST && strings.Contains(text, " | PENDING "))
	pink = pink || strings.HasPrefix(text, "TRAINING REVIEW:")
	switch {
	case di.CtlID == ID_LIBRARY_CHOICES && di.ItemState&ODS_SELECTED != 0:
		// Selected preview/install target remains magenta even if already installed.
	case di.CtlID == ID_LIBRARY_CHOICES && strings.Contains(text, "✓ INSTALLED"):
		procSetTextColor.Call(uintptr(di.HDC), rgb(60, 214, 115))
	case pink:
		if light {
			procSetTextColor.Call(uintptr(di.HDC), rgb(190, 0, 125))
		} else {
			procSetTextColor.Call(uintptr(di.HDC), rgb(255, 0, 180))
		}
	case green:
		if light {
			procSetTextColor.Call(uintptr(di.HDC), rgb(0, 125, 55))
		} else {
			procSetTextColor.Call(uintptr(di.HDC), rgb(70, 210, 105))
		}
	case red:
		if light {
			procSetTextColor.Call(uintptr(di.HDC), rgb(180, 40, 40))
		} else {
			procSetTextColor.Call(uintptr(di.HDC), rgb(230, 80, 80))
		}
	case amber:
		if light {
			procSetTextColor.Call(uintptr(di.HDC), rgb(150, 95, 0))
		} else {
			procSetTextColor.Call(uintptr(di.HDC), rgb(240, 190, 70))
		}
	case dim:
		procSetTextColor.Call(uintptr(di.HDC), rgb(155, 155, 150))
	}
	r := di.RcItem
	r.Left += 8
	r.Right -= 4
	procDrawTextW.Call(uintptr(di.HDC), uintptr(unsafe.Pointer(&buf[0])), uintptr(int32(ln)), uintptr(unsafe.Pointer(&r)), DT_VCENTER|DT_SINGLELINE|DT_END_ELLIPSIS)
}

func loaderWndProc(hwnd syscall.Handle, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_ERASEBKGND:
		// G13 provisioning loader paints the entire child window in WM_PAINT; suppress class-background flicker.
		return 1
	case WM_PAINT:
		var ps PAINTSTRUCT
		hdc, _, _ := procBeginPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))
		if app != nil {
			var rc RECT
			procGetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rc)))
			app.drawCommsLoader(syscall.Handle(hdc), int(rc.Right-rc.Left), int(rc.Bottom-rc.Top))
		}
		procEndPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
	return r
}

func wndProc(hwnd syscall.Handle, msg uint32, wParam, lParam uintptr) uintptr {
	a := app
	switch msg {
	case WM_ERASEBKGND:
		if a != nil {
			var rc RECT
			procGetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rc)))
			br := a.darkBrush
			if strings.EqualFold(a.settings.Theme, "Light") {
				br = a.lightBrush
			}
			procFillRect.Call(wParam, uintptr(unsafe.Pointer(&rc)), uintptr(br))
			a.drawTacticalChrome(syscall.Handle(wParam), int(rc.Right-rc.Left), int(rc.Bottom-rc.Top))
			return 1
		}
	case WM_PAINT:
		// Child STATICs repaint on content change; the shared Vosk/Piper
		// badges can update without erasing the whole parent/camo texture.
		var ps PAINTSTRUCT
		hdc, _, _ := procBeginPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))
		if a != nil && (a.currentScreen == "operations" || a.currentScreen == "settings" || a.currentScreen == "training" || a.currentScreen == "manual") {
			var rc RECT
			procGetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rc)))
			a.drawVoiceBadges(syscall.Handle(hdc), int(rc.Right-rc.Left))
		}
		procEndPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))
		return 0
	case WM_DRAWITEM:
		if a != nil && lParam != 0 {
			di := (*DRAWITEMSTRUCT)(unsafe.Pointer(lParam))
			if di.CtlType == ODT_LISTBOX {
				a.drawListItem(di)
			} else {
				a.drawButton(di)
			}
			return 1
		}
	case 0x0114: // WM_HSCROLL — actual audio position follows slider release.
		if a != nil && syscall.Handle(lParam) == a.controls[ID_MANUAL_SEEK] {
			code := wParam & 0xffff
			if code == 5 {
				a.manualSeekDragging = true
			} // TB_THUMBTRACK
			if code == 8 || code == 4 { // TB_ENDTRACK / TB_THUMBPOSITION
				a.manualSeekDragging = false
				pos, _, _ := procSendMessageW.Call(uintptr(lParam), 0x0400, 0, 0)
				if a.tourActive {
					index := int(pos) * len(guidedTour) / 10000
					if index >= len(guidedTour) {
						index = len(guidedTour) - 1
					}
					a.tourGoTo(index, !a.tourHold)
				} else {
					a.narratorSeek(int(pos))
				}
			}
		}
		return 0
	case WM_TIMER:
		if wParam == 80 && a != nil && (a.currentScreen == "manual" || a.tourActive) {
			a.narratorTickUI() // Reads real WAV position; avoids panel reconstruction/flicker.
			return 0
		}
		if wParam == 77 && app != nil && app.guideActive {
			// Ignore stale G13 pulse timer messages. Guide highlight is steady.
			app.guidePulse = true
			procKillTimer.Call(uintptr(app.hwnd), 77)
			procInvalidateRect.Call(uintptr(app.hwnd), 0, 0)
			return 0
		}
		if wParam == 78 && app != nil && app.loaderActive {
			app.tickCommsLoader()
			return 0
		}
		if wParam == 79 && a != nil {
			procKillTimer.Call(uintptr(hwnd), 79)
			return 0
		}

	case WM_SIZE:
		if a != nil && wParam != 1 { // SIZE_MINIMIZED == 1
			a.relayoutRegisteredControls()
			a.layoutLoaderOverlay()
			procInvalidateRect.Call(uintptr(hwnd), 0, 1)
		}
		return 0
	case WM_CLOSE:
		if a != nil {
			a.hideCommsLoader()
			a.voiceTrain.Active = false
			a.stopRecognizer()
			a.stopMicTest()
			if a.healthStop != nil {
				select {
				case <-a.healthStop:
				default:
					close(a.healthStop)
				}
			}
			if a.tourActive {
				a.endTour()
			}
			a.narratorStop(false)
			a.saveSettings()
		}
		procDestroyWindow(hwnd)
		return 0
	case WM_DESTROY:
		// Stop only the narrator UI clock. First-run provisioning owns timer 78.
		procKillTimer.Call(uintptr(hwnd), 80)
		procPostQuitMessage.Call(0)
		return 0
	case WM_COMMAND:
		if a == nil {
			return 0
		}
		id := int(wParam & 0xffff)
		code := int((wParam >> 16) & 0xffff)
		if a.isCoordinateID(id) && code == EN_CHANGE {
			if !a.keyboardProgrammatic {
				a.writeInteractionDiagnostic("KEYBOARD_EDIT", fmt.Sprintf("field=%d; value=%q", id, windowText(a.controls[id])), "EDITED", a.voiceDiagnosticState(), a.voiceDiagnosticState())
				a.keyboardDirtyID = id
				a.keyboardErrorID = 0
				// Existing field changes colour without relocating or resetting it.
				a.setControlClass(id, "pink")
			}
		} else if a.isCoordinateID(id) && (code == EN_SETFOCUS || code == EN_KILLFOCUS) {
			before := a.voiceDiagnosticState()
			a.coordinateFocusChanged(id, code)
			a.writeInteractionDiagnostic("MOUSE_OR_FOCUS", fmt.Sprintf("coordinate=%d; focus_event=%d; focus=%d", id, code, a.keyboardFocusID), "OBSERVED", before, a.voiceDiagnosticState())
		} else if code == BN_CLICKED || code == CBN_SELCHANGE {
			before := a.voiceDiagnosticState()
			a.handleCommand(id, code)
			if a.currentScreen == "operations" {
				a.writeInteractionDiagnostic("MOUSE_CONTROL", fmt.Sprintf("control=%d; notification=%d", id, code), "OBSERVED", before, a.voiceDiagnosticState())
			}
		}
		return 0
	case WM_APP_SCAN_DONE:
		if a != nil {
			a.applyAudioScan()
		}
		return 0
	case WM_APP_MIC_LEVEL:
		if a != nil {
			a.applyMicLevel()
		}
		return 0
	case WM_APP_VOICE_DONE:
		if a != nil {
			a.applyVoiceLanguage()
		}
		return 0
	case WM_APP_LIBRARY_DONE:
		if a != nil {
			a.applyLibraryCatalogResult()
		}
		return 0
	case WM_APP_LIBRARY_PREVIEW:
		if a != nil {
			a.applyLibraryPreviewResult()
		}
		return 0
	case WM_APP_LIBRARY_STAGE:
		if a != nil {
			a.applyLibraryInstallProgress()
		}
		return 0
	case WM_APP_RECO_EVENT:
		if a != nil {
			a.applyRecoEvents()
		}
		return 0
	case WM_APP_HEALTH:
		if a != nil {
			a.applyHealth()
		}
		return 0
	case WM_APP_VOSK_DONE:
		if a != nil {
			a.applyVoskInstallResult()
		}
		return 0
	case WM_APP_PIPER_DONE:
		if a != nil {
			a.applyPiperInstallResult()
		}
		return 0
	case WM_APP_TTS_DONE:
		if a != nil {
			a.applyTtsDone(uint64(wParam))
		}
		return 0
	case WM_APP_PROVISION_STAGE:
		if a != nil && a.loaderHwnd != 0 {
			procInvalidateRect.Call(uintptr(a.loaderHwnd), 0, 1)
		}
		return 0
	case WM_APP_PROVISION_DONE:
		if a != nil {
			a.provisionDone = true
			a.hideCommsLoader()
			a.voices, a.languages = a.queryVoiceLanguage()
			a.applyVoiceLanguage()
			a.refreshInstalledLibrarySelectors()
			if spec := a.chooseProvisionVosk(); a.voskReady() {
				a.settings.VoiceName = spec.ActiveDisplay
				a.settings.Language = spec.Language
				a.selectComboText(ID_VOICE_COMBO, spec.ActiveDisplay)
				a.selectComboText(ID_LANG_COMBO, spec.Language)
				a.selectComboText(ID_VOSK_LIBRARY_COMBO, spec.LibraryLabel)
			}
			if spec := a.chooseProvisionPiper(); a.piperRuntimeReady() && a.piperVoiceInstalled(spec) {
				a.settings.ActualVoice = spec.Key
				a.selectComboText(ID_ACTUAL_LIBRARY_COMBO, spec.Label)
			}
			a.updateVoskLibraryStatus()
			a.updateActualVoiceLibraryStatus()
			if a.provisionSucceeded {
				setText(a.controls[3004], "DEFAULT VOSK + PIPER PACKS ONLINE — PREPARING GUIDED SETUP.")
				if a.audioReady {
					if _, err := os.Stat(a.guideMarker); err != nil {
						a.startGuidedSetup()
					}
				}
			} else {
				setText(a.controls[3004], "AUTOMATIC DEFAULT PACK SETUP FAILED — USE THE VOSK / ACTUAL LIBRARY CONTROLS, THEN START GUIDE AGAIN.")
			}
		}
		return 0
	case WM_CTLCOLORSTATIC, WM_CTLCOLOREDIT, WM_CTLCOLORLISTBOX:
		if a != nil {
			return a.handleColor(msg, wParam, lParam)
		}
	}
	r, _, _ := procDefWindowProcW.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
	return r
}

func procDestroyWindow(hwnd syscall.Handle) {
	p := user32.NewProc("DestroyWindow")
	p.Call(uintptr(hwnd))
}

func (a *App) handleColor(msg uint32, wParam, lParam uintptr) uintptr {
	hdc := wParam
	// Voice Training's spoken exercise and the raw-vs-interpreted readback
	// use semantic colors. This touches no Operations, Settings, Field Manual
	// or the protected top area of Voice Training.
	if msg == WM_CTLCOLORSTATIC && a.currentScreen == "training" {
		id := 0
		for _, candidate := range []int{ID_ADAPT_PROMPT_LABEL, ID_ADAPT_EXPECTED, ID_ADAPT_HEARD, ID_ADAPT_INTERPRETATION, ID_ADAPT_STATUS, ID_ADAPT_LEGEND} {
			if h := a.controls[candidate]; h != 0 && lParam == uintptr(h) {
				id = candidate
				break
			}
		}
		if id != 0 {
			light := strings.EqualFold(a.settings.Theme, "Light")
			class := "neutral"
			switch id {
			case ID_ADAPT_PROMPT_LABEL, ID_ADAPT_EXPECTED:
				class = "gold"
			case ID_ADAPT_HEARD:
				// Confidence does NOT determine success; RAW stays amber until the
				// exercise has been compared against the requested phrase.
				if a.voiceTrain.LastRaw != "" {
					class = "amber"
					if a.voiceTrain.FeedbackClass == "green" || a.voiceTrain.FeedbackClass == "red" {
						class = a.voiceTrain.FeedbackClass
					}
				}
			case ID_ADAPT_INTERPRETATION, ID_ADAPT_STATUS:
				class = a.voiceTrain.FeedbackClass
			}
			r, g, b := byte(245), byte(228), byte(193)
			if light {
				r, g, b = 20, 20, 20
			}
			switch class {
			case "gold":
				if light {
					r, g, b = 135, 80, 0
				} else {
					r, g, b = 255, 215, 115
				}
			case "green":
				if light {
					r, g, b = 0, 125, 55
				} else {
					r, g, b = 70, 210, 105
				}
			case "pink":
				if light {
					r, g, b = 190, 0, 125
				} else {
					r, g, b = 255, 0, 180
				}
			case "amber":
				if light {
					r, g, b = 150, 95, 0
				} else {
					r, g, b = 240, 190, 70
				}
			case "red":
				if light {
					r, g, b = 180, 40, 40
				} else {
					r, g, b = 230, 80, 80
				}
			}
			procSetTextColor.Call(hdc, rgb(r, g, b))
			procSetBkMode.Call(hdc, TRANSPARENT)
			if light {
				return uintptr(a.lightBrush)
			}
			return uintptr(a.darkBrush)
		}
	}
	if msg == WM_CTLCOLORSTATIC && a.currentScreen == "manual" && a.manualHighlighted != 0 && syscall.Handle(lParam) == a.manualHighlighted {
		procSetTextColor.Call(hdc, rgb(28, 28, 24))
		procSetBkColor.Call(hdc, rgb(255, 247, 169))
		procSetBkMode.Call(hdc, OPAQUE)
		return uintptr(a.manualHighlightBrush)
	}
	if (msg == WM_CTLCOLORSTATIC || msg == WM_CTLCOLOREDIT) && a.manualPaper != nil && a.manualPaper[syscall.Handle(lParam)] {
		procSetTextColor.Call(hdc, rgb(28, 28, 24))
		procSetBkColor.Call(hdc, rgb(239, 226, 190))
		procSetBkMode.Call(hdc, OPAQUE)
		return uintptr(a.paperBrush)
	}
	if msg == WM_CTLCOLORSTATIC && a.labelPlate != nil && a.labelPlate[syscall.Handle(lParam)] {
		light := strings.EqualFold(a.settings.Theme, "Light")
		if light {
			procSetTextColor.Call(hdc, rgb(20, 20, 20))
			procSetBkColor.Call(hdc, rgb(220, 220, 212))
			procSetBkMode.Call(hdc, OPAQUE)
			return uintptr(a.lightLabelBrush)
		}
		procSetTextColor.Call(hdc, rgb(245, 228, 193))
		procSetBkColor.Call(hdc, rgb(15, 21, 23))
		procSetBkMode.Call(hdc, OPAQUE)
		return uintptr(a.darkLabelBrush)
	}
	if msg == WM_CTLCOLORSTATIC && a.controls[4105] != 0 && lParam == uintptr(a.controls[4105]) {
		switch a.lastRecoClass {
		case "pink":
			procSetTextColor.Call(hdc, rgb(255, 0, 180))
		case "green":
			procSetTextColor.Call(hdc, rgb(70, 210, 105))
		case "amber":
			procSetTextColor.Call(hdc, rgb(240, 190, 70))
		case "red":
			procSetTextColor.Call(hdc, rgb(230, 80, 80))
		default:
			procSetTextColor.Call(hdc, rgb(245, 228, 193))
		}
		procSetBkMode.Call(hdc, TRANSPARENT)
		if strings.EqualFold(a.settings.Theme, "Light") {
			return uintptr(a.lightBrush)
		}
		return uintptr(a.darkBrush)
	}
	if (msg == WM_CTLCOLORSTATIC || msg == WM_CTLCOLOREDIT) && a.textClass != nil {
		if cls := a.textClass[syscall.Handle(lParam)]; cls != "" {
			light := strings.EqualFold(a.settings.Theme, "Light")
			switch cls {
			case "pink":
				if light {
					procSetTextColor.Call(hdc, rgb(190, 0, 125))
				} else {
					procSetTextColor.Call(hdc, rgb(255, 0, 180))
				}
			case "green":
				if light {
					procSetTextColor.Call(hdc, rgb(0, 125, 55))
				} else {
					procSetTextColor.Call(hdc, rgb(70, 210, 105))
				}
			case "red":
				if light {
					procSetTextColor.Call(hdc, rgb(180, 40, 40))
				} else {
					procSetTextColor.Call(hdc, rgb(230, 80, 80))
				}
			case "dim":
				procSetTextColor.Call(hdc, rgb(150, 150, 145))
			}
			if msg == WM_CTLCOLOREDIT {
				procSetBkColor.Call(hdc, rgb(22, 27, 29))
				if light {
					procSetBkColor.Call(hdc, rgb(239, 239, 233))
					return uintptr(a.lightEditBrush)
				}
				return uintptr(a.darkEditBrush)
			}
			procSetBkMode.Call(hdc, TRANSPARENT)
			if light {
				return uintptr(a.lightBrush)
			}
			return uintptr(a.darkBrush)
		}
	}
	light := strings.EqualFold(a.settings.Theme, "Light")
	if light {
		procSetTextColor.Call(hdc, rgb(20, 20, 20))
		procSetBkColor.Call(hdc, rgb(239, 239, 233))
		if msg == WM_CTLCOLOREDIT || msg == WM_CTLCOLORLISTBOX {
			procSetBkMode.Call(hdc, OPAQUE)
			return uintptr(a.lightEditBrush)
		}
		procSetBkMode.Call(hdc, TRANSPARENT)
		return uintptr(a.lightBrush)
	}
	procSetTextColor.Call(hdc, rgb(245, 228, 193))
	procSetBkColor.Call(hdc, rgb(22, 27, 29))
	if msg == WM_CTLCOLOREDIT || msg == WM_CTLCOLORLISTBOX {
		procSetBkMode.Call(hdc, OPAQUE)
		return uintptr(a.darkEditBrush)
	}
	procSetBkMode.Call(hdc, TRANSPARENT)
	return uintptr(a.darkBrush)
}

func (a *App) handleCommand(id, code int) {
	switch id {
	case ID_BUY_ME_A_COFFEE:
		a.openBuyMeACoffee()
	case ID_NAV_OPS:
		a.showScreen("operations")
	case ID_NAV_SETTINGS:
		a.showScreen("settings")
	case ID_NAV_TRAINING:
		a.showScreen("training")
	case ID_NAV_MANUAL:
		a.showScreen("manual")
	case ID_VOICE_LIBRARY:
		a.openVoiceLibrary()
	case ID_GUIDE_NEXT:
		if a.tourActive {
			a.tourNext()
		} else if a.guideActive {
			setText(a.controls[3004], "GUIDE ADVANCES AUTOMATICALLY WHEN THE HIGHLIGHTED TASK IS COMPLETE.")
		} else if !a.actualVoiceReady() {
			// The guide teaches Vosk installation AFTER the operator hears
			// Piper. Missing recognition must not prevent starting setup.
			a.beginStartupProvisioning()
		} else {
			a.startGuidedSetup()
		}
	case ID_GUIDE_END:
		if a.tourActive {
			a.endTour()
		} else {
			a.finishGuidedSetup(false)
		}
	case ID_TOUR_PREV:
		a.tourPrev()
	case ID_TOUR_PLAY:
		a.tourPlay()
	case ID_TOUR_PAUSE:
		a.tourPause()
	case ID_TOUR_STOP:
		a.tourStop()
	case ID_TOUR_NEXT:
		a.tourNext()
	case ID_TOUR_RESTART:
		a.tourRestart()
	case ID_TOUR_INDEX:
		if code == CBN_SELCHANGE {
			pos, _ := a.comboSelection(ID_TOUR_INDEX)
			if pos >= 0 && pos < len(guidedTour) {
				if a.tourActive {
					a.tourGoTo(pos, !a.tourHold)
				} else {
					a.narratorJump(pos, true)
					a.selectGuideIndex(pos)
				}
			}
		}
	case ID_MANUAL_QUICK:
		a.setManualPage("quick")
	case ID_MANUAL_CUES:
		a.setManualPage("cues")
	case ID_MANUAL_VOICES:
		a.setManualPage("voices")
	case ID_MANUAL_NARRATED:
		a.setManualPage("narrated")
	case ID_MANUAL_MODELS:
		a.setManualPage("models")
	case ID_MANUAL_LIBRARIES:
		a.setManualPage("libraries")
	case ID_MANUAL_LIB_SETTINGS:
		a.narratorStop(true)
		a.showScreen("settings")
	case ID_LIBRARY_COUNTRY:
		if code == CBN_SELCHANGE {
			a.refreshLibraryBrowserList()
		}
	case ID_LIBRARY_VOSK_MODE:
		a.libraryMode = "vosk"
		a.refreshLibraryBrowserList()
	case ID_LIBRARY_PIPER_MODE:
		a.libraryMode = "piper"
		a.refreshLibraryBrowserList()
	case ID_LIBRARY_CHOICES:
		if code == CBN_SELCHANGE {
			a.updateLibraryBrowserDetails()
		}
	case ID_LIBRARY_REFRESH:
		a.refreshLibraryCatalogAsync(true)
	case ID_LIBRARY_INSTALL:
		a.installLibraryBrowserSelection()
	case ID_LIBRARY_PREVIEW:
		a.previewLibraryBrowserSelection()
	case ID_LIBRARY_SETTINGS:
		a.narratorStop(true)
		a.showScreen("settings")
	case ID_LIBRARY_REMOVE:
		a.removeLibraryBrowserSelection()
	case ID_MANUAL_LIB_VOSK_SOURCE:
		a.openOfficialLibraryPage(voskCatalogURL)
	case ID_MANUAL_LIB_PIPER_SOURCE:
		a.openOfficialLibraryPage(piperCatalogURL)
	case ID_MANUAL_MUTE:
		a.narratorToggleMute()
	case ID_MANUAL_READ:
		if a.tourActive {
			a.tourPlay()
		} else {
			a.startNarratedManual()
		}
	case ID_MANUAL_PAUSE:
		if a.tourActive {
			a.tourPause()
		} else {
			a.narratorPause()
		}
	case ID_MANUAL_STOP:
		if a.tourActive {
			a.tourStop()
		} else {
			a.narratorStop(true)
		}
		a.syncNarratorSeek()
	case ID_MANUAL_RESTART:
		if a.tourActive {
			a.tourRestart()
		} else {
			a.narratorRestart()
		}
		a.syncNarratorSeek()
	case ID_MANUAL_PREV:
		if a.tourActive {
			a.tourPrev()
		} else {
			a.narratorJump(-1, false)
		}
		a.syncNarratorSeek()
	case ID_MANUAL_NEXT:
		if a.tourActive {
			a.tourNext()
		} else {
			a.narratorJump(1, false)
		}
		a.syncNarratorSeek()
	case ID_MANUAL_END:
		a.endNarratedManual()
		if a.guideActive && a.guideStep == 6 {
			a.finishGuidedSetup(false)
		}
	case ID_PERMISSION_RUN:
		a.settings.Permission = "this_run"
		setText(a.controls[3004], "SYSTEM STATUS: RAWDOGS MIC CONSENT — THIS RUN")
		procInvalidateRect.Call(uintptr(a.hwnd), 0, 1)
		if a.guideActive && a.guideStep == 0 {
			a.advanceGuideAutomatic()
		}
	case ID_PERMISSION_ALWAYS:
		a.settings.Permission = "always"
		setText(a.controls[3004], "SYSTEM STATUS: RAWDOGS MIC CONSENT — ALWAYS ALLOW")
		procInvalidateRect.Call(uintptr(a.hwnd), 0, 1)
		if a.guideActive && a.guideStep == 0 {
			a.advanceGuideAutomatic()
		}
	case ID_PERMISSION_NOTNOW:
		a.settings.Permission = "not_now"
		a.stopMicTest()
		if a.guideActive {
			a.finishGuidedSetup(false)
		}
		setText(a.controls[3004], "VOICE SETUP DEFERRED — KEYBOARD / MOUSE OPERATION REMAINS AVAILABLE. START GUIDE WHEN YOU WANT VOICE.")
	case ID_OPEN_PRIVACY:
		a.openPrivacy()
	case ID_TEST_MIC:
		a.startStopMic()
	case ID_SETTINGS_LISTEN:
		if a.guideActive && a.guideStep == 2 && a.guideOutputAwaiting {
			a.guideOutputHeard = true
			a.guideOutputAwaiting = false
			setText(a.controls[3004], "ACTUAL OUTPUT CONFIRMED — CONTINUING TOUR.")
			a.advanceGuideAutomatic()
		} else {
			a.startRecognizer()
		}
	case ID_SETTINGS_STOP:
		if a.guideActive && a.guideStep == 2 && !a.guideOutputTesting {
			a.guideOutputAwaiting = false
			a.startFirstRunActualTest()
		} else {
			a.stopRecognizer()
		}
	case ID_ADAPT_START:
		a.voiceTrainingStart()
	case ID_ADAPT_STOP:
		a.voiceTrainingStop()
	case ID_ADAPT_REPEAT:
		a.voiceTrainingRepeat()
	case ID_ADAPT_CONFIRM:
		a.voiceTrainingConfirm()
	case ID_ADAPT_SKIP:
		a.voiceTrainingSkip()
	case ID_ADAPT_REFERENCE:
		a.voiceTrainingReferenceToggle()
	case ID_ADAPT_REMOVE:
		a.voiceTrainingRemoveSelected()
	case ID_ADAPT_LOAD_PROFILE:
		a.chooseVoiceProfile()
	case ID_TRAIN_LISTEN:
		a.startRecognizer()
	case ID_TRAIN_STOP:
		a.voiceTrainingStop()
		a.stopRecognizer()
	case ID_TRAIN_CLEAR:
		a.transcript = nil
		if h := a.controls[4102]; h != 0 {
			procSendMessageW.Call(uintptr(h), LB_RESETCONTENT, 0, 0)
			procSendMessageW.Call(uintptr(h), LB_ADDSTRING, 0, uintptr(unsafe.Pointer(utf16Ptr("Transcript cleared. Listener: "+a.currentListenerState()))))
		}
	case ID_OP_CLEAR_BUFFER:
		a.transcript = nil
		for _, id := range []int{5031, 5052} {
			if h := a.controls[id]; h != 0 {
				procSendMessageW.Call(uintptr(h), LB_RESETCONTENT, 0, 0)
			}
		}
		a.rotateVoiceDiagnosticLog()
		a.addTranscript("Diagnostic buffer cleared — fresh log session started")
	case ID_OP_LISTEN:
		a.startRecognizer()
	case ID_OP_STOP:
		a.stopRecognizer()
	case ID_OP_NEW_TARGET:
		a.startNewTarget()
	case ID_OP_CONFIRM:
		a.confirmPending()
	case ID_OP_CANCEL:
		a.cancelPending()
	case ID_OP_CLEAR_ALL:
		a.beginClearAll()
	case ID_OP_UNDO_CLEAR:
		a.undoClear()
	case ID_CLEAR_TARGET_1, ID_CLEAR_TARGET_2, ID_CLEAR_TARGET_3, ID_CLEAR_TARGET_4:
		a.clearOneTarget(id - ID_CLEAR_TARGET_1)
	case ID_CLEAR_UNSAVED:
		a.clearUnsavedTarget()
	case ID_CLEAR_ACTUAL_X:
		a.clearCoordinateValue("actual", "X")
	case ID_CLEAR_ACTUAL_Y:
		a.clearCoordinateValue("actual", "Y")
	case ID_CLEAR_TARGET_X:
		a.clearCoordinateValue("target", "X")
	case ID_CLEAR_TARGET_Y:
		a.clearCoordinateValue("target", "Y")
	case ID_OP_CLEAR_ACTUAL:
		a.beginClearActual()
	case ID_GAIN_COMBO:
		if code == CBN_SELCHANGE && a.micRunning {
			a.stopMicTest()
			time.Sleep(40 * time.Millisecond)
			a.startStopMic()
		}
	case ID_RESCAN_AUDIO:
		a.stopMicTest()
		go a.scanAudioAsync(true)
	case ID_REFRESH_VOICE:
		go a.refreshVoiceLanguageAsync(true)
	case ID_VOSK_INSTALL_SELECTED:
		a.installVoskAsync()
	case ID_VOSK_USE_SELECTED:
		a.useSelectedLibraryModel()
	case ID_ACTUAL_INSTALL_SELECTED:
		a.installSelectedActualVoice()
	case ID_ACTUAL_USE_SELECTED:
		a.useSelectedActualVoice()
	case ID_TEST_VOICE:
		if a.tourActive && a.tourIndex == 0 {
			a.tourOutputTested = true
		}
		a.testVoice()
	case ID_PIPER_MUTE:
		a.setPiperMute(!a.settings.PiperMuted)
	case ID_PIPER_STOP:
		a.stopPiperSpeech("OPERATOR STOP")
	case ID_SAVE_SETUP:
		a.saveSetupOnCommand()
	case ID_THEME_COMBO:
		if code == CBN_SELCHANGE {
			a.captureSettingsFromUI()
			a.applyTheme()
		}
	case ID_TEAM_COMBO:
		if code == CBN_SELCHANGE {
			a.captureSettingsFromUI()
			a.applyTheme()
		}
	case ID_VOSK_LIBRARY_COMBO:
		if code == CBN_SELCHANGE {
			a.updateVoskLibraryStatus()
		}
	case ID_ACTUAL_LIBRARY_COMBO:
		if code == CBN_SELCHANGE {
			a.updateActualVoiceLibraryStatus()
		}
	case ID_VOICE_COMBO:
		if code == CBN_SELCHANGE {
			a.captureSettingsFromUI()
			if _, model := a.comboSelection(ID_VOICE_COMBO); model != "" {
				if spec, ok := voskSpecByActiveDisplay(model); ok {
					a.syncLanguageToModel(spec)
				}
			}
			a.updateVoskLibraryStatus()
		}
	case ID_LANG_COMBO:
		if code == CBN_SELCHANGE {
			a.captureSettingsFromUI()
			if _, language := a.comboSelection(ID_LANG_COMBO); language != "" {
				a.syncModelToLanguage(language)
			}
			a.updateVoskLibraryStatus()
		}
	case ID_MIC_COMBO:
		if code == CBN_SELCHANGE {
			a.captureSettingsFromUI()
			a.micTested = false
		}
	case ID_OUTPUT_COMBO:
		if code == CBN_SELCHANGE {
			if a.tourActive && a.tourIndex == 0 {
				a.tourOutputTested = false
			}
			a.captureSettingsFromUI()
			if a.guideActive && a.guideStep == 2 {
				a.guideOutputAwaiting = false
				a.guideOutputHeard = false
				a.startFirstRunActualTest()
			}
		}
	case ID_CONF_COMBO, ID_SPEED_COMBO, ID_VOLUME_COMBO:
		if code == CBN_SELCHANGE {
			a.captureSettingsFromUI()
			if id == ID_SPEED_COMBO || id == ID_VOLUME_COMBO {
				if id == ID_VOLUME_COMBO && a.settings.VoiceVolume > 0 && a.settings.PiperMuted {
					a.settings.PiperMuted = false
					a.syncPiperMuteButton()
				}
				a.persistPiperControls()
			}
		}
	}
}

func (a *App) messageLoop() {
	var msg MSG
	for {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		// Native EDIT controls do not belong to a dialog; intercept Tab/Enter
		// only for these four fields, leaving all other UI keys unchanged.
		if a.handleCoordinateKeyLogged(&msg) {
			continue
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
}
