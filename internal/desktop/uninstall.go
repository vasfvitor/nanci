package main

import (
	"errors"
	"path/filepath"
	"strings"
)

// uninstallKeyPath is the entry the per-user NSIS installer writes for
// Programs and Features (build/windows/nsis/wails_tools.nsh, macro
// wails.writeUninstaller, with UNINST_KEY_NAME "NanciDesktop" from
// project.nsi), under HKEY_CURRENT_USER.
const uninstallKeyPath = `Software\Microsoft\Windows\CurrentVersion\Uninstall\NanciDesktop`

// errNoUninstallEntry means the key does not exist: a portable copy, a dev
// build or a system other than Windows.
var errNoUninstallEntry = errors.New("entrada de desinstalação não encontrada")

// uninstallEntry reads and writes string values of the uninstall key.
type uninstallEntry interface {
	ReadString(name string) (string, error)
	SetString(name, value string) error
	Close() error
}

// recordInstalledVersion keeps DisplayVersion of the uninstall entry in step
// with the running version. The installer writes it once, and an automatic
// update replaces only the exe, so after an update Programs and Features
// would still list the old version. The entry is touched only when its
// DisplayIcon, the installed exe, is the exe running, so a copy run from
// elsewhere leaves the installed one alone.
func recordInstalledVersion(open func() (uninstallEntry, error), exe, version string) error {
	entry, err := open()
	if errors.Is(err, errNoUninstallEntry) {
		return nil
	}
	if err != nil {
		return err
	}
	defer func() { _ = entry.Close() }()

	icon, err := entry.ReadString("DisplayIcon")
	if err != nil || !samePath(icon, exe) {
		return nil
	}
	current, _ := entry.ReadString("DisplayVersion")
	if current == version {
		return nil
	}
	return entry.SetString("DisplayVersion", version)
}

// samePath compares two Windows paths, ignoring case and quotes.
func samePath(a, b string) bool {
	clean := func(p string) string { return filepath.Clean(strings.Trim(p, `"`)) }
	return strings.EqualFold(clean(a), clean(b))
}
