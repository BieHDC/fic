package main

import (
	"flag"
	"fmt"
	"image/color"
	"path/filepath"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"

	ilp "github.com/BieHDC/fic/imagelistplayer"
	md "github.com/BieHDC/fic/mediadata"
)

func main() {
	a := app.NewWithID("biehdc.fic.v1") // i do not redeem the standard convention
	w := a.NewWindow("Fast Image Cycler")

	splash, cancel := makeSplash(a, w)
	w.SetContent(splash)
	go func() {
		w.SetContent(makeMain(a, w))
		cancel()
		//not sure if this is needed for the garbage collector to wipe the splash out of memory
		splash = nil
		cancel = nil
	}()

	w.Show()
	a.Run()
}

type Viewer struct {
	//general
	rootdir fyne.ListableURI

	//ui specific
	Menubar
	Statusbar
	Content

	//handlers
	md.MediaData

	//settings
	Settings
}

func NewViewer() *Viewer {
	v := &Viewer{}
	return v
}

func makeSplash(a fyne.App, w fyne.Window) (fyne.CanvasObject, func()) {
	// fixme we should do something fancy here
	// maybe once the app has a nice icon, we
	// should use that
	box := canvas.NewRectangle(color.RGBA{255, 0, 0, 255})
	box.Resize(fyne.NewSize(30, 30))
	box.Move(fyne.NewPos(0, 0))

	var anim *fyne.Animation

	label := widget.NewLabel("Please wait while the application is starting...")
	splash := container.NewCenter(label, container.NewWithoutLayout(box))

	anim = canvas.NewPositionAnimation(
		label.Position().AddXY(-80, 20), label.Position().AddXY(+80, 20),
		600*time.Millisecond, func(p fyne.Position) {
			box.Move(p)
			box.Refresh()
		})
	anim.Curve = fyne.AnimationEaseInOut
	anim.AutoReverse = true
	anim.RepeatCount = fyne.AnimationRepeatForever
	anim.Start()

	return splash, func() {
		//remove animation from run loop
		anim.Stop()
		//remove all references
		splash = nil
		anim = nil
		box = nil
		label = nil
	}
}

func makeMain(a fyne.App, w fyne.Window) fyne.CanvasObject {
	v := NewViewer()
	// Load settings
	v.LoadSettings()
	w.SetFullScreen(v.fullscreen)
	w.Resize(fyne.NewSize(v.winx, v.winy))
	w.SetOnClosed(func() {
		// save resolution
		scale := a.Settings().Scale()
		x, y := w.Canvas().Size().Components()
		v.SaveSettings(x*scale, y*scale, w.FullScreen())
	})

	// Point of Interest
	// init everything that should be initted before buildup
	statusbar := v.makeStatusbar(a, w)
	v.initMainContainer()
	v.imgplayer = ilp.NewImagePlayer()

	// continue as usual
	flag.Parse()
	var rootdir string
	if len(flag.Args()) > 0 {
		rootdir = flag.Args()[0]
	} else {
		rootdir = "."
	}
	rootdir, err := filepath.Abs(rootdir)
	if err != nil {
		return container.NewCenter(widget.NewLabel(err.Error()))
	}
	v.rootdir, _ = stringToListerURI(rootdir)
	if v.rootdir == nil {
		return container.NewCenter(widget.NewLabel(fmt.Sprintf("bad root dir: %s", rootdir)))
	}

	// keyboard controls
	w.Canvas().SetOnTypedKey(func(ke *fyne.KeyEvent) {
		switch ke.Name {
		case fyne.KeyEscape:
			a.Quit()
		case fyne.KeyLeft:
			v.imgplayer.Previous()
		case fyne.KeyRight:
			v.imgplayer.Next()
		}
	})

	content := container.NewHSplit(v.makeLeft(a, w), v.makeViewer(a, w))
	content.SetOffset(0.3)
	final := container.NewBorder(
		v.makeMenubar(a, w),
		statusbar,
		nil,
		nil,
		content,
	)

	// Point of Interest
	// things to do after everything has been initialised
	// and settings have been restored
	v.SetNewFolder(v.rootdir.String(), true, true)

	return final
}

func stringToListerURI(dir string) (fyne.ListableURI, error) {
	fileuri := storage.NewFileURI(dir)
	diruri, err := storage.ListerForURI(fileuri)
	if err != nil {
		return nil, err
	}
	return diruri, nil
}
