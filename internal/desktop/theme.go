package main

// fixedWindowTheme reports whether mode pins the window to dark or light.
// "system", and anything else, follows the system theme.
func fixedWindowTheme(mode string) (dark, fixed bool) {
	switch mode {
	case "dark":
		return true, true
	case "light":
		return false, true
	default:
		return false, false
	}
}
