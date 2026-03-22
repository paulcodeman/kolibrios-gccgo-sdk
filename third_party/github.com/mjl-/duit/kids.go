package duit

import (
	"fmt"
	"image"

	"9fans.net/go/draw"
)

type Kid struct {
	UI     UI
	R      image.Rectangle
	Draw   State
	Layout State
	ID     string
}

func (k *Kid) Mark(o UI, forLayout bool) (marked bool) {
	if o != k.UI {
		return false
	}
	if forLayout {
		k.Layout = Dirty
	} else {
		k.Draw = Dirty
	}
	return true
}

func NewKids(uis ...UI) []*Kid {
	kids := make([]*Kid, len(uis))
	for i, ui := range uis {
		kids[i] = &Kid{UI: ui}
	}
	return kids
}

func KidsLayout(dui *DUI, self *Kid, kids []*Kid, force bool) (done bool) {
	if force {
		self.Layout = Clean
		self.Draw = Dirty
		return false
	}
	switch self.Layout {
	case Clean:
		return true
	case Dirty:
		self.Layout = Clean
		self.Draw = Dirty
		return false
	}
	for _, k := range kids {
		if k.Layout == Clean {
			continue
		}
		k.UI.Layout(dui, k, k.R.Size(), false)
		if k.Layout != Clean {
			self.Layout = Dirty
			self.Draw = Dirty
			return false
		}
	}
	self.Layout = Clean
	self.Draw = Dirty
	return true
}

func KidsDraw(dui *DUI, self *Kid, kids []*Kid, uiSize image.Point, bg *draw.Image, img *draw.Image, orig image.Point, m draw.Mouse, force bool) {
	dui.debugDraw(self)
	force = force || self.Draw == Dirty
	if force {
		self.Draw = Dirty
	}
	if bg == nil {
		bg = dui.Background
	}
	if force && bg != nil {
		img.Draw(rect(uiSize).Add(orig), bg, nil, image.ZP)
	}
	for _, k := range kids {
		if !force && k.Draw == Clean {
			continue
		}
		if !force && k.Draw == Dirty && bg != nil {
			img.Draw(k.R.Add(orig), bg, nil, image.ZP)
		}
		mm := m
		mm.Point = mm.Point.Sub(k.R.Min)
		if force {
			k.Draw = Dirty
		}
		k.UI.Draw(dui, k, img, orig.Add(k.R.Min), mm, force)
		k.Draw = Clean
	}
	self.Draw = Clean
}

func propagateResult(dui *DUI, self *Kid, k *Kid) {
	if k.Layout != Clean {
		nk := *k
		k.UI.Layout(dui, &nk, k.R.Size(), false)
		if nk.R.Size() != k.R.Size() {
			self.Layout = Dirty
		} else {
			self.Layout = Clean
			k.Layout = Clean
			k.Draw = Dirty
			self.Draw = DirtyKid
		}
	} else if k.Draw != Clean {
		self.Draw = DirtyKid
	}
}

func KidsMouse(dui *DUI, self *Kid, kids []*Kid, m draw.Mouse, origM draw.Mouse, orig image.Point) (r Result) {
	for _, k := range kids {
		if !origM.Point.In(k.R) {
			continue
		}
		origM.Point = origM.Point.Sub(k.R.Min)
		m.Point = m.Point.Sub(k.R.Min)
		r = k.UI.Mouse(dui, k, m, origM, orig.Add(k.R.Min))
		if r.Hit == nil {
			r.Hit = k.UI
		}
		propagateResult(dui, self, k)
		return
	}
	return Result{}
}

func KidsKey(dui *DUI, self *Kid, kids []*Kid, key rune, m draw.Mouse, orig image.Point) (r Result) {
	for i, k := range kids {
		if !m.Point.In(k.R) {
			continue
		}
		m.Point = m.Point.Sub(k.R.Min)
		r = k.UI.Key(dui, k, key, m, orig.Add(k.R.Min))
		if !r.Consumed && key == '\t' {
			for next := i + 1; next < len(kids); next++ {
				first := kids[next].UI.FirstFocus(dui, kids[next])
				if first != nil {
					p := first.Add(orig).Add(kids[next].R.Min)
					r.Warp = &p
					r.Consumed = true
					r.Hit = kids[next].UI
					break
				}
			}
		}
		if r.Hit == nil {
			r.Hit = self.UI
		}
		propagateResult(dui, self, k)
		return
	}
	return Result{}
}

func KidsFirstFocus(dui *DUI, self *Kid, kids []*Kid) *image.Point {
	for _, k := range kids {
		if first := k.UI.FirstFocus(dui, k); first != nil {
			p := first.Add(k.R.Min)
			return &p
		}
	}
	return nil
}

func KidsFocus(dui *DUI, self *Kid, kids []*Kid, ui UI) *image.Point {
	for _, k := range kids {
		if p := k.UI.Focus(dui, k, ui); p != nil {
			pp := p.Add(k.R.Min)
			return &pp
		}
	}
	return nil
}

func KidsMark(self *Kid, kids []*Kid, o UI, forLayout bool) (marked bool) {
	if self.Mark(o, forLayout) {
		return true
	}
	for _, k := range kids {
		if k.UI.Mark(k, o, forLayout) {
			if forLayout {
				if self.Layout == Clean {
					self.Layout = DirtyKid
				}
			} else if self.Draw == Clean {
				self.Draw = DirtyKid
			}
			return true
		}
	}
	return false
}

func KidsPrint(kids []*Kid, indent int) {
	for _, k := range kids {
		k.UI.Print(k, indent)
	}
}

func PrintUI(s string, self *Kid, indent int) {
	prefix := ""
	if indent > 0 {
		prefix = fmt.Sprintf("%*s", indent*2, " ")
	}
	fmt.Printf("duit: %s%s r %v size %v layout=%d draw=%d\n", prefix, s, self.R, self.R.Size(), self.Layout, self.Draw)
}
