package main

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"

	memory "github.com/BieHDC/fic/memquery"
)

type Statusbar struct {
	statusbar       binding.String
	currentfilename binding.String
	xoutofy         binding.String
	memusage        binding.String
}

func (v *Viewer) setFileNumber(cf, sum int) {
	v.xoutofy.Set(fmt.Sprintf("(%d/%d)", cf+1, sum))
}

func (v *Viewer) setStatus(s string) {
	v.statusbar.Set(s)
}

func (v *Viewer) refreshMemoryUsage() uint {
	mi := memory.GetMemInfo()
	if mi == nil {
		v.memusage.Set("Failed to get Memory Info")
		return 0
	}

	rampc := (float64(mi.MemoryTotal-mi.MemoryFree) / float64(mi.MemoryTotal)) * 100
	swappc := (float64(mi.SwapTotal-mi.SwapFree) / float64(mi.SwapTotal)) * 100
	v.memusage.Set(fmt.Sprintf("Ram%%: %0.0f | Swap%%: %0.0f", rampc, swappc))
	return uint(rampc)
}

func (v *Viewer) makeStatusbar() fyne.CanvasObject {
	v.currentfilename = binding.NewString()
	v.currentfilename.Set("None Selected")
	v.xoutofy = binding.NewString()
	v.xoutofy.Set("0/0")

	v.memusage = binding.NewString()
	v.refreshMemoryUsage()

	v.statusbar = binding.NewString()
	v.statusbar.Set("Ready")
	return container.NewBorder(
		nil,
		nil,
		container.NewHBox(
			widget.NewLabelWithData(v.xoutofy),
			widget.NewLabelWithData(v.currentfilename),
		),
		widget.NewLabelWithData(v.memusage),
		container.NewHScroll(widget.NewLabelWithData(v.statusbar)),
	)
}
