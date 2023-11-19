package ilp

import (
	gp "github.com/BieHDC/fic/genericplayer"
)

type ImagePlayer struct {
	filelistlen int
	filelist    []string
	//
	player        *gp.GPlayer
	onFrame       func(int, []string, bool)
	onDataChanged func()
	onPlay        func()
}

func NewImagePlayer() *ImagePlayer {
	ip := &ImagePlayer{}

	ip.player = gp.NewPlayer(func(index int, block bool) {
		if ip.onFrame != nil {
			ip.onFrame(index, ip.filelist, block)
		}
	})

	return ip
}

func (ip *ImagePlayer) SetNewData(files []string) {
	ip.filelist = files
	ip.filelistlen = len(files)
	ip.player.SendEvent(gp.GPlayerConfig_SetMaxIndex, ip.filelistlen)
	if ip.onDataChanged != nil {
		ip.onDataChanged()
	}
}

func (ip *ImagePlayer) SetOnFrameFunc(cb func(int, []string, bool)) {
	ip.onFrame = cb
}

func (ip *ImagePlayer) SetOnDataChangedFunc(cb func()) {
	ip.onDataChanged = cb
}

func (ip *ImagePlayer) SetOnPlayFunc(cb func()) {
	ip.onPlay = cb
}

func (ip *ImagePlayer) PlayPause() bool {
	status := ip.player.SendEvent(gp.GPlayerAction_Playpause) == gp.GPlayerStatus_Playing
	if status {
		if ip.onPlay != nil {
			ip.onPlay()
		}
	}
	return status
}

func (ip *ImagePlayer) Stop() bool {
	return ip.player.SendEvent(gp.GPlayerAction_Stop) == gp.GPlayerStatus_Stopped
}

func (ip *ImagePlayer) Previous() bool {
	return ip.player.SendEvent(gp.GPlayerAction_Previous) == gp.GPlayerStatus_OK
}

func (ip *ImagePlayer) Next() bool {
	return ip.player.SendEvent(gp.GPlayerAction_Next) == gp.GPlayerStatus_OK
}

func (ip *ImagePlayer) GetSeekerBounds() (int, int) {
	return 0, ip.filelistlen - 1
}

func (ip *ImagePlayer) SeekTo(index int) bool {
	return ip.player.SendEvent(gp.GPlayerAction_Seek, index) == gp.GPlayerStatus_OK
}

func (ip *ImagePlayer) SeekToData(file string) bool {
	for i, id := range ip.filelist {
		if id == file {
			ip.SeekTo(i)
			return true
		}
	}
	return false
}

func (ip *ImagePlayer) Len() int {
	return ip.filelistlen
}

func (ip *ImagePlayer) List() []string {
	return ip.filelist
}

func (ip *ImagePlayer) Cursor() int {
	return ip.player.Cursor()
}
