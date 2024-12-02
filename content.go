package main

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	ft "github.com/BieHDC/fic/filetree"
	gp "github.com/BieHDC/fic/gifplayer"
)

type Content struct {
	filetree      *widget.Tree
	filetreedata  *ft.Filetreemaps
	mainContainer *fyne.Container
	selected      widget.TreeNodeID
}

func (v *Viewer) makeViewer() fyne.CanvasObject {
	r := canvas.NewImageFromResource(theme.BrokenImageIcon())
	r.SetMinSize(fyne.NewSquareSize(256))
	r.FillMode = canvas.ImageFillContain
	r.ScaleMode = canvas.ImageScaleSmooth

	v.mainContainer = container.NewStack(r)

	return v.mainContainer
}

func (v *Viewer) setMainContainer(o fyne.CanvasObject) {
	// this recursively walks all containers seeking GifPlayers
	// and telling them to stop playing
	gp.StopAllCanvasObjectsThatAreGifPlayers(v.mainContainer)
	v.mainContainer.RemoveAll()
	v.mainContainer.Add(o)
	v.mainContainer.Refresh() //needed
}

func (v *Viewer) initMainContainer() {
	v.mainContainer = container.NewStack()
}

func (v *Viewer) makeLeft() fyne.CanvasObject {
	v.InitialiseImageCache()

	parentinfo := func(uri fyne.URI) (int, int) {
		folders := 0
		files := 0
		items := v.filetreedata.Ids[uri.String()]

		for _, uri := range items {
			isDir := v.filetree.IsBranch(uri)
			if isDir {
				folders++
			} else {
				files++
			}
		}
		return folders, files
	}

	v.filetree = widget.NewTree(
		// childs
		func(id widget.TreeNodeID) []widget.TreeNodeID {
			return v.filetreedata.Ids[id]
		},
		// is parent
		func(id widget.TreeNodeID) bool {
			return len(v.filetree.ChildUIDs(id)) > 0
		},
		// create
		func(_ bool) fyne.CanvasObject {
			return widget.NewLabel("expected filename.png")
		},
		// update
		func(id widget.TreeNodeID, isBranch bool, obj fyne.CanvasObject) {
			l := obj.(*widget.Label)
			uri, ok := v.filetreedata.Values[id]
			if !ok {
				panic("bad id")
			}
			if isBranch {
				folders, files := parentinfo(uri)
				l.SetText(uri.Name() + fmt.Sprintf(" (%d|%d)", folders, files))
			} else {
				l.SetText(uri.Name())
			}
		},
	)

	v.filetree.OnSelected = func(id widget.TreeNodeID) {
		v.selected = id
		uri, ok := v.filetreedata.Values[id]
		if !ok {
			v.setStatus("error getting uri")
			return
		}

		li := v.filetree.IsBranch(uri.String())
		if li {
			v.SetNewFolder(uri.String(), false, true)
			defer v.displayLoadingScreen("Generating previews")()
			v.displayPreview(v.imgplayer.List())

			v.setFileNumber(0, v.imgplayer.Len())
			v.setStatus("Selected Folder: " + uri.Path() + "/")
			return
		}

		v.SetNewFolder(parentfromfile(uri).String(), false, true)
		v.imgplayer.SeekToData(uri.String())
		v.setStatus("Selected Folder: " + strings.TrimSuffix(uri.Path(), uri.Name()))
	}

	v.refreshFileTree(v.rootdir)
	v.filetree.OpenBranch(v.rootdir.String())

	searchbutton, searchcontent := v.makeSearchbar()
	return container.NewBorder(
		searchbutton,
		nil, nil, nil,
		container.NewStack(v.filetree, searchcontent),
	)
}

func (v *Viewer) makeSearchbar() (fyne.CanvasObject, fyne.CanvasObject) {
	var results_display []string
	var results_entry []string

	searchresults := widget.NewList(
		// length
		func() int {
			return len(results_display)
		},
		// create
		func() fyne.CanvasObject {
			return widget.NewLabel("expected filename.png")
		},
		// update
		func(lii widget.ListItemID, o fyne.CanvasObject) {
			l := o.(*widget.Label)
			l.SetText(results_display[lii])
		},
	)
	searchresults.OnSelected = func(id widget.ListItemID) {
		v.imgplayer.SeekToData(results_entry[id])
	}

	searchbox := widget.NewEntry()
	searchbox.OnChanged = func(s string) {
		results_display = results_display[:0]
		results_entry = results_entry[:0]
		if len(s) > 2 {
			for searchname, uri := range v.filetreedata.Values {
				name := uri.Name()
				if strings.Contains(name, s) {
					results_display = append(results_display, name)
					results_entry = append(results_entry, searchname)
				}
			}
		}
		if len(results_entry) >= 1 {
			v.imgplayer.SetNewData(results_entry)
		}
		searchresults.Refresh()
	}
	searchcontent := container.NewBorder(
		container.NewBorder(nil, nil, nil,
			widget.NewButtonWithIcon("", theme.CancelIcon(), func() { searchbox.SetText("") }),
			searchbox),
		nil, nil, nil,
		searchresults,
	)
	searchcontent.Hide()

	searchbutton := widget.NewButtonWithIcon("Search", theme.SearchIcon(), func() {
		if searchcontent.Hidden {
			searchcontent.Show()
			v.filetree.Hide()
			// so we can reselect the last selected folder
			v.filetree.UnselectAll()
			// restore last query
			if len(results_entry) >= 1 {
				v.imgplayer.SetNewData(results_entry)
			}
		} else {
			searchcontent.Hide()
			v.filetree.Show()
		}
	})

	return searchbutton, searchcontent
}

func (v *Viewer) displayLoadingScreen(msg string) func() {
	infprogressbar := widget.NewProgressBarInfinite()
	v.setMainContainer(
		container.NewCenter(
			container.NewVBox(
				widget.NewLabel(
					msg,
				),
				infprogressbar,
			),
		),
	)
	return func() {
		infprogressbar.Stop()
		infprogressbar = nil
	}
}

func (v *Viewer) refreshFileTree(dir fyne.ListableURI) {
	v.setStatus("Loading folder info...")
	defer v.displayLoadingScreen(fmt.Sprintf("Loading folder: %s", dir.Path()))()
	var took float64
	v.filetreedata, took = ft.Fillfiletree(binding.DataTreeRootID, dir, dir.String())
	v.setStatus(fmt.Sprintf("Loading finished! Took %0.3f sec", took))
}

func parentfromfile(uri fyne.URI) fyne.URI {
	child, err := storage.Parent(uri)
	if err != nil {
		return nil
	}
	parent, err := storage.ParseURI(strings.TrimSuffix(child.String(), "/"))
	if err != nil {
		return nil
	}
	return parent
	//return storage.NewFileURI(strings.TrimSuffix(uri.Path(), "/"+uri.Name()))
}
