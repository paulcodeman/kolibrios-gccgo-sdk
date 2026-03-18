package ui

func (list FragmentDisplayList) Paint(canvas *Canvas, full bool, dirty Rect) {
	list.PaintOffset(canvas, full, dirty, 0, 0)
}

func (list FragmentDisplayList) PaintOffset(canvas *Canvas, full bool, dirty Rect, offsetX int, offsetY int) {
	if canvas == nil {
		return
	}
	for _, item := range list.items {
		if item.Fragment == nil || item.Paint.Empty() {
			continue
		}
		paint := item.Paint
		if offsetX != 0 || offsetY != 0 {
			paint.X += offsetX
			paint.Y += offsetY
		}
		if !full {
			if dirty.Empty() {
				continue
			}
			paint = IntersectRect(paint, dirty)
			if paint.Empty() {
				continue
			}
		}
		if item.ClipSet {
			clip := item.Clip
			if offsetX != 0 || offsetY != 0 {
				clip.X += offsetX
				clip.Y += offsetY
			}
			canvas.PushClip(clip)
		}
		if !full {
			canvas.PushClip(paint)
		}
		item.Fragment.paintOffset(canvas, offsetX, offsetY)
		if !full {
			canvas.PopClip()
		}
		if item.ClipSet {
			canvas.PopClip()
		}
	}
}

func (list FragmentDisplayList) Find(x int, y int) *DocumentNode {
	for i := len(list.items) - 1; i >= 0; i-- {
		item := list.items[i]
		if item.Fragment == nil {
			continue
		}
		hit := item.Bounds
		if item.ClipSet {
			hit = IntersectRect(hit, item.Clip)
		}
		if hit.Contains(x, y) {
			return item.Fragment.Node
		}
	}
	return nil
}

func buildFragmentDisplayList(root *Fragment, viewport Rect) FragmentDisplayList {
	if root == nil {
		return FragmentDisplayList{}
	}
	items := make([]FragmentDisplayItem, 0, 16)
	appendFragmentDisplayItems(&items, root, clipState{}, viewport)
	return FragmentDisplayList{items: items}
}

func appendFragmentDisplayItems(items *[]FragmentDisplayItem, fragment *Fragment, clip clipState, viewport Rect) {
	if fragment == nil {
		return
	}
	paint := fragment.PaintBounds
	if paint.Empty() {
		paint = fragment.Bounds
	}
	if clip.set {
		paint = IntersectRect(paint, clip.rect)
	}
	if !paint.Empty() {
		*items = append(*items, FragmentDisplayItem{
			Fragment: fragment,
			Bounds:   fragment.Bounds,
			Paint:    paint,
			Clip:     clip.rect,
			ClipSet:  clip.set,
		})
	}
	childClip := clip
	if fragment.Kind == FragmentKindBlock {
		clipX, clipY := overflowClipAxes(fragment.Style)
		if clipX || clipY {
			childClip = mergeFragmentClip(viewport, clip, fragment.Content, clipX, clipY)
		}
	}
	for _, child := range fragment.Children {
		appendFragmentDisplayItems(items, child, childClip, viewport)
	}
}

func mergeFragmentClip(viewport Rect, parent clipState, rect Rect, clipX bool, clipY bool) clipState {
	if !clipX && !clipY {
		return parent
	}
	base := viewport
	if base.Empty() {
		base = rect
	}
	if parent.set {
		base = parent.rect
	}
	if clipX {
		base.X = rect.X
		base.Width = rect.Width
	}
	if clipY {
		base.Y = rect.Y
		base.Height = rect.Height
	}
	if !viewport.Empty() {
		base = IntersectRect(base, viewport)
	}
	if parent.set {
		base = IntersectRect(base, parent.rect)
	}
	return clipState{rect: base, set: true}
}

func (fragment *Fragment) paintOffset(canvas *Canvas, offsetX int, offsetY int) {
	if fragment == nil || canvas == nil {
		return
	}
	switch fragment.Kind {
	case FragmentKindText:
		fragment.paintTextOffset(canvas, offsetX, offsetY)
	default:
		bounds := fragment.Bounds
		if offsetX != 0 || offsetY != 0 {
			bounds.X += offsetX
			bounds.Y += offsetY
		}
		drawStyledBox(canvas, bounds, fragment.Style, bounds, nil)
	}
}

func (fragment *Fragment) paintTextOffset(canvas *Canvas, offsetX int, offsetY int) {
	if fragment == nil || canvas == nil || fragment.Bounds.Empty() || fragment.Text == "" {
		return
	}
	if FastNoText {
		return
	}
	lines := fragment.lines
	if len(lines) == 0 {
		return
	}
	foreground, ok := resolveColor(fragment.Style.Foreground)
	if !ok {
		foreground = Black
	}
	font := fragment.font
	charWidth := fragment.metrics.width
	lineHeight := fragment.metrics.height
	if charWidth <= 0 {
		charWidth = defaultCharWidth
	}
	if lineHeight <= 0 {
		lineHeight = defaultFontHeight
	}
	leftPad, topPad, rightPad, availableW := textPaddingAndWidth(fragment.Bounds, fragment.Style)
	shadow, shadowOK := resolveTextShadow(fragment.Style.TextShadow)
	if FastNoTextShadow || FastNoShadows {
		shadowOK = false
	}
	for i, line := range lines {
		if line.text == "" {
			continue
		}
		bounds := fragment.Bounds
		if offsetX != 0 || offsetY != 0 {
			bounds.X += offsetX
			bounds.Y += offsetY
		}
		x := textLineX(bounds, fragment.Style, leftPad, rightPad, availableW, line.text, font, charWidth)
		y := bounds.Y + topPad + i*lineHeight
		if shadowOK {
			if font != nil {
				canvas.DrawTextFont(x+shadow.OffsetX, y+shadow.OffsetY, shadow.Color, line.text, font)
			} else {
				canvas.DrawText(x+shadow.OffsetX, y+shadow.OffsetY, shadow.Color, line.text)
			}
		}
		if font != nil {
			canvas.DrawTextFont(x, y, foreground, line.text, font)
		} else {
			canvas.DrawText(x, y, foreground, line.text)
		}
	}
}
