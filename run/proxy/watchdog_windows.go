//go:build windows
// +build windows

package proxy

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const (
	runOnceKey     = `Software\Microsoft\Windows\CurrentVersion\RunOnce`
	stillActive    = 259
	detachedFlags  = windows.DETACHED_PROCESS | windows.CREATE_NEW_PROCESS_GROUP | windows.CREATE_NO_WINDOW
	breakawayFlags = detachedFlags | windows.CREATE_BREAKAWAY_FROM_JOB
)

// processAlive reports whether pid is still running the executable exe.
// Comparing the image path guards against PID reuse after a reboot.
func processAlive(pid int, exe string) bool {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer windows.CloseHandle(h)
	var code uint32
	if err := windows.GetExitCodeProcess(h, &code); err != nil || code != stillActive {
		return false
	}
	if exe == "" {
		return true
	}
	buf := make([]uint16, windows.MAX_LONG_PATH)
	size := uint32(len(buf))
	if err := windows.QueryFullProcessImageName(h, 0, &buf[0], &size); err != nil {
		return true // alive, image unknown: err on the side of not touching its proxy
	}
	return strings.EqualFold(filepath.Clean(windows.UTF16ToString(buf[:size])), filepath.Clean(exe))
}

// installRecovery registers a RunOnce entry so the next logon restores the
// proxy if this session ends without us doing it (shutdown, power loss), and
// starts the watchdog for kills/crashes within this session.
func (g *Guard) installRecovery(exe string) error {
	if !g.watchdogStarted {
		if err := spawnDetached(exe, argWatchdogLaunch, strconv.Itoa(os.Getpid()), g.StatePath); err != nil {
			return err
		}
		g.watchdogStarted = true
	}
	key, _, err := registry.CreateKey(registry.CURRENT_USER, runOnceKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	cmdline := syscall.EscapeArg(exe) + " " + argRestore + " " + syscall.EscapeArg(g.StatePath)
	return key.SetStringValue(g.runOnceName(), cmdline)
}

func removeRecovery(runOnceName string) error {
	key, err := registry.OpenKey(registry.CURRENT_USER, runOnceKey, registry.SET_VALUE)
	if err != nil {
		if err == registry.ErrNotExist {
			return nil
		}
		return err
	}
	defer key.Close()
	if err := key.DeleteValue(runOnceName); err != nil && err != registry.ErrNotExist {
		return err
	}
	return nil
}

// spawnDetached starts exe outside our console, process group and (if
// allowed) job, so killing us or our process tree does not take it along.
func spawnDetached(exe string, args ...string) error {
	start := func(flags uint32) error {
		cmd := exec.Command(exe, args...)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: flags}
		if err := cmd.Start(); err != nil {
			return err
		}
		go cmd.Wait() // reap; the launcher exits right away
		return nil
	}
	if err := start(breakawayFlags); err == nil {
		return nil
	}
	// the job we run in may forbid breakaway
	return start(detachedFlags)
}

// launchWatchdog runs in the short-lived launcher process: starting the real
// watchdog from here leaves it with a dead parent, so it is not part of the
// app's process tree ("End process tree", taskkill /T).
func launchWatchdog(pid, statePath string) {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	spawnDetached(exe, argWatchdog, pid, statePath)
}

// watch blocks until process pid exits, then undoes its proxy if needed.
func watch(pid int, statePath string) {
	if h, err := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(pid)); err == nil {
		windows.WaitForSingleObject(h, windows.INFINITE)
		windows.CloseHandle(h)
	}
	restore(statePath, true)
}
