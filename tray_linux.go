//go:build linux && !android
// +build linux,!android

package main

import "github.com/godbus/dbus/v5"

// trayAvailable reports whether a StatusNotifierItem host is running.
// fyne's systray on Linux only speaks SNI, so e.g. stock GNOME without the
// AppIndicator extension shows no tray icon at all.
func trayAvailable() bool {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return false
	}
	defer conn.Close()
	var has bool
	err = conn.BusObject().Call("org.freedesktop.DBus.NameHasOwner", 0, "org.kde.StatusNotifierWatcher").Store(&has)
	return err == nil && has
}
