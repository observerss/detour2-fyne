package run

import (
	"fmt"
	"log"
	"os"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	lo "github.com/observerss/detour2-fyne/layout"
	"github.com/observerss/detour2-fyne/profile"
	"github.com/observerss/detour2-fyne/run/proxy"
	"github.com/observerss/detour2-fyne/run/startup"
	"github.com/observerss/detour2/common"
	"github.com/observerss/detour2/deploy"
	"github.com/observerss/detour2/local"
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
	TextExit          = "退出程序"
	MAX_LINES         = 50
	HTTP_PORT         = "33331"
)

type UI struct {
	ProfileSelect  *widget.Select
	LocalPort      *widget.Entry
	RunOnStartup   *widget.Check
	UseGlobalProxy *widget.Check
	ToggleRun      *widget.Button
	Exit           *widget.Button
	LogEntry       *widget.Entry
	Parent         fyne.Window
	Started        bool
	Logs           *clog
	Socks5Server   *local.Local
	HTTPServer     *local.Local
	CanvasObject   fyne.CanvasObject
}

func NewUI(parent fyne.Window) *UI {
	ui := &UI{
		ProfileSelect:  widget.NewSelect([]string{}, func(s string) {}),
		LocalPort:      widget.NewEntry(),
		RunOnStartup:   widget.NewCheck(TextRunOnStartup, func(b bool) {}),
		UseGlobalProxy: widget.NewCheck(TextGlobalProxy, func(b bool) {}),
		ToggleRun:      widget.NewButton(TextStartFC, func() {}),
		Exit:           widget.NewButton(TextExit, func() { os.Exit(0) }),
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
		ui.RunOnStartup, layout.NewSpacer(),
	)

	submit := container.NewCenter(container.NewHBox(ui.ToggleRun, layout.NewSpacer(), ui.Exit))

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

	ui.CanvasObject = split
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
	ui.LoadRun()
}

func (ui *UI) SetupBindings() {
	ui.ToggleRun.OnTapped = ui.HandleToggleRun
	ui.RunOnStartup.OnChanged = func(b bool) {
		if b {
			startup.Enable()
		} else {
			startup.Disable()
		}
		ui.SaveRun()
	}
	ui.UseGlobalProxy.OnChanged = func(b bool) {
		if b {
			proxyUrl := fmt.Sprintf("localhost:%s", HTTP_PORT)
			err := proxy.SetGlobalProxy(proxyUrl)
			if err != nil {
				logger.Error.Println(err.Error())
			}
		} else {
			err := proxy.Off()
			if err != nil {
				logger.Error.Println(err.Error())
			}
		}
		ui.SaveRun()
	}
}

func (ui *UI) HandleToggleRun() {
	logger.Info.SetOutput(ui.Logs)
	logger.Error.SetOutput(ui.Logs)

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
		logger.Error.Println(err.Error())
		ui.ToggleRun.Enable()
		return
	}

	// run local server
	cli, err := deploy.NewClient(conf)
	if err != nil {
		logger.Error.Println(err.Error())
		ui.ToggleRun.Enable()
		return
	}
	wsurl, err := cli.GetWebsocketURL()
	if err != nil {
		logger.Error.Println(err.Error())
		ui.ToggleRun.Enable()
		return
	}
	lconf := &common.LocalConfig{
		Listen:   fmt.Sprintf("tcp://localhost:%s", ui.LocalPort.Text),
		Remotes:  wsurl,
		Password: conf.Password,
		Proto:    "socks5",
	}
	lconf2 := &common.LocalConfig{
		Listen:   fmt.Sprintf("tcp://localhost:%s", HTTP_PORT),
		Remotes:  wsurl,
		Password: conf.Password,
		Proto:    "http",
	}
	ui.Socks5Server = local.NewLocal(lconf)
	ui.HTTPServer = local.NewLocal(lconf2)
	go func() {
		err := ui.Socks5Server.RunLocal()
		if err != nil {
			logger.Error.Println(err)
			ui.Started = false
			ui.ToggleRun.SetText(TextStartFC)
		}
	}()

	go func() {
		err := ui.HTTPServer.RunLocal()
		if err != nil {
			logger.Error.Println(err)
		}
	}()

	ui.ToggleRun.Enable()
	ui.SaveRun()
	ui.Started = true
	ui.ToggleRun.SetText(TextStopFC)
}

func (ui *UI) StopRunning() {
	ui.ToggleRun.Disable()

	// stop server first
	ui.Socks5Server.StopLocal()
	ui.HTTPServer.StopLocal()

	profs, _ := profile.LoadProfiles()
	names := profile.GetProfileNames(profs)
	name := names[ui.ProfileSelect.SelectedIndex()]
	prof := profs[name]
	conf := profile.ConvertProfileToConfig(prof)
	conf.Remove = true
	err := deploy.DeployServer(conf)
	if err != nil {
		logger.Error.Println(err.Error())
		ui.ToggleRun.Enable()
		return
	}

	ui.ToggleRun.Enable()
	ui.Started = false
	ui.ToggleRun.SetText(TextStartFC)
}

func (ui *UI) SaveRun() {
	run := &Run{
		ProfileName:    ui.ProfileSelect.Selected,
		LocalPort:      ui.LocalPort.Text,
		RunOnStartup:   ui.RunOnStartup.Checked,
		UseGlobalProxy: ui.UseGlobalProxy.Checked,
	}
	err := SaveRun(run)
	if err != nil {
		log.Println(err)
	}
}

func (ui *UI) LoadRun() {
	run, err := LoadRun()
	if err != nil {
		log.Println(err)
	}
	ui.ProfileSelect.SetSelected(run.ProfileName)
	ui.LocalPort.SetText(run.LocalPort)
	ui.RunOnStartup.SetChecked(run.RunOnStartup)
	ui.UseGlobalProxy.SetChecked(run.UseGlobalProxy)

	if ui.CanvasObject != nil {
		ui.CanvasObject.Refresh()
	}
}

type clog struct {
	Entry *widget.Entry
	Lines []string
}

func (l *clog) Write(p []byte) (int, error) {
	l.Lines = append(l.Lines, string(p))
	if len(l.Lines) > MAX_LINES {
		l.Lines = l.Lines[len(l.Lines)-MAX_LINES:]
	}
	l.Entry.SetText(strings.Join(l.Lines, ""))
	l.Entry.CursorRow = len(l.Lines) - 1
	l.Entry.Refresh()
	return 0, nil
}
