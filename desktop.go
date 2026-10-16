//go:build !android && !ios
// +build !android,!ios

package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
)

func setupDesktop(a fyne.App, w fyne.Window) {
	if desk, ok := a.(desktop.App); ok {
		display := fyne.NewMenuItem(TextDisplay, func() {
			w.Show()
		})
		menu := fyne.NewMenu(Title, display)

		desk.SetSystemTrayMenu(menu)
		desk.SetSystemTrayIcon(resourceIconPng)
		// without a tray icon a hidden window could never be shown again,
		// so only hide-on-close when the tray is really there
		if trayAvailable() {
			w.SetCloseIntercept(func() {
				w.Hide()
				a.SendNotification(fyne.NewNotification(TextTrayMinimizedTitle, TextTrayMinimizedBody))
			})
		}
	}
}
