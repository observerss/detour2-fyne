//go:build !windows
// +build !windows

package proxy

// HandleArgs: only Windows needs extra recovery modes of the binary.
func HandleArgs(args []string) bool {
	return false
}

// Recover: other platforms restore their own state in Off (macOS snapshot)
// or lose the proxy together with the process (Android/iOS VPN).
func Recover() (bool, error) {
	return false, nil
}
