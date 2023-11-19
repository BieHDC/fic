package gp

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	fc "github.com/BieHDC/fic/filecard"
	gp "github.com/BieHDC/fic/genericplayer"
)

func NewGifPlayer(frames []*canvas.Image, delays []int) *GifPlayer {
	g := &GifPlayer{}
	g.ExtendBaseWidget(g)

	g.framedisplay = canvas.NewImageFromResource(nil)
	g.framedisplay.FillMode = canvas.ImageFillContain
	g.framedisplay.ScaleMode = canvas.ImageScaleSmooth
	g.content = container.NewStack(g.framedisplay)

	g.speedmodifier = 1.0
	g.player = gp.NewPlayer(func(index int, block bool) {
		if g.onFrame != nil {
			g.onFrame(index)
		}
		g.setContent(g.frames[index])
		if block {
			// *10000000 -> nanosecond to 100th of a gif second
			time.Sleep(time.Duration(float64(g.delays[index]) * 10000000.0 * g.speedmodifier))
		}
	})

	g.setFrames(frames, delays)

	return g
}

// according to the gif decoder, len images is always len delays too
func (g *GifPlayer) setFrames(frames []*canvas.Image, delays []int) {
	g.length = len(frames)
	g.frames = frames
	g.delays = delays
	//
	g.setContent(g.frames[0])
	//
	g.player.SendEvent(gp.GPlayerConfig_SetMaxIndex, g.length)
}

func (g *GifPlayer) setContent(frame *canvas.Image) {
	g.framedisplay.Image = frame.Image
	g.framedisplay.Refresh()
}

func (g *GifPlayer) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(g.content)
}

func (g *GifPlayer) WithControlPanel(controls fyne.CanvasObject) *GifPlayer {
	g.content = container.NewStack(
		container.NewBorder(nil, controls, nil, nil, g.framedisplay),
	)
	return g
}

func (g *GifPlayer) PlayStop() bool {
	return g.player.SendEvent(gp.GPlayerAction_Playstop) == gp.GPlayerStatus_Playing
}

func (g *GifPlayer) PlayPause() bool {
	return g.player.SendEvent(gp.GPlayerAction_Playpause) == gp.GPlayerStatus_Playing
}

func (g *GifPlayer) Stop() bool {
	return g.player.SendEvent(gp.GPlayerAction_Stop) == gp.GPlayerStatus_Stopped
}

func (g *GifPlayer) Previous() bool {
	return g.player.SendEvent(gp.GPlayerAction_Previous) == gp.GPlayerStatus_OK
}

func (g *GifPlayer) Next() bool {
	return g.player.SendEvent(gp.GPlayerAction_Next) == gp.GPlayerStatus_OK
}

func (g *GifPlayer) SetDirection(dir int) bool {
	return g.player.SendEvent(gp.GPlayerConfig_Direction, dir) == gp.GPlayerStatus_OK
}

func (g *GifPlayer) SetSpeedModifier(speed float64) {
	g.speedmodifier = 1 / speed
}

func (g *GifPlayer) GetSeekerBounds() (int, int) {
	return 0, g.length - 1
}

func (g *GifPlayer) SeekTo(index int) bool {
	return g.player.SendEvent(gp.GPlayerAction_Seek, index) == gp.GPlayerStatus_OK
}

func (g *GifPlayer) SetOnFrame(onFrame func(int)) {
	g.onFrame = onFrame
}

type GifPlayer struct {
	widget.BaseWidget
	//
	length int
	frames []*canvas.Image
	delays []int
	//
	framedisplay  *canvas.Image
	onFrame       func(int)
	content       *fyne.Container
	speedmodifier float64
	player        *gp.GPlayer
}

// Used for File Preview Cards
type minPlayer struct {
	*GifPlayer
	playstop *widget.Button
}

func NewMinimalGifPlayer(frames []*canvas.Image, delays []int) *minPlayer {
	var lgp *GifPlayer
	var playstop *widget.Button
	playstop = widget.NewButtonWithIcon("Play", theme.MediaPlayIcon(), func() {
		if lgp.PlayStop() {
			//is playing
			playstop.SetText("Stop")
			playstop.SetIcon(theme.MediaStopIcon())
		} else {
			//is stopped
			playstop.SetText("Play")
			playstop.SetIcon(theme.MediaPlayIcon())
		}
	})
	lgp = NewGifPlayer(frames, delays).WithControlPanel(playstop)
	return &minPlayer{
		GifPlayer: lgp,
		playstop:  playstop,
	}
}

// Used for the Viewer
type extendedPlayer struct {
	*GifPlayer
	playpause *widget.Button
}

func NewExtendedGifPlayer(frames []*canvas.Image, delays []int) *extendedPlayer {
	var lgp *GifPlayer

	var playpause *widget.Button
	playpause = widget.NewButtonWithIcon("Play", theme.MediaPlayIcon(), func() {
		if lgp.PlayPause() {
			//is playing
			playpause.SetText("Resume")
			playpause.SetIcon(theme.MediaPauseIcon())
		} else {
			//is stopped
			playpause.SetText("Play")
			playpause.SetIcon(theme.MediaPlayIcon())
		}
	})
	stop := widget.NewButtonWithIcon("Stop", theme.MediaStopIcon(), func() {
		if lgp.Stop() {
			playpause.SetText("Play")
			playpause.SetIcon(theme.MediaPlayIcon())
		}
	})

	previous := widget.NewButtonWithIcon("Previous", theme.MediaSkipPreviousIcon(), func() {
		lgp.Previous()
	})
	next := widget.NewButtonWithIcon("Next", theme.MediaSkipNextIcon(), func() {
		lgp.Next()
	})

	currentdirection := 1
	playdirection := widget.NewButtonWithIcon("Reverse", theme.MediaReplayIcon(), func() {
		if currentdirection == 1 {
			currentdirection = -1
		} else {
			currentdirection = 1
		}
		lgp.SetDirection(currentdirection)
	})
	speed := widget.NewSlider(0.1, 10.0)
	speed.Step = 0.1
	speed.SetValue(1.0)
	speed.OnChanged = func(f float64) {
		lgp.SetSpeedModifier(f)
	}

	controlbuttons := container.NewGridWithColumns(2, playpause, stop, previous, next, playdirection,
		container.NewBorder(nil, nil, widget.NewButtonWithIcon("", theme.ContentClearIcon(), func() {
			speed.SetValue(1.0)
		}), nil, speed))

	seeker := widget.NewSlider(0, 0)
	seeker.Step = 1
	seeker.OnChanged = func(f float64) {
		lgp.SeekTo(int(f))
	}

	controls := container.NewBorder(controlbuttons, nil, nil, nil, seeker)

	lgp = NewGifPlayer(frames, delays).WithControlPanel(controls)
	lower, upper := lgp.GetSeekerBounds()
	seeker.Min = float64(lower)
	seeker.Max = float64(upper)
	lgp.SetOnFrame(func(i int) {
		seeker.Value = float64(i)
		seeker.Refresh() //needed
	})
	return &extendedPlayer{
		GifPlayer: lgp,
		playpause: playpause,
	}
}

func StopAllCanvasObjectsThatAreGifPlayers(o fyne.CanvasObject) {
	if player, ok := o.(*extendedPlayer); ok {
		player.Stop()
		return
	}
	if player, ok := o.(*minPlayer); ok {
		player.Stop()
		return
	}
	if fc, ok := o.(*fc.FileCard); ok {
		StopAllCanvasObjectsThatAreGifPlayers(fc.GetImage())
		return
	}
	if cont, ok := o.(*fyne.Container); ok {
		for _, obj := range cont.Objects {
			StopAllCanvasObjectsThatAreGifPlayers(obj)
		}
		return
	}
}
