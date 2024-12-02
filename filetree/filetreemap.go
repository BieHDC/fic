package ft

import (
	"os"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/storage"
)

type Filetreemaps struct {
	mu     sync.Mutex
	Ids    map[string][]string
	Values map[string]fyne.URI
}

func newFiletreemaps() *Filetreemaps {
	return &Filetreemaps{
		Ids:    make(map[string][]string),
		Values: make(map[string]fyne.URI),
	}
}

func (ft *Filetreemaps) Nil() {
	ft.Ids = nil
	ft.Values = nil
}

func (ft *Filetreemaps) addEntryNotLocked(parent, id string, val fyne.URI, prepend bool) {
	lids, ok := ft.Ids[parent]
	if !ok {
		lids = make([]string, 0)
	}

	if prepend {
		ft.Ids[parent] = append([]string{id}, lids...)
	} else {
		ft.Ids[parent] = append(lids, id)
	}
	ft.Values[id] = val
}

func (ft *Filetreemaps) merge(childfolder string, cft *Filetreemaps, childuri fyne.URI) {
	ft.mu.Lock()
	// dont need to lock the child, it has to be finished before merge

	for k, v := range cft.Ids {
		ft.Ids[k] = v
	}
	for k, v := range cft.Values {
		ft.Values[k] = v
	}
	ft.Values[childfolder] = childuri

	ft.mu.Unlock()
}

func Fillfiletree(parent string, dir fyne.ListableURI, root string) (*Filetreemaps, float64) {
	ft := newFiletreemaps()
	ft.addEntryNotLocked(binding.DataTreeRootID, dir.String(), dir, true)

	cft := newFiletreemaps()
	start := time.Now()
	sem := make(chan struct{}, 200) // chosen by gut feeling
	cft.walkdirectory(dir.String(), dir, sem)

	close(sem)
	// nobody seems to really know if we need or should do this
	for range sem {
	}
	sem = nil

	ft.merge(dir.String(), cft, dir)

	return ft, time.Since(start).Seconds()
}

type entryFolder struct {
	parentfolder string
	nodeID       string
	uri          fyne.ListableURI
}

type entryFile struct {
	parentfolder string
	nodeID       string
	uri          fyne.URI
}

func walkfolder(parentfolder string, dir fyne.ListableURI) ([]entryFolder, []entryFile) {
	var folders []entryFolder
	var files []entryFile

	items, _ := dir.List()
	for _, uri := range items {
		uri := uri
		nodeID := uri.String()

		fileinfo, err := os.Lstat(uri.Path())
		if err != nil {
			continue
		}
		mode := fileinfo.Mode()

		if mode.IsDir() {
			uri, err := storage.ListerForURI(uri)
			if uri == nil || err != nil {
				continue
			}
			numitems, _ := uri.List()
			// do not add empty folders
			if len(numitems) > 0 {
				folders = append(folders, entryFolder{parentfolder, nodeID, uri})
			}
			continue
		}

		if mode.IsRegular() {
			files = append(files, entryFile{parentfolder, nodeID, uri})
			continue
		}
	}

	return folders, files
}

func (ft *Filetreemaps) walkdirectory(parentfolder string, dir fyne.ListableURI, sem chan struct{}) {
	folders, files := walkfolder(parentfolder, dir)

	var wg sync.WaitGroup
	wg.Add(len(folders))
	for _, folder := range folders {
		sem <- struct{}{}
		go func(folder entryFolder) {
			cft := newFiletreemaps()
			<-sem
			cft.walkdirectory(folder.nodeID, folder.uri, sem)
			ft.merge(folder.nodeID, cft, folder.uri)
			//lets help the gc a little out
			cft.Nil()
			cft = nil
			wg.Done()
		}(folder)
	}
	wg.Wait()

	for _, file := range files {
		ft.addEntryNotLocked(file.parentfolder, file.nodeID, file.uri, false)
	}

	lf := len(folders)
	if lf == 0 {
		return
	}
	for i := lf - 1; i >= 0; i-- {
		ff := folders[i]
		ft.addEntryNotLocked(ff.parentfolder, ff.nodeID, ff.uri, true)
	}

	// lets help the GC a little out
	files = nil
	folders = nil
}

/*
// working as expected, kept as archive
func (ft *Filetreemaps) walkdirectory(parentfolder string, dir fyne.ListableURI) {
	var folders []folder

	items, _ := dir.List()
	for _, uri := range items {
		uri := uri
		nodeID := uri.String()

		isDir, err := storage.CanList(uri)
		if err == nil && isDir {
			uri, _ := storage.ListerForURI(uri)
			fi, _ := uri.List()
			// do not add empty folders
			if len(fi) > 0 {
				folders = append(folders, folder{parentfolder, nodeID, uri})
				cft := newFiletreemaps()
				cft.walkdirectory(nodeID, uri)
				ft.merge(parentfolder, nodeID, cft, uri)
			}
			continue
		}

		if !isDir {
			ft.addEntry(parentfolder, nodeID, uri, false)
			continue
		}
	}

	lf := len(folders)
	if lf == 0 {
		return
	}
	for i := lf-1; i >= 0; i-- {
		ff := folders[i]
		ft.addEntry(ff.pf, ff.nid, ff.uri, true)
	}
}
*/
