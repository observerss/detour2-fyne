//go:build windows
// +build windows

// windows开机自启动
package startup

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

const (
	runKeyPath   = `Software\Microsoft\Windows\CurrentVersion\Run`
	runValueName = "Detour2"
)

// Enable 开机自启动
func Enable() error {
	path, err := os.Executable()
	if err != nil {
		return err
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return err
	}

	key, _, err := registry.CreateKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()

	if err := key.SetStringValue(runValueName, runCommand(path)); err != nil {
		return err
	}

	return removeLegacyStartupShortcut()
}

// runCommand quotes path for the Run key. Not %q: Go escaping would double
// every backslash, which is not how Windows command lines are quoted.
func runCommand(path string) string {
	return `"` + path + `"`
}

// Disable 开机不自启动
func Disable() error {
	key, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE)
	if err == nil {
		defer key.Close()
		if err := key.DeleteValue(runValueName); err != nil && err != registry.ErrNotExist {
			return err
		}
	}

	return removeLegacyStartupShortcut()
}

func removeLegacyStartupShortcut() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	file := filepath.Join(home, `AppData\Roaming\Microsoft\Windows\Start Menu\Programs\Startup\Detour2.lnk`)
	if err := os.Remove(file); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
