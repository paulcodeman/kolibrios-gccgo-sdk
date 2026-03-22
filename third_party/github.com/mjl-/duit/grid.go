package duit

import (
	"fmt"
	"image"

	"9fans.net/go/draw"
)

type Grid struct {
	Kids       []*Kid
	Columns    int
	Valign     []Valign
	Halign     []Halign
	Padding    []Space
	Width      int
	Background *draw.Image `json:"-"`

	widths  []int
	heights []int
	size    image.Point
}

var _ UI = &Grid{}

func (ui *Grid) Layout(dui *DUI, self *Kid, sizeAvail image.Point, force bool) {
	dui.debugLayout(self)
	if KidsLayout(dui, self, ui.Kids, force) {
		return
	}
	if ui.Columns <= 0 {
		panic("grid columns must be positive")
	}
	if len(ui.Kids)%ui.Columns != 0 {
		panic(fmt.Sprintf("len(kids)=%d should be multiple of columns=%d", len(ui.Kids), ui.Columns))
	}
	if ui.Valign != nil && len(ui.Valign) != ui.Columns {
		panic("grid valign mismatch")
	}
	if ui.Halign != nil && len(ui.Halign) != ui.Columns {
		panic("grid halign mismatch")
	}
	if ui.Padding != nil && len(ui.Padding) != ui.Columns {
		panic("grid padding mismatch")
	}
	scaledWidth := dui.Scale(ui.Width)
	if scaledWidth > 0 && scaledWidth < sizeAvail.X {
		sizeAvail.X = scaledWidth
	}
	spaces := make([]Space, ui.Columns)
	for i := range spaces {
		if ui.Padding != nil {
			spaces[i] = dui.ScaleSpace(ui.Padding[i])
		}
	}
	ui.widths = make([]int, ui.Columns)
	x := make([]int, ui.Columns)
	totalWidth := 0
	for col := 0; col < ui.Columns; col++ {
		if col > 0 {
			x[col] = x[col-1] + ui.widths[col-1]
		}
		space := spaces[col]
		colAvail := maximum(0, sizeAvail.X-space.Dx())
		if remainingCols := ui.Columns - col; remainingCols > 0 {
			remainingWidth := sizeAvail.X - totalWidth
			if remainingWidth > 0 {
				budget := remainingWidth / remainingCols
				if budget > 0 {
					colAvail = maximum(0, budget-space.Dx())
				}
			}
		}
		for i := col; i < len(ui.Kids); i += ui.Columns {
			k := ui.Kids[i]
			k.UI.Layout(dui, k, image.Pt(colAvail, sizeAvail.Y-space.Dy()), true)
			ui.widths[col] = maximum(ui.widths[col], k.R.Dx()+space.Dx())
		}
		totalWidth += ui.widths[col]
	}
	if totalWidth > sizeAvail.X {
		overflow := totalWidth - sizeAvail.X
		for col := len(ui.widths) - 1; col >= 0 && overflow > 0; col-- {
			minWidth := spaces[col].Dx() + 1
			shrink := minimum(overflow, maximum(0, ui.widths[col]-minWidth))
			ui.widths[col] -= shrink
			overflow -= shrink
		}
		totalWidth = 0
		for col := range ui.widths {
			totalWidth += ui.widths[col]
		}
	}
	x[0] = 0
	for i := 1; i < len(x); i++ {
		x[i] = x[i-1] + ui.widths[i-1]
	}
	if ui.Width < 0 && totalWidth < sizeAvail.X {
		leftover := sizeAvail.X - totalWidth
		for i := range ui.widths {
			add := leftover / len(ui.widths)
			if i == len(ui.widths)-1 {
				add = leftover
			}
			ui.widths[i] += add
			leftover -= add
		}
		for i := 1; i < len(x); i++ {
			x[i] = x[i-1] + ui.widths[i-1]
		}
		totalWidth = sizeAvail.X
	}
	rows := (len(ui.Kids) + ui.Columns - 1) / ui.Columns
	ui.heights = make([]int, rows)
	y := make([]int, rows)
	totalHeight := 0
	for row := 0; row < rows; row++ {
		if row > 0 {
			y[row] = y[row-1] + ui.heights[row-1]
		}
		for col := 0; col < ui.Columns; col++ {
			i := row*ui.Columns + col
			space := spaces[col]
			k := ui.Kids[i]
			k.UI.Layout(dui, k, image.Pt(ui.widths[col]-space.Dx(), sizeAvail.Y-y[row]-space.Dy()), true)
			k.R = k.R.Add(image.Pt(x[col], y[row]).Add(space.Topleft()))
			ui.heights[row] = maximum(ui.heights[row], k.R.Dy()+space.Dy())
		}
		totalHeight += ui.heights[row]
	}
	for i, k := range ui.Kids {
		row := i / ui.Columns
		col := i % ui.Columns
		space := spaces[col]
		valign := ValignTop
		halign := HalignLeft
		if ui.Valign != nil {
			valign = ui.Valign[col]
		}
		if ui.Halign != nil {
			halign = ui.Halign[col]
		}
		cellSize := image.Pt(ui.widths[col], ui.heights[row]).Sub(space.Size())
		shift := image.ZP
		switch halign {
		case HalignMiddle:
			shift.X = (cellSize.X - k.R.Dx()) / 2
		case HalignRight:
			shift.X = cellSize.X - k.R.Dx()
		}
		switch valign {
		case ValignMiddle:
			shift.Y = (cellSize.Y - k.R.Dy()) / 2
		case ValignBottom:
			shift.Y = cellSize.Y - k.R.Dy()
		}
		k.R = k.R.Add(shift)
	}
	ui.size = image.Pt(totalWidth, totalHeight)
	self.R = rect(ui.size)
}

func (ui *Grid) Draw(dui *DUI, self *Kid, img *draw.Image, orig image.Point, m draw.Mouse, force bool) {
	KidsDraw(dui, self, ui.Kids, ui.size, ui.Background, img, orig, m, force)
}
func (ui *Grid) Mouse(dui *DUI, self *Kid, m draw.Mouse, origM draw.Mouse, orig image.Point) Result {
	return KidsMouse(dui, self, ui.Kids, m, origM, orig)
}
func (ui *Grid) Key(dui *DUI, self *Kid, k rune, m draw.Mouse, orig image.Point) Result {
	return KidsKey(dui, self, ui.Kids, k, m, orig)
}
func (ui *Grid) FirstFocus(dui *DUI, self *Kid) *image.Point {
	return KidsFirstFocus(dui, self, ui.Kids)
}
func (ui *Grid) Focus(dui *DUI, self *Kid, o UI) *image.Point {
	return KidsFocus(dui, self, ui.Kids, o)
}
func (ui *Grid) Mark(self *Kid, o UI, forLayout bool) bool {
	return KidsMark(self, ui.Kids, o, forLayout)
}
func (ui *Grid) Print(self *Kid, indent int) {
	PrintUI("Grid", self, indent)
	KidsPrint(ui.Kids, indent+1)
}
