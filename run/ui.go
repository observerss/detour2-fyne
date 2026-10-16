package run

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	lo "github.com/observerss/detour2-fyne/layout"
	"github.com/observerss/detour2-fyne/profile"
	"github.com/observerss/detour2-fyne/run/proxy"
	"github.com/observerss/detour2-fyne/run/startup"
	"github.com/observerss/detour2/common"
	"github.com/observerss/detour2/local"
	"github.com/observerss/detour2/logger"
)

var (
	TextFCPlaceholder = "请选择上游配置"
	TextFCLabel       = "上游配置"
	TextSocks5Port    = "SOCKS5 代理端口"
	TextHTTPPort      = "HTTP 代理端口"
	TextPoolSize      = "WebSocket 连接池"
	TextProxyStatus   = "全局代理状态"
	DefaultSocks5Port = "7776"
	DefaultHTTPPort   = "7777"
	TextRunOnStartup  = "开机自动运行"
	TextGlobalProxy   = "启用全局代理"
	TextGlobalPending = "启动本地代理后将启用 Android VPN"
	TextStartFC       = "启动本地代理"
	TextStopFC        = "停止本地代理"
	TextExit          = "退出程序"
	MAX_LINES         = 50
	DefaultPoolSize   = 64
	TextEnableVPN     = "启用 VPN"
	TextDisableVPN    = "关闭 VPN"
	TextVPNHint       = "开启后自动启动本地转发并接管全局网络"
)

type UI struct {
	ProfileSelect  *widget.Select
	Socks5Port     *widget.Entry
	HTTPPort       *widget.Entry
	PoolSize       *widget.Entry
	RunOnStartup   *widget.Check
	UseGlobalProxy *widget.Check
	ProxyStatus    *widget.Label
	ToggleRun      *widget.Button
	EnableSwitch   *MobileSwitch
	Exit           *widget.Button
	LogEntry       *widget.Entry
	Parent         fyne.Window
	Started        bool
	ProxyApplied   bool // desktop: we currently own the system proxy setting
	Loading        bool // suppress OnChanged side effects while restoring saved state
	Logs           *clog
	Socks5Server   *local.Local
	HTTPServer     *local.Local
	CanvasObject   fyne.CanvasObject
	monitorOnce    sync.Once
	recoveringVPN  bool
	vpnRecoverLock sync.Mutex
}

func NewUI(parent fyne.Window) *UI {
	ui := &UI{
		ProfileSelect:  widget.NewSelect([]string{}, func(s string) {}),
		Socks5Port:     widget.NewEntry(),
		HTTPPort:       widget.NewEntry(),
		PoolSize:       widget.NewEntry(),
		RunOnStartup:   widget.NewCheck(TextRunOnStartup, func(b bool) {}),
		UseGlobalProxy: widget.NewCheck(TextGlobalProxy, func(b bool) {}),
		ProxyStatus:    widget.NewLabel(""),
		ToggleRun:      widget.NewButton(TextStartFC, func() {}),
		EnableSwitch:   NewMobileSwitch(func(b bool) {}),
		LogEntry:       widget.NewMultiLineEntry(),
		Parent:         parent,
	}
	ui.Exit = widget.NewButton(TextExit, func() {
		ui.Shutdown()
		fyne.CurrentApp().Quit()
	})
	ui.Logs = &clog{Entry: ui.LogEntry, Lines: make([]string, 0)}
	return ui
}

func (ui *UI) MakeUI() fyne.CanvasObject {
	ui.SetupBindings()
	ui.ResetUI()
	ui.startAndroidVPNMonitor()
	if mobileLayout() {
		ui.ProxyStatus.Wrapping = fyne.TextWrapWord
		fields := container.NewVBox(
			ui.mobileEnableCard(),
			mobileField(TextFCLabel, ui.ProfileSelect),
			mobileField(TextProxyStatus, ui.ProxyStatus),
			widget.NewSeparator(),
			mobileField(TextSocks5Port, ui.Socks5Port),
			mobileField(TextHTTPPort, ui.HTTPPort),
			mobileField(TextPoolSize, ui.PoolSize),
		)
		ui.CanvasObject = container.NewVScroll(lo.NewPaddingContainer(
			fields,
			&lo.Padding{Top: 16, Bottom: 16, Left: 16, Right: 16},
		))
		return ui.CanvasObject
	}

	grid := container.New(layout.NewFormLayout(),
		widget.NewLabel(TextFCLabel), ui.ProfileSelect,
		layout.NewSpacer(), layout.NewSpacer(),
		widget.NewLabel(TextSocks5Port), ui.Socks5Port,
		layout.NewSpacer(), layout.NewSpacer(),
		widget.NewLabel(TextHTTPPort), ui.HTTPPort,
		layout.NewSpacer(), layout.NewSpacer(),
		widget.NewLabel(TextPoolSize), ui.PoolSize,
		layout.NewSpacer(), layout.NewSpacer(),
		ui.RunOnStartup, ui.UseGlobalProxy,
		widget.NewLabel(TextProxyStatus), ui.ProxyStatus,
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

func (ui *UI) MakeLogUI() fyne.CanvasObject {
	ui.LogEntry.SetMinRowsVisible(16)
	ui.LogEntry.Wrapping = fyne.TextWrapWord
	return lo.NewPaddingContainer(
		container.NewMax(ui.LogEntry),
		&lo.Padding{Top: 12, Bottom: 12, Left: 12, Right: 12},
	)
}

func (ui *UI) mobileEnableCard() fyne.CanvasObject {
	title := widget.NewLabelWithStyle("Detour2 VPN", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	hint := widget.NewLabel(TextVPNHint)
	hint.Wrapping = fyne.TextWrapWord
	statusDot := canvas.NewCircle(theme.PrimaryColor())
	statusDot.Resize(fyne.NewSize(14, 14))
	switchBox := container.NewCenter(ui.EnableSwitch)
	return widget.NewCard("", "", container.NewVBox(
		container.NewBorder(nil, nil, container.NewHBox(statusDot, title), switchBox),
		hint,
	))
}

func mobileField(label string, object fyne.CanvasObject) fyne.CanvasObject {
	return container.NewVBox(widget.NewLabel(label), object)
}

func (ui *UI) ResetUI() {
	ui.ProfileSelect.PlaceHolder = TextFCPlaceholder
	ui.RefreshProfiles()
	ui.Socks5Port.SetPlaceHolder(DefaultSocks5Port)
	ui.Socks5Port.SetText(DefaultSocks5Port)
	ui.HTTPPort.SetPlaceHolder(DefaultHTTPPort)
	ui.HTTPPort.SetText(DefaultHTTPPort)
	ui.PoolSize.SetPlaceHolder(strconv.Itoa(DefaultPoolSize))
	ui.PoolSize.SetText(strconv.Itoa(DefaultPoolSize))
	ui.LoadRun()
}

// RefreshProfiles reloads the profile names (they may have been edited on the
// other tab) while keeping the current selection and the rest of the form.
func (ui *UI) RefreshProfiles() {
	profs, err := profile.LoadProfiles()
	if err != nil {
		logger.Error.Println(err.Error())
	}
	names := profile.GetProfileNames(profs)
	ui.ProfileSelect.SetOptions(names)
	if len(names) == 0 {
		ui.ProfileSelect.ClearSelected()
		return
	}
	selected := profile.NormalizeProfileName(ui.ProfileSelect.Selected)
	if _, ok := profs[selected]; !ok {
		selected = profile.DefaultProfileName
		if _, ok := profs[selected]; !ok {
			selected = names[0]
		}
	}
	ui.ProfileSelect.SetSelected(selected)
}

func (ui *UI) SetupBindings() {
	ui.ToggleRun.OnTapped = ui.HandleToggleRun
	ui.EnableSwitch.OnChanged = ui.HandleMobileEnableChanged
	ui.RunOnStartup.OnChanged = func(b bool) {
		if ui.Loading {
			return
		}
		if !supportsStartup() {
			ui.RunOnStartup.SetChecked(false)
			return
		}
		ui.SaveRun()
		ui.RunOnStartup.Disable()
		go func() {
			var err error
			if b {
				err = startup.Enable()
			} else {
				err = startup.Disable()
			}
			fyne.Do(func() {
				ui.RunOnStartup.Enable()
				if err == nil {
					return
				}
				logger.Error.Println("设置开机自动运行失败:", err)
				ui.Loading = true
				ui.RunOnStartup.SetChecked(!b)
				ui.Loading = false
				ui.SaveRun()
			})
		}()
	}
	ui.UseGlobalProxy.OnChanged = func(b bool) {
		if ui.Loading {
			return
		}
		if !supportsGlobalProxy() {
			ui.UseGlobalProxy.SetChecked(false)
			return
		}
		if !mobileLayout() {
			// desktop: the system proxy only points at us while we run
			ui.SyncGlobalProxy()
			ui.UpdateProxyStatus()
			ui.SaveRun()
			return
		}
		if runtime.GOOS == "android" && b && !ui.Started {
			ui.ProxyStatus.SetText(TextGlobalPending)
			ui.SaveRun()
			return
		}
		socks5Port, httpPort, err := ui.proxyPorts()
		if err != nil {
			logger.Error.Println(err.Error())
			return
		}
		if b {
			if runtime.GOOS == "ios" && ui.Started {
				ui.stopLocalServers()
			}
			err = ui.startGlobalProxy(socks5Port, httpPort)
			if err != nil {
				logger.Error.Println(err.Error())
			} else if runtime.GOOS == "ios" {
				ui.Started = true
				ui.ToggleRun.SetText(TextStopFC)
			}
		} else {
			err := proxy.Off()
			if err != nil {
				logger.Error.Println(err.Error())
			} else if runtime.GOOS == "ios" {
				ui.Started = false
				ui.ToggleRun.SetText(TextStartFC)
			}
		}
		ui.UpdateProxyStatus()
		ui.refreshAndroidProxyToggle()
		ui.SaveRun()
	}
	if !supportsStartup() {
		ui.RunOnStartup.Disable()
	}
	if !supportsGlobalProxy() {
		ui.UseGlobalProxy.Disable()
	}
	ui.UpdateProxyStatus()
}

// AttachLogger mirrors detour2 logs into the log panel for the app's lifetime.
// The panel goes first: on Windows GUI builds stdout is invalid and
// io.MultiWriter stops at the first failing writer.
func (ui *UI) AttachLogger() {
	logger.Info.SetOutput(io.MultiWriter(ui.Logs, os.Stdout))
	logger.Warn.SetOutput(io.MultiWriter(ui.Logs, os.Stdout))
	logger.Error.SetOutput(io.MultiWriter(ui.Logs, os.Stderr))
}

// DetachLogger sends logs back to stdio once the UI loop has ended.
func (ui *UI) DetachLogger() {
	logger.Info.SetOutput(os.Stdout)
	logger.Warn.SetOutput(os.Stdout)
	logger.Error.SetOutput(os.Stderr)
}

func (ui *UI) HandleToggleRun() {
	if !ui.Started {
		ui.StartRunning()
	} else {
		ui.StopRunning()
	}
}

func (ui *UI) HandleMobileEnableChanged(enabled bool) {
	if !mobileLayout() {
		return
	}
	if enabled == ui.Started {
		return
	}
	if enabled {
		ui.UseGlobalProxy.Checked = true
		ui.UseGlobalProxy.Refresh()
		ui.StartRunning()
		return
	}
	ui.StopRunning()
}

func (ui *UI) StartRunning() {
	ui.ToggleRun.Disable()
	if ui.EnableSwitch != nil {
		ui.EnableSwitch.Disable()
	}
	defer func() {
		if ui.EnableSwitch != nil {
			ui.EnableSwitch.Enable()
		}
	}()

	prof, err := ui.SelectedProfile()
	if err != nil {
		logger.Error.Println(err.Error())
		ui.ToggleRun.Enable()
		ui.setEnableSwitchChecked(false)
		return
	}
	remotes, err := profile.NormalizeRemoteAddress(prof.RemoteAddress)
	if err != nil {
		logger.Error.Println(err.Error())
		ui.ToggleRun.Enable()
		ui.setEnableSwitchChecked(false)
		return
	}
	socks5Port, httpPort, err := ui.proxyPorts()
	if err != nil {
		logger.Error.Println(err.Error())
		ui.ToggleRun.Enable()
		ui.setEnableSwitchChecked(false)
		return
	}
	poolSize, err := ui.poolSize()
	if err != nil {
		logger.Error.Println(err.Error())
		ui.ToggleRun.Enable()
		ui.setEnableSwitchChecked(false)
		return
	}
	if prof.Password == "" {
		logger.Error.Println("上游配置缺少密码:", prof.Name)
		ui.ToggleRun.Enable()
		ui.setEnableSwitchChecked(false)
		return
	}
	if runtime.GOOS == "ios" && ui.UseGlobalProxy.Checked {
		if err := ui.startGlobalProxy(socks5Port, httpPort); err != nil {
			logger.Error.Println(err.Error())
			ui.ToggleRun.Enable()
			ui.setEnableSwitchChecked(false)
			return
		}
		ui.UpdateProxyStatus()
		ui.ToggleRun.Enable()
		ui.setEnableSwitchChecked(true)
		ui.SaveRun()
		ui.Started = true
		ui.ToggleRun.SetText(TextStopFC)
		return
	}

	lconf := &common.LocalConfig{
		Listen:   fmt.Sprintf("tcp://127.0.0.1:%s", socks5Port),
		Remotes:  remotes,
		Password: prof.Password,
		Proto:    "socks5",
		PoolSize: poolSize,
	}
	lconf2 := &common.LocalConfig{
		Listen:   fmt.Sprintf("tcp://127.0.0.1:%s", httpPort),
		Remotes:  remotes,
		Password: prof.Password,
		Proto:    "http",
		PoolSize: poolSize,
	}
	ui.Socks5Server = local.NewLocal(lconf)
	ui.HTTPServer = local.NewLocal(lconf2)
	ui.Started = true
	go ui.serve(ui.Socks5Server)
	go ui.serve(ui.HTTPServer)
	if !mobileLayout() {
		ui.SyncGlobalProxy()
		ui.UpdateProxyStatus()
	} else if ui.UseGlobalProxy.Checked && supportsGlobalProxy() {
		if runtime.GOOS == "android" {
			if err := waitForTCPListener(net.JoinHostPort("127.0.0.1", socks5Port), 3*time.Second); err != nil {
				// stop everything, so Started matches the switch and a
				// later tap can start again
				logger.Error.Println(err.Error())
				ui.SaveRun()
				ui.StopRunning()
				ui.ProxyStatus.SetText(err.Error())
				return
			}
		}
		if err := ui.startGlobalProxy(socks5Port, httpPort); err != nil {
			logger.Error.Println(err.Error())
		}
		ui.UpdateProxyStatus()
		ui.refreshAndroidProxyToggle()
	}

	ui.ToggleRun.Enable()
	ui.setEnableSwitchChecked(true)
	ui.SaveRun()
	ui.Started = true
	ui.ToggleRun.SetText(TextStopFC)
	ui.startAndroidVPNMonitor()
}

func (ui *UI) StopRunning() {
	ui.ToggleRun.Disable()
	if ui.EnableSwitch != nil {
		ui.EnableSwitch.Disable()
	}
	defer func() {
		if ui.EnableSwitch != nil {
			ui.EnableSwitch.Enable()
		}
	}()

	ui.Started = false
	ui.recoveringVPN = false
	ui.stopLocalServers()
	if !mobileLayout() {
		ui.SyncGlobalProxy()
		ui.UpdateProxyStatus()
	} else if ui.UseGlobalProxy.Checked && supportsGlobalProxy() {
		if err := proxy.Off(); err != nil {
			logger.Error.Println(err.Error())
		}
		ui.UpdateProxyStatus()
	}

	ui.ToggleRun.Enable()
	ui.setEnableSwitchChecked(false)
	ui.ToggleRun.SetText(TextStartFC)
}

// serve runs srv until it exits. If it dies on its own (e.g. the port is in
// use) the whole pair is stopped, so we never keep a half-working proxy or a
// system proxy pointing at a dead port.
func (ui *UI) serve(srv *local.Local) {
	err := srv.RunLocal()
	if srv.IsStopped() {
		return
	}
	if err == nil {
		err = fmt.Errorf("本地代理意外退出: %s://%s", srv.Network, srv.Address)
	}
	logger.Error.Println(err)
	fyne.Do(func() {
		if ui.Socks5Server == srv || ui.HTTPServer == srv {
			ui.StopRunning()
		}
	})
}

// Shutdown stops the local proxies and restores the system proxy. It does not
// touch widgets, so it is safe after the UI loop has ended, and safe to call
// more than once. The iOS tunnel lives in its own extension and is left alone.
func (ui *UI) Shutdown() {
	wasRunning := ui.Started
	ui.Started = false
	ui.stopLocalServers()
	switch {
	case !mobileLayout():
		ui.SyncGlobalProxy()
	case runtime.GOOS == "android" && wasRunning && ui.UseGlobalProxy.Checked:
		// the VPN forwards into the local servers we just stopped
		if err := proxy.Off(); err != nil {
			logger.Error.Println(err.Error())
		}
	}
}

// SyncGlobalProxy (desktop) makes the system proxy point at us only while the
// local proxies run and the user asked for it, and restores it otherwise.
func (ui *UI) SyncGlobalProxy() {
	want := ui.Started && ui.UseGlobalProxy.Checked && supportsGlobalProxy()
	if want == ui.ProxyApplied {
		return
	}
	if want {
		socks5Port, httpPort, err := ui.proxyPorts()
		if err == nil {
			err = ui.startGlobalProxy(socks5Port, httpPort)
		}
		if err != nil {
			logger.Error.Println(err.Error())
			return
		}
		ui.ProxyApplied = true
		return
	}
	if err := proxy.Off(); err != nil {
		logger.Error.Println(err.Error())
		return
	}
	ui.ProxyApplied = false
}

func (ui *UI) SaveRun() {
	run := &Run{
		ProfileName:    ui.ProfileSelect.Selected,
		Socks5Port:     strings.TrimSpace(ui.Socks5Port.Text),
		HTTPPort:       strings.TrimSpace(ui.HTTPPort.Text),
		PoolSize:       parsePoolSizeForSave(ui.PoolSize.Text),
		RunOnStartup:   ui.RunOnStartup.Checked,
		UseGlobalProxy: mobileLayout() || ui.UseGlobalProxy.Checked,
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
		return
	}
	ui.Loading = true
	defer func() { ui.Loading = false }()
	if run.ProfileName != "" {
		ui.ProfileSelect.SetSelected(profile.NormalizeProfileName(run.ProfileName))
	}
	if run.Socks5Port != "" {
		ui.Socks5Port.SetText(run.Socks5Port)
	} else if run.LocalPort != "" {
		ui.Socks5Port.SetText(run.LocalPort)
	}
	if run.HTTPPort != "" {
		ui.HTTPPort.SetText(run.HTTPPort)
	}
	if run.PoolSize > 0 {
		ui.PoolSize.SetText(strconv.Itoa(run.PoolSize))
	}
	if supportsStartup() {
		ui.RunOnStartup.SetChecked(run.RunOnStartup)
	}
	if supportsGlobalProxy() {
		if runtime.GOOS == "android" {
			ui.setUseGlobalProxyChecked(true)
		} else {
			ui.UseGlobalProxy.SetChecked(run.UseGlobalProxy)
		}
	}
	ui.UpdateProxyStatus()
	if runtime.GOOS == "android" && run.UseGlobalProxy && !ui.Started {
		ui.ProxyStatus.SetText(TextGlobalPending)
	}

	if ui.CanvasObject != nil {
		ui.CanvasObject.Refresh()
	}
}

func (ui *UI) proxyPorts() (string, string, error) {
	socks5Port := strings.TrimSpace(ui.Socks5Port.Text)
	if socks5Port == "" {
		socks5Port = DefaultSocks5Port
		ui.Socks5Port.SetText(socks5Port)
	}
	httpPort := strings.TrimSpace(ui.HTTPPort.Text)
	if httpPort == "" {
		httpPort = DefaultHTTPPort
		ui.HTTPPort.SetText(httpPort)
	}
	if err := validatePort(TextSocks5Port, socks5Port); err != nil {
		return "", "", err
	}
	if err := validatePort(TextHTTPPort, httpPort); err != nil {
		return "", "", err
	}
	if socks5Port == httpPort {
		return "", "", fmt.Errorf("SOCKS5 和 HTTP 代理端口不能相同")
	}
	return socks5Port, httpPort, nil
}

func validatePort(label string, value string) error {
	port, err := strconv.Atoi(value)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("%s 必须是 1-65535 之间的数字", label)
	}
	return nil
}

func (ui *UI) poolSize() (int, error) {
	value := strings.TrimSpace(ui.PoolSize.Text)
	if value == "" {
		value = strconv.Itoa(DefaultPoolSize)
		ui.PoolSize.SetText(value)
	}
	poolSize, err := strconv.Atoi(value)
	if err != nil || poolSize < 1 {
		return 0, fmt.Errorf("%s 必须是大于 0 的数字", TextPoolSize)
	}
	return poolSize, nil
}

func parsePoolSizeForSave(value string) int {
	poolSize, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || poolSize < 1 {
		return DefaultPoolSize
	}
	return poolSize
}

func waitForTCPListener(address string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		conn, err := net.DialTimeout("tcp", address, 200*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("本地 SOCKS5 代理未就绪: %s", address)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func (ui *UI) setUseGlobalProxyChecked(checked bool) {
	ui.UseGlobalProxy.Checked = checked
	ui.UseGlobalProxy.Refresh()
}

func (ui *UI) setEnableSwitchChecked(checked bool) {
	if ui.EnableSwitch == nil {
		return
	}
	ui.EnableSwitch.SetChecked(checked)
	ui.EnableSwitch.Refresh()
}

func (ui *UI) SelectedProfile() (*profile.Profile, error) {
	profs, err := profile.LoadProfiles()
	if err != nil {
		return nil, err
	}
	name := ui.ProfileSelect.Selected
	if name == "" {
		names := profile.GetProfileNames(profs)
		if len(names) == 0 {
			return nil, fmt.Errorf("请先保存一个上游配置")
		}
		name = profile.DefaultProfileName
		if _, ok := profs[name]; !ok {
			name = names[0]
		}
		ui.ProfileSelect.SetSelected(name)
	}
	prof, ok := profs[name]
	if !ok || prof == nil {
		return nil, fmt.Errorf("找不到上游配置: %s", name)
	}
	return prof, nil
}

func (ui *UI) UpdateProxyStatus() {
	if ui.ProxyStatus == nil {
		return
	}
	if !supportsGlobalProxy() {
		ui.ProxyStatus.SetText("当前平台暂不支持全局代理")
		return
	}
	ui.ProxyStatus.SetText(proxy.Status())
}

func (ui *UI) startGlobalProxy(socks5Port string, httpPort string) error {
	proxyURL, err := ui.globalProxyConfig(socks5Port, httpPort)
	if err != nil {
		return err
	}
	return proxy.SetGlobalProxy(proxyURL)
}

func (ui *UI) stopLocalServers() {
	if ui.Socks5Server != nil {
		ui.Socks5Server.StopLocal()
		ui.Socks5Server = nil
	}
	if ui.HTTPServer != nil {
		ui.HTTPServer.StopLocal()
		ui.HTTPServer = nil
	}
}

func (ui *UI) refreshAndroidProxyToggle() {
	if runtime.GOOS != "android" || ui.UseGlobalProxy == nil {
		return
	}
	if strings.Contains(proxy.Status(), "需要授权") {
		ui.UseGlobalProxy.Checked = false
		ui.UseGlobalProxy.Refresh()
		return
	}
	go func() {
		time.Sleep(800 * time.Millisecond)
		fyne.Do(func() {
			ui.UpdateProxyStatus()
		})
	}()
}

func (ui *UI) startAndroidVPNMonitor() {
	if runtime.GOOS != "android" {
		return
	}
	ui.monitorOnce.Do(func() {
		go ui.monitorAndroidVPN()
	})
}

func (ui *UI) monitorAndroidVPN() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		if !ui.Started || !ui.UseGlobalProxy.Checked {
			continue
		}
		status := proxy.Status()
		if androidVPNStatusNeedsRecovery(status) {
			ui.recoverAndroidVPN(status)
			continue
		}
		fyne.Do(func() {
			ui.UpdateProxyStatus()
		})
	}
}

func androidVPNStatusNeedsRecovery(status string) bool {
	return strings.Contains(status, "已停止") ||
		strings.Contains(status, "已被系统断开") ||
		strings.Contains(status, "启动失败") ||
		strings.Contains(status, "未启用")
}

func (ui *UI) recoverAndroidVPN(status string) {
	ui.vpnRecoverLock.Lock()
	if ui.recoveringVPN {
		ui.vpnRecoverLock.Unlock()
		return
	}
	ui.recoveringVPN = true
	ui.vpnRecoverLock.Unlock()

	defer func() {
		ui.vpnRecoverLock.Lock()
		ui.recoveringVPN = false
		ui.vpnRecoverLock.Unlock()
	}()

	logger.Error.Println("Android VPN 状态异常，尝试自动恢复: " + status)
	socks5Port, httpPort, err := ui.proxyPorts()
	if err != nil {
		logger.Error.Println(err.Error())
		ui.syncAndroidVPNStopped(err.Error())
		return
	}
	if err := waitForTCPListener(net.JoinHostPort("127.0.0.1", socks5Port), 2*time.Second); err != nil {
		logger.Error.Println(err.Error())
		ui.syncAndroidVPNStopped(err.Error())
		return
	}
	if err := ui.startGlobalProxy(socks5Port, httpPort); err != nil {
		logger.Error.Println(err.Error())
		ui.syncAndroidVPNStopped(err.Error())
		return
	}
	go func() {
		time.Sleep(800 * time.Millisecond)
		fyne.Do(func() {
			ui.UpdateProxyStatus()
			ui.setEnableSwitchChecked(true)
			ui.ToggleRun.SetText(TextStopFC)
		})
	}()
}

func (ui *UI) syncAndroidVPNStopped(reason string) {
	fyne.Do(func() {
		ui.Started = false
		ui.setEnableSwitchChecked(false)
		ui.ToggleRun.SetText(TextStartFC)
		ui.ProxyStatus.SetText("Android VPN 已断开: " + reason)
	})
}

func supportsGlobalProxy() bool {
	switch runtime.GOOS {
	case "android", "darwin", "ios", "windows":
		return true
	default:
		return false
	}
}

func (ui *UI) globalProxyConfig(socks5Port string, httpPort string) (string, error) {
	if runtime.GOOS != "ios" {
		return globalProxyEndpoint(socks5Port, httpPort), nil
	}
	prof, err := ui.SelectedProfile()
	if err != nil {
		return "", err
	}
	remoteAddress, err := profile.NormalizeRemoteAddress(prof.RemoteAddress)
	if err != nil {
		return "", err
	}
	poolSize, err := ui.poolSize()
	if err != nil {
		return "", err
	}
	values := url.Values{}
	values.Set("providerBundleID", "github.com.observerss.detour2fyne.PacketTunnel")
	values.Set("remoteAddress", remoteAddress)
	values.Set("password", prof.Password)
	values.Set("mtu", "1500")
	values.Set("socks5Port", socks5Port)
	values.Set("poolSize", strconv.Itoa(poolSize))
	return values.Encode(), nil
}

func globalProxyEndpoint(socks5Port string, httpPort string) string {
	switch runtime.GOOS {
	case "android":
		return fmt.Sprintf("127.0.0.1:%s", socks5Port)
	case "darwin":
		return fmt.Sprintf("http=127.0.0.1:%s;socks5=127.0.0.1:%s", httpPort, socks5Port)
	default:
		return fmt.Sprintf("127.0.0.1:%s", httpPort)
	}
}

func mobileLayout() bool {
	switch runtime.GOOS {
	case "android", "ios":
		return true
	default:
		return false
	}
}

func supportsStartup() bool {
	return runtime.GOOS == "windows"
}

type clog struct {
	Entry *widget.Entry
	Lines []string
	Lock  sync.Mutex
}

func (l *clog) Write(p []byte) (int, error) {
	l.Lock.Lock()
	l.Lines = append(l.Lines, string(p))
	if len(l.Lines) > MAX_LINES {
		l.Lines = l.Lines[len(l.Lines)-MAX_LINES:]
	}
	text := strings.Join(l.Lines, "")
	row := len(l.Lines) - 1
	l.Lock.Unlock()
	fyne.Do(func() {
		if l.Entry == nil {
			return
		}
		l.Entry.SetText(text)
		l.Entry.CursorRow = row
		l.Entry.Refresh()
	})
	return len(p), nil
}
