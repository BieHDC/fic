package dg

import (
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
)

func NewDynamicGrid(minrows, mincols int, requestMore func(int) []fyne.CanvasObject, objects ...fyne.CanvasObject) *fyne.Container {
	return container.New(NewDynamicGridLayout(minrows, mincols, requestMore), objects...)
}

// Declare conformity with Layout interface
var _ fyne.Layout = (*dynamicGridLayout)(nil)

type dynamicGridLayout struct {
	minrows     int
	mincols     int
	requestMore func(int) []fyne.CanvasObject
	exhausted   bool
}

// NewDynamicGridLayout returns a new grid layout which uses columns when horizontal but rows when vertical.
func NewDynamicGridLayout(minrows, mincols int, requestMore func(int) []fyne.CanvasObject) fyne.Layout {
	minrows = max(minrows, 1)
	mincols = max(mincols, 1)
	exhausted := requestMore == nil //we will never be able to request more
	return &dynamicGridLayout{
		minrows:     minrows,
		mincols:     mincols,
		requestMore: requestMore,
		exhausted:   exhausted,
	}
}

func (g *dynamicGridLayout) horizontal() bool {
	return fyne.IsHorizontal(fyne.CurrentDevice().Orientation())
}

// Get the leading (top or left) edge of a grid cell.
// size is the ideal cell size and the offset is which col or row its on.
func getLeading(size float64, offset int) float32 {
	ret := (size + float64(theme.Padding())) * float64(offset)

	return float32(ret)
}

// Get the trailing (bottom or right) edge of a grid cell.
// size is the ideal cell size and the offset is which col or row its on.
func getTrailing(size float64, offset int) float32 {
	return getLeading(size, offset+1) - theme.Padding()
}

// We do uniform sizes
func (g *dynamicGridLayout) biggestChild(objects []fyne.CanvasObject) fyne.Size {
	childsize := fyne.NewSquareSize(1)
	for _, child := range objects {
		childsize = childsize.Max(child.MinSize())
	}
	return childsize
}

// fixme this needs a smarter implementation, but keep this one
// until a better one is actually discovered
func oldCalcer(num int, childsize fyne.Size, size fyne.Size) (int, int) {
	var numcols int
	var numrows int
	for {
		childsize = childsize.Add(fyne.NewSquareSize(1))
		numcols = max(int(math.Floor(float64(size.Width)/float64(childsize.Width))), 1)
		numrows = max(int(math.Floor(float64(size.Height)/float64(childsize.Height))), 1)
		if numcols*numrows <= num {
			childsize = childsize.Subtract(fyne.NewSquareSize(1))
			numcols = max(int(math.Floor(float64(size.Width)/float64(childsize.Width))), 1)
			numrows = max(int(math.Floor(float64(size.Height)/float64(childsize.Height))), 1)
			break
		}
	}
	return numcols, numrows
}

// Layout is called to pack all child objects into a specified size.
// For a DynamicGridLayout this will pack the needed amount of objects
// into a table format with at least the minumum specified columns and rows
// and if less content available to fill as much space as possible.
func (g *dynamicGridLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	childsize := g.biggestChild(objects)
	numcols := max(int(math.Floor(float64(size.Width)/float64(childsize.Width))), 1)
	numrows := max(int(math.Floor(float64(size.Height)/float64(childsize.Height))), 1)

	maxamount := numrows * numcols
	if !g.exhausted && (len(objects) < maxamount) {
		if g.requestMore != nil {
			objects = g.requestMore(maxamount)
			if len(objects) < maxamount {
				// stop trying to fetch more items
				g.exhausted = true
			}
		}
	}

	if g.exhausted && (len(objects) < maxamount) {
		ec, er := oldCalcer(len(objects), childsize, size)
		numcols = ec
		numrows = er
		/*
			fmt.Printf("expecting cols:%d rows:%d\n", ec, er)

			// part 1 of the new code
			// fmt.Printf("size of canvas: %f x %f\n", size.Width, size.Height)
			formula := func(p, c float64) int {
				delta := -math.Pow(c, 2) / (p + c)
				xx := p / (c + delta)
				fmt.Printf("\t%f\n", xx)
				return int(xx)
			}

			numcols = formula(float64(size.Width), float64(childsize.Width))
			numrows = formula(float64(size.Height), float64(childsize.Height))

			fmt.Printf("got cols:%d rows:%d\n", numcols, numrows)
		*/
		maxamount = numcols * numrows
	}

	padding := theme.Padding()
	padWidth := float32(numcols-1) * padding
	padHeight := float32(numrows-1) * padding
	cellWidth := float64(size.Width-padWidth) / float64(numcols)
	cellHeight := float64(size.Height-padHeight) / float64(numrows)

	if !g.horizontal() {
		padWidth, padHeight = padHeight, padWidth
		cellWidth = float64(size.Width-padWidth) / float64(numrows)
		cellHeight = float64(size.Height-padHeight) / float64(numcols)
	}

	row, col := 0, 0
	i := 0
	for _, child := range objects {
		if i == maxamount {
			// we own the childs visibility status
			child.Hide()
			continue
		}
		x1 := getLeading(cellWidth, col)
		y1 := getLeading(cellHeight, row)
		x2 := getTrailing(cellWidth, col)
		y2 := getTrailing(cellHeight, row)

		child.Move(fyne.NewPos(x1, y1))
		child.Resize(fyne.NewSize(x2-x1, y2-y1))
		child.Show()

		if g.horizontal() {
			if (i+1)%numcols == 0 {
				row++
				col = 0
			} else {
				col++
			}
		} else {
			if (i+1)%numcols == 0 {
				col++
				row = 0
			} else {
				row++
			}
		}
		i++
	}
}

// MinSize finds the smallest size that satisfies all the child objects.
// For a DynamicGridLayout this is the size of the largest child object
// multiplied by the minimum number of columns and rows, with
// appropriate padding between children.
func (g *dynamicGridLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	minSize := g.biggestChild(objects)

	if g.horizontal() {
		minContentSize := fyne.NewSize(minSize.Width*float32(g.mincols), minSize.Height*float32(g.minrows))
		return minContentSize.Add(fyne.NewSize(theme.Padding()*fyne.Max(float32(g.mincols-1), 0), theme.Padding()*fyne.Max(float32(g.minrows-1), 0)))
	}

	minContentSize := fyne.NewSize(minSize.Width*float32(g.minrows), minSize.Height*float32(g.mincols))
	return minContentSize.Add(fyne.NewSize(theme.Padding()*fyne.Max(float32(g.minrows-1), 0), theme.Padding()*fyne.Max(float32(g.mincols-1), 0)))
}
