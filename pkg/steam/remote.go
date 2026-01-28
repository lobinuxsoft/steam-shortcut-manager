// Package steam - remote operations support
package steam

import (
	"path"

	"github.com/shadowblip/steam-shortcut-manager/pkg/remote"
)

// RemoteClient is set by the command layer when remote mode is enabled
var RemoteClient *remote.Client

// SetRemoteClient sets the remote client for remote operations
func SetRemoteClient(client *remote.Client) {
	RemoteClient = client
}

// IsRemote returns true if remote mode is enabled
func IsRemote() bool {
	return RemoteClient != nil
}

// GetRemoteBaseDir returns the Steam base directory on the remote host
func GetRemoteBaseDir() (string, error) {
	// On Linux, Steam is typically at ~/.steam/steam
	homeDir, err := RemoteClient.GetHomeDir()
	if err != nil {
		return "", err
	}
	return path.Join(homeDir, ".steam", "steam"), nil
}

// GetRemoteUserDir returns the Steam userdata directory on the remote host
func GetRemoteUserDir() (string, error) {
	baseDir, err := GetRemoteBaseDir()
	if err != nil {
		return "", err
	}
	return path.Join(baseDir, "userdata"), nil
}

// GetRemoteUsers returns a list of Steam user IDs on the remote host
func GetRemoteUsers() ([]string, error) {
	userDir, err := GetRemoteUserDir()
	if err != nil {
		return nil, err
	}

	files, err := RemoteClient.ReadDir(userDir)
	if err != nil {
		return nil, err
	}

	users := []string{}
	for _, f := range files {
		if !f.IsDir() {
			continue
		}
		users = append(users, f.Name())
	}

	return users, nil
}

// GetRemoteShortcutsPath returns the path to shortcuts.vdf on the remote host
func GetRemoteShortcutsPath(user string) (string, error) {
	userDir, err := GetRemoteUserDir()
	if err != nil {
		return "", err
	}
	return path.Join(userDir, user, "config", "shortcuts.vdf"), nil
}

// RemoteHasShortcuts checks if the user has a shortcuts file on the remote host
func RemoteHasShortcuts(user string) bool {
	shortcutsPath, err := GetRemoteShortcutsPath(user)
	if err != nil {
		return false
	}
	return RemoteClient.FileExists(shortcutsPath)
}

// ReadRemoteFile reads a file from the remote host
func ReadRemoteFile(remotePath string) ([]byte, error) {
	return RemoteClient.ReadFile(remotePath)
}

// WriteRemoteFile writes a file to the remote host
func WriteRemoteFile(remotePath string, data []byte) error {
	return RemoteClient.WriteFile(remotePath, data, 0666)
}
