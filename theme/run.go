package theme

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type Run struct {
}

func RunTheme() fyne.Theme {
	return &Run{}
}

func (r *Run) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	return theme.DefaultTheme().Color(name, variant)
}

func (r *Run) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (r *Run) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (r *Run) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}
