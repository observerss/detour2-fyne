package profile

import (
	"fmt"
	"log"
	"runtime"
	"sort"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/gorilla/websocket"
	lo "github.com/observerss/detour2-fyne/layout"
	"github.com/observerss/detour2/logger"
)

type UI struct {
	Left          *widget.List
	Name          *widget.Entry
	RemoteAddress *widget.Entry
	Password      *widget.Entry
	TestStatus    *widget.Label
	New           *widget.Button
	Delete        *widget.Button
	Reset         *widget.Button
	Test          *widget.Button
	Save          *widget.Button
	Profs         map[string]*Profile
	Parent        fyne.Window
	CurrentIdx    int
	OriginalName  string
}

func NewUI(parent fyne.Window) *UI {
	remoteAddress := widget.NewMultiLineEntry()
	remoteAddress.SetMinRowsVisible(3)
	return &UI{
		Left:          &widget.List{},
		Name:          widget.NewEntry(),
		RemoteAddress: remoteAddress,
		Password:      widget.NewPasswordEntry(),
		Parent:        parent,
	}
}

func (ui *UI) MakeUI() fyne.CanvasObject {
	ui.TestStatus = widget.NewLabel("请选择或新建一个配置")
	ui.New = widget.NewButton("新建", ui.HandleNew)
	ui.Delete = widget.NewButton("删除", ui.HandleDelete)
	ui.Reset = widget.NewButton("重置", ui.ResetForm)
	ui.Test = widget.NewButton("测试连接", ui.HandleTest)
	ui.Save = widget.NewButton("保存", ui.HandleSave)
	ui.TestStatus.Wrapping = fyne.TextWrapWord

	ui.SetupBindings()
	ui.ResetForm()
	ui.ResetLeft()
	if profileMobileLayout() {
		listTitle := widget.NewLabelWithStyle("配置线路", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
		listBox := container.New(lo.NewFixedHeight(profileMobileListHeight()), ui.Left)
		form := container.NewVBox(
			mobileProfileField("配置名 *", ui.Name),
			mobileProfileField("上游地址 *", ui.RemoteAddress),
			mobileProfileField("密码 *", ui.Password),
		)
		buttons := container.NewVBox(
			container.NewGridWithColumns(2, ui.New, ui.Delete),
			container.NewGridWithColumns(2, ui.Reset, ui.Test),
			ui.Save,
		)
		content := container.NewVBox(
			listTitle,
			listBox,
			widget.NewSeparator(),
			form,
			ui.TestStatus,
			buttons,
		)
		return container.NewVScroll(lo.NewPaddingContainer(
			content,
			&lo.Padding{Top: 12, Bottom: 12, Left: 12, Right: 12},
		))
	}

	form := lo.NewPaddingContainer(container.New(layout.NewFormLayout(),
		widget.NewLabel("配置名 *"), ui.Name,
		widget.NewLabel("上游地址 *"), ui.RemoteAddress,
		widget.NewLabel("密码 *"), ui.Password,
	), &lo.Padding{Right: 10})
	buttons := container.NewCenter(container.NewHBox(ui.New, layout.NewSpacer(), ui.Delete, layout.NewSpacer(), ui.Reset, layout.NewSpacer(), ui.Test, layout.NewSpacer(), ui.Save))
	right := container.NewVBox(layout.NewSpacer(), form, layout.NewSpacer(), ui.TestStatus, layout.NewSpacer(), buttons, layout.NewSpacer())

	split := container.NewHSplit(container.NewHScroll(ui.Left), container.NewHScroll(right))
	split.SetOffset(0.2)

	return split
}

func mobileProfileField(label string, object fyne.CanvasObject) fyne.CanvasObject {
	return container.NewVBox(widget.NewLabel(label), object)
}

func profileMobileLayout() bool {
	switch runtime.GOOS {
	case "android", "ios":
		return true
	default:
		return false
	}
}

func profileMobileListHeight() float32 {
	return widget.NewLabel("线路").MinSize().Height*4 + theme.Padding()*8
}

func GetProfileNames(profs map[string]*Profile) []string {
	names := make([]string, 0, len(profs))
	for name, prof := range profs {
		name = NormalizeProfileName(name)
		if name == "" || prof == nil {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (ui *UI) SetupBindings() {
	ui.Left.OnSelected = func(id int) {
		ui.CurrentIdx = id
		names := GetProfileNames(ui.Profs)
		if len(names) > id {
			prof, ok := ui.Profs[names[id]]
			if ok {
				ui.SetForm(prof)
				if err := SetCurrentProfileName(names[id]); err != nil {
					logger.Error.Println(err.Error())
				}
			}
		}
	}
}

func (ui *UI) ResetLeft() {
	profs, err := LoadProfiles()
	if err != nil {
		logger.Error.Println(err.Error())
		profs = EmptyProfile
	}

	ui.Profs = profs
	names := GetProfileNames(profs)
	ui.Left.Length = func() int {
		return len(names)
	}
	ui.Left.CreateItem = func() fyne.CanvasObject {
		return widget.NewLabel("template")
	}
	ui.Left.UpdateItem = func(i widget.ListItemID, o fyne.CanvasObject) {
		o.(*widget.Label).SetText(names[i])
	}
	if ui.CurrentIdx >= len(names) {
		ui.CurrentIdx = len(names) - 1
	}
	if ui.CurrentIdx < 0 && len(names) > 0 {
		ui.CurrentIdx = indexOfProfileName(names, DefaultProfileName)
	}

	if ui.CurrentIdx >= 0 {
		ui.Left.Select(ui.CurrentIdx)
	}
}

func indexOfProfileName(names []string, name string) int {
	for i, item := range names {
		if item == name {
			return i
		}
	}
	return 0
}

func (ui *UI) ResetForm() {
	prof := NewDefaultProfile()
	ui.SetForm(prof)
	ui.RemoteAddress.SetPlaceHolder("tcp://host:port")
	ui.Password.SetPlaceHolder("请输入密码")
	ui.TestStatus.SetText("请选择或新建一个配置")
}

func (ui *UI) HandleNew() {
	profs, err := LoadProfiles()
	if err != nil {
		ui.PromptDialog("新建失败", err.Error(), func(bool) {})
		return
	}
	prof := NewDefaultProfile()
	prof.Name = nextNewProfileName(profs)
	profs[prof.Name] = prof
	if err := SaveProfiles(profs); err != nil {
		ui.PromptDialog("新建失败", err.Error(), func(bool) {})
		return
	}
	ui.Profs = profs
	ui.CurrentIdx = -1
	ui.ResetLeft()
	ui.Left.Refresh()
	ui.Left.UnselectAll()
	ui.selectProfile(prof.Name)
	ui.TestStatus.SetText("已新建配置，请编辑后保存或测试连接")
}

func (ui *UI) SetForm(prof *Profile) {
	if prof == nil {
		prof = NewDefaultProfile()
	}
	ui.OriginalName = prof.Name
	ui.Name.SetText(prof.Name)
	ui.RemoteAddress.SetText(prof.RemoteAddress)
	ui.Password.SetText(prof.Password)
}

func (ui *UI) ValidateForm() error {
	if strings.TrimSpace(ui.Name.Text) == "" {
		return fmt.Errorf("配置名不能为空")
	}
	if strings.TrimSpace(ui.Password.Text) == "" {
		return fmt.Errorf("密码不能为空")
	}
	_, err := NormalizeRemoteAddress(ui.RemoteAddress.Text)
	return err
}

func (ui *UI) GetForm() (*Profile, error) {
	if err := ui.ValidateForm(); err != nil {
		return nil, err
	}

	prof := &Profile{
		Name:          strings.TrimSpace(ui.Name.Text),
		RemoteAddress: strings.TrimSpace(ui.RemoteAddress.Text),
		Password:      ui.Password.Text,
	}
	return prof, nil
}

func (ui *UI) HandleTest() {
	prof, err := ui.GetForm()
	if err != nil {
		ui.PromptDialog("检查失败", err.Error(), func(bool) {})
		return
	}
	remotes, err := NormalizeRemoteAddress(prof.RemoteAddress)
	if err != nil {
		ui.PromptDialog("检查失败", err.Error(), func(bool) {})
		return
	}
	ui.Test.Disable()
	ui.TestStatus.SetText("正在测试上游连接...")
	go func() {
		err := testRemoteConnections(remotes)
		title := "测试成功"
		message := "上游 WebSocket 握手成功: " + remotes
		status := "测试成功：上游 WebSocket 握手成功"
		if err != nil {
			title = "测试失败"
			message = err.Error()
			status = "测试失败：" + err.Error()
		}
		fyne.Do(func() {
			ui.Test.Enable()
			ui.TestStatus.SetText(status)
			ui.PromptDialog(title, message, func(bool) {})
		})
	}()
}

func (ui *UI) HandleSave() {
	prof, err := ui.GetForm()
	if err != nil {
		ui.PromptDialog("保存失败", err.Error(), func(bool) {})
		return
	}

	profs, err := LoadProfiles()
	if err != nil || profs == nil {
		profs = make(map[string]*Profile)
	}
	if existing, ok := profs[prof.Name]; ok && (ui.OriginalName == "" || ui.OriginalName != prof.Name) && existing != nil {
		ui.PromptDialog("保存失败", fmt.Sprintf("配置名已存在: %s", prof.Name), func(bool) {})
		return
	}
	if ui.OriginalName != "" && ui.OriginalName != prof.Name {
		delete(profs, ui.OriginalName)
	}
	profs[prof.Name] = prof

	if err = SaveProfiles(profs); err != nil {
		ui.PromptDialog("保存失败", err.Error(), func(bool) {})
		return
	}
	if err := SetCurrentProfileName(prof.Name); err != nil {
		ui.PromptDialog("保存失败", err.Error(), func(bool) {})
		return
	}

	ui.Profs = profs
	ui.ResetLeft()
	ui.Left.Refresh()
	ui.selectProfile(prof.Name)
	ui.TestStatus.SetText("已保存，并设为当前上游配置")

	ui.PromptDialog("保存成功", "数据已保存，并设为当前上游配置", func(bool) {})
}

func (ui *UI) HandleDelete() {
	name := ui.OriginalName
	if name == "" {
		name = ui.selectedProfileName()
	}
	if name == "" {
		ui.PromptDialog("错误", "无可删除项", func(bool) {})
		return
	}
	prof, ok := ui.Profs[name]
	if !ok {
		log.Println("profile should exist")
		return
	}

	ui.PromptDialog("确认", fmt.Sprintf("请确认是否删除 %s", prof.Name), func(b bool) {
		if b {
			delete(ui.Profs, name)
			SaveProfiles(ui.Profs)
			ui.ResetLeft()
			ui.Left.Refresh()
			names := GetProfileNames(ui.Profs)
			if len(names) > 0 {
				ui.selectProfile(names[ui.CurrentIdx])
			} else {
				ui.ResetForm()
				ui.Left.UnselectAll()
				_ = SetCurrentProfileName("")
			}
		}
	})
}

func (ui *UI) selectedProfileName() string {
	names := GetProfileNames(ui.Profs)
	if ui.CurrentIdx >= 0 && ui.CurrentIdx < len(names) {
		return names[ui.CurrentIdx]
	}
	return ""
}

func (ui *UI) selectProfile(name string) {
	names := GetProfileNames(ui.Profs)
	for i, item := range names {
		if item == name {
			ui.CurrentIdx = i
			ui.Left.Select(i)
			return
		}
	}
}

func nextNewProfileName(profs map[string]*Profile) string {
	for idx := 1; ; idx++ {
		candidate := fmt.Sprintf("new%d", idx)
		if _, exists := profs[candidate]; !exists {
			return candidate
		}
	}
}

func testRemoteConnections(remotes string) error {
	dialer := websocket.Dialer{HandshakeTimeout: 5 * time.Second}
	for _, remote := range strings.Split(remotes, ",") {
		remote = strings.TrimSpace(remote)
		if remote == "" {
			continue
		}
		conn, resp, err := dialer.Dial(remote, nil)
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
		if err != nil {
			return fmt.Errorf("上游连接失败 %s: %w", remote, err)
		}
		_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "test finished"), time.Now().Add(time.Second))
		_ = conn.Close()
	}
	return nil
}

func (ui *UI) PromptDialog(title string, message string, cb func(bool)) {
	log.Println(title, message)
	d := dialog.NewForm(
		title,
		"确认",
		"关闭",
		[]*widget.FormItem{
			widget.NewFormItem(message, &layout.Spacer{}),
			widget.NewFormItem("", &layout.Spacer{}),
		},
		cb,
		ui.Parent,
	)
	d.Show()
}
