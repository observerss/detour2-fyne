//go:build windows
// +build windows

package proxy

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/windows/registry"
)

// The write tests target a dummy WinINet connection, but WinINet occasionally
// applies such writes to the LAN settings as well (seen on Windows 11), so they
// only run with DETOUR2_WININET_WRITE_TESTS=1, ideally in a throwaway VM. The
// LAN settings are compared before/after and put back if they changed.

const (
	envOwner      = "DETOUR2_TEST_OWNER"
	envConn       = "DETOUR2_TEST_CONN"
	envState      = "DETOUR2_TEST_STATE"
	envRunOnce    = "DETOUR2_TEST_RUNONCE"
	envWriteTests = "DETOUR2_WININET_WRITE_TESTS"
	testServer    = "127.0.0.1:33331"
	connectionKey = `Software\Microsoft\Windows\CurrentVersion\Internet Settings\Connections`
)

func TestMain(m *testing.M) {
	if HandleArgs(os.Args) {
		return
	}
	if os.Getenv(envOwner) != "" {
		runOwner()
		return
	}
	os.Exit(m.Run())
}

// runOwner plays the app: enable the proxy, then wait to be killed.
func runOwner() {
	g := &Guard{StatePath: os.Getenv(envState), conn: os.Getenv(envConn), runOnce: os.Getenv(envRunOnce)}
	if err := g.Enable(testServer); err != nil {
		fmt.Println("ERROR", err)
		os.Exit(1)
	}
	fmt.Println("ENABLED")
	time.Sleep(time.Hour)
}

func userSettings() *Settings {
	return &Settings{
		Flags:         _PROXY_TYPE_DIRECT | _PROXY_TYPE_AUTO_DETECT | _PROXY_TYPE_AUTO_PROXY_URL,
		ProxyServer:   "10.9.8.7:8080",
		ProxyBypass:   "*.corp;<local>",
		AutoConfigURL: "http://pac.example/proxy.pac",
	}
}

// setup gives each test its own dummy connection and RunOnce name, and checks
// afterwards that the real LAN settings were not touched.
func setup(t *testing.T) *Guard {
	t.Helper()
	if os.Getenv(envWriteTests) != "1" {
		t.Skip("modifies WinINet proxy settings; set " + envWriteTests + "=1 (ideally in a VM)")
	}
	lan, err := snapshot("")
	if err != nil {
		t.Fatal(err)
	}
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	g := &Guard{
		StatePath:       filepath.Join(t.TempDir(), "proxy-state.json"),
		conn:            "Detour2-Test-" + suffix,
		runOnce:         "Detour2-Test-RestoreProxy-" + suffix,
		watchdogStarted: true, // only TestWatchdogRestoresAfterKill wants one
	}
	t.Cleanup(func() {
		removeRecovery(g.runOnce)
		if key, err := registry.OpenKey(registry.CURRENT_USER, connectionKey, registry.SET_VALUE); err == nil {
			key.DeleteValue(g.conn)
			key.Close()
		}
		now, err := snapshot("")
		if err == nil && !reflect.DeepEqual(now, lan) {
			apply("", lan)
			t.Errorf("LAN proxy settings changed: %+v -> %+v (restored)", lan, now)
		}
	})
	if err := apply(g.conn, userSettings()); err != nil {
		t.Fatalf("cannot write dummy connection: %v", err)
	}
	if now, err := snapshot(""); err != nil || !reflect.DeepEqual(now, lan) {
		apply("", lan)
		t.Fatalf("dummy connection is not isolated from LAN settings here (%+v), aborting", now)
	}
	return g
}

func TestSameEndpoint(t *testing.T) {
	if !sameEndpoint("localhost:7777", "127.0.0.1:7777") || !sameEndpoint("LOCALHOST:7777 ", "127.0.0.1:7777") {
		t.Fatal("localhost should match 127.0.0.1")
	}
	if sameEndpoint("127.0.0.1:7777", "127.0.0.1:7778") || sameEndpoint("10.0.0.1:7777", "127.0.0.1:7777") {
		t.Fatal("different endpoints matched")
	}
}

func TestSnapshotLAN(t *testing.T) {
	s, err := snapshot("")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("LAN settings (read only): %+v", *s)
}

func TestApplySnapshotRoundTrip(t *testing.T) {
	g := setup(t)
	got, err := snapshot(g.conn)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, userSettings()) {
		t.Fatalf("snapshot = %+v, want %+v", got, userSettings())
	}
}

func TestEnableDisableRestoresPrevious(t *testing.T) {
	g := setup(t)
	if err := g.Enable(testServer); err != nil {
		t.Fatal(err)
	}
	cur, _ := snapshot(g.conn)
	if !pointsAt(cur, testServer) {
		t.Fatalf("after Enable: %+v", cur)
	}
	if !runOnceExists(t, g.runOnce) {
		t.Fatal("RunOnce entry missing after Enable")
	}
	if err := g.Disable(); err != nil {
		t.Fatal(err)
	}
	cur, _ = snapshot(g.conn)
	if !reflect.DeepEqual(cur, userSettings()) {
		t.Fatalf("after Disable = %+v, want %+v", cur, userSettings())
	}
	if _, err := os.Stat(g.StatePath); !os.IsNotExist(err) {
		t.Fatal("state file left behind")
	}
	if runOnceExists(t, g.runOnce) {
		t.Fatal("RunOnce entry left behind")
	}
}

func TestDisableKeepsUserChange(t *testing.T) {
	g := setup(t)
	if err := g.Enable(testServer); err != nil {
		t.Fatal(err)
	}
	changed := &Settings{Flags: _PROXY_TYPE_PROXY | _PROXY_TYPE_DIRECT, ProxyServer: "10.1.1.1:3128", ProxyBypass: "<local>"}
	if err := apply(g.conn, changed); err != nil {
		t.Fatal(err)
	}
	if err := g.Disable(); err != nil {
		t.Fatal(err)
	}
	cur, _ := snapshot(g.conn)
	if !reflect.DeepEqual(cur, changed) {
		t.Fatalf("user change overwritten: %+v", cur)
	}
}

func TestRecoverSkipsLiveOwner(t *testing.T) {
	g := setup(t)
	if err := g.Enable(testServer); err != nil {
		t.Fatal(err)
	}
	// the owner is this (live) process, seen from another one
	st, _ := readState(g.StatePath)
	st.PID = os.Getppid()
	st.Exe = ""
	writeState(g.StatePath, st)
	if restored, err := restore(g.StatePath, true); err != nil || restored {
		t.Fatalf("Recover with live owner: restored=%v err=%v", restored, err)
	}
	st.PID = 1 << 30 // no such process
	writeState(g.StatePath, st)
	if restored, err := restore(g.StatePath, true); err != nil || !restored {
		t.Fatalf("Recover with dead owner: restored=%v err=%v", restored, err)
	}
	cur, _ := snapshot(g.conn)
	if !reflect.DeepEqual(cur, userSettings()) {
		t.Fatalf("after Recover = %+v", cur)
	}
}

func TestWatchdogRestoresAfterKill(t *testing.T) {
	g := setup(t)
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(exe)
	cmd.Env = append(os.Environ(),
		envOwner+"=1", envConn+"="+g.conn, envState+"="+g.StatePath, envRunOnce+"="+g.runOnce)
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cmd.Process.Kill() })
	line, _ := bufio.NewReader(out).ReadString('\n')
	if strings.TrimSpace(line) != "ENABLED" {
		t.Fatalf("owner: %q", line)
	}
	cur, _ := snapshot(g.conn)
	if !pointsAt(cur, testServer) {
		t.Fatalf("owner did not set proxy: %+v", cur)
	}

	cmd.Process.Kill() // TerminateProcess, like taskkill /F
	cmd.Wait()

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		cur, _ = snapshot(g.conn)
		if reflect.DeepEqual(cur, userSettings()) {
			if _, err := os.Stat(g.StatePath); !os.IsNotExist(err) {
				time.Sleep(100 * time.Millisecond) // restore applies first, then cleans up
				continue
			}
			if runOnceExists(t, g.runOnce) {
				t.Fatal("RunOnce entry left behind")
			}
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("watchdog did not restore after kill: %+v", cur)
}

func runOnceExists(t *testing.T, name string) bool {
	t.Helper()
	key, err := registry.OpenKey(registry.CURRENT_USER, runOnceKey, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer key.Close()
	_, _, err = key.GetStringValue(name)
	return err == nil
}
