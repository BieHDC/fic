package fc

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type FileCard struct {
	widget.BaseWidget
	filename string
	image    fyne.CanvasObject
	tapped   func(*fyne.PointEvent)
}

var _ fyne.Tappable = (*FileCard)(nil)

func trimitiseFilename(s string) string {
	const targetStringLen = 26 //arbitrary number, subject to testing and change
	strlen := len(s)
	if strlen < targetStringLen {
		return s
	}
	amounttotrim := strlen - targetStringLen
	lowerend := (strlen / 2) - (amounttotrim / 2)
	higherend := (strlen / 2) + (amounttotrim / 2)

	return s[0:lowerend] + "[...]" + s[higherend:]
}

func NewFileCard(filename string, image fyne.CanvasObject) *FileCard {
	card := &FileCard{
		filename: trimitiseFilename(filename),
		image:    image,
	}
	card.ExtendBaseWidget(card)
	return card
}

func (c *FileCard) Tapped(pe *fyne.PointEvent) {
	if c.tapped != nil {
		c.tapped(pe)
	}
}

/*
func (c *FileCard) TappedSecondary(pe *fyne.PointEvent) {
	menu := fyne.NewMenu("Fileinfo",
			     fyne.NewMenuItem("Copy Link", func() {
				     fmt.Println("this is:" ,c.Filename)
			}))
	widget.ShowPopUpMenuAtPosition(menu, fyne.CurrentApp().Driver().CanvasForObject(c), pe.AbsolutePosition.AddXY(0,1))
}
*/

func (c *FileCard) WithCallback(cb func(*fyne.PointEvent)) *FileCard {
	c.tapped = cb
	return c
}

// CreateRenderer is a private method to Fyne which links this widget to its renderer
func (c *FileCard) CreateRenderer() fyne.WidgetRenderer {
	c.ExtendBaseWidget(c)

	filenameText := canvas.NewText(c.filename, theme.ForegroundColor())
	filenameText.Alignment = fyne.TextAlignCenter
	return &filecardRenderer{
		filenameText: filenameText,
		card:         c,
	}
}

// MinSize returns the size that this widget should not shrink below
func (c *FileCard) MinSize() fyne.Size {
	c.ExtendBaseWidget(c)
	return c.BaseWidget.MinSize()
}

// SetImage changes the image displayed above the title for this card.
// fixme untested
func (c *FileCard) SetImage(img fyne.CanvasObject) {
	c.image = img
	c.Refresh()
}

// Only used for the recursive GifPlayer stopper
func (c *FileCard) GetImage() fyne.CanvasObject {
	return c.image
}

// SetFilename updates the secondary title for this card.
// fixme untested
func (c *FileCard) SetFilename(text string) {
	c.filename = text
	c.Refresh()
}

type filecardRenderer struct {
	filenameText *canvas.Text
	card         *FileCard
}

var _ fyne.WidgetRenderer = (*filecardRenderer)(nil)

const (
	// fixme whats a nice value?
	cardMediaHeight   = 256
	cardMediaMinWidth = 256
)

func (c *filecardRenderer) Destroy() {}

func (c *filecardRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{c.filenameText, c.card.image}
}

// Layout the components of the card container.
func (c *filecardRenderer) Layout(size fyne.Size) {
	padding := theme.Padding()
	pos := fyne.NewSquareOffsetPos(padding / 2)
	size = size.Subtract(fyne.NewSquareSize(padding))

	if c.card.image != nil {
		c.card.image.Move(pos)
		c.card.image.Resize(fyne.NewSize(size.Width, cardMediaHeight))
		pos.Y += cardMediaHeight
	}

	if c.card.filename != "" {
		titlePad := padding * 2
		size.Width -= titlePad * 2
		height := c.filenameText.MinSize().Height
		c.filenameText.Move(pos)
		c.filenameText.Resize(fyne.NewSize(size.Width, height))
		pos.Y += height + padding
	}

	size.Width -= padding * 2
	pos.X += padding
}

// MinSize calculates the minimum size of a card.
// This is based on the filename text, image.
func (c *filecardRenderer) MinSize() fyne.Size {
	padding := theme.Padding()
	min := fyne.NewSquareSize(padding)
	if c.card.image != nil {
		min = fyne.NewSize(max(min.Width, cardMediaMinWidth), min.Height+cardMediaHeight)
	}
	if c.card.filename != "" {
		titlePad := padding * 2
		min = min.Add(fyne.NewSize(0, titlePad*2))
		subHeaderMin := c.filenameText.MinSize()
		min = fyne.NewSize(fyne.Max(min.Width, subHeaderMin.Width+titlePad*2+padding),
			min.Height+subHeaderMin.Height)
	}

	return min
}

func (c *filecardRenderer) Refresh() {
	c.Layout(c.card.BaseWidget.Size())
	if c.filenameText != nil {
		c.filenameText.Text = c.card.filename
		c.filenameText.TextSize = theme.TextSize()
		c.filenameText.Color = theme.ForegroundColor()
		c.filenameText.Refresh()
	}
	if c.card.image != nil {
		c.card.image.Refresh()
	}
}
