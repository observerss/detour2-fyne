package layout

import "fyne.io/fyne/v2"

type FixedHeight struct {
	Height float32
}

func NewFixedHeight(height float32) *FixedHeight {
	return &FixedHeight{Height: height}
}

func (f *FixedHeight) MinSize(objects []fyne.CanvasObject) fyne.Size {
	width := float32(0)
	for _, object := range objects {
		childSize := object.MinSize()
		if childSize.Width > width {
			width = childSize.Width
		}
	}
	return fyne.NewSize(width, f.Height)
}

func (f *FixedHeight) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	for _, object := range objects {
		object.Move(fyne.NewPos(0, 0))
		object.Resize(fyne.NewSize(size.Width, f.Height))
	}
}
