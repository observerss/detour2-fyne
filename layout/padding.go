package layout

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
)

type Padding struct {
	Left   float32
	Right  float32
	Top    float32
	Bottom float32
}

func NewPaddingContainer(obj fyne.CanvasObject, val *Padding) *fyne.Container {
	return container.New(val, obj)
}

func (p *Padding) MinSize(objects []fyne.CanvasObject) fyne.Size {
	w, h := float32(0), float32(0)
	for _, o := range objects {
		childSize := o.MinSize()

		w += childSize.Width
		h += childSize.Height
	}
	w += p.Left + p.Right
	h += p.Top + p.Bottom
	return fyne.NewSize(w, h)
}

func (p *Padding) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	pos := fyne.NewPos(p.Left, p.Top)
	newSize := fyne.Size{
		Width:  size.Width - p.Left - p.Right,
		Height: size.Height - p.Top - p.Bottom,
	}
	for _, o := range objects {
		o.Resize(newSize)
		o.Move(pos)
	}
}
