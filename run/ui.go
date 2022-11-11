package run

import (
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	lo "github.com/observerss/detour2-fyne/layout"
	"github.com/observerss/detour2-fyne/profile"
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
	TextStopFc        = "停止&移除"
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
}

func NewUI(parent fyne.Window) *UI {
	return &UI{
		ProfileSelect:  widget.NewSelect([]string{}, func(s string) {}),
		LocalPort:      widget.NewEntry(),
		RunOnStartup:   widget.NewCheck(TextRunOnStartup, func(b bool) {}),
		UseGlobalProxy: widget.NewCheck(TextGlobalProxy, func(b bool) {}),
		ToggleRun:      widget.NewButton(TextStartFC, func() {}),
		LogEntry:       widget.NewMultiLineEntry(),
		Parent:         parent,
	}

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
	ui.ProfileSelect.SetSelected(names[0])

	ui.LocalPort.SetPlaceHolder(DefaultPort)
	ui.LocalPort.SetText(DefaultPort)
}

func (ui *UI) SetupBindings() {
	ui.ToggleRun.OnTapped = ui.HandleToggleRun
}

func (ui *UI) HandleToggleRun() {
	logger.Info.SetOutput(&clog{Entry: ui.LogEntry})
	logger.Error.SetOutput(&clog{Entry: ui.LogEntry})
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

	// err := deploy.DeployServer(conf)

	// if err != nil {
	// 	text.SetText("测试失败: " + err.Error())
	// } else {
	// 	conf.Remove = true
	// 	deploy.DeployServer(conf)
	// 	text.SetText("测试成功, 已成功部署并销毁云函数")
	// }
}

func (ui *UI) StopRunning() {

}

type clog struct {
	Entry *widget.Entry
}

func (l *clog) Write(p []byte) (int, error) {
	return 0, nil
}
