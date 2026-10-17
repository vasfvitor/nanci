//go:build !windows

package main

// webView2Version is empty off Windows: macOS and Linux use WebKit.
func webView2Version() string {
	return ""
}
