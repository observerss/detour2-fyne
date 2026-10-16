//go:generate fyne bundle -o data.go Icon.png
package main

import (
	"os"
	"runtime"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"

	"github.com/observerss/detour2-fyne/profile"
	"github.com/observerss/detour2-fyne/run"
	"github.com/observerss/detour2-fyne/run/proxy"
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
	// recovery modes (proxy watchdog, RunOnce restore) of this same binary
	if proxy.HandleArgs(os.Args) {
		return
	}

	a := app.NewWithID(AppID)
	a.Settings().SetTheme(th.RunTheme())
	w := a.NewWindow(Title)
	w.SetIcon(resourceIconPng)
	w.Resize(ProfileSize)
	setupDesktop(a, w)

	runUI := run.NewUI(w)
	profileUI := profile.NewUI(w)
	runUI.AttachLogger()
	recoverSystemProxy()

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
			runUI.RefreshProfiles()
			if runtime.GOOS == "android" {
				runUI.CanvasObject.Refresh()
			}
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

	// ShowAndRun returns for every way of quitting (exit button, tray "Quit",
	// closing the last window); the UI loop is gone, so log to stdio again.
	runUI.DetachLogger()
	runUI.Shutdown()
}

// recoverSystemProxy undoes a system proxy left pointing at us by a previous
// run that was killed before it could restore it.
func recoverSystemProxy() {
	restored, err := proxy.Recover()
	if err != nil {
		logger.Error.Println(err)
	} else if restored {
		logger.Info.Println("已恢复上次异常退出时遗留的系统代理设置")
	}
}
