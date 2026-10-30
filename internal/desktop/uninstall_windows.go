//go:build windows

package main

import (
	"errors"

	"golang.org/x/sys/windows/registry"
)

// openUninstallEntry opens the per-user uninstall key for reading and
// writing.
func openUninstallEntry() (uninstallEntry, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, uninstallKeyPath, registry.QUERY_VALUE|registry.SET_VALUE)
	if errors.Is(err, registry.ErrNotExist) {
		return nil, errNoUninstallEntry
	}
	if err != nil {
		return nil, err
	}
	return registryEntry{key}, nil
}

type registryEntry struct{ key registry.Key }

func (e registryEntry) ReadString(name string) (string, error) {
	value, _, err := e.key.GetStringValue(name)
	return value, err
}

func (e registryEntry) SetString(name, value string) error {
	return e.key.SetStringValue(name, value)
}

func (e registryEntry) Close() error { return e.key.Close() }
