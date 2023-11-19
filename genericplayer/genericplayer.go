package gp

import (
	"sync/atomic"
)

type gplayerAction int

const (
	GPlayerAction_Stop      gplayerAction = iota
	GPlayerAction_Playpause               //toggle
	GPlayerAction_Playstop                //toggle
	GPlayerAction_Play
	GPlayerAction_Pause
	//
	GPlayerConfig_SetMaxIndex
	GPlayerConfig_Direction
	//
	GPlayerAction_Next
	GPlayerAction_Previous
	GPlayerAction_Seek
	//
	gplayerinternal_unblock
)

type GPlayer struct {
	index     int
	onFrame   func(int, bool)
	direction int
	//
	maxindex  int
	isrunning atomic.Bool
	// reset on stop
	paused    bool
	eventchan chan gplayerAction
}

// for the callback:
// the first parameter is the index it should show
// the 2nd parameter says if you should block on it
// for prev, next and seek for example, we want to immediatly return
// for the play-along player we want to want to display the animation
// with the right speed
func NewPlayer(onFrame func(int, bool)) *GPlayer {
	gp := &GPlayer{}
	gp.onFrame = onFrame
	gp.direction = 1
	return gp
}

func (gp *GPlayer) ensureUnblocked() {
	select {
	case gp.eventchan <- gplayerinternal_unblock:
	default:
	}
}

func (gp *GPlayer) play() {
	for {
		select {
		case evttype := <-gp.eventchan:
			switch evttype {
			case GPlayerAction_Stop:
				gp.paused = false
				gp.index = 0
				gp.onFrame(gp.index, false)
				gp.eventchan <- GPlayerAction_Stop //signal we are done
				return
			case GPlayerAction_Pause:
				gp.paused = true
				<-gp.eventchan
			case GPlayerAction_Play:
				gp.paused = false
			case gplayerinternal_unblock:
				//ignore
			default:
				println(evttype)
				panic("unexpected event")
			}

		default:
			gp.move(gp.direction)
			gp.onFrame(gp.index, true)
		}
	}
}

func (gp *GPlayer) move(offset int) {
	gp.index += offset

	// roll over
	if gp.index < 0 {
		gp.index = gp.maxindex - 1
	}
	if gp.index >= gp.maxindex {
		gp.index = 0
	}
}

type GPlayerStatus int

const (
	GPlayerStatus_Stopped GPlayerStatus = iota
	GPlayerStatus_Playing
	GPlayerStatus_Paused
	//
	GPlayerStatus_ArgCountMismatch
	GplayerStatus_EmptyPlaylist
	GPlayerStatus_OK
	GplayerStatus_Confused
)

func (gp *GPlayer) Cursor() int {
	return gp.index
}

// fixme technically this thing needs to be mutexed to safety, but unless
// someone makes tests or misuses it, i am not going to spend time on that,
// unless it happens to me
// if the issue comes, we need to stop the player, do the sets, then resume
// or go the cheapest way and mutexlock the play(), do the sets, and unlock them
// how expensive even is taking locks for fun?
func (gp *GPlayer) SendEvent(action gplayerAction, args ...int) GPlayerStatus {
	switch action {
	case GPlayerAction_Stop:
		if gp.isrunning.CompareAndSwap(true, false) {
			gp.ensureUnblocked()
			gp.eventchan <- GPlayerAction_Stop
			<-gp.eventchan
			close(gp.eventchan)
			gp.eventchan = nil
			return GPlayerStatus_Stopped
		}

	case GPlayerAction_Pause:
		if gp.isrunning.Load() {
			if !gp.paused {
				gp.eventchan <- GPlayerAction_Pause
				return GPlayerStatus_Paused
			} else {
				gp.ensureUnblocked()
				gp.eventchan <- GPlayerAction_Play
				return GPlayerStatus_Playing
			}
		}

	case GPlayerAction_Play:
		if gp.isrunning.CompareAndSwap(false, true) {
			if gp.maxindex <= 0 {
				return GplayerStatus_EmptyPlaylist
			}
			gp.eventchan = make(chan gplayerAction)
			go gp.play()
			return GPlayerStatus_Playing
		}

	case GPlayerAction_Playpause:
		if gp.isrunning.Load() {
			return gp.SendEvent(GPlayerAction_Pause)
		} else {
			return gp.SendEvent(GPlayerAction_Play)
		}

	case GPlayerAction_Playstop:
		if gp.isrunning.Load() {
			return gp.SendEvent(GPlayerAction_Stop)
		} else {
			return gp.SendEvent(GPlayerAction_Play)
		}

	case GPlayerConfig_SetMaxIndex:
		lenargs := len(args)
		if lenargs != 1 {
			return GPlayerStatus_ArgCountMismatch
		}
		gp.maxindex = args[0]
		if gp.index > gp.maxindex {
			gp.index = 0
			gp.onFrame(gp.index, false)
		}
		return GPlayerStatus_OK

	case GPlayerConfig_Direction:
		lenargs := len(args)
		if lenargs != 1 {
			return GPlayerStatus_ArgCountMismatch
		}
		gp.direction = args[0]
		return GPlayerStatus_OK

	case GPlayerAction_Seek:
		lenargs := len(args)
		if lenargs != 1 {
			return GPlayerStatus_ArgCountMismatch
		}
		if gp.index == args[0] {
			gp.onFrame(gp.index, false)
			return GPlayerStatus_OK
		}
		gp.index = args[0]
		// at least 0, at most gp.maxindex
		gp.index = max(min(gp.index, gp.maxindex), 0)
		gp.onFrame(gp.index, false)
		return GPlayerStatus_OK

	case GPlayerAction_Previous:
		gp.move(-1)
		gp.onFrame(gp.index, false)
		return GPlayerStatus_OK
	case GPlayerAction_Next:
		gp.move(1)
		gp.onFrame(gp.index, false)
		return GPlayerStatus_OK

	default:
		println("action not handled", action)
	}

	return GplayerStatus_Confused
}
