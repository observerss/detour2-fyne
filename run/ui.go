package run

import (
	"os"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	lo "github.com/observerss/detour2-fyne/layout"
	"github.com/observerss/detour2-fyne/profile"
	"github.com/observerss/detour2/deploy"
	"github.com/observerss/detour2/logger"
)

var (
	TextFCPlaceholder = "请选择配置"
	TextFCLabel       = "云函数配置"
	TextPortLabel     = "本地端口"
	DefaultPort       = "3333"
	TextRunOnStartup  = "开机自动运行"
	TextGlobalProxy   = "启用全局代理"
	TextStartFC       = "部署&启动"
	TextStopFC        = "停止&移除"
)

type UI struct {
	ProfileSelect  *widget.Select
	LocalPort      *widget.Entry
	RunOnStartup   *widget.Check
	UseGlobalProxy *widget.Check
	ToggleRun      *widget.Button
	LogEntry       *widget.Entry
	Parent         fyne.Window
	Started        bool
	Logs           *clog
}

func NewUI(parent fyne.Window) *UI {
	ui := &UI{
		ProfileSelect:  widget.NewSelect([]string{}, func(s string) {}),
		LocalPort:      widget.NewEntry(),
		RunOnStartup:   widget.NewCheck(TextRunOnStartup, func(b bool) {}),
		UseGlobalProxy: widget.NewCheck(TextGlobalProxy, func(b bool) {}),
		ToggleRun:      widget.NewButton(TextStartFC, func() {}),
		LogEntry:       widget.NewMultiLineEntry(),
		Parent:         parent,
	}
	ui.Logs = &clog{Entry: ui.LogEntry, Lines: make([]string, 0)}
	return ui
}

func (ui *UI) MakeUI() fyne.CanvasObject {
	// initializae
	ui.SetupBindings()
	ui.ResetUI()

	grid := container.New(layout.NewFormLayout(),
		widget.NewLabel(TextFCLabel), ui.ProfileSelect,
		layout.NewSpacer(), layout.NewSpacer(),
		widget.NewLabel(TextPortLabel), ui.LocalPort,
		layout.NewSpacer(), layout.NewSpacer(),
		ui.RunOnStartup, ui.UseGlobalProxy,
	)

	submit := container.NewCenter(ui.ToggleRun)

	form := lo.NewPaddingContainer(
		container.NewVBox(layout.NewSpacer(), grid, submit, layout.NewSpacer()),
		&lo.Padding{Left: 80, Right: 80},
	)

	logs := lo.NewPaddingContainer(
		container.NewMax(ui.LogEntry),
		&lo.Padding{Top: 5, Bottom: 10, Left: 10, Right: 10},
	)

	split := container.NewVSplit(form, logs)
	split.SetOffset(0.4)

	return split
}

func (ui *UI) ResetUI() {
	ui.ProfileSelect.PlaceHolder = TextFCPlaceholder
	profs, _ := profile.LoadProfiles()
	names := profile.GetProfileNames(profs)
	ui.ProfileSelect.Options = names
	if len(names) > 0 {
		ui.ProfileSelect.SetSelected(names[0])
	}
	ui.LocalPort.SetPlaceHolder(DefaultPort)
	ui.LocalPort.SetText(DefaultPort)
}

func (ui *UI) SetupBindings() {
	ui.ToggleRun.OnTapped = ui.HandleToggleRun
}

func (ui *UI) HandleToggleRun() {
	logger.Info.SetOutput(ui.Logs)
	logger.Error.SetOutput(ui.Logs)
	defer func() {
		logger.Info.SetOutput(os.Stdout)
		logger.Error.SetOutput(os.Stderr)
	}()

	if !ui.Started {
		ui.StartRunning()
	} else {
		ui.StopRunning()
	}
}

func (ui *UI) StartRunning() {
	ui.ToggleRun.Disable()

	profs, _ := profile.LoadProfiles()
	names := profile.GetProfileNames(profs)
	name := names[ui.ProfileSelect.SelectedIndex()]
	prof := profs[name]
	conf := profile.ConvertProfileToConfig(prof)
	err := deploy.DeployServer(conf)
	if err != nil {
		ui.ToggleRun.Enable()
		return
	}

	ui.ToggleRun.Enable()
	ui.Started = true
	ui.ToggleRun.SetText(TextStopFC)
	// if err != nil {
	// 	text.SetText("测试失败: " + err.Error())
	// } else {
	// 	conf.Remove = true
	// 	deploy.DeployServer(conf)
	// 	text.SetText("测试成功, 已成功部署并销毁云函数")
	// }
}

func (ui *UI) StopRunning() {
	ui.ToggleRun.Disable()

	profs, _ := profile.LoadProfiles()
	names := profile.GetProfileNames(profs)
	name := names[ui.ProfileSelect.SelectedIndex()]
	prof := profs[name]
	conf := profile.ConvertProfileToConfig(prof)
	conf.Remove = true
	err := deploy.DeployServer(conf)
	if err != nil {
		ui.ToggleRun.Enable()
		return
	}

	ui.ToggleRun.Enable()
	ui.Started = false
	ui.ToggleRun.SetText(TextStartFC)
}

type clog struct {
	Entry *widget.Entry
	Lines []string
}

func (l *clog) Write(p []byte) (int, error) {
	l.Lines = append(l.Lines, string(p))
	if len(l.Lines) > 1000 {
		l.Lines = l.Lines[len(l.Lines)-1000:]
	}
	l.Entry.SetText(strings.Join(l.Lines, ""))
	l.Entry.CursorRow = len(l.Lines) - 1
	l.Entry.Refresh()
	return 0, nil
}
