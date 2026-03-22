package duit

import "image"

func unionDirtyRect(a image.Rectangle, b image.Rectangle) image.Rectangle {
	if a.Empty() {
		return b
	}
	if b.Empty() {
		return a
	}
	if b.Min.X < a.Min.X {
		a.Min.X = b.Min.X
	}
	if b.Min.Y < a.Min.Y {
		a.Min.Y = b.Min.Y
	}
	if b.Max.X > a.Max.X {
		a.Max.X = b.Max.X
	}
	if b.Max.Y > a.Max.Y {
		a.Max.Y = b.Max.Y
	}
	return a
}

func dirtyBounds(k *Kid, orig image.Point) (image.Rectangle, bool) {
	if k == nil || k.UI == nil {
		return image.ZR, false
	}
	if k.Layout != Clean || k.Draw == Dirty {
		r := k.R.Add(orig)
		return r, !r.Empty()
	}
	switch ui := k.UI.(type) {
	case *Box:
		return kidsDirtyBounds(ui.Kids, orig)
	case *Grid:
		return kidsDirtyBounds(ui.Kids, orig)
	case *Scroll:
		return scrollDirtyBounds(ui, k, orig)
	default:
		if k.Draw != Clean {
			r := k.R.Add(orig)
			return r, !r.Empty()
		}
		return image.ZR, false
	}
}

func kidsDirtyBounds(kids []*Kid, orig image.Point) (image.Rectangle, bool) {
	var dirty image.Rectangle
	found := false
	for _, child := range kids {
		if child == nil {
			continue
		}
		if r, ok := dirtyBounds(child, orig.Add(child.R.Min)); ok {
			dirty = unionDirtyRect(dirty, r)
			found = true
		}
	}
	return dirty, found
}

func scrollDirtyBounds(ui *Scroll, self *Kid, orig image.Point) (image.Rectangle, bool) {
	if ui == nil || self == nil {
		return image.ZR, false
	}
	if self.Layout != Clean || self.Draw == Dirty {
		r := self.R.Add(orig)
		return r, !r.Empty()
	}
	if ui.Kid.Layout != Clean || ui.Kid.Draw == Dirty {
		r := ui.childR.Add(orig)
		return r, !r.Empty()
	}
	if r, ok := dirtyBounds(&ui.Kid, image.ZP); ok {
		view := image.Rect(0, ui.offset, ui.childR.Dx(), ui.offset+ui.childR.Dy())
		visible := r.Intersect(view)
		if !visible.Empty() {
			visible = visible.Add(orig.Add(ui.childR.Min).Sub(image.Pt(0, ui.offset)))
			return visible, true
		}
	}
	if self.Draw != Clean {
		r := self.R.Add(orig)
		return r, !r.Empty()
	}
	return image.ZR, false
}
