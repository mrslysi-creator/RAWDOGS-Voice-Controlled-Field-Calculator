package main

import "testing"

func TestVoiceTrainingPanelsResizeWithoutOverlap(t *testing.T) {
	for _, client := range [][2]int{{850, 680}, {950, 820}, {1280, 900}, {1440, 1020}, {1650, 1050}, {1920, 1200}, {2560, 1440}} {
		w, h := client[0], client[1]
		b := trainingBoxes(w, h)
		if b.leftTop.y != 330 || b.rightTop.y != 330 || b.leftTop.h < 185 {
			t.Errorf("client %dx%d: protected region changed or exercise panel too short: %+v", w, h, b)
		}
		if b.leftTop.x+b.leftTop.w+18 != b.rightTop.x || b.leftBottom.x+b.leftBottom.w+18 != b.rightBottom.x {
			t.Errorf("client %dx%d: horizontal overlap: %+v", w, h, b)
		}
		if b.rightTop.x+b.rightTop.w > w-20 || b.rightBottom.x+b.rightBottom.w > w-20 {
			t.Errorf("client %dx%d: overflow right edge: %+v", w, h, b)
		}
		if b.leftTop.y+b.leftTop.h >= b.leftBottom.y || b.rightTop.y+b.rightTop.h >= b.rightBottom.y {
			t.Errorf("client %dx%d: exercise overlaps transcript: %+v", w, h, b)
		}
		if b.leftBottom.y+b.leftBottom.h >= b.footer.y || b.rightBottom.y+b.rightBottom.h >= b.footer.y {
			t.Errorf("client %dx%d: transcript overlaps footer: %+v", w, h, b)
		}
		if b.footer.y+b.footer.h > h {
			t.Errorf("client %dx%d: footer offscreen: %+v", w, h, b)
		}
	}
}
