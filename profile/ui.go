package profile

import (
	"fmt"
	"log"
	"os"
	"sort"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	lo "github.com/observerss/detour2-fyne/layout"
	"github.com/observerss/detour2/common"
	"github.com/observerss/detour2/deploy"
	"github.com/observerss/detour2/logger"
)

type UI struct {
	Left            *widget.List
	Name            *widget.Entry
	CloudProvider   *widget.Select
	AccessKeyId     *widget.Entry
	AccessKeySecret *widget.Entry
	AccountId       *widget.Entry
	ServiceName     *widget.Entry
	FunctionName    *widget.Entry
	TriggerName     *widget.Entry
	Password        *widget.Entry
	Region          *widget.Select
	Image           *widget.Entry
	Delete          *widget.Button
	Reset           *widget.Button
	Test            *widget.Button
	Save            *widget.Button
	Profs           map[string]*Profile
	Parent          fyne.Window
	CurrentIdx      int
}

func NewUI(parent fyne.Window) *UI {
	return &UI{
		Left:            &widget.List{},
		Name:            widget.NewEntry(),
		CloudProvider:   widget.NewSelect([]string{}, func(s string) {}),
		AccessKeyId:     widget.NewEntry(),
		AccessKeySecret: widget.NewEntry(),
		AccountId:       widget.NewEntry(),
		ServiceName:     widget.NewEntry(),
		FunctionName:    widget.NewEntry(),
		TriggerName:     widget.NewEntry(),
		Password:        widget.NewEntry(),
		Image:           widget.NewEntry(),
		Region:          widget.NewSelect([]string{}, func(s string) {}),
		Parent:          parent,
	}
}

func (ui *UI) MakeUI() fyne.CanvasObject {
	ui.Delete = widget.NewButton("删除", ui.HandleDelete)
	ui.Reset = widget.NewButton("重置", ui.ResetForm)
	ui.Test = widget.NewButton("测试", ui.HandleTest)
	ui.Save = widget.NewButton("保存", ui.HandleSave)

	// initialize
	ui.SetupBindings()
	ui.ResetForm()
	ui.ResetLeft()

	// layout
	form := lo.NewPaddingContainer(container.New(layout.NewFormLayout(),
		widget.NewLabel("配置名 *"), ui.Name,
		widget.NewLabel("云服务商"), ui.CloudProvider,
		widget.NewLabel("AccessKeyId *"), ui.AccessKeyId,
		widget.NewLabel("AccessKeySecret *"), ui.AccessKeySecret,
		widget.NewLabel("主账号ID *"), ui.AccountId,
		widget.NewLabel("可用区"), ui.Region,
		widget.NewLabel("服务名"), ui.ServiceName,
		widget.NewLabel("函数名"), ui.FunctionName,
		widget.NewLabel("触发器名"), ui.TriggerName,
		widget.NewLabel("密码"), ui.Password,
		widget.NewLabel("镜像地址"), ui.Image,
	), &lo.Padding{Right: 10})
	buttons := container.NewCenter(container.NewHBox(ui.Delete, layout.NewSpacer(), ui.Reset, layout.NewSpacer(), ui.Test, layout.NewSpacer(), ui.Save))
	right := container.NewVBox(layout.NewSpacer(), form, layout.NewSpacer(), buttons, layout.NewSpacer())

	split := container.NewHSplit(container.NewHScroll(ui.Left), container.NewHScroll(right))
	split.SetOffset(0.2)

	return split
}

func GetProfileNames(profs map[string]*Profile) []string {
	names := make([]string, 0)
	for name := range profs {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (ui *UI) SetupBindings() {
	ui.Region.OnChanged = func(s string) {
		r, _ := GetAliyunRegionValue(s)
		ui.Image.SetText(GetImageByRegion(r))
	}
	ui.Left.OnSelected = func(id int) {
		ui.CurrentIdx = id
		names := GetProfileNames(ui.Profs)
		if len(names) > id {
			prof, ok := ui.Profs[names[id]]
			if ok {
				ui.SetForm(prof)
			}
		}
	}
}

func (ui *UI) ResetLeft() {
	profs, err := LoadProfiles()
	ui.Profs = profs
	var names []string
	if err != nil {
		names = []string{"default"}
	} else {
		names = GetProfileNames(profs)
	}
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
	if ui.CurrentIdx >= 0 {
		ui.Left.Select(ui.CurrentIdx)
	}
}

func (ui *UI) ResetForm() {
	regions := GetAliyunRegions()

	providers := GetCloudProviers()
	ui.CloudProvider.Options = providers
	ui.CloudProvider.SetSelected(providers[0])
	ui.CloudProvider.Disable()

	ui.Name.SetText("default")

	ui.AccessKeyId.SetText("")
	ui.AccessKeyId.SetPlaceHolder("e.g. LTasdfEAhGcfasdfTbfds")

	ui.AccessKeySecret.SetText("")
	ui.AccessKeySecret.SetPlaceHolder("e.g. 8CEghqeqweqwrehNPFJwreY ")

	ui.AccountId.SetText("")
	ui.AccountId.SetPlaceHolder("e.g. 14123451334236")

	ui.Region.Options = regions
	ui.Region.SetSelected(regions[0])

	sname, _ := GenerateRandomString(3)
	ui.ServiceName.SetText(sname)

	fname, _ := GenerateRandomString(3)
	ui.FunctionName.SetText(fname)

	tname, _ := GenerateRandomString(5)
	ui.TriggerName.SetText(tname)

	pvalue, _ := GenerateRandomString(8)
	ui.Password.SetText(pvalue)
}

func (ui *UI) SetForm(prof *Profile) {
	ui.Name.SetText(prof.Name)
	ui.CloudProvider.SetSelected(prof.CloudProvider.Display)
	ui.Region.SetSelected(prof.Region.Display)
	ui.AccessKeyId.SetText(prof.AccessKeyId)
	ui.AccessKeySecret.SetText(prof.AccessKeySecret)
	ui.AccountId.SetText(prof.AccountId)
	ui.ServiceName.SetText(prof.ServiceName)
	ui.FunctionName.SetText(prof.FunctionName)
	ui.TriggerName.SetText(prof.TriggerName)
	ui.Password.SetText(prof.Password)
}

func (ui *UI) ValidateForm() error {
	if ui.Name.Text == "" {
		return fmt.Errorf("Form.Name should not be empty")
	}

	if ui.AccessKeyId.Text == "" {
		return fmt.Errorf("Form.AccessKeyId should not be empty")
	}

	if ui.AccessKeySecret.Text == "" {
		return fmt.Errorf("Form.AccessKeySecret should not be empty")
	}

	if ui.AccountId.Text == "" {
		return fmt.Errorf("Form.AccountId should not be empty")
	}

	_, err := GetProvider(ui.CloudProvider.Selected)
	if err != nil {
		return err
	}

	_, err = GetAliyunRegion(ui.Region.Selected)
	if err != nil {
		return err
	}
	return nil
}

func (ui *UI) GetForm() (*Profile, error) {
	err := ui.ValidateForm()
	if err != nil {
		return nil, err
	}

	provider, _ := GetProvider(ui.CloudProvider.Selected)
	region, _ := GetAliyunRegion(ui.Region.Selected)

	prof := &Profile{
		Name:            ui.Name.Text,
		CloudProvider:   provider,
		AccessKeyId:     ui.AccessKeyId.Text,
		AccessKeySecret: ui.AccessKeySecret.Text,
		AccountId:       ui.AccountId.Text,
		Region:          region,
		ServiceName:     ui.ServiceName.Text,
		FunctionName:    ui.FunctionName.Text,
		TriggerName:     ui.TriggerName.Text,
		Password:        ui.Password.Text,
		Image:           ui.Image.Text,
	}
	return prof, nil
}

func (ui *UI) HandleTest() {
	prof, err := ui.GetForm()
	if err != nil {
		ui.PromptDialog("测试失败", err.Error(), func(b bool) {})
		return
	}
	conf := ConvertProfileToConfig(prof)

	bar := widget.NewProgressBar()
	text := widget.NewLabel("初始化中...")
	d := dialog.NewCustom(
		"测试中",
		"关闭",
		container.NewVBox(
			bar,
			text,
		),
		ui.Parent,
	)
	d.Resize(fyne.Size{Width: 640, Height: 240})
	d.Show()

	logger.Info.SetOutput(&clog{Text: text, Bar: bar})
	logger.Error.SetOutput(&clog{Text: text, Bar: bar})
	defer func() {
		logger.Info.SetOutput(os.Stdout)
		logger.Error.SetOutput(os.Stderr)
	}()

	err = deploy.DeployServer(conf)

	if err != nil {
		text.SetText("测试失败: " + err.Error())
	} else {
		conf.Remove = true
		deploy.DeployServer(conf)
		text.SetText("测试成功, 已成功部署并销毁云函数")
	}
}

func (ui *UI) HandleSave() {
	prof, err := ui.GetForm()
	if err != nil {
		ui.PromptDialog("保存失败", err.Error(), func(b bool) {})
		return
	}

	profs, err := LoadProfiles()
	if err != nil {
		ui.PromptDialog("保存失败", err.Error(), func(b bool) {})
		return
	}

	profs[prof.Name] = prof

	err = SaveProfiles(profs)
	if err != nil {
		ui.PromptDialog("保存失败", err.Error(), func(b bool) {})
		return
	}

	// update ui
	ui.Profs = profs
	ui.ResetLeft()
	ui.Left.Refresh()
	names := GetProfileNames(ui.Profs)
	prof = ui.Profs[names[ui.CurrentIdx]]
	ui.SetForm(prof)

	ui.PromptDialog("保存成功", "数据已保存", func(b bool) {})
}

func (ui *UI) HandleDelete() {
	names := GetProfileNames(ui.Profs)
	name := names[ui.CurrentIdx]
	prof, ok := ui.Profs[name]
	if !ok {
		log.Println("shouldn't happen")
		return
	}

	ui.PromptDialog("确认", fmt.Sprintf("请确认是否删除 %s", prof.Name), func(b bool) {
		if b {
			delete(ui.Profs, name)
			SaveProfiles(ui.Profs)
			ui.ResetLeft()
			ui.Left.Refresh()
			names := GetProfileNames(ui.Profs)
			prof = ui.Profs[names[ui.CurrentIdx]]
			ui.SetForm(prof)
		}
	})
}

func ConvertProfileToConfig(prof *Profile) *common.DeployConfig {
	conf := &common.DeployConfig{
		AccessKeyId:     prof.AccessKeyId,
		AccessKeySecret: prof.AccessKeySecret,
		AccountId:       prof.AccountId,
		Password:        prof.Password,
		Region:          prof.Region.Name,
		Remove:          false,
	}
	if prof.ServiceName != "" {
		conf.ServiceName = prof.ServiceName
	}
	if prof.FunctionName != "" {
		conf.FunctionName = prof.FunctionName
	}
	if prof.TriggerName != "" {
		conf.TriggerName = prof.TriggerName
	}
	if prof.Image != "" {
		conf.Image = prof.Image
	}
	return conf
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

type clog struct {
	Text *widget.Label
	Bar  *widget.ProgressBar
}

func (l *clog) Write(p []byte) (int, error) {
	l.Text.SetText(string(p))
	newvalue := l.Bar.Value + 0.1
	if newvalue <= 1.0 {
		l.Bar.SetValue(newvalue)
	}
	return 0, nil
}
