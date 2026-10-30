package main

import (
	"errors"
	"reflect"
	"testing"
)

// fakeUninstallEntry is the uninstall key in memory; writes records each
// SetString.
type fakeUninstallEntry struct {
	values map[string]string
	writes []string
	closed bool
}

func (f *fakeUninstallEntry) ReadString(name string) (string, error) {
	v, ok := f.values[name]
	if !ok {
		return "", errors.New("valor não encontrado")
	}
	return v, nil
}

func (f *fakeUninstallEntry) SetString(name, value string) error {
	f.writes = append(f.writes, name+"="+value)
	f.values[name] = value
	return nil
}

func (f *fakeUninstallEntry) Close() error {
	f.closed = true
	return nil
}

const installedExe = `C:\Users\ana\AppData\Local\Programs\Nanci Desktop\nanci-desktop.exe`

func openFake(entry *fakeUninstallEntry) func() (uninstallEntry, error) {
	return func() (uninstallEntry, error) { return entry, nil }
}

func TestRecordInstalledVersionUpdatesTheInstalledExe(t *testing.T) {
	entry := &fakeUninstallEntry{values: map[string]string{"DisplayIcon": installedExe, "DisplayVersion": "0.6.0"}}

	running := `c:\users\ana\appdata\local\programs\nanci desktop\NANCI-DESKTOP.exe`
	if err := recordInstalledVersion(openFake(entry), running, "0.7.0"); err != nil {
		t.Fatal(err)
	}
	if want := []string{"DisplayVersion=0.7.0"}; !reflect.DeepEqual(entry.writes, want) {
		t.Errorf("writes = %v, want %v", entry.writes, want)
	}
	if !entry.closed {
		t.Error("key not closed")
	}
}

func TestRecordInstalledVersionLeavesTheEntryAlone(t *testing.T) {
	tests := []struct {
		name   string
		values map[string]string
		exe    string
	}{
		{"same version", map[string]string{"DisplayIcon": installedExe, "DisplayVersion": "0.7.0"}, installedExe},
		{"another exe", map[string]string{"DisplayIcon": installedExe, "DisplayVersion": "0.6.0"}, `D:\portable\nanci-desktop.exe`},
		{"no icon", map[string]string{"DisplayVersion": "0.6.0"}, installedExe},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := &fakeUninstallEntry{values: tt.values}
			if err := recordInstalledVersion(openFake(entry), tt.exe, "0.7.0"); err != nil {
				t.Fatal(err)
			}
			if len(entry.writes) != 0 {
				t.Errorf("writes = %v, want none", entry.writes)
			}
		})
	}
}

func TestRecordInstalledVersionWithoutTheKey(t *testing.T) {
	open := func() (uninstallEntry, error) { return nil, errNoUninstallEntry }
	if err := recordInstalledVersion(open, installedExe, "0.7.0"); err != nil {
		t.Errorf("err = %v, want nil when the app was not installed", err)
	}
}

func TestRecordInstalledVersionPassesOpenErrors(t *testing.T) {
	open := func() (uninstallEntry, error) { return nil, errCore }
	if err := recordInstalledVersion(open, installedExe, "0.7.0"); !errors.Is(err, errCore) {
		t.Errorf("err = %v, want errCore", err)
	}
}
