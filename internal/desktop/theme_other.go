//go:build !windows

package main

import "github.com/wailsapp/wails/v3/pkg/application"

func systemPrefersDark() bool { return false }

func setNativeDarkMode(*application.WebviewWindow, bool) {}
