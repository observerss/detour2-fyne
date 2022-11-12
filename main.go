//go:generate fyne bundle -o data.go Icon.png
package main

import (
	"os"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"github.com/flopp/go-findfont"

	"github.com/observerss/detour2-fyne/profile"
	"github.com/observerss/detour2-fyne/run"
	th "github.com/observerss/detour2-fyne/theme"
	"github.com/observerss/detour2/logger"
)

func init() {
	// setup chinese font
	fontPaths := findfont.List()
	for _, name := range []string{"msyh.ttc", "simsun.ttf", "simhei.ttf", "simkai.ttf"} {
		for _, path := range fontPaths {
			if strings.Contains(path, name) {
				os.Setenv("FYNE_FONT", path)
				return
			}
		}
	}
}

var (
	Title       = "Detour2"
	TextRun     = "运行"
	TextProfile = "配置"
	TextDisplay = "Show"
	ProfileSize = fyne.Size{Width: 720, Height: 560}
)

func main() {
	a := app.New()
	a.Settings().SetTheme(th.RunTheme())
	w := a.NewWindow(Title)
	w.SetIcon(resourceIconPng)
	w.Resize(ProfileSize)

	if desk, ok := a.(desktop.App); ok {
		display := fyne.NewMenuItem(TextDisplay, func() {
			w.Show()
		})
		m := fyne.NewMenu(Title, display)

		desk.SetSystemTrayMenu(m)
		desk.SetSystemTrayIcon(resourceIconPng)
	}

	runUI := run.NewUI(w)
	profileUI := profile.NewUI(w)
	tabs := container.NewAppTabs(
		container.NewTabItem(TextRun, runUI.MakeUI()),
		container.NewTabItem(TextProfile, profileUI.MakeUI()),
	)
	tabs.SetTabLocation(container.TabLocationTop)
	tabs.OnSelected = func(t *container.TabItem) {
		if t.Text == TextRun {
			a.Settings().SetTheme(th.RunTheme())
			runUI.ResetUI()
		} else {
			a.Settings().SetTheme(theme.DefaultTheme())
			logger.Info.SetOutput(os.Stdout)
			logger.Error.SetOutput(os.Stderr)
		}
	}

	w.SetContent(tabs)
	w.SetCloseIntercept(func() {
		w.Hide()
	})
	w.ShowAndRun()
}
