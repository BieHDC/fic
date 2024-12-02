package main

import (
	"fmt"
	"runtime"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"fyne.io/fyne/v2/cmd/fyne_settings/settings"

	ilp "github.com/BieHDC/fic/imagelistplayer"
)

type Menubar struct {
	imgplayer      *ilp.ImagePlayer
	selectedfolder string
}

type contextMenuButton struct {
	widget.Button
	menu *fyne.Menu
}

func (b *contextMenuButton) Tapped(pe *fyne.PointEvent) {
	//we add 1unit down so clicking again closes the menu instead of invoking the first action
	widget.ShowPopUpMenuAtPosition(b.menu, fyne.CurrentApp().Driver().CanvasForObject(b), pe.AbsolutePosition.AddXY(0, 1))
}

func newContextMenuButton(label string, menu *fyne.Menu) *contextMenuButton {
	b := &contextMenuButton{menu: menu}
	b.Text = label

	b.ExtendBaseWidget(b)
	return b
}

func (v *Viewer) makeMenu(w fyne.Window) fyne.CanvasObject {
	setNewFolder := func(lu fyne.ListableURI) {
		v.InvalidateImageCache()
		v.rootdir = lu
		v.refreshFileTree(v.rootdir)
		v.filetree.OpenBranch(v.rootdir.String())
		v.filetree.ScrollToTop()
		v.filetree.Select(v.rootdir.String())
	}

	openfolder := widget.NewButtonWithIcon("Open Folder", theme.FolderOpenIcon(), func() {
		fo := dialog.NewFolderOpen(func(lu fyne.ListableURI, err error) {
			if err != nil {
				v.setStatus(err.Error())
				return
			}

			if lu == nil {
				v.setStatus("opening cancelled")
				return
			}

			setNewFolder(lu)
		}, w)

		fo.SetLocation(v.rootdir)
		fo.Show()
		fo.Resize(fo.MinSize().Add(fo.MinSize()))
	})
	w.SetOnDropped(func(_ fyne.Position, urix []fyne.URI) {
		if len(urix) < 1 {
			return
		}
		uri := urix[0]
		if uri == nil {
			return
		}
		isDir, _ := storage.CanList(uri)
		if !isDir {
			parent := parentfromfile(uri)
			if parent == nil {
				//fmt.Println("cannot open parent for file:", uri.String())
				return
			}
			defer v.imgplayer.SeekToData(uri.String())
			uri = parent
		}
		lu, _ := storage.ListerForURI(uri)
		setNewFolder(lu)
	})

	subfolders := widget.NewCheck("", func(_ bool) {})
	threads := newNumEntry()
	workers := func() uint {
		asuint, err := strconv.Atoi(threads.Text)
		if err != nil {
			return DefaultSettings.maxworkers
		}
		return uint(max(1, asuint))
	}
	maxfilesize := newNumEntry()
	filesize := func() uint {
		asuint, err := strconv.Atoi(maxfilesize.Text)
		if err != nil {
			return DefaultSettings.maxfilesize
		}
		return uint(max(1, asuint))
	}
	ficsettings := widget.NewForm(
		NewFormItemWithHintText("Include Subfolders", subfolders, "Used when selecting a folder"),
		NewFormItemWithHintText("Max Worker Threads", threads, "How many threads are loading images"),
		NewFormItemWithHintText("Max File Size in MB", maxfilesize, "Do not accidentally load too big images"),
	)

	resetSettingWidgetsValues := func() {
		subfolders.Checked = v.includesubfolders
		threads.Text = fmt.Sprintf("%d", v.maxworkers)
		maxfilesize.Text = fmt.Sprintf("%d", v.maxfilesize)
	}
	resetSettingWidgetsValues()

	menu := newContextMenuButton("Options", fyne.NewMenu("",
		fyne.NewMenuItem("Open Folder", func() { openfolder.Tapped(&fyne.PointEvent{}) }),
		fyne.NewMenuItem("Clear Cache", func() {
			v.InvalidateImageCache()
			runtime.GC()
			v.setStatus("Cache has been cleared")
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Fic Settings", func() {
			dialog.ShowCustomConfirm("Fic Settings", "Save", "Defaults", ficsettings,
				func(save bool) {
					if save {
						v.ApplySettings(workers(), filesize(), subfolders.Checked)
						v.SetNewFolder(v.selectedfolder, true, false) //dont seek when the flag is switched
						if subfolders.Checked {
							v.setStatus("Subfolders will be included")
						} else {
							v.setStatus("Subfolders will be ignored")
						}
					} else {
						// load the defaults on dismiss
						v.LoadDefaults()
						resetSettingWidgetsValues()
					}
				}, w)
		}),
		fyne.NewMenuItem("Appearance", func() {
			w := fyne.CurrentApp().NewWindow("Fyne Settings")
			settings := settings.NewSettings().LoadAppearanceScreen(w)
			w.SetContent(settings)
			minsz := settings.MinSize()
			w.Resize(minsz.AddWidthHeight(minsz.Width/3, minsz.Height/2))
			w.Show()
		}),
	))

	return menu
}

func (v *Viewer) makeMenubar(w fyne.Window) fyne.CanvasObject {
	prev := widget.NewButtonWithIcon("Previous", theme.NavigateBackIcon(), func() { v.imgplayer.Previous() })
	next := widget.NewButtonWithIcon("Next", theme.NavigateNextIcon(), func() { v.imgplayer.Next() })

	v.imgplayer.SetOnPlayFunc(func() {
		go v.CacheTask(v.filestringsToURI(v.imgplayer.List()),
			func(s string, _ bool) {
				v.setStatus(s)
			}, int64(v.maxworkers), int64(v.maxfilesize))
	})

	var playpause *widget.Button
	playpause = widget.NewButtonWithIcon("Play", theme.MediaPlayIcon(), func() {
		if v.imgplayer.PlayPause() {
			//is playing
			playpause.SetText("Pause")
			playpause.SetIcon(theme.MediaPauseIcon())
		} else {
			//is stopped
			playpause.SetText("Play")
			playpause.SetIcon(theme.MediaPlayIcon())
		}
	})

	stop := widget.NewButtonWithIcon("Stop", theme.MediaStopIcon(), func() {
		go func() {
			// do not freeze the ui while we wait for the player to stop
			if v.imgplayer.Stop() {
				playpause.SetText("Play")
				playpause.SetIcon(theme.MediaPlayIcon())
			}
		}()
	})

	var precache *widget.Button
	precache = widget.NewButtonWithIcon("Cache Folder", theme.MediaRecordIcon(), func() {
		if v.IsCaching() {
			precache.SetIcon(theme.MediaRecordIcon())
			precache.SetText("Start Caching")
			//kill caching
			go v.CancelCurrentCachetask()
		} else {
			precache.SetIcon(theme.MediaStopIcon())
			precache.SetText("Stop Caching")
			go v.CacheTask(v.filestringsToURI(v.imgplayer.List()),
				func(s string, done bool) {
					v.setStatus(s)
					if done {
						precache.SetIcon(theme.MediaRecordIcon())
						precache.SetText("Start Caching")
					}
				}, int64(v.maxworkers), int64(v.maxfilesize))

		}
	})

	seeker := widget.NewSlider(0, 0)
	seeker.Step = 1
	seeker.OnChanged = func(f float64) {
		v.imgplayer.SeekTo(int(f))
	}

	speed := widget.NewSlider(20, 2000)
	speed.Step = 5
	speed.SetValue(200) //fixme remember from default settings
	speedasstring := widget.NewLabel("")
	speedasstring.SetText(fmt.Sprintf("%04.0f", speed.Value))

	estimatedplaytime := widget.NewLabel("")
	estimatedplaytimeupdate := func() {
		estimate := (float64(v.imgplayer.Len()) * speed.Value) / 1000
		estimatedplaytime.SetText(fmt.Sprintf("%0.2f seconds", estimate))
	}
	speed.OnChanged = func(f float64) {
		estimatedplaytimeupdate()
		speedasstring.SetText(fmt.Sprintf("%04.0f", f))
	}

	v.imgplayer.SetOnFrameFunc(func(index int, data []string, block bool) {
		if index >= len(data) {
			index = 0
		}
		
		seeker.SetValue(float64(index))
		seeker.Refresh()
		//we unselect, so we can reclick the folder to display the preview
		v.filetree.UnselectAll()

		uri, ok := v.filetreedata.Values[data[index]]
		if !ok {
			v.setStatus(data[index] + " failed: does not exist")
			return
		}

		li := v.filetree.IsBranch(uri.String())
		if li {
			defer v.displayLoadingScreen("Generating previews")()
			v.displayPreview(v.filetreedata.Ids[data[index]])
		} else {
			err := v.displayImage(uri)
			if err != nil {
				v.setStatus(data[index] + " failed: " + err.Error())
				//move on quicker if the image was not an image
				block = false
			}
		}
		v.setFileNumber(index, v.imgplayer.Len())

		if block {
			time.Sleep(time.Duration(speed.Value) * time.Millisecond)
		}
	})
	v.imgplayer.SetOnDataChangedFunc(func() {
		low, high := v.imgplayer.GetSeekerBounds()
		seeker.Min = float64(low)
		seeker.Max = float64(high)
		estimatedplaytimeupdate()
		v.filetree.OnSelected(v.selected)
	})

	ticker := time.NewTicker(5 * time.Second)
	go func() {
		for {
			<-ticker.C
			ramusedpercent := v.refreshMemoryUsage()
			if ramusedpercent > 80 {
				precache.Importance = widget.DangerImportance
				runtime.GC() // lets pressure the gc a little
			} else {
				precache.Importance = widget.MediumImportance
			}
			precache.Refresh()
		}
	}()

	return container.NewVBox(
		container.NewBorder(
			nil, nil,
			container.NewHBox(
				v.makeMenu(w),
				precache,
				prev,
				next,
				playpause,
				stop,
				speedasstring,
			),
			estimatedplaytime,
			speed,
		),
		seeker,
	)
}

func (v *Viewer) filestringsToURI(files []string) []fyne.URI {
	uris := make([]fyne.URI, 0, len(files))
	for _, index := range files {
		uri, ok := v.filetreedata.Values[index]
		if !ok {
			continue
		}
		uris = append(uris, uri)
	}
	return uris
}

func (v *Viewer) walksubfolder(child string) []string {
	result := make([]string, 0, len(v.filetreedata.Ids[child]))
	for _, file := range v.filetreedata.Ids[child] {
		if len(v.filetreedata.Ids[file]) > 0 {
			//folder inside folder
			result = append(result, v.walksubfolder(file)...)
			continue
		}
		result = append(result, file)
	}
	return result
}

func (v *Viewer) SetNewFolder(selectedfolder string, force bool, seek bool) {
	if (selectedfolder == v.selectedfolder) && !force {
		return
	}
	v.selectedfolder = selectedfolder

	// it is most of the time around the selected folder len
	filelist := make([]string, 0, len(v.filetreedata.Ids[v.selectedfolder]))

	internaloffset := v.imgplayer.Cursor()
	newoffset := 0
	for offset, file := range v.filetreedata.Ids[v.selectedfolder] {
		clen := len(v.filetreedata.Ids[file])
		if v.includesubfolders && clen > 0 {
			//is a folder, walk it
			subfiles := v.walksubfolder(file)
			filelist = append(filelist, subfiles...)

			if internaloffset > offset {
				newoffset += len(subfiles)
			}
			continue
		}
		if clen > 0 {
			continue
		}

		filelist = append(filelist, file)
	}

	v.imgplayer.SetNewData(filelist)
	if seek {
		v.imgplayer.SeekTo(newoffset)
	}
}
