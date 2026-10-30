//go:build windows

package main

import (
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/w32"
)

func systemPrefersDark() bool {
	return w32.IsCurrentlyDarkMode()
}

// setNativeDarkMode switches the DWM dark mode attribute, which tints the Mica
// backdrop behind the translucent window.
func setNativeDarkMode(window *application.WebviewWindow, dark bool) {
	if window == nil {
		return
	}
	application.InvokeSync(func() {
		if hwnd := window.NativeWindow(); hwnd != nil {
			w32.SetTheme(uintptr(hwnd), dark)
		}
	})
}
