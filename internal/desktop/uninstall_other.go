//go:build !windows

package main

// openUninstallEntry: only the Windows installer writes an uninstall entry.
func openUninstallEntry() (uninstallEntry, error) {
	return nil, errNoUninstallEntry
}
