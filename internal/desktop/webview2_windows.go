//go:build windows

package main

import "github.com/wailsapp/go-webview2/webviewloader"

// webView2Version returns the version of the installed WebView2 runtime, as
// Wails finds it when it starts the webview ("141.0.3537.71", plus the
// channel name for Beta, Dev or Canary), or "" when it cannot be read. The
// app ships no fixed runtime, so the browser folder is empty.
func webView2Version() string {
	version, err := webviewloader.GetAvailableCoreWebView2BrowserVersionString("")
	if err != nil {
		return ""
	}
	return version
}
