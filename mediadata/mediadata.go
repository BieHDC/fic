package md

import (
	"context"
	"fmt"
	"image"
	"image/draw"
	"image/gif"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/semaphore"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/storage"
	"github.com/anthonynsimon/bild/transform"
)

type ImageType int

const (
	ImageStatic ImageType = iota
	ImageAnimated
)

type ImageDescriptor struct {
	Type   ImageType
	Images []*canvas.Image
	Delays []int
	valid  bool
}

type MediaData struct {
	mediacache map[string]ImageDescriptor
	medialock  sync.RWMutex
	iscaching  atomic.Bool
	cacherlock sync.Mutex
}

func (md *MediaData) InvalidateImageCache() {
	md.medialock.Lock()
	md.mediacache = make(map[string]ImageDescriptor)
	md.medialock.Unlock()
}

func (md *MediaData) InitialiseImageCache() {
	md.InvalidateImageCache()
}

func (md *MediaData) CancelCurrentCachetask() {
	md.iscaching.Store(false)
	md.cacherlock.Lock()
	md.cacherlock.Unlock()
}

func (md *MediaData) IsCaching() bool {
	return md.iscaching.Load()
}

func (md *MediaData) CacheTask(filelist []fyne.URI, status func(string, bool), maxworkers int64, maxfilesize int64) {
	md.cacherlock.Lock()
	defer md.cacherlock.Unlock()

	md.iscaching.Store(true)
	defer md.iscaching.Store(false)

	if status == nil {
		status = func(_ string, _ bool) {}
	}
	status("Caching process started!", false)

	sem := semaphore.NewWeighted(maxworkers)
	ctx := context.TODO()

	amount := len(filelist)
	var numprocessed atomic.Uint64

	starttime := time.Now()
	for _, uri := range filelist {
		//if false, caching has been killed off
		if !md.iscaching.Load() {
			break
		}

		err := sem.Acquire(ctx, 1)
		if err != nil {
			status(fmt.Sprintf("semaphore failed: %v", err), false)
			break
		}

		go func(uri fyne.URI) {
			md.CacheImage(uri, maxfilesize)
			finished := numprocessed.Add(1)
			status(fmt.Sprintf("Processing: %d/%d", finished, amount), false)
			sem.Release(1)
		}(uri)
	}
	// Force a little gc because it can be quite exhaustive
	runtime.GC()
	sem.Acquire(ctx, maxworkers) // ignore error, nothing we can do anyway
	runtime.GC()
	status(fmt.Sprintf("Caching done, took %0.2f seconds", time.Since(starttime).Seconds()), true)
}

func bToMb(b int64) int64 {
	return b / 1024 / 1024
}

func (md *MediaData) CacheImage(uri fyne.URI, maxfilesize int64) (*ImageDescriptor, error) {
	uristring := uri.String()
	md.medialock.RLock()
	cache, found := md.mediacache[uristring]
	md.medialock.RUnlock()
	if found {
		if cache.valid {
			return &cache, nil
		} else {
			return nil, fmt.Errorf("invalid file")
		}
	}

	imgdesc := ImageDescriptor{}
	defer func() {
		// works like charm
		md.medialock.Lock()
		md.mediacache[uristring] = imgdesc
		//fmt.Println("cached", uri)
		md.medialock.Unlock()
	}()

	file, err := os.Open(uri.Path())
	if err != nil {
		return nil, err
	}

	stat, err := file.Stat()
	if err != nil || stat.IsDir() {
		// it really shouldnt be able to be a dir at this point
		return nil, err
	}

	sz := stat.Size()
	maxsize := int64(1024 * 1024 * maxfilesize)
	if sz > maxsize {
		//fmt.Printf("%s -- %d\n", uri.Path(), bToMb(sz))
		return nil, fmt.Errorf("file too large: %d MB", bToMb(sz))
	}

	res, err := storage.Reader(uri)
	if err != nil {
		return nil, err
	}

	_, imageKind, err := image.DecodeConfig(res)
	if err != nil {
		res.Close()
		return nil, err
	}
	res.Close()
	//fmt.Println(uri.String(), "is a", imageKind)

	// it is easier to reopen the filestream than anything else
	res, err = storage.Reader(uri)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	switch imageKind {
	case "gif":
		//fixme should we resize large gifs too?
		//would be trivial, but who generally has >10mb large gifs?
		//would this impact quality too much then?
		gogif, err := gif.DecodeAll(res)
		if err != nil {
			return nil, err
		}

		img := make([]*canvas.Image, len(gogif.Image))
		dimension := image.Rect(0, 0, gogif.Image[0].Bounds().Dx(), gogif.Image[0].Bounds().Dy())
		last := image.NewRGBA(dimension)
		for i, frame := range gogif.Image {
			// process image
			current := image.NewRGBA(dimension)
			draw.Draw(current, last.Bounds(), last, last.Rect.Min, draw.Src)
			draw.Draw(current, frame.Bounds(), frame, frame.Rect.Min, draw.Over)
			last = current

			img[i] = canvas.NewImageFromImage(current)
			img[i].FillMode = canvas.ImageFillContain
			img[i].ScaleMode = canvas.ImageScaleSmooth

			// fixup delays if required
			if gogif.Delay[i] < 1 {
				// fix gifs with no delay to the default of 10
				// this is what other programs do
				gogif.Delay[i] = 10
			}
		}

		imgdesc.Type = ImageAnimated
		imgdesc.Images = img
		imgdesc.Delays = gogif.Delay

	default:
		goimg, _, err := image.Decode(res)
		if err != nil {
			return nil, err
		}

		imgsizeX := goimg.Bounds().Dx()
		imgsizeY := goimg.Bounds().Dy()
		const maxSide = 1024 //fixme up to debate
		if imgsizeX > maxSide || imgsizeY > maxSide {
			newsizeX, newsizeY := calculateNewResolution(imgsizeX, imgsizeY, maxSide)
			//fmt.Println("Sizing down", imgsizeX, "x", imgsizeY, "to", newsizeX, "x", newsizeY)
			goimg = transform.Resize(goimg, newsizeX, newsizeY, transform.NearestNeighbor)
		}

		img := canvas.NewImageFromImage(goimg)
		img.FillMode = canvas.ImageFillContain
		img.ScaleMode = canvas.ImageScaleSmooth

		imgdesc.Type = ImageStatic
		imgdesc.Images = append(imgdesc.Images, img)
	}

	imgdesc.valid = true
	return &imgdesc, nil
}

func calculateNewResolution(width, height, maxside int) (int, int) {
	if width > height {
		return maxside, int((float64(height) / float64(width)) * float64(maxside))
	} else {
		return int((float64(width) / float64(height)) * float64(maxside)), maxside
	}
}
