package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/validation"
	"fyne.io/fyne/v2/driver/mobile"
	"fyne.io/fyne/v2/widget"
)

type Settings struct {
	maxworkers        uint
	maxfilesize       uint
	includesubfolders bool
	//windowsize
	winx       float32
	winy       float32
	fullscreen bool
}

var DefaultSettings = Settings{
	maxworkers:        8,
	maxfilesize:       100,
	includesubfolders: true,
}

func (s *Settings) LoadSettings() {
	app := fyne.CurrentApp()
	s.maxworkers = uint(app.Preferences().IntWithFallback("maxworkers", int(DefaultSettings.maxworkers)))
	s.maxfilesize = uint(app.Preferences().IntWithFallback("maxfilesize", int(DefaultSettings.maxfilesize)))
	s.includesubfolders = app.Preferences().BoolWithFallback("includesubfolders", DefaultSettings.includesubfolders)
	//
	s.winx = float32(app.Preferences().FloatWithFallback("winx", 800))
	s.winy = float32(app.Preferences().FloatWithFallback("winy", 600))
	s.fullscreen = app.Preferences().BoolWithFallback("fullscreen", false)
}

func (s *Settings) LoadDefaults() {
	s.maxworkers = DefaultSettings.maxworkers
	s.maxfilesize = DefaultSettings.maxfilesize
	s.includesubfolders = DefaultSettings.includesubfolders
}

func (s *Settings) SaveSettings(winx, winy float32, fullscreen bool) {
	app := fyne.CurrentApp()
	app.Preferences().SetInt("maxworkers", int(s.maxworkers))
	app.Preferences().SetInt("maxfilesize", int(s.maxfilesize))
	app.Preferences().SetBool("includesubfolders", s.includesubfolders)
	//
	app.Preferences().SetFloat("winx", float64(winx))
	app.Preferences().SetFloat("winy", float64(winy))
	app.Preferences().SetBool("fullscreen", fullscreen)
}

func (s *Settings) ApplySettings(maxworkers, maxfilesize uint, includesubfolders bool) {
	s.maxworkers = maxworkers
	s.maxfilesize = maxfilesize
	s.includesubfolders = includesubfolders
}

func NewFormItemWithHintText(text string, o fyne.CanvasObject, hint string) *widget.FormItem {
	fi := widget.NewFormItem(text, o)
	fi.HintText = hint
	return fi
}

type numEntry struct {
	widget.Entry
}

func (n *numEntry) Keyboard() mobile.KeyboardType {
	return mobile.NumberKeyboard
}

func newNumEntry() *numEntry {
	e := &numEntry{}
	e.ExtendBaseWidget(e)
	e.Validator = validation.NewRegexp(`\d`, "Must contain a number")
	return e
}

// load
// save
// on change
// subwindowpopup
