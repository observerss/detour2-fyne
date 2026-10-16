//go:build !linux && !android && !ios
// +build !linux,!android,!ios

package main

// trayAvailable: Windows and macOS always have a notification area / menu bar.
func trayAvailable() bool {
	return true
}
