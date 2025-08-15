//go:build !windows

package steam

// Autor: Matias Galarza (Lobinux)

import (
	"os"
	"path"
)

// GetBaseDir will return the base steam config directory
func GetBaseDir() (string, error) {
	dirname, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return path.Join(dirname, ".steam", "steam"), nil
}
