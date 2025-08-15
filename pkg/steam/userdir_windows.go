//go:build windows

package steam

// Autor: Matias Galarza (Lobinux)

import (
	"errors"

	"golang.org/x/sys/windows/registry"
)

// GetBaseDir will return the base steam config directory
func GetBaseDir() (string, error) {
	// We check in two different locations for the key
	// The first one is for 64-bit systems
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Wow6432Node\Valve\Steam`, registry.QUERY_VALUE)
	if err != nil {
		// If it fails, we check the 32-bit key
		key, err = registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Valve\Steam`, registry.QUERY_VALUE)
		if err != nil {
			return "", errors.New("cannot find steam registry key")
		}
	}
	defer key.Close()

	steamPath, _, err := key.GetStringValue("InstallPath")
	if err != nil {
		return "", err
	}

	return steamPath, nil
}
