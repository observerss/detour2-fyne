package run

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type MobileSwitch struct {
	widget.BaseWidget
	Checked   bool
	Disabled  bool
	OnChanged func(bool)
}

func NewMobileSwitch(changed func(bool)) *MobileSwitch {
	s := &MobileSwitch{OnChanged: changed}
	s.ExtendBaseWidget(s)
	return s
}

func (s *MobileSwitch) SetChecked(checked bool) {
	if s.Checked == checked {
		return
	}
	s.Checked = checked
	s.Refresh()
}

func (s *MobileSwitch) Disable() {
	s.Disabled = true
	s.Refresh()
}

func (s *MobileSwitch) Enable() {
	s.Disabled = false
	s.Refresh()
}

func (s *MobileSwitch) Tapped(*fyne.PointEvent) {
	if s.Disabled {
		return
	}
	s.Checked = !s.Checked
	s.Refresh()
	if s.OnChanged != nil {
		s.OnChanged(s.Checked)
	}
}

func (s *MobileSwitch) TappedSecondary(*fyne.PointEvent) {}

func (s *MobileSwitch) CreateRenderer() fyne.WidgetRenderer {
	track := canvas.NewRectangle(s.trackColor())
	track.CornerRadius = 18
	thumb := canvas.NewCircle(color.White)
	thumb.StrokeWidth = 1
	thumb.StrokeColor = color.NRGBA{R: 0, G: 0, B: 0, A: 28}
	r := &mobileSwitchRenderer{sw: s, track: track, thumb: thumb, objects: []fyne.CanvasObject{track, thumb}}
	r.Layout(r.MinSize())
	return r
}

func (s *MobileSwitch) trackColor() color.Color {
	if s.Disabled {
		return color.NRGBA{R: 180, G: 180, B: 180, A: 150}
	}
	if s.Checked {
		return theme.PrimaryColor()
	}
	return color.NRGBA{R: 185, G: 190, B: 198, A: 255}
}

type mobileSwitchRenderer struct {
	sw      *MobileSwitch
	track   *canvas.Rectangle
	thumb   *canvas.Circle
	objects []fyne.CanvasObject
}

func (r *mobileSwitchRenderer) Layout(size fyne.Size) {
	trackSize := fyne.NewSize(size.Width, size.Height)
	r.track.Resize(trackSize)
	r.track.Move(fyne.NewPos(0, 0))

	pad := float32(4)
	thumbSize := size.Height - pad*2
	x := pad
	if r.sw.Checked {
		x = size.Width - thumbSize - pad
	}
	r.thumb.Resize(fyne.NewSize(thumbSize, thumbSize))
	r.thumb.Move(fyne.NewPos(x, pad))
}

func (r *mobileSwitchRenderer) MinSize() fyne.Size {
	return fyne.NewSize(64, 36)
}

func (r *mobileSwitchRenderer) Refresh() {
	r.track.FillColor = r.sw.trackColor()
	r.track.Refresh()
	r.thumb.Refresh()
	r.Layout(r.MinSize())
	canvas.Refresh(r.sw)
}

func (r *mobileSwitchRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *mobileSwitchRenderer) Destroy() {}
