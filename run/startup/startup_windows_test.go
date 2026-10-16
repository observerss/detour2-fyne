//go:build windows
// +build windows

package startup

import "testing"

func TestRunCommand(t *testing.T) {
	got := runCommand(`C:\Users\张三\AppData\Local\Programs\Detour2\Detour2.exe`)
	want := `"C:\Users\张三\AppData\Local\Programs\Detour2\Detour2.exe"`
	if got != want {
		t.Fatalf("runCommand = %s, want %s", got, want)
	}
}
