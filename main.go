//go:generate fyne bundle -o data.go Icon.png
package main

import (
	"io"
	"os"
	"runtime"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"

	"github.com/observerss/detour2-fyne/profile"
	"github.com/observerss/detour2-fyne/run"
	th "github.com/observerss/detour2-fyne/theme"
	"github.com/observerss/detour2/logger"
)

var (
	AppID                  = "github.com.observerss.detour2fyne"
	Title                  = "Detour2"
	TextRun                = "运行"
	TextProfile            = "配置"
	TextDisplay            = "显示主窗口"
	TextTrayMinimizedTitle = "已最小化到托盘"
	TextTrayMinimizedBody  = "Detour2 仍在后台运行，可通过托盘菜单重新打开。"
	TextLog                = "日志"
	ProfileSize            = fyne.Size{Width: 720, Height: 560}
)

func main() {
	a := app.NewWithID(AppID)
	a.Settings().SetTheme(th.RunTheme())
	w := a.NewWindow(Title)
	w.SetIcon(resourceIconPng)
	w.Resize(ProfileSize)
	setupDesktop(a, w)

	runUI := run.NewUI(w)
	profileUI := profile.NewUI(w)
	logger.Info.SetOutput(io.MultiWriter(os.Stdout, runUI.Logs))
	logger.Error.SetOutput(io.MultiWriter(os.Stderr, runUI.Logs))

	var tabs *container.AppTabs
	if runtime.GOOS == "android" || runtime.GOOS == "ios" {
		tabs = container.NewAppTabs(
			container.NewTabItemWithIcon(TextRun, theme.MediaPlayIcon(), runUI.MakeUI()),
			container.NewTabItemWithIcon(TextProfile, theme.SettingsIcon(), profileUI.MakeUI()),
			container.NewTabItemWithIcon(TextLog, theme.HistoryIcon(), runUI.MakeLogUI()),
		)
		tabs.SetTabLocation(container.TabLocationBottom)
	} else {
		tabs = container.NewAppTabs(
			container.NewTabItem(TextRun, runUI.MakeUI()),
			container.NewTabItem(TextProfile, profileUI.MakeUI()),
			container.NewTabItem(TextLog, runUI.MakeLogUI()),
		)
		tabs.SetTabLocation(container.TabLocationTop)
	}

	tabs.OnSelected = func(t *container.TabItem) {
		if t.Text == TextRun {
			a.Settings().SetTheme(th.RunTheme())
			runUI.ResetUI()
		} else if t.Text == TextProfile {
			// android needs manually refresh
			if runtime.GOOS == "android" {
				profileUI.Delete.Refresh()
				profileUI.Reset.Refresh()
				profileUI.Test.Refresh()
				profileUI.Save.Refresh()

				profileUI.Left.OnSelected(profileUI.CurrentIdx)
			}
		}
	}

	w.SetContent(tabs)
	w.ShowAndRun()
}
