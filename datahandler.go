package main

import (
	"cmp"
	"fmt"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/webp"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	dg "github.com/BieHDC/fic/dynamicgrid"
	fc "github.com/BieHDC/fic/filecard"
	gp "github.com/BieHDC/fic/gifplayer"
	md "github.com/BieHDC/fic/mediadata"
)

func (v *Viewer) displayImage(uri fyne.URI) error {
	img, err := v.CacheImage(uri, int64(v.maxfilesize))
	if err != nil {
		return err
	}
	if img == nil {
		// we add a folder that had empty folders inside of it
		// which should not happen. but it would make the dir
		// walker much more complex and its easier to just
		// fail here in that case.
		return fmt.Errorf("invalid file")
	}

	var disp fyne.CanvasObject
	if img.Type == md.ImageAnimated {
		disp = gp.NewExtendedGifPlayer(img.Images, img.Delays)
	} else {
		disp = img.Images[0]
	}

	v.setMainContainer(disp)
	v.currentfilename.Set(uri.Name())
	return nil
}

type previewInfo struct {
	index int //for sorting
	uri   fyne.URI
	img   *md.ImageDescriptor
}

// fixme with the current algorithm its hard to parallelise the preview gen
// we generally need a better sampling technique here
func (v *Viewer) gatherPreviews(files []string, amount int) []previewInfo {
	numitems := len(files)
	targets := make([]previewInfo, 0, amount)

	tryadd := func(index int, filename string) bool {
		isDir := v.filetree.IsBranch(filename)
		if isDir {
			// dont care about dirs
			return false
		}

		uri := v.filetreedata.Values[filename]
		img, err := v.CacheImage(uri, int64(v.maxfilesize))
		if err != nil || img == nil {
			// ignore this non-image
			// see hint at displayImage()
			return false
		}

		targets = append(targets, previewInfo{
			index: index,
			uri:   uri,
			img:   img,
		})

		v.setStatus(fmt.Sprintf("Generating Previews (%d/%d)", index, numitems))
		return true
	}

	addFirstUpToAmount := func() {
		successfullyadded := 0
		for i, uri := range files {
			if tryadd(i, uri) {
				successfullyadded++
			}
			// take the first x amount of valid images
			if successfullyadded >= amount {
				break
			}
		}
	}

	if numitems < amount {
		// the folder does not have enough items for random
		// source-ing anyway. do all of them.
		addFirstUpToAmount()
	} else {
		// this way of doing it might not be the even-est, as
		// it is accumulating pressure at the end of the list
		// however it gives satisfactory results overall
		// there should never be duplicates
		found := 0

		step := int(float64(numitems) / float64(amount))
		for index := 0; ; index += step {
			uri := files[index]
			for !tryadd(index, uri) {
				index++
				//restep with the rest
				step = int((float64(numitems - index)) / (float64(amount) - float64(found)))
				if index >= numitems || step <= 0 {
					//too little items available, do all successful ones
					//or too much pressure at the end of the loop
					//fmt.Println("this folder is littered with non-images, falling back to doing a linear all")
					// clear the results
					targets = make([]previewInfo, 0, amount)
					addFirstUpToAmount()
					//i prefer this over break-ing to a label
					//it just says we have found enough and can exit the loop
					found = amount
					break
				}
				uri = files[index]
			}

			found += 1
			//got enough images
			if found >= amount {
				break
			}
		}
	}

	if len(targets) < 1 {
		// we have found no usable image anywhere (empty folder, or no images)
		return nil
	}

	slices.SortFunc(targets, func(a, b previewInfo) int {
		return cmp.Compare(a.index, b.index)
	})

	return targets
}

func (v *Viewer) generateFilecards(finaltargets []previewInfo) []fyne.CanvasObject {
	cards := make([]fyne.CanvasObject, 0, len(finaltargets))

	for _, target := range finaltargets {
		uri := target.uri
		uriasstring := uri.String()
		img := target.img

		var disp fyne.CanvasObject
		if img.Type == md.ImageAnimated {
			disp = gp.NewMinimalGifPlayer(img.Images, img.Delays)
		} else {
			disp = img.Images[0]
		}
		card := fc.NewFileCard(uri.Name(), disp).WithCallback(func(_ *fyne.PointEvent) {
			parents := parentsfromfile(v.rootdir, uri)
			for _, parent := range parents {
				v.filetree.OpenBranch(parent.String())
			}
			v.filetree.ScrollTo(uriasstring)
			v.filetree.Select(uriasstring)
		})
		cards = append(cards, card)
	}

	return cards
}

func parentsfromfile(root, child fyne.URI) []fyne.URI {
	var list []fyne.URI
	rootasstring := root.String()

	for {
		pp := parentfromfile(child)
		if pp == nil {
			break
		}
		list = append(list, pp)
		if pp.String() == rootasstring {
			break
		}
		child = pp
	}

	return list
}

func (v *Viewer) displayPreview(files []string) {
	const imagesPerViewRows = 2
	const imagesPerViewColums = 2

	finaltargets := v.gatherPreviews(files, imagesPerViewRows*imagesPerViewColums)
	if finaltargets == nil {
		v.setMainContainer(container.NewCenter(widget.NewLabel("Nothing to display")))
		return
	}
	cards := v.generateFilecards(finaltargets)

	var preview *fyne.Container
	preview = dg.NewDynamicGrid(imagesPerViewRows, imagesPerViewColums, func(amount int) []fyne.CanvasObject {
		preview.Objects = v.generateFilecards(v.gatherPreviews(files, amount))
		v.setStatus("Previews finished Generating.")
		return preview.Objects
	})
	preview.Objects = cards
	v.setMainContainer(preview)
	v.setStatus("Previews finished Generating.")
}
