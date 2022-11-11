package run

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	lo "github.com/observerss/detour2-fyne/layout"
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
	Parent         fyne.Window
}

func MakeUI(parent fyne.Window) fyne.CanvasObject {
	ui := &UI{
		ProfileSelect:  widget.NewSelect([]string{}, func(s string) {}),
		LocalPort:      widget.NewEntry(),
		RunOnStartup:   widget.NewCheck(TextRunOnStartup, func(b bool) {}),
		UseGlobalProxy: widget.NewCheck(TextGlobalProxy, func(b bool) {}),
		ToggleRun:      widget.NewButton(TextStartFC, func() {}),
		Parent:         parent,
	}

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
		container.NewMax(widget.NewMultiLineEntry()),
		&lo.Padding{Top: 5, Bottom: 10, Left: 10, Right: 10},
	)

	split := container.NewVSplit(form, logs)
	split.SetOffset(0.4)

	return split
}

func (ui *UI) ResetUI() {
	ui.ProfileSelect.PlaceHolder = TextFCPlaceholder

	ui.LocalPort.SetPlaceHolder(DefaultPort)
	ui.LocalPort.SetText(DefaultPort)
}

func (ui *UI) SetupBindings() {

}
