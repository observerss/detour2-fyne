package utils

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

func PromptDialog(title string, message string, cb func(bool), parent fyne.Window) {
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
		parent,
	)
	d.Show()
}
