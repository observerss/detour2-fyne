//go:build windows
// +build windows

package proxy

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/observerss/detour2-fyne/utils"
	"github.com/observerss/detour2/logger"
)

// Command line modes of the app binary used for recovery (see HandleArgs).
const (
	argWatchdogLaunch = "--detour2-proxy-watchdog-launch"
	argWatchdog       = "--detour2-proxy-watchdog"
	argRestore        = "--detour2-restore-proxy"
)

const (
	defaultRunOnceName = "Detour2RestoreProxy"
	defaultBypass      = "<local>"
)

// Settings is a snapshot of the system proxy configuration.
type Settings struct {
	Flags         uint32 `json:"flags"`
	ProxyServer   string `json:"proxyServer"`
	ProxyBypass   string `json:"proxyBypass"`
	AutoConfigURL string `json:"autoConfigURL"`
}

// state is persisted while we own the system proxy, so whoever finds it after
// the owner died (watchdog, RunOnce at next logon, next app start) can put the
// user's previous settings back.
type state struct {
	Server   string    `json:"server"`
	PID      int       `json:"pid"`
	Exe      string    `json:"exe"`
	Conn     string    `json:"conn,omitempty"`
	RunOnce  string    `json:"runOnce,omitempty"`
	Previous *Settings `json:"previous"`
}

// Guard points the system proxy at us and guarantees it is put back: on
// Disable, and via the recovery paths if the process dies first.
type Guard struct {
	StatePath string

	conn            string // WinINet connection, "" = LAN; tests use a dummy one
	runOnce         string // RunOnce value name; tests use their own
	watchdogStarted bool
}

func NewGuard(statePath string) *Guard {
	return &Guard{StatePath: statePath}
}

var (
	defaultGuard     *Guard
	defaultGuardErr  error
	defaultGuardOnce sync.Once
)

func appGuard() (*Guard, error) {
	defaultGuardOnce.Do(func() {
		path, err := utils.AppConfigFile("proxy-state.json")
		if err != nil {
			defaultGuardErr = fmt.Errorf("无法定位系统代理恢复信息路径: %w", err)
			return
		}
		defaultGuard = NewGuard(path)
	})
	return defaultGuard, defaultGuardErr
}

// SetGlobalProxy 设置全局代理: 先记下当前系统代理设置, Off 时原样恢复;
// 进程异常退出时由看门狗 / RunOnce / 下次启动恢复.
// proxyServer: host:port, bypasses: 不走代理的地址, 默认 "<local>"
func SetGlobalProxy(proxyServer string, bypasses ...string) error {
	g, err := appGuard()
	if err != nil {
		return err
	}
	return g.Enable(proxyServer, bypasses...)
}

// Off 关闭代理: 恢复 SetGlobalProxy 之前的系统代理设置
func Off() error {
	g, err := appGuard()
	if err != nil {
		return err
	}
	return g.Disable()
}

func Status() string {
	cur, err := snapshot("")
	if err != nil {
		return "Windows 系统代理状态读取失败: " + err.Error()
	}
	if g, err := appGuard(); err == nil {
		if st, err := readState(g.StatePath); err == nil && pointsAt(cur, st.Server) {
			return "Windows 系统代理已指向 " + st.Server
		}
	}
	if cur.Flags&_PROXY_TYPE_PROXY != 0 {
		return "Windows 系统代理由其他程序设置: " + cur.ProxyServer
	}
	return "Windows 系统代理未启用"
}

// Recover undoes a system proxy left pointing at us by an instance that died
// without cleaning up. It reports whether anything was restored.
func Recover() (bool, error) {
	g, err := appGuard()
	if err != nil {
		return false, err
	}
	return restore(g.StatePath, true)
}

func (g *Guard) runOnceName() string {
	if g.runOnce != "" {
		return g.runOnce
	}
	return defaultRunOnceName
}

// Enable remembers the current system proxy settings and replaces them with
// a fixed proxy at server ("host:port").
func (g *Guard) Enable(server string, bypasses ...string) error {
	if server == "" {
		return fmt.Errorf("代理服务器(proxyServer)配置为空")
	}
	bypass := defaultBypass
	if len(bypasses) > 0 {
		bypass = strings.Join(bypasses, ";")
	}
	// undo a leftover from a crashed run first, or we would remember our own
	// dead proxy as the user's setting
	if _, err := restore(g.StatePath, true); err != nil {
		return err
	}
	prev, err := snapshot(g.conn)
	if err != nil {
		return fmt.Errorf("读取系统代理设置失败: %w", err)
	}
	if pointsAt(prev, server) {
		prev = defaultSettings() // stale, from a build that did not record state
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	st := &state{
		Server:   server,
		PID:      os.Getpid(),
		Exe:      exe,
		Conn:     g.conn,
		RunOnce:  g.runOnceName(),
		Previous: prev,
	}
	if err := writeState(g.StatePath, st); err != nil {
		return fmt.Errorf("保存系统代理恢复信息失败: %w", err)
	}
	if err := g.installRecovery(exe); err != nil {
		// the state file alone still lets the next launch recover
		logger.Warn.Println("系统代理异常退出保护未完全启用:", err)
	}
	if err := setProxy(g.conn, server, bypass); err != nil {
		os.Remove(g.StatePath)
		removeRecovery(st.RunOnce)
		return fmt.Errorf("设置系统代理失败: %w", err)
	}
	return nil
}

// Disable puts back the settings saved by Enable, unless the user changed the
// system proxy in the meantime.
func (g *Guard) Disable() error {
	_, err := restore(g.StatePath, false)
	return err
}

// HandleArgs runs the recovery modes of the binary. main must call it first
// and exit when it returns true.
func HandleArgs(args []string) bool {
	if len(args) < 2 {
		return false
	}
	switch args[1] {
	case argWatchdogLaunch:
		if len(args) == 4 {
			launchWatchdog(args[2], args[3])
		}
	case argWatchdog:
		if len(args) == 4 {
			if pid, err := strconv.Atoi(args[2]); err == nil {
				watch(pid, args[3])
			}
		}
	case argRestore:
		if len(args) == 3 {
			restore(args[2], true)
		}
	default:
		return false
	}
	return true
}

func restore(statePath string, onlyIfOwnerDead bool) (bool, error) {
	st, err := readState(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if onlyIfOwnerDead && st.PID != os.Getpid() && processAlive(st.PID, st.Exe) {
		return false, nil
	}
	cur, err := snapshot(st.Conn)
	if err != nil {
		return false, fmt.Errorf("读取系统代理设置失败: %w", err)
	}
	restored := false
	if pointsAt(cur, st.Server) {
		prev := st.Previous
		if prev == nil {
			prev = defaultSettings()
		}
		if err := apply(st.Conn, prev); err != nil {
			return false, fmt.Errorf("恢复系统代理失败: %w", err)
		}
		restored = true
	}
	// state file last: as long as it exists, someone can still finish the job
	if err := removeRecovery(st.RunOnce); err != nil {
		return restored, err
	}
	if err := os.Remove(statePath); err != nil && !os.IsNotExist(err) {
		return restored, err
	}
	return restored, nil
}

func readState(path string) (*state, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	st := &state{}
	if err := json.Unmarshal(content, st); err != nil {
		// unusable either way; drop it so it cannot block Enable forever
		os.Remove(path)
		return nil, fmt.Errorf("系统代理恢复信息已损坏, 已删除: %w", err)
	}
	return st, nil
}

func writeState(path string, st *state) error {
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return utils.WriteFileAtomic(path, data, 0o600)
}
