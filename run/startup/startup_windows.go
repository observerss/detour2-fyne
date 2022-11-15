//go:build windows
// +build windows

// windows开机自启动
package startup

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Enable 开机自启动
func Enable() error {
	path, _ := os.Executable()
	ps := New()
	WIN_CREATE_SHORTCUT = strings.Replace(WIN_CREATE_SHORTCUT, "PLACEHOLDER", path, 1)
	_, _, err := ps.execute(WIN_CREATE_SHORTCUT)
	return err
}

// Disable 开机不自启动
func Disable() error {
	home, _ := os.UserHomeDir()
	file := filepath.Join(home, `AppData\Roaming\Microsoft\Windows\Start Menu\Programs\Startup\Detour2.lnk`)
	return os.Remove(file)
}

type PowerShell struct {
	powerShell string
}

var WIN_CREATE_SHORTCUT = `$WshShell = New-Object -comObject WScript.Shell
$Shortcut = $WshShell.CreateShortcut("$HOME\AppData\Roaming\Microsoft\Windows\Start Menu\Programs\Startup\Detour2.lnk")
$Shortcut.TargetPath = "PLACEHOLDER"
$Shortcut.Save()`

// New create new session
func New() *PowerShell {
	ps, _ := exec.LookPath("powershell.exe")
	return &PowerShell{
		powerShell: ps,
	}
}

func (p *PowerShell) execute(args ...string) (stdOut string, stdErr string, err error) {
	args = append([]string{"-NoProfile", "-NonInteractive"}, args...)
	cmd := exec.Command(p.powerShell, args...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	stdOut, stdErr = stdout.String(), stderr.String()
	return
}
