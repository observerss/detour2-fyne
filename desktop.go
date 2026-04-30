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
		w.SetCloseIntercept(func() {
			w.Hide()
			a.SendNotification(fyne.NewNotification(TextTrayMinimizedTitle, TextTrayMinimizedBody))
		})
	}
}
