package duit

import (
	"image"

	"9fans.net/go/draw"
)

type UI interface {
	Layout(dui *DUI, self *Kid, sizeAvail image.Point, force bool)
	Draw(dui *DUI, self *Kid, img *draw.Image, orig image.Point, m draw.Mouse, force bool)
	Mouse(dui *DUI, self *Kid, m draw.Mouse, origM draw.Mouse, orig image.Point) (r Result)
	Key(dui *DUI, self *Kid, k rune, m draw.Mouse, orig image.Point) (r Result)
	FirstFocus(dui *DUI, self *Kid) (warp *image.Point)
	Focus(dui *DUI, self *Kid, o UI) (warp *image.Point)
	Mark(self *Kid, o UI, forLayout bool) (marked bool)
	Print(self *Kid, indent int)
}
