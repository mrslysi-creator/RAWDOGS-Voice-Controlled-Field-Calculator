//go:build windows

package main

import (
	"embed"
	"sync"
)

// Clean high-resolution blue/silver Walkman-style icons keep the existing look.
// No emoji or replacement icon font is used for the player.
//
//go:embed assets/manual_icons/*.png
var manualIconSources embed.FS

var manualIconsOnce sync.Once
var manualIcons map[int]*CamoImage

func initManualIcons() {
	manualIcons = map[int]*CamoImage{}
	refs := map[int]string{
		ID_MANUAL_PREV: "previous", ID_MANUAL_READ: "play", ID_MANUAL_PAUSE: "pause",
		ID_MANUAL_STOP: "stop", ID_MANUAL_NEXT: "next", ID_MANUAL_RESTART: "restart", ID_MANUAL_MUTE: "speaker",
		ID_TOUR_PREV: "previous", ID_TOUR_PLAY: "play", ID_TOUR_PAUSE: "pause",
		ID_TOUR_STOP: "stop", ID_TOUR_NEXT: "next", ID_TOUR_RESTART: "restart",
	}
	for id, name := range refs {
		raw, err := manualIconSources.ReadFile("assets/manual_icons/" + name + ".png")
		if err == nil {
			manualIcons[id] = decodeCamo(raw)
		}
	}
}
func (a *App) drawManualIconButton(di *DRAWITEMSTRUCT, id int) bool {
	switch id {
	case ID_MANUAL_PREV, ID_MANUAL_READ, ID_MANUAL_PAUSE, ID_MANUAL_STOP, ID_MANUAL_NEXT, ID_MANUAL_RESTART, ID_MANUAL_MUTE,
		ID_TOUR_PREV, ID_TOUR_PLAY, ID_TOUR_PAUSE, ID_TOUR_STOP, ID_TOUR_NEXT, ID_TOUR_RESTART:
	default:
		return false
	}
	manualIconsOnce.Do(initManualIcons)
	art := manualIcons[id]
	if art == nil {
		return false
	} // Not allowed to silently replace with an emoji.
	rc := di.RcItem
	width := int(rc.Right - rc.Left)
	height := int(rc.Bottom - rc.Top)
	a.drawCamoTile(di.HDC, art, int(rc.Left), int(rc.Top), width, height)
	if id == ID_MANUAL_MUTE {
		a.narratorMu.Lock()
		muted := a.manualPlayerMute
		a.narratorMu.Unlock()
		if muted {
			// Two crisp red crossing strokes on the refreshed speaker
			// icon indicate muted; icon image and button remain the same control.
			pen, _, _ := procCreatePen.Call(0, 4, rgb(219, 57, 56))
			prior, _, _ := procSelectObject.Call(uintptr(di.HDC), pen)
			left := int(rc.Left) + width/3
			right := int(rc.Left) + width*3/4
			upper := int(rc.Top) + height/4
			lower := int(rc.Top) + height*3/4
			procMoveToEx.Call(uintptr(di.HDC), uintptr(left), uintptr(upper), 0)
			procLineTo.Call(uintptr(di.HDC), uintptr(right), uintptr(lower))
			procMoveToEx.Call(uintptr(di.HDC), uintptr(left), uintptr(lower), 0)
			procLineTo.Call(uintptr(di.HDC), uintptr(right), uintptr(upper))
			procSelectObject.Call(uintptr(di.HDC), prior)
			procDeleteObject.Call(pen)
		}
	}
	if di.ItemState&ODS_SELECTED != 0 {
		frameColor(di.HDC, rc, 197, 160, 91)
	}
	if di.ItemState&ODS_DISABLED != 0 {
		// Retain the icon artwork, use a neutral frame rather than emoji state.
		frameColor(di.HDC, rc, 88, 92, 95)
	}
	return true
}
