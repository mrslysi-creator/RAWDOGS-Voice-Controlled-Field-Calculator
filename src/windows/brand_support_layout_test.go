package main

import (
	"os"
	"strings"
	"testing"
)

func TestCoffeeIsClickableAndDoesNotTriggerOnStartup(t *testing.T) {
	b, err := os.ReadFile("source_main_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, part := range []string{
		`const buyMeACoffeeURL = "https://buymeacoffee.com/silvesta"`,
		`a.addButton(ID_BUY_ME_A_COFFEE, guide.SupportX, 18, guide.SupportW, 58`,
		`case ID_BUY_ME_A_COFFEE:`,
		`a.openBuyMeACoffee()`,
		`procShellExecuteW.Call(uintptr(a.hwnd)`,
		`utf16Ptr(buyMeACoffeeURL)`,
	} {
		if !strings.Contains(s, part) {
			t.Errorf("support click wiring missing: %s", part)
		}
	}
}
